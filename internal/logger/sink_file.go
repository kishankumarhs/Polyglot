package logger

import "fmt"

type fileSink struct {
	w *rotatingWriter
}

func newFileSink(cfg *FileConfig) (*fileSink, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil file config")
	}
	rw, err := newRotatingWriter(cfg.Path, cfg.MaxSizeMB, cfg.MaxBackups, cfg.MaxAgeDays, cfg.Compress)
	if err != nil {
		return nil, err
	}
	return &fileSink{w: rw}, nil
}

func (s *fileSink) Write(line []byte) (int, error) {
	return s.w.Write(line)
}

func (s *fileSink) Flush() error {
	return s.w.Sync()
}

func (s *fileSink) Close() error {
	if err := s.w.Sync(); err != nil {
		_ = s.w.Close()
		return err
	}
	return s.w.Close()
}

func (s *fileSink) Name() string { return "file" }
