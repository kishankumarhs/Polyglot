# Monorepo integration

Drop this repo under `packages/logger` (submodule, subtree, or copy) and point services at the bindings + a built native lib.

```text
your-monorepo/
├── apps/
│   ├── api-node/
│   ├── worker-python/
│   └── billing-dotnet/
├── packages/logger/          ← this repo
│   ├── bindings/…
│   └── dist/
├── package.json
└── turbo.json
```

## Node

```json
{
  "workspaces": ["apps/*", "packages/logger/bindings/node"]
}
```

```bash
export POLYGLOT_LOGGER_LIB=/abs/path/to/packages/logger/dist/liblogger.so
make -C packages/logger build-native
```

## Python

```bash
pip install -e packages/logger/bindings/python
export POLYGLOT_LOGGER_LIB=/abs/path/to/packages/logger/dist/liblogger.so
```

## .NET

```xml
<ProjectReference Include="..\..\packages\logger\bindings\dotnet\Polyglot.Logger\Polyglot.Logger.csproj" />
```

Or pack an internal NuGet that includes the native lib.

## Shared config

Same JSON/YAML schema everywhere ([configuration.md](configuration.md)). Keep a shared `polyglot.yaml` or a small helper per language that maps env → options.

Only logger maintainers should run `make codegen` or edit `api/abi.json`.
