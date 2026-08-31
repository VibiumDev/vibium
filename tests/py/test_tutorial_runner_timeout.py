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


def _leaky_block(fake_vibium, pidfile):
    """A block that spawns a child named vibium, reports its pid, and hangs."""
    return (
        "import pathlib, subprocess, time\n"
        f"p = subprocess.Popen([{str(fake_vibium)!r}, '60'])\n"
        f"pathlib.Path({str(pidfile)!r}).write_text(str(p.pid))\n"
        "time.sleep(30)\n"
    )


def _assert_terminated(pid):
    """The reaped child is gone or a zombie awaiting its abandoned parent."""
    import subprocess
    import time

    out = ""
    for _ in range(50):
        out = subprocess.run(
            ["ps", "-o", "state=", "-p", str(pid)], capture_output=True, text=True
        ).stdout.strip()
        if not out or out.startswith("Z"):
            return
        time.sleep(0.1)
    raise AssertionError(f"leaked vibium child {pid} still running (state {out!r})")


def test_sync_timeout_kills_leaked_vibium(short_timeout, tmp_path):
    """A timed-out block's vibium child is terminated, not abandoned.

    Incident 12 (#397): the bound fired, but the wedged session held the
    pipes open and blocked the run's exit until the phase watchdog killed
    it, discarding the pytest report.
    """
    import os

    fake_vibium = tmp_path / "vibium"
    # A symlink, not a copy: macOS kills copied platform binaries over
    # their code signature, which would end this test vacuously.
    os.symlink("/bin/sleep", fake_vibium)
    pidfile = tmp_path / "pid"

    with pytest.raises(TimeoutError):
        run_sync_standalone(_leaky_block(fake_vibium, pidfile))

    _assert_terminated(int(pidfile.read_text()))


async def test_async_timeout_kills_leaked_vibium(short_timeout, tmp_path):
    import os

    fake_vibium = tmp_path / "vibium"
    os.symlink("/bin/sleep", fake_vibium)
    pidfile = tmp_path / "pid"

    code = _leaky_block(fake_vibium, pidfile).replace(
        "time.sleep(30)", "import asyncio\nawait asyncio.sleep(30)"
    )
    with pytest.raises(TimeoutError):
        await run_async_standalone(code)

    _assert_terminated(int(pidfile.read_text()))


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
