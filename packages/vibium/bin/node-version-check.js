// Old Node versions fail far from the real cause (#17): a Node 16 install
// surfaced as "Chrome failed to launch" with no mention of Node. Check the
// version up front and name the actual problem. Kept to older syntax so the
// check itself still parses on the versions it rejects.

var pkg = require('../package.json');

// The supported floor comes from the package's own engines field, so the two
// cannot drift apart.
var match = /\d+/.exec((pkg.engines && pkg.engines.node) || '');
var MIN_NODE_MAJOR = match ? parseInt(match[0], 10) : 18;

// nodeVersionError returns an error message when versionString (for example
// "16.20.2") is below the supported floor, else null.
function nodeVersionError(versionString, execPath) {
  var major = parseInt(String(versionString).split('.')[0], 10);
  if (isNaN(major) || major >= MIN_NODE_MAJOR) {
    return null;
  }
  return (
    'vibium requires Node.js ' + MIN_NODE_MAJOR + ' or newer; this is Node ' +
    versionString + (execPath ? ' at ' + execPath : '') + '.\n' +
    'Upgrade Node or switch versions (for example: nvm use ' + MIN_NODE_MAJOR + ') and run again.'
  );
}

module.exports = { nodeVersionError: nodeVersionError, MIN_NODE_MAJOR: MIN_NODE_MAJOR };
