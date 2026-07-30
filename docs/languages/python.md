# Python

Package: [`polyglot-logger`](https://pypi.org/project/polyglot-logger/) · repo: [polyglot-py](https://github.com/kishankumarhs/polyglot-py)

```bash
pip install polyglot-logger
# YAML config helpers: pip install polyglot-logger[yaml]
```

From this monorepo:

```bash
make build-native
pip install -e bindings/python
```

## Example

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
    log.error("charge failed", code="CARD_DECLINED")
    log.flush()
    print(log.stats())
```

`__exit__` calls `close()`.

## Config

```yaml
# polyglot.yaml
service: payments-api
environment: prod
level: info
async: true
file:
  enabled: true
  path: /var/log/payments.log
```

Env: `POLYGLOT_CONFIG_PATH`, `POLYGLOT_CONFIG_FILE`.

HTTP-only:

```python
with Logger(
    "payments-api",
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

## Constructor

| Parameter | Maps to |
| --- | --- |
| `service` (positional) | `service` |
| `service_version`, `environment`, `level` | same |
| `stdout`, `file` / `file_path`, `http` | sinks |
| `async_mode`, `queue_size`, `overflow` | queue |
| `fields` | base fields |

## Methods

| Method | Notes |
| --- | --- |
| `trace` … `fatal` | kwargs → fields; `fatal` does not exit |
| `log(level, message, **fields)` | Explicit level |
| `with_fields` / `bind` | Child logger |
| `set_fields` / `reload_config` | Context / hot reload |
| `stats` / `flush` / `close` | Observability / lifecycle |

Errors raise `LoggerError`. Safe across threads for log/flush/stats; close once.

```bash
POLYGLOT_LOGGER_LIB=$PWD/dist/liblogger.so pytest bindings/python/tests -q
python examples/python/main.py
```

See [user-guide.md](../user-guide.md) · [configuration.md](../configuration.md).
