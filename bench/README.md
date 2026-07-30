# Benchmarks

Compare Polyglot to Zap, Zerolog, Pino, Python `logging`, and Serilog. Also cover overflow, hot reload, FFI cost, and cross-language schema.

Every timed path reports mean / P50 / P95 / P99, not just ops/sec.

If you only ship one language, that language's best logger still wins on ergonomics. Polyglot's pitch is a shared runtime across languages — these benches check that the cost is acceptable.

## Run

```bash
make build-native
make bench              # full suite → results/latest.md + charts
make bench-smoke        # short CI run
make bench-pprof        # cpu/heap profiles
python scripts/bench-charts.py   # charts from summary.json only
```

## Charts

Numbers come from `make bench` into [`results/summary.json`](results/summary.json). Until you regenerate, charts may be a previous snapshot.

![Throughput](results/throughput.svg)

![P99 latency](results/latency.svg)

![Scale](results/scale.svg)

![FFI](results/ffi.svg)

![Python / .NET](results/bindings.svg)

## Payload shapes

FFI and serialization cost scale with object complexity. These harnesses intentionally use different shapes for different questions:

| Harness | Shape | Why |
| --- | --- | --- |
| Go / Node / Python / .NET competitor benches | **Rich ~18 fields** — flat strings/numbers/bools + `tags[]` + nested `meta.cart` | Production-ish; fair across languages |
| Cross-language (`bench/cross`) | **~10 fields**, same nested `meta`/`tags` | Schema consistency, not headline throughput |
| FFI baseline (`ffi_baseline.mjs`) | **4 flat fields** | Isolate native crossing vs full sync log |
| slog reference row | **~10 flat fields** | Stdlib reference only — not a headline competitor |

Rich payload keys (Go `RichFields` / Node `rich()` / Python & .NET mirrors):

`user_id`, `trace_id`, `span_id`, `service`, `region`, `latency_ms`, `ok`, `tags`, `meta.cart`, `error`, `request_id`, `tenant`, `env`, `version`, `attempt`, `bytes`, `cached`, `n`.

Changing payload depth moves the bottleneck: shallow strings favor the host language; nested maps tax JS/Python/`System.Text.Json` stringify **before** the Go core ever runs.

## Node FFI tax (why Pino still wins)

Today every Node `log.info(msg, fields)` does:

1. `JSON.stringify(fields)` in V8  
2. koffi C string copy across FFI  
3. Go `json.Unmarshal` into `map[string]any`  
4. Go re-marshal to the final NDJSON line  

Pino never leaves V8 and compiles schemas into string paths. Crossing itself is cheap (`ffi_baseline.mjs`: ~1–2 µs); **pre-serialization + Go parse** dominate the gap.

Tracked next steps (not ABI-breaking yet):

1. Optional `logRaw(level, message, fieldsJson)` to skip a second host-side object walk when callers already hold JSON  
2. ABI v2 buffer path: write UTF-8 into a fixed `Uint8Array` / `Buffer`, pass pointer+len to Go (bypasses V8 string extraction)  
3. Longer term: accept a pre-built NDJSON line and skip Go field re-parse entirely  

Until those land, treat Node vs Pino as “shared runtime cost,” not “Go core is slower than Pino.”

## What we measure

### Go (`bench/go`)

- Sync / async file vs Zap and Zerolog (slog is a reference row)
- Rich ~18-field payload
- `With()` child vs `zap.With` / `zerolog.With`
- Scale 1→64 writers
- Memory / GC for a fixed N

### Node / Bun (`bench/node`)

- Polyglot sync + async file vs Pino sync file (same rich payload)
- Same script on Bun when `bun` is on `PATH`

### Python (`bench/python`)

- Polyglot sync file vs stdlib `logging` + `FileHandler` (JSON via `json.dumps` in a `Formatter`)
- Same rich payload as Go/Node

### .NET (`bench/dotnet`)

- Polyglot sync file vs Serilog `WriteTo.File` with compact JSON
- Same rich payload as Go/Node

### FFI (`ffi_baseline.mjs`)

Splits native crossing (`logger_version`) from a full sync log. Crossing is cheap; serialize + disk dominate.

### Cross-language (`bench/cross`)

Go / Node / Python / .NET write the same schema through the same native lib. Smaller payload than competitor benches — proves shape, not speed.

### Overflow & reload

Ops behavior, not micro-ops/sec:

- Queue overflow (`TestOverflowBackpressure`) — flood a small queue, report dropped / flushed / recover time
- Hot reload (`TestHotReloadUnderLoad`) — `ReloadConfig` under load

## Fairness

- Same rich field shape across competitors within a language harness
- Sync file is the fair comparison; async Polyglot is labeled separately
- Child scopes use `With()` — not racing global `SetFields`
- Do not compare Go Zap numbers to Node Pino numbers as if they shared a runtime

## History

[HISTORY.md](HISTORY.md) — version rows from CI. Absolute numbers vary by machine; compare relative deltas on the same runner.
