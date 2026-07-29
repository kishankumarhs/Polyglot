# Polyglot Logger (.NET)

Idiomatic .NET bindings (`Polyglot.Logger`) for the Polyglot native structured logger.

**Full guide:** [docs/languages/dotnet.md](../../docs/languages/dotnet.md) · [User guide](../../docs/user-guide.md) · [Getting started](../../docs/getting-started.md)

```csharp
using Polyglot.Logger;

using var log = new Logger(new LoggerOptions
{
    Service = "billing-api",
    Environment = "prod",
    FilePath = "app.log",
    Stdout = false,
});

log.SetFields(new Dictionary<string, object?> { ["traceId"] = "abc" });
log.Info("invoice issued", new Dictionary<string, object?> { ["invoice_id"] = 99 });
log.LogSimple(Level.Error, "billing failed");
```

```bash
make build-native
dotnet build bindings/dotnet/Polyglot.Logger/Polyglot.Logger.csproj
```

**Thread safety:** concurrent `Info` / `Flush` / `Stats` / `SetFields` / `ReloadConfig` on one instance is safe; dispose once from a single owner.

Set `POLYGLOT_LOGGER_LIB` to the absolute path of `logger.dll`, `liblogger.so`, or `liblogger.dylib` if needed.
