#!/usr/bin/env bash
# Rewrite binding + core version strings for a release.
# Usage: set-release-version.sh <version>
# Accepts 0.3.1 or v0.3.1 (leading v stripped). Tag is source of truth in CI.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RAW="${1:?usage: set-release-version.sh <version>}"
VERSION="${RAW#v}"

if [[ ! "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+([.-][A-Za-z0-9.-]+)?$ ]]; then
  echo "error: version must be semver-like X.Y.Z or X.Y.Z-prerelease (got: $RAW)" >&2
  exit 1
fi

py() {
  if command -v python3 >/dev/null 2>&1; then
    python3 "$@"
  else
    python "$@"
  fi
}

sed_inplace() {
  local expr="$1"
  local file="$2"
  if sed --version >/dev/null 2>&1; then
    sed -i -E "$expr" "$file"
  else
    sed -i.bak -E "$expr" "$file"
    rm -f "${file}.bak"
  fi
}

echo "Setting release version to $VERSION"

# Node package.json (+ lockfile top-level / packages[""] if present)
NODE_PKG="$ROOT/bindings/node/package.json"
if [[ -f "$NODE_PKG" ]]; then
  VERSION="$VERSION" NODE_PKG="$NODE_PKG" py - <<'PY'
import json, os
from pathlib import Path

version = os.environ["VERSION"]
pkg_path = Path(os.environ["NODE_PKG"])
data = json.loads(pkg_path.read_text(encoding="utf-8"))
data["version"] = version
pkg_path.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")

lock_path = pkg_path.parent / "package-lock.json"
if lock_path.is_file():
    lock = json.loads(lock_path.read_text(encoding="utf-8"))
    lock["version"] = version
    packages = lock.get("packages")
    if isinstance(packages, dict) and "" in packages and isinstance(packages[""], dict):
        packages[""]["version"] = version
    lock_path.write_text(json.dumps(lock, indent=2) + "\n", encoding="utf-8")
PY
  echo "  updated bindings/node/package.json"
else
  echo "  skip bindings/node/package.json (missing — submodule not checked out?)" >&2
fi

# Python pyproject.toml
PY_TOML="$ROOT/bindings/python/pyproject.toml"
if [[ -f "$PY_TOML" ]]; then
  sed_inplace 's/^(version[[:space:]]*=[[:space:]]*")[^"]*(")/\1'"$VERSION"'\2/' "$PY_TOML"
  echo "  updated bindings/python/pyproject.toml"
else
  echo "  skip bindings/python/pyproject.toml (missing — submodule not checked out?)" >&2
fi

# .NET Version / PackageVersion
CSPROJ="$ROOT/bindings/dotnet/Polyglot.Logger/Polyglot.Logger.csproj"
if [[ ! -f "$CSPROJ" ]]; then
  # Older submodule layout
  for cand in "$ROOT"/bindings/dotnet/*/*.csproj; do
    if [[ -f "$cand" ]] && grep -q '<Version>' "$cand" 2>/dev/null; then
      CSPROJ="$cand"
      break
    fi
  done
fi
if [[ -f "$CSPROJ" ]]; then
  if grep -q '<Version>' "$CSPROJ"; then
    sed_inplace 's#(<Version>)[^<]*(</Version>)#\1'"$VERSION"'\2#' "$CSPROJ"
  else
    # Insert Version after first PropertyGroup open if somehow absent
    sed_inplace 's#(<PropertyGroup>)#\1\n    <Version>'"$VERSION"'</Version>#' "$CSPROJ"
  fi
  if grep -q '<PackageVersion>' "$CSPROJ"; then
    sed_inplace 's#(<PackageVersion>)[^<]*(</PackageVersion>)#\1'"$VERSION"'\2#' "$CSPROJ"
  fi
  echo "  updated $CSPROJ"
else
  echo "  skip .NET csproj (missing — submodule not checked out?)" >&2
fi

# Go core Version (reported by logger_version)
GO_FILE="$ROOT/internal/logger/logger.go"
if [[ -f "$GO_FILE" ]]; then
  sed_inplace 's/^([[:space:]]*Version[[:space:]]*=[[:space:]]*")[^"]*(")/\1'"$VERSION"'\2/' "$GO_FILE"
  echo "  updated internal/logger/logger.go"
else
  echo "  skip internal/logger/logger.go (missing)" >&2
fi

echo "Release version $VERSION applied."
