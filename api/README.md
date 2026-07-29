# ABI codegen

`api/abi.json` is the contract for the public C API.

```bash
go run ./cmd/codegen
# or: make codegen
```

This regenerates the C header and Python / Node / .NET FFI layers. Do not hand-edit generated files marked `DO NOT EDIT`.

**Full documentation:** [docs/abi.md](../docs/abi.md) · [Architecture](../docs/architecture.md)
