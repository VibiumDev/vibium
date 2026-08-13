"""Event-setup ordering tests (#351).

page.on_web_socket() is a void callback API, so its install command had
nothing for the caller to await. A socket opened by the very next command
could beat the install and its one-shot event was lost.

helpers/fake_engine.py stands in for the binary: it installs slowly and only
reports a socket once the install is answered, so these tests assert ordering
rather than how fast a browser happens to be.
"""

import asyncio
import os
import sys

import pytest
import pytest_asyncio

FAKE_ENGINE = os.path.join(os.path.dirname(__file__), "helpers", "fake_engine.py")

pytestmark = [
    # The stand-in is a shebang script, which Windows cannot spawn as a binary.
    pytest.mark.skipif(
        sys.platform == "win32", reason="shebang script not spawnable on Windows"
    ),
    # Tests default to the module-scoped loop (pytest.ini), but the fixture
    # below runs on a function-scoped one. Everything must share a loop or
    # the receiver resolves futures the test loop never wakes up for.
    pytest.mark.asyncio(loop_scope="function"),
]


@pytest_asyncio.fixture
async def page():
    from vibium.async_api import browser

    bro = await browser.start(headless=True, executable_path=FAKE_ENGINE)
    try:
        yield await bro.page()
    finally:
        await bro.stop()


async def test_socket_opened_by_next_command_is_still_seen(page):
    seen = []
    page.on_web_socket(lambda ws: seen.append(ws.url()))
    # No await in between: the exact shape that used to lose the event.
    await page.evaluate("openSocket()")

    assert seen == ["ws://127.0.0.1:1/live"], (
        "the command overtook the monitor install, so its socket went unseen"
    )


async def test_registering_twice_installs_the_monitor_once(page):
    seen = []
    page.on_web_socket(lambda ws: seen.append("a"))
    page.on_web_socket(lambda ws: seen.append("b"))
    await page.evaluate("openSocket()")

    assert seen == ["a", "b"]
    assert await page.evaluate("__installCount") == 1, (
        "a second registration re-sent the install command"
    )


async def test_failed_install_is_retried_by_the_next_registration(monkeypatch):
    # The stand-in reads this at startup, so set it before launching.
    monkeypatch.setenv("FAKE_ENGINE_FAIL_SETUP", "once")
    from vibium.async_api import browser

    bro = await browser.start(headless=True, executable_path=FAKE_ENGINE)
    try:
        page = await bro.page()
        seen = []
        page.on_web_socket(lambda ws: seen.append("a"))
        # Wait for the failure to land; the async API only debug-logs it.
        try:
            await page._when_web_socket_setup()
        except Exception:
            pass
        page.on_web_socket(lambda ws: seen.append("b"))
        await page.evaluate("openSocket()")

        assert seen == ["a", "b"], (
            "the second registration must retry the install, "
            "healing the first callback too"
        )
        assert await page.evaluate("__installCount") == 2, (
            "the retry must re-send the install command"
        )
    finally:
        await bro.stop()


async def test_command_parked_behind_setup_fails_when_the_connection_closes(page):
    # Closing the client directly rather than via stop() is the point: stop()
    # itself goes through the gate, so it cannot close the connection while a
    # command is still parked behind setup. That window is what this covers.
    from vibium import errors

    page.on_web_socket(lambda ws: None)
    held = asyncio.ensure_future(page.evaluate("1"))
    await asyncio.sleep(0)  # let it reach the gate

    await page._client.close()

    with pytest.raises(errors.ConnectionError):
        await asyncio.wait_for(held, timeout=5)
