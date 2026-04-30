#!/usr/bin/env node
// Find vibium binary from platform package and run it.

const { execFileSync } = require('child_process');
const path = require('path');
const os = require('os');

function getVibiumBinPath() {
  const platform = os.platform();
  const arch = os.arch() === 'x64' ? 'x64' : 'arm64';
  const packageName = `@vibium/${platform}-${arch}`;
  const binaryName = platform === 'win32' ? 'vibium.exe' : 'vibium';

  try {
    const packagePath = require.resolve(`${packageName}/package.json`);
    return path.join(path.dirname(packagePath), 'bin', binaryName);
  } catch {
    console.error(`Could not find vibium binary for ${platform}-${arch}`);
    process.exit(1);
  }
}

const vibiumPath = getVibiumBinPath();
const args = process.argv.slice(2);
const binName = path.basename(process.argv[1] || 'vibium', path.extname(process.argv[1] || ''));
try {
  execFileSync(vibiumPath, args, { stdio: 'inherit', argv0: binName });
} catch (err) {
  // Forward the binary's exit status without dumping a Node stack trace.
  process.exit(typeof err.status === 'number' ? err.status : 1);
}
