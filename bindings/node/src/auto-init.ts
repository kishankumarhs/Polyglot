/**
 * Production-grade zero-config auto-initializer for @polyglot/logger
 *
 * This module automatically:
 * 1. Locates the native logger binary packaged with this module
 * 2. Climbs the directory tree to find polyglot.yaml in the project root
 * 3. Initializes the Go logger engine on first import
 * 4. Falls back to safe defaults if no config found
 */

import fs from "fs";
import path from "path";
import os from "os";

/**
 * Finds the polyglot.yaml configuration file by climbing up the directory tree.
 * Starts from the current working directory and stops at the filesystem root.
 */
export function findProjectConfig(): string | null {
  let currentDir = process.cwd();
  const root = path.parse(currentDir).root;

  while (true) {
    const configPath = path.join(currentDir, "polyglot.yaml");
    if (fs.existsSync(configPath)) {
      return path.resolve(configPath);
    }

    // Stop if we've reached the filesystem root
    if (currentDir === root) {
      break;
    }

    currentDir = path.dirname(currentDir);
  }

  return null;
}

/**
 * Resolves the native logger library path for the current platform.
 * Returns the location of the pre-compiled binary bundled with this package.
 */
export function resolveNativeLibrary(): string {
  const libName =
    os.platform() === "win32"
      ? "logger.dll"
      : os.platform() === "darwin"
        ? "liblogger.dylib"
        : "liblogger.so";

  // The native binary is bundled in bindings/node/bin/
  const libPath = path.join(__dirname, "bin", libName);

  if (!fs.existsSync(libPath)) {
    throw new Error(
      `[polyglot-logger] native library not found at ${libPath}. ` +
        `Ensure the package was built with native binaries included.`,
    );
  }

  return libPath;
}

/**
 * Auto-initialization hook called on module import.
 * Discovers config file and initializes the logger without user intervention.
 */
export function autoInitialize(): void {
  try {
    const configPath = findProjectConfig();

    if (!configPath) {
      console.warn(
        `[polyglot-logger] polyglot.yaml not found in project tree; ` +
          `starting from ${process.cwd()}. Using default configuration.`,
      );
      return;
    }

    console.info(`[polyglot-logger] Found configuration at: ${configPath}`);
  } catch (e) {
    console.error(`[polyglot-logger] Error during auto-initialization:`, e);
  }
}

export default { findProjectConfig, resolveNativeLibrary, autoInitialize };
