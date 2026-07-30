package bench

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	core "polyglot/internal/logger"
)

// scaleStart/scaleElapsed use time.Now for wall-clock total throughput
// (not per-call latency), which is fine at the multi-second scale.

func BenchmarkPolyglotScale(b *testing.B) {
	for _, writers := range []int{1, 2, 4, 8, 16, 32, 64} {
		writers := writers
		b.Run(fmt.Sprintf("writers_%d", writers), func(b *testing.B) {
			log, _ := newPolyglot(b, false)
			h := NewLatencyHist(b.N)
			b.SetParallelism(writers)
			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				child := log.With(map[string]any{"worker": true})
				i := 0
				for pb.Next() {
					t0 := Start()
					_ = child.Info("checkout", RichFields(i))
					h.RecordElapsed(t0)
					i++
				}
			})
			b.StopTimer()
			_ = log.Flush()
			reportLatency(b, h)
			// Throughput hint for HISTORY / graphs
			if b.N > 0 && b.Elapsed().Seconds() > 0 {
				b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops/s")
			}
		})
	}
}

func BenchmarkZapScale(b *testing.B) {
	for _, writers := range []int{1, 2, 4, 8, 16, 32, 64} {
		writers := writers
		b.Run(fmt.Sprintf("writers_%d", writers), func(b *testing.B) {
			log, _ := newZapFile(b)
			h := NewLatencyHist(b.N)
			b.SetParallelism(writers)
			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					t0 := Start()
					log.Info("checkout", zapRich(i)...)
					h.RecordElapsed(t0)
					i++
				}
			})
			b.StopTimer()
			_ = log.Sync()
			reportLatency(b, h)
			if b.N > 0 && b.Elapsed().Seconds() > 0 {
				b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops/s")
			}
		})
	}
}

// TestScaleCSV writes a simple threads,ops CSV when BENCH_SCALE_CSV=1.
func TestScaleCSV(t *testing.T) {
	if os.Getenv("BENCH_SCALE_CSV") != "1" {
		t.Skip("set BENCH_SCALE_CSV=1 to emit scale CSV")
	}
	out := filepath.Join("..", "results", "scale.csv")
	_ = os.MkdirAll(filepath.Dir(out), 0o755)
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	fmt.Fprintln(f, "writers,polyglot_ops_s,zap_ops_s")
	const perWriter = 5000
	for _, writers := range []int{1, 2, 4, 8, 16, 32, 64} {
		pg := scalePolyglot(t, writers, perWriter)
		zp := scaleZap(t, writers, perWriter)
		fmt.Fprintf(f, "%d,%.0f,%.0f\n", writers, pg, zp)
	}
}

func scalePolyglot(t *testing.T, writers, perWriter int) float64 {
	t.Helper()
	path := filepath.Join(t.TempDir(), "pg.log")
	log, err := core.New(polyglotFileConfig(path, false, 10000, core.OverflowDropNewest))
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	var wg sync.WaitGroup
	start := time.Now()
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			child := log.With(map[string]any{"worker": id})
			for i := 0; i < perWriter; i++ {
				_ = child.Info("checkout", RichFields(i))
			}
		}(w)
	}
	wg.Wait()
	_ = log.Flush()
	elapsed := time.Since(start).Seconds()
	return float64(writers*perWriter) / elapsed
}

func scaleZap(t *testing.T, writers, perWriter int) float64 {
	t.Helper()
	log, _ := newZapFile(t)
	var wg sync.WaitGroup
	start := time.Now()
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perWriter; i++ {
				log.Info("checkout", zapRich(i)...)
			}
		}()
	}
	wg.Wait()
	_ = log.Sync()
	elapsed := time.Since(start).Seconds()
	return float64(writers*perWriter) / elapsed
}
