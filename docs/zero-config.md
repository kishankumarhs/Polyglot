# Zero-config

Put `polyglot.yaml` at the **git repository root** (or any ancestor of the process cwd). Bindings discover it when you construct a `Logger`.

## Discovery order

1. `POLYGLOT_CONFIG` (aliases: `POLYGLOT_CONFIG_PATH`, `POLYGLOT_CONFIG_FILE`)
2. `polyglot.yaml` / `.yml` / `.json` in the current working directory
3. Walk parent directories until found, a `.git` directory, or the filesystem root

```text
apps/orders   ← process cwd
    ↑
apps
    ↑
repo/polyglot.yaml   ← found
repo/.git            ← walk stops here if no yaml
```

## Example

```yaml
# polyglot.yaml (repo root)
service: my-app
environment: prod
level: info
stdout: true
http:
  enabled: true
  url: http://localhost:9999
  batch_size: 10
  flush_interval_ms: 500
```

```python
from polyglot_logger import Logger
log = Logger(service="my-app")  # picks up polyglot.yaml sinks
# or, if YAML already sets service:
log = Logger()
```

```js
import { Logger } from "@polyglot-logger/node";
const log = new Logger({ service: "my-app" });
```

```csharp
using var log = new Logger(new LoggerOptions { Service = "my-app" });
```

Constructor options **override** matching YAML keys. Merge happens in the native core (same behavior in every language).

## Startup diagnostics

On create, stderr shows version, config path, precedence, and sinks (unless `POLYGLOT_QUIET=1`):

```text
polyglot-logger 0.3.0

config: /repo/polyglot.yaml
service: my-app
source: YAML + constructor overrides
http: enabled (http://localhost:9999 batch_size=10 flush_interval_ms=500)
stdout: enabled
```

## Native library path

Config discovery is separate from finding the shared library:

```bash
export POLYGLOT_LOGGER_LIB=/path/to/liblogger.so
```

Published npm / PyPI / NuGet packages bundle natives for supported platforms.

Full schema: [configuration.md](configuration.md). Side-by-side APIs: [sdk.md](sdk.md). Stuck? `go run ./cmd/polyglot doctor`.
