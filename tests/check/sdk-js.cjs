const assert = require('node:assert/strict');
const { browser } = require('../../clients/javascript/dist/' + (process.env.CHECK_TEST_SYNC === '1' ? 'sync.js' : 'index.js'));
(async () => {
  if (process.env.CHECK_TEST_ARCHIVE_ONLY === '1') {
    const result = await browser.check('archive evidence', { record: process.env.CHECK_TEST_INPUT, provider: 'local', model: 'archive-override', baseURL: process.env.VIBIUM_AI_BASE_URL, reasoningEffort: '' });
    assert.equal(result.status, 'inconclusive');
    return;
  }
  const bro = await browser.start({ headless: true, engine: process.env.CHECK_TEST_ENGINE || 'chrome', channel: process.env.CHECK_TEST_ENGINE === 'firefox' ? 'beta' : undefined });
  try {
    const page = await bro.page();
    await page.go(process.env.CHECK_TEST_URL);
    await page.evaluate("window.name='original'; localStorage.setItem('builder','preserved'); sessionStorage.setItem('builder','preserved')");
    const other = await bro.newPage();
    await other.go(process.env.CHECK_TEST_URL + '/other');
    await page.context.recording.start({ video: false, path: process.env.CHECK_TEST_OUTPUT });
    const result = await page.check('changing my display name persists after refresh');
    assert.equal(result.status, 'passed');
    assert.equal(await (await page.find('#name')).value(), 'Updated');
    assert.equal(await page.evaluate('window.name'), 'original');
    assert.equal(await page.evaluate("localStorage.getItem('builder')"), 'preserved');
    assert.equal(await page.evaluate("sessionStorage.getItem('builder')"), 'preserved');
    assert.equal(await (await other.find('#name')).value(), 'Other');
    await page.context.recording.stop();
    const archive = await bro.check('archive evidence', { record: process.env.CHECK_TEST_INPUT, provider: 'local', model: 'archive-override', baseURL: process.env.VIBIUM_AI_BASE_URL, reasoningEffort: '' });
    assert.equal(archive.status, 'inconclusive');
  } finally { await bro.stop(); }
})().catch(e => { console.error(e); process.exitCode = 1; });
