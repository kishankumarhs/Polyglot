# Build and packaging

## Requirements

- Go 1.22+ (CI uses 1.25.x)
- CGO enabled (`CGO_ENABLED=1`)
- C compiler on `PATH` (gcc, clang, or MinGW on Windows)

## Make targets

```bash
make codegen        # regenerate header + FFI from api/abi.json
make build-native   # codegen + build shared library into dist/
make test           # go test ./internal/logger
make test-race      # race-enabled tests (needs CGO)
make demo           # go run ./cmd/logger-demo
make clean          # remove dist/build and staged native copies
```

## Scripts

```bash
bash scripts/build-native.sh dist
# Windows PowerShell alternative:
#   ./scripts/build-native.ps1
```

The Unix script:

1. Runs `go run ./cmd/codegen`
2. Builds `./native` with `-buildmode=c-shared`
3. Writes the shared library, header, and checksums under the output directory
4. Stages copies into binding `native/` folders when present

## Artifacts

| Platform | Library name |
| -------- | ------------ |
| Linux | `liblogger.so` |
| Windows | `logger.dll` |
| macOS | `liblogger.dylib` |

Also: `logger.h`, `checksums.sha256`.

## Locating the library at runtime

Bindings search, in order:

1. `EXIMIETAS_LOGGER_LIB` (absolute path — recommended in CI and deploys)
2. Package-local `native/<lib>`
3. Repo `dist/` / `build/` (when running from the monorepo)
4. Current working directory

Libraries load **lazily on first use**, so importing a binding package does not fail if the native file is missing until you create a logger.

## CI

[`.github/workflows/build.yml`](../.github/workflows/build.yml) runs:

- `gofmt`, `go vet`, `go test ./... -race`
- Codegen drift check (`go run ./cmd/codegen` then `git diff --exit-code` on generated files)
- Native builds on Linux, Windows, macOS
- Python / Node / .NET smoke tests against a freshly built library

## Editor (Windows / gopls)

If `gopls` reports “No packages found” for `native/export.go`, ensure the workspace sets `CGO_ENABLED=1` and a MinGW `PATH`. See [`.vscode/settings.json`](../.vscode/settings.json).

## Versioning

| Component | Version field |
| --------- | ------------- |
| Go core | `internal/logger.Version` (`0.2.0`) |
| ABI | `logger_abi_version()` → `1` |
| Python | `bindings/python/pyproject.toml` |
| Node | `bindings/node/package.json` |
| .NET | `Eximietas.Logger.csproj` |

Bump binding package versions together with the core when shipping a release consumers will install.
