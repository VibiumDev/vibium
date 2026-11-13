// The speech scenario of surfaces.test.js, through the JavaScript client.
const sync = process.env.SPEECH_TEST_SYNC === '1';
const { browser } = require('../../clients/javascript/dist/' + (sync ? 'sync.js' : 'index.js'));

const TEXT = 'Come here, Watson.';
const failure = async (call) => {
  try { await call(); } catch (e) { return { type: e.name, code: e.code, message: e.message }; }
  return null;
};

async function local() {
  const out = {};
  out.said = await browser.say(TEXT, { target: 'said.wav' });
  out.again = await browser.say(TEXT, { target: 'again.wav' });
  out.fresh = await browser.say(TEXT, { target: 'fresh.wav', cache: false });
  out.heard = await browser.listen({ target: 'said.wav', silence: 0 });
  out.unheard = await browser.listen({ target: 'said.wav', transcribe: false, path: 'copy.wav' });
  out.quiet = await failure(() => browser.listen({ target: 'quiet.wav' }));
  out.allowed = await browser.listen({ target: 'quiet.wav', allowSilence: true });
  out.noSession = await failure(() => browser.say(TEXT, { target: 'session' }));
  return out;
}

async function page() {
  const out = {};
  const bro = await browser.start({ headless: true, engine: process.env.SPEECH_TEST_ENGINE, channel: process.env.VIBIUM_ENGINE_CHANNEL || undefined });
  try {
    const page = await bro.page();
    await page.go(process.env.SPEECH_TEST_URL);
    await (await page.find({ css: '#probe' })).click();
    out.clock = await page.evaluate(process.env.SPEECH_TEST_CLOCK);
    if (!(out.clock.advanced >= 300)) return out;
    out.early = await failure(() => page.say('', { file: 'tone.wav' }));
    await (await page.find({ css: '#talk' })).click();
    out.said = await page.say('', { file: 'tone.wav' });
    out.heard = await bro.listen({ transcribe: false, timeout: 9000, path: 'heard.wav' });
  } finally { await bro.stop(); }
  return out;
}

(process.env.SPEECH_TEST_MODE === 'page' ? page() : local())
  .then((out) => console.log(JSON.stringify(out)))
  .catch((e) => { console.error(e); process.exitCode = 1; });
