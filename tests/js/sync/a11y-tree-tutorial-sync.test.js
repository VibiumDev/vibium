const { describe } = require('node:test');
const { browser } = require('../../../clients/javascript/dist/sync');
const { runTutorial } = require('../helpers/tutorial-runner');

describe('A11y Tree Tutorial (JS Sync)', () => {
  runTutorial('docs/tutorials/a11y-tree-js.md', { browser, mode: 'sync' });
});
