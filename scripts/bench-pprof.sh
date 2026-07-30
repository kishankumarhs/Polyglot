#!/usr/bin/env bash
# Capture CPU/heap profiles for Polyglot sync+async benches.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT/bench/go"
mkdir -p ../results

go test -bench=BenchmarkPolyglotSyncFile -benchtime=1s -cpuprofile=../results/cpu-sync.pprof -memprofile=../results/mem-sync.pprof
go test -bench=BenchmarkPolyglotAsyncFile -benchtime=1s -cpuprofile=../results/cpu-async.pprof -memprofile=../results/mem-async.pprof

if command -v go >/dev/null; then
  for kind in sync async; do
    if go tool pprof -svg ../results/cpu-$kind.pprof > ../results/cpu-$kind.svg 2>/dev/null; then
      echo "wrote bench/results/cpu-$kind.svg"
    else
      echo "pprof svg unavailable (install graphviz); kept .pprof files"
    fi
  done
fi
