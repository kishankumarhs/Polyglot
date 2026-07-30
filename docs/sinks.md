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

Buffers up to `batch_size`, also flushes on `flush_interval_ms` and on `flush` / `close`. A batch is sent when **any** of these happens: the buffer reaches `batch_size`, the flush interval elapses, or you call `flush` / `close`. With small traffic and `flush_interval_ms: 500`, logs can take up to 500ms to appear — that is expected, not a stuck sink.

POSTs **NDJSON** (newline-delimited JSON objects), not a JSON array. `Content-Type: application/x-ndjson`. Keeps the batch until a 2xx. Long outages cap the buffer (`batch_size × 20`) and increment `sink_dropped`.

Wire format example:

```http
POST /v1/logs HTTP/1.1
Content-Type: application/x-ndjson

{"timestamp":"…","level":"info","message":"a","service_name":"payments-api"}
{"timestamp":"…","level":"info","message":"b","service_name":"payments-api"}
```

```bash
curl -v -H 'Content-Type: application/x-ndjson' \
  --data-binary $'{"message":"ping","service_name":"curl"}\n' \
  "http://localhost:9999/v1/logs"
```

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

URL must be `http`/`https` with a host. Put tokens in env/secrets, not committed config. In Node/.NET options the same fields use camelCase (`batchSize`, `flushIntervalMs`).

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
