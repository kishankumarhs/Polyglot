package logger

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	kafka "github.com/segmentio/kafka-go"
)

// kafkaSink batches JSON lines and writes them to a Kafka topic.
// Delivery semantics mirror other remote sinks: batches are only cleared after
// successful broker acknowledgment based on required_acks.
type kafkaSink struct {
	mu       sync.Mutex
	writer   *kafka.Writer
	batch    [][]byte
	batchMax int
	buffMax  int
	dropped  int
	failures int
	interval time.Duration
	timeout  time.Duration
	stopCh   chan struct{}
	doneCh   chan struct{}
	closed   bool
}

func newKafkaSink(cfg *KafkaConfig) (*kafkaSink, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil kafka config")
	}
	requiredAcks, err := kafkaRequiredAcks(cfg.RequiredAcks)
	if err != nil {
		return nil, err
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 50
	}
	if cfg.FlushIntervalMS <= 0 {
		cfg.FlushIntervalMS = 1000
	}
	if cfg.TimeoutMS <= 0 {
		cfg.TimeoutMS = 5000
	}

	s := &kafkaSink{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(cfg.Brokers...),
			Topic:        cfg.Topic,
			RequiredAcks: requiredAcks,
			BatchSize:    cfg.BatchSize,
			BatchTimeout: time.Duration(cfg.FlushIntervalMS) * time.Millisecond,
			WriteTimeout: time.Duration(cfg.TimeoutMS) * time.Millisecond,
			ReadTimeout:  time.Duration(cfg.TimeoutMS) * time.Millisecond,
			Balancer:     &kafka.LeastBytes{},
			Async:        false,
		},
		batch:    make([][]byte, 0, cfg.BatchSize),
		batchMax: cfg.BatchSize,
		interval: time.Duration(cfg.FlushIntervalMS) * time.Millisecond,
		timeout:  time.Duration(cfg.TimeoutMS) * time.Millisecond,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
	s.buffMax = s.batchMax * maxBufferedBatches
	go s.tickerLoop()
	return s, nil
}

func kafkaRequiredAcks(v int) (kafka.RequiredAcks, error) {
	switch v {
	case -1:
		return kafka.RequireAll, nil
	case 0:
		return kafka.RequireNone, nil
	case 1:
		return kafka.RequireOne, nil
	default:
		return 0, fmt.Errorf("kafka.required_acks must be -1, 0, or 1")
	}
}

func (s *kafkaSink) Name() string { return "kafka" }

func (s *kafkaSink) Buffered() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.batch)
}

func (s *kafkaSink) DroppedLines() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dropped
}

func (s *kafkaSink) Write(line []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return 0, fmt.Errorf("kafka sink closed")
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

func (s *kafkaSink) Flush() error {
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

func (s *kafkaSink) Close() error {
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
	flushErr := s.flushLocked()
	s.mu.Unlock()
	closeErr := s.writer.Close()
	if flushErr != nil {
		return flushErr
	}
	return closeErr
}

func (s *kafkaSink) tickerLoop() {
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

func (s *kafkaSink) backoffLocked() time.Duration {
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

func (s *kafkaSink) flushLocked() error {
	if len(s.batch) == 0 {
		return nil
	}
	msgs := make([]kafka.Message, 0, len(s.batch))
	for _, line := range s.batch {
		msgs = append(msgs, kafka.Message{Value: bytes.TrimRight(line, "\n")})
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()
	if err := s.writer.WriteMessages(ctx, msgs...); err != nil {
		s.failures++
		return fmt.Errorf("kafka sink write (%d lines retained): %w", len(s.batch), err)
	}
	s.batch = s.batch[:0]
	s.failures = 0
	return nil
}

func (s *kafkaSink) trimLocked() {
	if s.buffMax <= 0 || len(s.batch) <= s.buffMax {
		return
	}
	excess := len(s.batch) - s.buffMax
	s.batch = append(s.batch[:0], s.batch[excess:]...)
	s.dropped += excess
}
