/**
 * CLI Tests: scroll targeting
 * The wheel must land at the real viewport center, not a fixed coordinate:
 * a fixed (400, 300) is out of bounds on small viewports (Firefox rejects
 * it, #444) and lands inside whatever scrollable element covers it on
 * larger ones (#443).
 */

const { test, describe } = require("../../helpers/capabilities").suite("core");
const assert = require('node:assert');
const { execSync } = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { VIBIUM } = require("../../helpers");

const tmpFile = path.join(os.tmpdir(), `vibium-scroll-${process.pid}.html`);
// A small scrollable box positioned over the old fixed point (400, 300),
// with a tall document behind it.
const html =
  '<html><body style="margin:0;height:4000px">' +
  '<div id="thief" style="position:fixed;left:350px;top:250px;width:100px;height:100px;overflow:auto">' +
  '<div style="height:1000px">tall</div></div>' +
  '</body></html>';

function run(cmd) {
  return execSync(`${VIBIUM} ${cmd}`, { encoding: 'utf-8', timeout: 30000 });
}

function evalNum(expr) {
  return parseInt(run(`eval "${expr}"`), 10);
}

// Wheel scrolling settles asynchronously; poll instead of reading once.
function pollNum(expr, ok, label) {
  const deadline = Date.now() + 5000;
  let value = NaN;
  while (Date.now() < deadline) {
    value = evalNum(expr);
    if (ok(value)) return value;
  }
  assert.fail(`${label}: last value ${value}`);
}

describe('CLI: scroll targets the viewport center (#443, #444)', () => {
  test('setup: write fixture', () => {
    fs.writeFileSync(tmpFile, html);
  });

  test('scroll works on a small viewport instead of going out of bounds', () => {
    run(`go "file://${tmpFile}?small"`);
    run('viewport 375 812');
    run('scroll down');
    pollNum('window.scrollY', (v) => v > 0, 'document should have scrolled');
  });

  test('a scrollable element over the old fixed point no longer steals the scroll', () => {
    run(`go "file://${tmpFile}?steal"`);
    run('viewport 1200 800');
    run('scroll down');
    pollNum('window.scrollY', (v) => v > 0, 'document should have scrolled');
    const thiefY = evalNum("document.getElementById('thief').scrollTop");
    assert.strictEqual(thiefY, 0, 'the element over the old fixed point must not have been scrolled');
  });

  test('--selector still scrolls within the named element', () => {
    run(`go "file://${tmpFile}?selector"`);
    run('viewport 1200 800');
    run('scroll down --selector "#thief"');
    pollNum("document.getElementById('thief').scrollTop", (v) => v > 0, 'named element should have scrolled');
  });

  test('cleanup: remove fixture', () => {
    fs.rmSync(tmpFile, { force: true });
  });
});
