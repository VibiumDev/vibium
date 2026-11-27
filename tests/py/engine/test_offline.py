"""page.set_offline tests (#713)."""

import pytest

pytestmark = pytest.mark.capability("core")

FETCHED = "fetch('/json').then(() => 'served').catch(() => 'failed')"


async def test_set_offline(async_page, test_server):
    await async_page.go(test_server)
    await async_page.set_offline(True)
    try:
        assert await async_page.evaluate(FETCHED) == "failed"
    finally:
        await async_page.set_offline(False)
    assert await async_page.evaluate(FETCHED) == "served"


def test_set_offline_sync(sync_page, test_server):
    sync_page.go(test_server)
    sync_page.set_offline(True)
    try:
        assert sync_page.evaluate(FETCHED) == "failed"
    finally:
        sync_page.set_offline(False)
    assert sync_page.evaluate(FETCHED) == "served"
