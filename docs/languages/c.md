# C and other FFI

Call the shared library through [`native/include/logger.h`](../../native/include/logger.h).

```bash
make build-native

# Linux
gcc examples/c/main.c -I native/include -L dist -llogger -o examples/c/demo
LD_LIBRARY_PATH=dist ./examples/c/demo

# Windows (MinGW)
gcc examples/c/main.c -I native/include -L dist -llogger -o examples/c/demo.exe
```

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
    logger_info(h, "hello from c", "{\"ok\":true}");
    logger_flush(h);
    logger_close(h);
    return 0;
}
```

See [abi.md](../abi.md) for the full surface. Prefer a generated binding when one exists for your language.
