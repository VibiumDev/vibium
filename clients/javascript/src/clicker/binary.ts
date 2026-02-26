import fs from 'fs';
import path from 'path';
import { getPlatform, getArch } from './platform';

/**
 * Resolve the path to the vibium binary.
 *
 * Search order:
 * 1. VIBIUM_BIN_PATH environment variable
 * 2. Platform-specific npm package (@vibium/{platform}-{arch})
 * 3. Local development paths (relative to cwd)
 */
export function getVibiumBinPath(): string {
  // 1. Check environment variable
  const envPath = process.env.VIBIUM_BIN_PATH;
  if (envPath && fs.existsSync(envPath)) {
    return envPath;
  }

  const platform = getPlatform();
  const arch = getArch();
  const packageName = `@vibium/${platform}-${arch}`;
  const binaryName = platform === 'win32' ? 'vibium.exe' : 'vibium';

  // 2. Check platform-specific npm package
  try {
    const packagePath = require.resolve(`${packageName}/package.json`);
    const packageDir = path.dirname(packagePath);
    const binaryPath = path.join(packageDir, 'bin', binaryName);

    if (fs.existsSync(binaryPath)) {
      return binaryPath;
    }
  } catch {
    // Package not installed, continue to fallback
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
    `Could not find vibium binary. ` +
    `Set VIBIUM_BIN_PATH environment variable or install ${packageName}`
  );
}
