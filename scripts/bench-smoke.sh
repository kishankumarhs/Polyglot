#!/usr/bin/env bash
# Short CI smoke: compile + tiny runtimes (no absolute ns/op gates).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
mkdir -p bench/results

PHASE_TOTAL=2
PHASE_NUM=0
PHASE_NAME=""
PHASE_T0=0

# Line-buffer when possible; stdbuf is often missing on git-bash/Windows.
stream() {
  if command -v stdbuf >/dev/null 2>&1; then
    stdbuf -oL -eL "$@"
  else
    "$@"
  fi
}

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

echo "bench-smoke: short compile + tiny runtimes (seconds, not minutes)."
echo

phase_begin 1 "Go smoke benches"
(
  cd bench/go
  stream go test -c -o /tmp/polyglot-bench.test 2>&1
  stream go test -v -run '^$' -bench='BenchmarkPolyglotSyncFile|BenchmarkZapJSONFile' -benchtime=50ms -benchmem -count=1 2>&1
  BENCH_OVERFLOW_N=5000 BENCH_MEM_N=2000 BENCH_RELOAD_WORKERS=8 \
    stream go test -v -count=1 -run 'TestOverflow|TestHotReload|TestMemory' -timeout 60s 2>&1
)
phase_end

phase_begin 2 "bindings smoke (if native lib present)"
if [[ -f dist/logger.dll || -f dist/liblogger.so || -f dist/liblogger.dylib ]]; then
  case "$(uname -s 2>/dev/null || echo Windows)" in
    MINGW*|MSYS*|CYGWIN*|Windows*) export POLYGLOT_LOGGER_LIB="$ROOT/dist/logger.dll" ;;
    Darwin*) export POLYGLOT_LOGGER_LIB="$ROOT/dist/liblogger.dylib" ;;
    *) export POLYGLOT_LOGGER_LIB="$ROOT/dist/liblogger.so" ;;
  esac
  # bench.mjs requires bindings/node, whose main is the compiled dist/index.js.
  (
    cd bindings/node
    npm install --silent
    npm run build --silent
  )
  (
    cd bench/node
    npm install --silent
    BENCH_NODE_N=500 BENCH_NODE_ITERS=1 stream node bench.mjs >/dev/null
    BENCH_FFI_N=2000 stream node ffi_baseline.mjs >/dev/null
  )
  BENCH_PY_N=500 BENCH_PY_ITERS=1 stream python bench/python/bench.py >/dev/null || echo "bench-smoke: python competitor skipped"
  if command -v dotnet >/dev/null 2>&1; then
    (
      cd bench/dotnet
      BENCH_DOTNET_N=500 BENCH_DOTNET_ITERS=1 stream dotnet run -c Release --verbosity quiet >/dev/null
    ) || echo "bench-smoke: dotnet competitor skipped"
  fi
  echo "bench-smoke: ok"
else
  echo "bench-smoke: Go ok; skipping Node (no dist native lib)"
fi
phase_end
