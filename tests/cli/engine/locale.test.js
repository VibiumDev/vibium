/**
 * CLI Tests: vibium locale (#717)
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
  try {
    execSync(`${VIBIUM} locale --reset`, { encoding: 'utf-8', timeout: 30000 });
  } catch {}
  if (serverProcess) serverProcess.kill();
});

const sh = (args) => execSync(`${VIBIUM} ${args}`, { encoding: 'utf-8', timeout: 30000 });
const formatted = () => sh('eval "(1234.5).toLocaleString()"').trim();

describe('CLI: locale', () => {
  test('sets the locale and --reset removes it', () => {
    sh(`go ${baseURL}/example`);
    const before = formatted();
    assert.match(sh('locale de-DE'), /Locale set to de-DE/);
    assert.match(formatted(), /1\.234,5/);
    assert.match(sh('locale --reset'), /Locale override removed/);
    assert.strictEqual(formatted(), before);
  });

  test('needs a tag or --reset, not both and not neither', () => {
    assert.throws(() => sh('locale'), /locale takes a language tag such as de-DE, or --reset/);
    assert.throws(() => sh('locale de-DE --reset'), /locale takes a language tag such as de-DE, or --reset/);
  });
});
