# ABI and codegen

## Contract

[`api/abi.json`](../api/abi.json) is the single source of truth for the public C API. Do not hand-edit generated files marked `DO NOT EDIT`.

## Generate

```bash
go run ./cmd/codegen
# or
make codegen
```

Outputs:

| File | Purpose |
| ---- | ------- |
| `native/include/logger.h` | Stable C header |
| `bindings/python/eximietas_logger/_ffi_generated.py` | ctypes signatures |
| `bindings/node/src/ffi.generated.ts` | Koffi bindings |
| `bindings/dotnet/Eximietas.Logger/NativeMethods.Generated.cs` | P/Invoke |
| `native/abi_exports.md` | Human checklist |

Codegen **fails** if `api/abi.json` and `native/export.go` disagree (missing export, undeclared export, or arg-count mismatch).

## ABI surface (v1)

| Function | Purpose |
| -------- | ------- |
| `logger_version` | Library semver string |
| `logger_abi_version` | Integer ABI version (`1`) |
| `logger_create_v1` | Create from JSON config → opaque handle |
| `logger_create` | Alias of `logger_create_v1` |
| `logger_log` | Log with level + optional fields JSON |
| `logger_log_simple` | Log without fields |
| `logger_set_fields` | Replace runtime context fields |
| `logger_reload_config` | Hot-reload config JSON |
| `logger_flush` | Drain queue + sync sinks |
| `logger_close` | Shutdown |
| `logger_stats` | Stats JSON (library-owned pointer) |
| `logger_last_error` | Last error string (`NULL` handle = global) |
| `logger_free_string` | No-op retained for binding safety |

Levels are integers (`LOGGER_TRACE` … `LOGGER_FATAL`). Return codes: `0` success, `-1` failure.

## Adding a new exported function

1. Implement behavior in `internal/logger/`
2. Add `//export` wrapper in `native/export.go`
3. Declare the function in `api/abi.json`
4. Run `make codegen`
5. Rebuild the native library
6. Optionally add an ergonomic method on each binding’s `Logger` class
7. Commit generated files so CI’s drift check stays green

## Versioning policy

- **ABI version** bumps only when the C surface is incompatible (removed/changed signatures).
- Prefer additive APIs (`logger_create_v2`, new functions) over breaking changes.
- Library **version** string can bump for behavior/bugfix without an ABI bump.

See also [`api/README.md`](../api/README.md).
