// Polyglot vs Serilog — sync file, rich payload, ops/s + p99.
// Run: dotnet run -c Release --project bench/dotnet
using System.Diagnostics;
using Polyglot.Logger;
using Serilog;
using Serilog.Formatting.Compact;

var n = int.TryParse(Environment.GetEnvironmentVariable("BENCH_DOTNET_N"), out var parsedN) ? parsedN : 20_000;
var iters = int.TryParse(Environment.GetEnvironmentVariable("BENCH_DOTNET_ITERS"), out var parsedI) ? parsedI : 3;

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

if (string.IsNullOrEmpty(Environment.GetEnvironmentVariable("POLYGLOT_LOGGER_LIB")))
{
    Console.Error.WriteLine("polyglot dotnet bench skipped: no native lib (set POLYGLOT_LOGGER_LIB)");
    return;
}

static Dictionary<string, object?> Rich(int i) => new()
{
    ["user_id"] = 7,
    ["trace_id"] = "abc123def456ghi789",
    ["span_id"] = "span-001",
    ["service"] = "payments",
    ["region"] = "us-east-1",
    ["latency_ms"] = 12.4,
    ["ok"] = true,
    ["tags"] = new[] { "a", "b", "c" },
    ["meta"] = new Dictionary<string, object?>
    {
        ["cart"] = new Dictionary<string, object?> { ["items"] = 3, ["currency"] = "USD" },
    },
    ["error"] = "optional message",
    ["request_id"] = "req-xyz",
    ["tenant"] = "acme",
    ["env"] = "prod",
    ["version"] = "1.2.3",
    ["attempt"] = 1,
    ["bytes"] = 4096,
    ["cached"] = false,
    ["n"] = i,
};

static (double Ops, double Mean, double P50, double P95, double P99) MedianRun(int n, int iters, Action<List<long>> body)
{
    var runs = new List<(double Ops, double Mean, double P50, double P95, double P99)>();
    for (var iter = 0; iter < iters; iter++)
    {
        var samples = new List<long>(n);
        var sw = Stopwatch.StartNew();
        body(samples);
        sw.Stop();
        samples.Sort();
        long At(double p) => samples[Math.Min(samples.Count - 1, (int)(p / 100.0 * samples.Count))];
        var mean = samples.Average();
        runs.Add((n / sw.Elapsed.TotalSeconds, mean, At(50), At(95), At(99)));
    }
    runs.Sort((a, b) => a.Ops.CompareTo(b.Ops));
    return runs[runs.Count / 2];
}

void PolyglotSync(List<long> samples)
{
    var path = Path.Combine(Path.GetTempPath(), $"pg-dotnet-{Environment.ProcessId}.log");
    if (File.Exists(path)) File.Delete(path);
    using var log = new Logger(new LoggerOptions
    {
        Service = "bench",
        Level = "info",
        Stdout = false,
        Async = false,
        FilePath = path,
    });
    for (var i = 0; i < n; i++)
    {
        var t0 = Stopwatch.GetTimestamp();
        log.Info("checkout", Rich(i));
        samples.Add((long)((Stopwatch.GetTimestamp() - t0) * (1e9 / Stopwatch.Frequency)));
    }
    log.Flush();
    try { File.Delete(path); } catch { /* ignore */ }
}

void SerilogSync(List<long> samples)
{
    var path = Path.Combine(Path.GetTempPath(), $"serilog-{Environment.ProcessId}.log");
    if (File.Exists(path)) File.Delete(path);
    using var log = new LoggerConfiguration()
        .MinimumLevel.Information()
        .WriteTo.File(new CompactJsonFormatter(), path, buffered: false)
        .CreateLogger();
    for (var i = 0; i < n; i++)
    {
        var fields = Rich(i);
        var t0 = Stopwatch.GetTimestamp();
        log.Information(
            "checkout {user_id} {trace_id} {span_id} {service} {region} {latency_ms} {ok} {tags} {meta} {error} {request_id} {tenant} {env} {version} {attempt} {bytes} {cached} {n}",
            fields["user_id"], fields["trace_id"], fields["span_id"], fields["service"], fields["region"],
            fields["latency_ms"], fields["ok"], fields["tags"], fields["meta"], fields["error"],
            fields["request_id"], fields["tenant"], fields["env"], fields["version"], fields["attempt"],
            fields["bytes"], fields["cached"], fields["n"]);
        samples.Add((long)((Stopwatch.GetTimestamp() - t0) * (1e9 / Stopwatch.Frequency)));
    }
    log.Dispose();
    try { File.Delete(path); } catch { /* ignore */ }
}

foreach (var (name, body) in new (string, Action<List<long>>)[]
{
    ("polyglot_sync_file", PolyglotSync),
    ("serilog_sync_file", SerilogSync),
})
{
    var mid = MedianRun(n, iters, body);
    Console.WriteLine(
        $"{name} runtime=dotnet ops/s={mid.Ops:F0} mean={mid.Mean:F0} p50={mid.P50:F0} p95={mid.P95:F0} p99={mid.P99:F0} n={n}");
}
