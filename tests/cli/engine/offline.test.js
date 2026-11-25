/**
 * CLI Tests: vibium offline (#713)
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
    execSync(`${VIBIUM} offline off`, { encoding: 'utf-8', timeout: 30000 });
  } catch {}
  if (serverProcess) serverProcess.kill();
});

const sh = (args) => execSync(`${VIBIUM} ${args}`, { encoding: 'utf-8', timeout: 30000 });
const fetched = () => sh(`eval "fetch('/example').then(() => 'served').catch(() => 'failed')"`).trim();

describe('CLI: offline', () => {
  test('on cuts the page off and off reconnects it', () => {
    sh(`go ${baseURL}/example`);
    assert.match(sh('offline on'), /Network offline/);
    assert.match(fetched(), /failed/);
    assert.match(sh('offline off'), /Network online/);
    assert.match(fetched(), /served/);
  });

  test('anything but on or off is an error', () => {
    assert.throws(() => sh('offline maybe'), /offline takes on or off/);
  });
});
