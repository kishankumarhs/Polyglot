# Compatibility & ABI policy

Bindings and the native core both carry versions. Library semver (`0.3.0`) can move while ABI stays at `1`.

| Kind | Example | Meaning |
| --- | --- | --- |
| Library version | `logger_version()` → `0.3.0` | Behavior, sinks, bugfixes |
| ABI version | `logger_abi_version()` → `1` | C surface compatibility |

Bump ABI only when the C API breaks. Prefer additive APIs (`logger_with`, new optional config keys).

## Startup behavior

**0.3.x today:** bindings load the shared lib from package `bin/` / `native/` (or `POLYGLOT_LOGGER_LIB`). Smoke tests assert ABI `== 1`, but a mismatch is not refused at load — wrong symbols usually fail on first call.

**Future 1.0 target (not published yet):** each binding embeds `EXPECTED_ABI = 1`, calls `logger_abi_version()` on first load, and throws if it disagrees. Optional escape hatch: `POLYLOG_ALLOW_ABI_MISMATCH=1` for experts. Until then, install **0.3.x** packages only.

## Binding semver

| Change | Bump |
| --- | --- |
| Bugfix, docs, no ABI change | PATCH |
| New ergonomic API on same ABI | MINOR |
| Needs newer ABI or breaks public API | MAJOR |

A GitHub release tag ships core + bindings at the same version (e.g. all `0.3.0`). Upgrade them together. Downgrading native under a newer binding is unsupported.

## Tests

- Smoke tests: `abi_version() == 1`
- Codegen fails if `api/abi.json` and `native/export.go` disagree
- CI builds native per platform into npm / PyPI / NuGet

Still open for 1.0: automated “wrong ABI refuses to start” test; mixed N / N-1 package matrix.

Matrix of packages and OS: [compatibility.md](compatibility.md).
