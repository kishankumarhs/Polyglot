# Polyglot

Cross-language structured logger: a **Go core** compiled to a native shared library (`.so` / `.dll` / `.dylib`), plus idiomatic bindings for **Python**, **Node.js/TypeScript**, and **.NET**.

## Install (app developers)

Prefer registry packages — **no Go or CGO required**:

```bash
# Node.js / TypeScript
npm install @polyglot-logger/node

# Python
pip install polyglot-logger

# .NET
dotnet add package Polyglot.Logger
```

Drop a [`polyglot.yaml`](polyglot.yaml) in your project root for [zero-config](docs/zero-config.md) setup:

```yaml
service: my-api
environment: prod
level: info
stdout: true
stdout_format: text   # or json
caller: true
```

```bash
# Diagnose install / config / sinks
go run ./cmd/polyglot doctor --config polyglot.yaml
```

Packages: [@polyglot-logger/node](https://github.com/kishankumarhs/polyglot-node) · [polyglot-logger](https://github.com/kishankumarhs/polyglot-py) · [Polyglot.Logger](https://github.com/kishankumarhs/polyglot-csharp)

## Features

- Structured JSON logs with optional **text** console format for local dev
- Levels `trace` … `fatal` (`fatal` does not exit the process)
- **Child loggers** via `With(fields)` / `logger_with` (safe for concurrent request scopes)
- Context helpers for `trace_id` / `span_id`
- Async queue with overflow + optional **sampling**
- Sinks: stdout, rotating file (**gzip** backups), HTTP NDJSON, **native Loki push**
- Hot reload, stats, Prometheus-style `MetricsText()`, `polyglot doctor` / `validate`
- Stable C ABI v1; panics in exports are recovered (host process is not aborted)

## Documentation

| Guide | Description |
| ----- | ----------- |
| [Zero-config](docs/zero-config.md) | `polyglot.yaml` auto-discovery |
| [Getting started](docs/getting-started.md) | First log + contributor build |
| [User guide](docs/user-guide.md) | Levels, fields, async, stats |
| [Configuration](docs/configuration.md) | Full schema (`strict`, `sampling`, `loki`, …) |
| [Sinks & Loki/Grafana](docs/sinks.md) | HTTP, Loki, Vector recipe |
| [Repositories](docs/REPOSITORIES.md) | Core + binding submodule layout |
| [Submodule workflow](docs/SUBMODULE-WORKFLOW.md) | Contributor clone (`--recurse-submodules`) |

## Contributor quick start

```bash
git clone --recurse-submodules https://github.com/kishankumarhs/logger.git
cd logger
bash scripts/check-submodules.sh
make build-native          # needs Go + CGO + gcc/clang/MinGW
go test ./internal/logger -race
go run ./cmd/polyglot doctor
```

If bindings look empty after a plain `git clone`, run:

```bash
git submodule update --init --recursive
```

Set `POLYGLOT_LOGGER_LIB` only when developing against a locally built native library.

### Central logs (Loki)

```json
{
  "service": "my-api",
  "stdout": false,
  "loki": {
    "enabled": true,
    "url": "http://loki:3100/loki/api/v1/push",
    "batch_size": 50,
    "flush_interval_ms": 1000
  }
}
```

Or keep the HTTP NDJSON sink and use the [Vector recipe](docs/sinks.md#vectoralloy-ndjson--loki).

## License

MIT
