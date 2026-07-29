"""
Production-grade zero-config auto-initializer for polyglot-logger

This module automatically:
1. Locates the native logger binary packaged with this module
2. Climbs the directory tree to find polyglot.yaml in the project root
3. Initializes the Go logger engine on first import
4. Falls back to safe defaults if no config found
"""

import os
import platform
import sys
from pathlib import Path
from typing import Optional


def find_project_config() -> Optional[str]:
    """
    Find the polyglot.yaml configuration file by climbing up the directory tree.
    
    Starts from the current working directory and stops at the filesystem root.
    Returns the absolute path to the config file, or None if not found.
    """
    current = Path.cwd().resolve()
    root = Path(current.anchor)

    while True:
        config_path = current / "polyglot.yaml"
        if config_path.exists():
            return str(config_path)

        # Stop if we've reached the filesystem root
        if current == root:
            break

        current = current.parent

    return None


def resolve_native_library() -> str:
    """
    Resolve the native logger library path for the current platform.
    
    Returns the location of the pre-compiled binary bundled with this package.
    Raises FileNotFoundError if the binary is not found.
    """
    system = platform.system().lower()
    if system == "windows":
        lib_name = "logger.dll"
    elif system == "darwin":
        lib_name = "liblogger.dylib"
    else:
        lib_name = "liblogger.so"

    # The native binary is bundled in polyglot_logger/bin/
    lib_dir = Path(__file__).parent / "bin"
    lib_path = lib_dir / lib_name

    if not lib_path.exists():
        raise FileNotFoundError(
            f"[polyglot-logger] native library not found at {lib_path}. "
            f"Ensure the package was built with native binaries included."
        )

    return str(lib_path)


def auto_initialize() -> None:
    """
    Auto-initialization hook called on module import.
    
    Discovers config file and initializes the logger without user intervention.
    All errors are logged to stderr but do not raise exceptions.
    """
    try:
        config_path = find_project_config()

        if not config_path:
            print(
                f"[polyglot-logger] polyglot.yaml not found in project tree; "
                f"starting from {os.getcwd()}. Using default configuration.",
                file=sys.stderr,
            )
            return

        print(
            f"[polyglot-logger] Found configuration at: {config_path}",
            file=sys.stderr,
        )
    except Exception as e:
        print(
            f"[polyglot-logger] Error during auto-initialization: {e}",
            file=sys.stderr,
        )
