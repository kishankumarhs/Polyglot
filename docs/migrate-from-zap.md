# Migrating from Zap

Same ideas: structured fields, levels, `With` children. Different API surface — maps instead of typed `zap.Field`, config-driven sinks, optional hot reload.

```go
// Before
log, _ := zap.NewProduction()
defer log.Sync()
log.Info("hello", zap.Int("user_id", 1))

// After
import core "polyglot/internal/logger"

log, err := core.New(core.Config{
    Service: "api", Level: "info", Stdout: true, Async: true,
})
if err != nil { panic(err) }
defer log.Close()
_ = log.Info("hello", map[string]any{"user_id": 1})
```

Keep Zap in pure-Go services if you prefer. Use Polyglot where Go + Node + Python need the same schema and ops.

Notes:

- Import path is `polyglot/...` inside this repo until published as a standalone module.
- `Fatal` does not call `os.Exit`.
- Prefer `With` over mutating shared fields under concurrency.

More: [languages/go.md](languages/go.md) · [compatibility.md](compatibility.md)
