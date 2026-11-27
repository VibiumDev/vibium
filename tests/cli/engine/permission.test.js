/**
 * CLI Tests: vibium permission (#711)
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

const sh = (args) => execSync(`${VIBIUM} ${args}`, { encoding: 'utf-8', timeout: 30000 });
const state = () => sh(`eval "navigator.permissions.query({ name: 'geolocation' }).then((s) => s.state)"`).trim();

describe('CLI: permission', () => {
  test('grants, denies and resets a permission for the current page', () => {
    sh(`go ${baseURL}/example`);
    assert.match(sh('permission geolocation granted'), new RegExp(`Permission geolocation set to granted for ${baseURL}`));
    assert.match(state(), /granted/);
    sh('permission geolocation denied');
    assert.match(state(), /denied/);
    sh(`permission geolocation prompt --origin ${baseURL}`);
    assert.match(state(), /prompt/);
  });

  test('an unknown state is an error that names the valid ones', () => {
    assert.throws(() => sh('permission geolocation allowed'), /state must be one of granted, denied, prompt/);
  });
});
