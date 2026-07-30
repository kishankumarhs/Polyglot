# Node.js / TypeScript

Package: [`@polyglot-logger/node`](https://www.npmjs.com/package/@polyglot-logger/node) · repo: [polyglot-node](https://github.com/kishankumarhs/polyglot-node)

```bash
npm install @polyglot-logger/node
```

From this monorepo:

```bash
make build-native
cd bindings/node && npm install && npm run build
```

## Example

```ts
import { Logger, Level, libraryVersion, abiVersion } from "@polyglot-logger/node";

console.log(libraryVersion(), abiVersion());

const log = new Logger({
  service: "checkout-api",
  environment: "prod",
  level: "info",
  stdout: false,
  file: { path: "/var/log/checkout.log", maxSizeMb: 100 },
  async: true,
  overflow: "drop_newest",
});

log.setFields({ traceId: "abc" });
log.info("checkout started", { cartId: "c-1" });
log.error("payment declined", { reason: "insufficient_funds" });
log.flush();
console.log(log.stats());
log.close();
```

## Config

```yaml
# polyglot.yaml — walked up from cwd
service: checkout-api
environment: prod
level: info
async: true
file:
  enabled: true
  path: /var/log/checkout.log
  max_size_mb: 100
```

Or set options in the constructor (they override the file). Env: `POLYGLOT_CONFIG` (aliases `POLYGLOT_CONFIG_PATH` / `POLYGLOT_CONFIG_FILE`). Discovery walks cwd → parents and stops at `.git`. See [sdk.md](../sdk.md) · [zero-config.md](../zero-config.md).

HTTP-only:

```ts
const log = new Logger({
  service: "checkout-api",
  stdout: false,
  http: {
    url: "https://collector.example/v1/logs",
    headers: { Authorization: `Bearer ${process.env.LOG_TOKEN}` },
    batchSize: 50,
    flushIntervalMs: 1000,
  },
});
```

## Options

| Option | Notes |
| --- | --- |
| `service` | Becomes `service_name` on each line |
| `serviceVersion`, `environment`, `level` | Metadata / min level |
| `stdout` | Default `true` |
| `file` | `{ path, enabled?, maxSizeMb?, maxBackups?, maxAgeDays? }` |
| `http` | `{ url, enabled?, timeoutMs?, headers?, batchSize?, flushIntervalMs? }` |
| `async`, `queueSize`, `overflow` | Queue |
| `fields` | Base fields |

## Methods

| Method | Notes |
| --- | --- |
| `trace` … `fatal` | Structured log; `fatal` does not exit |
| `log(level, message, fields?)` | Explicit level |
| `logSimple(level, message)` | No fields |
| `with(fields)` | Child logger |
| `setFields` / `reloadConfig` | Context / hot reload |
| `stats` / `flush` / `close` | Observability / lifecycle |

Errors throw `LoggerError`. Native lib loads on first use, not at `import`.

One instance is fine across worker threads for log/flush/stats; close from one owner.

```bash
export POLYGLOT_LOGGER_LIB=$PWD/dist/logger.dll   # or .so / .dylib
cd bindings/node && npm test
node examples/node/main.mjs
```

See [user-guide.md](../user-guide.md) · [configuration.md](../configuration.md).
