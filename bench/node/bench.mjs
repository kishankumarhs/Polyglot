/**
 * Node/Bun vs Pino file sink with P50/P95/P99 latency.
 * Runtime is detected via process.versions.bun.
 */
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { performance } from "node:perf_hooks";
import { createRequire } from "node:module";
import pino from "pino";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(__dirname, "../..");
const require = createRequire(import.meta.url);

const runtime = process.versions.bun ? "bun" : "node";
const N = Number(process.env.BENCH_NODE_N || 20000);
const ITER = Number(process.env.BENCH_NODE_ITERS || 3);

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
  throw new Error(`native lib missing: ${name}; run make build-native`);
}

process.env.POLYGLOT_LOGGER_LIB = resolveLib();
const { Logger } = require(path.join(root, "bindings/node"));

const rich = (n) => ({
  user_id: 7,
  trace_id: "abc123def456ghi789",
  span_id: "span-001",
  service: "payments",
  region: "us-east-1",
  latency_ms: 12.4,
  ok: true,
  tags: ["a", "b", "c"],
  meta: { cart: { items: 3, currency: "USD" } },
  error: "optional message",
  request_id: "req-xyz",
  tenant: "acme",
  env: "prod",
  version: "1.2.3",
  attempt: 1,
  bytes: 4096,
  cached: false,
  n,
});

function percentiles(samplesNs) {
  const s = samplesNs.slice().sort((a, b) => a - b);
  const at = (p) => s[Math.min(s.length - 1, Math.floor((p / 100) * s.length))];
  const mean = s.reduce((a, b) => a + b, 0) / s.length;
  return { mean, p50: at(50), p95: at(95), p99: at(99), max: s[s.length - 1] };
}

function runCase(name, fn) {
  const runs = [];
  for (let iter = 0; iter < ITER; iter++) {
    const samples = [];
    const t0 = performance.now();
    fn(samples);
    const wallMs = performance.now() - t0;
    const pct = percentiles(samples);
    runs.push({ wallMs, ops: (N / wallMs) * 1000, ...pct });
  }
  runs.sort((a, b) => a.ops - b.ops);
  const mid = runs[Math.floor(runs.length / 2)];
  return { name, runtime, n: N, ...mid };
}

function polyglotSync(samples) {
  const file = path.join(os.tmpdir(), `pg-sync-${process.pid}.log`);
  const log = new Logger({ service: "bench", level: "info", stdout: false, async: false, filePath: file });
  for (let i = 0; i < N; i++) {
    const s = performance.now();
    log.info("checkout", rich(i));
    samples.push((performance.now() - s) * 1e6);
  }
  log.flush();
  log.close();
  try {
    fs.unlinkSync(file);
  } catch {}
}

function polyglotAsync(samples) {
  const file = path.join(os.tmpdir(), `pg-async-${process.pid}.log`);
  const log = new Logger({
    service: "bench",
    level: "info",
    stdout: false,
    async: true,
    queueSize: 10000,
    overflow: "drop_newest",
    filePath: file,
  });
  for (let i = 0; i < N; i++) {
    const s = performance.now();
    log.info("checkout", rich(i));
    samples.push((performance.now() - s) * 1e6);
  }
  log.flush();
  log.close();
  try {
    fs.unlinkSync(file);
  } catch {}
}

function pinoFile(samples) {
  const file = path.join(os.tmpdir(), `pino-${process.pid}.log`);
  const dest = pino.destination({ dest: file, sync: true });
  const log = pino(dest);
  for (let i = 0; i < N; i++) {
    const s = performance.now();
    log.info(rich(i), "checkout");
    samples.push((performance.now() - s) * 1e6);
  }
  log.flush();
  dest.end();
  try {
    fs.unlinkSync(file);
  } catch {}
}

const results = [
  runCase("polyglot_sync_file", polyglotSync),
  runCase("polyglot_async_file", polyglotAsync),
  runCase("pino_sync_file", pinoFile),
];

console.log(JSON.stringify({ results }, null, 2));
for (const r of results) {
  console.log(
    `${r.name} runtime=${r.runtime} ops/s=${r.ops.toFixed(0)} mean_ns=${r.mean.toFixed(0)} p50=${r.p50.toFixed(0)} p95=${r.p95.toFixed(0)} p99=${r.p99.toFixed(0)}`,
  );
}
