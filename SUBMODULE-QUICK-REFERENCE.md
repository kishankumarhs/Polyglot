# Polyglot Submodule Quick Reference

## Initial Clone & Setup

```bash
# Clone with all submodules
git clone --recurse-submodules https://github.com/kishankumarhs/Polyglot.git
cd Polyglot

# If already cloned without submodules
git submodule update --init --recursive
```

## Submodule Status

```bash
# Check all submodule status
git submodule status

# Expected output:
#  55721d20d980d9bbcba27460a0b7ae200ae8847a bindings/node (heads/main)
#  bb966e3f2cf1f46b2c7f7ffb7809590f70d7a347 bindings/python (heads/main)
#  d280d908f5437e9ac61d7ee586da077e7bbaa199 bindings/dotnet (heads/main)
```

## Common Workflows

### Make Changes in Core (Go)

```bash
# Edit core code
vim internal/logger/logger.go
vim api/abi.json

# Commit to core
git add internal/logger/logger.go api/abi.json
git commit -m "feat: add feature description"
git push origin main

# If ABI changed, regenerate
go run ./cmd/codegen
```

### Fix Bug in One Binding (e.g., Python)

```bash
# Navigate to binding
cd bindings/python

# Make and test changes
vim polyglot_logger/auto_init.py
pytest tests/

# Commit in the binding
git add polyglot_logger/auto_init.py
git commit -m "fix: bug description"
git push origin main

# Return to core and update reference
cd ../..
git add bindings/python
git commit -m "chore: bump python binding"
git push origin main
```

### Update All Bindings to Latest

```bash
# Sync all to latest remote
git submodule update --remote --merge

# Commit parent repository
git add bindings/
git commit -m "chore: sync all bindings to latest"
git push origin main
```

### Push Changes in All Submodules

```bash
# Push to each submodule remote
git submodule foreach 'git push origin main'
```

## Repository Map

```
Core: https://github.com/kishankumarhs/Polyglot.git
├── bindings/node → https://github.com/kishankumarhs/polyglot-node.git
├── bindings/python → https://github.com/kishankumarhs/polyglot-py.git
└── bindings/dotnet → https://github.com/kishankumarhs/polyglot-csharp.git
```

## Key Files

- **Core ABI**: `api/abi.json`
- **Codegen**: `cmd/codegen/main.go`
- **Node binding**: `bindings/node/` (submodule)
- **Python binding**: `bindings/python/` (submodule)
- **.NET binding**: `bindings/dotnet/` (submodule)
- **Submodule config**: `.gitmodules`

## Troubleshooting

```bash
# Show detailed submodule info
git config -l | grep submodule

# Check submodule branch
cd bindings/node && git branch && cd ../..

# Reset submodule to parent's tracked commit
git submodule update --checkout

# Force all submodules to main branch
git submodule foreach 'git checkout main'

# View submodule commits being tracked
git diff --cached --submodule
```

## Important Notes

⚠️ **Always commit in the submodule first, then update the parent reference**

```bash
# WRONG: Don't commit only in parent
git add bindings/python
git commit -m "updated python"
# Changes are lost!

# RIGHT: Commit in submodule first
cd bindings/python
git commit -m "fix: change"
git push
cd ../..
git add bindings/python
git commit -m "chore: bump python"
git push
```

## Publishing Workflow

Each binding publishes independently:

1. **Node**: Tag in `polyglot-node` → GitHub Actions publishes to npm
2. **Python**: Tag in `polyglot-py` → GitHub Actions publishes to PyPI
3. **.NET**: Tag in `polyglot-csharp` → GitHub Actions publishes to NuGet

All binding repositories have independent GitHub Actions CI/CD workflows.

## Help

For detailed workflows, see: `docs/SUBMODULE-WORKFLOW.md`
