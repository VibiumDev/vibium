"""Cookie tests — cookies, setCookies, clearCookies, storageState, addInitScript (11 async tests)."""

import pytest


# --- Read ---

async def test_cookies_server_set(fresh_async_browser, test_server):
    ctx = await fresh_async_browser.new_context()
    try:
        vibe = await ctx.new_page()
        await vibe.go(test_server + "/set-cookie")
        cookies = await ctx.cookies()
        assert isinstance(cookies, list)
        names = [c["name"] for c in cookies]
        assert "test_cookie" in names
    finally:
        await ctx.close()


async def test_cookies_filter_url(fresh_async_browser, test_server):
    ctx = await fresh_async_browser.new_context()
    try:
        vibe = await ctx.new_page()
        await vibe.go(test_server + "/set-cookie")
        cookies = await ctx.cookies(urls=[test_server])
        assert isinstance(cookies, list)
        assert len(cookies) >= 1
    finally:
        await ctx.close()


# --- Set ---

async def test_set_cookies_domain(fresh_async_browser, test_server):
    ctx = await fresh_async_browser.new_context()
    try:
        vibe = await ctx.new_page()
        await vibe.go(test_server)
        await ctx.set_cookies([{
            "name": "custom",
            "value": "test123",
            "domain": "127.0.0.1",
            "path": "/",
        }])
        cookies = await ctx.cookies()
        names = [c["name"] for c in cookies]
        assert "custom" in names
    finally:
        await ctx.close()


async def test_set_cookies_url(fresh_async_browser, test_server):
    ctx = await fresh_async_browser.new_context()
    try:
        vibe = await ctx.new_page()
        await vibe.go(test_server)
        await ctx.set_cookies([{
            "name": "url_cookie",
            "value": "from_url",
            "url": test_server,
        }])
        cookies = await ctx.cookies()
        names = [c["name"] for c in cookies]
        assert "url_cookie" in names
    finally:
        await ctx.close()


# --- Clear ---

async def test_clear_cookies(fresh_async_browser, test_server):
    ctx = await fresh_async_browser.new_context()
    try:
        vibe = await ctx.new_page()
        await vibe.go(test_server + "/set-cookie")
        cookies_before = await ctx.cookies()
        assert len(cookies_before) >= 1
        await ctx.clear_cookies()
        cookies_after = await ctx.cookies()
        assert len(cookies_after) == 0
    finally:
        await ctx.close()


# --- Storage ---

async def test_storage_state(fresh_async_browser, test_server):
    ctx = await fresh_async_browser.new_context()
    try:
        vibe = await ctx.new_page()
        await vibe.go(test_server + "/set-cookie")
        state = await ctx.storage_state()
        assert isinstance(state, dict)
        assert "cookies" in state
    finally:
        await ctx.close()


# --- Init scripts ---

async def test_add_init_script(fresh_async_browser, test_server):
    ctx = await fresh_async_browser.new_context()
    try:
        await ctx.add_init_script("window.__initFlag = 'set'")
        vibe = await ctx.new_page()
        await vibe.go(test_server)
        result = await vibe.eval("window.__initFlag")
        assert result == "set"
    finally:
        await ctx.close()


async def test_init_script_persists(fresh_async_browser, test_server):
    ctx = await fresh_async_browser.new_context()
    try:
        await ctx.add_init_script("window.__persistent = true")
        vibe = await ctx.new_page()
        await vibe.go(test_server)
        assert await vibe.eval("window.__persistent") is True
        await vibe.go(test_server + "/subpage")
        assert await vibe.eval("window.__persistent") is True
    finally:
        await ctx.close()


# --- Cookie isolation ---

async def test_context_isolation(fresh_async_browser, test_server):
    ctx1 = await fresh_async_browser.new_context()
    ctx2 = await fresh_async_browser.new_context()
    try:
        p1 = await ctx1.new_page()
        await p1.go(test_server + "/set-cookie")
        c1 = await ctx1.cookies()
        c2 = await ctx2.cookies()
        assert len(c1) >= 1
        assert len(c2) == 0
    finally:
        await ctx1.close()
        await ctx2.close()


async def test_set_and_read_cookie(fresh_async_browser, test_server):
    ctx = await fresh_async_browser.new_context()
    try:
        vibe = await ctx.new_page()
        await vibe.go(test_server)
        await ctx.set_cookies([{
            "name": "roundtrip",
            "value": "works",
            "domain": "127.0.0.1",
            "path": "/",
        }])
        cookies = await ctx.cookies()
        cookie = next((c for c in cookies if c["name"] == "roundtrip"), None)
        assert cookie is not None
        assert cookie["value"] == "works"
    finally:
        await ctx.close()


# --- Checkpoint ---

async def test_round_trip(fresh_async_browser, test_server):
    """Set cookies, read back, clear, verify empty."""
    ctx = await fresh_async_browser.new_context()
    try:
        vibe = await ctx.new_page()
        await vibe.go(test_server)
        await ctx.set_cookies([{
            "name": "test",
            "value": "roundtrip",
            "domain": "127.0.0.1",
            "path": "/",
        }])
        cookies = await ctx.cookies()
        assert any(c["name"] == "test" for c in cookies)
        await ctx.clear_cookies()
        cookies = await ctx.cookies()
        assert len(cookies) == 0
    finally:
        await ctx.close()
