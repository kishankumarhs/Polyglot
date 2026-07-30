# Contributing

PRs welcome. Be kind — [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

## Setup

```bash
git clone --recurse-submodules https://github.com/YOUR-USERNAME/Polyglot.git
cd Polyglot
git remote add upstream https://github.com/kishankumarhs/Polyglot.git
git checkout -b feature/your-feature-name
make build-native
make test
```

Need Go 1.22+, a C toolchain with CGO, Make. Optional: Python 3.9+, Node 18+, .NET 8+. See [docs/build.md](docs/build.md).

Bindings are Git submodules — read [docs/SUBMODULE-WORKFLOW.md](docs/SUBMODULE-WORKFLOW.md) before changing them.

## Commits & style

- Imperative subject: `Add HTTP retry backoff`
- One logical change per commit
- Go / Python / TS / C# use each language's normal conventions
- `make lint` / `make test` before you push

## PRs

- Clear title and short summary of why
- How to test
- Don't break the C ABI without a version bump and notes
- Binding changes: push the binding repo first, then bump the submodule pointer here

## Where help is useful

- Core: queue, rotation, sinks, allocs
- Bindings: API ergonomics, errors, tests
- Docs and examples
- CI / packaging

## Issues

Search first. Include OS, language version, package version, and a minimal repro when you can.

Docs: [docs/](docs/) · packages: [polyglot-node](https://github.com/kishankumarhs/polyglot-node), [polyglot-py](https://github.com/kishankumarhs/polyglot-py), [polyglot-csharp](https://github.com/kishankumarhs/polyglot-csharp)
