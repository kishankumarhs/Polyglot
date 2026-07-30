# Getting started

App developers: install a registry package and skip to [first-log.md](first-log.md).

This page is for people building from source.

## Prerequisites

- Go 1.22+
- C toolchain (gcc / MinGW / MSVC) with CGO enabled
- Optional: Python 3.9+, Node 18+, .NET 8+ depending on which binding you touch

## Build

```bash
git clone --recurse-submodules https://github.com/kishankumarhs/Polyglot.git
cd Polyglot
bash scripts/check-submodules.sh
make build-native
```

Output lands in `dist/`:

| Platform | File |
| --- | --- |
| Linux | `liblogger.so` |
| Windows | `logger.dll` |
| macOS | `liblogger.dylib` |

Also: `dist/logger.h`, `dist/checksums.sha256`.

If a binding can't find the lib:

```bash
export POLYGLOT_LOGGER_LIB=/absolute/path/to/liblogger.so   # or logger.dll / liblogger.dylib
```

## Run an example

```bash
# Python
pip install -e bindings/python
python examples/python/main.py

# Node
cd bindings/node && npm install && npm run build && cd ../..
POLYGLOT_LOGGER_LIB="$PWD/dist/logger.dll" node examples/node/main.mjs

# .NET
export POLYGLOT_LOGGER_LIB="$PWD/dist/logger.dll"
dotnet run --project examples/dotnet

# Go (no shared library)
go run ./cmd/logger-demo
```

## What a line looks like

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

## Next

- [User guide](user-guide.md)
- [Configuration](configuration.md)
- [Sinks](sinks.md)
- [Monorepo](monorepo.md)
