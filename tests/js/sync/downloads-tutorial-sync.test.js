const { describe } = require('node:test');
const { runTutorial } = require('../helpers/tutorial-runner');

const vibium = require('../../../clients/javascript/dist');
const vibiumSync = require('../../../clients/javascript/dist/sync');

function tutorialRequire(id) {
  if (id === 'vibium') return vibium;
  if (id === 'vibium/sync') return vibiumSync;
  return require(id);
}

const SERVER_CODE = `
if (req.url === '/file') {
  res.writeHead(200, {
    'Content-Type': 'text/plain',
    'Content-Disposition': 'attachment; filename="hello.txt"',
  });
  res.end('hello world');
  return;
}
res.writeHead(200, { 'Content-Type': 'text/html' });
res.end('<a href="/file" id="dl-link">Download hello.txt</a>');
`;

describe('Downloads Tutorial (JS Sync)', () => {
  runTutorial('docs/tutorials/downloads-js.md', {
    mode: 'sync', standalone: true, serverCode: SERVER_CODE,
    requireFn: tutorialRequire,
  });
});
