const { test } = require('node:test');
const assert = require('node:assert/strict');
const { execFile } = require('node:child_process');
const { promisify } = require('node:util');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const exec = promisify(execFile);

// Run the actual tutorial script against a strict CLI stand-in. This catches
// removed commands in executable examples without visiting a public storefront.
for (const status of ['passed', 'failed', 'inconclusive']) {
  test(`cart tutorial uses Check and handles ${status} with recording cleanup`, { skip: process.platform === 'win32' }, async () => {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-tutorial-contract-'));
    const binary = path.join(dir, 'vibium');
    const callsPath = path.join(dir, 'calls.jsonl');
    fs.writeFileSync(binary, `#!/usr/bin/env node
const fs = require('node:fs');
const args = process.argv.slice(2).filter(a => a !== '--json' && a !== '--headless');
fs.appendFileSync(process.env.TUTORIAL_CALLS, JSON.stringify(args) + '\\n');
const [command, argument] = args;
if (!['check', 'go', 'record', 'map', 'find', 'click', 'daemon'].includes(command)) process.exit(9);
if (command === 'check' && argument === '--help') process.exit(0);
const result = command === 'check'
  ? { status: process.env.TUTORIAL_VERDICT, claim: argument, evidence: [] }
  : command === 'find' ? '@e1 fixture' : 'ok';
console.log(JSON.stringify({ok: true, result}));
`, { mode: 0o700 });
    try {
      let stdout, exitCode = 0;
      try {
        ({ stdout } = await exec(process.execPath, [path.resolve(__dirname, '../../scripts/var-parts-check.mjs')], {
          env: { ...process.env, VIBIUM_BINARY: binary, TUTORIAL_CALLS: callsPath, TUTORIAL_VERDICT: status },
          timeout: 15000,
        }));
      } catch (error) { stdout = error.stdout; exitCode = error.code; }
      assert.equal(exitCode, status === 'passed' ? 0 : 1);
      assert.equal(JSON.parse(stdout).status, status);
      const calls = fs.readFileSync(callsPath, 'utf8').trim().split('\n').map(JSON.parse);
      assert.deepEqual(calls[0], ['check', '--help']);
      assert.ok(calls.some(([cmd, claim]) => cmd === 'check' && claim.includes('current cart')));
      assert.deepEqual(calls.slice(-2), [['record', 'stop'], ['daemon', 'stop']]);
    } finally { fs.rmSync(dir, { recursive: true, force: true }); }
  });
}
