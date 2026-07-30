# Release checklist

## Cut a release

Tag is the source of truth. CI rewrites binding + core versions from the tag before pack/publish (no manual version commit required for publish to succeed).

```bash
# on latest main, after tests/docs/CHANGELOG are ready
git tag v0.3.1
git push origin v0.3.1
```

Tag must be semver-like: `vX.Y.Z` or `vX.Y.Z-prerelease`. Pushing `v0.3.0` again after that version is already on npm fails (npm refuses republish; CI also checks).

Optional: bump in-repo versions for local consistency (`scripts/set-release-version.sh 0.3.1` and commit in main + binding submodules). Not required for CI publish.

## Before tag

- [ ] On latest main
- [ ] `make test` + binding tests green
- [ ] CHANGELOG updated
- [ ] Docs match new options

## After Actions finishes

- [ ] npm / PyPI / NuGet show the **tag** version (not a stale 0.3.0)
- [ ] Smoke install on a clean machine (or CI)
- [ ] GitHub release notes look right

Details: [PUBLISHING.md](PUBLISHING.md).
