# Go guide (direct core)

You can use the logger **without** the shared library by importing the Go package directly. This is ideal for Go microservices in the same monorepo.

> Note: the package currently lives under `internal/logger`. Consumers outside this module need either a public export path (future) or to live in a module that can access it. Within this repo / `replace` directives, import:

```go
import core "polyglot/internal/logger"
```

## Quick start

```go
package main

import (
	"log"

	core "polyglot/internal/logger"
)

func main() {
	cfg := core.DefaultConfig()
	cfg.Service = "demo-api"
	cfg.Environment = "prod"
	cfg.Level = "info"
	cfg.Stdout = false
	cfg.File = &core.FileConfig{
		Enabled:    true,
		Path:       "app.log",
		MaxSizeMB:  100,
		MaxBackups: 10,
		MaxAgeDays: 14,
	}
	cfg.Async = true
	cfg.Overflow = core.OverflowDropNewest

	l, err := core.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := l.Close(); err != nil {
			log.Printf("logger close: %v", err)
		}
	}()

	_ = l.SetFields(map[string]any{"traceId": "abc"})
	_ = l.Info("hello", map[string]any{"k": "v"})
	_ = l.Flush()
	log.Printf("stats: %+v", l.Stats())
}
```

Demo binary:

```bash
go run ./cmd/logger-demo
```

## HTTP-only (centralized)

```go
cfg := core.DefaultConfig()
cfg.Service = "demo-api"
cfg.Stdout = false
cfg.File = &core.FileConfig{Enabled: false}
cfg.HTTP = &core.HTTPConfig{
	Enabled:         true,
	URL:             "https://collector.example/v1/logs",
	TimeoutMS:       5000,
	BatchSize:       50,
	FlushIntervalMS: 1000,
	Headers:         map[string]string{"Authorization": "Bearer " + token},
}
```

## API highlights

| Method | Notes |
| ------ | ----- |
| `New(Config)` | Validate + open sinks |
| `Log` / `Info` / … / `Fatal` | `Fatal` does not `os.Exit` |
| `LogJSON` / `LogInt` | Used by the ABI layer |
| `SetFields` / `SetFieldsJSON` | Context |
| `ReloadConfig` | Hot reload |
| `Stats` / `StatsJSON` | Counters including `WriteErrors`, `Buffered`, `SinkDropped` |
| `Flush` / `Close` | `Close` returns flush errors |

## Tests

```bash
CGO_ENABLED=1 go test ./internal/logger -race -count=1
```

Go services that only import the core do **not** need CGO at runtime. CGO is required for the `native` shared library and for race tests that touch CGO packages.

See also: [Architecture](../architecture.md), [Configuration](../configuration.md).
