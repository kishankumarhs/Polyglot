# C and other FFI languages

Any language that can call a C shared library can use the logger through [`native/include/logger.h`](../../native/include/logger.h).

## Build against the library

```bash
make build-native

# Linux
gcc examples/c/main.c -I native/include -L dist -llogger -o examples/c/demo
LD_LIBRARY_PATH=dist ./examples/c/demo

# Windows (MinGW)
gcc examples/c/main.c -I native/include -L dist -llogger -o examples/c/demo.exe
```

## Minimal usage

```c
#include "logger.h"
#include <stdio.h>

int main(void) {
    const char* config =
        "{"
        "\"service\":\"c-demo\","
        "\"environment\":\"dev\","
        "\"level\":\"info\","
        "\"stdout\":true,"
        "\"async\":true"
        "}";

    logger_handle h = logger_create_v1(config);
    if (!h) {
        fprintf(stderr, "%s\n", logger_last_error(NULL));
        return 1;
    }

    logger_set_fields(h, "{\"lang\":\"c\"}");
    if (logger_log(h, LOGGER_INFO, "hello", "{\"ok\":true}") != 0) {
        fprintf(stderr, "%s\n", logger_last_error(h));
    }
    logger_log_simple(h, LOGGER_WARN, "simple warning");
    logger_flush(h);
    printf("stats=%s\n", logger_stats(h));
    logger_close(h);
    return 0;
}
```

## Rules of the ABI

- Treat `logger_handle` as **opaque** — never dereference it.
- Levels are integers (`LOGGER_TRACE` … `LOGGER_FATAL`).
- Functions return `0` on success, `-1` on failure (except create → `NULL`, and string getters).
- Pointers from `logger_stats` / `logger_last_error` / `logger_version` are **owned by the library**. Do not free them with `free`; `logger_free_string` is a documented no-op for binding convenience.
- Call `logger_close` exactly once per successful create.

Full function table: [ABI & codegen](../abi.md). Config JSON: [Configuration](../configuration.md).

## Other languages (Rust, Java, Ruby, …)

1. Load `liblogger.so` / `logger.dll` / `liblogger.dylib`
2. Bind the symbols from `logger.h`
3. Pass UTF-8 C strings for config, messages, and JSON fields
4. Check return codes and `logger_last_error`

You do not need the Python/Node/.NET packages unless you want their ergonomic wrappers.
