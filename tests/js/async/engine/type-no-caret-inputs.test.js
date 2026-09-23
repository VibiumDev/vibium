/**
 * JS Library Tests: type and press on inputs that cannot carry a caret
 *
 * Same regression as the CLI suite of the same name (#507), through the client
 * router — handleVibiumType — rather than TypeInto. Both reach the shared
 * caretToEnd, and this pins that the language clients get the fix too.
 */

const { test, describe, before, after } = require("../../../helpers/capabilities").suite("core");
const assert = require('node:assert');

const { browser } = require('../../../../clients/javascript/dist');

let bro;

before(async () => { bro = await browser.start({ headless: true }); });
after(async () => { await bro.stop(); });

async function page(html) {
  const vibe = await bro.page();
  await vibe.go('about:blank');
  await vibe.setContent(html);
  return vibe;
}

describe('Type/press on inputs with no selection API (#507)', () => {
  test('type appends into number instead of splicing', async () => {
    const vibe = await page('<input id="n" type="number" value="1234567890" style="width:100px">');
    const input = await vibe.find('#n');
    await input.type('99');
    assert.strictEqual(await input.value(), '123456789099');
  });

  test('type appends into email instead of splicing', async () => {
    const vibe = await page('<input id="e" type="email" value="abcdefghij" style="width:100px">');
    const input = await vibe.find('#e');
    await input.type('99');
    assert.strictEqual(await input.value(), 'abcdefghij99');
  });

  test('press acts on the last character, not the click point', async () => {
    const vibe = await page('<input id="n" type="number" value="1234567890" style="width:100px">');
    const input = await vibe.find('#n');
    await input.press('Backspace');
    assert.strictEqual(await input.value(), '123456789');
  });

  test('a segmented date input is refused, not corrupted', async () => {
    const vibe = await page('<input id="d" type="date" value="2020-01-02">');
    const input = await vibe.find('#d');
    await assert.rejects(() => input.type('99'));
    assert.strictEqual(await input.value(), '2020-01-02');
  });
});
