"""page.expose in both forms (#298).

A string injects JS source that defines window[name] inside the page. A
callable exposes a host function: the page calls window[name](*args), the
callable runs in this process, and its return value resolves the page's
promise via vibium:expose.call/result. The sync API additionally runs host
functions on a worker thread, so they may call back into the sync API.
"""

import asyncio

import pytest

pytestmark = pytest.mark.capability("core")
import pytest_asyncio


@pytest_asyncio.fixture(scope="module", loop_scope="module")
async def expose_browser():
    from vibium.async_api import browser
    bro = await browser.start(headless=True)
    yield bro
    await bro.stop()


async def test_source_form_defines_a_page_function(expose_browser):
    vibe = await expose_browser.new_page()
    await vibe.set_content("<p>x</p>")

    await vibe.expose("vibium_double", "(n) => n * 2")
    assert await vibe.evaluate("window.vibium_double(21)") == 42
    await vibe.close()


async def test_host_function_runs_and_returns_its_value(expose_browser):
    vibe = await expose_browser.new_page()
    await vibe.set_content("<p>x</p>")

    await vibe.expose("vibium_add", lambda a, b: a + b)
    assert await vibe.evaluate("window.vibium_add(2, 3)") == 5
    await vibe.close()


async def test_async_host_function_and_object_result(expose_browser):
    vibe = await expose_browser.new_page()
    await vibe.set_content("<p>x</p>")

    async def lookup(key):
        await asyncio.sleep(0.02)
        return {"key": key, "found": True}

    await vibe.expose("vibium_lookup", lookup)
    result = await vibe.evaluate("window.vibium_lookup('user')")
    assert result == {"key": "user", "found": True}
    await vibe.close()


async def test_host_error_rejects_the_page_promise(expose_browser):
    vibe = await expose_browser.new_page()
    await vibe.set_content("<p>x</p>")

    def explode():
        raise RuntimeError("no fuel")

    await vibe.expose("vibium_explode", explode)
    message = await vibe.evaluate("window.vibium_explode().catch((e) => e.message)")
    assert message == "no fuel"
    await vibe.close()


async def test_host_function_survives_navigation_and_replacement(expose_browser, test_server):
    vibe = await expose_browser.new_page()
    await vibe.go(test_server + "/")

    await vibe.expose("vibium_answer", lambda: "first")
    await vibe.expose("vibium_answer", lambda: "second")
    await vibe.reload()
    assert await vibe.evaluate("window.vibium_answer()") == "second"
    await vibe.close()


def test_sync_host_function_can_call_the_sync_api(test_server):
    # The host function runs on a worker thread, so calling back into the
    # sync API from inside it must not deadlock the event loop delivering
    # the call.
    from vibium import browser

    bro = browser.start(headless=True)
    try:
        vibe = bro.page()
        vibe.go(test_server + "/")

        def titled(prefix):
            return prefix + ":" + vibe.title()

        vibe.expose("vibium_titled", titled)
        result = vibe.evaluate("window.vibium_titled('got')")
        assert result.startswith("got:")
    finally:
        bro.stop()
