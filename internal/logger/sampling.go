package logger

import "sync"

// sampler implements zap-style first-N-then-every sampling keyed by level+message.
type sampler struct {
	mu         sync.Mutex
	initial    int
	thereafter int
	counts     map[string]int
}

func newSampler(cfg *SamplingConfig) *sampler {
	if cfg == nil || !cfg.Enabled {
		return nil
	}
	return &sampler{
		initial:    cfg.Initial,
		thereafter: cfg.Thereafter,
		counts:     map[string]int{},
	}
}

func (s *sampler) allow(level Level, message string) bool {
	if s == nil {
		return true
	}
	key := level.String() + "\x00" + message
	s.mu.Lock()
	defer s.mu.Unlock()
	n := s.counts[key]
	s.counts[key] = n + 1
	if n < s.initial {
		return true
	}
	if s.thereafter <= 0 {
		return false
	}
	return (n-s.initial)%s.thereafter == 0
}
