const assert = require('node:assert/strict');
const { browser } = require('../../clients/javascript/dist/' + (process.env.VERIFY_SYNC === '1' ? 'sync.js' : 'index.js'));
(async () => {
  if (process.env.VERIFY_ARCHIVE_ONLY === '1') {
    const result = await browser.verify('archive evidence', { record: process.env.VERIFY_INPUT });
    assert.equal(result.status, 'inconclusive');
    return;
  }
  const bro = await browser.start({ headless: true, engine: process.env.VERIFY_ENGINE || 'chrome', channel: process.env.VERIFY_ENGINE === 'firefox' ? 'beta' : undefined });
  try {
    const page = await bro.page();
    await page.go(process.env.VERIFY_URL);
    await page.evaluate("window.name='original'; localStorage.setItem('builder','preserved'); sessionStorage.setItem('builder','preserved')");
    const other = await bro.newPage();
    await other.go(process.env.VERIFY_URL + '/other');
    await page.context.recording.start({ video: false, path: process.env.VERIFY_OUTPUT });
    const result = await page.verify('changing my display name persists after refresh');
    assert.equal(result.status, 'passed');
    assert.equal(await (await page.find('#name')).value(), 'Updated');
    assert.equal(await page.evaluate('window.name'), 'original');
    assert.equal(await page.evaluate("localStorage.getItem('builder')"), 'preserved');
    assert.equal(await page.evaluate("sessionStorage.getItem('builder')"), 'preserved');
    assert.equal(await (await other.find('#name')).value(), 'Other');
    await page.context.recording.stop();
    const archive = await bro.verify('archive evidence', { record: process.env.VERIFY_INPUT });
    assert.equal(archive.status, 'inconclusive');
  } finally { await bro.stop(); }
})().catch(e => { console.error(e); process.exitCode = 1; });
