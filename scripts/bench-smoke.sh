#!/usr/bin/env bash
# Short CI smoke: compile + tiny runtimes (no absolute ns/op gates).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
mkdir -p bench/results

(
  cd bench/go
  go test -c -o /tmp/polyglot-bench.test
  go test -bench='BenchmarkPolyglotSyncFile|BenchmarkZapJSONFile' -benchtime=50ms -benchmem -count=1
  BENCH_OVERFLOW_N=5000 BENCH_MEM_N=2000 go test -count=1 -run 'TestOverflow|TestHotReload|TestMemory' -timeout 60s
)

if [[ -f dist/logger.dll || -f dist/liblogger.so || -f dist/liblogger.dylib ]]; then
  case "$(uname -s 2>/dev/null || echo Windows)" in
    MINGW*|MSYS*|CYGWIN*|Windows*) export POLYGLOT_LOGGER_LIB="$ROOT/dist/logger.dll" ;;
    Darwin*) export POLYGLOT_LOGGER_LIB="$ROOT/dist/liblogger.dylib" ;;
    *) export POLYGLOT_LOGGER_LIB="$ROOT/dist/liblogger.so" ;;
  esac
  (
    cd bench/node
    npm install --silent
    BENCH_NODE_N=500 BENCH_NODE_ITERS=1 node bench.mjs >/dev/null
    BENCH_FFI_N=2000 node ffi_baseline.mjs >/dev/null
  )
  echo "bench-smoke: ok"
else
  echo "bench-smoke: Go ok; skipping Node (no dist native lib)"
fi
