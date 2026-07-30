#!/usr/bin/env python3
"""Parse bench result text into bench/results/summary.json for charts."""
from __future__ import annotations

import json
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RES = ROOT / "bench" / "results"


def parse_go_ops(text: str) -> dict[str, float]:
    out: dict[str, float] = {}
    # BenchmarkPolyglotSyncFile-12    1450    35981 ns/op
    for name, key in [
        ("BenchmarkPolyglotSyncFile", "Polyglot"),
        ("BenchmarkZapJSONFile", "Zap"),
        ("BenchmarkZerologFile", "Zerolog"),
        ("BenchmarkSlogJSONFile", "slog (ref)"),
    ]:
        m = re.search(rf"^{name}\S*\s+\d+\s+([\d.]+)\s+ns/op", text, re.M)
        if m:
            ns = float(m.group(1))
            if ns > 0:
                out[key] = 1e9 / ns
    return out


def parse_go_p99(text: str) -> dict[str, float]:
    out: dict[str, float] = {}
    # ReportMetric format: <value> ns/p99  (value PRECEDES the unit)
    for name, key in [
        ("BenchmarkPolyglotSyncFile", "Polyglot"),
        ("BenchmarkZapJSONFile", "Zap"),
        ("BenchmarkZerologFile", "Zerolog"),
    ]:
        block = re.search(rf"{name}\S*.*?([\d.]+)\s+ns/p99", text, re.S)
        if block:
            out[key] = float(block.group(1))
            continue
        # Fallback from b.Logf: p99=574000ns
        m = re.search(rf"{name}.*?p99=(\d+)ns", text, re.S)
        if m:
            out[key] = float(m.group(1))
    return out


def parse_node(text: str) -> tuple[dict[str, float], dict[str, float]]:
    ops: dict[str, float] = {}
    p99: dict[str, float] = {}
    for line in text.splitlines():
        # polyglot_sync_file runtime=node ops/s=12237 ... p99=411300
        m = re.search(r"^(polyglot_sync_file|pino_sync_file)\s+.*?ops/s=([\d.]+).*?p99=([\d.]+)", line)
        if not m:
            continue
        label = "Polyglot (Node)" if m.group(1).startswith("polyglot") else "Pino"
        ops[label] = float(m.group(2))
        p99[label] = float(m.group(3))
    return ops, p99


def parse_ffi(text: str) -> dict[str, float]:
    out: dict[str, float] = {}
    m = re.search(r"ffi_crossing≈([\d.]+)ns.*?full_log≈([\d.]+)ns", text)
    if m:
        out["FFI only"] = float(m.group(1))
        out["Full sync log"] = float(m.group(2))
    return out


def parse_binding_ops(text: str, polyglot_key: str, competitor_key: str, competitor_prefix: str) -> dict[str, float]:
    """Parse polyglot_* / competitor_* lines from python or dotnet benches."""
    out: dict[str, float] = {}
    for line in text.splitlines():
        m = re.search(r"^(polyglot_sync_file|" + competitor_prefix + r")\s+.*?ops/s=([\d.]+)", line)
        if not m:
            continue
        label = polyglot_key if m.group(1).startswith("polyglot") else competitor_key
        out[label] = float(m.group(2))
    return out


def parse_scale_csv(path: Path) -> dict[str, float]:
    if not path.exists():
        return {}
    out: dict[str, float] = {}
    for line in path.read_text(encoding="utf-8").splitlines()[1:]:
        parts = line.split(",")
        if len(parts) >= 2:
            out[parts[0]] = float(parts[1])
    return out


def main() -> None:
    RES.mkdir(parents=True, exist_ok=True)
    go = (RES / "go.txt").read_text(encoding="utf-8") if (RES / "go.txt").exists() else ""
    node = (RES / "node.txt").read_text(encoding="utf-8") if (RES / "node.txt").exists() else ""
    ffi = (RES / "ffi.txt").read_text(encoding="utf-8") if (RES / "ffi.txt").exists() else ""
    py = (RES / "python.txt").read_text(encoding="utf-8") if (RES / "python.txt").exists() else ""
    dn = (RES / "dotnet.txt").read_text(encoding="utf-8") if (RES / "dotnet.txt").exists() else ""

    sync_ops = parse_go_ops(go)
    lat = parse_go_p99(go)
    n_ops, n_p99 = parse_node(node)
    for k, v in n_p99.items():
        lat[k] = v

    py_ops = parse_binding_ops(py, "Polyglot (Python)", "stdlib logging", "stdlib_sync_file")
    dn_ops = parse_binding_ops(dn, "Polyglot (.NET)", "Serilog", "serilog_sync_file")
    bindings_ops = {**py_ops, **dn_ops}

    summary = {
        "sync_file_ops": sync_ops or {"Polyglot": 0, "Zap": 0, "Zerolog": 0},
        "latency_p99_ns": lat,
        "scale_ops": parse_scale_csv(RES / "scale.csv")
        or {"1": 0, "2": 0, "4": 0, "8": 0, "16": 0, "32": 0, "64": 0},
        "ffi_ns": parse_ffi(ffi) or {"FFI only": 0, "Full sync log": 0},
        "node_ops": n_ops,
        "bindings_ops": bindings_ops,
    }
    (RES / "summary.json").write_text(json.dumps(summary, indent=2) + "\n", encoding="utf-8")
    print(f"wrote {RES / 'summary.json'}")


if __name__ == "__main__":
    main()
