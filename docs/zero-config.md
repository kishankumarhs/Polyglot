# Zero-Config Project-Wise Shared Configuration

Polyglot Logger now supports **automatic project-wide configuration discovery**. This means you can place a single `polyglot.yaml` file in your project root, and all language bindings will automatically detect and apply it—without any manual configuration in your code.

## Quick Start

### 1. Create `polyglot.yaml` in Your Project Root

```yaml
service: my-app
environment: prod
level: info
stdout: true
file:
  enabled: true
  path: ./logs/app.log
  max_size_mb: 100
  maxBackups: 10
  maxAgeDays: 30
http:
  enabled: true
  url: https://loki.example.com/v1/logs
  batch_size: 50
  flush_interval_ms: 1000
async: true
queue_size: 10000
overflow: drop_newest
fields:
  region: us-east-1
```

### 2. Use Any Language Binding—No Code Configuration Needed

#### Python

```python
# No manual logger creation needed! The binding auto-initializes from polyglot.yaml
from polyglot_logger import Logger

logger = Logger("my-app")  # Will use config from polyglot.yaml
logger.info("Hello from Python", {"user_id": 123})
```

#### Node.js / TypeScript

```javascript
// Auto-initialization happens on import
import { Logger } from "@polyglot/logger";

const logger = new Logger({ service: "my-app" }); // Uses polyglot.yaml
logger.info("Hello from Node.js", { user_id: 123 });
```

#### Go (Direct)

```go
import "polyglot/internal/logger"

// Use the Go API directly; bindings auto-discover config
cfg, _ := logger.LoadConfigFromFile("polyglot.yaml")
log, _ := logger.New(cfg)
defer log.Close()

log.Info("Hello from Go", nil)
```

## Configuration Search Algorithm

The bindings use an **upward-climbing search** algorithm:

1. Start from the **current working directory** (`pwd()` / `process.cwd()` / `os.getcwd()`)
2. Look for a file named `polyglot.yaml`
3. If found, load and apply it; if not, move to the **parent directory**
4. Continue climbing until:
   - ✅ Config file is found (use it)
   - ❌ Filesystem root is reached (use defaults)

### Example Search Path

```
/home/user/project/
├── polyglot.yaml          ← FOUND HERE
├── services/
│   └── api/
│       └── main.py       ← Starting from here, climbs up
└── configs/
    └── app.yaml          ← NOT searched (looks for polyglot.yaml only)
```

When you run `python services/api/main.py`, the auto-finder climbs to `/home/user/project/` and finds `polyglot.yaml`.

## Configuration Format

The `polyglot.yaml` file uses the same schema as the JSON config but in YAML format. Both are **automatically detected** based on file extension.

### Full YAML Schema

```yaml
# Required
service: string

# Optional metadata
service_version: string
environment: string

# Logging
level: trace | debug | info | warn | error | fatal
stdout: bool
async: bool
queue_size: integer
overflow: drop_newest | drop_oldest | block

# File sink
file:
  enabled: bool
  path: string
  max_size_mb: integer
  max_backups: integer
  max_age_days: integer
  compress: bool

# HTTP/centralized logging
http:
  enabled: bool
  url: string
  timeout_ms: integer
  headers:
    key: value
  batch_size: integer
  flush_interval_ms: integer

# Custom context fields (merged into every log entry)
fields:
  region: string
  cluster: string
  # ... any key-value pairs
```

## Defensive Error Handling

All bindings implement **safe, non-crashing error handling**:

| Scenario                           | Behavior                                   |
| ---------------------------------- | ------------------------------------------ |
| **Config file not found**          | Use safe defaults (graceful degradation)   |
| **Invalid YAML syntax**            | Log warning to stderr, use defaults        |
| **Missing required field**         | Validate and report error, use defaults    |
| **Permission denied**              | Log error, use defaults                    |
| **Panic/exception in native code** | Caught, logged, and returned as error code |

**No exception will crash your application.** All errors are logged to stderr for debugging.

### Python: Missing PyYAML

If PyYAML is not installed:

```
[polyglot-logger] PyYAML not installed; skipping config file auto-load.
Install it with: pip install pyyaml
```

The logger will still work with defaults.

## Overriding Auto-Config

You can override the auto-discovered config in code if needed:

### Python

```python
from polyglot_logger import Logger

# Config from polyglot.yaml is auto-applied first
logger = Logger(
    "my-app",
    level="debug",  # Override the yaml setting
    file_path="/custom/path.log"
)
```

### Node.js

```javascript
import { Logger } from "@polyglot/logger";

// Override specific settings
const logger = new Logger({
  service: "my-app",
  level: "debug", // Override
  file: {
    enabled: true,
    path: "/custom/path.log",
  },
});
```

## Environment Variable Fallback

If you prefer explicit paths for the native library (not config):

```bash
export POLYGLOT_LOGGER_LIB=/path/to/liblogger.so
python app.py
```

This env var affects **only the binary location**, not config discovery.

## Examples

### Development Config (polyglot.yaml)

```yaml
service: myapp
environment: dev
level: debug
stdout: true
file:
  enabled: false
async: false # Sync for easier debugging
fields:
  region: local
  version: dev
```

### Production Config (polyglot.yaml)

```yaml
service: myapp
environment: prod
level: info
stdout: false
file:
  enabled: true
  path: /var/log/myapp/production.log
  max_size_mb: 500
  max_backups: 30
  max_age_days: 90
http:
  enabled: true
  url: https://logs.prod.company.com/v1/logs
  batch_size: 100
  flush_interval_ms: 2000
fields:
  region: us-east-1
  cluster: prod
```

### Multi-Service Monorepo (project root)

```
/monorepo/
├── polyglot.yaml         ← Shared config
├── services/
│   ├── auth/
│   │   └── main.py
│   ├── payments/
│   │   └── index.js
│   └── api/
│       └── main.go
```

All three services auto-discover the shared `polyglot.yaml` from their current directory and climb upward.

## Implementation Details

### Go Core (`internal/logger/config.go`)

- New function: `LoadConfigFromFile(path string) (Config, error)`
- Supports both JSON and YAML (auto-detected by extension)
- Graceful fallback to defaults on missing files

### Native Export (`native/export.go`)

- New export: `logger_create_from_config_file(configPath *C.char) void*`
- Wrapped in `defer recover()` for panic safety
- Logs structured warnings to stderr on errors

### Node.js Binding (`bindings/node/src/index.ts`)

- New function: `findProjectConfig(): string | null`
- Climbs from `process.cwd()` to filesystem root
- Auto-calls on module import via `initializeFromProjectConfig()`

### Python Binding (`bindings/python/polyglot_logger/__init__.py`)

- New function: `_find_project_config(): Optional[str]`
- Uses `pathlib.Path` for cross-platform support
- Auto-calls on module import via `_initialize_from_project_config()`

## Troubleshooting

### "Config file not found: ..."

- Check the file exists at expected path
- Verify it's named exactly `polyglot.yaml` (case-sensitive on Linux/macOS)
- Run from a directory within the project tree

### "Failed to initialize from config file: ..."

- Check YAML syntax: use a YAML validator
- Ensure `service` field is present and non-empty
- Check file permissions (readable by the application)

### "PyYAML not installed"

- Install with: `pip install pyyaml`
- Or use JSON format instead: `polyglot.json`

### Logger still using defaults

- Verify `polyglot.yaml` exists in an ancestor directory
- Check stderr for logged errors
- Try setting absolute path explicitly if issues persist

## Migration Guide

### Before (Manual Config)

```python
from polyglot_logger import Logger

config_file = os.path.join(os.path.dirname(__file__), "../config.json")
logger = Logger("my-app", config_file=config_file)
```

### After (Zero-Config)

```python
from polyglot_logger import Logger

# Just import! Config auto-discovered from polyglot.yaml
logger = Logger("my-app")
```

No changes to how you call `logger.info()`, `logger.debug()`, etc.
