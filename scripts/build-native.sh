#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${1:-$ROOT/dist}"
mkdir -p "$OUT_DIR"

cd "$ROOT"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$OS" in
  linux*)
    LIB_NAME="liblogger.so"
    ;;
  darwin*)
    LIB_NAME="liblogger.dylib"
    ;;
  msys*|mingw*|cygwin*)
    LIB_NAME="logger.dll"
    ;;
  *)
    echo "unsupported OS: $OS" >&2
    exit 1
    ;;
esac

# Prefer Scoop MinGW on Windows when present.
if [[ "$OS" == msys* || "$OS" == mingw* || "$OS" == cygwin* ]]; then
  MINGW_BIN="${MINGW_BIN:-/c/Users/Kishan/scoop/apps/mingw/current/bin}"
  if [[ -d "$MINGW_BIN" ]]; then
    export PATH="$MINGW_BIN:$PATH"
  fi
fi

export CGO_ENABLED=1

HEADER_SRC="$ROOT/native/include/logger.h"
if [[ ! -f "$HEADER_SRC" ]]; then
  echo "missing stable header: $HEADER_SRC" >&2
  exit 1
fi

echo "Running ABI codegen"
go run ./cmd/codegen

echo "Building native shared library -> $OUT_DIR/$LIB_NAME"
go build -buildmode=c-shared -o "$OUT_DIR/$LIB_NAME" ./native

# Go emits a generated header next to the shared library; keep only our stable ABI header.
rm -f "$OUT_DIR/liblogger.h" "$OUT_DIR/logger.h"
cp "$HEADER_SRC" "$OUT_DIR/logger.h"

(
  cd "$OUT_DIR"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$LIB_NAME" logger.h > checksums.sha256
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$LIB_NAME" logger.h > checksums.sha256
  fi
)

# Stage copies for language bindings convenience.
mkdir -p "$ROOT/bindings/python/polyglot_logger/native"
mkdir -p "$ROOT/bindings/node/native"
mkdir -p "$ROOT/bindings/dotnet/Polyglot.Logger/native"
cp "$OUT_DIR/$LIB_NAME" "$ROOT/bindings/python/polyglot_logger/native/"
cp "$OUT_DIR/$LIB_NAME" "$ROOT/bindings/node/native/"
cp "$OUT_DIR/$LIB_NAME" "$ROOT/bindings/dotnet/Polyglot.Logger/native/"

echo "Built $LIB_NAME for $OS/$ARCH"
