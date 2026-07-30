# Pino → Loki (for writeups)

Polyglot's architecture bench measures:

```text
Node app → @polyglot-logger/node (FFI) → native Loki sink → mock /loki/api/v1/push
```

A typical Pino deploy looks like:

```text
Node app → pino → stdout/file → Promtail or Alloy → Loki
```

We don't simulate that multi-hop path as an equal peer. In-process Polyglot→Loki vs Pino→file only compares serialize+disk, not Promtail. Mention this when quoting architecture results.
