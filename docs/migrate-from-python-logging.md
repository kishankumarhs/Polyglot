# Migrating from Python logging / Structlog

Handlers and processors move into the native core. Fields are kwargs (or a dict), not `extra=`.

```python
# Before
import logging
logging.basicConfig(level=logging.INFO)
logging.getLogger("api").info("hello", extra={"user_id": 1})

# After
from polyglot_logger import Logger

with Logger("api", level="info", stdout=True) as log:
    log.info("hello", user_id=1)
    child = log.with_fields(request_id="r1")
    child.info("handled")
    child.close()
```

Structlog-style bind:

```python
bound = log.bind(user_id=1)  # alias of with_fields
bound.info("hello")
```

Use `polyglot-logger` 0.3.x for bundled natives. `polyglot.yaml` is loaded by the native core (no PyYAML required). Don't share `set_fields` across threads for request data — use `with_fields`.

More: [first-log.md](first-log.md) · [languages/python.md](languages/python.md)
