import asyncio
import os
from pathlib import Path

claim = "changing my display name persists after refresh"

async def run_async():
    from vibium.async_api import browser
    if os.environ["CHECK_TEST_ARCHIVE_ONLY"] == "1":
        result = await browser.check("archive evidence", record=os.environ["CHECK_TEST_INPUT"], provider="local", model="archive-override", base_url=os.environ["VIBIUM_AI_BASE_URL"], reasoning_effort="")
        assert result["status"] == "inconclusive"
        return
    bro = await browser.start(headless=True, engine=os.environ.get("CHECK_TEST_ENGINE", "chrome"), channel="beta" if os.environ.get("CHECK_TEST_ENGINE") == "firefox" else None)
    try:
        page = await bro.page()
        await page.go(os.environ["CHECK_TEST_URL"])
        await page.evaluate("window.name='original'; localStorage.setItem('builder','preserved'); sessionStorage.setItem('builder','preserved')")
        other = await bro.new_page()
        await other.go(os.environ["CHECK_TEST_URL"] + "/other")
        await page.context.recording.start(video=False, path=os.environ["CHECK_TEST_OUTPUT"])
        result = await page.check(claim)
        assert result["status"] == "passed"
        assert await (await page.find("#name")).value() == "Updated"
        assert await page.evaluate("window.name") == "original"
        assert await page.evaluate("sessionStorage.getItem('builder')") == "preserved"
        assert await (await other.find("#name")).value() == "Other"
        await page.context.recording.stop()
        assert (await bro.check("archive evidence", record=os.environ["CHECK_TEST_INPUT"], provider="local", model="archive-override", base_url=os.environ["VIBIUM_AI_BASE_URL"], reasoning_effort=""))["status"] == "inconclusive"
    finally:
        await bro.stop()

def run_sync():
    from vibium import browser
    if os.environ["CHECK_TEST_ARCHIVE_ONLY"] == "1":
        assert browser.check("archive evidence", record=os.environ["CHECK_TEST_INPUT"], provider="local", model="archive-override", base_url=os.environ["VIBIUM_AI_BASE_URL"], reasoning_effort="")["status"] == "inconclusive"
        return
    bro = browser.start(headless=True)
    try:
        page = bro.page()
        page.go(os.environ["CHECK_TEST_URL"])
        page.evaluate("window.name='original'; sessionStorage.setItem('builder','preserved')")
        other = bro.new_page()
        other.go(os.environ["CHECK_TEST_URL"] + "/other")
        page.context.recording.start(video=False, path=os.environ["CHECK_TEST_OUTPUT"])
        assert page.check(claim)["status"] == "passed"
        assert page.find("#name").value() == "Updated"
        assert page.evaluate("window.name") == "original"
        assert page.evaluate("sessionStorage.getItem('builder')") == "preserved"
        assert other.find("#name").value() == "Other"
        page.context.recording.stop()
        assert bro.check("archive evidence", record=os.environ["CHECK_TEST_INPUT"], provider="local", model="archive-override", base_url=os.environ["VIBIUM_AI_BASE_URL"], reasoning_effort="")["status"] == "inconclusive"
    finally:
        bro.stop()

if os.environ["CHECK_TEST_SYNC"] == "1":
    run_sync()
else:
    asyncio.run(run_async())
