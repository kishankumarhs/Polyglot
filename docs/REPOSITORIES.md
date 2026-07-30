# Repositories

Core + three binding repos as submodules. Details: [architecture.md](architecture.md) · [SUBMODULE-WORKFLOW.md](SUBMODULE-WORKFLOW.md).

| Repo | URL | Package |
| --- | --- | --- |
| Polyglot (core) | https://github.com/kishankumarhs/Polyglot | — |
| polyglot-node | https://github.com/kishankumarhs/polyglot-node | `@polyglot-logger/node` |
| polyglot-py | https://github.com/kishankumarhs/polyglot-py | `polyglot-logger` |
| polyglot-csharp | https://github.com/kishankumarhs/polyglot-csharp | `Polyglot.Logger` |

```bash
git clone --recurse-submodules https://github.com/kishankumarhs/Polyglot.git
```

ABI (`api/abi.json`) and codegen live in the core. Bindings publish independently; a release tag should bump them together.
