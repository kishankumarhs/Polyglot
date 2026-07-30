# SDK comparison

Same concepts across languages. Config shape (`polyglot.yaml` / JSON) is shared; call syntax follows each language.

Current packages are **0.3.x**. Check the registry for the latest patch before pinning.

| Concept | Node | Python | .NET |
| --- | --- | --- | --- |
| Install | `npm install @polyglot-logger/node` | `pip install polyglot-logger` | `dotnet add package Polyglot.Logger` |
| Create | `new Logger({ service: "orders" })` | `Logger(service="orders")` | `new Logger(new LoggerOptions { Service = "orders" })` |
| Info | `log.info(msg, { user_id: 1 })` | `log.info(msg, user_id=1)` | `log.Info(msg, dict)` |
| Context | `log.with({ requestId })` | `log.with_fields(requestId=...)` | `log.With(dict)` |
| Shutdown | `log.close()` | `log.close()` / `with` | `Dispose()` / `using` |

## Create (equivalent)

```js
import { Logger } from "@polyglot-logger/node";
const log = new Logger({ service: "orders", environment: "dev" });
```

```python
from polyglot_logger import Logger
log = Logger(service="orders", environment="dev")
```

```csharp
using var log = new Logger(new LoggerOptions {
    Service = "orders",
    Environment = "dev",
});
```

Canonical option name is **`service`**. Deprecated aliases (`serviceName`, `service_name`, `ServiceName`) still work but print a one-time warning.

## Config file

Constructors load `polyglot.yaml` automatically:

1. `POLYGLOT_CONFIG` (also `POLYGLOT_CONFIG_PATH` / `POLYGLOT_CONFIG_FILE`)
2. cwd
3. parent directories (stops at `.git`)

Constructor options override file keys. `service` may come from YAML alone:

```js
new Logger({})  // OK if polyglot.yaml has service:
```

See [zero-config.md](zero-config.md) · [monorepo.md](monorepo.md) · [first-log.md](first-log.md).

When setup feels wrong:

```bash
go run ./cmd/polyglot doctor
# or: make doctor
```
