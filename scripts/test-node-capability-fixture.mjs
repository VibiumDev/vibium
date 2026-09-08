import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';

function run(file, env = {}) {
  const result = spawnSync(
    process.execPath,
    ['--test', '--test-reporter=spec', file],
    {
      cwd: process.cwd(),
      encoding: 'utf8',
      env: {
        ...process.env,
        VIBIUM_ENGINE: 'chrome',
        VIBIUM_CAPABILITY_AUDIT: '1',
        ...env,
      },
    }
  );
  assert.equal(result.status, 0, result.stderr || result.stdout);
  return result.stdout;
}

const counts = run('tests/capability-fixtures/node.test.js', {
  VIBIUM_CAPABILITY_COLLECT_ONLY: '1',
});
assert.match(counts, /engine=chrome collected=3 selected=1 skipped=2/);
assert.match(counts, /skip:audio=2/);
console.log('Node capability fixture counts ok');

// These two run without collect-only: hook suppression only matters when the
// hooks would otherwise execute.
const suiteSkip = run('tests/capability-fixtures/node-skipped-suite.test.js');
assert.match(suiteSkip, /engine=chrome collected=2 selected=0 skipped=2/);
assert.match(suiteSkip, /skip:audio=2/);
assert.doesNotMatch(suiteSkip, /-RAN/);

const describeSkip = run('tests/capability-fixtures/node-skipped-describe.test.js');
assert.match(describeSkip, /SUITE-BEFORE-RAN/);
assert.doesNotMatch(describeSkip, /AUDIO-BEFORE-RAN/);
assert.match(describeSkip, /engine=chrome collected=2 selected=1 skipped=1/);
assert.match(describeSkip, /skip:audio=1/);
console.log('Node capability fixture hook suppression ok');
