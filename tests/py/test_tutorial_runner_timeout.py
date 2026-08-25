"""The tutorial runners must bound a hung block so it fails its own test.

The event-delivery hangs (#397 incidents 9 and 10) ran until the 600s phase
watchdog killed pytest, which discarded the captured vibium stderr that
would have said why. A block that fails inside the runner's own bound gets
a normal pytest report, captured output included.
"""

import pytest

from helpers import tutorial_runner
from helpers.tutorial_runner import run_async_standalone, run_sync_standalone


@pytest.fixture
def short_timeout(monkeypatch):
    monkeypatch.setattr(tutorial_runner, "TUTORIAL_TIMEOUT", 1)


async def test_async_hung_block_times_out(short_timeout):
    with pytest.raises(TimeoutError):
        await run_async_standalone("import asyncio\nawait asyncio.sleep(30)")


def test_sync_hung_block_times_out(short_timeout):
    with pytest.raises(TimeoutError):
        run_sync_standalone("import time\ntime.sleep(30)")


async def test_async_block_still_runs_and_returns():
    await run_async_standalone("assert base_url == 'x'", base_url="x")


def test_sync_block_still_runs_and_returns():
    run_sync_standalone("assert base_url == 'x'", base_url="x")


def test_sync_block_failure_propagates():
    with pytest.raises(AssertionError, match="tutorial says no"):
        run_sync_standalone("assert False, 'tutorial says no'")


async def test_async_block_failure_propagates():
    with pytest.raises(AssertionError, match="tutorial says no"):
        await run_async_standalone("assert False, 'tutorial says no'")
