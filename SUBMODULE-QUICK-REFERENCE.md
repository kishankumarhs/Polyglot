# Submodule quick reference

```bash
git clone --recurse-submodules https://github.com/kishankumarhs/Polyglot.git
# already cloned?
git submodule update --init --recursive
```

```bash
git submodule status
```

## Core change

Edit in the parent repo, `make codegen` if ABI changed, commit, push.

## Binding change

```bash
cd bindings/node   # or python / dotnet
# commit + push to binding remote
cd ../..
git add bindings/node
git commit -m "chore: bump node binding"
```

## Bump all to remote tip

```bash
git submodule update --remote --merge
git add bindings/
git commit -m "chore: bump bindings"
```

Longer notes: [docs/SUBMODULE-WORKFLOW.md](docs/SUBMODULE-WORKFLOW.md).
