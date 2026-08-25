#!/usr/bin/env node
// Replace the bin shim with the platform binary, then download Chrome for Testing.

const { execFileSync } = require('child_process');
const path = require('path');
const os = require('os');
const { linkBinaryOverShim } = require('./scripts/link-binary');

function getVibiumBinPath() {
  const platform = os.platform();
  const arch = os.arch() === 'x64' ? 'x64' : 'arm64';
  const packageName = `@vibium/${platform}-${arch}`;
  const binaryName = platform === 'win32' ? 'vibium.exe' : 'vibium';

  try {
    const packagePath = require.resolve(`${packageName}/package.json`);
    return path.join(path.dirname(packagePath), 'bin', binaryName);
  } catch {
    // Binary not available for this platform
    return null;
  }
}

const vibiumPath = getVibiumBinPath();

// Every `vibium <cmd>` through the JS shim pays ~100ms of Node boot before
// any real work (#356). npm requires the published bin entry to be a JS
// file, so the shim ships and gets replaced with the real binary here.
// The shim must stay a script on Windows (npm's cmd shims invoke node
// explicitly) and under Yarn PnP (no node_modules layout to link into).
if (os.platform() !== 'win32' && !process.versions.pnp) {
  try {
    linkBinaryOverShim(path.join(__dirname, 'bin', 'cli.js'), vibiumPath);
  } catch (error) {
    // The shim still works, it is just slower.
    console.warn('Warning: could not replace the vibium shim with the binary:', error.message);
  }
}

if (process.env.VIBIUM_SKIP_BROWSER_DOWNLOAD === '1') {
  console.log('Skipping browser download (VIBIUM_SKIP_BROWSER_DOWNLOAD=1)');
  process.exit(0);
}

if (!vibiumPath) {
  process.exit(0);
}

try {
  console.log('Installing Chrome for Testing...');
  execFileSync(vibiumPath, ['install'], { stdio: 'inherit' });
} catch (error) {
  console.warn('Warning: Failed to install browser:', error.message);
  // Don't fail the install - user can run manually later
}
