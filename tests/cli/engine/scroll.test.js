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

// An outer page with a tall body and a small scrollable iframe far below the
// fold, so the frame's own viewport center maps to a point over the outer
// document (#511).
const innerFile = path.join(os.tmpdir(), `vibium-scroll-inner-${process.pid}.html`);
const innerHtml =
  '<html><body><h1>Inner</h1><div style="height:2000px">tall inner</div></body></html>';
const outerFile = path.join(os.tmpdir(), `vibium-scroll-outer-${process.pid}.html`);
const outerHtml =
  '<html><body><h1>Outer</h1><div style="height:2000px">tall</div>' +
  `<iframe name="inner" src="file://${innerFile}" width="300" height="100"></iframe>` +
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

describe('CLI: scroll reaches the active document (#511)', () => {
  test('setup: write frame fixtures', () => {
    fs.writeFileSync(innerFile, innerHtml);
    fs.writeFileSync(outerFile, outerHtml);
  });

  test('scroll inside a frame moves the frame, not the page behind it', () => {
    run(`go "file://${outerFile}"`);
    run('frame inner');
    // Confirm the frame is scrollable and we are inside it.
    pollNum('window.top !== window.self ? 1 : 0', (v) => v === 1, 'should be inside the frame');

    run('scroll down');
    run('scroll down');
    // The frame scrolled...
    pollNum('window.scrollY', (v) => v > 0, 'the frame should have scrolled');

    // ...and the outer page did not. Before the fix the wheel landed on the
    // outer document (Chrome scrolled it, Firefox dropped the event) while the
    // frame stayed at 0.
    run('page switch 0');
    const outerY = evalNum('window.scrollY');
    assert.strictEqual(outerY, 0, 'the page behind the frame must not have moved');
  });

  // The other #511 symptom — the first wheel after a navigation being dropped
  // before the new document can be hit-tested — is a race (reliable on Chrome,
  // intermittent on Firefox), so it does not make a deterministic fail-on-main
  // gate. The verify-and-fall-back-to-scrollBy fix covers it and is a safe
  // no-op when the wheel already moved; measured 3/3 dropped on stock vs 0/5
  // patched from a cold daemon.

  test('cleanup: remove frame fixtures', () => {
    fs.rmSync(innerFile, { force: true });
    fs.rmSync(outerFile, { force: true });
  });
});
