"""Storage tests — storage, set_storage, clear_storage."""

import pytest

pytestmark = pytest.mark.capability("core")


async def test_storage(fresh_async_browser, test_server):
    ctx = await fresh_async_browser.new_context()
    try:
        vibe = await ctx.new_page()
        await vibe.go(test_server + "/set-cookie")
        state = await ctx.storage()
        assert isinstance(state, dict)
        assert "cookies" in state
    finally:
        await ctx.close()


async def test_set_storage_round_trip(fresh_async_browser, test_server):
    """Get state → clear → set state → verify restored."""
    ctx = await fresh_async_browser.new_context()
    try:
        vibe = await ctx.new_page()
        await vibe.go(test_server)
        await ctx.set_cookies([{
            "name": "rt_cookie",
            "value": "rt_val",
            "domain": "127.0.0.1",
            "path": "/",
        }])
        await vibe.evaluate("localStorage.setItem('rt_key', 'rt_value')")

        # Capture state
        state = await ctx.storage()
        assert len(state["cookies"]) > 0

        # Clear everything
        await ctx.clear_storage()
        cleared = await ctx.storage()
        assert len(cleared["cookies"]) == 0

        # Strip read-only fields from cookies before restoring
        clean_state = {
            "cookies": [
                {"name": c["name"], "value": c["value"], "domain": c["domain"], "path": c["path"]}
                for c in state["cookies"]
            ],
            "origins": state.get("origins", []),
        }

        # Restore state
        await ctx.set_storage(clean_state)
        restored = await ctx.storage()
        cookie = next((c for c in restored["cookies"] if c["name"] == "rt_cookie"), None)
        assert cookie is not None
        assert cookie["value"] == "rt_val"
    finally:
        await ctx.close()


async def test_clear_storage(fresh_async_browser, test_server):
    """Clear cookies + localStorage + sessionStorage."""
    ctx = await fresh_async_browser.new_context()
    try:
        vibe = await ctx.new_page()
        await vibe.go(test_server)
        await ctx.set_cookies([{
            "name": "clr",
            "value": "val",
            "domain": "127.0.0.1",
            "path": "/",
        }])
        await vibe.evaluate("localStorage.setItem('clr_key', 'val')")

        await ctx.clear_storage()

        state = await ctx.storage()
        assert len(state["cookies"]) == 0
    finally:
        await ctx.close()
