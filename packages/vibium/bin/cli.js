#!/usr/bin/env node
// Find vibium binary from platform package and run it.

const { nodeVersionError } = require('./node-version-check');
const versionError = nodeVersionError(process.versions.node, process.execPath);
if (versionError) {
  console.error(versionError);
  process.exit(1);
}

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

// npm 12 runs no install scripts, so postinstall never links the binary
// over this shim (#678). Setup is the next step users run: link it then.
if (args.find((arg) => !arg.startsWith('-')) === 'setup') {
  const { linkBinaryOverShim, shouldReplaceShim } = require('../scripts/link-binary');
  if (shouldReplaceShim(os.platform(), process.versions.pnp, path.dirname(__dirname))) {
    try {
      linkBinaryOverShim(__filename, vibiumPath);
    } catch {
      // A read-only install keeps the shim, which still works.
    }
  }
}
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
