"""Event-setup ordering tests (#351).

``page.on_web_socket()`` is a void callback API, so its install command had nothing
for the caller to await and was handed to ``asyncio.ensure_future`` and forgotten —
a socket opened by the very next command could beat the install and its one-shot
event was lost.

``helpers/fake_engine.py`` stands in for the binary: it installs slowly and only
reports a socket once the install is answered, so these assert ordering rather than
how fast a browser happens to be.
"""

import asyncio
import os
import sys

import pytest

FAKE_ENGINE = os.path.join(os.path.dirname(__file__), "helpers", "fake_engine.py")

# The stand-in is a #! script, which Windows cannot spawn as a binary.
pytestmark = [
    pytest.mark.skipif(sys.platform == "win32", reason="shebang script not spawnable on Windows"),
    pytest.mark.asyncio(loop_scope="function"),
]


@pytest.fixture
async def page(monkeypatch):
    monkeypatch.setenv("VIBIUM_BIN_PATH", FAKE_ENGINE)
    from vibium.async_api import browser

    bro = await browser.start(headless=True)
    try:
        yield await bro.page()
    finally:
        await bro.stop()


async def test_socket_opened_by_next_command_is_still_seen(page):
    seen = []
    page.on_web_socket(lambda ws: seen.append(ws.url()))
    # No await in between — the exact shape that used to lose the event.
    await page.evaluate("openSocket()")

    assert seen == ["ws://127.0.0.1:1/live"], (
        "the command overtook the monitor install, so its socket went unseen"
    )


async def test_registering_twice_installs_the_monitor_once(page):
    seen = []
    page.on_web_socket(lambda ws: seen.append("a"))
    page.on_web_socket(lambda ws: seen.append("b"))
    await page.evaluate("openSocket()")

    # A second install would have re-armed the gate and restarted the delay.
    assert seen == ["a", "b"]


async def test_command_held_behind_setup_fails_when_the_connection_closes(page):
    """The gate can defer a command past close, after which nothing resolves it.

    Closing directly rather than via stop() is the point: stop() waits on the gate
    itself, so it cannot close while a command is still parked behind setup — which
    is exactly the window this covers.
    """
    from vibium import errors

    page.on_web_socket(lambda ws: None)
    held = asyncio.ensure_future(page.evaluate("1"))
    await asyncio.sleep(0)  # let it reach the gate

    await page._client.close()

    with pytest.raises(errors.ConnectionError):
        await asyncio.wait_for(held, timeout=5)
