package logger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveConfigPathEnvWins(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "custom.yaml")
	if err := os.WriteFile(cfgPath, []byte("service: from-env\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("POLYGLOT_CONFIG", cfgPath)
	t.Setenv("POLYGLOT_CONFIG_PATH", "")
	t.Setenv("POLYGLOT_CONFIG_FILE", "")

	info := ResolveConfigPath("")
	if info.Path != cfgPath && filepath.Clean(info.Path) != filepath.Clean(cfgPath) {
		// Abs may normalize; compare via same abs
		abs, _ := filepath.Abs(cfgPath)
		if info.Path != abs {
			t.Fatalf("path=%q want %q", info.Path, abs)
		}
	}
	if !info.FromEnv || info.EnvVar != "POLYGLOT_CONFIG" {
		t.Fatalf("expected FromEnv POLYGLOT_CONFIG, got %+v", info)
	}
}

func TestResolveConfigPathWalkAndGitStop(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "apps", "orders")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	// Config at repo root
	cfgPath := filepath.Join(root, "polyglot.yaml")
	if err := os.WriteFile(cfgPath, []byte("service: orders\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Outside repo (parent of root) — must not be found when cwd is nested
	outside := filepath.Join(filepath.Dir(root), "polyglot.yaml")
	_ = os.WriteFile(outside, []byte("service: outside\n"), 0o644)
	t.Cleanup(func() { _ = os.Remove(outside) })

	t.Setenv("POLYGLOT_CONFIG", "")
	t.Setenv("POLYGLOT_CONFIG_PATH", "")
	t.Setenv("POLYGLOT_CONFIG_FILE", "")

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(nested); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	info := ResolveConfigPath("")
	absWant, _ := filepath.Abs(cfgPath)
	if info.Path != absWant {
		t.Fatalf("path=%q want %q searched=%v", info.Path, absWant, info.Searched)
	}
}

func TestResolveConfigPathStopsAtGitWithoutConfig(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "apps")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Config only outside the repo
	outsideDir := t.TempDir() // different tree
	_ = outsideDir

	t.Setenv("POLYGLOT_CONFIG", "")
	t.Setenv("POLYGLOT_CONFIG_PATH", "")
	t.Setenv("POLYGLOT_CONFIG_FILE", "")

	oldWD, _ := os.Getwd()
	if err := os.Chdir(nested); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	info := ResolveConfigPath("")
	if info.Path != "" {
		t.Fatalf("expected no config, got %q", info.Path)
	}
	if !info.StoppedAtGit {
		t.Fatalf("expected StoppedAtGit, searched=%v", info.Searched)
	}
}

func TestCreateConfigMergeOverlay(t *testing.T) {
	t.Setenv("POLYGLOT_QUIET", "1")
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "polyglot.yaml")
	body := `
service: from-yaml
level: debug
stdout: false
http:
  enabled: true
  url: http://localhost:9999
  batch_size: 10
  flush_interval_ms: 500
`
	if err := os.WriteFile(cfgPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, diag, err := CreateConfigFromFileWithOverrides(cfgPath, []byte(`{"service":"from-ctor","level":"info"}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Service != "from-ctor" {
		t.Fatalf("service=%q", cfg.Service)
	}
	if cfg.Level != "info" {
		t.Fatalf("level=%q", cfg.Level)
	}
	if !cfg.HTTPEnabled() || cfg.HTTP.URL != "http://localhost:9999" {
		t.Fatalf("http not preserved: %+v", cfg.HTTP)
	}
	if cfg.HTTP.BatchSize != 10 {
		t.Fatalf("batch_size=%d", cfg.HTTP.BatchSize)
	}
	if !diag.HadFile || !diag.HadOverlay {
		t.Fatalf("diag=%+v", diag)
	}
}

func TestCreateConfigServiceFromYAMLOnly(t *testing.T) {
	t.Setenv("POLYGLOT_QUIET", "1")
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "polyglot.yaml")
	if err := os.WriteFile(cfgPath, []byte("service: yaml-svc\nstdout: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := CreateConfigFromFileWithOverrides(cfgPath, []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Service != "yaml-svc" {
		t.Fatalf("service=%q", cfg.Service)
	}
}

func TestCreateConfigOverlayQuietSuppressesDiagnostics(t *testing.T) {
	// Bindings pass quiet in the overlay because Go snapshots the environment at
	// startup on Unix; POLYGLOT_QUIET set after start would not be seen here.
	t.Setenv("POLYGLOT_QUIET", "")
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "polyglot.yaml")
	if err := os.WriteFile(cfgPath, []byte("service: quiet-svc\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	origStderr := os.Stderr
	os.Stderr = w

	_, _, err = CreateConfigFromFileWithOverrides(cfgPath, []byte(`{"quiet":true}`))

	os.Stderr = origStderr
	_ = w.Close()
	var buf [4096]byte
	n, _ := r.Read(buf[:])
	_ = r.Close()

	if err != nil {
		t.Fatal(err)
	}
	if n > 0 {
		t.Fatalf("expected no diagnostics, got %q", string(buf[:n]))
	}
}

func TestCreateConfigEmptyServiceFails(t *testing.T) {
	t.Setenv("POLYGLOT_QUIET", "1")
	_, _, err := CreateConfigFromFileWithOverrides("", []byte(`{"service":""}`))
	if err == nil {
		t.Fatal("expected error for empty service")
	}
}
