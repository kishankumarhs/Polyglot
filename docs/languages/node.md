# Node.js / TypeScript Guide

**Package:** `@polyglot-logger/node` ([npm](https://www.npmjs.com/package/@polyglot-logger/node))  
**Repository:** [polyglot-node](https://github.com/kishankumarhs/polyglot-node) (Git submodule)  
**Status:** Independent package with bundled native binaries

## Overview

This binding provides idiomatic Node.js/TypeScript access to the Polyglot native logger. It's part of a [modular monorepo](../architecture.md) where:

- **Core:** Go logger in [polyglot-go](https://github.com/kishankumarhs/Polyglot)
- **This binding:** Independent Node.js repository with its own releases
- **Generated code:** FFI bindings auto-generated from C ABI contract
- **Native binaries:** Pre-compiled for Windows/macOS/Linux, bundled in npm package

See [REPOSITORIES.md](../REPOSITORIES.md) for how all four repositories work together.

## Installation

### From npm (Recommended)

```bash
npm install @polyglot-logger/node
```

Pre-compiled native binaries are included. No build needed.

### From Source (Core Development)

If working on the core Polyglot repository:

```bash
git clone --recurse-submodules https://github.com/kishankumarhs/Polyglot.git
cd Polyglot

make build-native
cd bindings/node
npm install
npm run build
```

## Quick Start

```ts
import {
  Logger,
  Level,
  libraryVersion,
  abiVersion,
  type LoggerOptions,
} from "@polyglot-logger/node";

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

## Configuration

### Via polyglot.yaml

Create `polyglot.yaml` in project root (auto-discovered):

```yaml
service: checkout-api
environment: prod
logging:
  level: info
  async: true
file:
  enabled: true
  path: /var/log/checkout.log
  max_size_mb: 100
http:
  enabled: false
  endpoint: https://logs.company.com/ingest
```

### Programmatic Configuration

```ts
const log = new Logger({
  service: "checkout-api",
  environment: "prod",
  level: "info",
  async: true,
  stdout: false,
  file: {
    enabled: true,
    path: "/var/log/checkout.log",
    maxSizeMb: 100,
  },
});
```

### Environment Variables

```bash
export POLYGLOT_CONFIG_PATH=/etc/myapp/polyglot.yaml
export POLYGLOT_CONFIG_FILE=/etc/myapp/config.json
```

## API Reference

### Logger Constructor

## Options (`LoggerOptions`)

| Option                                   | Notes                                                                   |
| ---------------------------------------- | ----------------------------------------------------------------------- |
| `service`                                | Required for useful logs (`serviceName` deprecated)                     |
| `serviceVersion`, `environment`, `level` | Metadata / min level                                                    |
| `stdout`                                 | Default `true`                                                          |
| `file`                                   | `{ path, enabled?, maxSizeMb?, maxBackups?, maxAgeDays? }`              |
| `filePath`                               | Deprecated shortcut for file path                                       |
| `http`                                   | `{ url, enabled?, timeoutMs?, headers?, batchSize?, flushIntervalMs? }` |
| `async`, `queueSize`, `overflow`         | Queue behavior                                                          |
| `fields`                                 | Base fields                                                             |

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

| Method                                                  | Description      |
| ------------------------------------------------------- | ---------------- |
| `trace` / `debug` / `info` / `warn` / `error` / `fatal` | Structured log   |
| `log(level, message, fields?)`                          | Explicit level   |
| `logSimple(level, message)`                             | No fields        |
| `setFields(fields)`                                     | Context fields   |
| `reloadConfig(options)`                                 | Hot reload       |
| `stats()`                                               | Counters object  |
| `flush()` / `close()`                                   | Drain / shutdown |

`fatal` does **not** call `process.exit`.

Errors throw `LoggerError`.

The native library loads on **first use**, not at `import` time.

## Worker threads

One `Logger` instance is safe across worker threads for log/flush/stats. Prefer a single owner for `close()`.

## Tests & example

```bash
export POLYGLOT_LOGGER_LIB=$PWD/dist/liblogger.so
cd bindings/node && npm test
node examples/node/main.mjs
```

See also: [User guide](../user-guide.md), [Configuration](../configuration.md).
