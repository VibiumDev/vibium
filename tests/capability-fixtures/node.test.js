const { suite } = require('../helpers/capabilities');
const { test } = suite('core');

test('selected core test', () => {});
test.requires('audio')('no-engine capability', () => {});
test.requires('core', 'audio')('AND requirements', () => {});
