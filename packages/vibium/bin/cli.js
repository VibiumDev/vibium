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
  // A non-zero exit (e.g. element not found) makes execFileSync throw. With
  // inherited stdio the binary has already printed its own message, so just
  // propagate the exit code — don't dump Node's child_process stack trace on
  // top of it for ordinary command failures (#161, #111).
  if (typeof err.status === 'number') process.exit(err.status);
  if (err.code === 'ENOENT') {
    console.error(`vibium binary not found at ${vibiumPath}`);
    process.exit(1);
  }
  // Killed by a signal or another spawn failure.
  console.error(err.message);
  process.exit(1);
}
