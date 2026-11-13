/**
 * Say and listen through the client libraries.
 *
 * Each client runs the same two scenarios and prints what it got, and
 * every client must get the same. No speech engine, key or sound card is
 * needed: two shell scripts stand in for the engines, audio goes through
 * WAV files, and the page is the echo page of browser.test.js.
 */

const { test } = require('node:test');
const assert = require('node:assert/strict');
const { execFile } = require('node:child_process');
const fs = require('node:fs');
const http = require('node:http');
const os = require('node:os');
const path = require('node:path');
const { VIBIUM, ENGINE, ENGINE_CHANNEL } = require('../helpers');
const { RECORD_MS, THINK_MS, TONE_MS, PAGE, toneWAV } = require('./echo');

const TEXT = 'Come here, Watson.';
// Reads the echo page's idle audio clock: how far it moved in 600 ms.
const CLOCK = `new Promise((r) => { const p = window.probe, t0 = p.currentTime; setTimeout(() => r({ state: p.state, advanced: Math.round((p.currentTime - t0) * 1000), inMs: 600 }), 600); })`;

const root = path.resolve(__dirname, '../..');
const PYTHON = path.join(root, process.platform === 'win32' ? 'clients/python/.venv/Scripts/python.exe' : 'clients/python/.venv/bin/python');

const CLIENTS = [
  ['JavaScript async', process.execPath, [path.join(__dirname, 'sdk-js.cjs')], '0'],
  ['JavaScript sync', process.execPath, [path.join(__dirname, 'sdk-js.cjs')], '1'],
  ['Python async', PYTHON, [path.join(__dirname, 'sdk-python.py')], '0'],
  ['Python sync', PYTHON, [path.join(__dirname, 'sdk-python.py')], '1'],
];

// A client names its fields in its own language's style. The test reads
// them all as the wire does.
function camel(value) {
  if (Array.isArray(value)) return value.map(camel);
  if (value === null || typeof value !== 'object') return value;
  return Object.fromEntries(Object.entries(value).map(([k, v]) => [k.replace(/_(\w)/g, (m, c) => c.toUpperCase()), camel(v)]));
}

// The temporary directory by its real path, which is what a process
// started in it calls it.
const tempDir = (prefix) => fs.realpathSync(fs.mkdtempSync(path.join(os.tmpdir(), prefix)));

function run(binary, args, options) {
  return new Promise((resolve, reject) => {
    execFile(binary, args, { encoding: 'utf-8', timeout: 150000, ...options }, (error, stdout, stderr) => {
      if (error) return reject(new Error(`${error.message}\n${stdout}${stderr}`));
      resolve(camel(JSON.parse(stdout.trim().split('\n').pop())));
    });
  });
}

const skip = process.platform === 'win32' ? 'the fake engines are shell scripts' : false;

test('the clients say and listen with no browser', { skip, timeout: 300000 }, async (t) => {
  const work = tempDir('vibium-speech-sdk-');
  try {
    for (const [label, binary, args, sync] of CLIENTS) {
      await t.test(label, async () => {
        const dir = path.join(work, label.replaceAll(' ', '-'));
        fs.mkdirSync(dir);
        const file = (name, body) => { fs.writeFileSync(path.join(dir, name), body, { mode: 0o755 }); return path.join(dir, name); };
        file('voice.wav', toneWAV(400, 1000, 2500));
        file('quiet.wav', toneWAV(2000, 0, 0));
        const tts = file('tts.sh', `#!/bin/sh\necho x >> ${dir}/renders; cat > ${dir}/said.txt; cp ${dir}/voice.wav "$1"\n`);
        const stt = file('stt.sh', `#!/bin/sh\necho '{"text": "${TEXT}", "segments": [{"start": 0.3, "end": 1.3, "text": "${TEXT}"}]}'\n`);
        // Counts the engines the client starts.
        const engine = file('vibium.sh', `#!/bin/sh\necho x >> ${dir}/starts\nexec ${VIBIUM} "$@"\n`);
        const env = {
          ...process.env,
          VIBIUM_BIN_PATH: engine, VIBIUM_CONNECT_URL: '',
          VIBIUM_CONFIG_DIR: path.join(dir, 'config'), VIBIUM_CACHE_DIR: path.join(dir, 'cache'),
          OPENAI_API_KEY: '', DEEPGRAM_API_KEY: '', ELEVENLABS_API_KEY: '',
          VIBIUM_SAY_PROVIDER: 'command', VIBIUM_SAY_COMMAND: `${tts} {output}`, VIBIUM_SAY_MODEL: '', VIBIUM_SAY_VOICE: '',
          VIBIUM_LISTEN_PROVIDER: 'command', VIBIUM_LISTEN_COMMAND: `${stt} {input}`, VIBIUM_LISTEN_MODEL: '',
          SPEECH_TEST_MODE: 'local', SPEECH_TEST_SYNC: sync,
        };
        const out = await run(binary, args, { env, cwd: dir });

        assert.equal(out.said.text, TEXT);
        assert.equal(out.said.provider, 'command');
        assert.equal(out.said.cached, false);
        assert.equal(out.said.durationMs, 1100, 'say trims the engine\'s audio to the speech plus 50 ms at each end');
        assert.equal(out.said.target, path.join(dir, 'said.wav'), 'a relative target is the client\'s, not the engine\'s');
        assert.equal(out.said.path, path.join(dir, 'said.wav'));
        assert.equal(fs.readFileSync(path.join(dir, 'said.txt'), 'utf-8'), TEXT);
        assert.equal(out.again.cached, true);
        assert.equal(out.again.sha256, out.said.sha256);
        assert.equal(out.fresh.cached, false, 'cache: false renders again');
        assert.equal(fs.readFileSync(path.join(dir, 'renders'), 'utf-8').trim().split('\n').length, 2);

        assert.equal(out.heard.text, TEXT);
        assert.equal(out.heard.transcribed, true);
        assert.equal(out.heard.speech, true);
        assert.equal(out.heard.stoppedBy, 'eof');
        assert.equal(out.heard.provider, 'command');
        assert.equal(out.heard.segments.length, 1);
        assert.equal(out.heard.segments[0].text, TEXT);
        assert.ok(out.heard.segments[0].endMs > out.heard.segments[0].startMs);
        assert.ok(Math.abs(out.heard.onsetMs - 50) < 60, `onset at ${out.heard.onsetMs} ms`);

        assert.equal(out.unheard.transcribed, false);
        assert.equal(out.unheard.text, '');
        assert.equal(out.unheard.speech, true);
        assert.equal(out.unheard.path, path.join(dir, 'copy.wav'));
        assert.ok(fs.statSync(path.join(dir, 'copy.wav')).size > 30000);

        assert.equal(out.quiet.type, 'SpeechError');
        assert.equal(out.quiet.code, 'no_speech');
        assert.match(out.quiet.message, /^listened to .* and heard no speech/, 'the message carries no code');
        assert.equal(out.allowed.speech, false);
        assert.equal(out.allowed.text, '');
        assert.equal(out.noSession.type, 'SpeechError');
        assert.equal(out.noSession.code, 'invalid_target');

        assert.equal(fs.readFileSync(path.join(dir, 'starts'), 'utf-8').trim().split('\n').length, 1,
          'every call went to one engine process');
      });
    }
  } finally {
    fs.rmSync(work, { recursive: true, force: true });
  }
});

test('the clients say and listen through a page', { timeout: 600000 }, async (t) => {
  const work = tempDir('vibium-speech-sdk-page-');
  const server = http.createServer((req, res) => { res.setHeader('Content-Type', 'text/html'); res.end(PAGE); });
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  try {
    for (const [label, binary, args, sync] of CLIENTS) {
      await t.test(label, async (t) => {
        const dir = path.join(work, label.replaceAll(' ', '-'));
        fs.mkdirSync(dir);
        fs.writeFileSync(path.join(dir, 'tone.wav'), toneWAV(0, TONE_MS, 0, 8000));
        const env = {
          ...process.env,
          VIBIUM_BIN_PATH: VIBIUM, VIBIUM_CONNECT_URL: '', VIBIUM_ENGINE: ENGINE, VIBIUM_ENGINE_CHANNEL: ENGINE_CHANNEL,
          OPENAI_API_KEY: '', DEEPGRAM_API_KEY: '', ELEVENLABS_API_KEY: '',
          VIBIUM_SAY_PROVIDER: '', VIBIUM_SAY_COMMAND: '', VIBIUM_LISTEN_PROVIDER: '', VIBIUM_LISTEN_COMMAND: '',
          SPEECH_TEST_MODE: 'page', SPEECH_TEST_SYNC: sync, SPEECH_TEST_ENGINE: ENGINE,
          SPEECH_TEST_URL: `http://127.0.0.1:${server.address().port}/`, SPEECH_TEST_CLOCK: CLOCK,
        };
        const out = await run(binary, args, { env, cwd: dir });

        // A browser with no audio device to play to may not run its audio
        // clock. Pages have no sound there, with or without Vibium.
        if (!(out.clock.advanced >= 300)) {
          t.skip(`this browser's audio clock does not keep time here: ${JSON.stringify(out.clock)}`);
          return;
        }

        assert.equal(out.early.type, 'SpeechError');
        assert.equal(out.early.code, 'no_audio');
        assert.match(out.early.message, /^no page has the microphone open/);

        assert.equal(out.said.target, 'session');
        assert.equal(out.said.durationMs, TONE_MS);
        assert.equal(out.said.file, path.join(dir, 'tone.wav'));

        assert.equal(out.heard.target, 'session');
        assert.equal(out.heard.speech, true);
        assert.equal(out.heard.transcribed, false);
        assert.equal(out.heard.stoppedBy, 'silence');
        // The tone comes back RECORD_MS + THINK_MS after it went in, so its
        // start is heard that long, less its length, after its end was spoken.
        const expected = RECORD_MS + THINK_MS - TONE_MS;
        assert.ok(Math.abs(out.heard.latencyMs - expected) < 450, `latency ${out.heard.latencyMs} ms, expected about ${expected}`);
        const length = out.heard.speechEndMs - out.heard.onsetMs;
        assert.ok(Math.abs(length - TONE_MS) < 200, `heard ${length} ms of tone, spoke ${TONE_MS}`);
        assert.ok(fs.statSync(path.join(dir, 'heard.wav')).size > 60000);
      });
    }
  } finally {
    server.closeAllConnections();
    await new Promise((resolve) => server.close(resolve));
    fs.rmSync(work, { recursive: true, force: true });
  }
});
