package logger

import (
	"fmt"
	"os"
	"strings"
)

// quietDiagnostics is true when POLYGLOT_QUIET is set.
func quietDiagnostics() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("POLYGLOT_QUIET")))
	return v == "1" || v == "true" || v == "yes"
}

// MergeDiag describes how a merged config was produced (for startup diagnostics).
type MergeDiag struct {
	Resolve    ConfigResolveInfo
	HadFile    bool
	HadOverlay bool
	Config     Config
}

// PrintStartupDiagnostics writes a short stderr summary of config resolution.
func PrintStartupDiagnostics(d MergeDiag) {
	if quietDiagnostics() {
		return
	}

	fmt.Fprintf(os.Stderr, "polyglot-logger %s\n\n", Version)

	if d.HadFile && d.Resolve.Path != "" {
		fmt.Fprintf(os.Stderr, "config: %s\n", d.Resolve.Path)
	} else {
		fmt.Fprintf(os.Stderr, "config: (none)\n")
		if len(d.Resolve.Searched) > 0 {
			stop := ""
			if d.Resolve.StoppedAtGit {
				stop = "  (stopped at .git)"
			}
			fmt.Fprintf(os.Stderr, "searched: %s%s\n", strings.Join(d.Resolve.Searched, ", "), stop)
		}
		if d.Resolve.FromEnv && d.Resolve.EnvVar != "" {
			fmt.Fprintf(os.Stderr, "env %s set but file missing; using defaults\n", d.Resolve.EnvVar)
		}
	}

	fmt.Fprintf(os.Stderr, "service: %s\n", d.Config.Service)
	fmt.Fprintf(os.Stderr, "source: %s\n", sourceLabel(d))

	if d.Config.Stdout {
		fmt.Fprintf(os.Stderr, "stdout: enabled\n")
	} else {
		fmt.Fprintf(os.Stderr, "stdout: disabled\n")
	}
	if d.Config.FileEnabled() {
		fmt.Fprintf(os.Stderr, "file: enabled (%s)\n", d.Config.File.Path)
	}
	if d.Config.HTTPEnabled() {
		fmt.Fprintf(os.Stderr, "http: enabled (%s batch_size=%d flush_interval_ms=%d)\n",
			d.Config.HTTP.URL, d.Config.HTTP.BatchSize, d.Config.HTTP.FlushIntervalMS)
	}
	if d.Config.LokiEnabled() {
		fmt.Fprintf(os.Stderr, "loki: enabled (%s)\n", d.Config.Loki.URL)
	}
	fmt.Fprintln(os.Stderr)
}

func sourceLabel(d MergeDiag) string {
	switch {
	case d.HadFile && d.HadOverlay:
		return "YAML + constructor overrides"
	case d.HadFile:
		return "YAML"
	case d.HadOverlay:
		return "defaults + constructor overrides"
	default:
		return "defaults"
	}
}
