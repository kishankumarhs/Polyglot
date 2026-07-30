# Release checklist

## Before tag

- [ ] On latest main
- [ ] `make test` + binding tests green
- [ ] CHANGELOG updated
- [ ] Docs match new options
- [ ] Same version in node `package.json`, python `pyproject.toml`, dotnet `.csproj`

```bash
grep '"version"' bindings/node/package.json
grep '^version' bindings/python/pyproject.toml
grep '<Version>' bindings/dotnet/Polyglot.Logger/Polyglot.Logger.csproj
```

## Tag

```bash
git tag v0.3.0
git push origin v0.3.0
```

## After Actions finishes

- [ ] npm / PyPI / NuGet show the new version
- [ ] Smoke install on a clean machine (or CI)
- [ ] GitHub release notes look right

Details: [PUBLISHING.md](PUBLISHING.md).
