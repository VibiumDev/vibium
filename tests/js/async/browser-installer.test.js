const { test, describe } = require('node:test');
const assert = require('node:assert');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

const { browser } = require('../../../clients/javascript/dist');

function fakeVibium(logPath, installed) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-js-installer-'));
  const binary = path.join(dir, 'vibium');
  fs.writeFileSync(binary, `#!/bin/sh
printf '%s\\n' "$*" >> "${logPath}"
if [ "$1" = "is-installed" ]; then exit ${installed ? 0 : 1}; fi
if [ "$1" = "pipe" ]; then
  printf '%s\\n' '{"method":"vibium:lifecycle.ready","params":{}}'
  while read -r line; do
    printf '%s\\n' '{"id":1,"type":"success","result":{}}'
  done
fi
exit 0
`, { mode: 0o755 });
  return { binary, dir };
}

describe('JavaScript browser installer', () => {
  test('installs the selected Firefox channel when it is missing', async () => {
    const logPath = path.join(os.tmpdir(), `vibium-installer-${process.pid}-${Date.now()}.log`);
    const { binary, dir } = fakeVibium(logPath, false);
    try {
      const bro = await browser.start({
        engine: 'firefox',
        channel: 'beta',
        executablePath: binary,
      });
      await bro.stop();
      assert.deepStrictEqual(fs.readFileSync(logPath, 'utf8').trim().split('\n'), [
        'is-installed --engine firefox --firefox-channel beta',
        'install --engine firefox --firefox-channel beta',
        'pipe --engine firefox --firefox-channel beta',
      ]);
    } finally {
      fs.rmSync(dir, { recursive: true, force: true });
      fs.rmSync(logPath, { force: true });
    }
  });

  test('does not install an available browser', async () => {
    const logPath = path.join(os.tmpdir(), `vibium-installer-${process.pid}-${Date.now()}.log`);
    const { binary, dir } = fakeVibium(logPath, true);
    try {
      const bro = await browser.start({
        engine: 'firefox',
        channel: 'release',
        executablePath: binary,
      });
      await bro.stop();
      assert.deepStrictEqual(fs.readFileSync(logPath, 'utf8').trim().split('\n'), [
        'is-installed --engine firefox --firefox-channel release',
        'pipe --engine firefox --firefox-channel release',
      ]);
    } finally {
      fs.rmSync(dir, { recursive: true, force: true });
      fs.rmSync(logPath, { force: true });
    }
  });
});
