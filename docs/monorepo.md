# Monorepo integration (Turborepo / npm workspaces)

This repository is designed to be consumed as a **package** inside a larger monorepo that already has Python, Node, and .NET services.

## Recommended layout

```text
your-monorepo/
├── apps/
│   ├── api-node/
│   ├── worker-python/
│   └── billing-dotnet/
├── packages/
│   └── logger/                 ← this repo (git submodule, subtree, or copied package)
│       ├── bindings/node/
│       ├── bindings/python/
│       ├── bindings/dotnet/
│       ├── dist/               ← built native libs (or CI artifact)
│       └── ...
├── package.json                ← npm workspaces
└── turbo.json
```

## Node / TypeScript (npm workspaces)

1. Add the binding as a workspace package, for example:

```json
{
  "name": "your-monorepo",
  "private": true,
  "workspaces": [
    "apps/*",
    "packages/logger/bindings/node"
  ]
}
```

2. Depend on it from an app:

```json
{
  "dependencies": {
    "@eximietas/logger": "*"
  }
}
```

3. Build native once (CI or local), then point services at it:

```bash
export EXIMIETAS_LOGGER_LIB=/abs/path/to/packages/logger/dist/liblogger.so
```

4. Optional Turbo pipeline:

```json
{
  "tasks": {
    "build": {
      "dependsOn": ["^build"],
      "outputs": ["dist/**"]
    },
    "logger:native": {
      "cache": false,
      "outputs": ["dist/**"]
    }
  }
}
```

Run `make build-native` (or a Turbo task that shells out to it) before Node tests that actually create a logger.

## Python

Install editable from the monorepo path:

```bash
pip install -e packages/logger/bindings/python
```

Or publish an internal wheel that vendors `native/liblogger.so` (and siblings) under `eximietas_logger/native/`.

Set `EXIMIETAS_LOGGER_LIB` in service env if the packaged `native/` copy is not present.

## .NET

Reference the project:

```xml
<ProjectReference Include="..\..\packages\logger\bindings\dotnet\Eximietas.Logger\Eximietas.Logger.csproj" />
```

Or pack an internal NuGet that includes the platform-native library next to the managed DLL. At runtime, `EXIMIETAS_LOGGER_LIB` or the resolver’s `native/` candidate paths must find the shared library.

## Shared native artifact strategy

| Strategy | Pros | Cons |
| -------- | ---- | ---- |
| CI builds per OS, publish artifact | Clear, reproducible | Need matrix for linux/win/mac |
| Commit prebuilt libs under `dist/` | Simple local bootstrap | Large binaries in git |
| Build on each developer machine | Always matches local OS | Requires Go + C toolchain |

Prefer CI artifacts + `EXIMIETAS_LOGGER_LIB` in deployed services.

## One config contract everywhere

All languages use the same JSON config (see [configuration](configuration.md)). In a monorepo you can share a snippet:

```text
packages/logger/config/prod.http-only.json
```

and load it from each service, or map env vars into binding options in a small helper per language.

## Codegen ownership

Only the logger package maintainers should run `make codegen` / edit `api/abi.json`. App teams consume published bindings and never hand-edit `*.generated.*` files.
