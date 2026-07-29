# Polyglot

Cross-language structured logger: a **Go core** compiled to a native shared library (`.so` / `.dll` / `.dylib`), plus idiomatic bindings for **Python**, **Node.js/TypeScript**, and **.NET**.

One place owns JSON formatting, async queuing, file rotation, and HTTP shipping. Language SDKs stay thin wrappers over a stable **C ABI v1**.

## 📚 Documentation

**Start with:** [`docs/REPOSITORIES.md`](docs/REPOSITORIES.md) for the modular architecture overview

| Guide                                                                                                                                                          | Description                              |
| -------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------- |
| [Repositories](docs/REPOSITORIES.md)                                                                                                                           | All repositories & how they connect      |
| [Submodule Workflow](docs/SUBMODULE-WORKFLOW.md)                                                                                                               | Development with Git submodules          |
| [Quick Reference](SUBMODULE-QUICK-REFERENCE.md)                                                                                                                | Fast lookup for common tasks             |
| [Getting started](docs/getting-started.md)                                                                                                                     | Build the library and run your first log |
| [User guide](docs/user-guide.md)                                                                                                                               | Levels, fields, async, stats, lifecycle  |
| [Configuration](docs/configuration.md)                                                                                                                         | Full JSON schema and presets             |
| [Sinks & Loki/Grafana](docs/sinks.md)                                                                                                                          | Stdout, file, HTTP centralization        |
| [Python](docs/languages/python.md) · [Node](docs/languages/node.md) · [.NET](docs/languages/dotnet.md) · [Go](docs/languages/go.md) · [C](docs/languages/c.md) | Language APIs                            |
| [Build](docs/build.md) · [Architecture](docs/architecture.md) · [Troubleshooting](docs/troubleshooting.md)                                                     | Operators & maintainers                  |

## 🏗️ Repository Structure

```
polyglot-go (core)                  — Single source of truth for ABI & implementation
├── bindings/node                   — Git submodule → polyglot-node
├── bindings/python                 — Git submodule → polyglot-py
└── bindings/dotnet                 — Git submodule → polyglot-csharp
```

**Independent packages:**

- 📦 npm: `@polyglot/logger` ([polyglot-node](https://github.com/kishankumarhs/polyglot-node))
- 📦 PyPI: `polyglot-logger` ([polyglot-py](https://github.com/kishankumarhs/polyglot-py))
- 📦 NuGet: `Polyglot.Logger` ([polyglot-csharp](https://github.com/kishankumarhs/polyglot-csharp))

## Features

- Structured JSON logs (`timestamp`, `level`, `message`, `service_name`, fields)
- Levels `trace` … `fatal` (integer ABI enum; `fatal` does not exit the process)
- Async queue with overflow policies: `drop_newest` (default), `drop_oldest`, `block`
- Sinks: stdout, rotating file, HTTP NDJSON (central collector)
- Hot reload, context fields, runtime stats (`queued`, `dropped`, `write_errors`, …)
- Codegen from [`api/abi.json`](api/abi.json) → C header + FFI bindings

## Quick start

```bash
make build-native          # needs Go + CGO + gcc/clang/MinGW
pip install -e bindings/python
python examples/python/main.py
```

Set `POLYGLOT_LOGGER_LIB` to the absolute path of the shared library if auto-discovery fails.

### Minimal config (stdout)

```json
{
  "service": "my-api",
  "environment": "dev",
  "level": "info",
  "stdout": true,
  "async": true
}
```

### Central logs only (no local files)

```json
{
  "service": "my-api",
  "environment": "prod",
  "stdout": false,
  "file": { "enabled": false },
  "http": {
    "enabled": true,
    "url": "https://collector.example/v1/logs",
    "batch_size": 50,
    "flush_interval_ms": 1000
  }
}
```

> **Loki / Grafana:** the HTTP sink posts **NDJSON**, not Loki’s push JSON. Use an adapter or Vector/Alloy — see [docs/sinks.md](docs/sinks.md).

## Example log line

```json
{
  "timestamp": "2026-07-29T07:00:00.123456789Z",
  "level": "info",
  "message": "order created",
  "service_name": "payments-api",
  "environment": "prod",
  "fields": { "order_id": 123, "amount": 42.5 }
}
```

## Repository layout

```text
logger/
├── docs/                 ← user guide & references
├── api/abi.json          ← ABI contract (codegen input)
├── cmd/codegen/          ← generates header + FFI
├── cmd/logger-demo/
├── internal/logger/      ← Go core
├── native/               ← CGO exports + logger.h
├── bindings/{node,python,dotnet}/
├── examples/
├── scripts/
└── .github/workflows/
```

## License

MIT (see package metadata in each binding).
