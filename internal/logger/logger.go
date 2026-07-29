package logger

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const (
	Version    = "0.2.0"
	ABIVersion = 1
)

// Entry is a structured log record.
type Entry struct {
	Timestamp      string         `json:"timestamp"`
	Level          string         `json:"level"`
	Message        string         `json:"message"`
	ServiceName    string         `json:"service_name"`
	ServiceVersion string         `json:"service_version,omitempty"`
	Environment    string         `json:"environment,omitempty"`
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

	// dropOldestMu makes the pop-then-push sequence atomic so a concurrent
	// producer cannot refill the queue between the two steps and force the new
	// item to be dropped instead of the oldest one.
	dropOldestMu sync.Mutex

	// shutdownErr is written by the worker before doneCh closes, so Close can
	// report flush/close failures instead of silently succeeding.
	shutdownErr error

	closed atomic.Bool
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

// Log emits a structured log line at the given level.
func (l *Logger) Log(level Level, message string, fields map[string]any) error {
	if l.closed.Load() {
		return fmt.Errorf("logger is closed")
	}

	l.mu.RLock()
	minLevel := l.minLevel
	cfg := l.cfg
	ctx := cloneAnyMap(l.context)
	l.mu.RUnlock()

	if !level.Enabled(minLevel) {
		return nil
	}

	merged := make(map[string]any, len(cfg.Fields)+len(ctx)+len(fields))
	for k, v := range cfg.Fields {
		merged[k] = v
	}
	for k, v := range ctx {
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

	payload, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal log entry: %w", err)
	}
	payload = append(payload, '\n')

	if !l.async {
		return l.writePayload(payload)
	}
	return l.enqueue(payload)
}

// LogJSON emits a log using a level name and JSON object for fields.
func (l *Logger) LogJSON(levelName, message, fieldsJSON string) error {
	level, err := ParseLevel(levelName)
	if err != nil {
		return err
	}
	fields, err := parseFieldsJSON(fieldsJSON)
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
	fields, err := parseFieldsJSON(fieldsJSON)
	if err != nil {
		return err
	}
	return l.Log(level, message, fields)
}

// SetFields replaces runtime context fields merged into every subsequent log.
// Config base fields are unchanged; per-call fields still override context.
func (l *Logger) SetFields(fields map[string]any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if fields == nil {
		l.context = map[string]any{}
		return
	}
	l.context = cloneAnyMap(fields)
}

// SetFieldsJSON replaces runtime context fields from a JSON object.
func (l *Logger) SetFieldsJSON(fieldsJSON string) error {
	fields, err := parseFieldsJSON(fieldsJSON)
	if err != nil {
		return err
	}
	l.SetFields(fields)
	return nil
}

// Stats returns a snapshot of runtime counters.
func (l *Logger) Stats() Stats {
	s := l.stats.snapshot()
	l.mu.RLock()
	buffered, dropped := sinkBacklog(l.sinks)
	l.mu.RUnlock()
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

// ReloadConfig hot-reloads configuration (level, fields, sinks, async overflow settings).
func (l *Logger) ReloadConfig(cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	if l.closed.Load() {
		return fmt.Errorf("logger is closed")
	}
	if l.async {
		req := reloadRequest{cfg: cfg, done: make(chan error, 1)}
		select {
		case l.reloadCh <- req:
			return <-req.done
		case <-l.doneCh:
			return fmt.Errorf("logger is closed")
		}
	}
	return l.applyConfig(cfg)
}

// Flush drains the async queue (if any) and syncs sinks.
func (l *Logger) Flush() error {
	if l.closed.Load() {
		return fmt.Errorf("logger is closed")
	}
	if l.async {
		req := flushRequest{done: make(chan error, 1)}
		select {
		case l.flushCh <- req:
			return <-req.done
		case <-l.doneCh:
			return fmt.Errorf("logger is closed")
		}
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	return flushSinks(l.sinks)
}

// Close stops the worker cleanly, flushes, and closes sinks. It returns the
// first flush/close error so callers can detect an incomplete shutdown (for
// example logs the HTTP collector never accepted).
func (l *Logger) Close() error {
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

	// Read under the same lock that applyConfig uses to write it.
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
		// Serialize pop+push so the freed slot cannot be taken by another
		// producer, which would drop the newest item instead of the oldest.
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
	default: // drop_newest
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
		select {
		case item := <-l.queue:
			l.stats.queued.Add(-1)
			_ = l.writePayload(item.payload)
		case req := <-l.flushCh:
			l.drainQueue()
			l.mu.RLock()
			err := flushSinks(l.sinks)
			l.mu.RUnlock()
			req.done <- err
		case req := <-l.reloadCh:
			l.drainQueue()
			req.done <- l.applyConfig(req.cfg)
		case <-l.stopCh:
			l.drainQueue()
			l.mu.Lock()
			err := flushSinks(l.sinks)
			if closeErr := closeSinks(l.sinks); err == nil {
				err = closeErr
			}
			l.shutdownErr = err
			l.sinks = nil
			l.mu.Unlock()
			return
		}
	}
}

func (l *Logger) drainQueue() {
	for {
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
	// Held for the whole write so a concurrent reload cannot close the sinks
	// underneath an in-flight write (a closed file sink would silently reopen).
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
		// Async callers already got a nil error from Log(), so this counter is
		// the only way they learn that sinks are failing.
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
	// Async mode itself is fixed at construction; queue size stays as created.
	l.mu.Unlock()
	return nil
}

func parseFieldsJSON(fieldsJSON string) (map[string]any, error) {
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
	for k, v := range in {
		out[k] = v
	}
	return out
}

// Convenience helpers.
func (l *Logger) Trace(msg string, fields map[string]any) error {
	return l.Log(LevelTrace, msg, fields)
}
func (l *Logger) Debug(msg string, fields map[string]any) error {
	return l.Log(LevelDebug, msg, fields)
}
func (l *Logger) Info(msg string, fields map[string]any) error {
	return l.Log(LevelInfo, msg, fields)
}
func (l *Logger) Warn(msg string, fields map[string]any) error {
	return l.Log(LevelWarn, msg, fields)
}
func (l *Logger) Error(msg string, fields map[string]any) error {
	return l.Log(LevelError, msg, fields)
}

// Fatal writes at fatal level. It does NOT terminate the process; the caller
// decides whether to exit.
func (l *Logger) Fatal(msg string, fields map[string]any) error {
	return l.Log(LevelFatal, msg, fields)
}
