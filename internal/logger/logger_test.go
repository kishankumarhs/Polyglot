package logger

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestParseLevel(t *testing.T) {
	level, err := ParseLevel("WARN")
	if err != nil {
		t.Fatalf("ParseLevel: %v", err)
	}
	if level != LevelWarn {
		t.Fatalf("expected warn, got %v", level)
	}
	if _, err := ParseLevel("nope"); err == nil {
		t.Fatal("expected error for unknown level")
	}
	lv, err := LevelFromInt(2)
	if err != nil || lv != LevelInfo {
		t.Fatalf("LevelFromInt: %v %v", lv, err)
	}
}

func TestConfigNestedAndLegacy(t *testing.T) {
	nested := []byte(`{
		"service":"workflow-service",
		"service_version":"1.0.0",
		"environment":"prod",
		"level":"info",
		"stdout":false,
		"file":{"enabled":true,"path":"./logs/app.log","maxSizeMB":50,"maxBackups":3,"maxAgeDays":7,"compress":false,"fsync":true},
		"http":{"enabled":false,"url":"http://collector/v1/logs"},
		"async":true,
		"queueSize":100,
		"overflow":"drop_oldest",
		"fields":{"region":"us"}
	}`)
	cfg, err := ParseConfigJSON(nested)
	if err != nil {
		t.Fatalf("nested: %v", err)
	}
	if cfg.Service != "workflow-service" || cfg.Level != "info" || !cfg.FileEnabled() {
		t.Fatalf("unexpected nested cfg: %+v", cfg)
	}
	if cfg.Overflow != OverflowDropOldest || cfg.QueueSize != 100 {
		t.Fatalf("unexpected async cfg: %+v", cfg)
	}
	if !cfg.File.FSync {
		t.Fatalf("expected file.fsync to parse true")
	}

	legacy := []byte(`{
		"service_name":"legacy-svc",
		"min_level":"debug",
		"stdout":false,
		"file_path":"/tmp/a.log",
		"max_size_mb":20,
		"max_backups":2,
		"max_age_days":5
	}`)
	cfg, err = ParseConfigJSON(legacy)
	if err != nil {
		t.Fatalf("legacy: %v", err)
	}
	if cfg.Service != "legacy-svc" || cfg.Level != "debug" || !cfg.FileEnabled() || cfg.File.Path != "/tmp/a.log" {
		t.Fatalf("unexpected legacy cfg: %+v", cfg)
	}
}

func TestConfigValidation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Stdout = false
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error when no sinks configured")
	}

	cfg = DefaultConfig()
	cfg.Service = ""
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected service validation error")
	}

	cfg = DefaultConfig()
	cfg.Overflow = "nope"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected overflow validation error")
	}

	if _, err := ParseConfigJSON([]byte(`{"service":"","stdout":true}`)); err == nil {
		t.Fatal("expected explicit empty service to fail")
	}
}

func TestKafkaConfigValidation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Service = "kafka-check"
	cfg.Stdout = false
	cfg.File = &FileConfig{Enabled: false}
	cfg.Kafka = &KafkaConfig{Enabled: true, Topic: "", Brokers: []string{"localhost:9092"}}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected kafka.topic validation error")
	}

	cfg.Kafka.Topic = "logs"
	cfg.Kafka.RequiredAcks = 2
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected kafka.required_acks validation error")
	}

	cfg.Kafka.RequiredAcks = 1
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid kafka config: %v", err)
	}
}

func TestOTLPConfigValidation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Service = "otlp-check"
	cfg.Stdout = false
	cfg.File = &FileConfig{Enabled: false}
	cfg.OTLP = &OTLPConfig{Enabled: true, URL: ""}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected otlp.url validation error")
	}

	cfg.OTLP.URL = "https://collector.internal/v1/logs"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid otlp config: %v", err)
	}
}

func fileOnlyConfig(t *testing.T, path string) Config {
	t.Helper()
	cfg := DefaultConfig()
	cfg.Service = "test"
	cfg.Level = "info"
	cfg.Stdout = false
	cfg.Async = false
	cfg.File = &FileConfig{
		Enabled:    true,
		Path:       path,
		MaxSizeMB:  100,
		MaxBackups: 5,
		MaxAgeDays: 28,
	}
	return cfg
}

func TestLoggerWritesJSONFileAndFiltersLevel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")

	cfg := fileOnlyConfig(t, path)
	cfg.Service = "payments"
	cfg.ServiceVersion = "1.2.3"
	cfg.Environment = "test"
	cfg.Fields = map[string]any{"region": "us-east"}

	log, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer log.Close()

	if err := log.Debug("hidden", map[string]any{"n": 1}); err != nil {
		t.Fatalf("Debug: %v", err)
	}
	if err := log.Info("hello", map[string]any{"user_id": 42}); err != nil {
		t.Fatalf("Info: %v", err)
	}
	if err := log.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 log line, got %d: %q", len(lines), string(data))
	}

	var entry Entry
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if entry.Level != "info" || entry.Message != "hello" || entry.ServiceName != "payments" {
		t.Fatalf("unexpected entry: %+v", entry)
	}
	if entry.Fields["region"] != "us-east" {
		t.Fatalf("missing default field: %+v", entry.Fields)
	}
	if entry.Fields["user_id"].(float64) != 42 {
		t.Fatalf("missing user field: %+v", entry.Fields)
	}
}

func TestSetFieldsContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ctx.log")
	cfg := fileOnlyConfig(t, path)
	cfg.Fields = map[string]any{"base": "1"}
	log, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer log.Close()

	log.SetFields(map[string]any{"traceId": "abc", "tenantId": "t1"})
	if err := log.Info("with-context", map[string]any{"userId": "u1"}); err != nil {
		t.Fatalf("Info: %v", err)
	}
	_ = log.Flush()

	data, _ := os.ReadFile(path)
	var entry Entry
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(data))), &entry); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if entry.Fields["base"] != "1" || entry.Fields["traceId"] != "abc" || entry.Fields["userId"] != "u1" {
		t.Fatalf("fields merge failed: %+v", entry.Fields)
	}
}

func TestLoggerConcurrentWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "concurrent.log")
	cfg := fileOnlyConfig(t, path)
	cfg.Level = "debug"
	cfg.Async = true
	cfg.QueueSize = 5000
	cfg.Overflow = OverflowBlock

	log, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer log.Close()

	const workers = 20
	const perWorker = 50
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < perWorker; j++ {
				_ = log.Info("concurrent", map[string]any{"worker": id, "n": j})
			}
		}(i)
	}
	wg.Wait()
	if err := log.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		count++
		var entry Entry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			t.Fatalf("invalid json line: %v", err)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner: %v", err)
	}
	if count != workers*perWorker {
		t.Fatalf("expected %d lines, got %d", workers*perWorker, count)
	}
	st := log.Stats()
	if st.Flushed != uint64(workers*perWorker) {
		t.Fatalf("expected flushed=%d, got %d", workers*perWorker, st.Flushed)
	}
}

// gateSink parks the worker inside Write until the test releases it. Overflow
// behavior can then be asserted exactly, instead of hoping the producer outruns
// the worker (which it does not on a fast machine).
type gateSink struct {
	mu      sync.Mutex
	lines   []string
	entered chan struct{}
	once    sync.Once
	release chan struct{}
}

func newGateSink() *gateSink {
	return &gateSink{entered: make(chan struct{}), release: make(chan struct{})}
}

func (g *gateSink) Write(line []byte) (int, error) {
	g.mu.Lock()
	g.lines = append(g.lines, string(line))
	g.mu.Unlock()
	g.once.Do(func() { close(g.entered) })
	<-g.release
	return len(line), nil
}

func (g *gateSink) Flush() error  { return nil }
func (g *gateSink) Close() error  { return nil }
func (g *gateSink) Name() string  { return "gate" }
func (g *gateSink) Buffered() int { return 0 }

func (g *gateSink) written() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return strings.Join(g.lines, "")
}

func (g *gateSink) open() {
	close(g.release)
}

// stallWorker swaps in a gate sink and blocks the worker inside it, leaving the
// queue empty and guaranteed not to drain until the gate opens.
func stallWorker(t *testing.T, l *Logger) *gateSink {
	t.Helper()
	g := newGateSink()

	l.mu.Lock()
	old := l.sinks
	l.sinks = []Sink{g}
	l.mu.Unlock()
	_ = closeSinks(old)

	if err := l.Info("stall", nil); err != nil {
		t.Fatalf("stall write: %v", err)
	}
	select {
	case <-g.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("worker never reached the gate sink")
	}
	return g
}

func TestOverflowDropNewest(t *testing.T) {
	const total = 200

	cfg := fileOnlyConfig(t, filepath.Join(t.TempDir(), "drop_newest.log"))
	cfg.Async = true
	cfg.QueueSize = 8
	cfg.Overflow = OverflowDropNewest

	log, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer log.Close()

	gate := stallWorker(t, log)

	for i := 0; i < total; i++ {
		_ = log.Info("flood", map[string]any{"i": i})
	}

	// The queue holds QueueSize entries and nothing drains, so every remaining
	// entry must be rejected.
	st := log.Stats()
	if want := uint64(total - cfg.QueueSize); st.Dropped != want {
		t.Fatalf("expected dropped=%d, got %+v", want, st)
	}

	gate.open()
	if err := log.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	// drop_newest keeps the oldest queued entries and rejects later ones.
	got := gate.written()
	if !strings.Contains(got, `"i":0`) {
		t.Fatal("drop_newest discarded the oldest queued entry")
	}
	if strings.Contains(got, `"i":199`) {
		t.Fatal("drop_newest kept an entry that arrived after the queue was full")
	}
}

func TestOverflowDropOldest(t *testing.T) {
	const total = 100

	cfg := fileOnlyConfig(t, filepath.Join(t.TempDir(), "drop_oldest.log"))
	cfg.Async = true
	cfg.QueueSize = 4
	cfg.Overflow = OverflowDropOldest

	log, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer log.Close()

	gate := stallWorker(t, log)

	for i := 0; i < total; i++ {
		_ = log.Info("flood", map[string]any{"i": i})
	}

	st := log.Stats()
	if want := uint64(total - cfg.QueueSize); st.Dropped != want {
		t.Fatalf("expected dropped=%d, got %+v", want, st)
	}

	gate.open()
	if err := log.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	// drop_oldest evicts early entries and keeps the most recent ones.
	got := gate.written()
	if !strings.Contains(got, fmt.Sprintf(`"i":%d`, total-1)) {
		t.Fatal("drop_oldest discarded the newest entry")
	}
	if strings.Contains(got, `"i":0`) {
		t.Fatal("drop_oldest kept the oldest entry instead of evicting it")
	}
}

func TestReloadConfig(t *testing.T) {
	dir := t.TempDir()
	path1 := filepath.Join(dir, "a.log")
	path2 := filepath.Join(dir, "b.log")

	cfg := fileOnlyConfig(t, path1)
	cfg.Level = "info"
	log, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer log.Close()

	_ = log.Info("before", nil)
	_ = log.Flush()

	next := fileOnlyConfig(t, path2)
	next.Level = "error"
	next.Service = "reloaded"
	if err := log.ReloadConfig(next); err != nil {
		t.Fatalf("ReloadConfig: %v", err)
	}
	_ = log.Info("filtered", nil)
	_ = log.Error("after", nil)
	_ = log.Flush()

	if data, _ := os.ReadFile(path1); !strings.Contains(string(data), "before") {
		t.Fatalf("expected before in path1")
	}
	data2, err := os.ReadFile(path2)
	if err != nil {
		t.Fatalf("read path2: %v", err)
	}
	if strings.Contains(string(data2), "filtered") {
		t.Fatal("info should be filtered after reload to error")
	}
	if !strings.Contains(string(data2), "after") {
		t.Fatalf("expected after in path2: %s", data2)
	}
}

func TestReloadDuringConcurrentWrites(t *testing.T) {
	dir := t.TempDir()
	log, err := New(fileOnlyConfig(t, filepath.Join(dir, "reload-0.log")))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer log.Close()

	var (
		errMu     sync.Mutex
		firstErr  error
		stop      = make(chan struct{})
		wg        sync.WaitGroup
		writeSeen atomic.Int64
	)
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				if err := log.Info("during-reload", map[string]any{"worker": id}); err != nil {
					errMu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					errMu.Unlock()
					return
				}
				writeSeen.Add(1)
			}
		}(w)
	}

	for i := 1; i <= 20; i++ {
		next := fileOnlyConfig(t, filepath.Join(dir, fmt.Sprintf("reload-%d.log", i)))
		if err := log.ReloadConfig(next); err != nil {
			t.Errorf("ReloadConfig(%d): %v", i, err)
			break
		}
	}
	close(stop)
	wg.Wait()

	errMu.Lock()
	defer errMu.Unlock()
	if firstErr != nil {
		t.Fatalf("log write failed while config was reloading: %v", firstErr)
	}
	if writeSeen.Load() == 0 {
		t.Fatal("expected writes to land during reload")
	}
}

func TestHTTPSink(t *testing.T) {
	var (
		mu       sync.Mutex
		bodies   []string
		reqCount atomic.Int32
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqCount.Add(1)
		if r.Header.Get("Content-Type") != "application/x-ndjson" {
			t.Errorf("unexpected content-type: %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing auth header")
		}
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, string(b))
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.Service = "http-test"
	cfg.Stdout = false
	cfg.Async = false
	cfg.File = &FileConfig{Enabled: false}
	cfg.HTTP = &HTTPConfig{
		Enabled:         true,
		URL:             srv.URL,
		TimeoutMS:       2000,
		Headers:         map[string]string{"Authorization": "Bearer test-token"},
		BatchSize:       2,
		FlushIntervalMS: 50,
	}

	log, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer log.Close()

	_ = log.Info("one", map[string]any{"n": 1})
	_ = log.Info("two", map[string]any{"n": 2})
	_ = log.Flush()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if reqCount.Load() >= 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(bodies) == 0 {
		t.Fatal("expected http sink to POST at least one batch")
	}
	if !strings.Contains(bodies[0], `"message":"one"`) && !strings.Contains(strings.Join(bodies, ""), `"message":"one"`) {
		t.Fatalf("missing log content in bodies: %#v", bodies)
	}
}

func TestRotation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rotate.log")

	cfg := fileOnlyConfig(t, path)
	cfg.File.MaxBackups = 2
	cfg.File.MaxAgeDays = 1

	log, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer log.Close()

	forceFileMaxSize(log, 200)
	for i := 0; i < 50; i++ {
		if err := log.Info(strings.Repeat("x", 40), map[string]any{"i": i}); err != nil {
			t.Fatalf("Info: %v", err)
		}
	}
	if err := log.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	matches, err := filepath.Glob(path + ".*")
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("expected rotated backup files")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("active log missing: %v", err)
	}
}

func TestFileFSyncMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fsync.log")

	cfg := fileOnlyConfig(t, path)
	cfg.File.FSync = true
	log, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer log.Close()

	if err := log.Info("durable", map[string]any{"k": "v"}); err != nil {
		t.Fatalf("Info: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), `"message":"durable"`) {
		t.Fatalf("expected fsync file write to be visible, got: %s", string(data))
	}
}

func forceFileMaxSize(l *Logger, n int64) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	for _, s := range l.sinks {
		if fs, ok := s.(*fileSink); ok {
			fs.w.maxSize = n
		}
	}
}

func TestClosedLoggerRejectsWrites(t *testing.T) {
	cfg := fileOnlyConfig(t, filepath.Join(t.TempDir(), "closed.log"))
	log, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := log.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := log.Info("nope", nil); err == nil {
		t.Fatal("expected error after close")
	}
}

func TestLogJSONInvalidFields(t *testing.T) {
	cfg := fileOnlyConfig(t, filepath.Join(t.TempDir(), "json.log"))
	log, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer log.Close()

	if err := log.LogJSON("info", "msg", "{bad"); err == nil {
		t.Fatal("expected invalid fields error")
	}
}

func TestAsyncFlushStats(t *testing.T) {
	path := filepath.Join(t.TempDir(), "async.log")
	cfg := fileOnlyConfig(t, path)
	cfg.Async = true
	cfg.QueueSize = 1000
	cfg.Overflow = OverflowDropNewest

	log, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer log.Close()

	for i := 0; i < 10; i++ {
		_ = log.Info("async", map[string]any{"i": i})
	}
	if err := log.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	st := log.Stats()
	if st.Flushed != 10 || st.BytesWritten == 0 || st.Queued != 0 {
		t.Fatalf("unexpected stats: %+v", st)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if got := len(strings.Split(strings.TrimSpace(string(data)), "\n")); got != 10 {
		t.Fatalf("expected 10 lines, got %d", got)
	}
}
