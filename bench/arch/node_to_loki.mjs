/**
 * Architecture bench: Node → Polyglot Loki sink → loki-mock.
 * Start mock first: go run ./bench/arch/loki_mock.go -addr 127.0.0.1:3100
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
  for (const p of [
    path.join(root, "dist", name),
    path.join(root, "bindings/node/native", name),
  ]) {
    if (fs.existsSync(p)) return p;
  }
  throw new Error(`native lib not found (${name}); run make build-native`);
}

process.env.POLYGLOT_LOGGER_LIB = resolveLib();

const { Logger } = require(path.join(root, "bindings/node"));

const N = Number(process.env.BENCH_LOKI_N || 10000);
const url = process.env.LOKI_URL || "http://127.0.0.1:3100/loki/api/v1/push";
const samples = [];

const tmpFile = path.join(os.tmpdir(), `polyglot-loki-bench-${process.pid}.log`);
const log = new Logger({
  service: "bench-arch",
  level: "info",
  stdout: false,
  async: true,
  queueSize: 10000,
  overflow: "drop_newest",
  filePath: tmpFile,
});

// Switch to native Loki sink (binding options don't expose loki yet).
log.reloadConfig({
  service: "bench-arch",
  level: "info",
  stdout: false,
  async: true,
  queueSize: 10000,
  overflow: "drop_newest",
  loki: {
    enabled: true,
    url,
    batch_size: 50,
    flush_interval_ms: 200,
    timeout_ms: 5000,
    labels: { job: "bench" },
  },
});

const fields = {
  user_id: 7,
  trace_id: "abc123",
  span_id: "span-1",
  service: "payments",
  region: "us-east-1",
  latency_ms: 12.4,
  ok: true,
  tags: ["a", "b", "c"],
  meta: { cart: { items: 3, currency: "USD" } },
  error: "optional message",
};

const t0 = performance.now();
for (let i = 0; i < N; i++) {
  const s = performance.now();
  log.info("checkout", { ...fields, n: i });
  samples.push((performance.now() - s) * 1e6);
}
log.flush();
log.close();
const wallMs = performance.now() - t0;

samples.sort((a, b) => a - b);
const pct = (p) => samples[Math.min(samples.length - 1, Math.floor((p / 100) * samples.length))];

let mockStats = {};
try {
  const res = await fetch("http://127.0.0.1:3100/stats");
  mockStats = await res.json();
} catch (e) {
  mockStats = { error: String(e) };
}

console.log(
  JSON.stringify(
    {
      n: N,
      wall_ms: wallMs,
      ops_s: (N / wallMs) * 1000,
      ns_p50: pct(50),
      ns_p95: pct(95),
      ns_p99: pct(99),
      mock: mockStats,
      tmp: os.tmpdir(),
    },
    null,
    2,
  ),
);
