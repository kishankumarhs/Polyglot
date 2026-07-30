package bench

import (
	"testing"
)

func BenchmarkPolyglotSyncFile(b *testing.B) {
	log, _ := newPolyglot(b, false)
	h := NewLatencyHist(b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t0 := Start()
		_ = log.Info("checkout", RichFields(i))
		h.RecordElapsed(t0)
	}
	b.StopTimer()
	_ = log.Flush()
	reportLatency(b, h)
}

func BenchmarkPolyglotAsyncFile(b *testing.B) {
	log, _ := newPolyglot(b, true)
	h := NewLatencyHist(b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t0 := Start()
		_ = log.Info("checkout", RichFields(i))
		h.RecordElapsed(t0)
	}
	b.StopTimer()
	_ = log.Flush()
	reportLatency(b, h)
}

func BenchmarkZapJSONFile(b *testing.B) {
	log, _ := newZapFile(b)
	h := NewLatencyHist(b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t0 := Start()
		log.Info("checkout", zapRich(i)...)
		h.RecordElapsed(t0)
	}
	b.StopTimer()
	_ = log.Sync()
	reportLatency(b, h)
}

func BenchmarkZerologFile(b *testing.B) {
	log, _ := newZerologFile(b)
	h := NewLatencyHist(b.N)
	fields := RichFields(0)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fields["n"] = i
		t0 := Start()
		log.Info().Fields(fields).Msg("checkout")
		h.RecordElapsed(t0)
	}
	b.StopTimer()
	reportLatency(b, h)
}

// BenchmarkSlogJSONFile is a stdlib reference, not a headline competitor.
func BenchmarkSlogJSONFile(b *testing.B) {
	slogBench(b)
}
