//go:build windows

package bench

import (
	"sync"
	"syscall"
	"unsafe"
)

// Windows QPC-based high-resolution clock.
// Go 1.25's time.Since() uses a monotonic clock with ~500µs resolution on
// Windows, which makes sub-millisecond per-call latency measurement useless.
// QPC provides 100ns resolution on modern hardware.

var (
	kernel32       = syscall.NewLazyDLL("kernel32.dll")
	procQPC        = kernel32.NewProc("QueryPerformanceCounter")
	procQPF        = kernel32.NewProc("QueryPerformanceFrequency")
	qpcFreqOnce    sync.Once
	qpcFreqHz      float64
	qpcNsPerTickX1 float64 // 1e9 / freq, cached
)

func initQPCFreq() {
	qpcFreqOnce.Do(func() {
		var f int64
		procQPF.Call(uintptr(unsafe.Pointer(&f)))
		qpcFreqHz = float64(f)
		qpcNsPerTickX1 = 1e9 / qpcFreqHz
	})
}

func clockNano() int64 {
	initQPCFreq()
	var c int64
	procQPC.Call(uintptr(unsafe.Pointer(&c)))
	return int64(float64(c) * qpcNsPerTickX1)
}

func elapsedNano(startNano int64) int64 {
	d := clockNano() - startNano
	if d < 1 {
		return 1
	}
	return d
}
