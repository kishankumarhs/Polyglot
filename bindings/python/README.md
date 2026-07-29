# Eximietas Logger (Python)

Idiomatic Python bindings for the Eximietas native structured logger.

**Full guide:** [docs/languages/python.md](../../docs/languages/python.md) · [User guide](../../docs/user-guide.md) · [Getting started](../../docs/getting-started.md)

```python
from eximietas_logger import Logger, Level

with Logger("payments-api", environment="prod", file_path="/var/log/payments.log") as log:
    log.set_fields({"traceId": "abc"})
    log.info("order created", order_id=123, amount=42.5)
    log.log_simple(Level.ERROR, "payment failed")
    print(log.stats())
```

```bash
make build-native
pip install -e bindings/python
```

**Thread safety:** concurrent `info` / `flush` / `stats` / `set_fields` / `reload_config` on one instance is safe; call `close()` once from a single owner.

Set `EXIMIETAS_LOGGER_LIB` to the absolute path of `logger.dll`, `liblogger.so`, or `liblogger.dylib` if the library is not found automatically.
