package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"polyglot/internal/logger"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "doctor":
		os.Exit(runDoctor(os.Args[2:]))
	case "validate":
		os.Exit(runValidate(os.Args[2:]))
	case "version":
		fmt.Println(logger.Version)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `polyglot — developer tooling for Polyglot Logger

Usage:
  polyglot doctor [--config path] [--probe-http]
  polyglot validate [--config path]
  polyglot version

doctor   Resolves config, validates schema, prints sink summary, optional HTTP probe.
validate Parses and validates a config file (strict).
`)
}

func configPath(args []string) string {
	for i := 0; i < len(args); i++ {
		if args[i] == "--config" && i+1 < len(args) {
			return args[i+1]
		}
	}
	if p := os.Getenv("POLYGLOT_CONFIG_PATH"); p != "" {
		return p
	}
	if p := os.Getenv("POLYGLOT_CONFIG_FILE"); p != "" {
		return p
	}
	for _, name := range []string{"polyglot.yaml", "polyglot.yml", "polyglot.json"} {
		if _, err := os.Stat(name); err == nil {
			return name
		}
	}
	return ""
}

func runValidate(args []string) int {
	path := configPath(args)
	if path == "" {
		fmt.Fprintln(os.Stderr, "no config path found (pass --config or set POLYGLOT_CONFIG_PATH)")
		return 1
	}
	_ = os.Setenv("POLYLOG_STRICT", "1")
	cfg, err := logger.LoadConfigFromFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "validate failed: %v\n", err)
		return 1
	}
	fmt.Printf("OK %s\n", path)
	fmt.Printf("  service=%s level=%s stdout=%v format=%s\n", cfg.Service, cfg.Level, cfg.Stdout, cfg.StdoutFormat)
	fmt.Printf("  file=%v http=%v loki=%v async=%v\n", cfg.FileEnabled(), cfg.HTTPEnabled(), cfg.LokiEnabled(), cfg.Async)
	return 0
}

func runDoctor(args []string) int {
	path := configPath(args)
	probe := false
	for _, a := range args {
		if a == "--probe-http" {
			probe = true
		}
	}

	fmt.Println("Polyglot doctor")
	fmt.Printf("  version: %s (abi %d)\n", logger.Version, logger.ABIVersion)

	if path == "" {
		fmt.Println("  config: (none found — defaults would be used unless POLYLOG_STRICT=1)")
	} else {
		abs, _ := filepath.Abs(path)
		fmt.Printf("  config: %s\n", abs)
	}

	cfg, err := logger.LoadConfigFromFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ERROR loading config: %v\n", err)
		return 1
	}
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "  ERROR validating config: %v\n", err)
		return 1
	}

	fmt.Printf("  service: %s\n", cfg.Service)
	fmt.Printf("  level: %s\n", cfg.Level)
	fmt.Printf("  sinks: stdout=%v file=%v http=%v loki=%v\n", cfg.Stdout, cfg.FileEnabled(), cfg.HTTPEnabled(), cfg.LokiEnabled())
	if cfg.Caller {
		fmt.Println("  caller: enabled")
	}
	if cfg.Sampling != nil && cfg.Sampling.Enabled {
		fmt.Printf("  sampling: initial=%d thereafter=%d\n", cfg.Sampling.Initial, cfg.Sampling.Thereafter)
	}

	// Native library discovery hints for binding users.
	fmt.Println("  native lib search:")
	for _, c := range nativeCandidates() {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			fmt.Printf("    FOUND %s\n", c)
		} else {
			fmt.Printf("    miss  %s\n", c)
		}
	}

	if probe {
		if cfg.HTTPEnabled() {
			if err := probeURL(cfg.HTTP.URL); err != nil {
				fmt.Fprintf(os.Stderr, "  http probe FAILED: %v\n", err)
			} else {
				fmt.Println("  http probe: reachable (POST may still require auth)")
			}
		}
		if cfg.LokiEnabled() {
			if err := probeURL(cfg.Loki.URL); err != nil {
				fmt.Fprintf(os.Stderr, "  loki probe FAILED: %v\n", err)
			} else {
				fmt.Println("  loki probe: reachable (push may still require auth)")
			}
		}
	}

	// Smoke create + one log + close.
	log, err := logger.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ERROR creating logger: %v\n", err)
		return 1
	}
	_ = log.Info("polyglot doctor ok", map[string]any{"doctor": true})
	if err := log.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "  WARN flush: %v\n", err)
	}
	st := log.Stats()
	b, _ := json.Marshal(st)
	fmt.Printf("  stats: %s\n", string(b))
	if err := log.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "  WARN close: %v\n", err)
	}
	fmt.Println("doctor: OK")
	return 0
}

func nativeCandidates() []string {
	env := os.Getenv("POLYGLOT_LOGGER_LIB")
	names := []string{"logger.dll", "liblogger.so", "liblogger.dylib"}
	var out []string
	if env != "" {
		out = append(out, env)
	}
	roots := []string{".", "dist", "build", filepath.Join("bindings", "node", "native"), filepath.Join("bindings", "python", "polyglot_logger", "native")}
	for _, root := range roots {
		for _, name := range names {
			out = append(out, filepath.Join(root, name))
		}
	}
	return out
}

func probeURL(raw string) error {
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, raw, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 500 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}
