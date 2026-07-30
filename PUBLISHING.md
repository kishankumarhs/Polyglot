# Publishing

Ship npm / PyPI / NuGet packages with native binaries bundled. Prefer the release workflow; manual steps are below if you need them.

Platforms: Linux x86_64, Windows x86_64, macOS arm64. See [compatibility.md](docs/compatibility.md).

## Automated

`.github/workflows/release.yml` builds, packs, creates a GitHub release, and publishes on a version tag.

Secrets (Settings → Actions): `NPM_TOKEN`, `PYPI_TOKEN`, `NUGET_TOKEN`.

Bump versions in:

- `bindings/node/package.json`
- `bindings/python/pyproject.toml`
- `bindings/dotnet/Polyglot.Logger/Polyglot.Logger.csproj`

```bash
git tag v0.3.0
git push origin v0.3.0
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
