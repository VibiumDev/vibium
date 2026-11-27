/**
 * CLI Tests: Find commands return @refs
 * Tests that find, find --all return @refs in oneshot mode
 */

const { test, describe, before, after } = require("../../helpers/capabilities").suite("core");
const assert = require('node:assert');
const { execSync, spawn } = require('node:child_process');
const path = require('path');
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

describe('CLI: Find @refs', () => {
  test('find CSS selector returns @ref', () => {
    execSync(`${VIBIUM} go ${baseURL}/example`, { encoding: 'utf-8', timeout: 30000 });
    const result = execSync(`${VIBIUM} find css "a"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /@e1/, 'Should contain @e1 ref');
    assert.match(result, /\[a\]/, 'Should show [a] tag label');
  });

  test('find --all returns multiple @refs', () => {
    execSync(`${VIBIUM} go ${baseURL}/example`, { encoding: 'utf-8', timeout: 30000 });
    const result = execSync(`${VIBIUM} find css "p" --all`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /@e1/, 'Should contain @e1');
    assert.match(result, /@e2/, 'Should contain @e2');
  });

  test('find --all --limit 1 returns single @ref', () => {
    execSync(`${VIBIUM} go ${baseURL}/example`, { encoding: 'utf-8', timeout: 30000 });
    const result = execSync(`${VIBIUM} find css "p" --all --limit 1`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /@e1/, 'Should contain @e1');
    assert.ok(!result.includes('@e2'), 'Should not contain @e2');
  });

  test('find filters with --has-text and --has, and picks with --nth (#709)', () => {
    execSync(`${VIBIUM} go ${baseURL}/filter-list`, { encoding: 'utf-8', timeout: 30000 });
    const run = (flags) => execSync(`${VIBIUM} find css ".row" ${flags}`, { encoding: 'utf-8', timeout: 30000 });
    assert.match(run('--has-text Banan'), /Bananas/);
    assert.match(run('--has em'), /Cherries/);
    assert.match(run('--nth 2'), /Cherries/);
    assert.match(run('--nth -1'), /Dates/);
    const filtered = run('--has-text es --all');
    assert.match(filtered, /Apples/);
    assert.match(filtered, /Dates/);
    assert.ok(!filtered.includes('Bananas'), `Bananas does not contain "es": ${filtered}`);
  });
});
