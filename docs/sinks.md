# Sinks and centralized logging

A **sink** is a destination for serialized JSON log lines. The logger can write to one or more sinks at once.

## Built-in sinks

| Sink | Config | Format |
| ---- | ------ | ------ |
| Stdout | `"stdout": true` | One JSON object per line |
| File | `"file": { "enabled": true, ... }` | Same JSON lines, with size/age rotation |
| HTTP | `"http": { "enabled": true, ... }` | Batched **NDJSON** POST |

Future (interface reserved, not implemented): Kafka, syslog, OpenTelemetry native exporter, native Loki.

## Stdout

Best for containers (Docker/Kubernetes) where the platform already scrapes process output, and for local development.

## File (rotating)

Writes to `file.path`. When the active file exceeds `maxSizeMB`, it rotates. Old files are trimmed by `maxBackups` and `maxAgeDays`.

Use this when you need durable local logs. Cap retention carefully so VMs do not fill disks.

`compress` is accepted in config but **not implemented** yet.

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

### Can I point `http.url` at Loki directly?

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

#### A. Central adapter (recommended for this SDK)

```text
App (this logger, HTTP NDJSON)
        │
        ▼
  Small ingest service / gateway
  (NDJSON → Loki push JSON)
        │
        ▼
      Loki  ←── Grafana
```

The adapter maps labels from fields such as `service_name`, `environment`, `level`, and keeps the JSON line as the log payload.

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

#### D. Future native Loki sink

A dedicated `LokiSink` inside this repo would POST the correct push format. That is not implemented yet; track it as a follow-up if you want zero adapter.

### Grafana Cloud

Same idea: use Grafana Cloud’s Loki ingest URL and credentials **through an adapter or Alloy**, not by pasting the Loki push URL into `http.url` unless the body format matches.

## OpenTelemetry note

The README example port `:4318` is commonly associated with OTLP HTTP. This sink is **not** an OTLP protobuf/JSON exporter — it is plain NDJSON of Polyglot log entries. Point it at a service that understands that format, or transform before OTLP.

## Choosing a strategy

| Constraint | Suggestion |
| ---------- | ---------- |
| Stop filling VM disks | Disable file sink; enable HTTP → central store |
| Need Grafana dashboards | Loki (or similar) behind an adapter/collector |
| Already run Vector/Alloy | Prefer topology B |
| Minimal infrastructure | Topology A with a tiny convert service |
| Offline / air-gapped VMs | File sink with tight rotation, ship later |

## Related

- [Configuration](configuration.md)
- [Troubleshooting](troubleshooting.md#http--centralized-shipping)
