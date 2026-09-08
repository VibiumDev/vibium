import asyncio
import os

async def run_async():
    from vibium.async_api import browser
    bro = await browser.start(headless=True)
    try:
        page = await bro.page()
        await page.go(os.environ['RUN_TEST_URL'])
        await page.evaluate("window.name='original';sessionStorage.setItem('builder','preserved')")
        other = await bro.new_page()
        await other.go(os.environ['RUN_TEST_URL'] + '/other')
        await page.context.recording.start(video=False, path=os.environ['RUN_TEST_OUTPUT'])
        assert callable(bro) and callable(page)
        result = await page('change name')
        assert result['status'] == 'completed' and result['goal'] == 'change name'
        assert await (await page.find('#name')).value() == 'Updated'
        assert await page.evaluate('window.name') == 'original'
        assert await page.evaluate("sessionStorage.getItem('builder')") == 'preserved'
        assert await (await other.find('#name')).value() == 'Other'
        assert (await page.check('the name persisted'))['status'] == 'passed'
        overrides = dict(provider='google' if os.environ['VIBIUM_AI_PROVIDER'] == 'anthropic' else 'anthropic', model='run-model', base_url=os.environ['VIBIUM_AI_BASE_URL'], reasoning_effort='')
        assert (await page.run('change name', **overrides))['status'] == 'completed'
        overrides['model'] = 'check-model'
        assert (await page.check('the name persisted', **overrides))['status'] == 'passed'
        assert (await bro('not possible'))['status'] == 'not_completed'
        await page.context.recording.stop()
    finally:
        await bro.stop()

def run_sync():
    from vibium import browser
    bro = browser.start(headless=True)
    try:
        page = bro.page()
        page.go(os.environ['RUN_TEST_URL'])
        page.evaluate("window.name='original';sessionStorage.setItem('builder','preserved')")
        other = bro.new_page()
        other.go(os.environ['RUN_TEST_URL'] + '/other')
        page.context.recording.start(video=False, path=os.environ['RUN_TEST_OUTPUT'])
        assert callable(bro) and callable(page)
        result = page('change name')
        assert result['status'] == 'completed' and result['goal'] == 'change name'
        assert page.find('#name').value() == 'Updated'
        assert page.evaluate('window.name') == 'original'
        assert page.evaluate("sessionStorage.getItem('builder')") == 'preserved'
        assert other.find('#name').value() == 'Other'
        assert page.check('the name persisted')['status'] == 'passed'
        overrides = dict(provider='google' if os.environ['VIBIUM_AI_PROVIDER'] == 'anthropic' else 'anthropic', model='run-model', base_url=os.environ['VIBIUM_AI_BASE_URL'], reasoning_effort='')
        assert (page.run('change name', **overrides))['status'] == 'completed'
        overrides['model'] = 'check-model'
        assert (page.check('the name persisted', **overrides))['status'] == 'passed'
        assert bro('not possible')['status'] == 'not_completed'
        page.context.recording.stop()
    finally:
        bro.stop()

if os.environ.get('RUN_TEST_SYNC') == '1':
    run_sync()
else:
    asyncio.run(run_async())
