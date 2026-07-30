# Sinks and centralized logging

A **sink** is a destination for serialized JSON log lines. The logger can write to one or more sinks at once.

## Built-in sinks

| Sink | Config | Format |
| ---- | ------ | ------ |
| Stdout | `"stdout": true` | JSON lines, or `stdout_format: text` for local dev |
| File | `"file": { "enabled": true, ... }` | JSON lines, size/age rotation, optional gzip |
| HTTP | `"http": { "enabled": true, ... }` | Batched **NDJSON** POST |
| Loki | `"loki": { "enabled": true, "url": ".../loki/api/v1/push" }` | Native Loki push JSON |

Future (interface reserved, not implemented): Kafka, syslog, OpenTelemetry OTLP exporter.

## Stdout

Best for containers (Docker/Kubernetes) where the platform already scrapes process output, and for local development.

## File (rotating)

Writes to `file.path`. When the active file exceeds `maxSizeMB`, it rotates. Old files are trimmed by `maxBackups` and `maxAgeDays`.

Use this when you need durable local logs. Cap retention carefully so VMs do not fill disks.

When `compress: true`, rotated backups are gzipped asynchronously (`app.log.2026….gz`).

## HTTP (centralized)

When `http.enabled` is true, the logger:

1. Buffers lines up to `batch_size`
2. Also flushes on `flush_interval_ms` and on explicit `flush` / `close`
3. `POST`s the batch as **newline-delimited JSON** (`Content-Type: application/x-ndjson`)
4. Clears the batch only after a **2xx** response

### Delivery guarantees

| Situation | Behavior |
| --------- | -------- |
| Network error / non-2xx | Lines stay buffered and are retried on the next flush |
| Long outage | Buffer capped at `batch_size × 20` lines; oldest trimmed → `sink_dropped` |
| Successful 2xx | Batch cleared |

Watch stats: `buffered`, `write_errors`, `sink_dropped` (see [user guide](user-guide.md)).

### Security

- `url` must be `http` or `https` with a host (other schemes are rejected).
- Put auth tokens in `headers` from env/secret store, not from committed config.
- Header values never appear in logger error messages or stats JSON.

### Example: HTTP only (no VM disk growth from this logger)

```json
{
  "service": "payments-api",
  "environment": "prod",
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

## Grafana and Loki

### Native Loki sink (recommended)

```json
{
  "service": "payments-api",
  "environment": "prod",
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

Polyglot POSTs Loki’s push JSON (`streams` + nanosecond timestamps). Labels `service_name`, `level`, and `environment` are added automatically when not already set.

### Can I point `http.url` at Loki directly?

**Not with the HTTP NDJSON sink.** Use `"loki.enabled": true` instead, or the Vector recipe below.

### Vector/Alloy (NDJSON → Loki)

If you prefer the generic HTTP sink:

```toml
# vector.toml
[sources.polyglot]
type = "http_server"
address = "0.0.0.0:8686"
decoding.codec = "json"
framing.method = "newline_delimited"

[transforms.polyglot_remap]
type = "remap"
inputs = ["polyglot"]
source = '''
. = parse_json!(.message)
'''

[sinks.loki]
type = "loki"
inputs = ["polyglot_remap"]
endpoint = "http://loki:3100"
encoding.codec = "json"
labels.service_name = "{{ service_name }}"
labels.level = "{{ level }}"
```

Point Polyglot `http.url` at `http://vector:8686`.

### Can I point `http.url` at Loki directly? (legacy note)

**Not with the current HTTP sink.** Loki’s push API expects:

```http
POST /loki/api/v1/push
Content-Type: application/json
```

with a body shaped like:

```json
{
  "streams": [
    {
      "stream": { "service_name": "payments-api", "level": "info" },
      "values": [["<unix_ns>", "<log line or JSON string>"]]
    }
  ]
}
```

This logger ships **NDJSON batches of its own Entry objects**, which Loki will not accept as-is.

Grafana itself is the **UI**; it queries Loki (or another log backend). You still need a compatible ingest path.

### Recommended topologies

#### A. Native Loki (recommended for Grafana)

```text
App (this logger, loki sink)
        │
        ▼
      Loki  ←── Grafana
```

Or keep the generic HTTP NDJSON path and transform in a gateway:

```text
App (HTTP NDJSON) → ingest gateway (NDJSON → Loki push) → Loki ← Grafana
```

#### B. Collector sidecar (Vector, Grafana Alloy, Fluent Bit)

```text
App ──HTTP NDJSON──► Vector/Alloy (transform) ──► Loki ──► Grafana
```

Configure the collector’s HTTP source (or a tiny nginx/custom receiver) to accept NDJSON, then use its Loki sink.

#### C. File + Promtail / Alloy (uses local disk)

```text
App ──file──► Promtail/Alloy ──► Loki
```

Works, but **does not** remove VM storage pressure unless retention is tiny. Prefer A or B for “no local log files.”

#### D. Native Loki sink (preferred when talking to Loki directly)

```text
App ──loki sink──► Loki ──► Grafana
```

Use `"loki.enabled": true` (see above). Prefer this over HTTP NDJSON when the destination is Loki.

### Grafana Cloud

Prefer the native `loki` sink with Grafana Cloud’s Loki push URL and auth headers. Alternatively use Alloy/Vector in front of Cloud Loki; do not paste a Loki push URL into `http.url` (that sink speaks NDJSON, not Loki push JSON).

## OpenTelemetry note

The README example port `:4318` is commonly associated with OTLP HTTP. This sink is **not** an OTLP protobuf/JSON exporter — it is plain NDJSON of Polyglot log entries. Point it at a service that understands that format, or transform before OTLP.

## Choosing a strategy

| Constraint | Suggestion |
| ---------- | ---------- |
| Stop filling VM disks | Disable file; enable `loki` or `http` |
| Need Grafana dashboards | Native `loki` sink, or HTTP → Vector → Loki |
| Already run Vector/Alloy | HTTP NDJSON + Vector recipe above |
| Offline / air-gapped VMs | File sink with `compress: true` and tight rotation |

## Related

- [Configuration](configuration.md)
- [Troubleshooting](troubleshooting.md#http--centralized-shipping)
