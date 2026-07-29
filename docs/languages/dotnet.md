# .NET guide

Package: `Eximietas.Logger` (`bindings/dotnet/Eximietas.Logger`).

## Install / reference

```bash
dotnet add reference bindings/dotnet/Eximietas.Logger/Eximietas.Logger.csproj
```

Build the native library first (`make build-native`) and set `EXIMIETAS_LOGGER_LIB` when the DLL/`so` is not beside the managed assembly.

## Quick start

```csharp
using Eximietas.Logger;

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
log.Warn("slow query", new Dictionary<string, object?> { ["ms"] = 400 });
log.Error("billing failed", new Dictionary<string, object?> { ["code"] = "TIMEOUT" });
log.LogSimple(Level.Error, "fallback");
log.Flush();
foreach (var kv in log.Stats())
{
    Console.WriteLine($"{kv.Key}={kv.Value}");
}
```

`Logger` implements `IDisposable`: prefer `using` so `close` always runs.

## Options

| Property | Notes |
| -------- | ----- |
| `Service` | Required |
| `ServiceVersion`, `Environment`, `Level` | Metadata / min level |
| `Stdout` | Default `true` |
| `File` | `FileOptions` with `Path`, sizes, backups |
| `FilePath` | Convenience when you only need a path |
| `Http` | `HttpOptions` with `Url`, `Headers`, batch settings |
| `Async`, `QueueSize`, `Overflow` | Queue behavior |
| `Fields` | Base fields |

### HTTP-only example

```csharp
using var log = new Logger(new LoggerOptions
{
    Service = "billing-api",
    Environment = "prod",
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

## API

| Method | Description |
| ------ | ----------- |
| `Trace` / `Debug` / `Info` / `Warn` / `Error` / `Fatal` | Structured log |
| `Log(Level, message, fields?)` | Explicit level |
| `LogSimple(Level, message)` | No fields |
| `SetFields` / `ReloadConfig` | Context / hot reload |
| `Stats` / `Flush` / `Dispose` | Observability / lifecycle |

`Fatal` does **not** call `Environment.Exit`. Failures throw `LoggerException`.

## Thread safety

Safe for concurrent Tasks. Dispose once from a single owner. Close/`Dispose` before reading the active log file from the same process on Windows.

## Tests & example

```bash
export EXIMIETAS_LOGGER_LIB=$PWD/dist/liblogger.so
dotnet test bindings/dotnet/Eximietas.Logger.Tests/Eximietas.Logger.Tests.csproj -c Release
dotnet run --project examples/dotnet
```

See also: [User guide](../user-guide.md), [Configuration](../configuration.md).
