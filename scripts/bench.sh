#!/usr/bin/env bash
# Full Polyglot benchmark suite.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
mkdir -p bench/results

PHASE_TOTAL=8
PHASE_NUM=0
PHASE_NAME=""
PHASE_T0=0
_hb_pid=""

# Line-buffer when possible; stdbuf is often missing on git-bash/Windows.
stream() {
  if command -v stdbuf >/dev/null 2>&1; then
    stdbuf -oL -eL "$@"
  else
    "$@"
  fi
}

_stop_heartbeat() {
  if [[ -n "${_hb_pid:-}" ]]; then
    kill "${_hb_pid}" 2>/dev/null || true
    wait "${_hb_pid}" 2>/dev/null || true
    _hb_pid=""
  fi
}

_start_heartbeat() {
  _stop_heartbeat
  local t0
  t0=$(date +%s)
  (
    trap 'kill "${sleep_pid:-}" 2>/dev/null || true; exit 0' TERM INT
    while true; do
      sleep 30 &
      sleep_pid=$!
      wait "$sleep_pid" || exit 0
      now=$(date +%s)
      echo "    ... still running ($((now - t0))s elapsed)"
    done
  ) &
  _hb_pid=$!
}

trap '_stop_heartbeat' EXIT INT TERM

phase_begin() {
  PHASE_NUM=$1
  PHASE_NAME=$2
  PHASE_T0=$(date +%s)
  echo "==> [${PHASE_NUM}/${PHASE_TOTAL}] ${PHASE_NAME}"
}

phase_end() {
  local elapsed=$(( $(date +%s) - PHASE_T0 ))
  echo "<== [${PHASE_NUM}/${PHASE_TOTAL}] ${PHASE_NAME} done in ${elapsed}s"
}

echo "Full suite takes ~10-25 min. Go benches are the longest phase (progress every ~30s)."
echo "Faster: make bench-smoke, or BENCH_QUICK=1 make bench for reduced counts."
echo

GO_COUNT=3
GO_BENCHTIME_ARGS=()
if [[ "${BENCH_QUICK:-}" == "1" ]]; then
  GO_COUNT=1
  GO_BENCHTIME_ARGS=(-benchtime=200ms)
  export BENCH_NODE_N="${BENCH_NODE_N:-2000}"
  export BENCH_NODE_ITERS="${BENCH_NODE_ITERS:-1}"
  export BENCH_FFI_N="${BENCH_FFI_N:-5000}"
  export BENCH_PY_N="${BENCH_PY_N:-2000}"
  export BENCH_PY_ITERS="${BENCH_PY_ITERS:-1}"
  export BENCH_DOTNET_N="${BENCH_DOTNET_N:-2000}"
  export BENCH_DOTNET_ITERS="${BENCH_DOTNET_ITERS:-1}"
  export BENCH_CROSS_N="${BENCH_CROSS_N:-1000}"
  export BENCH_RELOAD_WORKERS="${BENCH_RELOAD_WORKERS:-16}"
  export BENCH_OVERFLOW_N="${BENCH_OVERFLOW_N:-5000}"
  export BENCH_MEM_N="${BENCH_MEM_N:-10000}"
  echo "BENCH_QUICK=1: count=1, benchtime=200ms, smaller Node/Python/.NET/cross/reload N"
  echo
fi

OUT_GO="bench/results/go.txt"
OUT_NODE="bench/results/node.txt"
OUT_FFI="bench/results/ffi.txt"
OUT_MD="bench/results/latest.md"

phase_begin 1 "build native library"
bash scripts/build-native.sh dist
phase_end

LIB=""
case "$(uname -s 2>/dev/null || echo Windows)" in
  MINGW*|MSYS*|CYGWIN*|Windows*) LIB="$ROOT/dist/logger.dll" ;;
  Darwin*) LIB="$ROOT/dist/liblogger.dylib" ;;
  *) LIB="$ROOT/dist/liblogger.so" ;;
esac
export POLYGLOT_LOGGER_LIB="$LIB"

phase_begin 2 "Go benches (zap / zerolog / polyglot)"
_start_heartbeat
(
  cd bench/go
  # -run '^$' skips package tests; otherwise go test -bench still runs them first.
  stream go test -v -run '^$' \
    -bench='BenchmarkPolyglotSyncFile|BenchmarkPolyglotAsyncFile|BenchmarkZapJSONFile|BenchmarkZerologFile|BenchmarkSlogJSONFile|BenchmarkPolyglotWithChild|BenchmarkZapWithChild|BenchmarkZerologWithChild|BenchmarkPolyglotMemoryAllocs' \
    -benchmem -count="$GO_COUNT" "${GO_BENCHTIME_ARGS[@]}" -timeout 30m 2>&1 | tee "../../$OUT_GO"
  stream go test -v -count=1 -run 'TestOverflow|TestHotReload|TestMemory' -timeout 10m 2>&1 | tee -a "../../$OUT_GO"
)
_stop_heartbeat
phase_end

phase_begin 3 "Go scale CSV"
(
  cd bench/go
  BENCH_SCALE_CSV=1 stream go test -v -count=1 -run 'TestScaleCSV' -timeout 10m 2>&1 | tee -a "../../$OUT_GO"
)
phase_end

phase_begin 4 "Node file benches (+ Bun if present)"
# bench.mjs requires bindings/node, whose main is the compiled dist/index.js.
(
  cd bindings/node
  npm install --silent
  npm run build --silent
)
(
  cd bench/node
  npm install --silent
  stream node bench.mjs 2>&1 | tee "../../$OUT_NODE"
  stream node ffi_baseline.mjs 2>&1 | tee "../../$OUT_FFI"
  if command -v bun >/dev/null 2>&1; then
    echo "--- bun ---" | tee -a "../../$OUT_NODE"
    stream bun bench.mjs 2>&1 | tee -a "../../$OUT_NODE"
  fi
)
phase_end

phase_begin 5 "Python competitor (stdlib logging)"
OUT_PY="bench/results/python.txt"
: > "$OUT_PY"
stream python bench/python/bench.py 2>&1 | tee "$OUT_PY" || echo "python competitor skipped" | tee "$OUT_PY"
phase_end

phase_begin 6 ".NET competitor (Serilog)"
OUT_DN="bench/results/dotnet.txt"
: > "$OUT_DN"
if command -v dotnet >/dev/null 2>&1; then
  (cd bench/dotnet && dotnet restore --verbosity quiet && stream dotnet run -c Release --verbosity quiet) 2>&1 | tee "$OUT_DN" || echo "dotnet competitor skipped" | tee "$OUT_DN"
else
  echo "dotnet competitor skipped (no dotnet)" | tee "$OUT_DN"
fi
phase_end

phase_begin 7 "Cross-language 100k (schema consistency)"
CROSS_OUT="bench/results/cross.txt"
: > "$CROSS_OUT"
BENCH_CROSS_N="${BENCH_CROSS_N:-10000}"
export BENCH_CROSS_N
stream go run ./bench/cross/run_go.go 2>&1 | tee -a "$CROSS_OUT"
stream node bench/cross/run.mjs 2>&1 | tee -a "$CROSS_OUT"
stream python bench/cross/run.py 2>&1 | tee -a "$CROSS_OUT" || echo "python cross skipped" | tee -a "$CROSS_OUT"
if command -v dotnet >/dev/null 2>&1; then
  (cd bench/cross/dotnet && dotnet clean -c Release --verbosity quiet 2>/dev/null; dotnet restore --verbosity quiet && stream dotnet run -c Release --verbosity quiet) 2>&1 | tee -a "$CROSS_OUT" || true
fi
phase_end

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
  echo "## Python / .NET"
  echo '```'
  cat "$OUT_PY"
  echo
  cat "$OUT_DN"
  echo '```'
  echo
  echo "## Cross-language"
  echo '```'
  cat "$CROSS_OUT"
  echo '```'
  echo
  echo "See [bench/README.md](../README.md) for methodology."
} > "$OUT_MD"

phase_begin 8 "summarize + charts"
python scripts/bench-summarize.py
python scripts/bench-charts.py
phase_end

echo
echo "Wrote $OUT_MD and bench/results/*.svg"
