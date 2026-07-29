# Polyglot Multi-Language Bindings - Submodule Workflow Guide

## Repository Structure

The Polyglot Logger project is now organized as a **modular monorepo** with independent language bindings tracked as Git submodules:

```
polyglot-go (Core Repository)
├── api/                    # C ABI contract (abi.json)
├── cmd/codegen/            # Code generation for bindings
├── internal/logger/        # Go logger implementation
├── native/                 # CGO exports
├── scripts/                # Build scripts
├── bindings/               # Language bindings (submodules)
│   ├── node → polyglot-node
│   ├── python → polyglot-py
│   └── dotnet → polyglot-csharp
└── .gitmodules            # Submodule configuration
```

### Submodule Repositories

| Language | Repository      | URL                                                  |
| -------- | --------------- | ---------------------------------------------------- |
| Node.js  | polyglot-node   | https://github.com/kishankumarhs/polyglot-node.git   |
| Python   | polyglot-py     | https://github.com/kishankumarhs/polyglot-py.git     |
| .NET     | polyglot-csharp | https://github.com/kishankumarhs/polyglot-csharp.git |

---

## Quick Start for New Contributors

### Clone with All Submodules

```bash
# Clone the core repository with all language bindings
git clone --recurse-submodules https://github.com/kishankumarhs/polyglot-go.git

cd polyglot-go
```

If you already cloned without submodules:

```bash
git submodule update --init --recursive
```

### Verify Submodules Are Loaded

```bash
git submodule status

# Expected output:
#  55721d20d980d9bbcba27460a0b7ae200ae8847a bindings/node (heads/main)
#  bb966e3f2cf1f46b2c7f7ffb7809590f70d7a347 bindings/python (heads/main)
#  d280d908f5437e9ac61d7ee586da077e7bbaa199 bindings/dotnet (heads/main)
```

---

## Development Workflows

### Workflow 1: Modify Core Go Code

When you change the Go logger implementation or C ABI contract:

```bash
# 1. Make changes in the core repository
cd polyglot-go
vim api/abi.json  # or internal/logger/logger.go

# 2. Commit to core repository
git add api/abi.json
git commit -m "feat: add new logging level"
git push origin main

# 3. Regenerate bindings (if ABI changed)
go run ./cmd/codegen

# 4. The codegen automatically writes to submodule paths
# Submodule changes will be detected by git
```

### Workflow 2: Fix Python Binding Bug

When you fix a bug inside `bindings/python`:

```bash
# 1. Navigate to the Python submodule
cd polyglot-go/bindings/python

# 2. Make changes and test
vim polyglot_logger/auto_init.py
pytest tests/

# 3. Commit within the submodule
git add polyglot_logger/auto_init.py
git commit -m "fix: resolve path traversal bug in auto-finder"
git push origin main

# 4. Return to core repository and update the reference
cd ../..
git add bindings/python
git commit -m "chore: bump python binding to latest commit"
git push origin main
```

### Workflow 3: Update Node.js FFI Bindings

When codegen updates FFI bindings:

```bash
# 1. Codegen writes to bindings/node/src/ffi.generated.ts
go run ./cmd/codegen

# 2. Test the new bindings
cd bindings/node
npm test

# 3. If good, commit in the submodule
git add src/ffi.generated.ts
git commit -m "chore: regenerated FFI for new ABI"
git push origin main

# 4. Update parent repository reference
cd ../..
git add bindings/node
git commit -m "chore: bump node binding to new codegen version"
git push origin main
```

### Workflow 4: Sync Submodules to Latest

To pull latest changes from all binding repositories:

```bash
cd polyglot-go
git submodule update --remote --merge

# This checks out the latest commits from each submodule's remote
git add bindings/
git commit -m "chore: sync all bindings to latest"
git push origin main
```

---

## Release Workflow

### Single Language Release (e.g., Node.js only)

```bash
# 1. Make changes in node binding
cd bindings/node
npm version minor  # Updates package.json version
git push origin main
git push origin --tags

# 2. Update parent repository to track new node version
cd ../..
git add bindings/node
git commit -m "chore: bump node binding to v1.1.0"
git push origin main

# GitHub Actions in polyglot-node publishes to npm automatically
```

### Multi-Language Release (All Bindings)

```bash
# 1. Update core version
vim api/abi.json  # Update abi_version

# 2. Run codegen to update all bindings
go run ./cmd/codegen

# 3. Release each binding
cd bindings/node && npm version patch && git push origin --tags && cd ../..
cd bindings/python && poetry version patch && git push origin --tags && cd ../..
cd bindings/dotnet && cd Polyglot.Logger && dotnet bump-version && cd ../.. && git push origin --tags && cd ../..

# 4. Update core repository
git add bindings/ api/
git commit -m "chore: release v0.3.0 - all language bindings"
git push origin main
```

---

## Common Tasks

### View Submodule Commit Details

```bash
# See which specific commits are tracked
git submodule status

# Show full commit info
git log --oneline -1 bindings/node
git log --oneline -1 bindings/python
git log --oneline -1 bindings/dotnet
```

### Check If Submodules Have Uncommitted Changes

```bash
git submodule foreach 'if [ -n "$(git status --porcelain)" ]; then echo "=== $name ==="; git status; fi'
```

### Push All Submodule Changes to Remote

```bash
git submodule foreach 'git push origin main'
```

### Create a Branch Across All Repositories

```bash
# Create branch in core repo
git checkout -b feature/new-sink

# Create matching branches in all submodules
git submodule foreach 'git checkout -b feature/new-sink'

# Make changes and commit
git add -A
git commit -m "feat: new sink implementation across all bindings"
git submodule foreach 'git add -A && git commit -m "feat: new sink implementation" || true'

# Push all
git push origin feature/new-sink
git submodule foreach 'git push origin feature/new-sink'
```

### List All Files in All Submodules

```bash
git ls-files --stage | grep 160000  # Shows submodule references
```

---

## Troubleshooting

### Submodule Shows "untracked content" After Clone

```bash
# Submodules might be at a detached HEAD state
git submodule update --checkout
git submodule foreach 'git checkout main'
```

### Codegen Doesn't See Submodule Changes

The codegen writes directly to submodule paths. If changes aren't appearing:

```bash
# Verify submodule is initialized
git submodule status

# Manually pull latest if needed
cd bindings/node && git pull origin main && cd ../..
cd bindings/python && git pull origin main && cd ../..
cd bindings/dotnet && git pull origin main && cd ../..
```

### Accidentally Edited Submodule Without Committing

```bash
# Discard changes in a submodule
cd bindings/node
git checkout .

# Or reset to tracked commit
git reset --hard
```

### Need to Remove a Submodule

```bash
# Remove submodule (e.g., node)
git submodule deinit bindings/node
git rm bindings/node
git config --remove-section submodule.bindings/node
git commit -m "chore: remove node binding submodule"
```

---

## Git Commands Reference

| Command                                        | Purpose                                       |
| ---------------------------------------------- | --------------------------------------------- |
| `git submodule status`                         | Show all submodules and their tracked commits |
| `git submodule update --init --recursive`      | Initialize all submodules on first clone      |
| `git submodule update --remote --merge`        | Fetch latest from all submodule remotes       |
| `git submodule foreach 'git status'`           | Run git status in each submodule              |
| `git submodule foreach 'git push origin main'` | Push changes in all submodules                |
| `git clone --recurse-submodules <url>`         | Clone with all submodules initialized         |

---

## CI/CD Implications

### GitHub Actions in Each Binding Repository

Each binding has its own GitHub Actions workflow:

- **polyglot-node**: `npm test`, `npm publish` on release
- **polyglot-py**: `pytest`, `twine upload` on release
- **polyglot-csharp**: `dotnet test`, `dotnet nuget push` on release

When you push to a submodule repository, its GitHub Actions automatically run.

### Testing Changes Across Binding + Core

```bash
# After making changes to bindings and core:
git submodule foreach 'npm test'  # Node tests
git submodule foreach 'pytest'    # Python tests
go test ./...                      # Core tests
```

---

## Best Practices

✅ **DO:**

- Always push submodule changes before updating parent references
- Use descriptive commit messages that reference submodule updates
- Test each submodule independently before updating core reference
- Keep submodule commits aligned with core releases

❌ **DON'T:**

- Force push to submodule branches (breaks other developers' tracking)
- Modify submodule files directly from the parent repository
- Leave uncommitted submodule changes before switching branches
- Assume submodules are automatically updated (use `--recurse-submodules` flag)

---

## FAQ

**Q: Do I need to clone all submodules if I only work on one binding?**

A: No! You can clone just the specific binding:

```bash
git clone https://github.com/kishankumarhs/polyglot-node.git
```

**Q: Can I push to a submodule independently?**

A: Yes! Each submodule is a complete Git repository. Changes pushed there are independent.

**Q: What happens if I modify a submodule without committing?**

A: The parent repository will show "untracked content" in `git status`. Commit and push within the submodule first.

**Q: How do I see what changed between two submodule versions?**

A: ```bash
cd bindings/node
git log --oneline v1.0.0..v1.1.0
git diff v1.0.0 v1.1.0

```

---

## Contact & Support

- Core Repository Issues: [polyglot-go](https://github.com/kishankumarhs/polyglot-go/issues)
- Node Binding Issues: [polyglot-node](https://github.com/kishankumarhs/polyglot-node/issues)
- Python Binding Issues: [polyglot-py](https://github.com/kishankumarhs/polyglot-py/issues)
- .NET Binding Issues: [polyglot-csharp](https://github.com/kishankumarhs/polyglot-csharp/issues)
```
