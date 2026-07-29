package logger

import "os"

type stdoutSink struct{}

func newStdoutSink() *stdoutSink {
	return &stdoutSink{}
}

func (s *stdoutSink) Write(line []byte) (int, error) {
	return os.Stdout.Write(line)
}

func (s *stdoutSink) Flush() error { return nil }

func (s *stdoutSink) Close() error { return nil }

func (s *stdoutSink) Name() string { return "stdout" }
