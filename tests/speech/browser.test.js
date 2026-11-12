/**
 * Say and listen through a browser page.
 *
 * A local page records its microphone for six seconds, waits half a
 * second, and plays the recording back. Six seconds leaves room for a
 * slow machine to get its say in before the page stops recording. The test speaks a tone into the
 * page and listens for it to come back. No speech engine, key, sound card
 * or network is needed: say plays a WAV file and listen does not
 * transcribe.
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { execFile, spawnSync } = require('node:child_process');
const fs = require('node:fs');
const http = require('node:http');
const path = require('node:path');
const os = require('node:os');
const { VIBIUM } = require('../helpers');

const SESSION = `sb${process.pid}`;
const RECORD_MS = 6000;
const THINK_MS = 500;
const TONE_MS = 1000;

const PAGE = `<!doctype html><title>echo</title>
<button id="probe">Probe</button><button id="talk">Talk</button><p id="status">idle</p>
<script>
// A context nothing is connected to. Its clock says whether this browser
// runs audio at all on this machine.
document.getElementById('probe').onclick = () => { window.probe = new AudioContext(); };
document.getElementById('talk').onclick = async () => {
  const status = document.getElementById('status');
  const ctx = new AudioContext();
  const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
  const source = ctx.createMediaStreamSource(stream);
  const tap = ctx.createScriptProcessor(2048, 1, 1);
  const chunks = [];
  tap.onaudioprocess = (e) => chunks.push(new Float32Array(e.inputBuffer.getChannelData(0)));
  source.connect(tap);
  tap.connect(ctx.destination);
  status.textContent = 'recording';
  await new Promise((r) => setTimeout(r, ${RECORD_MS}));
  tap.disconnect(); source.disconnect();
  stream.getTracks().forEach((t) => t.stop());
  status.textContent = 'thinking';
  await new Promise((r) => setTimeout(r, ${THINK_MS}));
  const total = chunks.reduce((n, a) => n + a.length, 0);
  const buf = ctx.createBuffer(1, total, ctx.sampleRate);
  let off = 0;
  for (const a of chunks) { buf.getChannelData(0).set(a, off); off += a.length; }
  const play = ctx.createBufferSource();
  play.buffer = buf;
  play.connect(ctx.destination);
  play.onended = () => { status.textContent = 'done'; };
  play.start();
  status.textContent = 'playing';
};
</script>`;

function toneWAV(ms) {
  const rate = 16000;
  const samples = new Int16Array(Math.round(rate * ms / 1000));
  for (let i = 0; i < samples.length; i++) samples[i] = Math.round(8000 * Math.sin(2 * Math.PI * 300 * i / rate));
  const data = Buffer.from(samples.buffer);
  const header = Buffer.alloc(44);
  header.write('RIFF', 0); header.writeUInt32LE(36 + data.length, 4); header.write('WAVEfmt ', 8);
  header.writeUInt32LE(16, 16); header.writeUInt16LE(1, 20); header.writeUInt16LE(1, 22);
  header.writeUInt32LE(rate, 24); header.writeUInt32LE(rate * 2, 28); header.writeUInt16LE(2, 32);
  header.writeUInt16LE(16, 34); header.write('data', 36); header.writeUInt32LE(data.length, 40);
  return Buffer.concat([header, data]);
}

describe('say and listen through a page', () => {
  let dir, server, url, env;

  // Asynchronous, so this process keeps serving the page while the
  // command runs.
  const cli = (...args) => new Promise((resolve) => {
    execFile(VIBIUM, args, { encoding: 'utf-8', timeout: 90000, env, cwd: dir }, (error, stdout, stderr) => {
      resolve({ status: error ? (error.code ?? 1) : 0, stdout, stderr });
    });
  });

  before(async () => {
    dir = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-speech-page-'));
    fs.writeFileSync(path.join(dir, 'tone.wav'), toneWAV(TONE_MS));
    server = http.createServer((req, res) => { res.setHeader('Content-Type', 'text/html'); res.end(PAGE); });
    await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
    url = `http://127.0.0.1:${server.address().port}/`;
    env = { ...process.env, VIBIUM_SESSION: SESSION, OPENAI_API_KEY: '', DEEPGRAM_API_KEY: '', ELEVENLABS_API_KEY: '',
      VIBIUM_SAY_PROVIDER: '', VIBIUM_SAY_COMMAND: '', VIBIUM_LISTEN_PROVIDER: '', VIBIUM_LISTEN_COMMAND: '' };
  });

  after(async () => {
    await cli('stop');
    await cli('daemon', 'stop');
    server.close();
    fs.rmSync(dir, { recursive: true, force: true });
  });

  // Ten seconds of this test are sound in real time, and each of its
  // commands starts a process. A slow machine needs more than the suite's
  // 30 seconds for that.
  test('a clip spoken into the page comes back out of it', { timeout: 120000 }, async (t) => {
    const go = await cli('go', url, '--headless');
    assert.strictEqual(go.status, 0, go.stderr);

    // A browser with no audio device to play to may not run its audio
    // clock. Pages have no sound there, with or without Vibium. The clock
    // must run at half of real time or better.
    assert.strictEqual((await cli('click', '#probe')).status, 0);
    const clock = await cli('eval', `new Promise((r) => { const p = window.probe, t0 = p.currentTime; setTimeout(() => r(JSON.stringify({ state: p.state, advanced: Math.round((p.currentTime - t0) * 1000), inMs: 600, rate: p.sampleRate })), 600); })`);
    const audio = (clock.stdout.match(/\{.*\}/) || [clock.stdout + clock.stderr])[0].replace(/\\"/g, '"');
    const advanced = Number((audio.match(/"advanced":\s*(-?\d+)/) || [])[1]);
    if (!(advanced >= 300)) {
      t.skip(`this browser's audio clock does not keep time here: ${audio}`);
      return;
    }

    // With a session open the default target is the page. Its microphone
    // is not open yet, and say must say so instead of using the speakers.
    const early = await cli('say', '--file', 'tone.wav');
    assert.notStrictEqual(early.status, 0);
    assert.match(early.stderr, /no_audio: no page has the microphone open/);

    assert.strictEqual((await cli('record', 'start', '-o', 'echo.zip', '--video=false')).status, 0);
    assert.strictEqual((await cli('click', '#talk')).status, 0);
    const say = await cli('say', '--file', 'tone.wav', '--json');
    assert.strictEqual(say.status, 0, `${say.stderr}\naudio: ${audio}`);
    const said = JSON.parse(say.stdout).result;
    assert.strictEqual(said.target, 'session');
    assert.strictEqual(said.durationMs, TONE_MS);

    const listen = await cli('listen', '--no-transcribe', '--timeout', '9000', '--json', '-o', 'heard.wav');
    assert.strictEqual(listen.status, 0, `${listen.stderr}${listen.stdout}\naudio: ${audio}`);
    const heard = JSON.parse(listen.stdout).result;
    assert.strictEqual(heard.target, 'session');
    assert.strictEqual(heard.speech, true);
    assert.strictEqual(heard.stoppedBy, 'silence');
    // The page plays the recording RECORD_MS + THINK_MS after it began to
    // record, and the tone sits in it where it was spoken. So the tone
    // comes back that long after it went in: its start is heard
    // RECORD_MS + THINK_MS - TONE_MS after its end was spoken.
    const expected = RECORD_MS + THINK_MS - TONE_MS;
    assert.ok(Math.abs(heard.latencyMs - expected) < 450, `latency ${heard.latencyMs} ms, expected about ${expected}`);
    const length = heard.speechEndMs - heard.onsetMs;
    assert.ok(Math.abs(length - TONE_MS) < 200, `heard ${length} ms of tone, spoke ${TONE_MS}`);
    assert.ok(fs.statSync(path.join(dir, 'heard.wav')).size > 60000);

    const stop = await cli('record', 'stop');
    assert.strictEqual(stop.status, 0, stop.stderr);
    const list = spawnSync('unzip', ['-Z1', path.join(dir, 'echo.zip')], { encoding: 'utf-8' }).stdout;
    for (const name of ['audio/index.json', 'audio/in-session.wav', 'audio/out-0001.wav']) {
      assert.ok(list.includes(name), `the recording has no ${name}:\n${list}`);
    }
  });
});
