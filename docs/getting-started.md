# Getting started

This guide takes you from a clean checkout to your first structured log line in under ten minutes.

## Prerequisites

| Tool | Notes |
| ---- | ----- |
| Go 1.22+ | Required to build the native library and run Go tests |
| C toolchain | gcc (Linux/macOS), MinGW or MSVC (Windows) — CGO must be enabled |
| Optional | Python 3.9+, Node 18+, .NET 8+ depending on which binding you use |

On Windows with Scoop MinGW, ensure gcc is on `PATH` (for example `C:\Users\<you>\scoop\apps\mingw\current\bin`) and set `CGO_ENABLED=1`.

## 1. Clone and build the native library

```bash
cd logger
make build-native
# equivalent: go run ./cmd/codegen && bash scripts/build-native.sh dist
```

Artifacts appear in `dist/`:

| Platform | File |
| -------- | ---- |
| Linux | `liblogger.so` |
| Windows | `logger.dll` |
| macOS | `liblogger.dylib` |

Also produced: `dist/logger.h`, `dist/checksums.sha256`. Binding packages stage a copy under their own `native/` folders when the build script runs.

If auto-discovery fails later, set:

```bash
export EXIMIETAS_LOGGER_LIB=/absolute/path/to/liblogger.so   # or logger.dll / liblogger.dylib
```

## 2. Pick a language and run an example

### Python

```bash
pip install -e bindings/python
python examples/python/main.py
```

### Node.js

```bash
cd bindings/node && npm install && npm run build && cd ../..
# Windows
EXIMIETAS_LOGGER_LIB="$PWD/dist/logger.dll" node examples/node/main.mjs
# Linux
EXIMIETAS_LOGGER_LIB="$PWD/dist/liblogger.so" node examples/node/main.mjs
```

### .NET

```bash
export EXIMIETAS_LOGGER_LIB="$PWD/dist/logger.dll"   # or liblogger.so / .dylib
dotnet run --project examples/dotnet
```

### Go (no shared library required)

```bash
go run ./cmd/logger-demo
```

### C

```bash
# Linux
gcc examples/c/main.c -I native/include -L dist -llogger -o examples/c/demo
LD_LIBRARY_PATH=dist ./examples/c/demo
```

## 3. What a log line looks like

```json
{
  "timestamp": "2026-07-29T07:00:00.123456789Z",
  "level": "info",
  "message": "order created",
  "service_name": "payments-api",
  "environment": "prod",
  "fields": { "order_id": 123, "amount": 42.5 }
}
```

## Next steps

- [User guide](user-guide.md) — levels, context fields, flush/close, stats
- [Configuration](configuration.md) — full schema
- [Sinks](sinks.md) — file rotation and shipping logs to a collector / Loki
- [Monorepo](monorepo.md) — use this package inside a Turborepo workspace
