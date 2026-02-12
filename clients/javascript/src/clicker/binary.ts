import fs from 'fs';
import path from 'path';
import { createRequire } from 'module';
import { getPlatform, getArch } from './platform';

/**
 * Resolve a package's location using either require.resolve (CJS)
 * or createRequire (ESM-compatible fallback).
 */
function resolvePackageJson(packageName: string): string | null {
  const specifier = `${packageName}/package.json`;

  // Try native require.resolve first (works in CJS context)
  try {
    if (typeof require !== 'undefined' && typeof require.resolve === 'function') {
      return require.resolve(specifier);
    }
  } catch {
    // Not found via native require
  }

  // Fallback: createRequire from cwd (works in ESM context)
  try {
    const esmRequire = createRequire(path.join(process.cwd(), 'package.json'));
    return esmRequire.resolve(specifier);
  } catch {
    // Not found via createRequire either
  }

  return null;
}

/**
 * Resolve the path to the clicker binary.
 *
 * Search order:
 * 1. CLICKER_PATH environment variable
 * 2. Platform-specific npm package (@vibium/{platform}-{arch})
 * 3. Local development paths (relative to cwd)
 */
export function getClickerPath(): string {
  // 1. Check environment variable
  const envPath = process.env.CLICKER_PATH;
  if (envPath && fs.existsSync(envPath)) {
    return envPath;
  }

  const platform = getPlatform();
  const arch = getArch();
  const packageName = `@vibium/${platform}-${arch}`;
  const binaryName = platform === 'win32' ? 'clicker.exe' : 'clicker';

  // 2. Check platform-specific npm package
  const packageJsonPath = resolvePackageJson(packageName);
  if (packageJsonPath) {
    const packageDir = path.dirname(packageJsonPath);
    const binaryPath = path.join(packageDir, 'bin', binaryName);

    if (fs.existsSync(binaryPath)) {
      return binaryPath;
    }
  }

  // 3. Check local development paths (relative to cwd)
  const localPaths = [
    // From vibium/ root
    path.resolve(process.cwd(), 'clicker', 'bin', binaryName),
    // From clients/javascript/
    path.resolve(process.cwd(), '..', '..', 'clicker', 'bin', binaryName),
  ];

  for (const localPath of localPaths) {
    if (fs.existsSync(localPath)) {
      return localPath;
    }
  }

  throw new Error(
    `Could not find clicker binary. ` +
    `Set CLICKER_PATH environment variable or install ${packageName}`
  );
}
