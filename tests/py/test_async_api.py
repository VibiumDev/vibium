"""Core async API smoke tests (8 tests)."""

import pytest


async def test_browser_launch_returns_browser(async_browser):
    from vibium.async_api import Browser
    assert isinstance(async_browser, Browser)


async def test_page_go_navigates(async_page, test_server):
    await async_page.go(test_server)
    title = await async_page.title()
    assert title == "Test App"


async def test_page_screenshot_returns_png(async_page, test_server):
    await async_page.go(test_server)
    data = await async_page.screenshot()
    assert isinstance(data, bytes)
    assert len(data) > 100
    # PNG magic bytes
    assert data[0] == 0x89
    assert data[1] == 0x50
    assert data[2] == 0x4E
    assert data[3] == 0x47


async def test_page_eval_executes_js(async_page, test_server):
    await async_page.go(test_server + "/eval")
    result = await async_page.eval("window.testVal")
    assert result == 42


async def test_page_find_locates_element(async_page, test_server):
    await async_page.go(test_server)
    el = await async_page.find("h1")
    text = await el.text()
    assert "Welcome" in text


async def test_element_click(async_page, test_server):
    await async_page.go(test_server)
    link = await async_page.find('a[href="/subpage"]')
    await link.click()
    await async_page.wait_until.url("**/subpage")
    title = await async_page.title()
    assert title == "Subpage"


async def test_element_type(async_page, test_server):
    await async_page.go(test_server + "/inputs")
    inp = await async_page.find("#text-input")
    await inp.type("hello")
    val = await inp.value()
    assert val == "hello"


async def test_element_text(async_page, test_server):
    await async_page.go(test_server)
    p = await async_page.find("#info")
    text = await p.text()
    assert text == "Some info text"
