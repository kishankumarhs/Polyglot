# Release Checklist - Polyglot Logger

Use this checklist when releasing a new version of Polyglot Logger to npm, PyPI, and NuGet.

## Pre-Release (1-2 days before)

### Code Preparation

- [ ] Pull latest from main/develop branch
- [ ] Run all tests
  ```bash
  make test  # or individual tests
  npm test --workspace=bindings/node
  pytest bindings/python/tests/
  dotnet test bindings/dotnet/
  ```
- [ ] Code review of changes
- [ ] Update CHANGELOG.md with new features/fixes

### Documentation Updates

- [ ] Update README.md with new features (if applicable)
- [ ] Update docs/ with new configuration options (if applicable)
- [ ] Review PUBLISHING.md for clarity

## Version Bumping

### Node.js

**File:** `bindings/node/package.json`

```json
{
  "version": "0.2.0" // Update this
}
```

### Python

**File:** `bindings/python/pyproject.toml`

```toml
version = "0.2.0"  # Update this
```

### .NET

**File:** `bindings/dotnet/Polyglot.Logger/Polyglot.Logger.csproj`

```xml
<Version>0.2.0</Version>  <!-- Update this -->
```

### Go Core (optional)

**File:** `api/abi.json`

- Consider updating `abi_version` if ABI has changed
- Document breaking changes in CHANGELOG

### Verification

```bash
# Verify all versions are synchronized
grep "version" bindings/node/package.json
grep "^version" bindings/python/pyproject.toml
grep "<Version>" bindings/dotnet/Polyglot.Logger/Polyglot.Logger.csproj
```

## Pre-Commit Testing

### Local Build & Package Test

```bash
# Build native libraries
./scripts/build-native.sh dist/

# Test Node.js build
cd bindings/node
npm install
npm run build
npm pack
npm install polyglot-logger-*.tgz --save-dev
npm test

# Test Python build
cd ../python
python -m pip install -e .
pytest
python -m build

# Test .NET build
cd ../dotnet/Polyglot.Logger
dotnet build -c Release
dotnet pack -c Release
```

### Manual Integration Test

```bash
# Create test project
mkdir -p /tmp/test-polyglot
cd /tmp/test-polyglot

# Create polyglot.yaml
cat > polyglot.yaml << EOF
service: test-app
logging:
  level: info
EOF

# Test Node.js
npm init -y
npm install ../../path/to/Polyglot/bindings/node/polyglot-logger-*.tgz
node -e "const {Logger} = require('@polyglot/logger'); console.log('✓ Node.js works')"

# Test Python
pip install ../../path/to/Polyglot/bindings/python/dist/polyglot-logger-*.whl
python -c "from polyglot_logger import Logger; print('✓ Python works')"
```

## GitHub Setup (One-Time)

### Add Secrets

**Location:** GitHub → Settings → Secrets and variables → Actions

1. **NPM_TOKEN**
   - Generate at: https://www.npmjs.com/settings/tokens
   - Scope: Publish
   - Add as secret: `NPM_TOKEN`

2. **PYPI_TOKEN**
   - Generate at: https://pypi.org/manage/account/tokens/
   - Add as secret: `PYPI_TOKEN`

3. **NUGET_TOKEN**
   - Generate at: https://www.nuget.org/account (Account Settings → API Keys)
   - Scope: Push new packages
   - Add as secret: `NUGET_TOKEN`

### Verify Workflow File

- [ ] `.github/workflows/release.yml` exists
- [ ] All job names are correct
- [ ] Environment variable names match secret names

## Release Day

### Commit & Tag

```bash
# Stage all changes
git add .

# Commit with version message
git commit -m "chore: release v0.2.0"

# Create annotated tag (recommended)
git tag -a v0.2.0 -m "Release v0.2.0 - Add [feature summary]"

# Push commits
git push origin main

# Push tag (this triggers GitHub Actions!)
git push origin v0.2.0
```

### Monitor Build

- [ ] Go to GitHub → Actions tab
- [ ] Find "Build & Release Polyglot Logger" workflow
- [ ] Watch progress:
  - `build-native` (5-15 min) - Compiles for all platforms
  - `package-npm`, `package-python`, `package-dotnet` (2-5 min each)
  - `create-release` (1-2 min) - Creates GitHub Release
  - `publish-npm`, `publish-pypi`, `publish-nuget` (1-3 min each)
- [ ] All jobs should complete with ✅ green checkmarks

### Post-Release Verification (Wait 5-10 minutes for registry sync)

#### npm

```bash
npm view @polyglot/logger@0.2.0
npm install @polyglot/logger@0.2.0 --dry-run
```

#### PyPI

```bash
pip index versions polyglot-logger | head -5
pip install --dry-run polyglot-logger==0.2.0
```

#### NuGet

```bash
curl https://api.nuget.org/v3-flatcontainer/polyglot.logger/0.2.0/polyglot.logger.0.2.0.nupkg
# Should return 200 OK
```

#### GitHub Releases

```bash
gh release view v0.2.0  # or check website
# Verify artifacts are attached
```

### Create Release Notes (GitHub)

- [ ] Go to GitHub Releases
- [ ] Find v0.2.0 release
- [ ] Add release notes:
  - Features added
  - Bug fixes
  - Breaking changes (if any)
  - Installation instructions
  - Links to package pages

### Announce Release

- [ ] Update project website/homepage
- [ ] Post to community forums (if applicable)
- [ ] Create discussion/announcement (GitHub)

## Rollback Plan

If something goes wrong after publishing:

### If unpublished

```bash
# Unpublish from npm (within 72 hours)
npm unpublish @polyglot/logger@0.2.0

# Unpublish from PyPI (within 24 hours)
pip install twine
twine upload --repository testpypi dist/*  # and delete from production

# Delete NuGet version
# NuGet doesn't allow unpublishing, but you can unlist
dotnet nuget delete Polyglot.Logger 0.2.0
```

### If published

```bash
# Create patch release
git tag v0.2.1
git push origin v0.2.1
# This triggers a new build with the fix
```

## Post-Release (Next Day)

- [ ] Monitor GitHub Issues for any crash reports
- [ ] Check if users are successfully installing
- [ ] Respond to any community questions
- [ ] Plan next release (features, bugs)

## Troubleshooting

### Build Failed

- Check GitHub Actions logs
- Common issues:
  - CGO not enabled: Set `CGO_ENABLED=1` in environment
  - Missing cross-compiler: Install MinGW/GCC for target platform
  - Token invalid: Verify secrets in GitHub Settings

### Package Not Appearing

- Wait 10-15 minutes for registry to sync
- Verify package.json/pyproject.toml version matches tag
- Check if publish job succeeded in GitHub Actions

### Build Succeeded but Old Version Installed

- Clear local cache
  ```bash
  npm cache clean --force
  pip cache purge
  ```
- Verify package listing includes new version

## Template Commands

### Typical Release Workflow

```bash
# 1. Bump versions in all files

# 2. Commit
git add .
git commit -m "chore: release v0.2.0"

# 3. Tag
git tag -a v0.2.0 -m "Release v0.2.0"

# 4. Push (triggers GitHub Actions)
git push origin main && git push origin v0.2.0

# 5. Wait for Actions to complete (~15 min)

# 6. Verify packages
npm view @polyglot/logger@0.2.0
pip index versions polyglot-logger | head -5
```

---

**Questions?** See PUBLISHING.md for detailed information.
