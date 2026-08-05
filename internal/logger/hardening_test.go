package logger

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	"google.golang.org/protobuf/proto"
)

func httpOnlyConfig(t *testing.T, url string) Config {
	t.Helper()
	cfg := DefaultConfig()
	cfg.Service = "hardening"
	cfg.Stdout = false
	cfg.Async = false
	cfg.File = &FileConfig{Enabled: false}
	cfg.HTTP = &HTTPConfig{
		Enabled:         true,
		URL:             url,
		TimeoutMS:       2000,
		BatchSize:       2,
		FlushIntervalMS: 10_000, // long, so tests drive flushes explicitly
	}
	return cfg
}

func otlpOnlyConfig(t *testing.T, url string) Config {
	t.Helper()
	cfg := DefaultConfig()
	cfg.Service = "hardening"
	cfg.Stdout = false
	cfg.Async = false
	cfg.File = &FileConfig{Enabled: false}
	cfg.OTLP = &OTLPConfig{
		Enabled:         true,
		URL:             url,
		TimeoutMS:       2000,
		BatchSize:       2,
		FlushIntervalMS: 10_000,
	}
	return cfg
}

// A failed POST must not discard the batch: the same lines have to be retried
// on the next flush, otherwise a collector outage silently loses logs.
func TestHTTPSinkRetainsBatchOnFailure(t *testing.T) {
	var (
		mu     sync.Mutex
		bodies []string
		fail   atomic.Bool
	)
	fail.Store(true)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, string(body))
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	log, err := New(httpOnlyConfig(t, srv.URL))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer log.Close()

	_ = log.Info("first", nil)
	_ = log.Info("second", nil)

	if err := log.Flush(); err == nil {
		t.Fatal("expected flush error while collector is failing")
	}

	sink := httpSinkOf(t, log)
	if got := sink.Buffered(); got == 0 {
		t.Fatal("batch was discarded after a failed POST; logs would be lost")
	}

	fail.Store(false)
	if err := log.Flush(); err != nil {
		t.Fatalf("Flush after recovery: %v", err)
	}
	if got := sink.Buffered(); got != 0 {
		t.Fatalf("expected buffer drained after success, still holding %d", got)
	}

	mu.Lock()
	defer mu.Unlock()
	joined := strings.Join(bodies, "")
	if !strings.Contains(joined, `"message":"first"`) || !strings.Contains(joined, `"message":"second"`) {
		t.Fatalf("retried batch lost log content: %q", joined)
	}
}

// The retry buffer must be bounded so a long outage cannot exhaust memory.
func TestHTTPSinkBoundsRetryBuffer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	cfg := httpOnlyConfig(t, srv.URL)
	cfg.HTTP.BatchSize = 2
	log, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer log.Close()

	sink := httpSinkOf(t, log)
	ceiling := cfg.HTTP.BatchSize * maxBufferedBatches

	for i := 0; i < ceiling*3; i++ {
		_ = log.Info("overflow", map[string]any{"i": i})
	}
	if got := sink.Buffered(); got > ceiling {
		t.Fatalf("retry buffer grew past ceiling: %d > %d", got, ceiling)
	}
	if sink.DroppedLines() == 0 {
		t.Fatal("expected dropped counter to record trimmed lines")
	}
	if st := log.Stats(); st.SinkDropped == 0 {
		t.Fatalf("trimmed lines missing from stats: %+v", st)
	}
}

// Async callers get nil from Log(), so sink failures must show up in stats.
func TestStatsReportWriteErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	cfg := httpOnlyConfig(t, srv.URL)
	cfg.HTTP.BatchSize = 1 // flush on every write so failures surface immediately
	log, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer log.Close()

	_ = log.Info("boom", nil)

	st := log.Stats()
	if st.WriteErrors == 0 {
		t.Fatal("expected write_errors to be recorded for a failing sink")
	}
	if st.Buffered == 0 {
		t.Fatal("expected buffered lines to be reported while collector is down")
	}
}

// Async Close must surface flush/close failures instead of returning nil.
func TestAsyncCloseSurfacesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg := httpOnlyConfig(t, srv.URL)
	cfg.Async = true
	cfg.QueueSize = 100
	log, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Whether the worker drains this before or during shutdown, the collector
	// rejects it, so the final flush must fail either way.
	_ = log.Info("pending", nil)

	if err := log.Close(); err == nil {
		t.Fatal("expected Close to report that logs were never accepted")
	}
}

func TestRejectsNonHTTPCollectorURL(t *testing.T) {
	for _, bad := range []string{"ftp://collector/logs", "file:///etc/passwd", "collector:4318", "https://"} {
		cfg := DefaultConfig()
		cfg.Service = "url-check"
		cfg.HTTP = &HTTPConfig{Enabled: true, URL: bad}
		if err := cfg.Validate(); err == nil {
			t.Fatalf("expected %q to be rejected", bad)
		}
	}

	cfg := DefaultConfig()
	cfg.Service = "url-check"
	cfg.HTTP = &HTTPConfig{Enabled: true, URL: "https://collector.internal:4318/v1/logs"}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid https url rejected: %v", err)
	}
}

// Reload rewrites overflow while producers read it; run with -race.
func TestReloadOverflowWhileLogging(t *testing.T) {
	path := filepath.Join(t.TempDir(), "overflow-reload.log")
	cfg := fileOnlyConfig(t, path)
	cfg.Async = true
	cfg.QueueSize = 8
	cfg.Overflow = OverflowDropNewest

	log, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer log.Close()

	var wg sync.WaitGroup
	stop := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
				_ = log.Info("racing", map[string]any{"i": i})
			}
		}
	}()

	policies := []string{OverflowDropOldest, OverflowBlock, OverflowDropNewest}
	for i := 0; i < 30; i++ {
		next := cfg
		next.Overflow = policies[i%len(policies)]
		if err := log.ReloadConfig(next); err != nil {
			close(stop)
			wg.Wait()
			t.Fatalf("ReloadConfig: %v", err)
		}
	}
	close(stop)
	wg.Wait()
}

// Header values are credentials; they must never appear in error strings or
// stats, which callers routinely log or ship to monitoring.
func TestHTTPHeaderSecretsNeverLeak(t *testing.T) {
	const secret = "super-secret-collector-token"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	cfg := httpOnlyConfig(t, srv.URL)
	cfg.HTTP.BatchSize = 1
	cfg.HTTP.Headers = map[string]string{"Authorization": "Bearer " + secret}

	log, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer log.Close()

	writeErr := log.Info("credentials check", nil)
	if writeErr == nil {
		t.Fatal("expected a write error from the unauthorized collector")
	}
	if strings.Contains(writeErr.Error(), secret) {
		t.Fatalf("write error leaked the header secret: %v", writeErr)
	}
	if stats := log.StatsJSON(); strings.Contains(stats, secret) {
		t.Fatalf("stats leaked the header secret: %s", stats)
	}

	bad := cfg
	bad.HTTP = &HTTPConfig{Enabled: true, URL: "ftp://collector", Headers: cfg.HTTP.Headers}
	if err := bad.Validate(); err == nil {
		t.Fatal("expected validation error")
	} else if strings.Contains(err.Error(), secret) {
		t.Fatalf("validation error leaked the header secret: %v", err)
	}
}

func TestOTLPSinkRetainsBatchOnFailure(t *testing.T) {
	var (
		mu     sync.Mutex
		bodies [][]byte
		fail   atomic.Bool
	)
	fail.Store(true)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, body)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	log, err := New(otlpOnlyConfig(t, srv.URL))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer log.Close()

	_ = log.Info("first", nil)
	_ = log.Info("second", nil)

	if err := log.Flush(); err == nil {
		t.Fatal("expected flush error while collector is failing")
	}

	sink := otlpSinkOf(t, log)
	if got := sink.Buffered(); got == 0 {
		t.Fatal("batch was discarded after a failed POST; logs would be lost")
	}

	fail.Store(false)
	if err := log.Flush(); err != nil {
		t.Fatalf("Flush after recovery: %v", err)
	}
	if got := sink.Buffered(); got != 0 {
		t.Fatalf("expected buffer drained after success, still holding %d", got)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(bodies) == 0 {
		t.Fatal("expected at least one OTLP request body")
	}

	var req collogspb.ExportLogsServiceRequest
	if err := proto.Unmarshal(bodies[0], &req); err != nil {
		t.Fatalf("unmarshal otlp request: %v", err)
	}
	foundFirst := false
	foundSecond := false
	for _, rl := range req.ResourceLogs {
		for _, sl := range rl.ScopeLogs {
			for _, lr := range sl.LogRecords {
				if lr.Body.GetStringValue() == "first" {
					foundFirst = true
				}
				if lr.Body.GetStringValue() == "second" {
					foundSecond = true
				}
			}
		}
	}
	if !foundFirst || !foundSecond {
		t.Fatalf("otlp payload missing expected bodies: first=%v second=%v", foundFirst, foundSecond)
	}
}

func httpSinkOf(t *testing.T, l *Logger) *httpSink {
	t.Helper()
	l.mu.RLock()
	defer l.mu.RUnlock()
	for _, s := range l.sinks {
		if hs, ok := s.(*httpSink); ok {
			return hs
		}
	}
	t.Fatal("logger has no http sink")
	return nil
}

func otlpSinkOf(t *testing.T, l *Logger) *otlpSink {
	t.Helper()
	l.mu.RLock()
	defer l.mu.RUnlock()
	for _, s := range l.sinks {
		if os, ok := s.(*otlpSink); ok {
			return os
		}
	}
	t.Fatal("logger has no otlp sink")
	return nil
}
