package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Overflow policies for the async queue.
const (
	OverflowDropNewest = "drop_newest"
	OverflowDropOldest = "drop_oldest"
	OverflowBlock      = "block"
)

// StdoutFormat controls how the stdout sink renders lines.
const (
	StdoutFormatJSON = "json"
	StdoutFormatText = "text"
)

// FileConfig configures the rotating file sink.
type FileConfig struct {
	Enabled    bool   `json:"enabled"`
	Path       string `json:"path"`
	MaxSizeMB  int    `json:"maxSizeMB"`
	MaxBackups int    `json:"maxBackups"`
	MaxAgeDays int    `json:"maxAgeDays"`
	Compress   bool   `json:"compress"`
}

// HTTPConfig configures the centralized HTTP log sink (NDJSON POST body).
type HTTPConfig struct {
	Enabled         bool              `json:"enabled"`
	URL             string            `json:"url"`
	TimeoutMS       int               `json:"timeout_ms"`
	Headers         map[string]string `json:"headers"`
	BatchSize       int               `json:"batch_size"`
	FlushIntervalMS int               `json:"flush_interval_ms"`
}

// LokiConfig configures a native Grafana Loki push sink.
type LokiConfig struct {
	Enabled         bool              `json:"enabled"`
	URL             string            `json:"url"` // e.g. http://loki:3100/loki/api/v1/push
	TimeoutMS       int               `json:"timeout_ms"`
	Headers         map[string]string `json:"headers"`
	BatchSize       int               `json:"batch_size"`
	FlushIntervalMS int               `json:"flush_interval_ms"`
	// Labels are static Loki stream labels. service_name/level/environment are
	// added automatically from each log entry when not already present.
	Labels map[string]string `json:"labels"`
}

// SamplingConfig is zap-style first-N-then-every sampling per level+message.
type SamplingConfig struct {
	Enabled    bool `json:"enabled"`
	Initial    int  `json:"initial"`    // always emit the first N matching logs
	Thereafter int  `json:"thereafter"` // then emit 1 of every Thereafter
}

// Config controls logger behavior.
type Config struct {
	Service        string          `json:"service"`
	ServiceVersion string          `json:"service_version,omitempty"`
	Environment    string          `json:"environment,omitempty"`
	Level          string          `json:"level"`
	Stdout         bool            `json:"stdout"`
	StdoutFormat   string          `json:"stdout_format,omitempty"` // json (default) | text
	Caller         bool            `json:"caller,omitempty"`
	Strict         bool            `json:"strict,omitempty"`
	File           *FileConfig     `json:"file,omitempty"`
	HTTP           *HTTPConfig     `json:"http,omitempty"`
	Loki           *LokiConfig     `json:"loki,omitempty"`
	Async          bool            `json:"async"`
	QueueSize      int             `json:"queueSize"`
	Overflow       string          `json:"overflow"`
	Sampling       *SamplingConfig `json:"sampling,omitempty"`
	Fields         map[string]any  `json:"fields,omitempty"`
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		Service:      "app",
		Level:        "info",
		Stdout:       true,
		StdoutFormat: StdoutFormatJSON,
		Async:        true,
		QueueSize:    10000,
		Overflow:     OverflowDropNewest,
		File: &FileConfig{
			Enabled:    false,
			MaxSizeMB:  100,
			MaxBackups: 10,
			MaxAgeDays: 30,
			Compress:   false,
		},
		HTTP: &HTTPConfig{
			Enabled:         false,
			TimeoutMS:       5000,
			BatchSize:       50,
			FlushIntervalMS: 1000,
			Headers:         map[string]string{},
		},
		Loki: &LokiConfig{
			Enabled:         false,
			TimeoutMS:       5000,
			BatchSize:       50,
			FlushIntervalMS: 1000,
			Headers:         map[string]string{},
			Labels:          map[string]string{},
		},
		Fields: map[string]any{},
	}
}

type rawConfig struct {
	Service        *string         `json:"service"`
	ServiceName    *string         `json:"service_name"`
	ServiceVersion string          `json:"service_version"`
	Environment    string          `json:"environment"`
	Level          *string         `json:"level"`
	MinLevel       *string         `json:"min_level"`
	Stdout         *bool           `json:"stdout"`
	StdoutFormat   string          `json:"stdout_format"`
	Caller         *bool           `json:"caller"`
	Strict         *bool           `json:"strict"`
	File           json.RawMessage `json:"file"`
	FilePath       string          `json:"file_path"`
	MaxSizeMB      int             `json:"max_size_mb"`
	MaxBackups     int             `json:"max_backups"`
	MaxAgeDays     int             `json:"max_age_days"`
	HTTP           *HTTPConfig     `json:"http"`
	Loki           *LokiConfig     `json:"loki"`
	Async          *bool           `json:"async"`
	QueueSize      int             `json:"queueSize"`
	Overflow       string          `json:"overflow"`
	Sampling       *SamplingConfig `json:"sampling"`
	Fields         map[string]any  `json:"fields"`
}

// applyRawOverlay merges sparse raw fields onto base (only present keys win).
func applyRawOverlay(base Config, raw rawConfig) (Config, error) {
	cfg := base

	if raw.Service != nil {
		cfg.Service = *raw.Service
	} else if raw.ServiceName != nil {
		cfg.Service = *raw.ServiceName
	}
	if raw.ServiceVersion != "" {
		cfg.ServiceVersion = raw.ServiceVersion
	}
	if raw.Environment != "" {
		cfg.Environment = raw.Environment
	}
	if raw.Level != nil {
		cfg.Level = *raw.Level
	} else if raw.MinLevel != nil {
		cfg.Level = *raw.MinLevel
	}
	if raw.Stdout != nil {
		cfg.Stdout = *raw.Stdout
	}
	if raw.StdoutFormat != "" {
		cfg.StdoutFormat = raw.StdoutFormat
	}
	if raw.Caller != nil {
		cfg.Caller = *raw.Caller
	}
	if raw.Strict != nil {
		cfg.Strict = *raw.Strict
	}
	if raw.Async != nil {
		cfg.Async = *raw.Async
	}
	if raw.QueueSize > 0 {
		cfg.QueueSize = raw.QueueSize
	}
	if raw.Overflow != "" {
		cfg.Overflow = raw.Overflow
	}
	if raw.Fields != nil {
		cfg.Fields = raw.Fields
	}
	if raw.Sampling != nil {
		cfg.Sampling = raw.Sampling
	}

	if len(raw.File) > 0 && string(raw.File) != "null" {
		var fc FileConfig
		if err := json.Unmarshal(raw.File, &fc); err != nil {
			return Config{}, fmt.Errorf("invalid file config: %w", err)
		}
		cfg.File = &fc
	} else if raw.FilePath != "" {
		fc := FileConfig{
			Enabled:    true,
			Path:       raw.FilePath,
			MaxSizeMB:  100,
			MaxBackups: 10,
			MaxAgeDays: 30,
		}
		if raw.MaxSizeMB > 0 {
			fc.MaxSizeMB = raw.MaxSizeMB
		}
		if raw.MaxBackups > 0 {
			fc.MaxBackups = raw.MaxBackups
		}
		if raw.MaxAgeDays > 0 {
			fc.MaxAgeDays = raw.MaxAgeDays
		}
		cfg.File = &fc
	}

	if raw.HTTP != nil {
		cfg.HTTP = raw.HTTP
	}
	if raw.Loki != nil {
		cfg.Loki = raw.Loki
	}

	return cfg, nil
}

// ParseConfigJSON parses and validates configuration from JSON.
func ParseConfigJSON(data []byte) (Config, error) {
	cfg := DefaultConfig()
	if len(data) == 0 {
		return cfg, nil
	}

	var raw rawConfig
	if err := json.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("invalid config json: %w", err)
	}

	cfg, err := applyRawOverlay(cfg, raw)
	if err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// ApplyConfigOverlay merges sparse overlay JSON onto base. Empty overlay is a no-op.
func ApplyConfigOverlay(base Config, overlayJSON []byte) (Config, error) {
	trimmed := strings.TrimSpace(string(overlayJSON))
	if trimmed == "" || trimmed == "{}" || trimmed == "null" {
		return base, nil
	}
	var raw rawConfig
	if err := json.Unmarshal(overlayJSON, &raw); err != nil {
		return Config{}, fmt.Errorf("invalid overlay json: %w", err)
	}
	return applyRawOverlay(base, raw)
}

// hasOverlayKeys reports whether overlay JSON sets any config fields.
func hasOverlayKeys(overlayJSON []byte) bool {
	m := overlayKeys(overlayJSON)
	for k := range m {
		if k != "quiet" {
			return true
		}
	}
	return false
}

func overlayKeys(overlayJSON []byte) map[string]json.RawMessage {
	trimmed := strings.TrimSpace(string(overlayJSON))
	if trimmed == "" || trimmed == "{}" || trimmed == "null" {
		return nil
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(overlayJSON, &m); err != nil {
		return nil
	}
	return m
}

// overlayQuiet reads the caller-supplied "quiet" flag. Bindings send this because
// Go snapshots the environment at startup on Unix, so POLYGLOT_QUIET set after
// process start would otherwise be ignored.
func overlayQuiet(overlayJSON []byte) *bool {
	raw, ok := overlayKeys(overlayJSON)["quiet"]
	if !ok {
		return nil
	}
	var v bool
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil
	}
	return &v
}

// CreateConfigFromFileWithOverrides loads a config file (or discovers one),
// applies sparse constructor overlay JSON, validates, and optionally prints diagnostics.
//
// If explicitPath is empty, uses ResolveConfigPath (env → cwd → parents, stop at .git).
func CreateConfigFromFileWithOverrides(explicitPath string, overlayJSON []byte) (Config, MergeDiag, error) {
	resolve := ResolveConfigPath(explicitPath)
	diag := MergeDiag{
		Resolve:       resolve,
		HadOverlay:    hasOverlayKeys(overlayJSON),
		QuietOverride: overlayQuiet(overlayJSON),
	}

	var base Config
	var err error
	if resolve.Path != "" {
		if _, statErr := os.Stat(resolve.Path); statErr == nil {
			base, err = loadConfigFileExact(resolve.Path)
			if err != nil {
				return Config{}, diag, err
			}
			diag.HadFile = true
		} else if os.IsNotExist(statErr) {
			if strictRequested(false) {
				return Config{}, diag, fmt.Errorf("strict mode: config file not found: %s", resolve.Path)
			}
			fmt.Fprintf(os.Stderr, "[polyglot-logger] config file not found: %s; using defaults\n", resolve.Path)
			base = DefaultConfig()
			diag.HadFile = false
		} else {
			return Config{}, diag, fmt.Errorf("failed to stat config file %s: %w", resolve.Path, statErr)
		}
	} else {
		if strictRequested(false) {
			return Config{}, diag, fmt.Errorf("strict mode: no config path (set POLYGLOT_CONFIG or pass a file)")
		}
		base = DefaultConfig()
	}

	cfg, err := ApplyConfigOverlay(base, overlayJSON)
	if err != nil {
		return Config{}, diag, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, diag, err
	}
	diag.Config = cfg
	PrintStartupDiagnostics(diag)
	return cfg, diag, nil
}

// loadConfigFileExact reads and parses a config file at an exact path (no env fallback).
func loadConfigFileExact(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if strictRequested(false) {
				return Config{}, fmt.Errorf("strict mode: config file not found: %s", path)
			}
			fmt.Fprintf(os.Stderr, "[polyglot-logger] config file not found: %s; using defaults\n", path)
			return DefaultConfig(), nil
		}
		return Config{}, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	lower := strings.ToLower(path)
	var cfg Config
	if strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml") {
		cfg, err = ParseConfigYAML(data)
	} else {
		cfg, err = ParseConfigJSON(data)
	}
	if err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate checks configuration values and applies defaults.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.Service) == "" {
		return fmt.Errorf("service is required")
	}
	if _, err := ParseLevel(c.Level); err != nil {
		return err
	}
	switch strings.ToLower(strings.TrimSpace(c.Overflow)) {
	case "", OverflowDropNewest:
		c.Overflow = OverflowDropNewest
	case OverflowDropOldest, OverflowBlock:
		c.Overflow = strings.ToLower(strings.TrimSpace(c.Overflow))
	default:
		return fmt.Errorf("overflow must be drop_newest, drop_oldest, or block")
	}
	switch strings.ToLower(strings.TrimSpace(c.StdoutFormat)) {
	case "", StdoutFormatJSON:
		c.StdoutFormat = StdoutFormatJSON
	case StdoutFormatText:
		c.StdoutFormat = StdoutFormatText
	default:
		return fmt.Errorf("stdout_format must be json or text")
	}
	if c.QueueSize < 0 {
		return fmt.Errorf("queueSize must be >= 0")
	}
	if c.QueueSize == 0 {
		c.QueueSize = 10000
	}
	if c.Fields == nil {
		c.Fields = map[string]any{}
	}
	if c.Sampling != nil && c.Sampling.Enabled {
		if c.Sampling.Initial < 0 {
			return fmt.Errorf("sampling.initial must be >= 0")
		}
		if c.Sampling.Thereafter <= 0 {
			c.Sampling.Thereafter = 100
		}
		if c.Sampling.Initial == 0 {
			c.Sampling.Initial = 100
		}
	}

	fileEnabled := c.FileEnabled()
	httpEnabled := c.HTTPEnabled()
	lokiEnabled := c.LokiEnabled()
	if !c.Stdout && !fileEnabled && !httpEnabled && !lokiEnabled {
		return fmt.Errorf("at least one sink must be enabled (stdout, file, http, or loki)")
	}

	if c.File == nil {
		c.File = &FileConfig{MaxSizeMB: 100, MaxBackups: 10, MaxAgeDays: 30}
	}
	if c.File.MaxSizeMB < 0 {
		return fmt.Errorf("file.maxSizeMB must be >= 0")
	}
	if c.File.MaxSizeMB == 0 {
		c.File.MaxSizeMB = 100
	}
	if c.File.MaxBackups < 0 {
		return fmt.Errorf("file.maxBackups must be >= 0")
	}
	if c.File.MaxAgeDays < 0 {
		return fmt.Errorf("file.maxAgeDays must be >= 0")
	}
	if fileEnabled && strings.TrimSpace(c.File.Path) == "" {
		return fmt.Errorf("file.path is required when file sink is enabled")
	}

	if c.HTTP == nil {
		c.HTTP = &HTTPConfig{
			TimeoutMS:       5000,
			BatchSize:       50,
			FlushIntervalMS: 1000,
			Headers:         map[string]string{},
		}
	}
	if c.HTTP.Headers == nil {
		c.HTTP.Headers = map[string]string{}
	}
	if httpEnabled {
		if strings.TrimSpace(c.HTTP.URL) == "" {
			return fmt.Errorf("http.url is required when http sink is enabled")
		}
		if err := validateCollectorURL(c.HTTP.URL); err != nil {
			return err
		}
		if c.HTTP.TimeoutMS <= 0 {
			c.HTTP.TimeoutMS = 5000
		}
		if c.HTTP.BatchSize <= 0 {
			c.HTTP.BatchSize = 50
		}
		if c.HTTP.FlushIntervalMS <= 0 {
			c.HTTP.FlushIntervalMS = 1000
		}
	}

	if c.Loki == nil {
		c.Loki = &LokiConfig{
			TimeoutMS:       5000,
			BatchSize:       50,
			FlushIntervalMS: 1000,
			Headers:         map[string]string{},
			Labels:          map[string]string{},
		}
	}
	if c.Loki.Headers == nil {
		c.Loki.Headers = map[string]string{}
	}
	if c.Loki.Labels == nil {
		c.Loki.Labels = map[string]string{}
	}
	if lokiEnabled {
		if strings.TrimSpace(c.Loki.URL) == "" {
			return fmt.Errorf("loki.url is required when loki sink is enabled")
		}
		if err := validateCollectorURL(c.Loki.URL); err != nil {
			return err
		}
		if c.Loki.TimeoutMS <= 0 {
			c.Loki.TimeoutMS = 5000
		}
		if c.Loki.BatchSize <= 0 {
			c.Loki.BatchSize = 50
		}
		if c.Loki.FlushIntervalMS <= 0 {
			c.Loki.FlushIntervalMS = 1000
		}
	}
	return nil
}

// FileEnabled reports whether the file sink should be active.
func (c Config) FileEnabled() bool {
	if c.File == nil {
		return false
	}
	if c.File.Enabled {
		return strings.TrimSpace(c.File.Path) != ""
	}
	return false
}

// HTTPEnabled reports whether the HTTP sink should be active.
func (c Config) HTTPEnabled() bool {
	return c.HTTP != nil && c.HTTP.Enabled && strings.TrimSpace(c.HTTP.URL) != ""
}

// LokiEnabled reports whether the Loki sink should be active.
func (c Config) LokiEnabled() bool {
	return c.Loki != nil && c.Loki.Enabled && strings.TrimSpace(c.Loki.URL) != ""
}

// MinLevelValue returns the parsed minimum level.
func (c Config) MinLevelValue() Level {
	level, err := ParseLevel(c.Level)
	if err != nil {
		return LevelInfo
	}
	return level
}

// EnsureFileDir creates the parent directory for the log file if needed.
func (c Config) EnsureFileDir() error {
	if !c.FileEnabled() {
		return nil
	}
	dir := parentDir(c.File.Path)
	if dir == "" || dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

func parentDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return ""
}

// ParseConfigYAML parses and validates configuration from YAML bytes.
func ParseConfigYAML(data []byte) (Config, error) {
	if len(data) == 0 {
		return DefaultConfig(), nil
	}
	var yamlData interface{}
	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		return Config{}, fmt.Errorf("invalid config yaml: %w", err)
	}
	jsonData, err := json.Marshal(yamlData)
	if err != nil {
		return Config{}, fmt.Errorf("failed to convert yaml to json: %w", err)
	}
	return ParseConfigJSON(jsonData)
}

// strictRequested is true when config or environment demands fail-fast loading.
func strictRequested(cfgStrict bool) bool {
	if cfgStrict {
		return true
	}
	v := strings.TrimSpace(strings.ToLower(os.Getenv("POLYLOG_STRICT")))
	return v == "1" || v == "true" || v == "yes"
}

// LoadConfigFromFile loads configuration from a file path.
// Automatically detects YAML (.yaml, .yml) or JSON (.json) format based on extension.
// If path is empty, uses ResolveConfigPath (POLYGLOT_CONFIG → cwd → parents, stop at .git).
//
// When strict mode is off (default), a missing file returns DefaultConfig.
// When strict is on (config.strict or POLYLOG_STRICT=1), a missing/empty path is an error.
// Does not print startup diagnostics (use CreateConfigFromFileWithOverrides for that).
func LoadConfigFromFile(filePath string) (Config, error) {
	resolve := ResolveConfigPath(filePath)
	if resolve.Path == "" {
		if strictRequested(false) {
			return Config{}, fmt.Errorf("strict mode: no config path (set POLYGLOT_CONFIG or pass a file)")
		}
		return DefaultConfig(), nil
	}

	cfg, err := loadConfigFileExact(resolve.Path)
	if err != nil {
		return Config{}, err
	}
	// Config file may itself enable strict for subsequent reloads / tooling.
	if strictRequested(cfg.Strict) && strings.TrimSpace(cfg.Service) == "" {
		return Config{}, fmt.Errorf("strict mode: service is required")
	}
	return cfg, nil
}
