"""Browser lifecycle tests — page, newPage, pages, close, bringToFront, context (9 async tests)."""

import pytest


async def test_page_returns_default(fresh_async_browser):
    vibe = await fresh_async_browser.page()
    assert vibe.id


async def test_new_page_unique_id(fresh_async_browser):
    p1 = await fresh_async_browser.page()
    p2 = await fresh_async_browser.new_page()
    assert p1.id != p2.id


async def test_pages_lists_all(fresh_async_browser):
    await fresh_async_browser.new_page()
    pages = await fresh_async_browser.pages()
    assert len(pages) >= 2


async def test_page_close(fresh_async_browser, test_server):
    p = await fresh_async_browser.new_page()
    await p.go(test_server)
    pid = p.id
    await p.close()
    pages = await fresh_async_browser.pages()
    assert all(pg.id != pid for pg in pages)


async def test_bring_to_front(fresh_async_browser, test_server):
    vibe = await fresh_async_browser.page()
    await vibe.go(test_server)
    await vibe.bring_to_front()  # should not throw
    assert await vibe.title() == "Test App"


async def test_new_context(fresh_async_browser):
    ctx = await fresh_async_browser.new_context()
    assert ctx.id
    await ctx.close()


async def test_context_close(fresh_async_browser, test_server):
    ctx = await fresh_async_browser.new_context()
    p = await ctx.new_page()
    await p.go(test_server)
    await ctx.close()
    # context is gone; creating a new one should still work
    ctx2 = await fresh_async_browser.new_context()
    assert ctx2.id
    await ctx2.close()


async def test_multiple_pages_independent(fresh_async_browser, test_server):
    p1 = await fresh_async_browser.page()
    p2 = await fresh_async_browser.new_page()
    await p1.go(test_server)
    await p2.go(test_server + "/subpage")
    assert await p1.title() == "Test App"
    assert await p2.title() == "Subpage"


async def test_on_page(fresh_async_browser):
    pages = []
    fresh_async_browser.on_page(lambda p: pages.append(p))
    await fresh_async_browser.new_page()
    import asyncio
    await asyncio.sleep(0.2)
    assert len(pages) == 1
    assert pages[0].id
    fresh_async_browser.remove_all_listeners("page")


async def test_on_popup(fresh_async_browser, test_server):
    popups = []
    fresh_async_browser.on_popup(lambda p: popups.append(p))
    p = await fresh_async_browser.page()
    await p.go(test_server)
    await p.evaluate("window.open('about:blank')")
    import asyncio
    await asyncio.sleep(0.2)
    assert len(popups) == 1
    assert popups[0].id
    fresh_async_browser.remove_all_listeners("popup")


async def test_remove_all_listeners(fresh_async_browser):
    pages = []
    fresh_async_browser.on_page(lambda p: pages.append(p))
    await fresh_async_browser.new_page()
    import asyncio
    await asyncio.sleep(0.2)
    assert len(pages) == 1

    fresh_async_browser.remove_all_listeners("page")
    await fresh_async_browser.new_page()
    await asyncio.sleep(0.2)
    assert len(pages) == 1, "Should still be 1 after removing listener"


def test_on_page_sync(fresh_sync_browser):
    import time
    initial = fresh_sync_browser.page()
    pages = []
    fresh_sync_browser.on_page(lambda p: pages.append(p))
    fresh_sync_browser.new_page()
    time.sleep(0.3)
    # Filter out the initial page's late contextCreated event if it arrived
    new_pages = [p for p in pages if p.id != initial.id]
    assert len(new_pages) == 1
    assert new_pages[0].id
    fresh_sync_browser.remove_all_listeners("page")


def test_on_popup_sync(fresh_sync_browser, test_server):
    import time
    popups = []
    fresh_sync_browser.on_popup(lambda p: popups.append(p))
    p = fresh_sync_browser.page()
    p.go(test_server)
    p.evaluate("window.open('about:blank')")
    time.sleep(0.3)
    assert len(popups) == 1
    assert popups[0].id
    fresh_sync_browser.remove_all_listeners("popup")


def test_remove_all_listeners_sync(fresh_sync_browser):
    import time
    initial = fresh_sync_browser.page()
    pages = []
    fresh_sync_browser.on_page(lambda p: pages.append(p))
    fresh_sync_browser.new_page()
    time.sleep(0.3)
    new_pages = [p for p in pages if p.id != initial.id]
    assert len(new_pages) == 1

    fresh_sync_browser.remove_all_listeners("page")
    before_count = len(pages)
    fresh_sync_browser.new_page()
    time.sleep(0.3)
    assert len(pages) == before_count, "Should not grow after removing listener"


async def test_browser_close(test_server):
    from vibium.async_api import browser
    bro = await browser.launch(headless=True)
    vibe = await bro.page()
    await vibe.go(test_server)
    await bro.close()
    # After close, should not be able to use the browser
    # (we just verify close completes without error)
