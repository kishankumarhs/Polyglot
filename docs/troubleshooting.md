# Troubleshooting

## Native library not found

```bash
make build-native
export POLYGLOT_LOGGER_LIB=/abs/path/to/logger.dll   # or liblogger.so / .dylib
```

File must match the OS/arch of the process.

## gopls: no packages for `native/export.go`

Enable `CGO_ENABLED=1` and put gcc on `PATH` (see `.vscode/settings.json`).

## Race tests need CGO

```bash
export CGO_ENABLED=1
export PATH="/path/to/mingw/bin:$PATH"   # Windows
go test ./... -race
```

Prefer MinGW gcc on Windows if MSVC `/Werror` fights CGO.

## Codegen drift

```bash
make codegen
# commit regenerated files — don't hand-edit *.generated.* or logger.h
```

If CI says an export is missing from `api/abi.json`, add it (or remove the stray `//export`).

## HTTP / Loki

| Symptom | Likely cause |
| --- | --- |
| `write_errors` climbing | Collector down, bad URL, TLS/auth |
| `buffered` stuck high | Not getting 2xx |
| `sink_dropped` rising | Outage longer than retry buffer |
| Invalid `http.url` | Need `http`/`https` with a host |
| Loki rejects body | You pointed NDJSON HTTP at Loki — use `loki` sink ([sinks.md](sinks.md)) |

```bash
curl -v -H 'Content-Type: application/x-ndjson' \
  --data-binary $'{"message":"ping"}\n' "$URL"
```

Check `stats()` after a few logs + `flush()`. On shutdown, check `close()` for errors.

## Logs missing after crash

Deliberate trade-off:

| Mode | Behavior |
| --- | --- |
| `async: true` (default) | Highest throughput; lines still in the queue or HTTP buffer can be lost on hard crash |
| `async: false` | Lower throughput; wait for sinks before return — stronger durability after the call completes |

Prefer `flush` + `close` on graceful shutdown either way. Do not treat “async + no flush” as crash-safe.

## `fatal` did not exit

By design. Exit the process yourself if you need that.

## File locked on Windows (.NET)

Dispose/close the logger before reading the active log file from the same process.

## Stale handle after close

Returns an error. Drop your reference after close.

## Import works, first `Logger()` fails

Bindings load the native lib lazily. Build or set `POLYGLOT_LOGGER_LIB` before create.

## Config not loading / wrong sinks

Constructors resolve config as `POLYGLOT_CONFIG` → cwd → parent dirs (stop at `.git`). Put `polyglot.yaml` at the **git repo root**, or set `POLYGLOT_CONFIG`. Watch startup diagnostics on stderr (`POLYGLOT_QUIET=1` to silence). Run `go run ./cmd/polyglot doctor` for a checklist.

## Port already in use / TIME_WAIT (ops)

`EADDRINUSE` on 3000/8001/9999 after killing a process is a general OS/TCP issue, not Polyglot-specific. Wait for `TIME_WAIT` to clear, change the port, or use a process manager (Docker, systemd, PM2).

## Connection refused right after start (ops)

Give HTTP services a moment to bind before generating traffic, or hit a health endpoint first. Short `sleep` in scripts is fine; prefer readiness checks in real deployments.

## Curl to HTTP sink looks broken (ops)

Send **NDJSON** (`Content-Type: application/x-ndjson`), one JSON object per line ending with `\n` — not a JSON array. See [sinks.md](sinks.md).
