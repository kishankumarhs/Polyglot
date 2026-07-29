import json
import sys
import threading
from pathlib import Path

import pytest

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from eximietas_logger import Logger, LoggerError, Level, abi_version, library_version


def _has_native_lib() -> bool:
    candidates = [
        ROOT / "dist" / "logger.dll",
        ROOT / "dist" / "liblogger.so",
        ROOT / "dist" / "liblogger.dylib",
        Path(__file__).resolve().parents[1] / "eximietas_logger" / "native" / "logger.dll",
        Path(__file__).resolve().parents[1] / "eximietas_logger" / "native" / "liblogger.so",
        Path(__file__).resolve().parents[1] / "eximietas_logger" / "native" / "liblogger.dylib",
    ]
    return any(p.exists() for p in candidates)


pytestmark = pytest.mark.skipif(not _has_native_lib(), reason="native logger library not built")


def test_writes_json_and_filters_levels(tmp_path: Path):
    path = tmp_path / "app.log"
    with Logger(
        "python-smoke",
        environment="test",
        level="info",
        stdout=False,
        file_path=str(path),
        async_mode=False,
        fields={"team": "platform"},
    ) as log:
        log.debug("hidden", n=1)
        log.info("hello", user_id=7)
        log.set_fields({"traceId": "t-1"})
        log.log_simple(Level.INFO, "simple")
        log.flush()
        st = log.stats()
        assert st["flushed"] >= 2

    lines = path.read_text(encoding="utf-8").strip().splitlines()
    assert len(lines) == 2
    entry = json.loads(lines[0])
    assert entry["level"] == "info"
    assert entry["message"] == "hello"
    assert entry["service_name"] == "python-smoke"
    assert entry["fields"]["team"] == "platform"
    assert entry["fields"]["user_id"] == 7
    assert abi_version() == 1
    assert library_version()


def test_concurrent_writes(tmp_path: Path):
    path = tmp_path / "concurrent.log"
    with Logger(
        "python-concurrent",
        stdout=False,
        file_path=str(path),
        level="debug",
        async_mode=True,
        overflow="block",
        queue_size=5000,
    ) as log:
        threads = []

        def worker(i: int):
            for j in range(25):
                log.info("concurrent", worker=i, n=j)

        for i in range(8):
            t = threading.Thread(target=worker, args=(i,))
            threads.append(t)
            t.start()
        for t in threads:
            t.join()
        log.flush()

    lines = path.read_text(encoding="utf-8").strip().splitlines()
    assert len(lines) == 8 * 25


def test_invalid_config():
    with pytest.raises(LoggerError):
        Logger("", stdout=True)
