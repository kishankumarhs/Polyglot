# Sinks

Destinations for serialized JSON lines. You can enable more than one.

| Sink | Config | Format |
| --- | --- | --- |
| Stdout | `"stdout": true` | JSON lines, or `stdout_format: text` |
| File | `"file": { "enabled": true, ... }` | JSON lines, rotation, optional gzip |
| HTTP | `"http": { "enabled": true, ... }` | Batched NDJSON POST |
| Loki | `"loki": { "enabled": true, "url": "..." }` | Loki push JSON |

Kafka / syslog / OTLP are reserved, not implemented.

## Stdout

Good for containers and local dev.

## File

Writes to `file.path`, rotates on `maxSizeMB`, trims with `maxBackups` / `maxAgeDays`. `compress: true` gzips rotated backups.

## HTTP

Buffers up to `batch_size`, also flushes on `flush_interval_ms` and on `flush` / `close`. POSTs NDJSON; keeps the batch until a 2xx. Long outages cap the buffer (`batch_size × 20`) and increment `sink_dropped`.

```json
{
  "service": "payments-api",
  "level": "info",
  "stdout": false,
  "file": { "enabled": false },
  "http": {
    "enabled": true,
    "url": "https://collector.internal/v1/logs",
    "timeout_ms": 5000,
    "batch_size": 50,
    "flush_interval_ms": 1000,
    "headers": { "Authorization": "Bearer <token>" }
  },
  "async": true,
  "queueSize": 10000,
  "overflow": "drop_newest"
}
```

URL must be `http`/`https` with a host. Put tokens in env/secrets, not committed config.

## Loki / Grafana

Prefer the native sink — do not point `http.url` at Loki (that sink speaks NDJSON, not Loki push JSON).

```json
{
  "service": "payments-api",
  "stdout": false,
  "loki": {
    "enabled": true,
    "url": "http://loki:3100/loki/api/v1/push",
    "batch_size": 50,
    "flush_interval_ms": 1000,
    "labels": { "job": "payments-api" }
  }
}
```

Labels `service_name`, `level`, and `environment` are added when missing.

Or keep HTTP NDJSON and transform with Vector/Alloy:

```toml
# vector.toml (sketch)
[sources.polyglot]
type = "http_server"
address = "0.0.0.0:8686"
decoding.codec = "json"
framing.method = "newline_delimited"

[sinks.loki]
type = "loki"
inputs = ["polyglot"]
endpoint = "http://loki:3100"
encoding.codec = "json"
```

Point Polyglot `http.url` at `http://vector:8686`.

```text
App ──loki sink──► Loki ←── Grafana
App ──HTTP NDJSON──► Vector/Alloy ──► Loki
App ──file──► Promtail/Alloy ──► Loki   (uses local disk)
```

## Choosing

| Constraint | Suggestion |
| --- | --- |
| Don't fill VM disks | Disable file; enable `loki` or `http` |
| Grafana | Native `loki`, or HTTP → Vector → Loki |
| Already run Vector/Alloy | HTTP NDJSON + recipe above |
| Air-gapped | File + `compress` + tight rotation |

See [configuration.md](configuration.md) · [troubleshooting.md](troubleshooting.md).
