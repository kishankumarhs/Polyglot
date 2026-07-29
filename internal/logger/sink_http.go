package logger

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// maxBufferedBatches bounds how many batches worth of lines the HTTP sink keeps
// while the collector is unreachable. Beyond this the oldest lines are dropped
// so an outage cannot grow memory without limit.
const maxBufferedBatches = 20

// httpSink batches NDJSON log lines and POSTs them to a collector URL.
// Request body is newline-delimited JSON (application/x-ndjson).
//
// Delivery: a batch is only cleared after a 2xx response. On network errors or
// non-2xx status the lines are retained and retried on the next flush (ticker,
// explicit Flush, or Close), up to maxBufferedBatches*batchSize lines.
type httpSink struct {
	mu       sync.Mutex
	url      string
	headers  map[string]string
	client   *http.Client
	batch    [][]byte
	batchMax int
	buffMax  int
	dropped  int
	interval time.Duration
	stopCh   chan struct{}
	doneCh   chan struct{}
	closed   bool
}

func newHTTPSink(cfg *HTTPConfig) (*httpSink, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil http config")
	}
	if err := validateCollectorURL(cfg.URL); err != nil {
		return nil, err
	}
	s := &httpSink{
		url:      cfg.URL,
		headers:  cloneStringMap(cfg.Headers),
		client:   &http.Client{Timeout: time.Duration(cfg.TimeoutMS) * time.Millisecond},
		batch:    make([][]byte, 0, cfg.BatchSize),
		batchMax: cfg.BatchSize,
		interval: time.Duration(cfg.FlushIntervalMS) * time.Millisecond,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
	if s.batchMax <= 0 {
		s.batchMax = 50
	}
	if s.interval <= 0 {
		s.interval = time.Second
	}
	s.buffMax = s.batchMax * maxBufferedBatches
	go s.tickerLoop()
	return s, nil
}

// validateCollectorURL restricts the sink to http/https with an absolute host,
// so a malformed or hostile config cannot point the sink at unexpected schemes.
func validateCollectorURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("invalid http.url: %w", err)
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
	default:
		return fmt.Errorf("http.url scheme must be http or https, got %q", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("http.url must include a host")
	}
	return nil
}

func (s *httpSink) Name() string { return "http" }

// Buffered reports lines held locally and not yet accepted by the collector.
func (s *httpSink) Buffered() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.batch)
}

func (s *httpSink) Write(line []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return 0, fmt.Errorf("http sink closed")
	}
	cp := make([]byte, len(line))
	copy(cp, line)
	s.batch = append(s.batch, cp)
	n := len(line)

	if len(s.batch) >= s.batchMax {
		if err := s.flushLocked(); err != nil {
			s.trimLocked()
			return n, err
		}
	}
	return n, nil
}

func (s *httpSink) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	err := s.flushLocked()
	if err != nil {
		s.trimLocked()
	}
	return err
}

func (s *httpSink) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.mu.Unlock()

	select {
	case <-s.stopCh:
	default:
		close(s.stopCh)
	}
	<-s.doneCh

	s.mu.Lock()
	defer s.mu.Unlock()
	// Final delivery attempt; anything still buffered is reported as an error so
	// the caller knows logs were not accepted by the collector.
	return s.flushLocked()
}

func (s *httpSink) tickerLoop() {
	defer close(s.doneCh)
	t := time.NewTicker(s.interval)
	defer t.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-t.C:
			_ = s.Flush()
		}
	}
}

// flushLocked POSTs the buffered lines. The batch is retained unless the
// collector returns 2xx, so a failed request does not lose logs.
func (s *httpSink) flushLocked() error {
	if len(s.batch) == 0 {
		return nil
	}

	var buf bytes.Buffer
	for _, line := range s.batch {
		buf.Write(line)
		if len(line) == 0 || line[len(line)-1] != '\n' {
			buf.WriteByte('\n')
		}
	}

	req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(buf.Bytes()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-ndjson")
	for k, v := range s.headers {
		req.Header.Set(k, v)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http sink status %d (%d lines retained)", resp.StatusCode, len(s.batch))
	}

	s.batch = s.batch[:0]
	return nil
}

// trimLocked enforces the buffer ceiling by discarding the oldest lines.
func (s *httpSink) trimLocked() {
	if s.buffMax <= 0 || len(s.batch) <= s.buffMax {
		return
	}
	excess := len(s.batch) - s.buffMax
	s.batch = append(s.batch[:0], s.batch[excess:]...)
	s.dropped += excess
}

// DroppedLines reports lines discarded because the retry buffer was full.
func (s *httpSink) DroppedLines() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dropped
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
