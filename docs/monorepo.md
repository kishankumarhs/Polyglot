# Monorepo integration

## Shared config (recommended)

Keep one `polyglot.yaml` at the **git repo root**. Each app runs with cwd under `apps/…`; discovery walks up to the root (and stops at `.git`).

```text
your-monorepo/
├── apps/
│   ├── api-node/          ← cwd when running; finds ../../polyglot.yaml
│   ├── worker-python/
│   └── billing-dotnet/
├── packages/logger/       ← this repo (optional submodule)
├── polyglot.yaml          ← shared sinks (http aggregator, batching, …)
└── .git/
```

Per-service identity belongs in the constructor (or a service-specific YAML via `POLYGLOT_CONFIG`):

```js
new Logger({ service: "api-node" }); // YAML supplies http sink, etc.
```

```python
Logger(service="worker-python")
```

```csharp
new Logger(new LoggerOptions { Service = "billing-dotnet" });
```

Override the shared file when needed:

```bash
export POLYGLOT_CONFIG=/abs/path/to/apps/api-node/polyglot.yaml
```

## Consuming from source

Drop this repo under `packages/logger` (submodule, subtree, or copy) and point services at the bindings + a built native lib.

### Node

```json
{
  "workspaces": ["apps/*", "packages/logger/bindings/node"]
}
```

```bash
export POLYGLOT_LOGGER_LIB=/abs/path/to/packages/logger/dist/liblogger.so
make -C packages/logger build-native
```

### Python

```bash
pip install -e packages/logger/bindings/python
export POLYGLOT_LOGGER_LIB=/abs/path/to/packages/logger/dist/liblogger.so
```

### .NET

```xml
<ProjectReference Include="..\..\packages\logger\bindings\dotnet\Polyglot.Logger\Polyglot.Logger.csproj" />
```

Or pack an internal NuGet that includes the native lib.

## Verify

```bash
cd apps/api-node
go run ../../packages/logger/cmd/polyglot doctor
# or from the logger repo: make doctor
```

You should see the root `polyglot.yaml` path and sink summary.

Same JSON/YAML schema everywhere ([configuration.md](configuration.md)). API cheat sheet: [sdk.md](sdk.md).

Only logger maintainers should run `make codegen` or edit `api/abi.json`.
