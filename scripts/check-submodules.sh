#!/usr/bin/env bash
# Fail fast when binding submodules were not initialized after clone.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

missing=0

check() {
  local path="$1"
  local hint="$2"
  if [[ ! -e "$path" ]]; then
    echo "MISSING: $path ($hint)" >&2
    missing=1
  fi
}

check "bindings/node/package.json" "node submodule — run: git submodule update --init --recursive"
check "bindings/python/pyproject.toml" "python submodule — run: git submodule update --init --recursive"
# .NET project file name may vary slightly across submodule revisions.
if [[ ! -f bindings/dotnet/Polyglot.Logger/Polyglot.Logger.csproj ]] && \
   [[ ! -f bindings/dotnet/Eximietas.Logger/Eximietas.Logger.csproj ]]; then
  # Accept any *.csproj under bindings/dotnet as present.
  if ! compgen -G "bindings/dotnet/**/*.csproj" > /dev/null 2>&1; then
    echo "MISSING: bindings/dotnet/*.csproj (dotnet submodule — run: git submodule update --init --recursive)" >&2
    missing=1
  fi
fi

if [[ "$missing" -ne 0 ]]; then
  echo >&2
  echo "Clone with submodules:" >&2
  echo "  git clone --recurse-submodules <repo-url>" >&2
  echo "Or from an existing clone:" >&2
  echo "  git submodule update --init --recursive" >&2
  exit 1
fi

echo "submodules: OK"
