"""Sync event-setup ordering tests (#351).

The sync API has no coroutine for the caller to await, so its blocking
on_web_socket() must not return until the engine has acknowledged the
install, and must raise if the install was rejected, the only place a sync
caller can see it. See helpers/fake_engine.py.
"""

import os
import sys

import pytest

FAKE_ENGINE = os.path.join(os.path.dirname(__file__), "helpers", "fake_engine.py")

# The stand-in is a shebang script, which Windows cannot spawn as a binary.
pytestmark = pytest.mark.skipif(
    sys.platform == "win32", reason="shebang script not spawnable on Windows"
)


@pytest.fixture
def sync_page():
    from vibium import browser

    bro = browser.start(headless=True, executable_path=FAKE_ENGINE)
    try:
        yield bro.page()
    finally:
        bro.stop()


def test_socket_opened_by_next_call_is_still_seen(sync_page):
    seen = []
    sync_page.on_web_socket(lambda ws: seen.append(ws.url()))
    sync_page.evaluate("openSocket()")
    # Drain: the event reached the loop thread with the response above.
    sync_page.evaluate("1")

    assert seen == ["ws://127.0.0.1:1/live"], (
        "on_web_socket returned before the install was acknowledged, "
        "so the socket went unseen"
    )


def test_rejected_install_raises_from_on_web_socket(monkeypatch):
    # The stand-in reads this at startup, so set it before launching.
    monkeypatch.setenv("FAKE_ENGINE_FAIL_SETUP", "1")
    from vibium import browser

    bro = browser.start(headless=True, executable_path=FAKE_ENGINE)
    try:
        with pytest.raises(Exception, match="no preload scripts here"):
            bro.page().on_web_socket(lambda ws: None)
    finally:
        bro.stop()
