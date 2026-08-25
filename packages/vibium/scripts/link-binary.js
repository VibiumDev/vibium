// Replace the published JS bin shim with the platform binary, so running
// `vibium <cmd>` executes the binary directly instead of booting Node first:
// the shim costs ~100ms of runtime start per invocation, which dominates
// scripted CLI use (#356). npm's bin links (global bin, node_modules/.bin,
// npx) all point at the shim's path, so changing the file behind that path is
// what switches every entry point at once. Same approach as esbuild.

const fs = require('fs');

// linkBinaryOverShim replaces shimPath with binaryPath via an atomic rename,
// so the bin path always resolves to either the shim or the finished binary.
// Returns false without touching anything when the binary is absent.
function linkBinaryOverShim(shimPath, binaryPath) {
  if (!binaryPath || !fs.existsSync(binaryPath)) {
    return false;
  }
  const tmp = shimPath + '.link-tmp';
  fs.rmSync(tmp, { force: true });
  try {
    // Hard link when possible; both files live in the same package tree, so
    // the same filesystem is the normal case. Copy as the fallback.
    try {
      fs.linkSync(binaryPath, tmp);
    } catch {
      fs.copyFileSync(binaryPath, tmp);
    }
    fs.chmodSync(tmp, 0o755);
    fs.renameSync(tmp, shimPath);
    return true;
  } catch (error) {
    fs.rmSync(tmp, { force: true });
    throw error;
  }
}

module.exports = { linkBinaryOverShim };
