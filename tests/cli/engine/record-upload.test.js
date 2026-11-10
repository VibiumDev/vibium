/**
 * CLI Tests: record stop --upload
 *
 * `vibium record stop --upload-to URL` sends the finished ZIP to a station's
 * Record Store: an authorization from POST /v1/records/upload-auth with the
 * connection's API key, then the bytes to PUT /upload with that
 * authorization. A small server in its own process stands in for the
 * station (tests/helpers/fake-station.js). Without a station or a key the
 * stop is refused before anything stops.
 */

const { test, describe, before, after } = require("../../helpers/capabilities").suite("core");
const assert = require('node:assert');
const { execSync, spawn } = require('node:child_process');
const crypto = require('node:crypto');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { VIBIUM } = require("../../helpers");

// Its own daemon, started with the API key in its environment: the shared
// session's daemon was started without one.
const ENV = { ...process.env, VIBIUM_SESSION: `record-upload-${process.pid}`, VIBIUM_CONNECT_API_KEY: 'stk_test_key' };

function cli(args) {
  return execSync(`${VIBIUM} --headless ${args}`, { encoding: 'utf-8', timeout: 60000, env: ENV });
}

// What a failed command printed, on either stream.
function failing(args) {
  try {
    cli(args);
  } catch (e) {
    return String(e.stdout) + String(e.stderr);
  }
  assert.fail(`${args} should have failed`);
}

let station, stationURL, tmp;

async function seen() {
  return (await fetch(`${stationURL}/seen`)).json();
}

before(async () => {
  tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'record-upload-'));
  station = spawn('node', [path.join(__dirname, '../../helpers/fake-station.js')], { stdio: ['pipe', 'pipe', 'pipe'] });
  stationURL = await new Promise((resolve) => station.stdout.once('data', (d) => resolve(d.toString().trim())));
  cli('stop');
});

after(() => {
  try { cli('stop'); } catch { /* nothing running */ }
  if (station) station.kill();
  fs.rmSync(tmp, { recursive: true, force: true });
});

describe('CLI: record stop --upload sends the ZIP to a station', () => {
  test('the ZIP reaches the station with the key, the authorization and the right hash', { timeout: 120000 }, async () => {
    const out = path.join(tmp, 'run.zip');
    cli(`record start -o ${out}`);
    cli(`content '<p>recorded</p>'`);
    const stdout = cli(`record stop --upload-to ${stationURL}`);
    assert.ok(fs.existsSync(out), 'the ZIP is still saved locally');
    const local = crypto.createHash('sha256').update(fs.readFileSync(out)).digest('hex');
    const got = await seen();
    assert.strictEqual(got.auth.authorization, 'Bearer stk_test_key');
    assert.strictEqual(got.auth.body.name, 'run.zip');
    assert.strictEqual(got.auth.body.type, 'application/zip');
    assert.strictEqual(got.auth.body.sha256, local, "the authorization names the saved file's hash");
    assert.strictEqual(got.put.authorization, 'Nostr dGVzdA==');
    assert.strictEqual(got.put.type, 'application/zip');
    assert.strictEqual(got.put.sha256, local);
    assert.match(stdout, /Saved .*run\.zip/);
    assert.match(stdout, new RegExp(`Uploaded to ${stationURL}/${local}\\.zip \\(device rig_1/phone, hold hld_test\\)`));
  });

  test('without a station the stop is refused and the recording keeps going', { timeout: 120000 }, () => {
    const out = path.join(tmp, 'kept.zip');
    cli(`record start -o ${out}`);
    assert.match(failing('record stop --upload'), /needs a station/);
    assert.ok(!fs.existsSync(out), 'nothing was written by the refused stop');
    assert.match(cli('record stop'), /Saved .*kept\.zip/);
    assert.ok(fs.existsSync(out));
  });

  test('a station that refuses leaves the ZIP saved and says why', { timeout: 120000 }, () => {
    const out = path.join(tmp, 'refused.zip');
    cli(`record start -o ${out}`);
    const text = failing(`record stop --upload-to ${stationURL}`);
    assert.match(text, /413/);
    assert.match(text, /at most 4 bytes/);
    assert.match(text, /saved to .*refused\.zip/);
    assert.ok(fs.existsSync(out), 'the ZIP is saved even though the station refused it');
  });
});
