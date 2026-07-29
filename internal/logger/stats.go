package logger

import "sync/atomic"

// Stats holds runtime counters for a logger instance.
type Stats struct {
	Queued       uint64 `json:"queued"`
	Dropped      uint64 `json:"dropped"`
	Flushed      uint64 `json:"flushed"`
	BytesWritten uint64 `json:"bytes_written"`
	// WriteErrors counts payloads where at least one sink write failed. Async
	// callers cannot see per-write errors, so this is the signal that sinks are
	// unhealthy even though Log() returned success.
	WriteErrors uint64 `json:"write_errors"`
	// Buffered is the number of log lines currently held by sinks that batch
	// (currently the HTTP sink) and not yet accepted by the collector.
	Buffered uint64 `json:"buffered"`
	// SinkDropped counts lines a sink discarded because its retry buffer was
	// full. Unlike Dropped, these were accepted by the queue and then lost
	// downstream.
	SinkDropped uint64 `json:"sink_dropped"`
}

type statsCounters struct {
	queued       atomic.Int64
	dropped      atomic.Uint64
	flushed      atomic.Uint64
	bytesWritten atomic.Uint64
	writeErrors  atomic.Uint64
}

func (s *statsCounters) snapshot() Stats {
	q := s.queued.Load()
	if q < 0 {
		q = 0
	}
	return Stats{
		Queued:       uint64(q),
		Dropped:      s.dropped.Load(),
		Flushed:      s.flushed.Load(),
		BytesWritten: s.bytesWritten.Load(),
		WriteErrors:  s.writeErrors.Load(),
	}
}
