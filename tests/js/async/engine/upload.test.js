/**
 * JS Library Tests: setFiles path validation and error delivery
 *
 * Regression tests for #480 (a nonexistent path was accepted and Chrome
 * attached a zero-byte file named after it) and #481 (an engine rejection
 * of setFiles never reached callers of the language clients).
 */

const { test, describe, before, after } = require("../../../helpers/capabilities").suite("core");
const assert = require('node:assert');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

const { browser } = require('../../../../clients/javascript/dist');

let bro, tmpdir;

before(async () => {
  tmpdir = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-upload-'));
  bro = await browser.start({ headless: true });
});

after(async () => {
  await bro.stop();
  fs.rmSync(tmpdir, { recursive: true, force: true });
});

async function fileInputPage() {
  const vibe = await bro.page();
  await vibe.go('about:blank');
  await vibe.setContent('<input id="f" type="file"><input id="m" type="file" multiple>');
  return vibe;
}

describe('Upload: setFiles path validation', () => {
  test('a nonexistent path rejects instead of attaching a zero-byte file', async () => {
    const vibe = await fileInputPage();
    const input = await vibe.find('#f');

    await assert.rejects(
      input.setFiles([path.join(tmpdir, 'definitely-not-here.pdf')]),
      /does not exist/
    );

    const count = await vibe.evaluate('document.querySelector("#f").files.length');
    assert.strictEqual(Number(count), 0, 'nothing may be attached on a rejected call');
  });

  test('a directory rejects', async () => {
    const vibe = await fileInputPage();
    const input = await vibe.find('#f');
    await assert.rejects(input.setFiles([tmpdir]), /is a directory/);
  });

  test('one bad path in a multi-file call rejects the whole call', async () => {
    const real = path.join(tmpdir, 'real.txt');
    fs.writeFileSync(real, 'content');

    const vibe = await fileInputPage();
    const input = await vibe.find('#m');
    await assert.rejects(
      input.setFiles([real, path.join(tmpdir, 'missing.txt')]),
      /does not exist/
    );

    const count = await vibe.evaluate('document.querySelector("#m").files.length');
    assert.strictEqual(Number(count), 0, 'a half-filled input hides the failure');
  });

  test('a real file still uploads with its content', async () => {
    const real = path.join(tmpdir, 'attachment.txt');
    fs.writeFileSync(real, 'seven b');

    const vibe = await fileInputPage();
    const input = await vibe.find('#f');
    await input.setFiles([real]);

    const got = await vibe.evaluate(
      'JSON.stringify([...document.querySelector("#f").files].map(f => ({name: f.name, size: f.size})))'
    );
    assert.deepStrictEqual(JSON.parse(got), [{ name: 'attachment.txt', size: 7 }]);
  });
});
