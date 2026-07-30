#!/usr/bin/env python3
"""Cross-language consistency: 100k rich logs via Python binding."""
from __future__ import annotations

import json
import os
import sys
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "bindings" / "python"))

name = "logger.dll" if os.name == "nt" else ("liblogger.dylib" if sys.platform == "darwin" else "liblogger.so")
for candidate in [ROOT / "dist" / name, ROOT / "bindings" / "python" / "polyglot_logger" / "native" / name]:
    if candidate.exists():
        os.environ.setdefault("POLYGLOT_LOGGER_LIB", str(candidate))
        break

from polyglot_logger import Logger  # noqa: E402

N = int(os.environ.get("BENCH_CROSS_N", "100000"))
out = Path(os.environ.get("TMPDIR") or os.environ.get("TEMP") or "/tmp") / "polyglot-cross-python.log"
if out.exists():
    out.unlink()

fields = {
    "user_id": 7,
    "trace_id": "abc123",
    "span_id": "span-1",
    "service": "payments",
    "region": "us-east-1",
    "latency_ms": 12.4,
    "ok": True,
    "tags": ["a", "b", "c"],
    "meta": {"cart": {"items": 3, "currency": "USD"}},
    "error": "optional message",
}

with Logger(
    "bench-cross",
    level="info",
    stdout=False,
    file_path=str(out),
    async_mode=False,
) as log:
    t0 = time.perf_counter()
    for i in range(N):
        log.info("checkout", **fields, n=i)
    log.flush()
    elapsed = time.perf_counter() - t0

first = json.loads(out.read_text(encoding="utf-8").splitlines()[0])
print(
    f"lang=python n={N} elapsed={elapsed:.3f}s ops_s={N/elapsed:.0f} path={out} "
    f"schema_keys={','.join(first.keys())}"
)
