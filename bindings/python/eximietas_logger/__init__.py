"""Eximietas native logger Python bindings."""

from __future__ import annotations

import ctypes
import json
import os
import platform
from pathlib import Path
from typing import Any, Mapping, Optional

from ._ffi_generated import Level, bind


class LoggerError(RuntimeError):
    """Raised when the native logger returns an error."""


def _library_name() -> str:
    system = platform.system().lower()
    if system == "windows":
        return "logger.dll"
    if system == "darwin":
        return "liblogger.dylib"
    return "liblogger.so"


def _candidate_paths() -> list[Path]:
    here = Path(__file__).resolve().parent
    name = _library_name()
    env = os.environ.get("EXIMIETAS_LOGGER_LIB")
    paths: list[Path] = []
    if env:
        paths.append(Path(env))
    paths.extend(
        [
            here / "native" / name,
            here.parent.parent / "dist" / name,
            here.parent.parent / "build" / name,
            Path.cwd() / name,
            Path.cwd() / "dist" / name,
            Path.cwd() / "build" / name,
        ]
    )
    return paths


def _load_library() -> ctypes.CDLL:
    last_error: Optional[Exception] = None
    for path in _candidate_paths():
        if not path.exists():
            continue
        try:
            if platform.system().lower() == "windows":
                raw = ctypes.WinDLL(str(path))
            else:
                raw = ctypes.CDLL(str(path))
            return bind(raw)
        except OSError as exc:
            last_error = exc
    searched = ", ".join(str(p) for p in _candidate_paths())
    raise LoggerError(
        f"unable to load native logger library {_library_name()}; searched: {searched}"
        + (f"; last error: {last_error}" if last_error else "")
    )


class _Native:
    def __init__(self) -> None:
        self.lib = _load_library()

    def last_error(self, handle: Optional[int] = None) -> str:
        raw = self.lib.logger_last_error(handle)
        if not raw:
            return ""
        if isinstance(raw, bytes):
            return raw.decode("utf-8")
        return str(raw)


_NATIVE: Optional[_Native] = None


def _native() -> _Native:
    global _NATIVE
    if _NATIVE is None:
        _NATIVE = _Native()
    return _NATIVE


def library_version() -> str:
    raw = _native().lib.logger_version()
    if isinstance(raw, bytes):
        return raw.decode("utf-8")
    return str(raw or "")


def abi_version() -> int:
    return int(_native().lib.logger_abi_version())


def _build_config(
    service: str,
    *,
    service_version: str = "",
    environment: str = "",
    level: str = "info",
    stdout: bool = True,
    file: Optional[Mapping[str, Any]] = None,
    file_path: Optional[str] = None,
    max_size_mb: int = 100,
    max_backups: int = 10,
    max_age_days: int = 30,
    compress: bool = False,
    http: Optional[Mapping[str, Any]] = None,
    async_mode: bool = True,
    queue_size: int = 10000,
    overflow: str = "drop_newest",
    fields: Optional[Mapping[str, Any]] = None,
) -> dict[str, Any]:
    config: dict[str, Any] = {
        "service": service,
        "service_version": service_version,
        "environment": environment,
        "level": level,
        "stdout": stdout,
        "async": async_mode,
        "queueSize": queue_size,
        "overflow": overflow,
        "fields": dict(fields or {}),
    }
    if file is not None:
        config["file"] = dict(file)
    elif file_path:
        config["file"] = {
            "enabled": True,
            "path": file_path,
            "maxSizeMB": max_size_mb,
            "maxBackups": max_backups,
            "maxAgeDays": max_age_days,
            "compress": compress,
        }
    if http is not None:
        config["http"] = dict(http)
    return config


class Logger:
    """Idiomatic Python wrapper around the native Eximietas logger.

    Thread-safety: log/flush/stats/set_fields/reload are safe for concurrent use
    on the same instance. close() should be called once from a single owner.
    """

    def __init__(
        self,
        service: str,
        *,
        service_name: Optional[str] = None,
        service_version: str = "",
        environment: str = "",
        level: str = "info",
        min_level: Optional[str] = None,
        stdout: bool = True,
        file: Optional[Mapping[str, Any]] = None,
        file_path: Optional[str] = None,
        max_size_mb: int = 100,
        max_backups: int = 10,
        max_age_days: int = 30,
        compress: bool = False,
        http: Optional[Mapping[str, Any]] = None,
        async_mode: bool = True,
        queue_size: int = 10000,
        overflow: str = "drop_newest",
        fields: Optional[Mapping[str, Any]] = None,
    ) -> None:
        name = service_name or service
        config = _build_config(
            name,
            service_version=service_version,
            environment=environment,
            level=min_level or level,
            stdout=stdout,
            file=file,
            file_path=file_path,
            max_size_mb=max_size_mb,
            max_backups=max_backups,
            max_age_days=max_age_days,
            compress=compress,
            http=http,
            async_mode=async_mode,
            queue_size=queue_size,
            overflow=overflow,
            fields=fields,
        )
        native = _native()
        handle = native.lib.logger_create_v1(json.dumps(config).encode("utf-8"))
        if not handle:
            raise LoggerError(native.last_error(None) or "logger_create_v1 failed")
        self._handle = handle

    def _log(self, level: Level | int, message: str, fields: Optional[Mapping[str, Any]] = None) -> None:
        if not self._handle:
            raise LoggerError("logger is closed")
        native = _native()
        payload = json.dumps(dict(fields or {})).encode("utf-8")
        rc = native.lib.logger_log(
            self._handle,
            int(level),
            message.encode("utf-8"),
            payload,
        )
        if rc != 0:
            raise LoggerError(native.last_error(self._handle) or f"logger_log({level}) failed")

    def log(self, level: Level | int, message: str, **fields: Any) -> None:
        self._log(level, message, fields)

    def log_simple(self, level: Level | int, message: str) -> None:
        if not self._handle:
            raise LoggerError("logger is closed")
        native = _native()
        rc = native.lib.logger_log_simple(self._handle, int(level), message.encode("utf-8"))
        if rc != 0:
            raise LoggerError(native.last_error(self._handle) or "logger_log_simple failed")

    def trace(self, message: str, **fields: Any) -> None:
        self._log(Level.TRACE, message, fields)

    def debug(self, message: str, **fields: Any) -> None:
        self._log(Level.DEBUG, message, fields)

    def info(self, message: str, **fields: Any) -> None:
        self._log(Level.INFO, message, fields)

    def warn(self, message: str, **fields: Any) -> None:
        self._log(Level.WARN, message, fields)

    def error(self, message: str, **fields: Any) -> None:
        self._log(Level.ERROR, message, fields)

    def fatal(self, message: str, **fields: Any) -> None:
        """Write at fatal level. Does NOT exit the process."""
        self._log(Level.FATAL, message, fields)

    def set_fields(self, fields: Mapping[str, Any]) -> None:
        if not self._handle:
            raise LoggerError("logger is closed")
        native = _native()
        rc = native.lib.logger_set_fields(
            self._handle, json.dumps(dict(fields)).encode("utf-8")
        )
        if rc != 0:
            raise LoggerError(native.last_error(self._handle) or "logger_set_fields failed")

    def reload_config(self, config: Mapping[str, Any]) -> None:
        if not self._handle:
            raise LoggerError("logger is closed")
        native = _native()
        rc = native.lib.logger_reload_config(
            self._handle, json.dumps(dict(config)).encode("utf-8")
        )
        if rc != 0:
            raise LoggerError(native.last_error(self._handle) or "logger_reload_config failed")

    def stats(self) -> dict[str, Any]:
        if not self._handle:
            raise LoggerError("logger is closed")
        native = _native()
        raw = native.lib.logger_stats(self._handle)
        if not raw:
            return {}
        text = raw.decode("utf-8") if isinstance(raw, bytes) else str(raw)
        return json.loads(text)

    def flush(self) -> None:
        if not self._handle:
            return
        native = _native()
        if native.lib.logger_flush(self._handle) != 0:
            raise LoggerError(native.last_error(self._handle) or "logger_flush failed")

    def close(self) -> None:
        if not self._handle:
            return
        native = _native()
        handle = self._handle
        self._handle = 0
        if native.lib.logger_close(handle) != 0:
            raise LoggerError(native.last_error(None) or "logger_close failed")

    def __enter__(self) -> "Logger":
        return self

    def __exit__(self, exc_type, exc, tb) -> None:
        self.close()

    def __del__(self) -> None:
        try:
            self.close()
        except Exception:
            pass


__all__ = ["Logger", "LoggerError", "Level", "library_version", "abi_version"]
