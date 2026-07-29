using System.Text.Json;
using Xunit;

namespace Eximietas.Logger.Tests;

public class SmokeTests
{
    [Fact]
    public void WritesStructuredJsonAndFiltersLevels()
    {
        var path = Path.Combine(Path.GetTempPath(), $"eximietas-logger-{Guid.NewGuid():N}.log");
        try
        {
            using (var log = new Logger(new LoggerOptions
            {
                Service = "dotnet-smoke",
                Environment = "test",
                Level = "info",
                Stdout = false,
                FilePath = path,
                Async = false,
                Fields = new Dictionary<string, object?> { ["team"] = "platform" },
            }))
            {
                log.Debug("hidden", new Dictionary<string, object?> { ["n"] = 1 });
                log.Info("hello", new Dictionary<string, object?> { ["user_id"] = 7 });
                log.SetFields(new Dictionary<string, object?> { ["traceId"] = "t-1" });
                log.LogSimple(Level.Info, "simple");
                log.Flush();
                var stats = log.Stats();
                Assert.True(stats["flushed"].GetInt64() >= 2);
            }

            var lines = File.ReadAllLines(path);
            Assert.Equal(2, lines.Length);

            using var doc = JsonDocument.Parse(lines[0]);
            var root = doc.RootElement;
            Assert.Equal("info", root.GetProperty("level").GetString());
            Assert.Equal("hello", root.GetProperty("message").GetString());
            Assert.Equal("dotnet-smoke", root.GetProperty("service_name").GetString());
            Assert.Equal("platform", root.GetProperty("fields").GetProperty("team").GetString());
            Assert.Equal(7, root.GetProperty("fields").GetProperty("user_id").GetInt32());
            Assert.Equal(1, Logger.AbiVersion());
            Assert.False(string.IsNullOrWhiteSpace(Logger.LibraryVersion()));
        }
        finally
        {
            if (File.Exists(path))
            {
                File.Delete(path);
            }
        }
    }

    [Fact]
    public void InvalidConfigThrows()
    {
        Assert.Throws<LoggerException>(() =>
        {
            _ = new Logger(new LoggerOptions
            {
                Service = "",
                Stdout = true,
            });
        });
    }
}
