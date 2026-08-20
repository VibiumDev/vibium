const { suite } = require('../helpers/capabilities');
const { test, describe, before } = suite('core');

before(() => console.log('SUITE-BEFORE-RAN'));

test('selected test', () => {});

describe.requires('audio')('audio group', () => {
  before(() => console.log('AUDIO-BEFORE-RAN'));
  test('audio test', () => {});
});
