# Examples

## Using published packages

```bash
npm install @polyglot-logger/node
pip install polyglot-logger
dotnet add package Polyglot.Logger
```

Walkthrough: [docs/first-log.md](../docs/first-log.md).

## From this repo

Build once, then point bindings at the local lib:

```bash
make build-native
export POLYGLOT_LOGGER_LIB=$PWD/dist/logger.dll   # or liblogger.so / .dylib
```

| Example | Run |
| --- | --- |
| [python/main.py](python/main.py) | `pip install -e bindings/python && python examples/python/main.py` |
| [node/main.mjs](node/main.mjs) | Build `bindings/node`, then `node examples/node/main.mjs` |
| [dotnet/](dotnet/) | `dotnet run --project examples/dotnet` |
| [c/main.c](c/main.c) | See comments at the top of the file for gcc lines |

Docs: [docs/](../docs/README.md) · Benches: [bench/](../bench/README.md)
