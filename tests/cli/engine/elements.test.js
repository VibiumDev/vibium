/**
 * CLI Tests: Element Finding, Click, and Type
 * Tests the vibium binary directly
 */

const { test, describe, before, after } = require("../../helpers/capabilities").suite("core");
const assert = require('node:assert');
const { execSync, spawn } = require('node:child_process');
const path = require('path');
const os = require('os');
const fs = require('fs');
const { VIBIUM } = require("../../helpers");

let serverProcess, baseURL;

before(async () => {
  serverProcess = spawn('node', [path.join(__dirname, '../../helpers/test-server.js')], {
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

describe('CLI: Elements', () => {
  test('find command locates element and returns @ref', () => {
    const result = execSync(`${VIBIUM} find ${baseURL}/example "a"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /@e1/, 'Should return @e1 ref');
    assert.match(result, /\[a\]/, 'Should show [a] tag label');
    // Link text may be "More information..." or "Learn more" depending on page version
    assert.match(result, /(More information|Learn more)/i, 'Should show link text');
  });

  test('find xpath subcommand locates element', () => {
    execSync(`${VIBIUM} go ${baseURL}/inputs`, { encoding: 'utf-8', timeout: 30000 });
    const result = execSync(`${VIBIUM} find xpath "//input"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /@e\d+/, 'Should return an element ref');
    assert.match(result, /\[input/, 'Should show [input] tag label');
  });

  test('click command navigates via link', () => {
    const result = execSync(`${VIBIUM} click ${baseURL}/example "a"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /clicked/i, 'Should confirm element was clicked');
  });

  test('type command enters text into input', () => {
    const result = execSync(
      `${VIBIUM} type ${baseURL}/inputs "input" "12345"`,
      {
        encoding: 'utf-8',
        timeout: 30000,
      }
    );
    assert.match(result, /typed/i, 'Should confirm text was typed');
  });

  test('type enters text containing a carriage return (#61)', () => {
    execSync(`${VIBIUM} go ${baseURL}/inputs`, { encoding: 'utf-8', timeout: 30000 });
    execSync(`${VIBIUM} type "#multiline" "$(printf '123\\r456')"`, {
      encoding: 'utf-8',
      timeout: 30000,
      shell: '/bin/bash',
    });

    const value = execSync(`${VIBIUM} eval "document.querySelector('#multiline').value"`, {
      encoding: 'utf-8',
      timeout: 30000,
    }).trim();
    assert.strictEqual(value, '123\n456');
  });

  test('type enters text containing a newline (#61)', () => {
    // A literal newline is not a key. It used to be sent as a key value the
    // browser ignored, so "123\n456" arrived as "123456".
    execSync(`${VIBIUM} go ${baseURL}/inputs`, { encoding: 'utf-8', timeout: 30000 });
    execSync(`${VIBIUM} type "#multiline" "123
456"`, { encoding: 'utf-8', timeout: 30000 });

    const value = execSync(`${VIBIUM} eval "document.querySelector('#multiline').value"`, {
      encoding: 'utf-8',
      timeout: 30000,
    }).trim();
    assert.strictEqual(value, '123\n456');
  });

});

describe('CLI: shadow DOM pierce', () => {
  test('>> crosses one shadow boundary, >>> crosses nested ones (#118)', () => {
    execSync(`${VIBIUM} go ${baseURL}/shadow`, { encoding: 'utf-8', timeout: 30000 });

    // Plain CSS cannot cross the boundary — the baseline the issue documents.
    assert.throws(
      () => execSync(`${VIBIUM} find "my-card p" --timeout 2s`, {
        encoding: 'utf-8', timeout: 30000, stdio: 'pipe',
      }),
      /not found/,
    );

    assert.match(execSync(`${VIBIUM} find "my-card >> p"`, { encoding: 'utf-8', timeout: 30000 }), /shadow text/);
    assert.match(execSync(`${VIBIUM} find "my-card >>> #deep"`, { encoding: 'utf-8', timeout: 30000 }), /Deep Button/);
  });

  test('wait and focus resolve a pierced selector (#748)', () => {
    // Their resolvers used plain querySelector, so ">>" was an invalid selector
    // there, and "wait --state hidden" passed on it without looking.
    execSync(`${VIBIUM} go ${baseURL}/shadow`, { encoding: 'utf-8', timeout: 30000 });
    assert.match(execSync(`${VIBIUM} wait "my-card >> #i" --state visible --timeout 2s`, { encoding: 'utf-8', timeout: 30000 }), /reached state: visible/);
    assert.throws(() => execSync(`${VIBIUM} wait "my-card >> #i" --state hidden --timeout 1s`, { encoding: 'utf-8', timeout: 30000, stdio: 'pipe' }), /still visible/);
    execSync(`${VIBIUM} focus "my-card >> #i"`, { encoding: 'utf-8', timeout: 30000 });
    const active = execSync(`${VIBIUM} eval "document.querySelector('my-card').shadowRoot.activeElement.id"`, { encoding: 'utf-8', timeout: 30000 }).trim();
    assert.strictEqual(active, 'i');
  });

  test('actions work on a pierced element', () => {
    execSync(`${VIBIUM} go ${baseURL}/shadow`, { encoding: 'utf-8', timeout: 30000 });

    execSync(`${VIBIUM} fill "my-card >> #i" hello`, { encoding: 'utf-8', timeout: 30000 });
    execSync(`${VIBIUM} click "my-card >> #b"`, { encoding: 'utf-8', timeout: 30000 });

    // The component's own handler read the value, so fill and click both landed
    // inside the shadow root — including the elementFromPoint hit test.
    const out = execSync(`${VIBIUM} text "#out"`, { encoding: 'utf-8', timeout: 30000 });
    assert.match(out, /clicked:hello/);
  });

  // Since #283 map lists shadow elements, but their refs were document paths
  // that stopped at the shadow root: they matched nothing or a light-DOM element.
  const refFor = (out, pattern) => {
    const m = out.match(new RegExp('(@e\\d+) ' + pattern));
    assert.ok(m, `should list ${pattern}:\n${out}`);
    return m[1];
  };
  const clickAndRead = (ref) => {
    execSync(`${VIBIUM} click ${ref} --timeout 5s`, { encoding: 'utf-8', timeout: 30000 });
    return execSync(`${VIBIUM} text "#out"`, { encoding: 'utf-8', timeout: 30000 }).trim();
  };

  test('map refs inside shadow roots click the mapped element (#748)', () => {
    execSync(`${VIBIUM} go ${baseURL}/shadow-refs`, { encoding: 'utf-8', timeout: 30000 });
    const out = execSync(`${VIBIUM} map`, { encoding: 'utf-8', timeout: 30000 });
    for (const label of ['Save', 'B', 'C', 'Deep']) {
      assert.strictEqual(clickAndRead(refFor(out, `\\[button\\] "${label}"`)), `shadow:${label}`);
    }

    // A shadow element with an id: the id only holds inside its root.
    execSync(`${VIBIUM} go ${baseURL}/shadow`, { encoding: 'utf-8', timeout: 30000 });
    const card = execSync(`${VIBIUM} map`, { encoding: 'utf-8', timeout: 30000 });
    assert.strictEqual(clickAndRead(refFor(card, '\\[button\\] "Shadow Button"')), 'clicked:');
  });

  test('find --all refs inside a shadow root click the found element (#748)', () => {
    execSync(`${VIBIUM} go ${baseURL}/shadow-refs`, { encoding: 'utf-8', timeout: 30000 });
    const out = execSync(`${VIBIUM} find --all "x-panel >> div > button"`, { encoding: 'utf-8', timeout: 30000 });
    assert.strictEqual(clickAndRead(refFor(out, '\\[button\\] "C"')), 'shadow:C');
  });

  test('a ref keeps the escapes in a host id (#748)', () => {
    execSync(`${VIBIUM} go ${baseURL}/shadow-refs`, { encoding: 'utf-8', timeout: 30000 });
    const out = execSync(`${VIBIUM} map`, { encoding: 'utf-8', timeout: 30000 });
    assert.strictEqual(clickAndRead(refFor(out, '\\[button\\] "Spaced"')), 'shadow:Spaced');
    assert.strictEqual(clickAndRead(refFor(out, '\\[button\\] "Quoted"')), 'shadow:Quoted');
  });

  test('the >> split skips a quoted >> and trims whitespace (#748)', () => {
    execSync(`${VIBIUM} go ${baseURL}/shadow-refs`, { encoding: 'utf-8', timeout: 30000 });
    const last = `'button[aria-label=">>"]'`;
    assert.strictEqual(execSync(`${VIBIUM} count ${last}`, { encoding: 'utf-8', timeout: 30000 }).trim(), '1');
    execSync(`${VIBIUM} click ${last} --timeout 5s`, { encoding: 'utf-8', timeout: 30000 });
    assert.strictEqual(execSync(`${VIBIUM} text "#out"`, { encoding: 'utf-8', timeout: 30000 }).trim(), 'last');
    // Whitespace around >> is trimmed, including a pasted no-break space.
    assert.strictEqual(execSync(`${VIBIUM} count "x-panel\u00a0>> button"`, { encoding: 'utf-8', timeout: 30000 }).trim(), '6');
  });

  test('count, highlight, annotate and upload resolve shadow refs (#748)', () => {
    execSync(`${VIBIUM} go ${baseURL}/shadow-refs`, { encoding: 'utf-8', timeout: 30000 });
    const map = execSync(`${VIBIUM} map`, { encoding: 'utf-8', timeout: 30000 });
    const save = refFor(map, '\\[button\\] "Save"');
    const b = refFor(map, '\\[button\\] "B"');
    assert.strictEqual(execSync(`${VIBIUM} count ${b}`, { encoding: 'utf-8', timeout: 30000 }).trim(), '1');
    assert.strictEqual(execSync(`${VIBIUM} count "x-panel >> button"`, { encoding: 'utf-8', timeout: 30000 }).trim(), '6');
    // The outline itself is removed after 3s, too soon to read reliably from
    // another process; the click tests show this selector reaches Save.
    assert.match(execSync(`${VIBIUM} highlight ${save}`, { encoding: 'utf-8', timeout: 30000 }), /Highlighted body > x-panel >> :host > button /);
    // Annotate drops an element it can't resolve; this guards against it
    // throwing on the ">>" in a shadow ref.
    const shot = path.join(os.tmpdir(), `shadow-refs-annotate-${process.pid}.png`);
    try {
      execSync(`${VIBIUM} screenshot --annotate -o ${JSON.stringify(shot)}`, { encoding: 'utf-8', timeout: 30000, stdio: 'pipe' });
    } finally {
      fs.rmSync(shot, { force: true });
    }
    const file = refFor(map, '\\[input type="file"\\]');
    execSync(`${VIBIUM} upload ${file} ${JSON.stringify(__filename)} --timeout 5s`, { encoding: 'utf-8', timeout: 30000 });
    const name = execSync(`${VIBIUM} eval "document.querySelector('x-panel').shadowRoot.querySelector('input').files[0].name"`, { encoding: 'utf-8', timeout: 30000 }).trim();
    assert.strictEqual(name, path.basename(__filename));
  });

  test('map lists elements inside shadow roots (#203)', () => {
    execSync(`${VIBIUM} go ${baseURL}/shadow`, { encoding: 'utf-8', timeout: 30000 });
    const out = execSync(`${VIBIUM} map`, { encoding: 'utf-8', timeout: 30000 });
    assert.match(out, /Shadow Button/, 'shadow-root elements should be mapped');
    assert.match(out, /Deep Button/, 'nested shadow roots too');
  });
});
