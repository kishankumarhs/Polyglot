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
	// QuietOverride comes from the caller's overlay. Bindings pass it because Go
	// snapshots the environment at startup on Unix, so a POLYGLOT_QUIET set after
	// process start is invisible to os.Getenv here.
	QuietOverride *bool
}

// PrintStartupDiagnostics writes a short stderr summary of config resolution.
func PrintStartupDiagnostics(d MergeDiag) {
	if d.QuietOverride != nil {
		if *d.QuietOverride {
			return
		}
	} else if quietDiagnostics() {
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
	if d.Config.OTLPEnabled() {
		fmt.Fprintf(os.Stderr, "otlp: enabled (%s batch_size=%d flush_interval_ms=%d)\n",
			d.Config.OTLP.URL, d.Config.OTLP.BatchSize, d.Config.OTLP.FlushIntervalMS)
	}
	if d.Config.KafkaEnabled() {
		fmt.Fprintf(os.Stderr, "kafka: enabled (topic=%s brokers=%d)\n", d.Config.Kafka.Topic, len(d.Config.Kafka.Brokers))
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
