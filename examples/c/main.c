/*
 * Minimal C consumer example for the Polyglot logger shared library.
 *
 * Build (Linux):
 *   gcc examples/c/main.c -I native/include -L dist -llogger -o examples/c/demo
 *   LD_LIBRARY_PATH=dist ./examples/c/demo
 *
 * Build (Windows MinGW):
 *   gcc examples/c/main.c -I native/include -L dist -llogger -o examples/c/demo.exe
 */
#include <stdio.h>
#include <stdlib.h>

#include "logger.h"

int main(void) {
    const char* config =
        "{"
        "\"service\":\"c-demo\","
        "\"environment\":\"dev\","
        "\"level\":\"debug\","
        "\"stdout\":true,"
        "\"async\":false"
        "}";

    logger_handle handle = logger_create_v1(config);
    if (handle == NULL) {
        fprintf(stderr, "create failed: %s\n", logger_last_error(NULL));
        return 1;
    }

    printf("version=%s abi=%d\n", logger_version(), logger_abi_version());

    if (logger_set_fields(handle, "{\"lang\":\"c\"}") != 0) {
        fprintf(stderr, "set_fields failed: %s\n", logger_last_error(handle));
        logger_close(handle);
        return 1;
    }

    if (logger_log(handle, LOGGER_INFO, "hello from C", "{\"example\":true}") != 0) {
        fprintf(stderr, "log failed: %s\n", logger_last_error(handle));
        logger_close(handle);
        return 1;
    }

    logger_log_simple(handle, LOGGER_WARN, "simple warning");
    logger_flush(handle);
    printf("stats=%s\n", logger_stats(handle));
    logger_close(handle);
    return 0;
}
