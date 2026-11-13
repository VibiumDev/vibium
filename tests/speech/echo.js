/**
 * What the speech tests share: a tone as a WAV file, and a page that
 * records its microphone, waits, and plays the recording back.
 */

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

// A 300 Hz tone at 16 kHz with silence before and after it, as a WAV.
function toneWAV(leadMs, toneMs, tailMs, level = 6000) {
  const rate = 16000;
  const n = (ms) => Math.round(rate * ms / 1000);
  const samples = new Int16Array(n(leadMs) + n(toneMs) + n(tailMs));
  for (let i = 0; i < n(toneMs); i++) {
    samples[n(leadMs) + i] = Math.round(level * Math.sin(2 * Math.PI * 300 * i / rate));
  }
  const data = Buffer.from(samples.buffer);
  const header = Buffer.alloc(44);
  header.write('RIFF', 0); header.writeUInt32LE(36 + data.length, 4); header.write('WAVEfmt ', 8);
  header.writeUInt32LE(16, 16); header.writeUInt16LE(1, 20); header.writeUInt16LE(1, 22);
  header.writeUInt32LE(rate, 24); header.writeUInt32LE(rate * 2, 28); header.writeUInt16LE(2, 32);
  header.writeUInt16LE(16, 34); header.write('data', 36); header.writeUInt32LE(data.length, 40);
  return Buffer.concat([header, data]);
}

module.exports = { RECORD_MS, THINK_MS, TONE_MS, PAGE, toneWAV };
