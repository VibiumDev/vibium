/**
 * CLI Tests: type and press on inputs that cannot carry a caret
 *
 * Follow-on to #488. caretToEnd restores the append contract by collapsing the
 * caret, which needs a selection API. input[type=number] and [type=email] have
 * none, so typed keys landed wherever the focusing click did. The caret is
 * placed by borrowing the selection API the same element has while its type is
 * text; the date family has no caret to place at all and is refused.
 *
 * The fixture is a 100px field so the focusing click lands deep inside the
 * value and the splice is unmistakable. A default-width field is decided by
 * font metrics to within one character, so it is not safe to assert on.
 */

const { test, describe, after } = require("../../helpers/capabilities").suite("core");
const assert = require('node:assert');
const { execSync } = require('node:child_process');
const { VIBIUM } = require("../../helpers");

function cli(args) {
  return execSync(`${VIBIUM} --headless ${args}`, { encoding: 'utf-8', timeout: 30000 });
}

function cliFails(args) {
  try { cli(args); return null; } catch (e) { return `${e.stdout || ''}${e.stderr || ''}`; }
}

function page(html) { cli(`go 'data:text/html,${html}'`); }

after(() => {
  try { cli('stop'); } catch { /* no session left behind either way */ }
});

describe('CLI: type/press on inputs with no selection API (#507)', () => {
  test('type appends into number instead of splicing', () => {
    page('<input id="n" type="number" value="1234567890" style="width:100px">');
    cli(`type "#n" "99"`);
    assert.strictEqual(cli(`value "#n"`).trim(), '123456789099');
  });

  test('type appends into email instead of splicing', () => {
    page('<input id="e" type="email" value="abcdefghij" style="width:100px">');
    cli(`type "#e" "99"`);
    assert.strictEqual(cli(`value "#e"`).trim(), 'abcdefghij99');
  });

  test('press acts on the last character of a number, not the click point', () => {
    page('<input id="n" type="number" value="1234567890" style="width:100px">');
    cli(`press "Backspace" "#n"`);
    assert.strictEqual(cli(`value "#n"`).trim(), '123456789');
  });

  test('press acts on the last character of an email', () => {
    page('<input id="e" type="email" value="abcdefghij" style="width:100px">');
    cli(`press "Backspace" "#e"`);
    assert.strictEqual(cli(`value "#e"`).trim(), 'abcdefghi');
  });

  test('a segmented date input is refused, not corrupted', () => {
    page('<input id="d" type="date" value="2020-01-02">');
    const err = cliFails(`type "#d" "99"`);
    assert.ok(err, 'expected type to fail');
    assert.match(err, /segments/);
    assert.strictEqual(cli(`value "#d"`).trim(), '2020-01-02');
  });

  test('maxlength still applies, because the keys are still real', () => {
    page('<input id="m" type="email" maxlength="12" value="abcdefghij">');
    cli(`type "#m" "XYZ"`);
    assert.strictEqual(cli(`value "#m"`).trim(), 'abcdefghijXY');
  });

  test('a readonly field is left alone', () => {
    page('<input id="r" type="number" value="1234567890" readonly>');
    cli(`type "#r" "9"`);
    assert.strictEqual(cli(`value "#r"`).trim(), '1234567890');
  });

  test('an empty number input still takes the text', () => {
    page('<input id="n" type="number" value="">');
    cli(`type "#n" "99"`);
    assert.strictEqual(cli(`value "#n"`).trim(), '99');
  });

  test('text is unaffected at the same width', () => {
    page('<input id="t" type="text" value="1234567890" style="width:100px">');
    cli(`type "#t" "99"`);
    assert.strictEqual(cli(`value "#t"`).trim(), '123456789099');
  });
});
