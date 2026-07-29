import koffi from "koffi";
import fs from "fs";
import os from "os";
import path from "path";
import { Level, bindNative, type NativeFns } from "./ffi.generated";

export type LogFields = Record<string, unknown>;
export { Level };

/**
 * findProjectConfig() recursively searches for a `polyglot.yaml` file.
 * Starting from the current working directory, it climbs up the directory tree
 * until the file is found or the file system root is reached.
 * Returns the absolute path to the config file, or null if not found.
 */
function findProjectConfig(): string | null {
  let current = process.cwd();
  const root = path.parse(current).root;

  while (true) {
    const configPath = path.join(current, "polyglot.yaml");
    if (fs.existsSync(configPath)) {
      return path.resolve(configPath);
    }

    if (current === root) {
      // Reached filesystem root without finding config
      return null;
    }

    current = path.dirname(current);
  }
}

/**
 * initializeFromProjectConfig() automatically loads and applies configuration
 * from a project-wide polyglot.yaml file if found.
 * This is called on module import for zero-config initialization.
 * Failures are logged to stderr but do not throw (defensive error handling).
 */
function initializeFromProjectConfig(): void {
  try {
    const configPath = findProjectConfig();
    if (!configPath) {
      // No config file found; use defaults (not an error)
      return;
    }

    // Call the native logger_create_from_config_file to initialize
    const handle = native().logger_create_from_config_file(configPath);
    if (!handle) {
      const err = lastError(null);
      console.error(
        `[polyglot-logger] failed to initialize from config file ${configPath}: ${err || "unknown error"}`,
      );
      return;
    }

    // Store the global logger handle for later use if needed
    globalLoggerHandle = handle;
  } catch (e) {
    console.error(`[polyglot-logger] error during auto-initialization:`, e);
  }
}

// Global logger handle initialized from project config
let globalLoggerHandle: unknown = null;

export interface FileOptions {
  enabled?: boolean;
  path: string;
  maxSizeMb?: number;
  maxBackups?: number;
  maxAgeDays?: number;
  compress?: boolean;
}

export interface HttpOptions {
  enabled?: boolean;
  url: string;
  timeoutMs?: number;
  headers?: Record<string, string>;
  batchSize?: number;
  flushIntervalMs?: number;
}

export interface LoggerOptions {
  service?: string;
  /** @deprecated use service */
  serviceName?: string;
  serviceVersion?: string;
  environment?: string;
  level?: string;
  /** @deprecated use level */
  minLevel?: string;
  stdout?: boolean;
  file?: FileOptions;
  /** @deprecated use file.path */
  filePath?: string;
  maxSizeMb?: number;
  maxBackups?: number;
  maxAgeDays?: number;
  http?: HttpOptions;
  async?: boolean;
  queueSize?: number;
  overflow?: "drop_newest" | "drop_oldest" | "block";
  fields?: LogFields;
}

export class LoggerError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "LoggerError";
  }
}

function libraryName(): string {
  switch (os.platform()) {
    case "win32":
      return "logger.dll";
    case "darwin":
      return "liblogger.dylib";
    default:
      return "liblogger.so";
  }
}

function candidatePaths(): string[] {
  const name = libraryName();
  const env = process.env.POLYGLOT_LOGGER_LIB;
  const roots = [
    env,
    path.join(__dirname, "..", "native", name),
    path.join(__dirname, "..", "..", "..", "dist", name),
    path.join(__dirname, "..", "..", "..", "build", name),
    path.join(process.cwd(), name),
    path.join(process.cwd(), "dist", name),
    path.join(process.cwd(), "build", name),
  ];
  return roots.filter((p): p is string => Boolean(p));
}

function resolveLibraryPath(): string {
  for (const candidate of candidatePaths()) {
    if (fs.existsSync(candidate)) {
      return candidate;
    }
  }
  throw new LoggerError(
    `unable to load native logger library ${libraryName()}; searched: ${candidatePaths().join(", ")}`,
  );
}

// The shared library is loaded on first use rather than at import time, so
// importing this package cannot crash a service (or a test run) that has not
// built the native artifact yet.
let cached: NativeFns | undefined;

function native(): NativeFns {
  if (!cached) {
    cached = bindNative(koffi.load(resolveLibraryPath()));
  }
  return cached;
}

function lastError(handle: unknown = null): string {
  return (native().logger_last_error(handle as never) as string) || "";
}

export function libraryVersion(): string {
  return (native().logger_version() as string) || "";
}

export function abiVersion(): number {
  return native().logger_abi_version() as number;
}

function buildConfig(options: LoggerOptions): Record<string, unknown> {
  const service = options.service || options.serviceName || "";
  const config: Record<string, unknown> = {
    service,
    service_version: options.serviceVersion ?? "",
    environment: options.environment ?? "",
    level: options.level ?? options.minLevel ?? "info",
    stdout: options.stdout ?? true,
    async: options.async ?? true,
    queueSize: options.queueSize ?? 10000,
    overflow: options.overflow ?? "drop_newest",
    fields: options.fields ?? {},
  };
  if (options.file) {
    config.file = {
      enabled: options.file.enabled ?? true,
      path: options.file.path,
      maxSizeMB: options.file.maxSizeMb ?? 100,
      maxBackups: options.file.maxBackups ?? 10,
      maxAgeDays: options.file.maxAgeDays ?? 30,
      compress: options.file.compress ?? false,
    };
  } else if (options.filePath) {
    config.file = {
      enabled: true,
      path: options.filePath,
      maxSizeMB: options.maxSizeMb ?? 100,
      maxBackups: options.maxBackups ?? 10,
      maxAgeDays: options.maxAgeDays ?? 30,
      compress: false,
    };
  }
  if (options.http) {
    config.http = {
      enabled: options.http.enabled ?? true,
      url: options.http.url,
      timeout_ms: options.http.timeoutMs ?? 5000,
      headers: options.http.headers ?? {},
      batch_size: options.http.batchSize ?? 50,
      flush_interval_ms: options.http.flushIntervalMs ?? 1000,
    };
  }
  return config;
}

/**
 * Thread-safety: log/flush/stats/setFields/reloadConfig are safe for concurrent
 * use on one instance. Prefer a single owner for close().
 */
export class Logger {
  private handle: unknown;

  constructor(options: LoggerOptions) {
    const handle = native().logger_create_v1(
      JSON.stringify(buildConfig(options)),
    );
    if (!handle) {
      throw new LoggerError(lastError(null) || "logger_create_v1 failed");
    }
    this.handle = handle;
  }

  private ensureOpen(): void {
    if (!this.handle) {
      throw new LoggerError("logger is closed");
    }
  }

  log(level: Level | number, message: string, fields?: LogFields): void {
    this.ensureOpen();
    const rc = native().logger_log(
      this.handle,
      level,
      message,
      JSON.stringify(fields ?? {}),
    ) as number;
    if (rc !== 0) {
      throw new LoggerError(
        lastError(this.handle) || `logger_log(${level}) failed`,
      );
    }
  }

  logSimple(level: Level | number, message: string): void {
    this.ensureOpen();
    const rc = native().logger_log_simple(
      this.handle,
      level,
      message,
    ) as number;
    if (rc !== 0) {
      throw new LoggerError(
        lastError(this.handle) || "logger_log_simple failed",
      );
    }
  }

  trace(message: string, fields?: LogFields): void {
    this.log(Level.TRACE, message, fields);
  }

  debug(message: string, fields?: LogFields): void {
    this.log(Level.DEBUG, message, fields);
  }

  info(message: string, fields?: LogFields): void {
    this.log(Level.INFO, message, fields);
  }

  warn(message: string, fields?: LogFields): void {
    this.log(Level.WARN, message, fields);
  }

  error(message: string, fields?: LogFields): void {
    this.log(Level.ERROR, message, fields);
  }

  /** Writes at fatal level. Does NOT exit the process. */
  fatal(message: string, fields?: LogFields): void {
    this.log(Level.FATAL, message, fields);
  }

  setFields(fields: LogFields): void {
    this.ensureOpen();
    if (
      (native().logger_set_fields(
        this.handle,
        JSON.stringify(fields),
      ) as number) !== 0
    ) {
      throw new LoggerError(
        lastError(this.handle) || "logger_set_fields failed",
      );
    }
  }

  reloadConfig(config: Record<string, unknown>): void {
    this.ensureOpen();
    if (
      (native().logger_reload_config(
        this.handle,
        JSON.stringify(config),
      ) as number) !== 0
    ) {
      throw new LoggerError(
        lastError(this.handle) || "logger_reload_config failed",
      );
    }
  }

  stats(): Record<string, number> {
    this.ensureOpen();
    const raw = (native().logger_stats(this.handle) as string) || "{}";
    return JSON.parse(raw) as Record<string, number>;
  }

  flush(): void {
    if (!this.handle) {
      return;
    }
    if ((native().logger_flush(this.handle) as number) !== 0) {
      throw new LoggerError(lastError(this.handle) || "logger_flush failed");
    }
  }

  close(): void {
    if (!this.handle) {
      return;
    }
    const handle = this.handle;
    this.handle = null;
    if ((native().logger_close(handle) as number) !== 0) {
      throw new LoggerError(lastError(null) || "logger_close failed");
    }
  }
}

// Auto-initialize from project-wide polyglot.yaml on module import
initializeFromProjectConfig();
