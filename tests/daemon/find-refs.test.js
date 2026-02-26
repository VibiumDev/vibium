/**
 * Daemon Tests: Find commands return @refs and refs are clickable
 * Tests the find → click workflow in daemon mode
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { execSync } = require('node:child_process');
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
  const result = clicker(`${args} --json`, opts);
  return JSON.parse(result);
}

function stopDaemon() {
  try {
    execSync(`${VIBIUM} daemon stop`, { encoding: 'utf-8', timeout: 10000 });
  } catch (e) {
    // Daemon may not be running
  }
}

describe('Daemon: Find @refs workflow', () => {
  before(() => {
    stopDaemon();
    clicker('daemon start -d --headless');
    clicker('go https://example.com');
  });

  after(() => {
    stopDaemon();
  });

  test('find --text returns @ref', () => {
    clicker('go https://example.com');
    const findResult = clickerJSON('find --text "Example Domain"');
    assert.strictEqual(findResult.ok, true);
    assert.ok(findResult.result.includes('@e1'), 'find should return @e1');
    assert.ok(findResult.result.includes('[h1]'), 'Should find h1 element');
  });

  test('find CSS selector returns @ref and click @e1 navigates', () => {
    clicker('go https://example.com');
    const findResult = clickerJSON('find "a"');
    assert.strictEqual(findResult.ok, true);
    assert.ok(findResult.result.includes('@e1'), 'find should return @e1');
    assert.ok(findResult.result.includes('[a]'), 'Should show [a] tag label');

    clickerJSON('click @e1');
    clickerJSON('wait-for-load');
    const urlResult = clickerJSON('url');
    assert.ok(urlResult.result.includes('iana.org'), 'Should navigate to IANA after clicking');
  });

  test('find --role returns @ref and click @e1 navigates', () => {
    clicker('go https://example.com');
    const findResult = clickerJSON('find --role link');
    assert.strictEqual(findResult.ok, true);
    assert.ok(findResult.result.includes('@e1'), 'find --role should return @e1');
    assert.ok(findResult.result.includes('[a]'), 'Should show [a] tag label');

    clickerJSON('click @e1');
    clickerJSON('wait-for-load');
    const urlResult = clickerJSON('url');
    assert.ok(urlResult.result.includes('iana.org'), 'Should navigate to IANA after clicking');
  });

  test('find-all returns multiple @refs', () => {
    clicker('go https://example.com');
    const findResult = clickerJSON('find-all "p"');
    assert.strictEqual(findResult.ok, true);
    assert.ok(findResult.result.includes('@e1'), 'find-all should return @e1');
    assert.ok(findResult.result.includes('@e2'), 'find-all should return @e2');
  });

  test('find resets refMap from previous map', () => {
    clicker('go https://example.com');
    // First map to get refs
    const mapResult = clickerJSON('map');
    assert.strictEqual(mapResult.ok, true);
    assert.ok(mapResult.result.includes('@e1'), 'map should have @e1');

    // Now find resets the refMap — @e1 now points to h1
    const findResult = clickerJSON('find --text "Example Domain"');
    assert.strictEqual(findResult.ok, true);
    assert.ok(findResult.result.includes('@e1'), 'find should return @e1');
    assert.ok(findResult.result.includes('[h1]'), 'find should return h1');

    // Clicking @e1 (h1) should not navigate away
    clickerJSON('click @e1');
    const urlResult = clickerJSON('url');
    assert.ok(urlResult.result.includes('example.com'), 'Should still be on example.com');
  });
});
