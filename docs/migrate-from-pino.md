# Migrating from Pino

Argument order flips: Pino is `(fields, message)`, Polyglot is `(message, fields)`. Child loggers map as `child()` → `with()`.

```js
// Before
import pino from "pino";
const log = pino({ level: "info" });
const child = log.child({ reqId: "r1" });
child.info({ userId: 1 }, "hello");

// After
import { Logger } from "@polyglot-logger/node";
const log = new Logger({ service: "api", level: "info", stdout: true });
const child = log.with({ reqId: "r1" });
child.info("hello", { userId: 1 });
child.close();
log.close();
```

Transports become config sinks (stdout, file, HTTP, Loki). Don't use `setFields` for per-request data — use `with()`. Call `flush()` / `close()` on shutdown in short scripts.

More: [first-log.md](first-log.md) · [compatibility.md](compatibility.md)
