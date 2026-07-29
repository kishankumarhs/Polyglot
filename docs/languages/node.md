# Node.js / TypeScript guide

Package: `@eximietas/logger` (`bindings/node`).

## Install

```bash
cd bindings/node
npm install
npm run build
```

In a monorepo workspace, depend on `"@eximietas/logger": "*"` and ensure the native library is built. See [monorepo](../monorepo.md).

## Quick start

```ts
import {
  Logger,
  Level,
  libraryVersion,
  abiVersion,
  type LoggerOptions,
} from "@eximietas/logger";

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
log.warn("retrying payment", { attempt: 2 });
log.error("payment declined", { reason: "insufficient_funds" });
log.logSimple(Level.ERROR, "fallback");
log.flush();
console.log(log.stats());
log.close();
```

## Options (`LoggerOptions`)

| Option | Notes |
| ------ | ----- |
| `service` | Required for useful logs (`serviceName` deprecated) |
| `serviceVersion`, `environment`, `level` | Metadata / min level |
| `stdout` | Default `true` |
| `file` | `{ path, enabled?, maxSizeMb?, maxBackups?, maxAgeDays? }` |
| `filePath` | Deprecated shortcut for file path |
| `http` | `{ url, enabled?, timeoutMs?, headers?, batchSize?, flushIntervalMs? }` |
| `async`, `queueSize`, `overflow` | Queue behavior |
| `fields` | Base fields |

### HTTP-only example

```ts
const log = new Logger({
  service: "checkout-api",
  environment: "prod",
  stdout: false,
  http: {
    url: "https://collector.example/v1/logs",
    headers: { Authorization: `Bearer ${process.env.LOG_TOKEN}` },
    batchSize: 50,
    flushIntervalMs: 1000,
  },
});
```

## API

| Method | Description |
| ------ | ----------- |
| `trace` / `debug` / `info` / `warn` / `error` / `fatal` | Structured log |
| `log(level, message, fields?)` | Explicit level |
| `logSimple(level, message)` | No fields |
| `setFields(fields)` | Context fields |
| `reloadConfig(options)` | Hot reload |
| `stats()` | Counters object |
| `flush()` / `close()` | Drain / shutdown |

`fatal` does **not** call `process.exit`.

Errors throw `LoggerError`.

The native library loads on **first use**, not at `import` time.

## Worker threads

One `Logger` instance is safe across worker threads for log/flush/stats. Prefer a single owner for `close()`.

## Tests & example

```bash
export EXIMIETAS_LOGGER_LIB=$PWD/dist/liblogger.so
cd bindings/node && npm test
node examples/node/main.mjs
```

See also: [User guide](../user-guide.md), [Configuration](../configuration.md).
