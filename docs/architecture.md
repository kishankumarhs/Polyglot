# Architecture

## Modular Monorepo Structure

Polyglot uses a **modular monorepo** pattern where the core Go implementation and each language binding are independent Git repositories connected via Git submodules.

```text
┌─────────────────────────────────────────────────────────────┐
│              polyglot-go (Core Repository)                  │
│          Single source of truth for ABI & logging           │
│                                                             │
│  api/abi.json ──→ cmd/codegen ──→ FFI for all languages   │
│      ↓              ↓                    ↓                  │
│  C ABI v1      Go implementation   Language bindings        │
└─────────────────────────────────────────────────────────────┘
      │                    │                  │
      ├────────────────────┼──────────────────┤
      │                    │                  │
      ▼                    ▼                  ▼
 ┌─────────────┐  ┌──────────────┐  ┌──────────────────┐
 │ polyglot-   │  │ polyglot-    │  │ polyglot-csharp  │
 │ node        │  │ py           │  │ (Git Submodule)  │
 │ (Git        │  │ (Git         │  │                  │
 │ Submodule)  │  │ Submodule)   │  │ npm package:     │
 │             │  │              │  │ Polyglot.Logger  │
 │ npm package:│  │ npm package: │  └──────────────────┘
 │ @polyglot/  │  │ polyglot-    │
 │ logger      │  │ logger       │
 └─────────────┘  └──────────────┘
```

**Key principle:** Each binding is a complete, independently-versioned Git repository. The core repository tracks specific commits from each binding via `.gitmodules`.

## Repository Details

| Repository          | Type      | Purpose                          | Package               |
| ------------------- | --------- | -------------------------------- | --------------------- |
| **polyglot-go**     | Core      | Go logger, ABI contract, codegen | N/A                   |
| **polyglot-node**   | Submodule | Node.js/TypeScript bindings      | npm @polyglot-logger/node  |
| **polyglot-py**     | Submodule | Python bindings                  | PyPI polyglot-logger  |
| **polyglot-csharp** | Submodule | .NET bindings                    | NuGet Polyglot.Logger |

## Core Repository Structure

| Path               | Role                                        |
| ------------------ | ------------------------------------------- |
| `api/abi.json`     | **C ABI contract** — single source of truth |
| `internal/logger/` | Go core implementation                      |
| `native/`          | CGO exports + `logger.h` generation         |
| `cmd/codegen/`     | Generates FFI for all bindings              |
| `cmd/logger-demo/` | Go demo/test program                        |
| `bindings/node/`   | Git submodule → polyglot-node               |
| `bindings/python/` | Git submodule → polyglot-py                 |
| `bindings/dotnet/` | Git submodule → polyglot-csharp             |
| `examples/`        | Example usage across languages              |
| `scripts/`         | Cross-platform native build scripts         |
| `docs/`            | Documentation hub                           |

## Code Generation Flow

The `cmd/codegen` tool is the **bridge** between core implementation and language bindings:

```text
api/abi.json (C ABI contract)
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

**Important:** Codegen-generated files are **read-only**. Hand-write SDK code in separate files.

## Async Pipeline

Async logging uses a single worker goroutine per logger instance to serialize all sink writes:

```text
Log() call (any language)
    ↓
Create Entry struct
    ↓
enqueue(entry, overflow_policy)
    ├─ drop_newest: Drop oldest if queue full
    ├─ drop_oldest: Drop newest if queue full
    └─ block: Block caller until space available
    ↓
Worker Goroutine processes continuously:
    ├─→ Dequeue entry
    ├─→ Format Entry → JSON
    ├─→ For each Sink: Sink.Write(json)
    └─→ Update stats (queued, dropped, errors)
    ↓
Flush/Close: Wait for worker to finish pending work
```

**Key:** All sink writes are serialized. No race conditions.

## Handle Lifecycle (Native)

Opaque `logger_handle` values are distinct addresses used as map keys:

```text
logger_create()
    ├─ Acquire address from handle pool
    ├─ Create new Logger struct
    ├─ Insert into global map: handles[addr] = logger
    └─ Return addr to caller

logger_log(handle, ...)
    ├─ Look up: logger = handles[handle]
    └─ If found: queue entry; else: error

logger_close(handle)
    ├─ Look up: logger = handles[handle]
    ├─ Flush queued entries
    ├─ Close sinks
    ├─ Remove from map
    └─ Mark address as retired (available for reuse)
```

**Stale handles:** If a caller uses a handle after `close()`, the lookup fails. No aliasing to a different logger.

## Codegen Verification

`cmd/codegen` ensures consistency before writing any output:

```text
For each function in api/abi.json:
    ├─ Verify //export exists in native/export.go
    ├─ Verify argument count matches
    ├─ Verify return type matches
    └─ If any mismatch: FAIL and report error

For each //export in native/export.go:
    ├─ Verify function in api/abi.json
    └─ If not found: FAIL and report error
```

This ensures the C header and all FFI bindings stay in sync.

## Thread Safety

The Go core ensures thread safety through serialization:

- **Async mode:** Single worker goroutine serializes all sink writes
- **Sync mode:** Logger mutex protects shared state
- **Bindings:** Just pass through; no additional locking needed

A single logger instance is **safe for concurrent `Log`, `Stats`, `SetFields`, `ReloadConfig` calls**. Only `Close` should be single-threaded.

## Submodule Integration Benefits

1. **Independent Versioning:** Each language can release at its own pace
2. **Clean History:** Core repo history is about Go implementation
3. **Modular Development:** Add a new language? Just add a new submodule
4. **Stable ABIs:** Bindings only depend on C function signatures
5. **Parallel Testing:** Each binding's tests run independently

## Related Documentation

- [All Repositories Overview](REPOSITORIES.md)
- [Submodule Workflow](SUBMODULE-WORKFLOW.md)
- [ABI Contract Details](abi.md)
- [Building Guide](build.md)
- [Getting Started](getting-started.md)
