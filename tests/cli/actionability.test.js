/**
 * CLI Tests: Actionability Checks
 * Tests auto-wait and actionability behavior
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { execSync, spawn } = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { VIBIUM } = require('../helpers');

let serverProcess, baseURL;

before(async () => {
  serverProcess = spawn('node', [path.join(__dirname, '../helpers/test-server.js')], {
    stdio: ['pipe', 'pipe', 'pipe'],
  });
  baseURL = await new Promise((resolve) => {
    serverProcess.stdout.once('data', (data) => {
      resolve(data.toString().trim());
    });
  });
});

after(() => {
  if (serverProcess) serverProcess.kill();
});

describe('CLI: Actionability', () => {
  test('is actionable reports visibility status', () => {
    const result = execSync(`${VIBIUM} is actionable ${baseURL}/example "a"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /Visible.*true/i, 'Link should be visible');
    assert.match(result, /Stable.*true/i, 'Link should be stable');
    assert.match(result, /ReceivesEvents.*true/i, 'Link should receive events');
    assert.match(result, /Enabled.*true/i, 'Link should be enabled');
  });

  test('click with short timeout fails on non-existent element', () => {
    assert.throws(
      () => {
        execSync(`${VIBIUM} click ${baseURL}/example "#does-not-exist" --timeout 1s`, {
          encoding: 'utf-8',
          timeout: 10000,
          stdio: 'pipe', // capture the expected failure instead of leaking it to the console
        });
      },
      /timeout|not found/i,
      'Should timeout or report not found'
    );
  });
});

describe('CLI: --timeout flag formats', () => {
  // Write a page where #late only appears ~1s after load, so a click must
  // auto-wait for it. Uses a temp file to avoid shell-quoting a data: URL.
  const tmpFile = path.join(os.tmpdir(), `vibium-timeout-${process.pid}.html`);
  const html =
    '<body><button id="late" style="display:none">Go</button>' +
    '<script>setTimeout(function(){document.getElementById("late").style.display="block"},1000)</script></body>';
  const fileURL = 'file://' + tmpFile;

  test('setup: write delayed-element fixture', () => {
    fs.writeFileSync(tmpFile, html);
  });

  test('accepts duration form (5s) and auto-waits for a late element', () => {
    execSync(`${VIBIUM} go "${fileURL}"`, { encoding: 'utf-8', timeout: 30000 });
    const result = execSync(`${VIBIUM} click "#late" --timeout 5s`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /Clicked/i, '5s timeout should auto-wait then click');
  });

  test('accepts bare-millisecond form (5000) and auto-waits for a late element', () => {
    execSync(`${VIBIUM} go "${fileURL}"`, { encoding: 'utf-8', timeout: 30000 });
    const result = execSync(`${VIBIUM} click "#late" --timeout 5000`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /Clicked/i, '5000ms timeout should auto-wait then click');
  });

  test('bare-millisecond timeout bounds the wait (reported in the error)', () => {
    assert.throws(
      () => {
        execSync(`${VIBIUM} click "#does-not-exist" --timeout 800`, {
          encoding: 'utf-8',
          timeout: 10000,
          stdio: 'pipe', // capture the expected failure instead of leaking it to the console
        });
      },
      /800ms|not found/i,
      'Should fail reporting the 800ms bound'
    );
  });

  test('rejects an invalid timeout value', () => {
    assert.throws(
      () => {
        execSync(`${VIBIUM} click "#x" --timeout 5q`, { encoding: 'utf-8', timeout: 10000, stdio: 'pipe' });
      },
      /invalid timeout/i,
      'Should reject "5q" with a clear message'
    );
  });

  test('newly-flagged action (hover) accepts --timeout', () => {
    execSync(`${VIBIUM} go "${fileURL}"`, { encoding: 'utf-8', timeout: 30000 });
    const result = execSync(`${VIBIUM} hover "#late" --timeout 5s`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /Hovered/i, 'hover should honor --timeout and wait for #late');
  });

  test('wait command accepts duration form (5s)', () => {
    execSync(`${VIBIUM} go "${fileURL}"`, { encoding: 'utf-8', timeout: 30000 });
    const result = execSync(`${VIBIUM} wait "#late" --state visible --timeout 5s`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /visible/i, 'wait should accept 5s and resolve when #late shows');
  });

  test('teardown: remove fixture', () => {
    fs.rmSync(tmpFile, { force: true });
  });
});

describe('CLI: fillable input types', () => {
  test('fill sets an input[type=range] value (regression: #188)', () => {
    execSync(`${VIBIUM} content '<input type="range" id="s" min="0" max="10" value="5">'`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    const result = execSync(`${VIBIUM} fill "#s" "3"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /Filled/, 'range should be fillable, not rejected as "not editable"');
    const value = execSync(`${VIBIUM} eval 'document.getElementById("s").value'`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.strictEqual(value.trim(), '3', 'range value should be set to 3');
  });

  test('fill still rejects a non-fillable input type (checkbox)', () => {
    execSync(`${VIBIUM} content '<input type="checkbox" id="cb">'`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.throws(
      () => {
        // --timeout before the positionals (fill uses SetInterspersed(false));
        // 1s so the expected failure returns quickly instead of auto-waiting.
        execSync(`${VIBIUM} fill --timeout 1s "#cb" "x"`, {
          encoding: 'utf-8',
          timeout: 10000,
          stdio: 'pipe',
        });
      },
      /not editable/i,
      'checkbox should not be fillable'
    );
  });
});

describe('CLI: receivesEvents hit-tests the in-view center (#340)', () => {
  const run = (cmd, opts = {}) =>
    execSync(`${VIBIUM} ${cmd}`, { encoding: 'utf-8', timeout: 30000, ...opts });

  // The reported reproduction: a full-page scrim three viewports tall. Its
  // bounding-rect center is off-screen, elementFromPoint returns null there, and
  // the check called it obscured — leaving the element permanently unclickable.
  // Sized in vh so the case holds at any viewport, with no sleeps or constants.
  const TALL_SCRIM =
    '<body style="margin:0"><div style="position:relative;height:300vh;background:#ddd">x' +
    '<div id="scrim" style="position:absolute;top:0;left:0;width:100%;height:100%;' +
    'background:#333;opacity:.3;z-index:12"></div></div>';

  test('clicks an element three viewports tall, at the center of its visible area', () => {
    run(`content '${TALL_SCRIM}'`);
    // Record where the pointer actually landed: this must prove the click
    // coordinate, not just that the check returned ok.
    run(
      `eval 'window.__hits = []; document.getElementById("scrim").addEventListener("click", function (e) { window.__hits.push([e.clientX, e.clientY]) })'`
    );

    assert.match(run(`click "#scrim" --timeout 5s`), /Clicked/i, 'a full-page scrim must be clickable');

    const { hits, w, h } = JSON.parse(
      run(`eval 'JSON.stringify({ hits: window.__hits, w: window.innerWidth, h: window.innerHeight })'`)
    );
    assert.strictEqual(hits.length, 1, 'the scrim should have received exactly one click');
    const [x, y] = hits[0];
    // The scrim spans the viewport vertically, so the in-view center row is
    // exactly floor(height / 2). The bounding-rect center was 1.5x the viewport.
    assert.strictEqual(y, Math.floor(h / 2), 'click must land on the in-view center row');
    assert.ok(x >= 0 && x < w, `click x ${x} must be inside the 0..${w} viewport`);
  });

  const VEILED_BUTTON =
    '<body style="margin:0"><button id="go" style="width:100px;height:40px">Go</button>' +
    '<div id="veil" style="position:fixed;top:0;left:0;width:100vw;height:100vh;' +
    'background:#0008;z-index:99"></div>';

  test('a covered button still reports obscured', () => {
    run(`content '${VEILED_BUTTON}'`);

    assert.throws(
      () => run(`click "#go" --timeout 1s`, { stdio: 'pipe' }),
      /receivesEvents check failed — element is obscured by <div#veil>/i,
      'clipping the probe point must not weaken the check'
    );
  });
});

describe('CLI: operation preconditions', () => {
  test('check refuses a non-checkbox instead of silently succeeding (#195)', () => {
    execSync(`${VIBIUM} content '<p id="p">not a checkbox</p><input type="checkbox" id="cb">'`, {
      encoding: 'utf-8',
      timeout: 30000,
    });

    assert.throws(
      () => {
        execSync(`${VIBIUM} check "#p"`, { encoding: 'utf-8', timeout: 30000, stdio: 'pipe' });
      },
      /not a checkbox or radio/i,
      'check on a <p> should be refused, not reported as checked'
    );

    // The real checkbox must still work.
    const ok = execSync(`${VIBIUM} check "#cb"`, { encoding: 'utf-8', timeout: 30000 });
    assert.match(ok, /Checked/);
  });

  test('upload refuses a non-file-input with a readable error (#197)', () => {
    execSync(`${VIBIUM} content '<p id="p">not an input</p>'`, {
      encoding: 'utf-8',
      timeout: 30000,
    });

    assert.throws(
      () => {
        execSync(`${VIBIUM} upload "#p" /etc/hosts`, {
          encoding: 'utf-8',
          timeout: 30000,
          stdio: 'pipe',
        });
      },
      /input type="file"/i,
      'should name the expected element type rather than surfacing a raw BiDi error'
    );
  });
});
