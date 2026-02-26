"""Auto-wait tests — find waits, click waits, timeout errors (5 async tests)."""

import pytest


async def test_find_waits(fresh_async_browser, test_server):
    """find() auto-waits for elements to appear."""
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/dynamic-loading")
    el = await vibe.find("#loaded", timeout=5000)
    text = await el.text()
    assert text == "Loaded!"


async def test_click_waits(fresh_async_browser, test_server):
    """click() waits for element to be actionable."""
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/dynamic-loading")
    el = await vibe.find("#loaded", timeout=5000)
    await el.click(timeout=5000)
    # Just verify click completes without error


async def test_find_timeout(fresh_async_browser, test_server):
    """find() times out for non-existent element."""
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server)
    with pytest.raises(Exception):
        await vibe.find("#does-not-exist", timeout=1000)


async def test_timeout_error_message(fresh_async_browser, test_server):
    """Timeout error message is clear."""
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server)
    try:
        await vibe.find("#nonexistent-element-xyz", timeout=1000)
        assert False, "Should have thrown"
    except Exception as err:
        msg = str(err).lower()
        assert "timeout" in msg or "nonexistent" in msg


async def test_navigation_error_message(fresh_async_browser, test_server):
    """Navigation to invalid domain gives clear error."""
    vibe = await fresh_async_browser.new_page()
    with pytest.raises(Exception):
        await vibe.go("https://test.invalid")
