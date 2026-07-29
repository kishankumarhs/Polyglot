package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Overflow policies for the async queue.
const (
	OverflowDropNewest = "drop_newest"
	OverflowDropOldest = "drop_oldest"
	OverflowBlock      = "block"
)

// FileConfig configures the rotating file sink.
type FileConfig struct {
	Enabled    bool   `json:"enabled"`
	Path       string `json:"path"`
	MaxSizeMB  int    `json:"maxSizeMB"`
	MaxBackups int    `json:"maxBackups"`
	MaxAgeDays int    `json:"maxAgeDays"`
	Compress   bool   `json:"compress"` // reserved; compression not yet implemented
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

// Config controls logger behavior.
type Config struct {
	Service        string         `json:"service"`
	ServiceVersion string         `json:"service_version,omitempty"`
	Environment    string         `json:"environment,omitempty"`
	Level          string         `json:"level"`
	Stdout         bool           `json:"stdout"`
	File           *FileConfig    `json:"file,omitempty"`
	HTTP           *HTTPConfig    `json:"http,omitempty"`
	Async          bool           `json:"async"`
	QueueSize      int            `json:"queueSize"`
	Overflow       string         `json:"overflow"`
	Fields         map[string]any `json:"fields,omitempty"`
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		Service:   "app",
		Level:     "info",
		Stdout:    true,
		Async:     true,
		QueueSize: 10000,
		Overflow:  OverflowDropNewest,
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
	File           json.RawMessage `json:"file"`
	FilePath       string          `json:"file_path"`
	MaxSizeMB      int             `json:"max_size_mb"`
	MaxBackups     int             `json:"max_backups"`
	MaxAgeDays     int             `json:"max_age_days"`
	HTTP           *HTTPConfig     `json:"http"`
	Async          *bool           `json:"async"`
	QueueSize      int             `json:"queueSize"`
	Overflow       string          `json:"overflow"`
	Fields         map[string]any  `json:"fields"`
}

// ParseConfigJSON parses and validates configuration from JSON.
// Accepts nested schema and legacy flat keys (service_name, min_level, file_path, ...).
func ParseConfigJSON(data []byte) (Config, error) {
	cfg := DefaultConfig()
	if len(data) == 0 {
		return cfg, nil
	}

	var raw rawConfig
	if err := json.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("invalid config json: %w", err)
	}

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

	if err := cfg.Validate(); err != nil {
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
	if c.QueueSize < 0 {
		return fmt.Errorf("queueSize must be >= 0")
	}
	if c.QueueSize == 0 {
		c.QueueSize = 10000
	}
	if c.Fields == nil {
		c.Fields = map[string]any{}
	}

	fileEnabled := c.FileEnabled()
	httpEnabled := c.HTTPEnabled()
	if !c.Stdout && !fileEnabled && !httpEnabled {
		return fmt.Errorf("at least one sink must be enabled (stdout, file, or http)")
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
	// Nested object with path but enabled omitted/false: treat path as enable signal only when Enabled is true.
	return false
}

// HTTPEnabled reports whether the HTTP sink should be active.
func (c Config) HTTPEnabled() bool {
	return c.HTTP != nil && c.HTTP.Enabled && strings.TrimSpace(c.HTTP.URL) != ""
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
