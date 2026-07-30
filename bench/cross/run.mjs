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
const { Logger } = require(path.join(root, "bindings/node"));

const N = Number(process.env.BENCH_CROSS_N || 100000);
const out = path.join(os.tmpdir(), "polyglot-cross-node.log");
try {
  fs.unlinkSync(out);
} catch {}

const log = new Logger({
  service: "bench-cross",
  level: "info",
  stdout: false,
  async: false,
  filePath: out,
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
  log.info("checkout", { ...fields, n: i });
}
log.flush();
log.close();
const elapsed = performance.now() - t0;
const first = JSON.parse(fs.readFileSync(out, "utf8").trim().split(/\r?\n/)[0]);
console.log(
  `lang=node n=${N} elapsed=${(elapsed / 1000).toFixed(3)}s ops_s=${((N / elapsed) * 1000).toFixed(0)} path=${out} schema_keys=${Object.keys(first).join(",")}`,
);
