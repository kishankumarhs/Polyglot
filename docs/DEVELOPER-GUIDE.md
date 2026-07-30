# Developer guide

Core + bindings via Git submodules. ABI in `api/abi.json` drives codegen — don't hand-edit `*.generated.*` or `logger.h`.

```text
Polyglot (core)
├── Go logger, api/abi.json, cmd/codegen
└── bindings/ (submodules → node / python / dotnet)
```

## Setup

```bash
git clone --recurse-submodules https://github.com/kishankumarhs/Polyglot.git
cd Polyglot
bash scripts/check-submodules.sh
make build-native
go test ./internal/logger ./native -count=1
```

## What you can edit

| Path | Edit? |
| --- | --- |
| `internal/logger/*.go`, `native/export.go`, `api/abi.json` | yes |
| Binding SDK (`index.ts`, `__init__.py`, `Logger.cs`, …) | yes |
| `logger.h`, `*.generated.*`, `NativeMethods.Generated.cs` | no — run `make codegen` |

## ABI change

1. Edit `api/abi.json` and `native/export.go`
2. `make codegen`
3. Commit generated files
4. Update binding wrappers if you added a new ergonomic API

## Binding change

Commit and push inside `bindings/<lang>`, then bump the submodule pointer in the parent repo. See [SUBMODULE-WORKFLOW.md](SUBMODULE-WORKFLOW.md).

## Also useful

- [getting-started.md](getting-started.md)
- [architecture.md](architecture.md)
- [abi.md](abi.md)
- [build.md](build.md)
- [troubleshooting.md](troubleshooting.md)
- [CONTRIBUTING.md](../CONTRIBUTING.md)
