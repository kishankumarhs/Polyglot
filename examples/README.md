# Examples

Minimal runnable samples. Build the native library first (`make build-native` from the repo root), then:

| Example | Command |
| ------- | ------- |
| [python/main.py](python/main.py) | `pip install -e ../bindings/python && python examples/python/main.py` |
| [node/main.mjs](node/main.mjs) | Build `bindings/node`, set `POLYGLOT_LOGGER_LIB`, run `node examples/node/main.mjs` |
| [dotnet/](dotnet/) | `dotnet run --project examples/dotnet` with `POLYGLOT_LOGGER_LIB` set |
| [c/main.c](c/main.c) | See comments at top of the file for gcc link lines |

Full documentation: [docs/](../docs/README.md).
