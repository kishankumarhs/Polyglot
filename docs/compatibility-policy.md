# Compatibility & ABI policy

Bindings and the native core both carry versions. Library semver (`0.3.0`) can move while ABI stays at `1`.

| Kind | Example | Meaning |
| --- | --- | --- |
| Library version | `logger_version()` → `0.3.0` | Behavior, sinks, bugfixes |
| ABI version | `logger_abi_version()` → `1` | C surface compatibility |

Bump ABI only when the C API breaks. Prefer additive APIs (`logger_with`, new optional config keys).

## 1.x compatibility guarantees (target)

Once `v1.0.0` ships, platform teams can rely on these within the **1.x** line. Until then (0.3.x), treat them as intent — pin patch versions and upgrade core + bindings together.

| Guarantee | Promise |
| --- | --- |
| **`polyglot.yaml` / create JSON** | Stable within 1.x: existing keys keep meaning; new keys are additive. Unknown keys ignored unless `strict: true`. Breaking renames require 2.0. |
| **C ABI** | ABI major stays at the published 1.x ABI (today `1`) for the life of 1.x. Additive exports only; removals/signature breaks bump ABI and package major. |
| **Log JSON field names** | Default line shape (`timestamp`, `level`, `message`, `service_name`, …) stays backward-compatible. New optional fields may appear; renames/removals are major. |
| **Semver (core + wrappers)** | Same release tag for Go core and Node / Python / .NET packages. PATCH = fixes; MINOR = additive APIs/config; MAJOR = public API or ABI break. |

These reduce upgrade risk for platform rollouts — often as important as new sinks.

## Startup behavior

**0.3.x today:** bindings load the shared lib from package `bin/` / `native/` (or `POLYGLOT_LOGGER_LIB`). Smoke tests assert ABI `== 1`, but a mismatch is not refused at load — wrong symbols usually fail on first call.

**1.0 target:** each binding embeds `EXPECTED_ABI = 1`, calls `logger_abi_version()` on first load, and throws if it disagrees. Optional escape hatch: `POLYLOG_ALLOW_ABI_MISMATCH=1` for experts. Until then, install **0.3.x** packages only.

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
