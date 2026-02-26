#!/usr/bin/env node
// Run vibium install to download Chrome for Testing

if (process.env.VIBIUM_SKIP_BROWSER_DOWNLOAD === '1') {
  console.log('Skipping browser download (VIBIUM_SKIP_BROWSER_DOWNLOAD=1)');
  process.exit(0);
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
    // Binary not available for this platform - skip silently
    process.exit(0);
  }
}

try {
  const vibiumPath = getVibiumBinPath();
  console.log('Installing Chrome for Testing...');
  execFileSync(vibiumPath, ['install'], { stdio: 'inherit' });
} catch (error) {
  console.warn('Warning: Failed to install browser:', error.message);
  // Don't fail the install - user can run manually later
}
