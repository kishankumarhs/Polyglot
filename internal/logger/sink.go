package logger

import "fmt"

// Sink writes serialized log lines (typically one JSON object + newline).
type Sink interface {
	Write(line []byte) (int, error)
	Flush() error
	Close() error
	Name() string
}

// bufferedSink is implemented by sinks that hold lines until a remote accepts
// them, so the logger can report a backlog and its own drops through Stats.
type bufferedSink interface {
	Buffered() int
	DroppedLines() int
}

// Future sink interfaces (not implemented):
//   - KafkaSink
//   - SyslogSink
//   - OTelSink

func buildSinks(cfg Config) ([]Sink, error) {
	var sinks []Sink
	if cfg.Stdout {
		sinks = append(sinks, newStdoutSink(cfg.StdoutFormat))
	}
	if cfg.FileEnabled() {
		fs, err := newFileSink(cfg.File)
		if err != nil {
			return nil, fmt.Errorf("file sink: %w", err)
		}
		sinks = append(sinks, fs)
	}
	if cfg.HTTPEnabled() {
		hs, err := newHTTPSink(cfg.HTTP)
		if err != nil {
			_ = closeSinks(sinks)
			return nil, fmt.Errorf("http sink: %w", err)
		}
		sinks = append(sinks, hs)
	}
	if cfg.LokiEnabled() {
		ls, err := newLokiSink(cfg.Loki)
		if err != nil {
			_ = closeSinks(sinks)
			return nil, fmt.Errorf("loki sink: %w", err)
		}
		sinks = append(sinks, ls)
	}
	if len(sinks) == 0 {
		return nil, fmt.Errorf("no sinks configured")
	}
	return sinks, nil
}

func closeSinks(sinks []Sink) error {
	var first error
	for _, s := range sinks {
		if s == nil {
			continue
		}
		if err := s.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func flushSinks(sinks []Sink) error {
	var first error
	for _, s := range sinks {
		if s == nil {
			continue
		}
		if err := s.Flush(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// sinkBacklog totals the lines sinks are still holding and the lines they had
// to discard because their retry buffer was full.
func sinkBacklog(sinks []Sink) (buffered, dropped int) {
	for _, s := range sinks {
		if bs, ok := s.(bufferedSink); ok {
			buffered += bs.Buffered()
			dropped += bs.DroppedLines()
		}
	}
	return buffered, dropped
}
