// Cross-language consistency: 100k rich logs via .NET binding.
// Run: dotnet run -c Release --project bench/cross/dotnet
using System.Diagnostics;
using System.Text.Json;
using Polyglot.Logger;

var n = int.TryParse(Environment.GetEnvironmentVariable("BENCH_CROSS_N"), out var parsed) ? parsed : 100_000;
var root = Path.GetFullPath(Path.Combine(AppContext.BaseDirectory, "..", "..", "..", "..", ".."));
var libName = OperatingSystem.IsWindows() ? "logger.dll" : OperatingSystem.IsMacOS() ? "liblogger.dylib" : "liblogger.so";
foreach (var candidate in new[]
{
    Path.Combine(root, "dist", libName),
    Path.Combine(Directory.GetCurrentDirectory(), "dist", libName),
})
{
    if (File.Exists(candidate))
    {
        Environment.SetEnvironmentVariable("POLYGLOT_LOGGER_LIB", candidate);
        break;
    }
}

var outPath = Path.Combine(Path.GetTempPath(), "polyglot-cross-dotnet.log");
if (File.Exists(outPath)) File.Delete(outPath);

TimeSpan elapsed;
{
    using var log = new Logger(new LoggerOptions
    {
        Service = "bench-cross",
        Level = "info",
        Stdout = false,
        Async = false,
        FilePath = outPath,
    });

    var fields = new Dictionary<string, object?>
    {
        ["user_id"] = 7,
        ["trace_id"] = "abc123",
        ["span_id"] = "span-1",
        ["service"] = "payments",
        ["region"] = "us-east-1",
        ["latency_ms"] = 12.4,
        ["ok"] = true,
        ["tags"] = new[] { "a", "b", "c" },
        ["meta"] = new Dictionary<string, object?> { ["cart"] = new Dictionary<string, object?> { ["items"] = 3, ["currency"] = "USD" } },
        ["error"] = "optional message",
    };

    var sw = Stopwatch.StartNew();
    for (var i = 0; i < n; i++)
    {
        fields["n"] = i;
        log.Info("checkout", fields);
    }
    log.Flush();
    sw.Stop();
    elapsed = sw.Elapsed;
}

var first = JsonDocument.Parse(File.ReadLines(outPath).First());
var keys = string.Join(",", first.RootElement.EnumerateObject().Select(p => p.Name));
Console.WriteLine($"lang=dotnet n={n} elapsed={elapsed.TotalSeconds:F3}s ops_s={n / elapsed.TotalSeconds:F0} path={outPath} schema_keys={keys}");
