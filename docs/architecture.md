# Architecture

Core Go logger + C ABI live in this repo. Language bindings are separate Git repos, pulled in as submodules under `bindings/`.

```text
Polyglot (core)
  api/abi.json → cmd/codegen → generated FFI stubs
  internal/logger → native shared lib (.so / .dll / .dylib)
  bindings/
    node   → polyglot-node   (@polyglot-logger/node)
    python → polyglot-py     (polyglot-logger)
    dotnet → polyglot-csharp (Polyglot.Logger)
```

Each binding versions and publishes on its own. The core repo pins submodule commits in `.gitmodules`.

## Repositories

| Repo | Role | Package |
| --- | --- | --- |
| Polyglot (this) | Go core, ABI, codegen | — |
| polyglot-node | Node bindings | npm `@polyglot-logger/node` |
| polyglot-py | Python bindings | PyPI `polyglot-logger` |
| polyglot-csharp | .NET bindings | NuGet `Polyglot.Logger` |

## Layout

| Path | Role |
| --- | --- |
| `api/abi.json` | C ABI contract |
| `internal/logger/` | Go core |
| `native/` | CGO exports + `logger.h` |
| `cmd/codegen/` | Generates FFI stubs |
| `cmd/logger-demo/` | Go demo |
| `bindings/*/` | Submodules |
| `examples/` | Samples |
| `scripts/` | Native build |
| `docs/` | Docs |

## Codegen

```text
api/abi.json
        │
        ▼ go run ./cmd/codegen
        │
        ├─→ Parses abi.json
        ├─→ Verifies consistency with native/export.go
        ├─→ Generates native/include/logger.h
        │
        └─→ For each language binding:
            ├─→ bindings/node/src/ffi.generated.ts
            ├─→ bindings/python/polyglot_logger/_ffi_generated.py
            └─→ bindings/dotnet/Polyglot.Logger/NativeMethods.Generated.cs
```

Generated files are read-only. Put hand-written SDK code in separate files.

## Async pipeline

One worker goroutine per logger drains the queue and writes sinks (no concurrent sink writes).

```text
Log() → serialize → enqueue (drop_newest / drop_oldest / block)
                 → worker → sinks → stats
Flush/Close waits for the worker.
```

## Handles

Opaque `logger_handle` values key a map of live loggers. After `close`, lookups fail — no reuse of a closed address for a different logger.

## Thread safety

Async: worker serializes sinks. Sync: mutex on shared state. Bindings pass through. Concurrent `Log` / `Stats` / `SetFields` / `ReloadConfig` are fine; `Close` from one owner.

## Related

- [REPOSITORIES.md](REPOSITORIES.md)
- [SUBMODULE-WORKFLOW.md](SUBMODULE-WORKFLOW.md)
- [abi.md](abi.md)
- [build.md](build.md)
- [getting-started.md](getting-started.md)
