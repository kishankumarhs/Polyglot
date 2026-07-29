# Architecture

## Overview

```text
api/abi.json
     │
     ▼  go run ./cmd/codegen
┌────────────────────────────────────────────┐
│  native/include/logger.h                   │
│  bindings/*/… FFI generated files          │
└────────────────────────────────────────────┘
     │
Language SDK (hand-written ergonomics)
     │
     ▼
C ABI v1  (shared library: .so / .dll / .dylib)
     │
     ▼
Go core  (internal/logger)
     │
     ├─ async queue + overflow
     ├─ JSON Entry serialization
     └─ sinks: stdout | file | http
```

Goals:

- **Single source of truth** for formatting, rotation, async, and shipping
- **Stable ABI** so internals can change without breaking consumers
- **Thin bindings** — no duplicated log logic in Python/Node/.NET

## Packages

| Path | Role |
| ---- | ---- |
| `internal/logger` | Core implementation |
| `native/` | CGO `//export` wrappers + generated header |
| `api/abi.json` | ABI contract for codegen |
| `cmd/codegen` | Generates header + FFI stubs |
| `cmd/logger-demo` | Go demo binary |
| `bindings/{python,node,dotnet}` | Idiomatic SDKs |
| `examples/` | Minimal consumers |
| `scripts/` | Native build |
| `docs/` | This documentation |

## Async pipeline

```text
Log() → serialize Entry → enqueue(overflow policy)
                              │
                         worker goroutine
                              │
                    writePayload → each Sink.Write
                              │
                    Flush / Close → flushSinks + closeSinks
```

## Handle lifecycle (native)

Opaque `logger_handle` values are distinct C addresses used as map keys (never dereferenced by callers).

- On create: acquire an address from a handle pool
- On close: remove from map, free string buffers, **retire** the address
- Stale handles resolve to “invalid logger handle” rather than aliasing another logger (until addresses recycle after 1024 retirements)

## Codegen boundary

| Hand-written | Generated |
| ------------ | --------- |
| `internal/logger/*` | `native/include/logger.h` |
| `native/export.go` | `*_ffi_generated.py`, `ffi.generated.ts`, `NativeMethods.Generated.cs` |
| Binding `Logger` classes | `native/abi_exports.md` checklist |

`cmd/codegen` verifies that every `api/abi.json` function has a matching `//export` (and vice versa), including argument count, **before** writing outputs.

## Thread safety model

The Go core serializes sink access through the worker (async) or through logger locks (sync). Bindings pass through; they do not add their own global locks beyond what the native library provides.

## Related

- [ABI & codegen](abi.md)
- [Build](build.md)
