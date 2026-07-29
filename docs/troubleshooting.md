# Troubleshooting

## Native library not found

**Symptom:** `unable to load native logger library` / DllNotFound / similar.

**Fix:**

1. Build: `make build-native`
2. Set absolute path:
   ```bash
   export EXIMIETAS_LOGGER_LIB=/abs/path/to/logger.dll   # or liblogger.so / .dylib
   ```
3. Confirm the file exists and matches the OS/arch of the process (don’t load a Linux `.so` into Windows Python).

## gopls: “No packages found” for `native/export.go`

CGO is required for the `native` package. Enable `CGO_ENABLED=1` and put gcc on `PATH` for the editor (see `.vscode/settings.json`).

## CGO / race tests fail

```text
go: -race requires cgo; enable cgo by setting CGO_ENABLED=1
```

```bash
export CGO_ENABLED=1
export PATH="/path/to/mingw/bin:$PATH"   # Windows
go test ./... -race
```

## MSVC `/Werror` with CGO

Prefer MinGW gcc for CGO on Windows if MSVC flags conflict. The project’s build scripts assume a gcc-compatible toolchain.

## Codegen drift in CI

```text
codegen drift: run go run ./cmd/codegen and commit
```

Run `make codegen` locally and commit the regenerated files. Do not hand-edit `*.generated.*` or `logger.h`.

```text
ABI mismatch: … exported from native/export.go but missing from api/abi.json
```

Add the function to `api/abi.json` (or remove the stray `//export`).

## HTTP / centralized shipping

| Symptom | Likely cause |
| ------- | ------------ |
| `write_errors` climbing | Collector down, wrong URL, TLS/auth failure |
| `buffered` stuck high | Collector not returning 2xx; batches retained |
| `sink_dropped` rising | Outage longer than retry buffer capacity |
| Create fails: invalid `http.url` | Must be `http`/`https` with a host |
| Loki rejects body | NDJSON ≠ Loki push format — need adapter (see [sinks](sinks.md)) |

Debug checklist:

1. `curl -v -H 'Content-Type: application/x-ndjson' --data-binary $'{"message":"ping"}\n' "$URL"`
2. Inspect `stats()` after a few logs and a `flush()`
3. Confirm headers/token without logging them
4. On shutdown, check whether `close()` reports an error

## Logs missing after crash

Async mode may still have entries in the queue or HTTP buffer. Prefer graceful shutdown with `flush` + `close`. For crash-only durability, use sync mode (`"async": false`) or accept possible loss under `drop_*` policies.

## `fatal` did not exit

By design. `fatal` is a severity label. Exit the process yourself if needed.

## File locked / cannot read while logging (.NET / Windows)

Dispose/`Close` the logger before reading the active log file from the same process. The file sink keeps the file open while the logger is alive.

## Stale handle after close

Using a closed handle returns an error (`invalid logger handle`). After many create/close cycles addresses may eventually recycle; do not rely on stale-handle detection in a tight loop — always drop your reference after close.

## Import works but first `Logger()` crashes

Bindings load the native library lazily. Import succeeds without the `.so`/`.dll`; the first create/version call fails if it is missing. Build or set `EXIMIETAS_LOGGER_LIB` before creating a logger.
