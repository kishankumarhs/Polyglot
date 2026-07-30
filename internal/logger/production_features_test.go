package logger

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestWithChildLoggerDoesNotRaceParentFields(t *testing.T) {
	cfg := fileOnlyConfig(t, filepath.Join(t.TempDir(), "with.log"))
	cfg.Async = false
	root, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer root.Close()

	childA := root.With(map[string]any{"requestId": "a"})
	childB := root.With(map[string]any{"requestId": "b"})
	if err := childA.Info("one", nil); err != nil {
		t.Fatal(err)
	}
	if err := childB.Info("two", nil); err != nil {
		t.Fatal(err)
	}
	if err := childA.Close(); err != nil {
		t.Fatalf("child close should be no-op: %v", err)
	}
	// Root still open and usable.
	if err := root.Info("root", nil); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(cfg.File.Path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !strings.Contains(body, `"requestId":"a"`) || !strings.Contains(body, `"requestId":"b"`) {
		t.Fatalf("missing child fields: %s", body)
	}
}

func TestLogContextTraceFields(t *testing.T) {
	cfg := fileOnlyConfig(t, filepath.Join(t.TempDir(), "ctx.log"))
	cfg.Async = false
	log, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	ctx := ContextWithTrace(context.Background(), "trace-1", "span-9")
	if err := log.LogContext(ctx, LevelInfo, "traced", nil); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(cfg.File.Path)
	if !strings.Contains(string(data), `"trace_id":"trace-1"`) || !strings.Contains(string(data), `"span_id":"span-9"`) {
		t.Fatalf("missing trace fields: %s", data)
	}
}

func TestCallerAndLogError(t *testing.T) {
	cfg := fileOnlyConfig(t, filepath.Join(t.TempDir(), "caller.log"))
	cfg.Async = false
	cfg.Caller = true
	log, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	_ = log.LogError(os.ErrNotExist, "failed open", nil)
	data, _ := os.ReadFile(cfg.File.Path)
	body := string(data)
	if !strings.Contains(body, `"caller":`) {
		t.Fatalf("expected caller: %s", body)
	}
	if !strings.Contains(body, `"error":`) {
		t.Fatalf("expected error field: %s", body)
	}
}

func TestStdoutTextFormat(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Service = "text-test"
	cfg.Stdout = true
	cfg.StdoutFormat = StdoutFormatText
	cfg.Async = false
	cfg.File = &FileConfig{Enabled: false}
	log, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	if err := log.Info("hello text", map[string]any{"n": 1}); err != nil {
		t.Fatal(err)
	}
}

func TestSamplingDropsRepeats(t *testing.T) {
	cfg := fileOnlyConfig(t, filepath.Join(t.TempDir(), "sample.log"))
	cfg.Async = false
	cfg.Sampling = &SamplingConfig{Enabled: true, Initial: 2, Thereafter: 10}
	log, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	for i := 0; i < 22; i++ {
		_ = log.Info("hot", nil)
	}
	data, _ := os.ReadFile(cfg.File.Path)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	// initial 2 + every 10th of the remaining 20 → 2 + 2 = 4 (n=2 and n=12 after initial)
	// counts: n=0,1 emitted (initial); n=2..21: emit when (n-2)%10==0 → n=2,12 → total 4
	if len(lines) != 4 {
		t.Fatalf("expected 4 sampled lines, got %d (%q)", len(lines), data)
	}
	if log.Stats().Dropped == 0 {
		t.Fatal("expected sampling drops")
	}
}

func TestStrictConfigMissingFile(t *testing.T) {
	t.Setenv("POLYLOG_STRICT", "1")
	t.Setenv("POLYGLOT_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.yaml"))
	if _, err := LoadConfigFromFile(""); err == nil {
		t.Fatal("expected strict error for missing config")
	}
}

func TestLokiSinkPushFormat(t *testing.T) {
	var got atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got.Store(string(b))
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content-type=%s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.Service = "loki-test"
	cfg.Stdout = false
	cfg.Async = false
	cfg.File = &FileConfig{Enabled: false}
	cfg.Loki = &LokiConfig{
		Enabled:         true,
		URL:             srv.URL,
		BatchSize:       1,
		FlushIntervalMS: 60_000,
		Labels:          map[string]string{"job": "polyglot"},
	}
	log, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	_ = log.Info("to loki", map[string]any{"k": 1})
	_ = log.Flush()

	body, _ := got.Load().(string)
	if !strings.Contains(body, `"streams"`) || !strings.Contains(body, `"service_name"`) {
		t.Fatalf("unexpected loki body: %s", body)
	}
	if !strings.Contains(body, `"job":"polyglot"`) {
		t.Fatalf("missing static label: %s", body)
	}
}

func TestMetricsText(t *testing.T) {
	cfg := fileOnlyConfig(t, filepath.Join(t.TempDir(), "metrics.log"))
	cfg.Async = false
	log, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	_ = log.Info("m", nil)
	text := log.MetricsText()
	if !strings.Contains(text, "polyglot_logger_flushed") {
		t.Fatalf("missing metric: %s", text)
	}
}
