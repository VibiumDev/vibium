const { describe } = require("../../../helpers/capabilities").suite("core");
const { browser } = require('../../../../clients/javascript/dist');
const { runTutorial } = require('../../helpers/tutorial-runner');

describe('A11y Tree Tutorial (JS Async)', () => {
  runTutorial('docs/how-to-guides/accessibility-tree.md', { browser, mode: 'async' });
});
