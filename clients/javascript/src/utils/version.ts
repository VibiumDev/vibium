const MIN_NODE_MAJOR = 18;

/**
 * Check that the current Node.js version meets Vibium's minimum requirement.
 * Throws a clear, actionable error if the version is too old.
 */
export function checkNodeVersion(): void {
  const major = parseInt(process.versions.node.split('.')[0], 10);
  if (major < MIN_NODE_MAJOR) {
    throw new Error(
      `Vibium requires Node.js >= ${MIN_NODE_MAJOR}.0.0, but you are running Node.js ${process.versions.node}. ` +
        `Please upgrade Node.js: https://nodejs.org/`
    );
  }
}
