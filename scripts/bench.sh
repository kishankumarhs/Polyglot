#!/usr/bin/env bash
# Full Polyglot benchmark suite.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
mkdir -p bench/results

echo "==> build native library"
bash scripts/build-native.sh dist

LIB=""
case "$(uname -s 2>/dev/null || echo Windows)" in
  MINGW*|MSYS*|CYGWIN*|Windows*) LIB="$ROOT/dist/logger.dll" ;;
  Darwin*) LIB="$ROOT/dist/liblogger.dylib" ;;
  *) LIB="$ROOT/dist/liblogger.so" ;;
esac
export POLYGLOT_LOGGER_LIB="$LIB"

OUT_GO="bench/results/go.txt"
OUT_NODE="bench/results/node.txt"
OUT_FFI="bench/results/ffi.txt"
OUT_MD="bench/results/latest.md"

echo "==> Go benches (zap / zerolog / polyglot)"
(
  cd bench/go
  go test -bench='BenchmarkPolyglotSyncFile|BenchmarkPolyglotAsyncFile|BenchmarkZapJSONFile|BenchmarkZerologFile|BenchmarkSlogJSONFile|BenchmarkPolyglotWithChild|BenchmarkZapWithChild|BenchmarkZerologWithChild|BenchmarkPolyglotMemoryAllocs' \
    -benchmem -count=3 -timeout 30m | tee "../../$OUT_GO"
  go test -count=1 -run 'TestOverflow|TestHotReload|TestMemory' -timeout 10m | tee -a "../../$OUT_GO"
  BENCH_SCALE_CSV=1 go test -count=1 -run 'TestScaleCSV' -timeout 10m | tee -a "../../$OUT_GO"
)

echo "==> Node file benches (+ Bun if present)"
# bench.mjs requires bindings/node, whose main is the compiled dist/index.js.
(
  cd bindings/node
  npm install --silent
  npm run build --silent
)
(
  cd bench/node
  npm install --silent
  node bench.mjs | tee "../../$OUT_NODE"
  node ffi_baseline.mjs | tee "../../$OUT_FFI"
  if command -v bun >/dev/null 2>&1; then
    echo "--- bun ---" | tee -a "../../$OUT_NODE"
    bun bench.mjs | tee -a "../../$OUT_NODE"
  fi
)

echo "==> Cross-language 100k (schema consistency)"
CROSS_OUT="bench/results/cross.txt"
: > "$CROSS_OUT"
BENCH_CROSS_N="${BENCH_CROSS_N:-10000}"
export BENCH_CROSS_N
go run ./bench/cross/run_go.go | tee -a "$CROSS_OUT"
node bench/cross/run.mjs | tee -a "$CROSS_OUT"
python bench/cross/run.py | tee -a "$CROSS_OUT" || echo "python cross skipped" | tee -a "$CROSS_OUT"
if command -v dotnet >/dev/null 2>&1; then
  (cd bench/cross/dotnet && dotnet clean -c Release --verbosity quiet 2>/dev/null; dotnet restore --verbosity quiet && dotnet run -c Release --verbosity quiet) | tee -a "$CROSS_OUT" || true
fi

{
  echo "# Polyglot bench results"
  echo
  echo "Generated: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "Native: \`$POLYGLOT_LOGGER_LIB\`"
  echo
  echo "## Go"
  echo '```'
  cat "$OUT_GO"
  echo '```'
  echo
  echo "## Node / FFI"
  echo '```'
  cat "$OUT_NODE"
  echo
  cat "$OUT_FFI"
  echo '```'
  echo
  echo "## Cross-language"
  echo '```'
  cat "$CROSS_OUT"
  echo '```'
  echo
  echo "See [bench/README.md](../README.md) for methodology."
} > "$OUT_MD"

echo "==> summarize + charts"
python scripts/bench-summarize.py
python scripts/bench-charts.py

echo
echo "Wrote $OUT_MD and bench/results/*.svg"
