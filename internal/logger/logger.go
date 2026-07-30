package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const (
	Version    = "0.3.0"
	ABIVersion = 1
)

// Context keys for optional trace correlation (language-agnostic contract).
type ctxKey string

const (
	CtxTraceID ctxKey = "polyglot.trace_id"
	CtxSpanID  ctxKey = "polyglot.span_id"
)

// Entry is a structured log record.
type Entry struct {
	Timestamp      string         `json:"timestamp"`
	Level          string         `json:"level"`
	Message        string         `json:"message"`
	ServiceName    string         `json:"service_name"`
	ServiceVersion string         `json:"service_version,omitempty"`
	Environment    string         `json:"environment,omitempty"`
	Caller         string         `json:"caller,omitempty"`
	Fields         map[string]any `json:"fields,omitempty"`
}

type queueItem struct {
	payload []byte
}

type flushRequest struct {
	done chan error
}

type reloadRequest struct {
	cfg  Config
	done chan error
}

// Logger writes structured JSON logs to configured sinks.
type Logger struct {
	mu       sync.RWMutex
	cfg      Config
	minLevel Level
	sinks    []Sink
	context  map[string]any
	stats    statsCounters

	async    bool
	overflow string
	queue    chan queueItem
	flushCh  chan flushRequest
	reloadCh chan reloadRequest
	stopCh   chan struct{}
	doneCh   chan struct{}

	dropOldestMu sync.Mutex
	shutdownErr  error
	closed       atomic.Bool

	// owner is non-nil for child loggers created by With(); they share the
	// root's sinks/queue and must not Close the root.
	owner *Logger
	extra map[string]any

	sampler *sampler
}

// New creates a Logger from Config.
func New(cfg Config) (*Logger, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if err := cfg.EnsureFileDir(); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	sinks, err := buildSinks(cfg)
	if err != nil {
		return nil, err
	}

	l := &Logger{
		cfg:      cfg,
		minLevel: cfg.MinLevelValue(),
		sinks:    sinks,
		context:  map[string]any{},
		async:    cfg.Async,
		overflow: cfg.Overflow,
		sampler:  newSampler(cfg.Sampling),
	}

	if cfg.Async {
		l.queue = make(chan queueItem, cfg.QueueSize)
		l.flushCh = make(chan flushRequest)
		l.reloadCh = make(chan reloadRequest)
		l.stopCh = make(chan struct{})
		l.doneCh = make(chan struct{})
		go l.worker()
	}
	return l, nil
}

func (l *Logger) root() *Logger {
	if l.owner != nil {
		return l.owner
	}
	return l
}

// With returns a child logger that inherits sinks and queue from the root but
// carries additional immutable fields. Closing a child is a no-op; close the root.
func (l *Logger) With(fields map[string]any) *Logger {
	r := l.root()
	extras := mergeMaps(l.extra, fields)
	return &Logger{
		owner: r,
		extra: extras,
	}
}

// IsChild reports whether this logger was created by With().
func (l *Logger) IsChild() bool {
	return l.owner != nil
}

// Log emits a structured log line at the given level.
func (l *Logger) Log(level Level, message string, fields map[string]any) error {
	return l.logAt(level, message, fields, 3)
}

func (l *Logger) logAt(level Level, message string, fields map[string]any, callerSkip int) error {
	r := l.root()
	if r.closed.Load() {
		return fmt.Errorf("logger is closed")
	}

	r.mu.RLock()
	minLevel := r.minLevel
	cfg := r.cfg
	ctx := cloneAnyMap(r.context)
	sampler := r.sampler
	r.mu.RUnlock()

	if !level.Enabled(minLevel) {
		return nil
	}
	if sampler != nil && !sampler.allow(level, message) {
		r.stats.dropped.Add(1)
		return nil
	}

	merged := make(map[string]any, len(cfg.Fields)+len(ctx)+len(l.extra)+len(fields))
	for k, v := range cfg.Fields {
		merged[k] = v
	}
	for k, v := range ctx {
		merged[k] = v
	}
	for k, v := range l.extra {
		merged[k] = v
	}
	for k, v := range fields {
		merged[k] = v
	}
	if len(merged) == 0 {
		merged = nil
	}

	entry := Entry{
		Timestamp:      time.Now().UTC().Format(time.RFC3339Nano),
		Level:          level.String(),
		Message:        message,
		ServiceName:    cfg.Service,
		ServiceVersion: cfg.ServiceVersion,
		Environment:    cfg.Environment,
		Fields:         merged,
	}
	if cfg.Caller {
		entry.Caller = callerFrame(callerSkip)
	}

	var payload []byte
	var err error
	payload, err = marshalEntry(&entry)
	if err != nil {
		return fmt.Errorf("marshal log entry: %w", err)
	}
	payload = append(payload, '\n')

	if !r.async {
		return r.writePayload(payload)
	}
	return r.enqueue(payload)
}

// LogContext emits a log and merges trace_id/span_id from ctx when present.
func (l *Logger) LogContext(ctx context.Context, level Level, message string, fields map[string]any) error {
	merged := cloneAnyMap(fields)
	if merged == nil {
		merged = map[string]any{}
	}
	if ctx != nil {
		if v := ctx.Value(CtxTraceID); v != nil {
			merged["trace_id"] = fmt.Sprint(v)
		}
		if v := ctx.Value(CtxSpanID); v != nil {
			merged["span_id"] = fmt.Sprint(v)
		}
	}
	return l.logAt(level, message, merged, 3)
}

// LogError logs at error level and attaches err under fields["error"].
func (l *Logger) LogError(err error, message string, fields map[string]any) error {
	merged := cloneAnyMap(fields)
	if merged == nil {
		merged = map[string]any{}
	}
	if err != nil {
		merged["error"] = err.Error()
	}
	return l.logAt(LevelError, message, merged, 3)
}

// LogJSON emits a log using a level name and JSON object for fields.
func (l *Logger) LogJSON(levelName, message, fieldsJSON string) error {
	level, err := ParseLevel(levelName)
	if err != nil {
		return err
	}
	fields, err := ParseFieldsJSON(fieldsJSON)
	if err != nil {
		return err
	}
	return l.Log(level, message, fields)
}

// LogInt emits a log using an ABI integer level.
func (l *Logger) LogInt(levelInt int, message, fieldsJSON string) error {
	level, err := LevelFromInt(levelInt)
	if err != nil {
		return err
	}
	fields, err := ParseFieldsJSON(fieldsJSON)
	if err != nil {
		return err
	}
	return l.Log(level, message, fields)
}

// SetFields replaces runtime context fields merged into every subsequent log.
// Config base fields are unchanged; per-call fields still override context.
// On a child logger this updates the root's context (shared).
func (l *Logger) SetFields(fields map[string]any) {
	r := l.root()
	r.mu.Lock()
	defer r.mu.Unlock()
	if fields == nil {
		r.context = map[string]any{}
		return
	}
	r.context = cloneAnyMap(fields)
}

// SetFieldsJSON replaces runtime context fields from a JSON object.
func (l *Logger) SetFieldsJSON(fieldsJSON string) error {
	fields, err := ParseFieldsJSON(fieldsJSON)
	if err != nil {
		return err
	}
	l.SetFields(fields)
	return nil
}

// Stats returns a snapshot of runtime counters.
func (l *Logger) Stats() Stats {
	r := l.root()
	s := r.stats.snapshot()
	r.mu.RLock()
	buffered, dropped := sinkBacklog(r.sinks)
	r.mu.RUnlock()
	s.Buffered = uint64(buffered)
	s.SinkDropped = uint64(dropped)
	return s
}

// StatsJSON returns stats as a JSON object.
func (l *Logger) StatsJSON() string {
	data, err := json.Marshal(l.Stats())
	if err != nil {
		return `{"queued":0,"dropped":0,"flushed":0,"bytes_written":0,"write_errors":0,"buffered":0,"sink_dropped":0}`
	}
	return string(data)
}

// MetricsText returns a Prometheus-style text dump of counters for scraping.
func (l *Logger) MetricsText() string {
	s := l.Stats()
	return fmt.Sprintf(
		"# HELP polyglot_logger_queued Entries waiting in the async queue\n"+
			"# TYPE polyglot_logger_queued gauge\n"+
			"polyglot_logger_queued %d\n"+
			"# HELP polyglot_logger_dropped Entries dropped by overflow or sampling\n"+
			"# TYPE polyglot_logger_dropped counter\n"+
			"polyglot_logger_dropped %d\n"+
			"# HELP polyglot_logger_flushed Entries handed to sinks\n"+
			"# TYPE polyglot_logger_flushed counter\n"+
			"polyglot_logger_flushed %d\n"+
			"# HELP polyglot_logger_bytes_written Serialized bytes handed to sinks\n"+
			"# TYPE polyglot_logger_bytes_written counter\n"+
			"polyglot_logger_bytes_written %d\n"+
			"# HELP polyglot_logger_write_errors Payloads with at least one sink write failure\n"+
			"# TYPE polyglot_logger_write_errors counter\n"+
			"polyglot_logger_write_errors %d\n"+
			"# HELP polyglot_logger_buffered Lines buffered in sinks pending delivery\n"+
			"# TYPE polyglot_logger_buffered gauge\n"+
			"polyglot_logger_buffered %d\n"+
			"# HELP polyglot_logger_sink_dropped Lines discarded by sink retry buffers\n"+
			"# TYPE polyglot_logger_sink_dropped counter\n"+
			"polyglot_logger_sink_dropped %d\n",
		s.Queued, s.Dropped, s.Flushed, s.BytesWritten, s.WriteErrors, s.Buffered, s.SinkDropped,
	)
}

// ReloadConfig hot-reloads configuration (level, fields, sinks, overflow).
func (l *Logger) ReloadConfig(cfg Config) error {
	r := l.root()
	if err := cfg.Validate(); err != nil {
		return err
	}
	if r.closed.Load() {
		return fmt.Errorf("logger is closed")
	}
	if r.async {
		req := reloadRequest{cfg: cfg, done: make(chan error, 1)}
		select {
		case r.reloadCh <- req:
			return <-req.done
		case <-r.doneCh:
			return fmt.Errorf("logger is closed")
		}
	}
	return r.applyConfig(cfg)
}

// Flush drains the async queue (if any) and syncs sinks.
func (l *Logger) Flush() error {
	r := l.root()
	if r.closed.Load() {
		return fmt.Errorf("logger is closed")
	}
	if r.async {
		req := flushRequest{done: make(chan error, 1)}
		select {
		case r.flushCh <- req:
			return <-req.done
		case <-r.doneCh:
			return fmt.Errorf("logger is closed")
		}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return flushSinks(r.sinks)
}

// Close stops the worker cleanly, flushes, and closes sinks.
// Closing a child logger is a no-op; close the root logger instead.
func (l *Logger) Close() error {
	if l.owner != nil {
		return nil
	}
	if !l.closed.CompareAndSwap(false, true) {
		return nil
	}
	if l.async {
		close(l.stopCh)
		<-l.doneCh
		l.mu.RLock()
		defer l.mu.RUnlock()
		return l.shutdownErr
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	err := flushSinks(l.sinks)
	if closeErr := closeSinks(l.sinks); err == nil {
		err = closeErr
	}
	l.sinks = nil
	return err
}

func (l *Logger) enqueue(payload []byte) error {
	item := queueItem{payload: payload}

	l.mu.RLock()
	overflow := l.overflow
	l.mu.RUnlock()

	switch overflow {
	case OverflowBlock:
		select {
		case l.queue <- item:
			l.stats.queued.Add(1)
			return nil
		case <-l.stopCh:
			return fmt.Errorf("logger is closed")
		}
	case OverflowDropOldest:
		select {
		case l.queue <- item:
			l.stats.queued.Add(1)
			return nil
		default:
		}
		l.dropOldestMu.Lock()
		defer l.dropOldestMu.Unlock()
		select {
		case <-l.queue:
			l.stats.queued.Add(-1)
			l.stats.dropped.Add(1)
		default:
		}
		select {
		case l.queue <- item:
			l.stats.queued.Add(1)
			return nil
		default:
			l.stats.dropped.Add(1)
			return nil
		}
	default:
		select {
		case l.queue <- item:
			l.stats.queued.Add(1)
			return nil
		default:
			l.stats.dropped.Add(1)
			return nil
		}
	}
}

func (l *Logger) worker() {
	defer close(l.doneCh)
	for {
		// Prefer flush/reload/stop over the hot queue. A plain multi-way select
		// can starve control channels when writers keep the queue ready.
		if l.serviceControl() {
			return
		}
		select {
		case item := <-l.queue:
			l.stats.queued.Add(-1)
			_ = l.writePayload(item.payload)
		case req := <-l.flushCh:
			l.handleFlush(req)
		case req := <-l.reloadCh:
			l.handleReload(req)
		case <-l.stopCh:
			l.handleStop()
			return
		}
	}
}

// serviceControl drains one pending control message without blocking.
// Returns true when the worker should exit.
func (l *Logger) serviceControl() bool {
	select {
	case req := <-l.flushCh:
		l.handleFlush(req)
	case req := <-l.reloadCh:
		l.handleReload(req)
	case <-l.stopCh:
		l.handleStop()
		return true
	default:
	}
	return false
}

func (l *Logger) handleFlush(req flushRequest) {
	l.drainQueue()
	l.mu.RLock()
	err := flushSinks(l.sinks)
	l.mu.RUnlock()
	req.done <- err
}

func (l *Logger) handleReload(req reloadRequest) {
	// Payloads are already serialized; remaining queue items write to the new
	// sinks after applyConfig. Avoid unbounded drain under a live flood.
	l.drainQueue()
	req.done <- l.applyConfig(req.cfg)
}

func (l *Logger) handleStop() {
	l.drainQueue()
	l.mu.Lock()
	err := flushSinks(l.sinks)
	if closeErr := closeSinks(l.sinks); err == nil {
		err = closeErr
	}
	l.shutdownErr = err
	l.sinks = nil
	l.mu.Unlock()
}

func (l *Logger) drainQueue() {
	// Cap iterations so a live producer cannot keep the queue non-empty forever
	// (reload/flush/stop would otherwise hang).
	max := cap(l.queue) + 1
	if max < 1 {
		max = 1
	}
	for i := 0; i < max; i++ {
		select {
		case item := <-l.queue:
			l.stats.queued.Add(-1)
			_ = l.writePayload(item.payload)
		default:
			return
		}
	}
}

func (l *Logger) writePayload(payload []byte) error {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var first error
	for _, s := range l.sinks {
		if _, err := s.Write(payload); err != nil && first == nil {
			first = fmt.Errorf("write %s: %w", s.Name(), err)
		}
	}
	l.stats.flushed.Add(1)
	l.stats.bytesWritten.Add(uint64(len(payload)))
	if first != nil {
		l.stats.writeErrors.Add(1)
	}
	return first
}

func (l *Logger) applyConfig(cfg Config) error {
	if err := cfg.EnsureFileDir(); err != nil {
		return fmt.Errorf("create log directory: %w", err)
	}
	newSinks, err := buildSinks(cfg)
	if err != nil {
		return err
	}

	l.mu.Lock()
	old := l.sinks
	_ = flushSinks(old)
	_ = closeSinks(old)
	l.cfg = cfg
	l.minLevel = cfg.MinLevelValue()
	l.sinks = newSinks
	l.overflow = cfg.Overflow
	l.sampler = newSampler(cfg.Sampling)
	l.mu.Unlock()
	return nil
}

// ParseFieldsJSON parses a JSON object into a field map.
func ParseFieldsJSON(fieldsJSON string) (map[string]any, error) {
	if fieldsJSON == "" || fieldsJSON == "null" || fieldsJSON == "{}" {
		return nil, nil
	}
	var fields map[string]any
	if err := json.Unmarshal([]byte(fieldsJSON), &fields); err != nil {
		return nil, fmt.Errorf("invalid fields json: %w", err)
	}
	return fields, nil
}

func cloneAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	maps.Copy(out, in)
	return out
}

func mergeMaps(base, overlay map[string]any) map[string]any {
	out := cloneAnyMap(base)
	maps.Copy(out, overlay)
	if len(out) == 0 {
		return nil
	}
	return out
}

func callerFrame(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%s:%d", filepath.Base(file), line)
}

// Convenience helpers.
func (l *Logger) Trace(msg string, fields map[string]any) error {
	return l.logAt(LevelTrace, msg, fields, 3)
}
func (l *Logger) Debug(msg string, fields map[string]any) error {
	return l.logAt(LevelDebug, msg, fields, 3)
}
func (l *Logger) Info(msg string, fields map[string]any) error {
	return l.logAt(LevelInfo, msg, fields, 3)
}
func (l *Logger) Warn(msg string, fields map[string]any) error {
	return l.logAt(LevelWarn, msg, fields, 3)
}
func (l *Logger) Error(msg string, fields map[string]any) error {
	return l.logAt(LevelError, msg, fields, 3)
}

// Fatal writes at fatal level. It does NOT terminate the process; the caller
// decides whether to exit.
func (l *Logger) Fatal(msg string, fields map[string]any) error {
	return l.logAt(LevelFatal, msg, fields, 3)
}

// ContextWithTrace returns a child context carrying trace and span ids.
func ContextWithTrace(ctx context.Context, traceID, spanID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if traceID != "" {
		ctx = context.WithValue(ctx, CtxTraceID, traceID)
	}
	if spanID != "" {
		ctx = context.WithValue(ctx, CtxSpanID, spanID)
	}
	return ctx
}
