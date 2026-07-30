package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

type stdoutSink struct {
	format string
}

func newStdoutSink(format string) *stdoutSink {
	if format == "" {
		format = StdoutFormatJSON
	}
	return &stdoutSink{format: format}
}

func (s *stdoutSink) Write(line []byte) (int, error) {
	if s.format != StdoutFormatText {
		return os.Stdout.Write(line)
	}
	text, err := jsonLineToText(line)
	if err != nil {
		// Fall back to raw bytes if the line is not JSON.
		return os.Stdout.Write(line)
	}
	return os.Stdout.Write([]byte(text))
}

func (s *stdoutSink) Flush() error { return nil }

func (s *stdoutSink) Close() error { return nil }

func (s *stdoutSink) Name() string { return "stdout" }

func jsonLineToText(line []byte) (string, error) {
	line = bytes.TrimSpace(line)
	var entry Entry
	if err := json.Unmarshal(line, &entry); err != nil {
		return "", err
	}
	out := fmt.Sprintf("%s %-5s %s", entry.Timestamp, entry.Level, entry.Message)
	if entry.Caller != "" {
		out += " (" + entry.Caller + ")"
	}
	if len(entry.Fields) > 0 {
		b, err := json.Marshal(entry.Fields)
		if err == nil {
			out += " " + string(b)
		}
	}
	return out + "\n", nil
}
