package logger

import (
	"os"
	"path/filepath"
	"strings"
)

// ConfigResolveInfo describes how a config path was found (or not).
type ConfigResolveInfo struct {
	Path         string   // absolute path, or empty if none
	Searched     []string // directories examined during walk (cwd → parents)
	StoppedAtGit bool
	FromEnv      bool
	EnvVar       string // which env var supplied the path, if any
}

// ExplicitConfigEnv returns the first non-empty config path from environment.
// Order: POLYGLOT_CONFIG, POLYGLOT_CONFIG_PATH, POLYGLOT_CONFIG_FILE.
func ExplicitConfigEnv() (path string, envVar string) {
	for _, key := range []string{"POLYGLOT_CONFIG", "POLYGLOT_CONFIG_PATH", "POLYGLOT_CONFIG_FILE"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v, key
		}
	}
	return "", ""
}

// ResolveConfigPath finds polyglot.yaml / .yml / .json.
//
// Resolution order:
//  1. explicitPath if non-empty
//  2. POLYGLOT_CONFIG / POLYGLOT_CONFIG_PATH / POLYGLOT_CONFIG_FILE
//  3. cwd, then parent directories until found, .git directory, or filesystem root
func ResolveConfigPath(explicitPath string) ConfigResolveInfo {
	info := ConfigResolveInfo{}

	if p := strings.TrimSpace(explicitPath); p != "" {
		abs, err := filepath.Abs(p)
		if err != nil {
			abs = p
		}
		info.Path = abs
		info.FromEnv = false
		return info
	}

	if envPath, key := ExplicitConfigEnv(); envPath != "" {
		abs, err := filepath.Abs(envPath)
		if err != nil {
			abs = envPath
		}
		info.Path = abs
		info.FromEnv = true
		info.EnvVar = key
		return info
	}

	cwd, err := os.Getwd()
	if err != nil {
		return info
	}

	current := cwd
	for {
		info.Searched = append(info.Searched, current)
		for _, name := range []string{"polyglot.yaml", "polyglot.yml", "polyglot.json"} {
			candidate := filepath.Join(current, name)
			if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
				abs, _ := filepath.Abs(candidate)
				info.Path = abs
				return info
			}
		}

		// Stop at git repo root (do not walk outside the project).
		if st, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			_ = st
			info.StoppedAtGit = true
			return info
		}

		parent := filepath.Dir(current)
		if parent == current {
			return info
		}
		current = parent
	}
}
