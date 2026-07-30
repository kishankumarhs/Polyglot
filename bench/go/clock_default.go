//go:build !windows

package bench

import "time"

func clockNano() int64 {
	return time.Now().UnixNano()
}

func elapsedNano(startNano int64) int64 {
	d := clockNano() - startNano
	if d < 1 {
		return 1
	}
	return d
}
