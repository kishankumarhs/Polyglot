package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

doctor   Resolves config (POLYGLOT_CONFIG → cwd → parents, stop at .git),
         validates schema, prints checklist, optional HTTP probe.
validate Parses and validates a config file (strict).
`)
}

func flagConfig(args []string) string {
	for i := 0; i < len(args); i++ {
		if args[i] == "--config" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func runValidate(args []string) int {
	_ = os.Setenv("POLYLOG_STRICT", "1")
	_ = os.Setenv("POLYGLOT_QUIET", "1")
	resolve := logger.ResolveConfigPath(flagConfig(args))
	if resolve.Path == "" {
		fmt.Fprintln(os.Stderr, "no config path found (pass --config or set POLYGLOT_CONFIG)")
		return 1
	}
	cfg, err := logger.LoadConfigFromFile(resolve.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "validate failed: %v\n", err)
		return 1
	}
	fmt.Printf("OK %s\n", resolve.Path)
	fmt.Printf("  service=%s level=%s stdout=%v format=%s\n", cfg.Service, cfg.Level, cfg.Stdout, cfg.StdoutFormat)
	fmt.Printf("  file=%v http=%v loki=%v async=%v\n", cfg.FileEnabled(), cfg.HTTPEnabled(), cfg.LokiEnabled(), cfg.Async)
	return 0
}

func runDoctor(args []string) int {
	probe := false
	for _, a := range args {
		if a == "--probe-http" {
			probe = true
		}
	}

	_ = os.Setenv("POLYGLOT_QUIET", "1") // avoid double diagnostics from create path

	fmt.Println("polyglot doctor")
	fmt.Printf("Version %s (abi %d)\n\n", logger.Version, logger.ABIVersion)

	resolve := logger.ResolveConfigPath(flagConfig(args))
	ok := true

	if resolve.Path == "" {
		fmt.Println("✗ Config found")
		if len(resolve.Searched) > 0 {
			stop := ""
			if resolve.StoppedAtGit {
				stop = " (stopped at .git)"
			}
			fmt.Printf("  searched: %s%s\n", strings.Join(resolve.Searched, ", "), stop)
		}
		fmt.Println("  hint: set POLYGLOT_CONFIG or add polyglot.yaml at repo root")
		ok = false
	} else {
		fmt.Println("✓ Config found")
		fmt.Printf("  %s\n", resolve.Path)
		if resolve.FromEnv {
			fmt.Printf("  via %s\n", resolve.EnvVar)
		}
	}

	cfg, err := logger.LoadConfigFromFile(resolve.Path)
	if err != nil {
		fmt.Printf("✗ Parsed\n  %v\n", err)
		return 1
	}
	if err := cfg.Validate(); err != nil {
		fmt.Printf("✗ Parsed\n  %v\n", err)
		return 1
	}
	fmt.Println("✓ Parsed")
	fmt.Printf("  service=%s level=%s\n", cfg.Service, cfg.Level)

	if cfg.HTTPEnabled() {
		if probe {
			if err := probeURL(cfg.HTTP.URL); err != nil {
				fmt.Printf("✗ HTTP sink reachable\n  %v\n", err)
				ok = false
			} else {
				fmt.Println("✓ HTTP sink reachable")
				fmt.Printf("  %s (POST may still require auth)\n", cfg.HTTP.URL)
			}
		} else {
			fmt.Println("· HTTP sink configured (pass --probe-http to check reachability)")
			fmt.Printf("  %s batch_size=%d flush_interval_ms=%d\n",
				cfg.HTTP.URL, cfg.HTTP.BatchSize, cfg.HTTP.FlushIntervalMS)
		}
	}

	if cfg.FileEnabled() {
		if err := checkFileWritable(cfg.File.Path); err != nil {
			fmt.Printf("✗ Log file writable\n  %v\n", err)
			ok = false
		} else {
			fmt.Println("✓ Log file writable")
			fmt.Printf("  %s\n", cfg.File.Path)
		}
	}

	fmt.Println("  native lib search:")
	foundNative := false
	for _, c := range nativeCandidates() {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			fmt.Printf("    FOUND %s\n", c)
			foundNative = true
		}
	}
	if !foundNative {
		fmt.Println("    (none in common paths — bindings bundle natives in published packages)")
	}

	log, err := logger.New(cfg)
	if err != nil {
		fmt.Printf("✗ Runtime initialized\n  %v\n", err)
		return 1
	}
	_ = log.Info("polyglot doctor ok", map[string]any{"doctor": true})
	if err := log.Flush(); err != nil {
		fmt.Printf("  WARN flush: %v\n", err)
	}
	st := log.Stats()
	b, _ := json.Marshal(st)
	fmt.Printf("  stats: %s\n", string(b))
	if err := log.Close(); err != nil {
		fmt.Printf("  WARN close: %v\n", err)
	}
	fmt.Println("✓ Runtime initialized")

	fmt.Println()
	if ok {
		fmt.Println("doctor: OK")
		return 0
	}
	fmt.Println("doctor: issues found")
	return 1
}

func checkFileWritable(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	return f.Close()
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
