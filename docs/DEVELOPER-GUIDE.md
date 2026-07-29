# Developer Onboarding Guide

Welcome to Polyglot Logger development! This guide explains the modular architecture and how to work effectively across the core repository and language bindings.

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Initial Setup](#initial-setup)
3. [Repository Structure](#repository-structure)
4. [Development Workflows](#development-workflows)
5. [Making Changes](#making-changes)
6. [Testing](#testing)
7. [Committing & Pushing](#committing--pushing)
8. [Common Tasks](#common-tasks)
9. [Troubleshooting](#troubleshooting)

## Architecture Overview

Polyglot Logger uses a **modular monorepo with Git submodules**:

```
polyglot-go (core)
├── Go logger implementation
├── C ABI contract (api/abi.json)
├── Code generator (cmd/codegen)
│
└── bindings/ (Git submodules)
    ├── node → polyglot-node (independent npm package)
    ├── python → polyglot-py (independent PyPI package)
    └── dotnet → polyglot-csharp (independent NuGet package)
```

**Key principles:**

- **Single source of truth:** C ABI in `api/abi.json` drives all language bindings
- **Independent repositories:** Each binding is a separate GitHub repository
- **Code generation:** Language FFI files are auto-generated, never hand-edited
- **Independent versioning:** Each binding versions independently (Node v1.2.3, Python v2.0.0, etc.)

## Initial Setup

### Clone with All Submodules

Always use `--recurse-submodules` when cloning:

```bash
git clone --recurse-submodules https://github.com/kishankumarhs/Polyglot.git
cd Polyglot
```

If you already cloned without submodules:

```bash
git submodule update --init --recursive
```

### Verify Submodule Status

```bash
git submodule status
```

Expected output:

```
 <commit> bindings/node (HEAD detached at <commit>)
 <commit> bindings/python (HEAD detached at <commit>)
 <commit> bindings/dotnet (HEAD detached at <commit>)
```

All submodules should be at a specific commit (no `+` or `-` markers).

### Build Native Library

```bash
# Install build dependencies (Go, Rust, C compiler, etc.)
# See docs/build.md for platform-specific instructions

# Build cross-platform natives
make build-native
```

This produces:

- `bin/logger.dll` (Windows)
- `bin/liblogger.dylib` (macOS)
- `bin/liblogger.so` (Linux)

### Install Language Tools

Depending on which binding you'll work on:

```bash
# Node.js
cd bindings/node && npm install

# Python
cd bindings/python && pip install -e .

# .NET
# Requires .NET 8.0+
dotnet --version
```

## Repository Structure

### Core Repository (polyglot-go)

```
.
├── api/
│   ├── abi.json              # C ABI contract (single source of truth)
│   └── README.md
├── cmd/
│   ├── codegen/              # Generates FFI for all bindings
│   │   └── main.go
│   └── logger-demo/          # Demo program
├── internal/
│   └── logger/               # Go implementation
│       ├── logger.go
│       ├── config.go
│       ├── sink_*.go
│       └── *_test.go
├── native/
│   ├── export.go             # CGO exports (calls internal/logger)
│   ├── include/
│   │   └── logger.h          # Generated C header
│   └── *_test.go
├── bindings/                 # Git submodules
│   ├── node → polyglot-node
│   ├── python → polyglot-py
│   └── dotnet → polyglot-csharp
├── scripts/
│   ├── build-native.sh       # macOS/Linux build
│   └── build-native.ps1      # Windows build
├── docs/                     # Documentation
├── go.mod                    # Go module definition
├── Makefile                  # Build automation
└── README.md
```

### Binding Repository (polyglot-node example)

```
.
├── src/
│   ├── index.ts              # Main export
│   ├── auto-init.ts          # Auto-discovery & initialization
│   └── ffi.generated.ts      # Generated FFI (read-only!)
├── test/
│   └── smoke.test.js
├── bin/                      # Native binaries (populated on npm install)
│   ├── logger.dll
│   ├── liblogger.dylib
│   └── liblogger.so
├── package.json              # npm metadata
├── tsconfig.json
├── README.md
└── LICENSE
```

Similar structure for Python and .NET bindings.

## Development Workflows

### Scenario 1: Fix Bug in Go Core

```bash
# 1. You're in polyglot-go main directory
cd /path/to/Polyglot

# 2. Make changes to Go code
vim internal/logger/logger.go

# 3. Run tests
go test ./internal/logger/...

# 4. Rebuild native library
make build-native

# 5. Test in a binding (e.g., Node.js)
cd bindings/node
npm test

# 6. Commit to core
cd ../..
git add internal/logger/
git commit -m "fix: goroutine memory leak"
git push origin main
```

### Scenario 2: Update ABI (Add New Function)

```bash
# 1. Edit api/abi.json to add function signature
vim api/abi.json

# 2. Implement in Go
vim internal/logger/logger.go

# 3. Export via CGO
vim native/export.go
# Add: //export new_function_name

# 4. Run codegen
go run ./cmd/codegen

# 5. Verify generated files
git diff native/include/logger.h
git diff bindings/node/src/ffi.generated.ts
git diff bindings/python/polyglot_logger/_ffi_generated.py
git diff bindings/dotnet/Polyglot.Logger/NativeMethods.Generated.cs

# 6. If good, commit
git add api/abi.json internal/logger/ native/export.go native/include/logger.h
git add bindings/*/src/*.generated.* bindings/*/*.generated.*
git commit -m "feat: add new_function_name to ABI"

# 7. Each binding must still be independently tested/released
cd bindings/node
npm test && npm publish  # Separate release
```

**Important:** Codegen verifies consistency before writing outputs. If ABI doesn't match exports, codegen fails.

### Scenario 3: Fix Bug in Node.js Binding

```bash
# 1. Submodule changes are isolated
cd bindings/node

# 2. Make changes (never edit *.generated.ts!)
vim src/index.ts
vim src/auto-init.ts

# 3. Test
npm test

# 4. Commit & push to polyglot-node
git add src/
git commit -m "fix: handle initialization race condition"
git push origin main

# 5. Update core superproject reference
cd ../..
git add bindings/node
git commit -m "chore: bump node binding"
git push origin main

# At this point:
# - polyglot-node repo has the fix
# - polyglot-go repo references latest polyglot-node commit
# - You can release polyglot-node independently (npm publish)
```

### Scenario 4: Work on All Three Bindings Together

Example: You discover all bindings have a thread safety issue.

```bash
# 1. Document the issue
# Create a GitHub issue in polyglot-go with [BINDINGS] tag

# 2. Fix in each binding independently

# Fix Node.js
cd bindings/node
# Fix code, test
npm test
git commit -m "fix: thread safety in concurrent log calls"
git push origin main

# Fix Python
cd ../python
pip install -e .
pytest tests/
git commit -m "fix: thread safety in concurrent log calls"
git push origin main

# Fix .NET
cd ../dotnet
dotnet test
git commit -m "fix: thread safety in concurrent log calls"
git push origin main

# 3. Update core superproject to pin all three
cd ../..
git add bindings/node bindings/python bindings/dotnet
git commit -m "chore: bump all bindings with thread safety fixes"
git push origin main

# 4. Each binding team can now release:
# Node: npm publish
# Python: python -m build && twine upload dist/
# .NET: dotnet pack && nuget push
```

## Making Changes

### Hand-Written vs. Generated Code

| Location                                     | Type         | Edit?  | Notes                         |
| -------------------------------------------- | ------------ | ------ | ----------------------------- |
| `internal/logger/*.go`                       | Hand-written | ✅ YES | Core implementation           |
| `native/export.go`                           | Hand-written | ✅ YES | CGO exports & wrapping        |
| `api/abi.json`                               | Hand-written | ✅ YES | ABI contract                  |
| `native/include/logger.h`                    | Generated    | ❌ NO  | Run codegen after ABI changes |
| `*_ffi_generated.ts`                         | Generated    | ❌ NO  | Regenerated from ABI          |
| `*_ffi_generated.py`                         | Generated    | ❌ NO  | Regenerated from ABI          |
| `NativeMethods.Generated.cs`                 | Generated    | ❌ NO  | Regenerated from ABI          |
| `src/index.ts` / `__init__.py` / `Logger.cs` | Hand-written | ✅ YES | Binding SDK code              |

### Modifying the ABI

When changing `api/abi.json`:

1. **Edit api/abi.json**

   ```json
   {
     "functions": [
       {
         "name": "logger_create",
         "arguments": ["const char*", "const char*"],
         "returns": "handle"
       }
     ]
   }
   ```

2. **Implement in Go**

   ```go
   // internal/logger/logger.go
   func NewLogger(service, env string) *Logger {
     // ...
   }
   ```

3. **Export via CGO**

   ```go
   // native/export.go
   //export logger_create
   func logger_create(service *C.char, env *C.char) C.uintptr_t {
     // Wrap Go implementation
   }
   ```

4. **Run codegen**

   ```bash
   go run ./cmd/codegen
   ```

   Codegen verifies:
   - Every function in `api/abi.json` has a `//export` in `native/export.go`
   - Argument count matches
   - Return type is recognized

5. **Commit everything**

   ```bash
   git add api/abi.json internal/logger/ native/
   git commit -m "feat: add logger_create to ABI"
   git push origin main
   ```

6. **Each binding auto-updates on pull**

   ```bash
   # In polyglot-node repo
   git pull origin main
   # ffi.generated.ts is now updated
   npm test
   npm publish
   ```

## Testing

### Testing Go Core

```bash
# From polyglot-go root
go test ./internal/logger/...
go test ./native/...

# With coverage
go test -cover ./...
```

### Testing Node.js Binding

```bash
cd bindings/node
npm install
npm test
```

### Testing Python Binding

```bash
cd bindings/python
pip install -e .[dev]  # Install test dependencies
pytest tests/
```

### Testing .NET Binding

```bash
cd bindings/dotnet
dotnet test
```

### Integration Testing (All Together)

```bash
# From polyglot-go root
make build-native

# Test all bindings
cd bindings/node && npm test
cd ../python && pytest tests/
cd ../dotnet && dotnet test

# Or use Makefile
make test  # If configured
```

## Committing & Pushing

### Core Repository Commits

When working on the core (`polyglot-go`):

```bash
# Stage changes
git add internal/logger/ native/ api/
git add docs/  # If updating docs

# Commit with detailed message
git commit -m "feat: async logger with worker pool

- Implement worker goroutine for serialized writes
- Support configurable overflow policies
- Add stats tracking (queued, flushed, dropped, errors)
- Closes #42"

# Push to GitHub
git push origin main
```

### Submodule Commits

When working in a binding (`bindings/node`):

```bash
cd bindings/node

# Make changes (never touch *.generated.*)
git add src/index.ts src/auto-init.ts

# Commit locally
git commit -m "fix: handle auto-initialization race condition"

# Push to binding repo (polyglot-node)
git push origin main

# Return to core and update reference
cd ../..
git add bindings/node  # This stages the new commit reference

git commit -m "chore: bump node binding to latest main"

git push origin main  # Update core to point to new binding commit
```

### Commit Message Guidelines

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add new feature
fix: correct a bug
docs: documentation updates
test: add or update tests
chore: maintenance (deps, build, etc.)
refactor: code restructuring
perf: performance improvements
```

Examples:

```
feat: add logger_create_from_config_file export
fix: handle panic in JSON parsing
docs: update Node.js API documentation
chore: bump polyglot-py to v2.0.0
```

## Common Tasks

### Task 1: Add a New Field to Logger Config

1. Update `api/abi.json` to add field to config struct
2. Update `internal/logger/config.go` to parse the new field
3. Run `go run ./cmd/codegen`
4. Test in each binding
5. Commit across all repos if needed

### Task 2: Update Documentation

```bash
# Edit docs
vim docs/architecture.md docs/user-guide.md

# Verify links work
# Check markdown syntax

# Commit
git add docs/
git commit -m "docs: clarify async pipeline behavior"
git push origin main
```

### Task 3: Prepare a Release

See [RELEASE-CHECKLIST.md](../RELEASE-CHECKLIST.md) for detailed instructions.

Quick version:

```bash
# Core release
git tag v1.2.3
git push origin v1.2.3

# Each binding release (independent)
cd bindings/node
npm version minor
npm publish

cd ../python
python -m build
twine upload dist/

cd ../dotnet
dotnet pack
# Push to NuGet (manual for security)
```

### Task 4: Sync Submodules to Latest

```bash
# Pull latest from all submodules
git submodule update --remote

# Review changes
git status

# Update references in core
git add bindings/
git commit -m "chore: sync submodules to latest"
git push origin main
```

### Task 5: Create a Feature Branch

For complex features, use feature branches:

```bash
# Create branch in core
git checkout -b feature/new-sink-type

# Also create in binding (if needed)
cd bindings/python
git checkout -b feature/new-sink-type

# Make changes in both
# Commit in each
git commit -m "..."
git push origin feature/new-sink-type

# When ready, create PR in each
# After merge, update core reference
cd ../..
git pull origin main  # Already merged core branch
git add bindings/python
git commit -m "chore: merge feature/new-sink-type"
git push origin main
```

## Troubleshooting

### "fatal: No submodule mapping found"

**Problem:** Tried to clone without `--recurse-submodules`

**Solution:**

```bash
git submodule update --init --recursive
```

### "error: You are not currently on a branch"

**Problem:** Submodule is in detached HEAD state (normal!)

**Solution:** This is expected. Each submodule pins to a specific commit. Proceed normally:

```bash
cd bindings/node
git checkout main  # If you want to work on main branch
# Or stay in detached HEAD if just reviewing
```

### "cannot find native library"

**Problem:** Native binary not compiled

**Solution:**

```bash
cd /path/to/Polyglot
make build-native

# Verify binaries exist
ls -la bin/

# Copy to binding (if not auto-discovered)
cp bin/liblogger.so bindings/python/polyglot_logger/bin/
```

### "codegen fails: ABI mismatch"

**Problem:** `native/export.go` doesn't match `api/abi.json`

**Solution:**

```bash
# Read the error message
go run ./cmd/codegen

# Fix the mismatches (usually missing //export or wrong signature)
vim native/export.go

# Retry
go run ./cmd/codegen
```

### "npm test fails: cannot find module"

**Problem:** Dependencies not installed

**Solution:**

```bash
cd bindings/node
rm -rf node_modules package-lock.json
npm install
npm test
```

### "Python import fails"

**Problem:** Package not installed in dev mode

**Solution:**

```bash
cd bindings/python
pip install -e .

# Verify
python -c "from polyglot_logger import Logger; print(Logger)"
```

### "Binding tests fail but core tests pass"

**Problem:** Binding is out-of-sync with core

**Solution:**

```bash
# Update submodule to latest
git submodule update --remote bindings/node

# Rebuild natives
make build-native

# Retest binding
cd bindings/node
npm test
```

## Next Steps

- Read [REPOSITORIES.md](REPOSITORIES.md) for overview of all 4 repositories
- See [SUBMODULE-WORKFLOW.md](SUBMODULE-WORKFLOW.md) for Git submodule workflow details
- Check [build.md](build.md) for platform-specific build instructions
- Review [PUBLISHING.md](../PUBLISHING.md) for release process

## Questions?

Open an issue with label `[DOCS]` or `[DEV]` on the main repository.

Happy coding! 🚀
