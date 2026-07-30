#!/usr/bin/env python3
"""Polyglot vs stdlib logging — sync file, rich payload, ops/s + p99."""
from __future__ import annotations

import json
import logging
import os
import statistics
import sys
import tempfile
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

N = int(os.environ.get("BENCH_PY_N", "20000"))
ITERS = int(os.environ.get("BENCH_PY_ITERS", "3"))


def rich(n: int) -> dict:
    return {
        "user_id": 7,
        "trace_id": "abc123def456ghi789",
        "span_id": "span-001",
        "service": "payments",
        "region": "us-east-1",
        "latency_ms": 12.4,
        "ok": True,
        "tags": ["a", "b", "c"],
        "meta": {"cart": {"items": 3, "currency": "USD"}},
        "error": "optional message",
        "request_id": "req-xyz",
        "tenant": "acme",
        "env": "prod",
        "version": "1.2.3",
        "attempt": 1,
        "bytes": 4096,
        "cached": False,
        "n": n,
    }


def pct(samples_ns: list[float]) -> dict[str, float]:
    s = sorted(samples_ns)
    def at(p: float) -> float:
        return s[min(len(s) - 1, int((p / 100) * len(s)))]
    return {
        "mean": statistics.fmean(s),
        "p50": at(50),
        "p95": at(95),
        "p99": at(99),
    }


def run_case(name: str, fn) -> dict:
    runs = []
    for _ in range(ITERS):
        samples: list[float] = []
        t0 = time.perf_counter()
        fn(samples)
        wall = time.perf_counter() - t0
        runs.append({"ops": N / wall, "wall": wall, **pct(samples)})
    runs.sort(key=lambda r: r["ops"])
    mid = runs[len(runs) // 2]
    return {"name": name, "n": N, **mid}


def polyglot_sync(samples: list[float]) -> None:
    path = Path(tempfile.gettempdir()) / f"pg-py-{os.getpid()}.log"
    if path.exists():
        path.unlink()
    with Logger(
        "bench",
        level="info",
        stdout=False,
        file_path=str(path),
        async_mode=False,
    ) as log:
        for i in range(N):
            t0 = time.perf_counter_ns()
            log.info("checkout", **rich(i))
            samples.append(time.perf_counter_ns() - t0)
        log.flush()
    path.unlink(missing_ok=True)


class JsonFormatter(logging.Formatter):
    def format(self, record: logging.LogRecord) -> str:
        payload = {
            "level": record.levelname.lower(),
            "message": record.getMessage(),
        }
        extra = getattr(record, "polyglot_fields", None)
        if isinstance(extra, dict):
            payload.update(extra)
        return json.dumps(payload, separators=(",", ":"))


def stdlib_sync(samples: list[float]) -> None:
    path = Path(tempfile.gettempdir()) / f"stdlib-py-{os.getpid()}.log"
    if path.exists():
        path.unlink()
    logger = logging.getLogger(f"stdlib-{os.getpid()}")
    logger.handlers.clear()
    logger.setLevel(logging.INFO)
    logger.propagate = False
    handler = logging.FileHandler(path, encoding="utf-8")
    handler.setFormatter(JsonFormatter())
    logger.addHandler(handler)
    for i in range(N):
        fields = rich(i)
        t0 = time.perf_counter_ns()
        logger.info("checkout", extra={"polyglot_fields": fields})
        samples.append(time.perf_counter_ns() - t0)
    handler.close()
    logger.handlers.clear()
    path.unlink(missing_ok=True)


def main() -> None:
    if not os.environ.get("POLYGLOT_LOGGER_LIB"):
        print("polyglot python bench skipped: no native lib (set POLYGLOT_LOGGER_LIB)", file=sys.stderr)
        sys.exit(0)
    for case in (run_case("polyglot_sync_file", polyglot_sync), run_case("stdlib_sync_file", stdlib_sync)):
        print(
            f"{case['name']} runtime=python ops/s={case['ops']:.0f} "
            f"mean={case['mean']:.0f} p50={case['p50']:.0f} p95={case['p95']:.0f} p99={case['p99']:.0f} n={case['n']}"
        )


if __name__ == "__main__":
    main()
