/**
 * FFI decomposition: cheap ABI call vs full structured log.
 */
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { performance } from "node:perf_hooks";
import { createRequire } from "node:module";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(__dirname, "../..");
const require = createRequire(import.meta.url);

function resolveLib() {
  if (process.env.POLYGLOT_LOGGER_LIB) return process.env.POLYGLOT_LOGGER_LIB;
  const name =
    process.platform === "win32"
      ? "logger.dll"
      : process.platform === "darwin"
        ? "liblogger.dylib"
        : "liblogger.so";
  for (const p of [path.join(root, "dist", name), path.join(root, "bindings/node/native", name)]) {
    if (fs.existsSync(p)) return p;
  }
  throw new Error(`native lib missing: ${name}`);
}

process.env.POLYGLOT_LOGGER_LIB = resolveLib();
const { Logger, libraryVersion, abiVersion } = require(path.join(root, "bindings/node"));

const N = Number(process.env.BENCH_FFI_N || 100000);

function timed(label, fn) {
  const samples = [];
  const t0 = performance.now();
  for (let i = 0; i < N; i++) {
    const s = performance.now();
    fn(i);
    samples.push((performance.now() - s) * 1e6);
  }
  const wall = performance.now() - t0;
  samples.sort((a, b) => a - b);
  const at = (p) => samples[Math.min(samples.length - 1, Math.floor((p / 100) * samples.length))];
  const mean = samples.reduce((a, b) => a + b, 0) / samples.length;
  return {
    label,
    n: N,
    wall_ms: wall,
    ops_s: (N / wall) * 1000,
    mean_ns: mean,
    p50: at(50),
    p95: at(95),
    p99: at(99),
  };
}

// Warm native load
libraryVersion();
abiVersion();

const ffiOnly = timed("ffi_logger_version", () => {
  libraryVersion();
});

const file = path.join(os.tmpdir(), `ffi-full-${process.pid}.log`);
const log = new Logger({ service: "ffi-bench", level: "info", stdout: false, async: false, filePath: file });
const full = timed("full_sync_info", (i) => {
  log.info("checkout", { user_id: 7, n: i, tags: ["a", "b"], ok: true });
});
log.flush();
log.close();
try {
  fs.unlinkSync(file);
} catch {}

const ratio = full.mean_ns / Math.max(ffiOnly.mean_ns, 1);
console.log(JSON.stringify({ ffiOnly, full, full_over_ffi_mean: ratio }, null, 2));
console.log(
  `ffi_crossing≈${ffiOnly.mean_ns.toFixed(0)}ns mean; full_log≈${full.mean_ns.toFixed(0)}ns mean; ratio=${ratio.toFixed(1)}x`,
);
