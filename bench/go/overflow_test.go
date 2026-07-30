package bench

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	core "polyglot/internal/logger"
)

func TestOverflowBackpressure(t *testing.T) {
	n := mustEnvInt("BENCH_OVERFLOW_N", 50_000) // use 10_000_000 for full suite
	path := filepath.Join(t.TempDir(), "overflow.log")
	log, err := core.New(polyglotFileConfig(path, true, 1000, core.OverflowDropNewest))
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)

	start := time.Now()
	for i := 0; i < n; i++ {
		_ = log.Info("flood", RichFields(i))
	}
	floodElapsed := time.Since(start)

	recoverStart := time.Now()
	if err := log.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}
	recoverElapsed := time.Since(recoverStart)

	runtime.ReadMemStats(&after)
	st := log.Stats()

	t.Logf("overflow drop_newest n=%d flood=%s recover=%s dropped=%d flushed=%d queued=%d heapΔ=%dKiB",
		n, floodElapsed, recoverElapsed, st.Dropped, st.Flushed, st.Queued,
		int64(after.HeapAlloc-before.HeapAlloc)/1024)

	if st.Dropped == 0 && n > 50_000 {
		t.Log("warning: expected some drops with tiny queue under flood (may pass on very fast disks)")
	}

	// Also exercise drop_oldest once at smaller N.
	path2 := filepath.Join(t.TempDir(), "overflow_oldest.log")
	log2, err := core.New(polyglotFileConfig(path2, true, 1000, core.OverflowDropOldest))
	if err != nil {
		t.Fatal(err)
	}
	defer log2.Close()
	for i := 0; i < min(n, 50_000); i++ {
		_ = log2.Info("flood", RichFields(i))
	}
	_ = log2.Flush()
	st2 := log2.Stats()
	t.Logf("overflow drop_oldest flushed=%d dropped=%d", st2.Flushed, st2.Dropped)
}

func TestCompetitorSyncFloodReference(t *testing.T) {
	if os.Getenv("BENCH_FULL") != "1" {
		t.Skip("set BENCH_FULL=1 for competitor flood reference")
	}
	n := mustEnvInt("BENCH_OVERFLOW_N", 200_000)
	log, _ := newZapFile(t)
	start := time.Now()
	for i := 0; i < n; i++ {
		log.Info("flood", zapRich(i)...)
	}
	_ = log.Sync()
	t.Logf("zap sync flood n=%d elapsed=%s (not equivalent to async overflow)", n, time.Since(start))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
