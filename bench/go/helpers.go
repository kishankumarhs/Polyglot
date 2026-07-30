package bench

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	core "polyglot/internal/logger"

	"github.com/rs/zerolog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// RichFields returns a production-shaped ~20-field payload.
func RichFields(n int) map[string]any {
	return map[string]any{
		"user_id":    7,
		"trace_id":   "abc123def456ghi789",
		"span_id":    "span-001",
		"service":    "payments",
		"region":     "us-east-1",
		"latency_ms": 12.4,
		"ok":         true,
		"tags":       []string{"a", "b", "c"},
		"meta": map[string]any{
			"cart": map[string]any{
				"items":    3,
				"currency": "USD",
			},
		},
		"error":      "optional message",
		"request_id": "req-xyz",
		"tenant":     "acme",
		"env":        "prod",
		"version":    "1.2.3",
		"attempt":    1,
		"bytes":      4096,
		"cached":     false,
		"n":          n,
	}
}

func polyglotFileConfig(path string, async bool, queue int, overflow string) core.Config {
	if queue <= 0 {
		queue = 10000
	}
	if overflow == "" {
		overflow = core.OverflowDropNewest
	}
	return core.Config{
		Service:   "bench",
		Level:     "info",
		Stdout:    false,
		Async:     async,
		QueueSize: queue,
		Overflow:  overflow,
		File: &core.FileConfig{
			Enabled:    true,
			Path:       path,
			MaxSizeMB:  512,
			MaxBackups: 2,
			MaxAgeDays: 1,
		},
	}
}

func newPolyglot(tb testing.TB, async bool) (*core.Logger, string) {
	tb.Helper()
	path := filepath.Join(tb.TempDir(), "polyglot.log")
	log, err := core.New(polyglotFileConfig(path, async, 10000, core.OverflowDropNewest))
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { _ = log.Close() })
	return log, path
}

func newZapFile(tb testing.TB) (*zap.Logger, string) {
	tb.Helper()
	path := filepath.Join(tb.TempDir(), "zap.log")
	f, err := os.Create(path)
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { _ = f.Close() })
	enc := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	coreZ := zapcore.NewCore(enc, zapcore.AddSync(f), zapcore.InfoLevel)
	log := zap.New(coreZ)
	tb.Cleanup(func() { _ = log.Sync() })
	return log, path
}

func newZerologFile(tb testing.TB) (zerolog.Logger, string) {
	tb.Helper()
	path := filepath.Join(tb.TempDir(), "zerolog.log")
	f, err := os.Create(path)
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { _ = f.Close() })
	return zerolog.New(f).With().Timestamp().Logger(), path
}

func zapRich(n int) []zap.Field {
	return []zap.Field{
		zap.Int("user_id", 7),
		zap.String("trace_id", "abc123def456ghi789"),
		zap.String("span_id", "span-001"),
		zap.String("service", "payments"),
		zap.String("region", "us-east-1"),
		zap.Float64("latency_ms", 12.4),
		zap.Bool("ok", true),
		zap.Strings("tags", []string{"a", "b", "c"}),
		zap.Any("meta", map[string]any{"cart": map[string]any{"items": 3, "currency": "USD"}}),
		zap.String("error", "optional message"),
		zap.String("request_id", "req-xyz"),
		zap.String("tenant", "acme"),
		zap.String("env", "prod"),
		zap.String("version", "1.2.3"),
		zap.Int("attempt", 1),
		zap.Int("bytes", 4096),
		zap.Bool("cached", false),
		zap.Int("n", n),
	}
}

func reportLatency(b *testing.B, h *LatencyHist) {
	b.Helper()
	p := h.Snapshot()
	p.Report(b)
	b.Logf("latency count=%d mean=%.0fns p50=%dns p95=%dns p99=%dns max=%dns",
		p.Count, p.Mean, p.P50, p.P95, p.P99, p.Max)
}

func mustEnvInt(name string, def int) int {
	v := os.Getenv(name)
	if v == "" {
		return def
	}
	var n int
	_, err := fmt.Sscanf(v, "%d", &n)
	if err != nil || n <= 0 {
		return def
	}
	return n
}
