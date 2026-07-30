# First log

Get a structured log line without compiling Go. Current publish line is **0.3.x** (check the registry for the latest patch: `npm view @polyglot-logger/node version`, PyPI, or NuGet). Native libs ship inside the package for Linux, Windows, and macOS Apple Silicon.

Intel Macs: arm64 is what we ship today. For amd64 you may need to set `POLYGLOT_LOGGER_LIB` until CI covers Intel again.

## Node.js

```bash
mkdir demo-node && cd demo-node
npm init -y
npm install @polyglot-logger/node
# or pin the current line: npm install @polyglot-logger/node@^0.3.0
```

```js
// index.mjs
import { Logger } from "@polyglot-logger/node";

const log = new Logger({ service: "demo", environment: "dev", stdout: true });
log.info("hello from node", { user_id: 1 });
log.flush();
log.close();
```

```bash
node index.mjs
```

## Python

```bash
python -m venv .venv
source .venv/bin/activate   # Windows: .venv\Scripts\activate
pip install polyglot-logger
```

```python
# main.py
from polyglot_logger import Logger

with Logger("demo", environment="dev", stdout=True) as log:
    log.info("hello from python", user_id=1)
```

```bash
python main.py
```

## .NET

```bash
dotnet new console -n DemoDotnet -o demo-dotnet
cd demo-dotnet
dotnet add package Polyglot.Logger
```

```csharp
// Program.cs
using Polyglot.Logger;

using var log = new Logger(new LoggerOptions
{
    Service = "demo",
    Environment = "dev",
    Stdout = true,
});
log.Info("hello from dotnet", new Dictionary<string, object?> { ["user_id"] = 1 });
```

```bash
dotnet run
```

## Shared config (optional)

```yaml
# polyglot.yaml
service: demo
environment: dev
level: info
stdout: true
stdout_format: text
```

Bindings resolve `POLYGLOT_CONFIG` → cwd → parents (stop at `.git`). See [zero-config.md](zero-config.md) · [sdk.md](sdk.md).

## Stuck?

```bash
go run ./cmd/polyglot doctor
# or: make doctor
```

| Symptom | Fix |
| --- | --- |
| `unable to load native logger library` | Package 0.3.x, or set `POLYGLOT_LOGGER_LIB` to a built native lib |
| Empty / wrong config | Put `polyglot.yaml` at the git repo root, or set `POLYGLOT_CONFIG`; check startup diagnostics on stderr |
| Need a local diagnose | `go run ./cmd/polyglot doctor` |

## Working on the core?

```bash
git clone --recurse-submodules https://github.com/kishankumarhs/Polyglot.git
cd Polyglot && make build-native
export POLYGLOT_LOGGER_LIB=$PWD/dist/liblogger.so   # .dll / .dylib on other OSes
```

Then see [examples/](../examples/README.md).
