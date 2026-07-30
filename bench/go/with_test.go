package bench

import (
	"testing"

	"go.uber.org/zap"
)

func BenchmarkPolyglotWithChild(b *testing.B) {
	root, _ := newPolyglot(b, false)
	child := root.With(map[string]any{"request_id": "req-1", "handler": "checkout"})
	h := NewLatencyHist(b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t0 := Start()
		_ = child.Info("checkout", RichFields(i))
		h.RecordElapsed(t0)
	}
	b.StopTimer()
	_ = root.Flush()
	reportLatency(b, h)
}

func BenchmarkZapWithChild(b *testing.B) {
	root, _ := newZapFile(b)
	child := root.With(zap.String("request_id", "req-1"), zap.String("handler", "checkout"))
	h := NewLatencyHist(b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t0 := Start()
		child.Info("checkout", zapRich(i)...)
		h.RecordElapsed(t0)
	}
	b.StopTimer()
	_ = root.Sync()
	reportLatency(b, h)
}

func BenchmarkZerologWithChild(b *testing.B) {
	root, _ := newZerologFile(b)
	child := root.With().Str("request_id", "req-1").Str("handler", "checkout").Logger()
	fields := RichFields(0)
	h := NewLatencyHist(b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fields["n"] = i
		t0 := Start()
		child.Info().Fields(fields).Msg("checkout")
		h.RecordElapsed(t0)
	}
	b.StopTimer()
	reportLatency(b, h)
}
