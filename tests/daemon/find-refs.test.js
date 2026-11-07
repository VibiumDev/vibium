/**
 * Daemon Tests: Find commands return @refs and refs are clickable
 * Tests the find → click workflow in daemon mode
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { execSync, spawn } = require('node:child_process');
const path = require('node:path');
const { VIBIUM } = require('../helpers');

function clicker(args, opts = {}) {
  const result = execSync(`${VIBIUM} ${args}`, {
    encoding: 'utf-8',
    timeout: opts.timeout || 60000,
    env: { ...process.env, ...opts.env },
  });
  return result.trim();
}

function clickerJSON(args, opts = {}) {
  const result = clicker(`--json ${args}`, opts);
  return JSON.parse(result);
}

// A failed command exits non-zero, with its --json result on stdout.
function clickerResult(args) {
  try {
    return clickerJSON(args);
  } catch (e) {
    return JSON.parse(e.stdout.trim());
  }
}

function stopDaemon() {
  try {
    execSync(`${VIBIUM} daemon stop`, { encoding: 'utf-8', timeout: 10000 });
  } catch (e) {
    // Daemon may not be running
  }
}

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

describe('Daemon: Find @refs workflow', () => {
  before(() => {
    stopDaemon();
    clicker('daemon start --headless');
    clicker(`go ${baseURL}/example`);
  });

  after(() => {
    stopDaemon();
  });

  test('find text returns @ref', () => {
    clicker(`go ${baseURL}/example`);
    const findResult = clickerJSON('find text "Example Domain"');
    assert.strictEqual(findResult.ok, true);
    assert.ok(findResult.result.includes('@e1'), 'find should return @e1');
    assert.ok(findResult.result.includes('[h1]'), 'Should find h1 element');
  });

  test('find CSS selector returns @ref and click @e1 navigates', () => {
    clicker(`go ${baseURL}/example`);
    const findResult = clickerJSON('find "a"');
    assert.strictEqual(findResult.ok, true);
    assert.ok(findResult.result.includes('@e1'), 'find should return @e1');
    assert.ok(findResult.result.includes('[a]'), 'Should show [a] tag label');

    clickerJSON('click @e1');
    clickerJSON('wait load');
    const urlResult = clickerJSON('url');
    assert.ok(urlResult.result.includes('/more-information'), 'Should navigate to linked page after clicking');
  });

  test('find role returns @ref and click @e1 navigates', () => {
    clicker(`go ${baseURL}/example`);
    const findResult = clickerJSON('find role link');
    assert.strictEqual(findResult.ok, true);
    assert.ok(findResult.result.includes('@e1'), 'find role should return @e1');
    assert.ok(findResult.result.includes('[a]'), 'Should show [a] tag label');

    clickerJSON('click @e1');
    clickerJSON('wait load');
    const urlResult = clickerJSON('url');
    assert.ok(urlResult.result.includes('/more-information'), 'Should navigate to linked page after clicking');
  });

  test('find --all returns multiple @refs', () => {
    clicker(`go ${baseURL}/example`);
    const findResult = clickerJSON('find --all "p"');
    assert.strictEqual(findResult.ok, true);
    assert.ok(findResult.result.includes('@e1'), 'find --all should return @e1');
    assert.ok(findResult.result.includes('@e2'), 'find --all should return @e2');
  });

  test('find resets refMap from previous map', () => {
    clicker(`go ${baseURL}/example`);
    // First map to get refs
    const mapResult = clickerJSON('map');
    assert.strictEqual(mapResult.ok, true);
    assert.ok(mapResult.result.includes('@e1'), 'map should have @e1');

    // Now find resets the refMap — @e1 now points to h1
    const findResult = clickerJSON('find text "Example Domain"');
    assert.strictEqual(findResult.ok, true);
    assert.ok(findResult.result.includes('@e1'), 'find should return @e1');
    assert.ok(findResult.result.includes('[h1]'), 'find should return h1');

    // Clicking @e1 (h1) should not navigate away
    clickerJSON('click @e1');
    const urlResult = clickerJSON('url');
    assert.ok(urlResult.result.includes('/example'), 'Should still be on /example');
  });

  test('ref whose element moved fails as stale instead of clicking its neighbor', () => {
    clicker(`go ${baseURL}/add_remove_elements/`);
    clicker(`eval "addElement(); addElement(); addElement(); document.querySelectorAll('.added-manually').forEach((b, i) => b.dataset.n = i)"`);
    const mapResult = clickerJSON('map');
    assert.ok(mapResult.result.includes('@e4'), 'map should list the three Delete buttons');

    clickerJSON('click @e2');
    const result = clickerResult('click @e3');
    assert.strictEqual(result.ok, false, 'clicking a moved ref should fail');
    assert.match(result.error, /stale ref @e3: its element is still on the page/);
    const left = clickerJSON(`eval "[...document.querySelectorAll('.added-manually')].map(b => b.dataset.n).join()"`);
    assert.strictEqual(left.result, '1,2', 'the third button must not have been deleted');
  });

  test('ref into a re-rendered list fails as stale for actions and questions', () => {
    const b = (t) => `<li><button onclick="document.title=this.textContent">${t}</button></li>`;
    clicker(`content '<ul id="l">${b('A')}${b('B')}${b('C')}</ul>'`);
    clickerJSON('map');
    clicker(`eval "const l = document.getElementById('l'); l.firstElementChild.remove(); l.innerHTML = l.innerHTML; 'ok'"`);
    // C now sits where B was, so "is B visible?" must not answer about C.
    const asked = clickerResult('is visible @e2');
    assert.strictEqual(asked.ok, false, `is visible answered ${asked.result}`);
    assert.match(asked.error, /stale ref @e2: the page changed/);
    const clicked = clickerResult('click @e2');
    assert.strictEqual(clicked.ok, false, 'clicking B through its old ref should fail');
    assert.match(clicked.error, /stale ref @e2: the page changed/);
    assert.notStrictEqual(clickerJSON('title').result, 'C', 'C must not have been clicked');
  });

  test('a CSS find ref is not checked against an earlier map', () => {
    // Checked against map's @e1, the first #go, this ref would fail as moved.
    clicker(`content '<form id="f1"><button type="button" id="go" onclick="document.title=1">Go</button></form><form id="f2"><button type="button" id="go" onclick="document.title=2">Go</button></form>'`);
    clickerJSON('map');
    clickerJSON('find "#f2 #go"');
    clickerJSON('click @e1');
    assert.strictEqual(clickerJSON('title').result, '2');
  });

  test('questions about absence answer as on main when nothing took the element\'s place', () => {
    clicker(`go ${baseURL}/add_remove_elements/`);
    clicker('eval "addElement()"');
    clickerJSON('map');
    clicker(`eval "document.querySelector('.added-manually').remove()"`);
    assert.strictEqual(clickerJSON('is visible @e2').result, 'false');
    assert.strictEqual(clickerJSON('count @e2').result, '0');
    assert.match(clickerJSON('wait @e2 --state hidden --timeout 2000').result, /reached state: hidden/);
  });

  test('a live element another one now shadows is not reported gone', () => {
    clicker(`content '<div id="slot"></div><button id="save">Save</button>'`);
    clickerJSON('map');
    // A second #save earlier in the page takes over @e1's selector.
    clicker(`eval "document.getElementById('slot').innerHTML = '<button id=save>Other</button>'"`);
    const result = clickerResult('is visible @e1');
    assert.strictEqual(result.ok, false, 'a visible element must not answer "not visible"');
    assert.match(result.error, /still on the page/);
  });

  test('an open dialog does not hold a ref command for the full command timeout', () => {
    clicker(`content '<button onclick="setTimeout(() => alert(1))">Alert</button><button>Other</button>'`);
    clickerJSON('map');
    clicker('click @e1');
    const started = Date.now();
    try {
      clicker('click @e2 --timeout 2000');
    } catch {
      /* failing is fine; hanging is not */
    }
    const took = Date.now() - started;
    try { clicker('dialog dismiss'); } catch { /* already gone */ }
    assert.ok(took < 10000, `click @e2 under an open alert took ${took}ms`);
  });

  test('the ref store is invisible to the page', () => {
    clicker(`go ${baseURL}/checkboxes`);
    clickerJSON('map');
    assert.strictEqual(clickerJSON('eval "typeof vibiumRefs"').result, 'undefined');
  });

  test('a CSS find ref re-resolves its selector after a re-render', () => {
    // The caller's selector means whatever matches .spinner now.
    clicker(`content '<div id="box"><span class="spinner">wait</span></div>'`);
    clickerJSON('find ".spinner"');
    clickerJSON('find @e1');
    clicker(`eval "document.getElementById('box').innerHTML = '<span class=spinner>still</span>'"`);
    assert.strictEqual(clickerJSON('is visible @e1').result, 'true');
    assert.strictEqual(clickerJSON('count @e1').result, '1');
  });

  for (const [name, mint] of [['map', 'map'], ['find --all', 'find --all "#checkboxes input"'], ['a semantic find', 'find role checkbox']]) {
    test(`ref from ${name} on a previous document fails as stale`, () => {
      clicker(`go ${baseURL}/checkboxes`);
      clickerJSON(mint);
      clicker(`go ${baseURL}/checkboxes`);
      const result = clickerResult('click @e1');
      assert.strictEqual(result.ok, false, 'clicking a ref from the old document should fail');
      assert.match(result.error, /stale ref @e1/);
      const checked = clickerJSON(`eval "document.querySelector('#checkboxes input').checked"`);
      assert.strictEqual(checked.result, 'false', 'the new page must not have been clicked');
    });
  }

  test('drag checks the refs it is given', () => {
    clicker(`go ${baseURL}/checkboxes`);
    clickerJSON('map');
    clicker(`go ${baseURL}/checkboxes`);
    assert.match(clickerResult('drag @e1 "#checkboxes"').error ?? '', /stale ref @e1/);
    assert.match(clickerResult('drag "#checkboxes" @e2').error ?? '', /stale ref @e2/);
  });
});
