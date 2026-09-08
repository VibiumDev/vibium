/**
 * JS Library Tests: type and press act at the end of the value
 *
 * Regression tests for #488: type and press focus by clicking the element's
 * center, and a click in a text field puts the caret at the character
 * nearest that x-coordinate. In a field narrower than its value the new
 * text was spliced into the middle at a width- and engine-dependent index,
 * and press Backspace deleted a different character per engine.
 *
 * The fixture is the report's: a 100px field holding a 26-character value,
 * so the focusing click always lands inside the text.
 */

const { test, describe, before, after } = require("../../../helpers/capabilities").suite("core");
const assert = require('node:assert');

const { browser } = require('../../../../clients/javascript/dist');

const ALPHABET = 'abcdefghijklmnopqrstuvwxyz';

let bro;

before(async () => {
  bro = await browser.start({ headless: true });
});

after(async () => {
  await bro.stop();
});

async function narrowFieldPage() {
  const vibe = await bro.page();
  await vibe.go('about:blank');
  await vibe.setContent(
    '<input id="n" type="text" style="width:100px">' +
    '<div id="ce" contenteditable style="width:100px"></div>'
  );
  return vibe;
}

describe('Type/press: caret moves to the end of the value (#488)', () => {
  test('type appends when the value is wider than the field', async () => {
    const vibe = await narrowFieldPage();
    const input = await vibe.find('#n');
    await input.fill(ALPHABET);
    await input.type('XX');
    assert.strictEqual(await input.value(), ALPHABET + 'XX');
  });

  test('press acts on the last character, whatever the field width', async () => {
    const vibe = await narrowFieldPage();
    const input = await vibe.find('#n');
    await input.fill(ALPHABET);
    await input.press('Backspace');
    assert.strictEqual(await input.value(), ALPHABET.slice(0, -1));
  });

  test('type appends in contenteditable', async () => {
    const vibe = await narrowFieldPage();
    const ce = await vibe.find('#ce');
    await ce.type(ALPHABET);
    await ce.type('XX');
    assert.strictEqual(await ce.text(), ALPHABET + 'XX');
  });

  test('type into an empty field is unchanged', async () => {
    const vibe = await narrowFieldPage();
    const input = await vibe.find('#n');
    await input.type('hello');
    assert.strictEqual(await input.value(), 'hello');
  });
});
