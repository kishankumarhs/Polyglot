# Polyglot Logger v0.2.0 Release Guide

## 📦 Release Status

### ✅ Complete

- **GitHub Release v0.2.0:** https://github.com/kishankumarhs/Polyglot/releases/tag/v0.2.0
- **Build Status:** ✅ SUCCESS (all platforms tested)
- **npm Package:** Built and ready for publishing
- **Workflow Updates:** All GitHub Actions updated to latest versions

### ⏳ Pending

- Publishing to npm registry (@polyglot/logger)
- Publishing to PyPI (polyglot-logger)
- Publishing to NuGet (Polyglot.Logger)

## 🚀 Publishing All Submodules

### Prerequisites

You'll need authentication tokens for each registry:

- `NPM_TOKEN` - from https://www.npmjs.com/settings/~/tokens
- `PYPI_TOKEN` - from https://pypi.org/manage/account/tokens/
- `NUGET_API_KEY` - from https://www.nuget.org/account/apikeys

### Step 1: Publish to npm

```bash
cd bindings/node

# Option A: Using npm token
npm publish --access public

# Option B: Using npm login
npm login  # Follow prompts with your npm credentials
npm publish
```

**Expected Output:**

```
npm notice Publishing to https://registry.npmjs.org/ with tag latest and default access
```

### Step 2: Publish to PyPI

```bash
cd bindings/python

# Install build tools
pip install build twine

# Build distribution
python -m build

# Publish (use __token__ as username, PYPI_TOKEN as password)
twine upload dist/*
```

**Expected Output:**

```
Uploading polyglot_logger-0.2.0-py3-none-any.whl
```

### Step 3: Publish to NuGet

```bash
cd bindings/dotnet/Polyglot.Logger

# Build package
dotnet pack -c Release -o ../dist/

# Publish (requires NUGET_API_KEY)
cd ../dist
dotnet nuget push Polyglot.Logger.0.2.0.nupkg \
  -k $NUGET_API_KEY \
  -s https://api.nuget.org/v3/index.json
```

## 📋 Package Details

### npm (@polyglot/logger)

- **Version:** 0.2.0
- **Size:** 2.6 MB (with macOS dylib)
- **Files:** TypeScript compiled output + native binary
- **Location:** `bindings/node/polyglot-logger-0.2.0.tgz`

### PyPI (polyglot-logger)

- **Version:** 0.2.0
- **Package:** `bindings/python/pyproject.toml`
- **Requirements:** Python 3.9+

### NuGet (Polyglot.Logger)

- **Version:** 0.2.0
- **Framework:** .NET 8.0
- **Package:** `bindings/dotnet/Polyglot.Logger/Polyglot.Logger.csproj`

## 🔍 Verification

After publishing, verify each package:

```bash
# Check npm
npm view @polyglot/logger@0.2.0

# Check PyPI
pip index versions polyglot-logger

# Check NuGet
dotnet package search Polyglot.Logger
```

## 📝 Automated Workflow Status

The GitHub Actions `release.yml` workflow is configured to:

1. Build native libraries for Windows/macOS/Linux
2. Package for npm, PyPI, NuGet
3. Automatically publish when tag matches `v*.*.*`

**Current Status:** Manual publishing required due to CI environment differences (Linux build configuration needs adjustment)

## 🆘 Troubleshooting

### npm publish fails

- Verify NPM_TOKEN is valid: `npm token list`
- Check package name isn't already published: `npm view @polyglot/logger`
- Ensure you're publishing to the correct registry: `npm config get registry`

### PyPI upload fails

- Verify PYPI_TOKEN is valid by logging in manually
- Check distribution files: `ls -la bindings/python/dist/`
- Use verbose flag for debugging: `twine upload -v dist/*`

### NuGet push fails

- Verify API key is valid: `dotnet nuget verify`
- Ensure .nupkg file exists: `ls -la bindings/dotnet/dist/`
- Check NuGet registry URL is correct (should be https://api.nuget.org/v3/index.json)

## 📞 Support

For issues with:

- **Build/Release Workflow:** Check `.github/workflows/release.yml`
- **Package Configuration:** See respective `package.json`, `pyproject.toml`, `.csproj`
- **Native Binaries:** Check `scripts/build-native.sh`
