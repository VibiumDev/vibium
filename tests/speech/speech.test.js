/**
 * Say and listen, through the MCP server and the CLI.
 *
 * No speech engine, key or sound card is needed. The `command` provider
 * runs two shell scripts that stand in for a text-to-speech and a
 * speech-to-text engine, and audio goes through WAV files.
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { spawn, spawnSync } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');
const os = require('node:os');
const { VIBIUM } = require('../helpers');

const SESSION = `sp${process.pid}`;
let dir, env, renders;

// A second of a 300 Hz tone at 16 kHz, with silence around it, as a WAV.
function toneWAV(leadMs, toneMs, tailMs) {
  const rate = 16000;
  const n = (ms) => Math.round(rate * ms / 1000);
  const samples = new Int16Array(n(leadMs) + n(toneMs) + n(tailMs));
  for (let i = 0; i < n(toneMs); i++) {
    samples[n(leadMs) + i] = Math.round(6000 * Math.sin(2 * Math.PI * 300 * i / rate));
  }
  const data = Buffer.from(samples.buffer);
  const header = Buffer.alloc(44);
  header.write('RIFF', 0); header.writeUInt32LE(36 + data.length, 4); header.write('WAVEfmt ', 8);
  header.writeUInt32LE(16, 16); header.writeUInt16LE(1, 20); header.writeUInt16LE(1, 22);
  header.writeUInt32LE(rate, 24); header.writeUInt32LE(rate * 2, 28); header.writeUInt16LE(2, 32);
  header.writeUInt16LE(16, 34); header.write('data', 36); header.writeUInt32LE(data.length, 40);
  return Buffer.concat([header, data]);
}

function script(name, body) {
  const file = path.join(dir, name);
  fs.writeFileSync(file, `#!/bin/sh\n${body}\n`, { mode: 0o755 });
  return file;
}

function cli(args, extraEnv = {}) {
  return spawnSync(VIBIUM, args, { encoding: 'utf-8', timeout: 60000, env: { ...env, ...extraEnv }, cwd: dir });
}

function mcp(calls) {
  return new Promise((resolve, reject) => {
    const proc = spawn(VIBIUM, ['mcp', '--headless'], { stdio: ['pipe', 'pipe', 'pipe'], env, cwd: dir });
    const responses = [];
    let buffer = '';
    const timer = setTimeout(() => { proc.kill(); reject(new Error('mcp timed out')); }, 60000);
    proc.stdout.on('data', (chunk) => {
      buffer += chunk.toString();
      const lines = buffer.split('\n');
      buffer = lines.pop();
      for (const line of lines) {
        if (!line.trim()) continue;
        responses.push(JSON.parse(line));
        if (responses.length === calls.length + 1) {
          clearTimeout(timer);
          proc.stdin.end();
          resolve(responses.slice(1));
        }
      }
    });
    proc.on('error', reject);
    const send = (id, method, params) => proc.stdin.write(JSON.stringify({ jsonrpc: '2.0', id, method, params }) + '\n');
    send(0, 'initialize', { protocolVersion: '2024-11-05', capabilities: {}, clientInfo: { name: 'test', version: '1' } });
    calls.forEach((c, i) => send(i + 1, 'tools/call', c));
  });
}

const toolResult = (response) => JSON.parse(response.result.content[0].text);

describe('say and listen', () => {
  before(() => {
    if (process.platform === 'win32') return;
    dir = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-speech-'));
    fs.writeFileSync(path.join(dir, 'voice.wav'), toneWAV(400, 1000, 2500));
    renders = path.join(dir, 'renders');
    const tts = script('tts.sh', `echo x >> ${renders}; cat > ${path.join(dir, 'said.txt')}; cp ${path.join(dir, 'voice.wav')} "$1"`);
    const stt = script('stt.sh', `echo '{"text": "Come here, Watson.", "segments": [{"start": 0.3, "end": 1.3, "text": "Come here, Watson."}]}'`);
    env = {
      ...process.env,
      HOME: dir,
      VIBIUM_CONFIG_DIR: path.join(dir, 'config'),
      VIBIUM_CACHE_DIR: path.join(dir, 'cache'),
      VIBIUM_SESSION: SESSION,
      VIBIUM_SKIP_BROWSER_DOWNLOAD: '1',
      OPENAI_API_KEY: '', DEEPGRAM_API_KEY: '', ELEVENLABS_API_KEY: '',
      VIBIUM_SAY_PROVIDER: 'command', VIBIUM_SAY_COMMAND: `${tts} {output}`, VIBIUM_SAY_MODEL: '', VIBIUM_SAY_VOICE: '',
      VIBIUM_LISTEN_PROVIDER: 'command', VIBIUM_LISTEN_COMMAND: `${stt} {input}`, VIBIUM_LISTEN_MODEL: '',
    };
  });

  after(() => {
    if (!dir) return;
    cli(['daemon', 'stop']);
    fs.rmSync(dir, { recursive: true, force: true });
  });

  const skip = process.platform === 'win32' ? 'the fake engines are shell scripts' : false;

  test('the MCP server says into a file and hears it back, with no browser', { skip }, async () => {
    // The sandbox has no browser and downloads are off: a tool that needed
    // a browser would fail here.
    const [said, again, heard] = await mcp([
      { name: 'vibium_say', arguments: { text: 'Come here, Watson.', target: path.join(dir, 'mcp.wav') } },
      { name: 'vibium_say', arguments: { text: 'Come here, Watson.', target: path.join(dir, 'mcp2.wav') } },
      { name: 'vibium_listen', arguments: { target: path.join(dir, 'mcp.wav'), silence: 0 } },
    ]);
    const first = toolResult(said);
    assert.strictEqual(first.provider, 'command');
    assert.strictEqual(first.cached, false);
    assert.strictEqual(first.durationMs, 1100, 'the engine padded its audio; say trims it to the speech plus 50 ms at each end');
    assert.strictEqual(toolResult(again).cached, true);
    assert.strictEqual(toolResult(again).sha256, first.sha256);
    assert.deepStrictEqual(fs.readFileSync(path.join(dir, 'mcp.wav')), fs.readFileSync(path.join(dir, 'mcp2.wav')));
    assert.strictEqual(fs.readFileSync(path.join(dir, 'said.txt'), 'utf-8'), 'Come here, Watson.');

    const reply = toolResult(heard);
    assert.strictEqual(reply.text, 'Come here, Watson.');
    assert.strictEqual(reply.speech, true);
    assert.strictEqual(reply.stoppedBy, 'eof');
    assert.strictEqual(reply.latencyMs, undefined, 'a file has no preceding say to count from');
    assert.ok(reply.onsetMs >= 0 && reply.onsetMs <= 80, `onset ${reply.onsetMs}`);
  });

  test('the CLI says and listens through the daemon', { skip }, () => {
    fs.rmSync(renders, { force: true });
    const say = cli(['say', 'Set a timer for ten minutes', '--target', 'cli.wav', '--cache-dir', 'spoken']);
    assert.strictEqual(say.status, 0, say.stderr);
    assert.match(say.stdout, /^Wrote cli\.wav \(1\.1 s, command\)\n$/);

    const cached = cli(['say', 'Set a timer for ten minutes', '--target', 'cli2.wav', '--cache-dir', 'spoken', '--json']);
    const envelope = JSON.parse(cached.stdout);
    assert.strictEqual(envelope.ok, true);
    assert.strictEqual(envelope.result.cached, true);
    assert.strictEqual(fs.readFileSync(renders, 'utf-8'), 'x\n', 'the second say must replay the cache, not render again');
    assert.ok(fs.readdirSync(path.join(dir, 'spoken')).length > 0, 'the cache directory is relative to where the command ran');

    const listen = cli(['listen', '--target', 'cli.wav', '--silence', '0']);
    assert.strictEqual(listen.status, 0, listen.stderr);
    assert.strictEqual(listen.stdout, 'Come here, Watson.\n', 'plain output is the transcript alone');

    const json = JSON.parse(cli(['listen', '--target', 'cli.wav', '--silence', '0', '--json', '-o', 'heard.wav']).stdout);
    assert.strictEqual(json.ok, true);
    assert.strictEqual(json.result.provider, 'command');
    assert.strictEqual(json.result.segments.length, 1);
    assert.ok(fs.statSync(path.join(dir, 'heard.wav')).size > 30000);
  });

  test('text from stdin, and text that starts with a dash', { skip }, () => {
    const piped = spawnSync(VIBIUM, ['say', '-', '--target', 'stdin.wav', '--no-cache'], { input: 'What time is it?\n', encoding: 'utf-8', env, cwd: dir, timeout: 60000 });
    assert.strictEqual(piped.status, 0, piped.stderr);
    assert.strictEqual(fs.readFileSync(path.join(dir, 'said.txt'), 'utf-8'), 'What time is it?');
    const dashed = cli(['say', '--target', 'dash.wav', '--no-cache', '--', '-5 degrees']);
    assert.strictEqual(dashed.status, 0, dashed.stderr);
    assert.strictEqual(fs.readFileSync(path.join(dir, 'said.txt'), 'utf-8'), '-5 degrees');
  });

  test('failures carry a code', { skip }, () => {
    fs.writeFileSync(path.join(dir, 'silent.wav'), toneWAV(3000, 0, 0));
    const silent = cli(['listen', '--target', 'silent.wav', '--json']);
    assert.notStrictEqual(silent.status, 0);
    assert.match(JSON.parse(silent.stdout).error, /no_speech: listened to .* for 3\.0 s/);

    const allowed = JSON.parse(cli(['listen', '--target', 'silent.wav', '--allow-silence', '--json']).stdout);
    assert.strictEqual(allowed.result.speech, false);
    assert.strictEqual(allowed.result.text, '');

    // The daemon started with a provider in its environment. This call has
    // none in its own, and the call's settings are the ones that count.
    const unset = cli(['listen', '--target', 'cli.wav'], { VIBIUM_LISTEN_PROVIDER: '', VIBIUM_LISTEN_COMMAND: '' });
    assert.notStrictEqual(unset.status, 0);
    assert.match(unset.stderr, /not_configured: no listen provider is configured/);

    assert.match(cli(['say', 'hi', '--target', 'session']).stderr, /invalid_target: no session is open/);
    assert.match(cli(['say', 'hi', '--target', 'speaker.mp3']).stderr, /invalid_target/);
    assert.match(cli(['listen', '--target', 'missing.wav']).stderr, /no_audio: cannot read/);
  });
});
