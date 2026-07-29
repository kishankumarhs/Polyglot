# Eximietas Logger (Node.js / TypeScript)

Idiomatic Node/TypeScript bindings (`@eximietas/logger`) for the Eximietas native structured logger.

**Full guide:** [docs/languages/node.md](../../docs/languages/node.md) · [User guide](../../docs/user-guide.md) · [Getting started](../../docs/getting-started.md)

```ts
import { Logger, Level } from "@eximietas/logger";

const log = new Logger({
  service: "checkout-api",
  environment: "prod",
  filePath: "/var/log/checkout.log",
});

log.setFields({ traceId: "abc" });
log.info("checkout started", { cartId: "c-1" });
log.logSimple(Level.ERROR, "payment declined");
console.log(log.stats());
log.close();
```

```bash
make build-native
cd bindings/node && npm install && npm run build
```

**Thread safety:** concurrent `log` / `flush` / `stats` / `setFields` / `reloadConfig` on one instance is safe; prefer a single owner for `close()`.

The native library loads on first use. Set `EXIMIETAS_LOGGER_LIB` if auto-discovery fails.
