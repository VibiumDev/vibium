/**
 * CLI Tests: the startup tab is a content page
 *
 * Firefox 159 ignores the about:blank launch argument and, unless
 * browser.startup.page is 0, starts on chrome://browser/content/blanktab.html.
 * There, viewport and page switch fail as privileged scope, and content
 * reports success but sets nothing, so the next fill waits out its timeout.
 * browsers.json pins an earlier Firefox, so only beta-watch exercises this
 * until the pin moves.
 */

const { test, describe, before, after } = require("../../helpers/capabilities").suite("core");
const assert = require('node:assert');
const { execSync } = require('node:child_process');
const { VIBIUM } = require('../../helpers');

// A fresh browser is the point, so this runs under its own session: stopping
// the shared one would leave the next file to start it headed.
const ENV = { ...process.env, VIBIUM_SESSION: `startup-tab-${process.pid}` };

function cli(args) {
  return execSync(`${VIBIUM} --headless ${args}`, { encoding: 'utf-8', timeout: 60000, env: ENV });
}

// stop exits 0 when nothing is running, so a failure here is real.
before(() => cli('stop'));
after(() => cli('stop'));

describe('CLI: the startup tab is a content page', () => {
  test('a fresh browser starts on about:blank and its tab takes commands', { timeout: 120000 }, () => {
    const pages = cli('pages');
    assert.match(pages, /^\[0\] about:blank \(page: /, pages);
    assert.doesNotMatch(pages, /^\[1\]/m, pages);

    cli(`content '<input id="n">'`);
    cli('fill "#n" "hi" --timeout 5s');
    assert.strictEqual(cli('value "#n"').trim(), 'hi');

    assert.match(cli('viewport 800 600'), /800x600/);
    cli('page new');
    assert.match(cli('page switch 0'), /switched/i);
  });
});
