package bench

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func slogBench(b *testing.B) {
	path := filepath.Join(b.TempDir(), "slog.log")
	f, err := os.Create(path)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = f.Close() })
	hnd := slog.NewJSONHandler(f, &slog.HandlerOptions{Level: slog.LevelInfo})
	log := slog.New(hnd)
	h := NewLatencyHist(b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t0 := Start()
		log.Info("checkout",
			"user_id", 7,
			"trace_id", "abc123def456ghi789",
			"span_id", "span-001",
			"service", "payments",
			"region", "us-east-1",
			"latency_ms", 12.4,
			"ok", true,
			"n", i,
		)
		h.RecordElapsed(t0)
	}
	b.StopTimer()
	reportLatency(b, h)
}
