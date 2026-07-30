# Compatibility matrix

Supported combinations for Polyglot 0.3.x (ABI v1). Install binding and native from the same release tag — mixing 0.3 with 0.2 is unsupported.

## Packages ↔ core

| Component | Package / artifact | Version | Requires ABI | Native lib |
| --- | --- | --- | --- | --- |
| Core runtime | `logger.dll` / `liblogger.so` / `liblogger.dylib` | 0.3.x | 1 | — |
| Node | `@polyglot-logger/node` | 0.3.x | 1 | bundled (`bin/` + `native/`) |
| Python | `polyglot-logger` | 0.3.x | 1 | bundled in wheel |
| .NET | `Polyglot.Logger` | 0.3.x | 1 | bundled in NuGet |
| Go (in-repo) | `polyglot/internal/logger` | 0.3.x | n/a | optional for demos |

## Platforms

| OS | Arch | Status (0.3.0) |
| --- | --- | --- |
| Linux | x86_64 | Supported (published) |
| Windows | x86_64 | Supported (published) |
| macOS | arm64 (Apple Silicon) | Supported (published) |
| macOS | x86_64 (Intel) | Limited — not in CI right now; use `POLYGLOT_LOGGER_LIB` or Rosetta carefully |

## Language runtimes

| Binding | Runtime | Status |
| --- | --- | --- |
| Node | Node.js 18+ | Supported |
| Node | Bun (recent) | Best-effort (same package; benches run when `bun` present) |
| Python | 3.9+ | Supported |
| .NET | net8.0 | Supported |

## Config / schema

| Artifact | Versioning |
| --- | --- |
| `polyglot.yaml` / JSON create config | Additive within ABI v1; unknown fields ignored unless `strict: true` |
| Log JSON schema (`service_name`, `level`, `fields`, …) | Stable for consumers; new optional keys may appear |
| Stats JSON | Documented in `api/abi.json` / user guide; additive fields OK |

## ABI negotiation

Full policy: [compatibility-policy.md](compatibility-policy.md).

| Scenario | Today (0.3) | Target 1.0 |
| --- | --- | --- |
| Binding expects ABI 1, lib returns 1 | Works | Works |
| Binding expects ABI 1, lib returns 2+ | Not refused — may misbehave | Fail fast |
| Binding too new for lib | Fails at FFI call | Fail fast at startup |

Check at runtime:

```js
import { abiVersion, libraryVersion } from "@polyglot-logger/node";
console.log(libraryVersion(), abiVersion()); // expect abi === 1
```

```python
from polyglot_logger import abi_version, library_version
assert abi_version() == 1
```

## How we release

One GitHub release tag (e.g. `v0.3.0`) publishes npm + PyPI + NuGet + native artifacts together. Prefer installing binding and native from the **same** tag / version number.
