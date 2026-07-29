# Go Guide (Core Implementation)

**Repository:** [Polyglot](https://github.com/kishankumarhs/Polyglot) (polyglot-go)  
**Status:** Core implementation — all language bindings build from this

## Overview

Go is the core implementation language for Polyglot Logger. This guide covers two use cases:

1. **Direct import** — Use the Go package directly (ideal for monorepo services)
2. **Via FFI** — Use the shared library from non-Go services

This is part of a [modular monorepo](../architecture.md) where:

- **Core Go package:** The actual logger implementation
- **Shared library:** Native `.so`/`.dll`/`.dylib` compiled from Go via CGO
- **Bindings:** Node.js, Python, .NET call the shared library via FFI

See [REPOSITORIES.md](../REPOSITORIES.md) for how all repositories work together.

## Use Case 1: Direct Import (Go Microservices)

For Go services in the same repository or using `go.mod replace` directives:

```go
import core "polyglot/internal/logger"
```

### Quick Start

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
