# v1.0 checklist

Ship `v1.0.0` when these are done with evidence (docs, tests, or release artifacts). Prefer validation and docs over new sinks until then.

## API & ABI

- [ ] Public binding APIs frozen for 1.0 (Node / Python / .NET)
- [ ] C ABI v1 frozen; policy in [compatibility-policy.md](docs/compatibility-policy.md)
- [ ] Bindings fail fast on ABI mismatch at load
- [ ] `logger_with` / child-logger semantics documented and tested in all bindings
- [ ] Config schema documented; `strict` mode tested

## Packages & platforms

- [ ] npm / PyPI / NuGet install works without Go/CGO on Linux, Windows, macOS arm64 (Intel documented)
- [ ] Same release tag publishes all registries + native artifacts
- [ ] [Compatibility matrix](docs/compatibility.md) linked from README
- [ ] Semver policy for core + bindings documented

## Proof & ops

- [ ] Benchmarks reproducible (`make bench`); charts not placeholders
- [ ] Overflow / reload / FFI / cross-lang results summarized
- [ ] `polyglot doctor` / `validate` in first-run docs
- [ ] Panic recover on `//export` paths covered by tests

## Documentation

- [x] Root README covers install + why
- [x] [First log](docs/first-log.md) for Node / Python / .NET
- [x] Migration guides: Pino, Zap, Serilog, Python logging
- [ ] Upgrade guide (0.x → 1.0)
- [x] Compatibility matrix for supported OS/arch

## Examples & validation

- [ ] Production-shaped examples: Express, FastAPI, ASP.NET, Gin
- [ ] At least one real service dogfooding Polyglot (write-up)
- [ ] Timed UX study with ~10 experienced developers
- [ ] External feedback pass (“Why wouldn’t you use this?”) → FAQ

## Process

- [ ] CODEOWNERS / release checklist for 1.0 tag
- [ ] Security / vulnerability reporting noted
- [ ] Changelog for 1.0 with breaking changes empty or explicit
