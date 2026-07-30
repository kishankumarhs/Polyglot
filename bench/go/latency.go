package bench

import (
	"math"
	"sort"
	"sync"
)

// LatencyHist records nanosecond samples and reports percentiles.
type LatencyHist struct {
	mu      sync.Mutex
	samples []int64
}

func NewLatencyHist(capHint int) *LatencyHist {
	if capHint < 1024 {
		capHint = 1024
	}
	return &LatencyHist{samples: make([]int64, 0, capHint)}
}

func (h *LatencyHist) RecordNano(ns int64) {
	if ns < 1 {
		ns = 1
	}
	h.mu.Lock()
	h.samples = append(h.samples, ns)
	h.mu.Unlock()
}

// Start returns a high-resolution timestamp for latency measurement.
// Pair with RecordElapsed.
func Start() int64 { return clockNano() }

// RecordElapsed records the nanoseconds elapsed since a Start() timestamp.
func (h *LatencyHist) RecordElapsed(startNano int64) {
	h.RecordNano(elapsedNano(startNano))
}

type Percentiles struct {
	Count int
	Mean  float64
	P50   int64
	P95   int64
	P99   int64
	Max   int64
}

func (h *LatencyHist) Snapshot() Percentiles {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := len(h.samples)
	if n == 0 {
		return Percentiles{}
	}
	cp := make([]int64, n)
	copy(cp, h.samples)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	var sum int64
	for _, v := range cp {
		sum += v
	}
	return Percentiles{
		Count: n,
		Mean:  float64(sum) / float64(n),
		P50:   percentile(cp, 50),
		P95:   percentile(cp, 95),
		P99:   percentile(cp, 99),
		Max:   cp[n-1],
	}
}

func percentile(sorted []int64, p int) int64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}
	// Nearest-rank with 1-based index.
	rank := int(math.Ceil(float64(p) / 100 * float64(len(sorted))))
	if rank < 1 {
		rank = 1
	}
	if rank > len(sorted) {
		rank = len(sorted)
	}
	return sorted[rank-1]
}

func (p Percentiles) Report(b interface{ ReportMetric(n float64, unit string) }) {
	if p.Count == 0 {
		return
	}
	b.ReportMetric(p.Mean, "ns/mean")
	b.ReportMetric(float64(p.P50), "ns/p50")
	b.ReportMetric(float64(p.P95), "ns/p95")
	b.ReportMetric(float64(p.P99), "ns/p99")
}
