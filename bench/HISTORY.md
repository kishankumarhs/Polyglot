# Benchmark history

CI on `ubuntu-latest` appends rows. Don't gate releases on absolute numbers.

| Version | Zap sync ops/s | Zerolog sync ops/s | Polyglot sync ops/s | Polyglot sync P99 ns | Notes |
| --- | ---: | ---: | ---: | ---: | --- |
| v0.3.0 | — | — | — | — | harness introduced |

## Allocations (2026-07-30)

Custom encoder (`json_fast.go`) replaced `encoding/json.Marshal` on the hot path:

| Metric | Before | After |
| --- | ---: | ---: |
| allocs/op | 67 | 21 |
| B/op | 5,445 | 4,594 |
| ops/s vs Zap | ~0.8× | ~1.1× |

What's left: field map merge, timestamp string, output buffer, bench fixture (`RichFields`), misc.

## Timer fix (2026-07-30)

Go 1.25 `time.Since()` on Windows is coarse (~500µs). Histograms use `QueryPerformanceCounter` (`clock_windows.go`, ~100ns). Before that, P50/P95 looked like `1ns`.
