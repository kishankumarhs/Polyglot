const { test } = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");

const root = path.resolve(__dirname, "../../..");
const distCandidates = [
  path.join(root, "dist", "logger.dll"),
  path.join(root, "dist", "liblogger.so"),
  path.join(root, "dist", "liblogger.dylib"),
  path.join(__dirname, "..", "native", "logger.dll"),
  path.join(__dirname, "..", "native", "liblogger.so"),
  path.join(__dirname, "..", "native", "liblogger.dylib"),
];

const lib = distCandidates.find((p) => fs.existsSync(p));
if (!lib) {
  test("native library missing", { skip: "native logger library not built" }, () => {});
} else {
  process.env.EXIMIETAS_LOGGER_LIB = lib;
  const { Logger, Level, abiVersion, libraryVersion } = require("../dist/index.js");

  test("writes structured json and filters levels", () => {
    const filePath = path.join(os.tmpdir(), `eximietas-logger-${Date.now()}.log`);
    try {
      const log = new Logger({
        service: "node-smoke",
        environment: "test",
        level: "info",
        stdout: false,
        filePath,
        async: false,
        fields: { team: "platform" },
      });
      log.debug("hidden", { n: 1 });
      log.info("hello", { user_id: 7 });
      log.setFields({ traceId: "t-1" });
      log.logSimple(Level.INFO, "simple");
      log.flush();
      const st = log.stats();
      assert.ok(st.flushed >= 2);
      log.close();

      const lines = fs.readFileSync(filePath, "utf8").trim().split(/\r?\n/);
      assert.equal(lines.length, 2);
      const entry = JSON.parse(lines[0]);
      assert.equal(entry.level, "info");
      assert.equal(entry.message, "hello");
      assert.equal(entry.service_name, "node-smoke");
      assert.equal(entry.fields.team, "platform");
      assert.equal(entry.fields.user_id, 7);
      assert.equal(abiVersion(), 1);
      assert.ok(libraryVersion());
    } finally {
      if (fs.existsSync(filePath)) fs.unlinkSync(filePath);
    }
  });

  test("invalid config throws", () => {
    assert.throws(() => {
      new Logger({ service: "", stdout: true });
    });
  });
}
