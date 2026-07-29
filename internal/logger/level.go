package logger

import (
	"fmt"
	"strings"
)

// Level represents a log severity. Values match the C ABI LoggerLevel enum.
type Level int

const (
	LevelTrace Level = iota
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

var levelNames = map[Level]string{
	LevelTrace: "trace",
	LevelDebug: "debug",
	LevelInfo:  "info",
	LevelWarn:  "warn",
	LevelError: "error",
	LevelFatal: "fatal",
}

var levelByName = map[string]Level{
	"trace":   LevelTrace,
	"debug":   LevelDebug,
	"info":    LevelInfo,
	"warn":    LevelWarn,
	"warning": LevelWarn,
	"error":   LevelError,
	"fatal":   LevelFatal,
}

// String returns the canonical lowercase name for a level.
func (l Level) String() string {
	if name, ok := levelNames[l]; ok {
		return name
	}
	return fmt.Sprintf("level(%d)", int(l))
}

// ParseLevel parses a level name (case-insensitive).
func ParseLevel(s string) (Level, error) {
	level, ok := levelByName[strings.ToLower(strings.TrimSpace(s))]
	if !ok {
		return 0, fmt.Errorf("unknown log level %q", s)
	}
	return level, nil
}

// LevelFromInt converts an ABI integer level to Level.
func LevelFromInt(n int) (Level, error) {
	if n < int(LevelTrace) || n > int(LevelFatal) {
		return 0, fmt.Errorf("unknown log level %d", n)
	}
	return Level(n), nil
}

// Enabled reports whether level should be emitted given the minimum threshold.
func (l Level) Enabled(min Level) bool {
	return l >= min
}
