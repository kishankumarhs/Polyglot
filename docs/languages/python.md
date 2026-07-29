# Python guide

Package: `polyglot-logger` (`bindings/python`).

## Install

```bash
# from repo root, after make build-native
pip install -e bindings/python
```

Ensure the native library is discoverable (`dist/` relative to the repo, package `native/`, or `POLYGLOT_LOGGER_LIB`).

## Quick start

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

## Constructor options

| Parameter | Maps to config |
| --------- | -------------- |
| `service` (positional) | `service` |
| `service_version` | `service_version` |
| `environment` | `environment` |
| `level` | `level` |
| `stdout` | `stdout` |
| `file` / `file_path` + size/backup/age | `file` |
| `http` | `http` dict (`url`, `timeout_ms`, `headers`, …) |
| `async_mode` | `async` |
| `queue_size` | `queueSize` |
| `overflow` | `overflow` |
| `fields` | `fields` |

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

| Method | Description |
| ------ | ----------- |
| `trace` / `debug` / `info` / `warn` / `error` / `fatal` | Structured log; kwargs → fields |
| `log(level, message, **fields)` | Explicit level |
| `log_simple(level, message)` | No extra fields |
| `set_fields(mapping)` | Replace context fields |
| `reload_config(...)` | Hot reload (same kwargs as create helpers) |
| `stats()` | Counter dict |
| `flush()` / `close()` | Drain / shutdown |

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
