package logger

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/protobuf/proto"
)

type otlpSink struct {
	mu       sync.Mutex
	url      string
	headers  map[string]string
	client   *http.Client
	batch    [][]byte
	batchMax int
	buffMax  int
	dropped  int
	interval time.Duration
	failures int
	stopCh   chan struct{}
	doneCh   chan struct{}
	closed   bool
}

func newOTLPSink(cfg *OTLPConfig) (*otlpSink, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil otlp config")
	}
	if err := validateCollectorURL(cfg.URL); err != nil {
		return nil, err
	}
	s := &otlpSink{
		url:      normalizeOTLPLogsURL(cfg.URL),
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

func normalizeOTLPLogsURL(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimRight(raw, "/")
	if strings.HasSuffix(raw, "/v1/logs") {
		return raw
	}
	return raw + "/v1/logs"
}

func (s *otlpSink) Name() string { return "otlp" }

func (s *otlpSink) Buffered() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.batch)
}

func (s *otlpSink) DroppedLines() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dropped
}

func (s *otlpSink) Write(line []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return 0, fmt.Errorf("otlp sink closed")
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

func (s *otlpSink) Flush() error {
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

func (s *otlpSink) Close() error {
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

func (s *otlpSink) tickerLoop() {
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

func (s *otlpSink) backoffLocked() time.Duration {
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

func (s *otlpSink) flushLocked() error {
	if len(s.batch) == 0 {
		return nil
	}

	body, err := buildOTLPExportRequest(s.batch)
	if err != nil {
		s.failures++
		return err
	}

	req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(body))
	if err != nil {
		s.failures++
		return err
	}
	req.Header.Set("Content-Type", "application/x-protobuf")
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
		return fmt.Errorf("otlp sink status %d (%d lines retained)", resp.StatusCode, len(s.batch))
	}

	s.batch = s.batch[:0]
	s.failures = 0
	return nil
}

func (s *otlpSink) trimLocked() {
	if s.buffMax <= 0 || len(s.batch) <= s.buffMax {
		return
	}
	excess := len(s.batch) - s.buffMax
	s.batch = append(s.batch[:0], s.batch[excess:]...)
	s.dropped += excess
}

func buildOTLPExportRequest(lines [][]byte) ([]byte, error) {
	records := make([]*logspb.LogRecord, 0, len(lines))
	for _, line := range lines {
		r := lineToOTLPRecord(line)
		records = append(records, r)
	}
	req := &collogspb.ExportLogsServiceRequest{
		ResourceLogs: []*logspb.ResourceLogs{{
			Resource: &resourcepb.Resource{},
			ScopeLogs: []*logspb.ScopeLogs{{
				Scope:      &commonpb.InstrumentationScope{Name: "polyglot-logger"},
				LogRecords: records,
			}},
		}},
	}
	return proto.Marshal(req)
}

func lineToOTLPRecord(line []byte) *logspb.LogRecord {
	raw := bytes.TrimSpace(line)
	rec := &logspb.LogRecord{
		TimeUnixNano:   uint64(time.Now().UTC().UnixNano()),
		SeverityText:   "info",
		SeverityNumber: logspb.SeverityNumber_SEVERITY_NUMBER_INFO,
		Body:           &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: string(raw)}},
	}

	var entry Entry
	if err := json.Unmarshal(raw, &entry); err != nil {
		return rec
	}

	if entry.Timestamp != "" {
		if ts, err := time.Parse(time.RFC3339Nano, entry.Timestamp); err == nil {
			rec.TimeUnixNano = uint64(ts.UnixNano())
		}
	}
	if entry.Message != "" {
		rec.Body = &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: entry.Message}}
	}
	if entry.Level != "" {
		rec.SeverityText = strings.ToLower(entry.Level)
		rec.SeverityNumber = otlpSeverityNumber(entry.Level)
	}

	attrs := make([]*commonpb.KeyValue, 0, 8)
	if entry.ServiceName != "" {
		attrs = append(attrs, &commonpb.KeyValue{Key: "service_name", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: entry.ServiceName}}})
	}
	if entry.ServiceVersion != "" {
		attrs = append(attrs, &commonpb.KeyValue{Key: "service_version", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: entry.ServiceVersion}}})
	}
	if entry.Environment != "" {
		attrs = append(attrs, &commonpb.KeyValue{Key: "environment", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: entry.Environment}}})
	}
	if entry.Caller != "" {
		attrs = append(attrs, &commonpb.KeyValue{Key: "caller", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: entry.Caller}}})
	}
	for k, v := range entry.Fields {
		attrs = append(attrs, &commonpb.KeyValue{Key: k, Value: toOTLPAnyValue(v)})
	}
	rec.Attributes = attrs

	if v, ok := entry.Fields["trace_id"]; ok {
		if b, ok := decodeFixedHex(fmt.Sprint(v), 16); ok {
			rec.TraceId = b
		}
	}
	if v, ok := entry.Fields["span_id"]; ok {
		if b, ok := decodeFixedHex(fmt.Sprint(v), 8); ok {
			rec.SpanId = b
		}
	}

	return rec
}

func otlpSeverityNumber(level string) logspb.SeverityNumber {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "trace":
		return logspb.SeverityNumber_SEVERITY_NUMBER_TRACE
	case "debug":
		return logspb.SeverityNumber_SEVERITY_NUMBER_DEBUG
	case "info":
		return logspb.SeverityNumber_SEVERITY_NUMBER_INFO
	case "warn", "warning":
		return logspb.SeverityNumber_SEVERITY_NUMBER_WARN
	case "error":
		return logspb.SeverityNumber_SEVERITY_NUMBER_ERROR
	case "fatal":
		return logspb.SeverityNumber_SEVERITY_NUMBER_FATAL
	default:
		return logspb.SeverityNumber_SEVERITY_NUMBER_UNSPECIFIED
	}
}

func toOTLPAnyValue(v any) *commonpb.AnyValue {
	switch x := v.(type) {
	case nil:
		return &commonpb.AnyValue{}
	case string:
		return &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: x}}
	case bool:
		return &commonpb.AnyValue{Value: &commonpb.AnyValue_BoolValue{BoolValue: x}}
	case float64:
		return &commonpb.AnyValue{Value: &commonpb.AnyValue_DoubleValue{DoubleValue: x}}
	case float32:
		return &commonpb.AnyValue{Value: &commonpb.AnyValue_DoubleValue{DoubleValue: float64(x)}}
	case int:
		return &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: int64(x)}}
	case int8:
		return &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: int64(x)}}
	case int16:
		return &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: int64(x)}}
	case int32:
		return &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: int64(x)}}
	case int64:
		return &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: x}}
	case uint:
		return &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: int64(x)}}
	case uint8:
		return &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: int64(x)}}
	case uint16:
		return &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: int64(x)}}
	case uint32:
		return &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: int64(x)}}
	case uint64:
		if x > uint64(math.MaxInt64) {
			return &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: fmt.Sprint(x)}}
		}
		return &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: int64(x)}}
	case []any:
		vals := make([]*commonpb.AnyValue, 0, len(x))
		for _, item := range x {
			vals = append(vals, toOTLPAnyValue(item))
		}
		return &commonpb.AnyValue{Value: &commonpb.AnyValue_ArrayValue{ArrayValue: &commonpb.ArrayValue{Values: vals}}}
	case map[string]any:
		kvs := make([]*commonpb.KeyValue, 0, len(x))
		for k, item := range x {
			kvs = append(kvs, &commonpb.KeyValue{Key: k, Value: toOTLPAnyValue(item)})
		}
		return &commonpb.AnyValue{Value: &commonpb.AnyValue_KvlistValue{KvlistValue: &commonpb.KeyValueList{Values: kvs}}}
	default:
		return &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: fmt.Sprint(v)}}
	}
}

func decodeFixedHex(s string, wantBytes int) ([]byte, bool) {
	s = strings.TrimSpace(s)
	if len(s) != wantBytes*2 {
		return nil, false
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, false
	}
	if len(b) != wantBytes {
		return nil, false
	}
	return b, true
}
