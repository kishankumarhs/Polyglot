# Configuration reference

Configuration is a JSON object passed to `logger_create_v1` (or built by language bindings). Preferred shape is the **nested** schema below. Legacy flat keys still parse for older configs.

## Full schema (nested)

```json
{
  "service": "workflow-service",
  "service_version": "1.0.0",
  "environment": "prod",
  "level": "info",
  "stdout": true,
  "file": {
    "enabled": true,
    "path": "./logs/app.log",
    "maxSizeMB": 100,
    "maxBackups": 10,
    "maxAgeDays": 30,
    "compress": false
  },
  "http": {
    "enabled": true,
    "url": "https://collector.example/v1/logs",
    "timeout_ms": 5000,
    "headers": {
      "Authorization": "Bearer <token>"
    },
    "batch_size": 50,
    "flush_interval_ms": 1000
  },
  "async": true,
  "queueSize": 10000,
  "overflow": "drop_newest",
  "fields": {
    "region": "ap-south-1"
  }
}
```

At least one sink must be enabled (`stdout`, `file.enabled`, or `http.enabled`). Validation fails if none are.

## Field reference

### Top level

| Key | Type | Default | Notes |
| --- | ---- | ------- | ----- |
| `service` | string | `"app"` | Becomes `service_name` on every log line. Required for meaningful queries. |
| `service_version` | string | `""` | Optional semver / build id |
| `environment` | string | `""` | e.g. `dev`, `staging`, `prod` |
| `level` | string | `"info"` | Minimum level: `trace`…`fatal` |
| `stdout` | bool | `true` | Write JSON lines to process stdout |
| `file` | object | see below | Rotating file sink |
| `http` | object | see below | Centralized NDJSON POST sink |
| `async` | bool | `true` | Background queue (fixed at create) |
| `queueSize` | int | `10000` | Async queue capacity (fixed at create) |
| `overflow` | string | `"drop_newest"` | `drop_newest` \| `drop_oldest` \| `block` |
| `fields` | object | `{}` | Base fields merged into every log |

### `file`

| Key | Type | Default | Notes |
| --- | ---- | ------- | ----- |
| `enabled` | bool | `false` | Must be true to write files |
| `path` | string | required when enabled | Active log file path |
| `maxSizeMB` | int | `100` | Rotate when size exceeds this |
| `maxBackups` | int | `10` | How many rotated files to keep |
| `maxAgeDays` | int | `30` | Delete backups older than this |
| `compress` | bool | `false` | Accepted but **not implemented** yet |

### `http`

| Key | Type | Default | Notes |
| --- | ---- | ------- | ----- |
| `enabled` | bool | `false` | Must be true to ship remotely |
| `url` | string | required when enabled | Absolute `http://` or `https://` URL with host |
| `timeout_ms` | int | `5000` | Per-request timeout |
| `headers` | object | `{}` | Extra HTTP headers (treat values as secrets) |
| `batch_size` | int | `50` | Lines per POST (also drives retry buffer size) |
| `flush_interval_ms` | int | `1000` | Periodic flush ticker |

URL schemes other than `http`/`https`, or URLs without a host, are rejected at validation.

## Legacy flat keys

Still accepted and mapped into the nested schema:

| Legacy | Maps to |
| ------ | ------- |
| `service_name` | `service` |
| `min_level` | `level` |
| `file_path` | `file.path` (+ enables file) |
| `max_size_mb` | `file.maxSizeMB` |
| `max_backups` | `file.maxBackups` |
| `max_age_days` | `file.maxAgeDays` |

Prefer the nested form for new services.

## Common presets

### Local development

```json
{
  "service": "my-api",
  "environment": "dev",
  "level": "debug",
  "stdout": true,
  "async": true
}
```

### Rotating file on a VM

```json
{
  "service": "my-api",
  "environment": "prod",
  "level": "info",
  "stdout": false,
  "file": {
    "enabled": true,
    "path": "/var/log/my-api/app.log",
    "maxSizeMB": 100,
    "maxBackups": 10,
    "maxAgeDays": 14
  }
}
```

### Central only (no local disk)

```json
{
  "service": "my-api",
  "environment": "prod",
  "level": "info",
  "stdout": false,
  "file": { "enabled": false },
  "http": {
    "enabled": true,
    "url": "https://logs.example.com/v1/ingest",
    "timeout_ms": 5000,
    "batch_size": 100,
    "flush_interval_ms": 2000,
    "headers": { "Authorization": "Bearer ${LOG_INGEST_TOKEN}" }
  },
  "async": true,
  "queueSize": 20000,
  "overflow": "drop_newest"
}
```

Replace the bearer token from the environment or a secret store — do not commit real credentials. The logger never echoes header values into error strings or stats.

## Hot reload vs recreate

| Change | Hot reload | Need new logger |
| ------ | ---------- | --------------- |
| Level, fields, overflow | Yes | No |
| Enable/disable or retarget sinks | Yes | No |
| `async` on/off | No | Yes |
| `queueSize` | No | Yes |

See also [User guide](user-guide.md) and [Sinks](sinks.md).
