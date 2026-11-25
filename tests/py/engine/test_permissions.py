"""page.set_permission tests (#711)."""

import pytest

pytestmark = pytest.mark.capability("core")

STATE = "navigator.permissions.query({ name: 'geolocation' }).then((s) => s.state)"


async def test_set_permission(async_page, test_server):
    await async_page.go(test_server)
    await async_page.set_permission("geolocation", "granted")
    assert await async_page.evaluate(STATE) == "granted"
    await async_page.set_permission("geolocation", "denied")
    assert await async_page.evaluate(STATE) == "denied"
    await async_page.set_permission("geolocation", "prompt", origin=test_server)
    assert await async_page.evaluate(STATE) == "prompt"


def test_set_permission_sync(sync_page, test_server):
    sync_page.go(test_server)
    sync_page.set_permission("geolocation", "granted")
    assert sync_page.evaluate(STATE) == "granted"
    sync_page.set_permission("geolocation", "prompt")
    assert sync_page.evaluate(STATE) == "prompt"
