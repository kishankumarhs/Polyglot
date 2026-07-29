using Eximietas.Logger;

Console.WriteLine($"version={Logger.LibraryVersion()} abi={Logger.AbiVersion()}");

using var log = new Logger(new LoggerOptions
{
    Service = "dotnet-example",
    Environment = "dev",
    Level = "debug",
    Stdout = true,
    Async = true,
    Overflow = "drop_newest",
});

log.SetFields(new Dictionary<string, object?> { ["traceId"] = "demo-trace" });
log.Info("hello from dotnet", new Dictionary<string, object?> { ["userId"] = 1 });
log.LogSimple(Level.Warn, "simple warning");
log.Flush();
Console.WriteLine($"stats={string.Join(",", log.Stats().Select(kv => $"{kv.Key}={kv.Value}"))}");
