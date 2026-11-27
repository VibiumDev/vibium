"""page.set_locale tests (#717)."""

import pytest

pytestmark = pytest.mark.capability("core")

FORMATTED = "(1234.5).toLocaleString()"


async def test_set_locale(async_page, test_server):
    await async_page.go(test_server)
    before = await async_page.evaluate(FORMATTED)
    await async_page.set_locale("de-DE")
    try:
        assert await async_page.evaluate("navigator.language") == "de-DE"
        assert await async_page.evaluate(FORMATTED) == "1.234,5"
    finally:
        await async_page.set_locale(None)
    assert await async_page.evaluate(FORMATTED) == before


def test_set_locale_sync(sync_page, test_server):
    sync_page.go(test_server)
    sync_page.set_locale("de-DE")
    try:
        assert sync_page.evaluate(FORMATTED) == "1.234,5"
    finally:
        sync_page.set_locale(None)
