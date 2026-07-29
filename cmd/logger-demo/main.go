package main

import (
	"fmt"
	"os"

	"polyglot/internal/logger"
)

func main() {
	cfg := logger.DefaultConfig()
	cfg.Service = "logger-demo"
	cfg.Environment = "dev"
	cfg.Level = "debug"
	cfg.Stdout = true
	cfg.Async = true
	if len(os.Args) > 1 {
		cfg.File = &logger.FileConfig{
			Enabled:    true,
			Path:       os.Args[1],
			MaxSizeMB:  100,
			MaxBackups: 5,
			MaxAgeDays: 28,
		}
	}

	log, err := logger.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Close()

	_ = log.Info("logger demo started", map[string]any{
		"pid": os.Getpid(),
	})
	_ = log.Warn("sample warning", map[string]any{
		"code": "DEMO_WARN",
	})
	_ = log.Flush()
	fmt.Fprintln(os.Stderr, "demo log lines written; stats=", log.StatsJSON())
}
