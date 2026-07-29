using System.Reflection;
using System.Runtime.InteropServices;
using System.Text.Json;

namespace Polyglot.Logger;

public sealed class LoggerException : Exception
{
    public LoggerException(string message) : base(message) { }
}

internal sealed class LoggerHandle : SafeHandle
{
    public LoggerHandle() : base(IntPtr.Zero, ownsHandle: true) { }

    public LoggerHandle(IntPtr handle) : base(IntPtr.Zero, ownsHandle: true)
    {
        SetHandle(handle);
    }

    public override bool IsInvalid => handle == IntPtr.Zero;

    public IntPtr Value => handle;

    protected override bool ReleaseHandle()
    {
        if (IsInvalid)
        {
            return true;
        }
        return NativeMethods.logger_close(handle) == 0;
    }
}

internal static partial class NativeMethods
{
    static NativeMethods()
    {
        NativeLibrary.SetDllImportResolver(typeof(NativeMethods).Assembly, Resolve);
    }

    private static IntPtr Resolve(string libraryName, Assembly assembly, DllImportSearchPath? searchPath)
    {
        if (!string.Equals(libraryName, LibraryName, StringComparison.OrdinalIgnoreCase))
        {
            return IntPtr.Zero;
        }

        foreach (var candidate in CandidatePaths())
        {
            if (File.Exists(candidate) && NativeLibrary.TryLoad(candidate, out var handle))
            {
                return handle;
            }
        }

        return IntPtr.Zero;
    }

    private static IEnumerable<string> CandidatePaths()
    {
        var name = RuntimeInformation.IsOSPlatform(OSPlatform.Windows)
            ? "logger.dll"
            : RuntimeInformation.IsOSPlatform(OSPlatform.OSX)
                ? "liblogger.dylib"
                : "liblogger.so";

        var env = Environment.GetEnvironmentVariable("POLYGLOT_LOGGER_LIB");
        if (!string.IsNullOrWhiteSpace(env))
        {
            yield return env;
        }

        var baseDir = AppContext.BaseDirectory;
        yield return Path.Combine(baseDir, name);
        yield return Path.Combine(baseDir, "native", name);
        yield return Path.Combine(baseDir, "..", "..", "..", "..", "..", "dist", name);
        yield return Path.Combine(Directory.GetCurrentDirectory(), name);
        yield return Path.Combine(Directory.GetCurrentDirectory(), "dist", name);
        yield return Path.Combine(Directory.GetCurrentDirectory(), "build", name);
    }

    internal static string PtrToUtf8(IntPtr ptr)
    {
        if (ptr == IntPtr.Zero)
        {
            return string.Empty;
        }
        return Marshal.PtrToStringUTF8(ptr) ?? string.Empty;
    }

    internal static string LastError(IntPtr handle) => PtrToUtf8(logger_last_error(handle));
}

public sealed class FileOptions
{
    public bool Enabled { get; init; } = true;
    public required string Path { get; init; }
    public int MaxSizeMb { get; init; } = 100;
    public int MaxBackups { get; init; } = 10;
    public int MaxAgeDays { get; init; } = 30;
    public bool Compress { get; init; }
}

public sealed class HttpOptions
{
    public bool Enabled { get; init; } = true;
    public required string Url { get; init; }
    public int TimeoutMs { get; init; } = 5000;
    public Dictionary<string, string> Headers { get; init; } = new();
    public int BatchSize { get; init; } = 50;
    public int FlushIntervalMs { get; init; } = 1000;
}

public sealed class LoggerOptions
{
    public required string Service { get; init; }
    public string ServiceVersion { get; init; } = "";
    public string Environment { get; init; } = "";
    public string Level { get; init; } = "info";
    public bool Stdout { get; init; } = true;
    public FileOptions? File { get; init; }
    public string? FilePath { get; init; }
    public int MaxSizeMb { get; init; } = 100;
    public int MaxBackups { get; init; } = 10;
    public int MaxAgeDays { get; init; } = 30;
    public HttpOptions? Http { get; init; }
    public bool Async { get; init; } = true;
    public int QueueSize { get; init; } = 10000;
    public string Overflow { get; init; } = "drop_newest";
    public Dictionary<string, object?> Fields { get; init; } = new();

    // Legacy aliases
    public string? ServiceName { get; init; }
    public string? MinLevel { get; init; }
}

/// <summary>
/// Thread-safety: Log/Flush/Stats/SetFields/ReloadConfig are safe for concurrent use
/// on one instance. Prefer a single owner for Dispose/Close.
/// </summary>
public sealed class Logger : IDisposable
{
    private readonly LoggerHandle _handle;
    private bool _disposed;

    public Logger(LoggerOptions options)
    {
        var service = options.ServiceName ?? options.Service;
        var config = new Dictionary<string, object?>
        {
            ["service"] = service,
            ["service_version"] = options.ServiceVersion,
            ["environment"] = options.Environment,
            ["level"] = options.MinLevel ?? options.Level,
            ["stdout"] = options.Stdout,
            ["async"] = options.Async,
            ["queueSize"] = options.QueueSize,
            ["overflow"] = options.Overflow,
            ["fields"] = options.Fields,
        };

        if (options.File is not null)
        {
            config["file"] = new Dictionary<string, object?>
            {
                ["enabled"] = options.File.Enabled,
                ["path"] = options.File.Path,
                ["maxSizeMB"] = options.File.MaxSizeMb,
                ["maxBackups"] = options.File.MaxBackups,
                ["maxAgeDays"] = options.File.MaxAgeDays,
                ["compress"] = options.File.Compress,
            };
        }
        else if (!string.IsNullOrWhiteSpace(options.FilePath))
        {
            config["file"] = new Dictionary<string, object?>
            {
                ["enabled"] = true,
                ["path"] = options.FilePath,
                ["maxSizeMB"] = options.MaxSizeMb,
                ["maxBackups"] = options.MaxBackups,
                ["maxAgeDays"] = options.MaxAgeDays,
                ["compress"] = false,
            };
        }

        if (options.Http is not null)
        {
            config["http"] = new Dictionary<string, object?>
            {
                ["enabled"] = options.Http.Enabled,
                ["url"] = options.Http.Url,
                ["timeout_ms"] = options.Http.TimeoutMs,
                ["headers"] = options.Http.Headers,
                ["batch_size"] = options.Http.BatchSize,
                ["flush_interval_ms"] = options.Http.FlushIntervalMs,
            };
        }

        var handle = NativeMethods.logger_create_v1(JsonSerializer.Serialize(config));
        if (handle == IntPtr.Zero)
        {
            var err = NativeMethods.LastError(IntPtr.Zero);
            throw new LoggerException(string.IsNullOrEmpty(err) ? "logger_create_v1 failed" : err);
        }

        _handle = new LoggerHandle(handle);
    }

    public static string LibraryVersion() => NativeMethods.PtrToUtf8(NativeMethods.logger_version());

    public static int AbiVersion() => NativeMethods.logger_abi_version();

    private void Log(Level level, string message, IReadOnlyDictionary<string, object?>? fields)
    {
        ObjectDisposedException.ThrowIf(_disposed || _handle.IsInvalid, this);
        var payload = JsonSerializer.Serialize(fields ?? new Dictionary<string, object?>());
        if (NativeMethods.logger_log(_handle.Value, (int)level, message, payload) != 0)
        {
            var err = NativeMethods.LastError(_handle.Value);
            throw new LoggerException(string.IsNullOrEmpty(err) ? $"logger_log({level}) failed" : err);
        }
    }

    public void LogSimple(Level level, string message)
    {
        ObjectDisposedException.ThrowIf(_disposed || _handle.IsInvalid, this);
        if (NativeMethods.logger_log_simple(_handle.Value, (int)level, message) != 0)
        {
            var err = NativeMethods.LastError(_handle.Value);
            throw new LoggerException(string.IsNullOrEmpty(err) ? "logger_log_simple failed" : err);
        }
    }

    public void Trace(string message, IReadOnlyDictionary<string, object?>? fields = null) => Log(Level.Trace, message, fields);
    public void Debug(string message, IReadOnlyDictionary<string, object?>? fields = null) => Log(Level.Debug, message, fields);
    public void Info(string message, IReadOnlyDictionary<string, object?>? fields = null) => Log(Level.Info, message, fields);
    public void Warn(string message, IReadOnlyDictionary<string, object?>? fields = null) => Log(Level.Warn, message, fields);
    public void Error(string message, IReadOnlyDictionary<string, object?>? fields = null) => Log(Level.Error, message, fields);
    /// <summary>Writes at fatal level. Does NOT exit the process.</summary>
    public void Fatal(string message, IReadOnlyDictionary<string, object?>? fields = null) => Log(Level.Fatal, message, fields);

    public void SetFields(IReadOnlyDictionary<string, object?> fields)
    {
        ObjectDisposedException.ThrowIf(_disposed || _handle.IsInvalid, this);
        if (NativeMethods.logger_set_fields(_handle.Value, JsonSerializer.Serialize(fields)) != 0)
        {
            var err = NativeMethods.LastError(_handle.Value);
            throw new LoggerException(string.IsNullOrEmpty(err) ? "logger_set_fields failed" : err);
        }
    }

    public void ReloadConfig(object config)
    {
        ObjectDisposedException.ThrowIf(_disposed || _handle.IsInvalid, this);
        if (NativeMethods.logger_reload_config(_handle.Value, JsonSerializer.Serialize(config)) != 0)
        {
            var err = NativeMethods.LastError(_handle.Value);
            throw new LoggerException(string.IsNullOrEmpty(err) ? "logger_reload_config failed" : err);
        }
    }

    public Dictionary<string, JsonElement> Stats()
    {
        ObjectDisposedException.ThrowIf(_disposed || _handle.IsInvalid, this);
        var raw = NativeMethods.PtrToUtf8(NativeMethods.logger_stats(_handle.Value));
        return JsonSerializer.Deserialize<Dictionary<string, JsonElement>>(raw)
               ?? new Dictionary<string, JsonElement>();
    }

    public void Flush()
    {
        ObjectDisposedException.ThrowIf(_disposed || _handle.IsInvalid, this);
        if (NativeMethods.logger_flush(_handle.Value) != 0)
        {
            var err = NativeMethods.LastError(_handle.Value);
            throw new LoggerException(string.IsNullOrEmpty(err) ? "logger_flush failed" : err);
        }
    }

    public void Dispose()
    {
        if (_disposed)
        {
            return;
        }
        _disposed = true;
        _handle.Dispose();
        GC.SuppressFinalize(this);
    }
}
