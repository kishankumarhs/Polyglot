# Publishing

Ship npm / PyPI / NuGet packages with native binaries bundled. Prefer the release workflow; manual steps are below if you need them.

Platforms: Linux x86_64, Windows x86_64, macOS arm64. See [compatibility.md](docs/compatibility.md).

## Automated

`.github/workflows/release.yml` builds, packs, creates a GitHub release, and publishes on a version tag.

Secrets (Settings → Actions): `NPM_TOKEN`, `PYPI_TOKEN`, `NUGET_TOKEN`.

**Versioning:** the git tag is source of truth. On release, CI runs `scripts/set-release-version.sh` so binding manifests and the Go `Version` const match the tag (`v0.3.1` → `0.3.1`) before `npm pack` / `python -m build` / `dotnet pack`. You do **not** need a prior commit of version bumps for publish to work.

This avoids republishing a stale submodule version (e.g. npm `403` when tagging a new release while `package.json` still said `0.3.0`).

```bash
git tag v0.3.1
git push origin v0.3.1
```

Optional local sync (docs / submodule pins):

```bash
bash scripts/set-release-version.sh 0.3.1
```

Install binding + native from the same tag.

## Manual

```bash
bash scripts/build-native.sh dist
# or scripts/package-native-libs.sh if that's what release uses
```

### Node

```bash
# stage natives into bindings/node/bin (or native/)
cd bindings/node && npm install && npm run build && npm publish
```

### Python

```bash
cd bindings/python && python -m build && twine upload dist/*
```

### .NET

```bash
cd bindings/dotnet/Polyglot.Logger
dotnet pack -c Release
dotnet nuget push bin/Release/*.nupkg -k "$NUGET_TOKEN" -s https://api.nuget.org/v3/index.json
```

## After publish

```bash
npm install @polyglot-logger/node
pip install polyglot-logger
dotnet add package Polyglot.Logger
```

Config discovery: [docs/zero-config.md](docs/zero-config.md). Checklist: [RELEASE-CHECKLIST.md](RELEASE-CHECKLIST.md).
