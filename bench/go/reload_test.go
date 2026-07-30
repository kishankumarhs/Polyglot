package bench

import (
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	core "polyglot/internal/logger"
)

func TestHotReloadUnderLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reload.log")
	cfg := polyglotFileConfig(path, true, 1000, core.OverflowDropNewest)
	cfg.Level = "debug"
	log, err := core.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	workers := mustEnvInt("BENCH_RELOAD_WORKERS", 100)
	var stop atomic.Bool
	var logged atomic.Int64
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			child := log.With(map[string]any{"worker": id})
			i := 0
			for !stop.Load() {
				_ = child.Info("steady", RichFields(i))
				logged.Add(1)
				i++
				// Yield so the async worker can service reloadCh under load.
				if i%64 == 0 {
					time.Sleep(time.Microsecond)
				}
			}
		}(w)
	}

	time.Sleep(100 * time.Millisecond)
	reloadCfg := cfg
	reloadCfg.Level = "error"
	reloadStart := time.Now()
	errCh := make(chan error, 1)
	go func() {
		errCh <- log.ReloadConfig(reloadCfg)
	}()
	select {
	case err := <-errCh:
		if err != nil {
			stop.Store(true)
			wg.Wait()
			t.Fatalf("reload under load: %v", err)
		}
	case <-time.After(15 * time.Second):
		stop.Store(true)
		wg.Wait()
		t.Fatal("reload under load timed out after 15s (control channel starved?)")
	}
	reloadUnderLoad := time.Since(reloadStart)

	time.Sleep(50 * time.Millisecond)
	stop.Store(true)
	wg.Wait()
	if err := log.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}
	st := log.Stats()
	t.Logf("reload_under_load=%s workers=%d logged=%d flushed=%d dropped=%d",
		reloadUnderLoad, workers, logged.Load(), st.Flushed, st.Dropped)

	// Quiescent reload latency (capability floor).
	quiet := cfg
	quiet.Level = "warn"
	qStart := time.Now()
	if err := log.ReloadConfig(quiet); err != nil {
		t.Fatalf("quiet reload: %v", err)
	}
	t.Logf("reload_quiescent=%s", time.Since(qStart))
}
