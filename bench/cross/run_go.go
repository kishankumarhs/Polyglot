package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	core "polyglot/internal/logger"
)

func main() {
	n := 100_000
	if v := os.Getenv("BENCH_CROSS_N"); v != "" {
		fmt.Sscanf(v, "%d", &n)
	}
	dir := os.TempDir()
	path := filepath.Join(dir, "polyglot-cross-go.log")
	_ = os.Remove(path)
	log, err := core.New(core.Config{
		Service:   "bench-cross",
		Level:     "info",
		Stdout:    false,
		Async:     false,
		QueueSize: 10000,
		Overflow:  core.OverflowDropNewest,
		File: &core.FileConfig{
			Enabled: true, Path: path, MaxSizeMB: 200, MaxBackups: 1, MaxAgeDays: 1,
		},
	})
	if err != nil {
		fatal(err)
	}
	fields := map[string]any{
		"user_id": 7, "trace_id": "abc123", "span_id": "span-1",
		"service": "payments", "region": "us-east-1", "latency_ms": 12.4,
		"ok": true, "tags": []string{"a", "b", "c"},
		"meta": map[string]any{"cart": map[string]any{"items": 3, "currency": "USD"}},
		"error": "optional message",
	}
	start := time.Now()
	for i := 0; i < n; i++ {
		fields["n"] = i
		_ = log.Info("checkout", fields)
	}
	_ = log.Flush()
	_ = log.Close()
	elapsed := time.Since(start)

	line, _ := os.ReadFile(path)
	var first map[string]any
	for _, l := range splitLines(line) {
		if len(l) == 0 {
			continue
		}
		_ = json.Unmarshal(l, &first)
		break
	}
	fmt.Printf("lang=go n=%d elapsed=%s ops_s=%.0f path=%s schema_keys=%v\n",
		n, elapsed, float64(n)/elapsed.Seconds(), path, keys(first))
}

func splitLines(b []byte) [][]byte {
	var out [][]byte
	start := 0
	for i, c := range b {
		if c == '\n' {
			out = append(out, b[start:i])
			start = i + 1
		}
	}
	if start < len(b) {
		out = append(out, b[start:])
	}
	return out
}

func keys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
