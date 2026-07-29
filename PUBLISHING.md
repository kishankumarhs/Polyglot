# Publishing Polyglot Logger - Production-Grade Zero-Config Bindings

This guide explains how to publish Polyglot Logger as independent language packages to npm, PyPI, and NuGet with fully bundled native binaries.

## Architecture

Each language binding is published as a **completely independent, self-contained package** that includes:

- ✅ Pre-compiled native binaries for Windows, macOS, and Linux
- ✅ Language-specific FFI/bindings
- ✅ Auto-discovery code to find `polyglot.yaml` in the project root
- ✅ Automatic initialization on package import (zero-config!)

```text
User's Application
  ├── npm install @polyglot/logger
  ├── pip install polyglot-logger
  └── dotnet add package Polyglot.Logger
      ↓
  Each package includes:
  ├── Compiled bindings code
  ├── Pre-built native liblogger.so/.dll/.dylib
  └── Auto-init code for zero-config setup
      ↓
  On first import:
  ├── Auto-discover polyglot.yaml in project tree
  ├── Initialize Go logger engine
  └── User's code just works!
```

## Supported Platforms

Pre-compiled binaries are built for:

- **Linux**: x86_64
- **macOS**: x86_64 + arm64 (Apple Silicon)
- **Windows**: x86_64

## Publishing Workflow

### Automated via GitHub Actions (Recommended)

The project includes a complete CI/CD pipeline (`.github/workflows/release.yml`) that:

1. **Builds** native libraries for all platforms
2. **Packages** each language binding with binaries
3. **Creates** GitHub Release with all artifacts
4. **Publishes** to npm, PyPI, and NuGet simultaneously

#### Setup

1. **Add Secrets to GitHub**

   ```
   Settings → Secrets and variables → Actions → New repository secret
   ```

   Add:
   - `NPM_TOKEN`: Your npm token (generate at npmjs.com/settings/tokens)
   - `PYPI_TOKEN`: Your PyPI token (generate at pypi.org/account/tokens)
   - `NUGET_TOKEN`: Your NuGet API key (from nuget.org account)

2. **Tag and Push Release**

   ```bash
   # Update version in:
   # - api/abi.json (abi_version)
   # - bindings/node/package.json (version)
   # - bindings/python/pyproject.toml (version)
   # - bindings/dotnet/Polyglot.Logger/Polyglot.Logger.csproj (Version)

   git add .
   git commit -m "chore: release v0.2.0"
   git tag v0.2.0
   git push origin v0.2.0  # Triggers GitHub Actions!
   ```

3. **Monitor Build**
   - Go to GitHub → Actions tab
   - Watch the `Build & Release Polyglot Logger` workflow
   - It will automatically publish to all three registries on success

### Manual Publishing

If you prefer manual control over each step:

#### 1. Build Native Libraries

```bash
# Build for all platforms (requires cross-compilation setup)
./scripts/build-native.sh dist/

# Or individually
GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -buildmode=c-shared \
  -o dist/linux/x86_64/liblogger.so ./native

GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -buildmode=c-shared \
  -o dist/macos/x86_64/liblogger.dylib ./native

GOOS=windows GOARCH=amd64 CGO_ENABLED=1 go build -buildmode=c-shared \
  -o dist/windows/x86_64/logger.dll ./native
```

#### 2. Package Node.js

```bash
# Copy binaries
mkdir -p bindings/node/bin
cp dist/linux/x86_64/liblogger.so bindings/node/bin/
cp dist/macos/x86_64/liblogger.dylib bindings/node/bin/
cp dist/windows/x86_64/logger.dll bindings/node/bin/

# Build and package
cd bindings/node
npm install
npm run build
npm pack

# Publish
npm publish polyglot-logger-0.2.0.tgz
```

#### 3. Package Python

```bash
# Copy binaries
mkdir -p bindings/python/polyglot_logger/bin
cp dist/linux/x86_64/liblogger.so bindings/python/polyglot_logger/bin/
cp dist/macos/x86_64/liblogger.dylib bindings/python/polyglot_logger/bin/
cp dist/windows/x86_64/logger.dll bindings/python/polyglot_logger/bin/

# Build and package
cd bindings/python
python -m build

# Publish
twine upload dist/*
```

#### 4. Package .NET

```bash
# Copy binaries
mkdir -p bindings/dotnet/Polyglot.Logger/native
cp dist/linux/x86_64/liblogger.so bindings/dotnet/Polyglot.Logger/native/
cp dist/macos/x86_64/liblogger.dylib bindings/dotnet/Polyglot.Logger/native/
cp dist/windows/x86_64/logger.dll bindings/dotnet/Polyglot.Logger/native/

# Build and package
cd bindings/dotnet/Polyglot.Logger
dotnet pack -c Release

# Publish
dotnet nuget push bin/Release/*.nupkg -k YOUR_NUGET_KEY -s https://api.nuget.org/v3/index.json
```

## Package Installation & Zero-Config Usage

Once published, users can install any binding and use it immediately:

### Node.js

```bash
npm install @polyglot/logger
```

```javascript
// Just import - auto-discovers polyglot.yaml and initializes!
import { Logger } from "@polyglot/logger";

const logger = new Logger({ service: "my-app" });
logger.info("This works without any config!", { userId: 123 });
```

### Python

```bash
pip install polyglot-logger
```

```python
# Just import - auto-discovers polyglot.yaml and initializes!
from polyglot_logger import Logger

logger = Logger("my-app")
logger.info("This works without any config!", {"user_id": 123})
```

### .NET

```bash
dotnet add package Polyglot.Logger
```

```csharp
// Just use - auto-discovers polyglot.yaml and initializes!
using Polyglot.Logger;

var logger = new Logger("my-app");
logger.Info("This works without any config!", new { userId = 123 });
```

## Auto-Discovery Mechanism

Each package automatically:

1. **On Import** → Calls the auto-finder
2. **Climb Directory Tree** → From `cwd()` up to filesystem root looking for `polyglot.yaml`
3. **Find Config** → If found, passes absolute path to native logger
4. **Initialize** → Native logger loads YAML/JSON and starts
5. **Fallback** → If no config found, uses safe defaults
6. **Environment Variables** → Also checks `POLYGLOT_CONFIG_PATH` env var

## Config File Resolution Order

1. **`polyglot.yaml` (or `.yml`)** in project root or parent directories
2. **`POLYGLOT_CONFIG_PATH`** environment variable (if set)
3. **`POLYGLOT_CONFIG_FILE`** environment variable (legacy, if set)
4. **Safe defaults** (if nothing found)

## Multi-Platform Binary Distribution

The GitHub Actions workflow automatically creates multi-platform packages:

**npm package (`@polyglot/logger@0.2.0`)** includes:

```
@polyglot/logger/
├── dist/
│   ├── index.js
│   ├── auto-init.js
│   └── index.d.ts
├── bin/
│   ├── liblogger.so     (Linux)
│   ├── liblogger.dylib  (macOS)
│   └── logger.dll       (Windows)
└── package.json
```

**PyPI package (`polyglot-logger 0.2.0`)** includes:

```
polyglot_logger/
├── __init__.py
├── auto_init.py
├── _ffi_generated.py
└── bin/
    ├── liblogger.so     (Linux)
    ├── liblogger.dylib  (macOS)
    └── logger.dll       (Windows)
```

**.NET package (`Polyglot.Logger 0.2.0`)** includes:

```
runtimes/
├── linux-x64/native/
│   └── liblogger.so
├── osx-x64/native/
│   └── liblogger.dylib
└── win-x64/native/
    └── logger.dll
```

## Troubleshooting

### "Native library not found"

Ensure the package includes binaries:

```bash
# Node.js
ls node_modules/@polyglot/logger/bin/

# Python
python -c "from pathlib import Path; print(Path('polyglot_logger').parent / 'bin')"

# .NET
ls ~/.nuget/packages/polyglot.logger/0.2.0/runtimes/*/native/
```

### "polyglot.yaml not found"

Explicitly set via environment variable:

```bash
export POLYGLOT_CONFIG_PATH=/path/to/polyglot.yaml
node app.js
```

### Different versions of bindings installed

Ensure all versions match in your project:

```bash
npm ls @polyglot/logger
pip show polyglot-logger
dotnet list package --include-transitive | grep Polyglot
```

## Release Checklist

Before tagging a release:

- [ ] Update versions in all package files
- [ ] Run tests: `npm test`, `pytest`, `dotnet test`
- [ ] Verify native builds compile for all platforms
- [ ] Test installation: `npm install`, `pip install`, `dotnet add package`
- [ ] Test zero-config initialization with polyglot.yaml
- [ ] Update CHANGELOG.md with new features/fixes
- [ ] Commit and tag: `git tag v0.2.0 && git push --tags`
- [ ] Monitor GitHub Actions workflow
- [ ] Verify packages appear on npm/PyPI/NuGet registries
- [ ] Create GitHub Release with release notes

## Version Management

Keep versions synchronized across all bindings:

**`bindings/node/package.json`**

```json
{ "version": "0.2.0" }
```

**`bindings/python/pyproject.toml`**

```toml
version = "0.2.0"
```

**`bindings/dotnet/Polyglot.Logger/Polyglot.Logger.csproj`**

```xml
<Version>0.2.0</Version>
```

Use SemVer format: `MAJOR.MINOR.PATCH`

## Advanced: Building Your Own Cross-Compiler

If GitHub Actions' standard build environment doesn't meet your needs:

```bash
# Build for specific targets with your own compiler
./scripts/build-native.sh dist/ \
  --targets=linux/x86_64,macos/x86_64,macos/arm64,windows/x86_64 \
  --gcc-path=/custom/gcc \
  --clang-path=/custom/clang
```

Modify `scripts/build-native.sh` to accept custom compiler paths if needed.

## Support & Issues

- **npm**: https://www.npmjs.com/package/@polyglot/logger
- **PyPI**: https://pypi.org/project/polyglot-logger/
- **NuGet**: https://www.nuget.org/packages/Polyglot.Logger/
- **GitHub**: https://github.com/polyglot/logger (issues & discussions)
