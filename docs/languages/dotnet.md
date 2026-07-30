# .NET

Package: [`Polyglot.Logger`](https://www.nuget.org/packages/Polyglot.Logger/) · repo: [polyglot-csharp](https://github.com/kishankumarhs/polyglot-csharp)

```bash
dotnet add package Polyglot.Logger
```

From this monorepo:

```bash
make build-native
dotnet add reference bindings/dotnet/Polyglot.Logger/Polyglot.Logger.csproj
```

## Example

```csharp
using Polyglot.Logger;

Console.WriteLine($"version={Logger.LibraryVersion()} abi={Logger.AbiVersion()}");

using var log = new Logger(new LoggerOptions
{
    Service = "billing-api",
    Environment = "prod",
    Level = "info",
    Stdout = false,
    File = new FileOptions { Path = "app.log", MaxSizeMb = 100 },
    Async = true,
    Overflow = "drop_newest",
});

log.SetFields(new Dictionary<string, object?> { ["traceId"] = "abc" });
log.Info("invoice issued", new Dictionary<string, object?> { ["invoice_id"] = 99 });
log.Error("billing failed", new Dictionary<string, object?> { ["code"] = "TIMEOUT" });
log.Flush();
```

Prefer `using` so dispose always runs.

## Config

```yaml
# polyglot.yaml
service: billing-api
environment: prod
level: info
async: true
file:
  enabled: true
  path: app.log
```

Env: `POLYGLOT_CONFIG` (aliases `POLYGLOT_CONFIG_PATH` / `POLYGLOT_CONFIG_FILE`). Discovery walks cwd → parents and stops at `.git`. See [sdk.md](../sdk.md) · [zero-config.md](../zero-config.md).

HTTP-only:

```csharp
using var log = new Logger(new LoggerOptions
{
    Service = "billing-api",
    Stdout = false,
    Http = new HttpOptions
    {
        Url = "https://collector.example/v1/logs",
        Headers = new Dictionary<string, string>
        {
            ["Authorization"] = $"Bearer {Environment.GetEnvironmentVariable("LOG_TOKEN")}"
        },
        BatchSize = 50,
        FlushIntervalMs = 1000,
    },
});
```

## Options

| Property | Notes |
| --- | --- |
| `Service` | Required for useful logs |
| `ServiceVersion`, `Environment`, `Level` | Metadata / min level |
| `Stdout` | Default `true` |
| `File` / `FilePath` | Rotating file |
| `Http` | Remote NDJSON |
| `Async`, `QueueSize`, `Overflow` | Queue |
| `Fields` | Base fields |

## Methods

| Method | Notes |
| --- | --- |
| `Trace` … `Fatal` | `Fatal` does not call `Environment.Exit` |
| `Log` / `LogSimple` | Explicit level |
| `With` | Child logger |
| `SetFields` / `ReloadConfig` | Context / hot reload |
| `Stats` / `Flush` / `Dispose` | Observability / lifecycle |

Failures throw `LoggerException`. Safe for concurrent Tasks; dispose once. On Windows, dispose before reading the active log file from the same process.

```bash
export POLYGLOT_LOGGER_LIB=$PWD/dist/logger.dll
dotnet test bindings/dotnet/Polyglot.Logger.Tests/Polyglot.Logger.Tests.csproj -c Release
dotnet run --project examples/dotnet
```

See [user-guide.md](../user-guide.md) · [configuration.md](../configuration.md).
