# Python Guide

**Package:** `polyglot-logger` ([PyPI](https://pypi.org/project/polyglot-logger/))  
**Repository:** [polyglot-py](https://github.com/kishankumarhs/polyglot-py) (Git submodule)  
**Status:** Independent package with bundled native binaries

## Overview

This binding provides idiomatic Python access to the Polyglot native logger. It's part of a [modular monorepo](../architecture.md) where:

- **Core:** Go logger in [polyglot-go](https://github.com/kishankumarhs/Polyglot)
- **This binding:** Independent Python repository with its own releases
- **Generated code:** FFI bindings auto-generated from C ABI contract
- **Native binaries:** Pre-compiled for Windows/macOS/Linux, bundled in wheel

See [REPOSITORIES.md](../REPOSITORIES.md) for how all four repositories work together.

## Installation

### From PyPI (Recommended)

```bash
pip install polyglot-logger
```

Pre-compiled native binaries are included. No build needed.

### From Source (Core Development)

If working on the core Polyglot repository:

```bash
git clone --recurse-submodules https://github.com/kishankumarhs/Polyglot.git
cd Polyglot

make build-native
pip install -e bindings/python
```

## Quick Start

```python
from polyglot_logger import Logger, Level, library_version, abi_version

print(library_version(), abi_version())

with Logger(
    "payments-api",
    environment="prod",
    level="info",
    stdout=False,
    file_path="/var/log/payments.log",
    async_mode=True,
    overflow="drop_newest",
) as log:
    log.set_fields({"traceId": "abc"})
    log.info("order created", order_id=123, amount=42.5)
    log.warn("slow payment", latency_ms=800)
    log.error("charge failed", code="CARD_DECLINED")
    log.log_simple(Level.ERROR, "fallback message")
    log.flush()
    print(log.stats())
```

`Logger` is a context manager: `__exit__` calls `close()`.

## Configuration

### Via polyglot.yaml

Create `polyglot.yaml` in project root (auto-discovered):

```yaml
service: payments-api
environment: prod
logging:
  level: info
  async: true
file:
  enabled: true
  path: /var/log/payments.log
http:
  enabled: false
  endpoint: https://logs.company.com/ingest
```

### Programmatic Configuration

```python
log = Logger(
    service="payments-api",
    environment="prod",
    level="info",
    async_mode=True,
    file_path="/var/log/payments.log",
)
```

### Environment Variables

```bash
export POLYGLOT_CONFIG_PATH=/etc/myapp/polyglot.yaml
export POLYGLOT_CONFIG_FILE=/etc/myapp/config.json
```

## API Reference

### Logger Constructor

## Constructor Options

| Parameter                              | Maps to config                                  |
| -------------------------------------- | ----------------------------------------------- |
| `service` (positional)                 | `service`                                       |
| `service_version`                      | `service_version`                               |
| `environment`                          | `environment`                                   |
| `level`                                | `level`                                         |
| `stdout`                               | `stdout`                                        |
| `file` / `file_path` + size/backup/age | `file`                                          |
| `http`                                 | `http` dict (`url`, `timeout_ms`, `headers`, …) |
| `async_mode`                           | `async`                                         |
| `queue_size`                           | `queueSize`                                     |
| `overflow`                             | `overflow`                                      |
| `fields`                               | `fields`                                        |

### HTTP-only example

```python
with Logger(
    "payments-api",
    environment="prod",
    stdout=False,
    http={
        "enabled": True,
        "url": "https://collector.example/v1/logs",
        "headers": {"Authorization": f"Bearer {token}"},
        "batch_size": 50,
        "flush_interval_ms": 1000,
    },
) as log:
    log.info("shipped remotely")
```

## API

| Method                                                  | Description                                |
| ------------------------------------------------------- | ------------------------------------------ |
| `trace` / `debug` / `info` / `warn` / `error` / `fatal` | Structured log; kwargs → fields            |
| `log(level, message, **fields)`                         | Explicit level                             |
| `log_simple(level, message)`                            | No extra fields                            |
| `set_fields(mapping)`                                   | Replace context fields                     |
| `reload_config(...)`                                    | Hot reload (same kwargs as create helpers) |
| `stats()`                                               | Counter dict                               |
| `flush()` / `close()`                                   | Drain / shutdown                           |

`fatal` does **not** exit the process.

Errors raise `LoggerError`.

## Thread safety

Safe to call log/flush/stats/set_fields/reload from multiple threads. Close once from a single owner.

## Tests & example

```bash
POLYGLOT_LOGGER_LIB=$PWD/dist/liblogger.so pytest bindings/python/tests -q
python examples/python/main.py
```

See also: [User guide](../user-guide.md), [Configuration](../configuration.md).
