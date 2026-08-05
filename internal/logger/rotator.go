package logger

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// rotatingWriter writes to a file and rotates when size exceeds the limit.
type rotatingWriter struct {
	mu         sync.Mutex
	filename   string
	maxSize    int64
	maxBackups int
	maxAge     time.Duration
	compress   bool
	fsync      bool
	size       int64
	file       *os.File
	closed     bool
}

func newRotatingWriter(filename string, maxSizeMB, maxBackups, maxAgeDays int, compress bool) (*rotatingWriter, error) {
	w := &rotatingWriter{
		filename:   filename,
		maxSize:    int64(maxSizeMB) * 1024 * 1024,
		maxBackups: maxBackups,
		maxAge:     time.Duration(maxAgeDays) * 24 * time.Hour,
		compress:   compress,
	}
	if err := w.openExistingOrNew(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *rotatingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return 0, fmt.Errorf("log file %s is closed", w.filename)
	}
	if w.file == nil {
		if err := w.openExistingOrNew(); err != nil {
			return 0, err
		}
	}

	writeLen := int64(len(p))
	if w.maxSize > 0 && w.size+writeLen > w.maxSize {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}

	n, err := w.file.Write(p)
	w.size += int64(n)
	if err != nil {
		return n, err
	}
	if w.fsync {
		if syncErr := w.file.Sync(); syncErr != nil {
			return n, syncErr
		}
	}
	return n, err
}

func (w *rotatingWriter) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	return w.file.Sync()
}

func (w *rotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = true
	return w.closeFile()
}

func (w *rotatingWriter) openExistingOrNew() error {
	if err := os.MkdirAll(filepath.Dir(w.filename), 0o755); err != nil {
		if filepath.Dir(w.filename) != "." {
			return err
		}
	}

	info, err := os.Stat(w.filename)
	if os.IsNotExist(err) {
		return w.openNew()
	}
	if err != nil {
		return err
	}

	f, err := os.OpenFile(w.filename, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	w.file = f
	w.size = info.Size()
	return nil
}

func (w *rotatingWriter) openNew() error {
	f, err := os.OpenFile(w.filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	w.file = f
	w.size = 0
	return nil
}

func (w *rotatingWriter) closeFile() error {
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

func (w *rotatingWriter) rotate() error {
	if err := w.closeFile(); err != nil {
		return err
	}

	timestamp := time.Now().UTC().Format("20060102T150405.000")
	backupName := fmt.Sprintf("%s.%s", w.filename, timestamp)
	if err := os.Rename(w.filename, backupName); err != nil && !os.IsNotExist(err) {
		return err
	}
	if w.compress {
		go compressBackup(backupName)
	}

	if err := w.openNew(); err != nil {
		return err
	}
	go w.cleanup()
	return nil
}

func (w *rotatingWriter) cleanup() {
	matches, err := filepath.Glob(w.filename + ".*")
	if err != nil {
		return
	}

	type backup struct {
		path    string
		modTime time.Time
	}
	var backups []backup
	prefix := w.filename + "."
	for _, path := range matches {
		base := filepath.Base(path)
		name := filepath.Base(w.filename)
		if !strings.HasPrefix(base, name+".") {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		// Ignore non-backup suffixes that don't look like timestamps.
		suffix := strings.TrimPrefix(path, prefix)
		if suffix == "" {
			continue
		}
		backups = append(backups, backup{path: path, modTime: info.ModTime()})
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].modTime.After(backups[j].modTime)
	})

	now := time.Now()
	for i, b := range backups {
		expired := w.maxAge > 0 && now.Sub(b.modTime) > w.maxAge
		excess := w.maxBackups > 0 && i >= w.maxBackups
		if expired || excess {
			_ = os.Remove(b.path)
		}
	}
}

func compressBackup(path string) {
	in, err := os.Open(path)
	if err != nil {
		return
	}
	defer in.Close()
	outPath := path + ".gz"
	out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return
	}
	gz := gzip.NewWriter(out)
	if _, err := io.Copy(gz, in); err != nil {
		_ = gz.Close()
		_ = out.Close()
		_ = os.Remove(outPath)
		return
	}
	if err := gz.Close(); err != nil {
		_ = out.Close()
		_ = os.Remove(outPath)
		return
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(outPath)
		return
	}
	_ = os.Remove(path)
}

// Ensure rotatingWriter implements io.WriteCloser.
var _ io.WriteCloser = (*rotatingWriter)(nil)
