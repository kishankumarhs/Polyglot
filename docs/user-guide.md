# User guide

How to use the logger day to day from any supported language. Language-specific APIs are in [languages/](languages/); this guide is language-agnostic.

## Mental model

```text
Your app  →  Logger API  →  (optional async queue)  →  sinks
                                                      ├─ stdout
                                                      ├─ rotating file
                                                      └─ HTTP (NDJSON)
```

- **One Go core** owns formatting, JSON, rotation, queueing, and HTTP shipping.
- Bindings are thin wrappers over the C ABI. They do **not** reimplement logging logic.
- You can create **multiple logger instances** (auth, payments, workflow), each with its own config.

## Log levels

| Level | Integer | Typical use |
| ----- | ------- | ----------- |
| `trace` | 0 | Very verbose diagnostics |
| `debug` | 1 | Development detail |
| `info` | 2 | Normal operations |
| `warn` | 3 | Unexpected but recoverable |
| `error` | 4 | Failures that need attention |
| `fatal` | 5 | Critical failure **label only** |

Minimum level is configured with `"level": "info"` (or binding options). Messages below that level are discarded before enqueue.

**Important:** `fatal` does **not** exit the process. Call `os.Exit` / `process.exit` / `Environment.Exit` yourself if that is the behavior you want.

## Structured fields

Every log is one JSON object. Extra data goes in `fields`.

Three layers merge on every write:

1. **Config fields** — set at create / reload (`fields` in JSON config)
2. **Context fields** — set at runtime (`set_fields` / `setFields` / `SetFields`)
3. **Per-call fields** — passed on a single `info` / `log` call

Later layers override earlier keys with the same name.

Typical context fields: `traceId`, `requestId`, `tenantId`, `userId`.

```text
config fields  →  context fields  →  per-call fields
```

## Async logging and overflow

With `"async": true` (default):

1. Your thread serializes the entry and pushes it onto a bounded queue.
2. A background worker drains the queue and writes to sinks.
3. Your call returns as soon as the entry is queued (or dropped/blocked).

| `overflow` | When the queue is full |
| ---------- | --------------------- |
| `drop_newest` (default) | Reject the new entry; increment `dropped` |
| `drop_oldest` | Discard one queued entry, then enqueue the new one |
| `block` | Wait until there is space (or the logger closes) |

`queueSize` and `async` are fixed at creation. Everything else (level, sinks, overflow, fields) can change via hot reload.

## Flush and close

| Call | Behavior |
| ---- | -------- |
| `flush` | Drain the async queue and sync sinks (HTTP POST pending batches, flush file buffers) |
| `close` / `Dispose` | Stop the worker, flush, close sinks. Prefer a **single owner** |

`close` returns an error (or throws, depending on binding) if the final flush failed — for example if the HTTP collector never accepted remaining lines. Check it on shutdown.

Always close loggers in long-running services during graceful shutdown so the last batches leave the process.

## Stats (delivery visibility)

In async mode a successful `info()` only means “queued,” not “written to disk/collector.” Scrape stats:

| Counter | Meaning |
| ------- | ------- |
| `queued` | Entries waiting in the async queue |
| `dropped` | Lost at the queue (overflow policy) |
| `flushed` | Handed to sinks |
| `bytes_written` | Serialized bytes handed to sinks |
| `write_errors` | Payloads where at least one sink write failed |
| `buffered` | Lines the HTTP sink still holds (pending 2xx) |
| `sink_dropped` | Lines discarded because the HTTP retry buffer was full |

Alert on rising `dropped`, `write_errors`, or `sink_dropped` if you care about delivery.

## Hot reload

`reload_config` / `reloadConfig` / `ReloadConfig` applies a new JSON config:

- Can change: level, stdout/file/http sinks, overflow, base fields
- Cannot change: `async`, `queueSize` (create a new logger if you need different queue settings)

Reload is safe while other threads are logging.

## Thread safety

- Concurrent `log` / `flush` / `stats` / `set_fields` / `reload` on one instance is safe (Node worker threads, Python threads, .NET Tasks).
- Call `close` once from a single owner.
- Do not use a handle after `close`.

## Choosing sinks for your environment

| Goal | Suggested config |
| ---- | ---------------- |
| Local development | `stdout: true`, file optional |
| Single VM with disk | Rotating file (`maxSizeMB`, `maxBackups`, `maxAgeDays`) |
| Avoid filling VM disks | `file.enabled: false`, `http.enabled: true` → central collector |
| Grafana / Loki | HTTP → adapter or Vector/Alloy → Loki (see [sinks](sinks.md)) |

## Multiple services / instances

Create one logger per logical service or pipeline:

```text
auth-logger      →  service: "auth-api"
payments-logger  →  service: "payments-api"
```

Each has independent sinks, levels, and stats.

## Error model

- Native ABI returns `0` on success, `-1` on failure.
- `logger_last_error(handle)` returns a message; `NULL` handle → create/global errors.
- Bindings raise / throw native exceptions (`LoggerError`, `LoggerException`).

## Next

- [Configuration reference](configuration.md)
- [Sinks & centralized logging](sinks.md)
- Language guides under [languages/](languages/)
