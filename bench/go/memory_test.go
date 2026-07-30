package bench

import (
	"runtime"
	"testing"
	"time"
)

func TestMemoryFixedMillion(t *testing.T) {
	n := mustEnvInt("BENCH_MEM_N", 100_000) // 1_000_000 for full suite
	log, _ := newPolyglot(t, false)

	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)

	start := time.Now()
	for i := 0; i < n; i++ {
		_ = log.Info("checkout", RichFields(i))
	}
	_ = log.Flush()
	elapsed := time.Since(start)

	runtime.ReadMemStats(&after)
	t.Logf("memory n=%d elapsed=%s heap_inuse=%dKiB→%dKiB total_allocΔ=%dMiB num_gc=%d→%d pause_totalΔ=%s",
		n, elapsed,
		before.HeapInuse/1024, after.HeapInuse/1024,
		(after.TotalAlloc-before.TotalAlloc)/(1024*1024),
		before.NumGC, after.NumGC,
		time.Duration(after.PauseTotalNs-before.PauseTotalNs),
	)
}

func BenchmarkPolyglotMemoryAllocs(b *testing.B) {
	log, _ := newPolyglot(b, false)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = log.Info("checkout", RichFields(i))
	}
	b.StopTimer()
	_ = log.Flush()
}
