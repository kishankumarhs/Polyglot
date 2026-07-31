# User guide

Day-to-day usage. Language APIs: [languages/](languages/).

## How it works

```text
Your app  →  Logger API  →  (optional async queue)  →  sinks
                                                      ├─ stdout
                                                      ├─ rotating file
                                                      └─ HTTP / Loki
```

Go core owns formatting, queueing, and shipping. Bindings are thin FFI wrappers. Multiple logger instances are fine (auth, payments, …).

## Levels

| Level   | Int | Use                        |
| ------- | --- | -------------------------- |
| `trace` | 0   | Verbose diagnostics        |
| `debug` | 1   | Dev detail                 |
| `info`  | 2   | Normal ops                 |
| `warn`  | 3   | Recoverable surprise       |
| `error` | 4   | Needs attention            |
| `fatal` | 5   | Label only — does not exit |

Floor: `"level": "info"`. Exit yourself if you need process death on fatal.

## Fields

Three layers merge on every write (later wins):

1. Config fields
2. Context (`set_fields` / `setFields` / `SetFields`)
3. Per-call fields

For request scope use `With` / `with` / `with_fields`, not a shared mutable context.

## Async & overflow

Default `async: true`: serialize, enqueue, return. Worker writes sinks.

| Mode                    | Throughput | Durability                                                                             |
| ----------------------- | ---------- | -------------------------------------------------------------------------------------- |
| `async: true` (default) | Highest    | In-flight queue / HTTP buffer can be lost on crash; call `flush` + `close` on shutdown |
| `async: false`          | Lower      | Each log waits for sinks; stronger durability for process crashes after return         |

For file workloads, `file.fsync: true` adds an explicit per-write `fsync` path for stronger crash durability at the cost of latency and throughput.

This is a deliberate trade-off (same pattern as many production loggers). Choose per workload — latency-sensitive services usually keep async; audit-critical paths may prefer sync or explicit flush.

| `overflow`    | When full                     |
| ------------- | ----------------------------- |
| `drop_newest` | Drop the new entry            |
| `drop_oldest` | Drop one queued, then enqueue |
| `block`       | Wait for space                |

`queueSize` and `async` are fixed at create. Level, sinks, overflow, and fields can hot-reload.

See also [troubleshooting — logs missing after crash](troubleshooting.md#logs-missing-after-crash).

## Flush & close

| Call                | Behavior                        |
| ------------------- | ------------------------------- |
| `flush`             | Drain queue + sync sinks        |
| `close` / `Dispose` | Stop worker, flush, close sinks |

Check close errors on shutdown so the last HTTP batch isn't lost.

## Stats

In async mode, `info()` means queued, not written. Useful counters: `queued`, `dropped`, `flushed`, `write_errors`, `buffered`, `sink_dropped`.

## Hot reload

Can change level, sinks, overflow, base fields. Cannot change `async` / `queueSize` — create a new logger for those.

## Thread safety

Concurrent log/flush/stats/set_fields/reload on one instance is fine. Close once. Don't use a handle after close.

## Child loggers

```go
reqLog := root.With(map[string]any{"requestId": id})
_ = reqLog.Info("handling", nil)
```

Closing a child only frees that handle; close the root to shut down sinks.

## Trace helpers

```go
ctx := logger.ContextWithTrace(ctx, traceID, spanID)
_ = log.LogContext(ctx, logger.LevelInfo, "ok", nil)
```

## Next

[configuration.md](configuration.md) · [sinks.md](sinks.md) · [languages/](languages/)
