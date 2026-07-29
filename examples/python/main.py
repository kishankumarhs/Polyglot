"""Minimal Python example."""

from polyglot_logger import Logger, Level

with Logger(
    "python-example",
    environment="dev",
    level="debug",
    stdout=True,
    async_mode=True,
    overflow="drop_newest",
) as log:
    log.set_fields({"traceId": "demo-trace"})
    log.info("hello from python", user_id=1)
    log.log_simple(Level.WARN, "simple warning")
    log.flush()
    print("stats", log.stats())
