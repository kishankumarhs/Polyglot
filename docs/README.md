# Eximietas Logger documentation

Cross-language structured logging: one Go core, a stable C ABI, and thin bindings for **Python**, **Node.js/TypeScript**, and **.NET**.

## Start here

| Doc | Audience |
| --- | --- |
| [Getting started](getting-started.md) | First install, build, and hello-world |
| [User guide](user-guide.md) | Everyday usage: levels, fields, async, stats, lifecycle |
| [Configuration reference](configuration.md) | Full JSON schema and defaults |
| [Sinks & centralized logging](sinks.md) | Stdout, file rotation, HTTP, Loki/Grafana |

## By language

| Language | Guide |
| -------- | ----- |
| [Python](languages/python.md) | `pip` install, API, examples |
| [Node.js / TypeScript](languages/node.md) | npm package, API, examples |
| [.NET](languages/dotnet.md) | NuGet-style project, API, examples |
| [Go (direct)](languages/go.md) | Import `internal/logger` without the shared library |
| [C / other FFI](languages/c.md) | Call the C ABI from any language |

## Operators & maintainers

| Doc | Topic |
| --- | ----- |
| [Build & packaging](build.md) | Native `.so` / `.dll` / `.dylib`, CI, env vars |
| [Monorepo integration](monorepo.md) | Turborepo / npm workspaces |
| [Architecture](architecture.md) | Core, ABI, codegen, handle lifecycle |
| [ABI & codegen](abi.md) | `api/abi.json`, adding exports |
| [Troubleshooting](troubleshooting.md) | Common failures and how to fix them |

## Quick links

- Root overview: [`../README.md`](../README.md)
- Examples: [`../examples/`](../examples/)
- ABI contract: [`../api/abi.json`](../api/abi.json)
