import { Logger, Level, abiVersion, libraryVersion } from "../../bindings/node/dist/index.js";

const log = new Logger({
  service: "node-example",
  environment: "dev",
  level: "debug",
  stdout: true,
  async: true,
  overflow: "drop_newest",
});

console.log("version", libraryVersion(), "abi", abiVersion());
log.setFields({ traceId: "demo-trace" });
log.info("hello from node", { userId: 1 });
log.logSimple(Level.WARN, "simple warning");
log.flush();
console.log("stats", log.stats());
log.close();
