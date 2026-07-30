#!/usr/bin/env bash
# Organize downloaded release artifacts into binding package layouts.
# Usage: package-native-libs.sh <artifacts-root> <dest-root>
# Dest gets both bin/ and native/ (platform-arch dirs + flat / arch-qualified macOS).
set -euo pipefail

ARTIFACTS="${1:?artifacts root}"
DEST="${2:?destination root (e.g. bindings/node)}"

mkdir -p "$DEST/bin" "$DEST/native"

copy_lib() {
  local src="$1"
  local dest="$2"
  if [[ -f "$src" ]]; then
    mkdir -p "$(dirname "$dest")"
    cp "$src" "$dest"
  fi
}

# Artifact dirs look like: artifacts/native-linux-x86_64/liblogger.so
shopt -s nullglob
for art in "$ARTIFACTS"/native-*; do
  [[ -d "$art" ]] || continue
  name="$(basename "$art")"
  plat="${name#native-}" # linux-x86_64 | windows-x86_64 | macos-x86_64 | macos-arm-arm64
  for lib in "$art"/*.{so,dll,dylib}; do
    [[ -f "$lib" ]] || continue
    base="$(basename "$lib")"
    copy_lib "$lib" "$DEST/bin/$plat/$base"
    copy_lib "$lib" "$DEST/native/$plat/$base"
  done
done

# Flat copies for simple loaders (macOS uses arch-qualified names to avoid overwrite).
copy_lib "$ARTIFACTS/native-linux-x86_64/liblogger.so" "$DEST/bin/liblogger.so"
copy_lib "$ARTIFACTS/native-linux-x86_64/liblogger.so" "$DEST/native/liblogger.so"
copy_lib "$ARTIFACTS/native-windows-x86_64/logger.dll" "$DEST/bin/logger.dll"
copy_lib "$ARTIFACTS/native-windows-x86_64/logger.dll" "$DEST/native/logger.dll"

if [[ -f "$ARTIFACTS/native-macos-arm-arm64/liblogger.dylib" ]]; then
  copy_lib "$ARTIFACTS/native-macos-arm-arm64/liblogger.dylib" "$DEST/bin/liblogger.arm64.dylib"
  copy_lib "$ARTIFACTS/native-macos-arm-arm64/liblogger.dylib" "$DEST/native/liblogger.arm64.dylib"
  copy_lib "$ARTIFACTS/native-macos-arm-arm64/liblogger.dylib" "$DEST/bin/liblogger.dylib"
  copy_lib "$ARTIFACTS/native-macos-arm-arm64/liblogger.dylib" "$DEST/native/liblogger.dylib"
fi
if [[ -f "$ARTIFACTS/native-macos-x86_64/liblogger.dylib" ]]; then
  copy_lib "$ARTIFACTS/native-macos-x86_64/liblogger.dylib" "$DEST/bin/liblogger.x86_64.dylib"
  copy_lib "$ARTIFACTS/native-macos-x86_64/liblogger.dylib" "$DEST/native/liblogger.x86_64.dylib"
  if [[ ! -f "$DEST/bin/liblogger.dylib" ]]; then
    copy_lib "$ARTIFACTS/native-macos-x86_64/liblogger.dylib" "$DEST/bin/liblogger.dylib"
    copy_lib "$ARTIFACTS/native-macos-x86_64/liblogger.dylib" "$DEST/native/liblogger.dylib"
  fi
fi

echo "Packaged native libs into $DEST:"
find "$DEST/bin" "$DEST/native" -type f 2>/dev/null | sort || true
