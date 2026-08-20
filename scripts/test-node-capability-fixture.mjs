import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';

const result = spawnSync(
  process.execPath,
  ['--test', '--test-reporter=spec', 'tests/capability-fixtures/node.test.js'],
  {
    cwd: process.cwd(),
    encoding: 'utf8',
    env: {
      ...process.env,
      VIBIUM_ENGINE: 'chrome',
      VIBIUM_CAPABILITY_AUDIT: '1',
      VIBIUM_CAPABILITY_COLLECT_ONLY: '1',
    },
  }
);
assert.equal(result.status, 0, result.stderr || result.stdout);
assert.match(result.stdout, /engine=chrome collected=3 selected=1 skipped=2/);
assert.match(result.stdout, /skip:audio=2/);
console.log('Node capability fixture counts ok');
