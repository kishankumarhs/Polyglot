package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// lokiSink batches log lines and POSTs Loki push API payloads.
// URL should be the full push endpoint, e.g. http://loki:3100/loki/api/v1/push
type lokiSink struct {
	mu       sync.Mutex
	url      string
	headers  map[string]string
	labels   map[string]string
	client   *http.Client
	batch    []lokiLine
	batchMax int
	buffMax  int
	dropped  int
	failures int
	interval time.Duration
	stopCh   chan struct{}
	doneCh   chan struct{}
	closed   bool
}

type lokiLine struct {
	tsNs  string
	line  string
	level string
	svc   string
	env   string
}

func newLokiSink(cfg *LokiConfig) (*lokiSink, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil loki config")
	}
	if err := validateCollectorURL(cfg.URL); err != nil {
		return nil, err
	}
	s := &lokiSink{
		url:      cfg.URL,
		headers:  cloneStringMap(cfg.Headers),
		labels:   cloneStringMap(cfg.Labels),
		client:   &http.Client{Timeout: time.Duration(cfg.TimeoutMS) * time.Millisecond},
		batch:    make([]lokiLine, 0, cfg.BatchSize),
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

func (s *lokiSink) Name() string { return "loki" }

func (s *lokiSink) Buffered() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.batch)
}

func (s *lokiSink) DroppedLines() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dropped
}

func (s *lokiSink) Write(line []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return 0, fmt.Errorf("loki sink closed")
	}
	entry, tsNs, err := parseEntryForLoki(line)
	if err != nil {
		// Still ship the raw line so we never drop on parse quirks.
		tsNs = strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
		entry = Entry{Message: string(bytes.TrimSpace(line)), Level: "info"}
	}
	s.batch = append(s.batch, lokiLine{
		tsNs:  tsNs,
		line:  string(bytes.TrimSpace(line)),
		level: entry.Level,
		svc:   entry.ServiceName,
		env:   entry.Environment,
	})
	n := len(line)
	if len(s.batch) >= s.batchMax {
		if err := s.flushLocked(); err != nil {
			s.trimLocked()
			return n, err
		}
	}
	return n, nil
}

func (s *lokiSink) Flush() error {
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

func (s *lokiSink) Close() error {
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
	return s.flushLocked()
}

func (s *lokiSink) tickerLoop() {
	defer close(s.doneCh)
	t := time.NewTicker(s.interval)
	defer t.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-t.C:
			s.mu.Lock()
			wait := s.backoffLocked()
			s.mu.Unlock()
			if wait > 0 {
				timer := time.NewTimer(wait)
				select {
				case <-s.stopCh:
					timer.Stop()
					return
				case <-timer.C:
				}
			}
			_ = s.Flush()
		}
	}
}

func (s *lokiSink) backoffLocked() time.Duration {
	if s.failures <= 0 {
		return 0
	}
	exp := s.failures
	if exp > 8 {
		exp = 8
	}
	base := time.Duration(100*(1<<uint(exp))) * time.Millisecond
	if base > 30*time.Second {
		base = 30 * time.Second
	}
	return base
}

func (s *lokiSink) flushLocked() error {
	if len(s.batch) == 0 {
		return nil
	}

	// Group by label set so Loki receives proper streams.
	type streamKey struct {
		svc, level, env string
	}
	groups := map[streamKey][][2]string{}
	for _, item := range s.batch {
		key := streamKey{svc: item.svc, level: item.level, env: item.env}
		groups[key] = append(groups[key], [2]string{item.tsNs, item.line})
	}

	type lokiStream struct {
		Stream map[string]string `json:"stream"`
		Values [][2]string       `json:"values"`
	}
	type lokiPush struct {
		Streams []lokiStream `json:"streams"`
	}

	payload := lokiPush{}
	for key, values := range groups {
		labels := cloneStringMap(s.labels)
		if key.svc != "" {
			if _, ok := labels["service_name"]; !ok {
				labels["service_name"] = key.svc
			}
		}
		if key.level != "" {
			if _, ok := labels["level"]; !ok {
				labels["level"] = key.level
			}
		}
		if key.env != "" {
			if _, ok := labels["environment"]; !ok {
				labels["environment"] = key.env
			}
		}
		payload.Streams = append(payload.Streams, lokiStream{Stream: labels, Values: values})
	}

	body, err := json.Marshal(payload)
	if err != nil {
		s.failures++
		return err
	}
	req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(body))
	if err != nil {
		s.failures++
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range s.headers {
		req.Header.Set(k, v)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		s.failures++
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.failures++
		return fmt.Errorf("loki sink status %d (%d lines retained)", resp.StatusCode, len(s.batch))
	}
	s.batch = s.batch[:0]
	s.failures = 0
	return nil
}

func (s *lokiSink) trimLocked() {
	if s.buffMax <= 0 || len(s.batch) <= s.buffMax {
		return
	}
	excess := len(s.batch) - s.buffMax
	s.batch = append(s.batch[:0], s.batch[excess:]...)
	s.dropped += excess
}

func parseEntryForLoki(line []byte) (Entry, string, error) {
	var entry Entry
	if err := json.Unmarshal(bytes.TrimSpace(line), &entry); err != nil {
		return Entry{}, "", err
	}
	ts := time.Now().UTC()
	if entry.Timestamp != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, entry.Timestamp); err == nil {
			ts = parsed
		}
	}
	return entry, strconv.FormatInt(ts.UnixNano(), 10), nil
}
