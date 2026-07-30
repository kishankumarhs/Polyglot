# Submodule workflow

Bindings live in separate repos and are tracked as submodules under `bindings/`.

| Language | Repo | URL |
| --- | --- | --- |
| Node | polyglot-node | https://github.com/kishankumarhs/polyglot-node.git |
| Python | polyglot-py | https://github.com/kishankumarhs/polyglot-py.git |
| .NET | polyglot-csharp | https://github.com/kishankumarhs/polyglot-csharp.git |

## Clone

```bash
git clone --recurse-submodules https://github.com/kishankumarhs/Polyglot.git
# or later:
git submodule update --init --recursive
```

## Change a binding

```bash
cd bindings/node   # or python / dotnet
# edit, commit, push to that binding's remote
git push origin HEAD

cd ../..
git add bindings/node
git commit -m "chore: bump node binding"
```

## Change the core

Work in the parent repo as usual. Run `make codegen` after ABI changes and commit generated files.

## Update all submodules to latest remote

```bash
git submodule update --remote --merge
git add bindings/
git commit -m "chore: bump bindings"
```

## Status / check

```bash
git submodule status
bash scripts/check-submodules.sh
```

Quick cheat sheet: [../SUBMODULE-QUICK-REFERENCE.md](../SUBMODULE-QUICK-REFERENCE.md).
