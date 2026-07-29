# .NET Guide

**Package:** `Polyglot.Logger` ([NuGet](https://www.nuget.org/packages/Polyglot.Logger/))  
**Repository:** [polyglot-csharp](https://github.com/kishankumarhs/polyglot-csharp) (Git submodule)  
**Status:** Independent package with bundled native binaries

## Overview

This binding provides idiomatic .NET access to the Polyglot native logger. It's part of a [modular monorepo](../architecture.md) where:

- **Core:** Go logger in [polyglot-go](https://github.com/kishankumarhs/Polyglot)
- **This binding:** Independent .NET repository with its own releases
- **Generated code:** P/Invoke bindings auto-generated from C ABI contract
- **Native binaries:** Pre-compiled for Windows/macOS/Linux, bundled in NuGet package

See [REPOSITORIES.md](../REPOSITORIES.md) for how all four repositories work together.

## Installation

### From NuGet (Recommended)

```bash
dotnet add package Polyglot.Logger
```

Pre-compiled native binaries are included. No build needed.

### From Source (Core Development)

If working on the core Polyglot repository:

```bash
git clone --recurse-submodules https://github.com/kishankumarhs/Polyglot.git
cd Polyglot

make build-native
dotnet add reference bindings/dotnet/Polyglot.Logger/Polyglot.Logger.csproj
```

## Quick Start

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

## Configuration

### Via polyglot.yaml

Create `polyglot.yaml` in project root (auto-discovered):

```yaml
service: billing-api
environment: prod
logging:
  level: info
  async: true
file:
  enabled: true
  path: app.log
  max_size_mb: 100
http:
  enabled: false
  endpoint: https://logs.company.com/ingest
```

### Programmatic Configuration

```csharp
var options = new LoggerOptions
{
    Service = "billing-api",
    Environment = "prod",
    Level = "info",
    Async = true,
    File = new FileOptions
    {
        Path = "app.log",
        MaxSizeMb = 100
    }
};

using var log = new Logger(options);
```

### Environment Variables

```csharp
Environment.SetEnvironmentVariable("POLYGLOT_CONFIG_PATH", "/etc/myapp/polyglot.yaml");
Environment.SetEnvironmentVariable("POLYGLOT_CONFIG_FILE", "/etc/myapp/config.json");
```

## API Reference

### Logger Class

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
export POLYGLOT_LOGGER_LIB=$PWD/dist/liblogger.so
dotnet test bindings/dotnet/Polyglot.Logger.Tests/Polyglot.Logger.Tests.csproj -c Release
dotnet run --project examples/dotnet
```

See also: [User guide](../user-guide.md), [Configuration](../configuration.md).
