const { test, describe } = require('node:test');
const assert = require('node:assert');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

const { browser, BrowserCrashedError } = require('../../../clients/javascript/dist');

// The binary owns browser install now (#312): the client must launch via
// `pipe` alone, never `is-installed` or `install`. These fakes log every
// invocation so the tests can pin that.
function fakeVibium(logPath, script) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-js-pipe-'));
  const binary = path.join(dir, 'vibium');
  fs.writeFileSync(binary, `#!/bin/sh
printf '%s\\n' "$*" >> "${logPath}"
${script}
`, { mode: 0o755 });
  return { binary, dir };
}

const READY_SCRIPT = `if [ "$1" = "pipe" ]; then
  printf '%s\\n' '{"method":"vibium:lifecycle.ready","params":{}}'
  while read -r line; do
    printf '%s\\n' '{"id":1,"type":"success","result":{}}'
  done
fi
exit 0`;

describe('JavaScript pipe launch', () => {
  test('launches via pipe only, engine env resolution left to the binary', async () => {
    const logPath = path.join(os.tmpdir(), `vibium-pipe-${process.pid}-${Date.now()}.log`);
    const { binary, dir } = fakeVibium(logPath, READY_SCRIPT);
    const oldEngine = process.env.VIBIUM_ENGINE;
    const oldChannel = process.env.VIBIUM_FIREFOX_CHANNEL;
    process.env.VIBIUM_ENGINE = 'firefox';
    process.env.VIBIUM_FIREFOX_CHANNEL = 'beta';
    try {
      const bro = await browser.start({ executablePath: binary });
      await bro.stop();
      // No --engine/--firefox-channel: the binary reads the env itself.
      assert.deepStrictEqual(fs.readFileSync(logPath, 'utf8').trim().split('\n'), [
        'pipe',
      ]);
    } finally {
      if (oldEngine === undefined) delete process.env.VIBIUM_ENGINE;
      else process.env.VIBIUM_ENGINE = oldEngine;
      if (oldChannel === undefined) delete process.env.VIBIUM_FIREFOX_CHANNEL;
      else process.env.VIBIUM_FIREFOX_CHANNEL = oldChannel;
      fs.rmSync(dir, { recursive: true, force: true });
      fs.rmSync(logPath, { force: true });
    }
  });

  test('forwards explicit engine and channel to pipe, nothing else', async () => {
    const logPath = path.join(os.tmpdir(), `vibium-pipe-${process.pid}-${Date.now()}.log`);
    const { binary, dir } = fakeVibium(logPath, READY_SCRIPT);
    try {
      const bro = await browser.start({
        engine: 'firefox',
        channel: 'beta',
        executablePath: binary,
      });
      await bro.stop();
      assert.deepStrictEqual(fs.readFileSync(logPath, 'utf8').trim().split('\n'), [
        'pipe --engine firefox --firefox-channel beta',
      ]);
    } finally {
      fs.rmSync(dir, { recursive: true, force: true });
      fs.rmSync(logPath, { force: true });
    }
  });

  test('still becomes ready when pipe reports an install first', async () => {
    const logPath = path.join(os.tmpdir(), `vibium-pipe-${process.pid}-${Date.now()}.log`);
    // The marker vibium prints before downloading; the client extends its
    // ready deadline when it sees it. Ready follows, so this only exercises
    // the extension path — it must not break a normal launch.
    const { binary, dir } = fakeVibium(logPath, `if [ "$1" = "pipe" ]; then
  echo '[pipe] installing browser (chrome)' >&2
  sleep 0.2
  printf '%s\\n' '{"method":"vibium:lifecycle.ready","params":{}}'
  while read -r line; do
    printf '%s\\n' '{"id":1,"type":"success","result":{}}'
  done
fi
exit 0`);
    try {
      const bro = await browser.start({ executablePath: binary });
      await bro.stop();
      assert.deepStrictEqual(fs.readFileSync(logPath, 'utf8').trim().split('\n'), [
        'pipe',
      ]);
    } finally {
      fs.rmSync(dir, { recursive: true, force: true });
      fs.rmSync(logPath, { force: true });
    }
  });

  test('surfaces stderr in the crash error when pipe exits before ready', async () => {
    const logPath = path.join(os.tmpdir(), `vibium-pipe-${process.pid}-${Date.now()}.log`);
    const { binary, dir } = fakeVibium(logPath, `echo '[pipe] Failed to install browser: download refused' >&2
exit 3`);
    try {
      await assert.rejects(
        browser.start({ executablePath: binary }),
        (err) => {
          assert.ok(err instanceof BrowserCrashedError);
          assert.match(err.message, /download refused/);
          return true;
        },
      );
    } finally {
      fs.rmSync(dir, { recursive: true, force: true });
      fs.rmSync(logPath, { force: true });
    }
  });
});
