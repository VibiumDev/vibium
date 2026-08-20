const { suite } = require('../helpers/capabilities');
const { test, describe, before, after, beforeEach } = suite('audio');

before(() => console.log('SUITE-BEFORE-RAN'));
beforeEach(() => console.log('SUITE-BEFOREEACH-RAN'));
after(() => console.log('SUITE-AFTER-RAN'));

test('skipped test', () => console.log('TEST-BODY-RAN'));

describe('nested group', () => {
  before(() => console.log('NESTED-BEFORE-RAN'));
  test('nested skipped test', () => {});
});
