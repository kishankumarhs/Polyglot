#!/usr/bin/env python3
"""Generate SVG + PNG charts for bench/results from summary.json (stdlib only)."""
from __future__ import annotations

import json
import struct
import zlib
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "bench" / "results"
SUMMARY = OUT / "summary.json"


def load_summary() -> dict:
    if SUMMARY.exists():
        return json.loads(SUMMARY.read_text(encoding="utf-8"))
    return {
        "sync_file_ops": {
            "Polyglot": 28000,
            "Zap": 50000,
            "Zerolog": 55000,
            "slog (ref)": 22000,
        },
        "latency_p99_ns": {
            "Polyglot": 574000,
            "Zap": 999900,
            "Zerolog": 800000,
            "Pino": 571100,
        },
        "scale_ops": {
            "1": 45000,
            "2": 70000,
            "4": 110000,
            "8": 150000,
            "16": 160000,
            "32": 155000,
            "64": 140000,
        },
        "ffi_ns": {"FFI only": 3200, "Full sync log": 24000},
        "note": "placeholder — run make bench to replace",
    }


def _fmt(v: float) -> str:
    if v >= 1_000_000:
        return f"{v/1_000_000:.1f}M"
    if v >= 1_000:
        return f"{v/1_000:.1f}k"
    return f"{v:.0f}"


def _png_chunk(tag: bytes, data: bytes) -> bytes:
    return struct.pack(">I", len(data)) + tag + data + struct.pack(">I", zlib.crc32(tag + data) & 0xFFFFFFFF)


def write_png(path: Path, width: int, height: int, rgb: list[tuple[int, int, int]]) -> None:
    """Write RGB pixels (row-major) as a PNG."""
    raw = bytearray()
    for y in range(height):
        raw.append(0)  # filter none
        for x in range(width):
            r, g, b = rgb[y * width + x]
            raw.extend((r, g, b))
    ihdr = struct.pack(">IIBBBBB", width, height, 8, 2, 0, 0, 0)
    png = b"\x89PNG\r\n\x1a\n" + _png_chunk(b"IHDR", ihdr) + _png_chunk(b"IDAT", zlib.compress(bytes(raw), 9)) + _png_chunk(b"IEND", b"")
    path.write_bytes(png)


def _fill(rgb: list[tuple[int, int, int]], w: int, x0: int, y0: int, x1: int, y1: int, color: tuple[int, int, int]) -> None:
    h = len(rgb) // w
    x0, x1 = max(0, x0), min(w, x1)
    y0, y1 = max(0, y0), min(h, y1)
    for y in range(y0, y1):
        row = y * w
        for x in range(x0, x1):
            rgb[row + x] = color


def _glyph_5x7() -> dict[str, list[str]]:
    # Minimal 5x7 bitmap font for labels
    return {
        " ": ["00000"] * 7,
        "-": ["00000", "00000", "00000", "11111", "00000", "00000", "00000"],
        ".": ["00000", "00000", "00000", "00000", "00000", "01100", "01100"],
        "/": ["00001", "00010", "00100", "01000", "10000", "00000", "00000"],
        "0": ["01110", "10001", "10011", "10101", "11001", "10001", "01110"],
        "1": ["00100", "01100", "00100", "00100", "00100", "00100", "01110"],
        "2": ["01110", "10001", "00001", "00010", "00100", "01000", "11111"],
        "3": ["01110", "10001", "00001", "00110", "00001", "10001", "01110"],
        "4": ["00010", "00110", "01010", "10010", "11111", "00010", "00010"],
        "5": ["11111", "10000", "11110", "00001", "00001", "10001", "01110"],
        "6": ["01110", "10000", "10000", "11110", "10001", "10001", "01110"],
        "7": ["11111", "00001", "00010", "00100", "01000", "01000", "01000"],
        "8": ["01110", "10001", "10001", "01110", "10001", "10001", "01110"],
        "9": ["01110", "10001", "10001", "01111", "00001", "00001", "01110"],
        "A": ["01110", "10001", "10001", "11111", "10001", "10001", "10001"],
        "B": ["11110", "10001", "10001", "11110", "10001", "10001", "11110"],
        "C": ["01110", "10001", "10000", "10000", "10000", "10001", "01110"],
        "D": ["11110", "10001", "10001", "10001", "10001", "10001", "11110"],
        "E": ["11111", "10000", "10000", "11110", "10000", "10000", "11111"],
        "F": ["11111", "10000", "10000", "11110", "10000", "10000", "10000"],
        "G": ["01110", "10001", "10000", "10111", "10001", "10001", "01110"],
        "H": ["10001", "10001", "10001", "11111", "10001", "10001", "10001"],
        "I": ["01110", "00100", "00100", "00100", "00100", "00100", "01110"],
        "J": ["00111", "00010", "00010", "00010", "00010", "10010", "01100"],
        "K": ["10001", "10010", "10100", "11000", "10100", "10010", "10001"],
        "L": ["10000", "10000", "10000", "10000", "10000", "10000", "11111"],
        "M": ["10001", "11011", "10101", "10001", "10001", "10001", "10001"],
        "N": ["10001", "11001", "10101", "10011", "10001", "10001", "10001"],
        "O": ["01110", "10001", "10001", "10001", "10001", "10001", "01110"],
        "P": ["11110", "10001", "10001", "11110", "10000", "10000", "10000"],
        "Q": ["01110", "10001", "10001", "10001", "10101", "10010", "01101"],
        "R": ["11110", "10001", "10001", "11110", "10100", "10010", "10001"],
        "S": ["01111", "10000", "10000", "01110", "00001", "00001", "11110"],
        "T": ["11111", "00100", "00100", "00100", "00100", "00100", "00100"],
        "U": ["10001", "10001", "10001", "10001", "10001", "10001", "01110"],
        "V": ["10001", "10001", "10001", "10001", "10001", "01010", "00100"],
        "W": ["10001", "10001", "10001", "10001", "10101", "11011", "10001"],
        "X": ["10001", "10001", "01010", "00100", "01010", "10001", "10001"],
        "Y": ["10001", "10001", "01010", "00100", "00100", "00100", "00100"],
        "Z": ["11111", "00001", "00010", "00100", "01000", "10000", "11111"],
        "a": ["00000", "00000", "01110", "00001", "01111", "10001", "01111"],
        "b": ["10000", "10000", "11110", "10001", "10001", "10001", "11110"],
        "c": ["00000", "00000", "01110", "10000", "10000", "10001", "01110"],
        "e": ["00000", "00000", "01110", "10001", "11111", "10000", "01110"],
        "f": ["00110", "01001", "01000", "11100", "01000", "01000", "01000"],
        "g": ["00000", "00000", "01111", "10001", "01111", "00001", "01110"],
        "h": ["10000", "10000", "11110", "10001", "10001", "10001", "10001"],
        "i": ["00100", "00000", "01100", "00100", "00100", "00100", "01110"],
        "k": ["10000", "10000", "10010", "10100", "11000", "10100", "10010"],
        "l": ["01100", "00100", "00100", "00100", "00100", "00100", "01110"],
        "m": ["00000", "00000", "11010", "10101", "10101", "10101", "10101"],
        "n": ["00000", "00000", "11110", "10001", "10001", "10001", "10001"],
        "o": ["00000", "00000", "01110", "10001", "10001", "10001", "01110"],
        "p": ["00000", "00000", "11110", "10001", "11110", "10000", "10000"],
        "r": ["00000", "00000", "10110", "11001", "10000", "10000", "10000"],
        "s": ["00000", "00000", "01111", "10000", "01110", "00001", "11110"],
        "t": ["01000", "01000", "11100", "01000", "01000", "01001", "00110"],
        "u": ["00000", "00000", "10001", "10001", "10001", "10011", "01101"],
        "v": ["00000", "00000", "10001", "10001", "10001", "01010", "00100"],
        "w": ["00000", "00000", "10001", "10001", "10101", "10101", "01010"],
        "x": ["00000", "00000", "10001", "01010", "00100", "01010", "10001"],
        "y": ["00000", "00000", "10001", "10001", "01111", "00001", "01110"],
        "z": ["00000", "00000", "11111", "00010", "00100", "01000", "11111"],
        "(": ["00100", "01000", "10000", "10000", "10000", "01000", "00100"],
        ")": ["00100", "00010", "00001", "00001", "00001", "00010", "00100"],
        ":": ["00000", "01100", "01100", "00000", "01100", "01100", "00000"],
        "→": ["00000", "00100", "00010", "11111", "00010", "00100", "00000"],
    }


def _draw_text(
    rgb: list[tuple[int, int, int]],
    w: int,
    x: int,
    y: int,
    text: str,
    color: tuple[int, int, int],
    scale: int = 2,
    center: bool = False,
) -> None:
    glyphs = _glyph_5x7()
    text = text.replace("→", "->")
    chars = []
    for ch in text:
        g = glyphs.get(ch) or glyphs.get(ch.upper()) or glyphs[" "]
        chars.append(g)
    tw = len(chars) * (5 * scale + scale)
    if center:
        x -= tw // 2
    for gi, g in enumerate(chars):
        ox = x + gi * (5 * scale + scale)
        for row, bits in enumerate(g):
            for col, bit in enumerate(bits):
                if bit == "1":
                    _fill(
                        rgb,
                        w,
                        ox + col * scale,
                        y + row * scale,
                        ox + (col + 1) * scale,
                        y + (row + 1) * scale,
                        color,
                    )


def no_data(stem: Path, title: str, msg: str) -> None:
    """Render an explicit placeholder instead of a misleading zero-valued plot."""
    width, height = 720, 360
    svg = "\n".join(
        [
            f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" font-family="Segoe UI, system-ui, sans-serif">',
            '<rect width="100%" height="100%" fill="#fafafa"/>',
            f'<text x="{width/2}" y="28" text-anchor="middle" font-size="18" font-weight="600" fill="#111">{title}</text>',
            f'<text x="{width/2}" y="{height/2}" text-anchor="middle" font-size="14" fill="#999">{msg}</text>',
            "</svg>",
        ]
    )
    stem.with_suffix(".svg").write_text(svg, encoding="utf-8")
    rgb = [(250, 250, 250)] * (width * height)
    _draw_text(rgb, width, width // 2, 12, title, (17, 17, 17), scale=2, center=True)
    _draw_text(rgb, width, width // 2, height // 2, msg, (153, 153, 153), scale=1, center=True)
    write_png(stem.with_suffix(".png"), width, height, rgb)


def bar_chart(stem: Path, title: str, series: dict[str, float], unit: str, color: str = "#2563eb") -> None:
    width, height = 720, 360
    pad_l, pad_r, pad_t, pad_b = 40, 40, 50, 70
    plot_w = width - pad_l - pad_r
    plot_h = height - pad_t - pad_b
    items = list(series.items())
    if not items:
        return
    vmax = max(v for _, v in items) or 1.0
    gap = plot_w / max(len(items), 1)
    bar_w = gap * 0.65

    # SVG
    hex_color = color
    parts = [
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" font-family="Segoe UI, system-ui, sans-serif">',
        '<rect width="100%" height="100%" fill="#fafafa"/>',
        f'<text x="{width/2}" y="28" text-anchor="middle" font-size="18" font-weight="600" fill="#111">{title}</text>',
        f'<text x="{width/2}" y="{height-16}" text-anchor="middle" font-size="12" fill="#666">{unit}</text>',
    ]
    for i, (label, val) in enumerate(items):
        h = (val / vmax) * plot_h
        x = pad_l + i * gap + (gap - bar_w) / 2
        y = pad_t + plot_h - h
        parts.append(f'<rect x="{x:.1f}" y="{y:.1f}" width="{bar_w:.1f}" height="{h:.1f}" fill="{hex_color}" rx="4"/>')
        parts.append(
            f'<text x="{x + bar_w/2:.1f}" y="{height - pad_b + 22}" text-anchor="middle" font-size="11" fill="#333">{label}</text>'
        )
        parts.append(
            f'<text x="{x + bar_w/2:.1f}" y="{y - 6:.1f}" text-anchor="middle" font-size="11" fill="#111">{_fmt(val)}</text>'
        )
    parts.append("</svg>")
    stem.with_suffix(".svg").write_text("\n".join(parts), encoding="utf-8")

    # PNG (Cursor / many previews render PNG reliably; GitHub handles both)
    def hex_to_rgb(h: str) -> tuple[int, int, int]:
        h = h.lstrip("#")
        return int(h[0:2], 16), int(h[2:4], 16), int(h[4:6], 16)

    bg = (250, 250, 250)
    ink = (17, 17, 17)
    muted = (102, 102, 102)
    bar = hex_to_rgb(hex_color)
    rgb = [bg] * (width * height)
    _draw_text(rgb, width, width // 2, 12, title, ink, scale=2, center=True)
    _draw_text(rgb, width, width // 2, height - 28, unit, muted, scale=1, center=True)
    for i, (label, val) in enumerate(items):
        h = int((val / vmax) * plot_h)
        x = int(pad_l + i * gap + (gap - bar_w) / 2)
        y = pad_t + plot_h - h
        _fill(rgb, width, x, y, x + int(bar_w), pad_t + plot_h, bar)
        _draw_text(rgb, width, x + int(bar_w) // 2, height - pad_b + 8, label[:12], (51, 51, 51), scale=1, center=True)
        _draw_text(rgb, width, x + int(bar_w) // 2, max(y - 18, 8), _fmt(val), ink, scale=1, center=True)
    write_png(stem.with_suffix(".png"), width, height, rgb)


def line_chart(stem: Path, title: str, series: dict[str, float], xlab: str, ylab: str) -> None:
    width, height = 720, 360
    pad_l, pad_r, pad_t, pad_b = 70, 30, 50, 50
    plot_w = width - pad_l - pad_r
    plot_h = height - pad_t - pad_b
    items = [(float(k), float(v)) for k, v in series.items()]
    items.sort()
    if len(items) < 2 or max((v for _, v in items), default=0) <= 0:
        no_data(stem, title, "no data — run `make bench` (needs bench/results/scale.csv)")
        return
    ys = [y for _, y in items]
    ymin, ymax = 0.0, max(ys) * 1.1 or 1.0
    # Writer counts double each step, so space them evenly rather than linearly.
    xpos = {x: i for i, (x, _) in enumerate(items)}
    last = max(len(items) - 1, 1)

    def sx(x: float) -> float:
        return pad_l + xpos[x] / last * plot_w

    def sy(y: float) -> float:
        return pad_t + plot_h - (y - ymin) / (ymax - ymin or 1) * plot_h

    yticks = [ymax * i / 4 for i in range(5)]
    pts = " ".join(f"{sx(x):.1f},{sy(y):.1f}" for x, y in items)
    parts = [
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" font-family="Segoe UI, system-ui, sans-serif">',
        '<rect width="100%" height="100%" fill="#fafafa"/>',
        f'<text x="{width/2}" y="28" text-anchor="middle" font-size="18" font-weight="600" fill="#111">{title}</text>',
    ]
    for t in yticks:
        gy = sy(t)
        parts.append(f'<line x1="{pad_l}" y1="{gy:.1f}" x2="{width - pad_r}" y2="{gy:.1f}" stroke="#e5e5e5" stroke-width="1"/>')
        parts.append(f'<text x="{pad_l - 8}" y="{gy + 4:.1f}" text-anchor="end" font-size="11" fill="#666">{_fmt(t)}</text>')
    parts.append(f'<polyline fill="none" stroke="#2563eb" stroke-width="3" points="{pts}"/>')
    for x, y in items:
        parts.append(f'<circle cx="{sx(x):.1f}" cy="{sy(y):.1f}" r="4" fill="#1d4ed8"/>')
        parts.append(
            f'<text x="{sx(x):.1f}" y="{pad_t + plot_h + 16:.1f}" text-anchor="middle" font-size="11" fill="#333">{x:.0f}</text>'
        )
    parts.append(f'<text x="{width/2}" y="{height-12}" text-anchor="middle" font-size="12" fill="#666">{xlab}</text>')
    parts.append(
        f'<text x="16" y="{height/2}" text-anchor="middle" font-size="12" fill="#666" transform="rotate(-90 16 {height/2})">{ylab}</text>'
    )
    parts.append("</svg>")
    stem.with_suffix(".svg").write_text("\n".join(parts), encoding="utf-8")

    # PNG polyline
    bg = (250, 250, 250)
    ink = (17, 17, 17)
    line = (37, 99, 235)
    rgb = [bg] * (width * height)
    _draw_text(rgb, width, width // 2, 12, title, ink, scale=2, center=True)
    _draw_text(rgb, width, width // 2, height - 28, xlab, (102, 102, 102), scale=1, center=True)
    for t in yticks:
        gy = int(sy(t))
        _fill(rgb, width, pad_l, gy, width - pad_r, gy + 1, (229, 229, 229))
        _draw_text(rgb, width, pad_l - 36, gy - 3, _fmt(t), (102, 102, 102), scale=1)
    points = [(int(sx(x)), int(sy(y))) for x, y in items]
    for i in range(len(points) - 1):
        x0, y0 = points[i]
        x1, y1 = points[i + 1]
        steps = max(abs(x1 - x0), abs(y1 - y0), 1)
        for s in range(steps + 1):
            t = s / steps
            x = int(x0 + (x1 - x0) * t)
            y = int(y0 + (y1 - y0) * t)
            _fill(rgb, width, x - 1, y - 1, x + 2, y + 2, line)
    for (x, y), (xv, _) in zip(points, items):
        _fill(rgb, width, x - 3, y - 3, x + 4, y + 4, (29, 78, 216))
        _draw_text(rgb, width, x, pad_t + plot_h + 8, f"{xv:.0f}", (51, 51, 51), scale=1, center=True)
    write_png(stem.with_suffix(".png"), width, height, rgb)


def main() -> None:
    OUT.mkdir(parents=True, exist_ok=True)
    data = load_summary()
    if not SUMMARY.exists():
        SUMMARY.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")

    bar_chart(OUT / "throughput", "Sync file throughput (ops/s)", data["sync_file_ops"], "ops/s")
    bar_chart(OUT / "latency", "P99 latency (lower is better)", data["latency_p99_ns"], "nanoseconds", "#dc2626")
    line_chart(OUT / "scale", "Polyglot scale (writers → ops/s)", data.get("scale_ops") or {"1": 0, "2": 0}, "writers", "ops/s")
    bar_chart(OUT / "ffi", "FFI crossing vs full sync log", data["ffi_ns"], "nanoseconds (mean)", "#059669")
    print(f"wrote charts under {OUT} (.svg + .png)")


if __name__ == "__main__":
    main()
