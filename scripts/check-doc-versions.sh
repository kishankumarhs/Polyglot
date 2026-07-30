#!/usr/bin/env bash
# Fail if docs/READMEs pin polyglot packages at 1.x while the project is still 0.x.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# Current library line from Go source of truth
LINE="$(grep -E '^\s*Version\s*=' internal/logger/logger.go | head -1 | sed -E 's/.*"([^"]+)".*/\1/')"
MAJOR="${LINE%%.*}"

if [[ "$MAJOR" != "0" ]]; then
  echo "check-doc-versions: library is $LINE (major=$MAJOR); skipping 0.x pin guard"
  exit 0
fi

# Paths that are product docs (exclude node_modules, vendor, bench deps)
mapfile -t FILES < <(git ls-files '*.md' ':!:bench/**' 2>/dev/null || find . -name '*.md' -not -path './bench/*' -not -path './.git/*')

BAD=0
PATTERNS=(
  '@polyglot-logger/node@\^1'
  '@polyglot-logger/node@1\.'
  'polyglot-logger==1\.'
  'polyglot-logger~=1\.'
  'Polyglot\.Logger" Version="1\.'
  'dotnet add package Polyglot\.Logger --version 1\.'
)

for f in "${FILES[@]}"; do
  [[ -f "$f" ]] || continue
  for pat in "${PATTERNS[@]}"; do
    if grep -nE "$pat" "$f" >/dev/null 2>&1; then
      echo "error: $f pins a 1.x install while library is still $LINE"
      grep -nE "$pat" "$f" || true
      BAD=1
    fi
  done
done

if [[ "$BAD" -ne 0 ]]; then
  echo "Use 0.3.x (or unpinned) install snippets until v1.0.0 is published."
  exit 1
fi

echo "check-doc-versions: OK (library $LINE, no fake 1.x install pins)"
