/**
 * CLI Tests: Page Reading Tools
 * Tests text, html, find --all commands
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { execSync, spawn } = require('node:child_process');
const path = require('path');
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

describe('CLI: Page Reading', () => {
  test('text command returns page text', () => {
    const result = execSync(`${VIBIUM} text ${baseURL}/example`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /Example Domain/, 'Should contain page text');
  });

  test('text command with selector returns element text', () => {
    const result = execSync(`${VIBIUM} text ${baseURL}/example "h1"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /Example Domain/, 'Should contain h1 text');
  });

  test('html command returns page HTML', () => {
    const result = execSync(`${VIBIUM} html ${baseURL}/example "h1"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /Example Domain/, 'Should contain HTML');
  });

  test('html command with --outer returns outer HTML', () => {
    const result = execSync(`${VIBIUM} html ${baseURL}/example "h1" --outer`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /<h1>/, 'Should contain h1 tag');
    assert.match(result, /Example Domain/, 'Should contain text');
  });

  test('find --all returns multiple @refs', () => {
    const result = execSync(`${VIBIUM} find ${baseURL}/example "p" --all`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /@e1/, 'Should contain @e1 ref');
    assert.match(result, /\[p\]/, 'Should contain [p] tag label');
  });

  test('find --all with --limit', () => {
    const result = execSync(`${VIBIUM} find ${baseURL}/example "p" --all --limit 1`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /@e1/, 'Should contain @e1 ref');
    assert.ok(!result.includes('@e2'), 'Should not contain @e2 ref');
  });
});
