# v1.0 checklist

Ship `v1.0.0` when these are done with evidence (docs, tests, or release artifacts). Freeze API/ABI and prove the shared ops model first; add the observability sinks below in priority order.

## Enterprise integrations (priority order)

Platform differentiator work before / into 1.0. Do these in order unless a customer blocks on a later item.

1. [ ] **OpenTelemetry context (flagship)** — automatic `trace_id` / `span_id` injection from active span (no manual field passing) in Go + bindings
2. [ ] **OTLP sink** — companion to (1); export logs via OTLP (HTTP and/or gRPC); reserved today in [`sink.go`](internal/logger/sink.go)
3. [ ] **Kafka sink** — native produce for centralized log pipelines
4. [ ] **Crash durability** — trade-off documented (async vs sync); optional stronger sync/`fsync` path remains open (async stays default)
5. [x] **File compression** — `compress: true` gzips rotated backups (implemented); add focused tests + keep docs honest

Post-1.0 / later: syslog, CloudWatch, lock-free ring buffer, zero-copy JSON.

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
- [x] Semver + 1.x guarantees documented in [compatibility-policy.md](docs/compatibility-policy.md) (enforce fail-fast ABI at 1.0)

## Proof & ops

- [ ] Benchmarks reproducible (`make bench`); charts not placeholders
- [ ] Overflow / reload / FFI / cross-lang results summarized
- [x] `polyglot doctor` / `validate` in first-run docs
- [ ] Panic recover on `//export` paths covered by tests
- [ ] File rotation + `compress` covered by tests

## Documentation

- [x] Root README covers install + platform positioning
- [x] [First log](docs/first-log.md) for Node / Python / .NET
- [x] Migration guides: Pino, Zap, Serilog, Python logging
- [ ] Upgrade guide (0.x → 1.0)
- [x] Compatibility matrix for supported OS/arch
- [x] Durability / async vs sync trade-off in [user guide](docs/user-guide.md) + [troubleshooting](docs/troubleshooting.md)
- [x] 1.x compatibility guarantees (`polyglot.yaml`, ABI, JSON fields, semver) in [compatibility-policy.md](docs/compatibility-policy.md)

## Examples & validation

- [ ] Production-shaped examples: Express, FastAPI, ASP.NET, Gin
- [ ] At least one real service dogfooding Polyglot (write-up)
- [ ] Timed UX study with ~10 experienced developers
- [ ] External feedback pass (“Why wouldn’t you use this?”) → FAQ

## Process

- [ ] CODEOWNERS / release checklist for 1.0 tag
- [ ] Security / vulnerability reporting noted
- [ ] Changelog for 1.0 with breaking changes empty or explicit
