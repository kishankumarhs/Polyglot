# Zero-config

Put `polyglot.yaml` in the project root. Bindings walk up from `cwd` until they find it.

```yaml
# polyglot.yaml
service: my-app
environment: prod
level: info
stdout: true
file:
  enabled: true
  path: ./logs/app.log
  max_size_mb: 100
  max_backups: 10
  max_age_days: 30
async: true
queue_size: 10000
overflow: drop_newest
fields:
  region: us-east-1
```

```python
from polyglot_logger import Logger
log = Logger("my-app")  # picks up polyglot.yaml
log.info("hello", user_id=123)
```

```js
import { Logger } from "@polyglot-logger/node";
const log = new Logger({ service: "my-app" });
log.info("hello", { user_id: 123 });
```

Constructor options override YAML. Missing or invalid config falls back to defaults (warnings go to stderr). Python needs PyYAML for YAML load: `pip install polyglot-logger[yaml]`.

Native library path (not config):

```bash
export POLYGLOT_LOGGER_LIB=/path/to/liblogger.so
```

Full schema: [configuration.md](configuration.md).
