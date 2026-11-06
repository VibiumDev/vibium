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

// find --json answers with an array of refs; render it like the text form.
function refs(r) {
  return r.result.map((m) => `${m.ref} ${m.label}`).join('\n');
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
    assert.ok(refs(findResult).includes('@e1'), 'find should return @e1');
    assert.ok(refs(findResult).includes('[h1]'), 'Should find h1 element');
  });

  test('find CSS selector returns @ref and click @e1 navigates', () => {
    clicker(`go ${baseURL}/example`);
    const findResult = clickerJSON('find css "a"');
    assert.strictEqual(findResult.ok, true);
    assert.ok(refs(findResult).includes('@e1'), 'find should return @e1');
    assert.ok(refs(findResult).includes('[a]'), 'Should show [a] tag label');

    clickerJSON('click @e1');
    clickerJSON('wait load');
    const urlResult = clickerJSON('url');
    assert.ok(urlResult.result.includes('/more-information'), 'Should navigate to linked page after clicking');
  });

  test('find role returns @ref and click @e1 navigates', () => {
    clicker(`go ${baseURL}/example`);
    const findResult = clickerJSON('find role link');
    assert.strictEqual(findResult.ok, true);
    assert.ok(refs(findResult).includes('@e1'), 'find role should return @e1');
    assert.ok(refs(findResult).includes('[a]'), 'Should show [a] tag label');

    clickerJSON('click @e1');
    clickerJSON('wait load');
    const urlResult = clickerJSON('url');
    assert.ok(urlResult.result.includes('/more-information'), 'Should navigate to linked page after clicking');
  });

  test('find --all returns multiple @refs', () => {
    clicker(`go ${baseURL}/example`);
    const findResult = clickerJSON('find css "p" --all');
    assert.strictEqual(findResult.ok, true);
    assert.ok(refs(findResult).includes('@e1'), 'find --all should return @e1');
    assert.ok(refs(findResult).includes('@e2'), 'find --all should return @e2');
  });

  test('find keeps the map\'s refs and adds its own', () => {
    clicker(`go ${baseURL}/example`);
    const mapResult = clickerJSON('map');
    assert.strictEqual(mapResult.ok, true);
    assert.ok(mapResult.result.includes('@e1'), 'map should have @e1');

    // The heading was not in the map, so it takes the next number; @e1 still
    // names the map's first element.
    const findResult = clickerJSON('find text "Example Domain"');
    assert.strictEqual(findResult.ok, true);
    const [h1] = findResult.result;
    assert.notStrictEqual(h1.ref, '@e1', 'find must not renumber the map');
    assert.ok(h1.label.includes('[h1]'), 'find should return h1');
    assert.strictEqual(refs(clickerJSON('find text "Example Domain"')), `${h1.ref} ${h1.label}`, 'the same element keeps its ref');

    // Clicking the heading should not navigate away
    clickerJSON(`click ${h1.ref}`);
    const urlResult = clickerJSON('url');
    assert.ok(urlResult.result.includes('/example'), 'Should still be on /example');
  });
});
