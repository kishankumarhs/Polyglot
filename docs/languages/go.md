# Go

Core lives here. Import the package directly, or call the shared library via FFI from other languages.

## Direct import

```go
import core "polyglot/internal/logger"

cfg := core.DefaultConfig()
cfg.Service = "demo-api"
cfg.Environment = "prod"
cfg.Level = "info"
cfg.Stdout = false
cfg.File = &core.FileConfig{
	Enabled: true, Path: "app.log", MaxSizeMB: 100, MaxBackups: 10, MaxAgeDays: 14,
}
cfg.Async = true
cfg.Overflow = core.OverflowDropNewest

l, err := core.New(cfg)
if err != nil {
	log.Fatal(err)
}
defer l.Close()

_ = l.SetFields(map[string]any{"traceId": "abc"})
_ = l.Info("hello", map[string]any{"k": "v"})
_ = l.Flush()
```

```bash
go run ./cmd/logger-demo
```

## HTTP-only

```go
cfg := core.DefaultConfig()
cfg.Service = "demo-api"
cfg.Stdout = false
cfg.File = &core.FileConfig{Enabled: false}
cfg.HTTP = &core.HTTPConfig{
	Enabled: true,
	URL: "https://collector.example/v1/logs",
	TimeoutMS: 5000,
	BatchSize: 50,
	FlushIntervalMS: 1000,
	Headers: map[string]string{"Authorization": "Bearer " + token},
}
```

## API

| Method | Notes |
| --- | --- |
| `New(Config)` | Validate + open sinks |
| `Log` / `Info` / … / `Fatal` | `Fatal` does not `os.Exit` |
| `With` | Child logger |
| `SetFields` / `ReloadConfig` | Context / hot reload |
| `Stats` / `Flush` / `Close` | `Close` returns flush errors |

## Tests

```bash
CGO_ENABLED=1 go test ./internal/logger -race -count=1
```

Importing the core does not need CGO at runtime. CGO is for the `native` shared library and race tests that touch CGO packages.

See [architecture.md](../architecture.md) · [configuration.md](../configuration.md).
