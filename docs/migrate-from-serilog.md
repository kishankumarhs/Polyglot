# Migrating from Serilog

Message templates become a plain message string plus a field dictionary. Enrichment maps to `With` / `RequestLogging.ForRequest`.

```csharp
// Before
Log.Logger = new LoggerConfiguration().WriteTo.Console().CreateLogger();
Log.Information("Hello {UserId}", 1);

// After
using Polyglot.Logger;

using var log = new Logger(new LoggerOptions {
    Service = "api",
    Stdout = true,
    Level = "info",
});
log.Info("Hello", new Dictionary<string, object?> { ["UserId"] = 1 });
```

Request scope:

```csharp
using var reqLog = log.With(new Dictionary<string, object?> { ["requestId"] = id });
reqLog.Info("handled");
```

Dispose the root logger to shut down sinks; disposing a child only frees the child handle. Native lib loads from the package `native/` / `bin/` or `POLYGLOT_LOGGER_LIB`.

More: [first-log.md](first-log.md) · [languages/dotnet.md](languages/dotnet.md)
