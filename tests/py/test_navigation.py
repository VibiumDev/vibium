"""Navigation tests — go, back, forward, reload, url, title, content, waitUntil.url, waitUntil.loaded (9 async tests)."""

import pytest


async def test_go(async_page, test_server):
    await async_page.go(test_server)
    url = await async_page.url()
    assert test_server.replace("http://", "") in url


async def test_back_forward(async_page, test_server):
    await async_page.go(test_server)
    await async_page.go(test_server + "/subpage")
    assert await async_page.title() == "Subpage"
    await async_page.back()
    assert await async_page.title() == "Test App"
    await async_page.forward()
    assert await async_page.title() == "Subpage"


async def test_reload(async_page, test_server):
    await async_page.go(test_server)
    await async_page.reload()
    assert await async_page.title() == "Test App"


async def test_url(async_page, test_server):
    await async_page.go(test_server + "/subpage")
    url = await async_page.url()
    assert "/subpage" in url


async def test_title(async_page, test_server):
    await async_page.go(test_server)
    assert await async_page.title() == "Test App"


async def test_content(async_page, test_server):
    await async_page.go(test_server)
    html = await async_page.content()
    assert "Welcome to test-app" in html


async def test_wait_until_url(async_page, test_server):
    await async_page.go(test_server)
    link = await async_page.find('a[href="/subpage"]')
    await link.click()
    await async_page.wait_until.url("**/subpage")
    assert "/subpage" in await async_page.url()


async def test_wait_until_loaded(async_page, test_server):
    await async_page.go(test_server)
    await async_page.wait_until.loaded()
    assert await async_page.title() == "Test App"


async def test_wait_until_url_timeout(async_page, test_server):
    await async_page.go(test_server)
    with pytest.raises(Exception):
        await async_page.wait_until.url("**/never-going-here", timeout=1000)
