const assert = require('node:assert/strict');
const { browser } = require('../../clients/javascript/dist/' + (process.env.RUN_TEST_SYNC === '1' ? 'sync.js' : 'index.js'));
(async () => {
  const bro = await browser.start({ headless: true, engine: process.env.RUN_TEST_ENGINE || 'chrome' });
  try {
    const page = await bro.page();
    await page.go(process.env.RUN_TEST_URL);
    await page.evaluate("window.name='original';sessionStorage.setItem('builder','preserved')");
    const other = await bro.newPage(); await other.go(process.env.RUN_TEST_URL + '/other');
    await page.context.recording.start({ video: false, path: process.env.RUN_TEST_OUTPUT });
    if (process.env.RUN_TEST_PRIVACY === '1') {
      await page.evaluate("(() => { const input=document.createElement('input');input.type='password';input.id='password';document.body.append(input); })()");
      assert.equal((await page.run('credential test')).status, 'completed');
      assert.equal(await page.evaluate("document.querySelector('#password').value"), 'TEST-PASSWORD-SECRET');
      await page.context.recording.stop();
      return;
    }
    assert.equal(typeof bro, 'function');
    assert.equal(typeof page, 'function');
    const result = await page('change name');
    assert.equal(result.status, 'completed'); assert.equal(result.goal, 'change name'); assert.ok(result.evidence.length);
    assert.equal(await (await page.find('#name')).value(), 'Updated');
    assert.equal(await page.evaluate('window.name'), 'original');
    assert.equal(await page.evaluate("sessionStorage.getItem('builder')"), 'preserved');
    assert.equal(await (await other.find('#name')).value(), 'Other');
    assert.equal((await page.check('the name persisted')).status, 'passed');
    const overrides = { provider: process.env.VIBIUM_AI_PROVIDER === 'anthropic' ? 'google' : 'anthropic', model: 'run-model', baseURL: process.env.VIBIUM_AI_BASE_URL, reasoningEffort: '' };
    assert.equal((await page.run('change name', overrides)).status, 'completed');
    assert.equal((await page.check('the name persisted', { ...overrides, model: 'check-model' })).status, 'passed');
    assert.equal((await bro('not possible')).status, 'not_completed');
    await page.context.recording.stop();
  } finally { await bro.stop(); }
})().catch(e => { console.error(e); process.exitCode = 1; });
