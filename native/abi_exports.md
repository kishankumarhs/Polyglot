# ABI export checklist

Generated from `api/abi.json`. Each function must have a matching
`//export <name>` implementation in [`native/export.go`](export.go).

| Function | Returns | Args |
|----------|---------|------|
| `logger_version` | `string` |  |
| `logger_abi_version` | `int` |  |
| `logger_create_v1` | `handle` | string config_json |
| `logger_create` | `handle` | string config_json |
| `logger_log` | `int` | handle handle, int level, string message, string fields_json |
| `logger_log_simple` | `int` | handle handle, int level, string message |
| `logger_set_fields` | `int` | handle handle, string fields_json |
| `logger_reload_config` | `int` | handle handle, string config_json |
| `logger_flush` | `int` | handle handle |
| `logger_close` | `int` | handle handle |
| `logger_stats` | `string` | handle handle |
| `logger_last_error` | `string` | handle handle |
| `logger_free_string` | `void` | string s |

After editing `api/abi.json` or `native/export.go`, run:

```bash
go run ./cmd/codegen
```
