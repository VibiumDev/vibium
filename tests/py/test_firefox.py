"""Firefox smoke tests (sync API).

Skip when Firefox is not installed, so the suite stays green on machines
that only have Chrome. Install with: vibium install --engine firefox
"""

import os
import subprocess

import pytest

from vibium import browser, firefox


def _firefox_installed() -> bool:
    bin_path = os.environ.get("VIBIUM_BIN_PATH")
    if not bin_path:
        return False
    try:
        result = subprocess.run(
            [bin_path, "is-installed", "--engine", "firefox"],
            capture_output=True,
            timeout=10,
        )
        return result.returncode == 0
    except (subprocess.TimeoutExpired, subprocess.SubprocessError):
        return False


pytestmark = pytest.mark.skipif(
    not _firefox_installed(), reason="Firefox not installed"
)


def test_firefox_smoke(test_server):
    # Named launcher: firefox.start() == browser.start(engine="firefox")
    bro = firefox.start(headless=True)
    try:
        vibe = bro.page()
        vibe.go(test_server)
        assert vibe.title() == "Test App"
        assert len(vibe.screenshot()) > 1000
    finally:
        bro.stop()


def test_firefox_screencast(test_server):
    bro = browser.start(engine="firefox", headless=True)
    try:
        vibe = bro.page()
        vibe.go(test_server)
        try:
            vibe.screencast.start()
        except Exception as err:
            # Firefox gains BiDi screencast in 154; self-skip on older builds
            # so this activates on its own once release catches up.
            if "not supported" in str(err):
                pytest.skip("this Firefox does not support screencast yet")
            raise
        # Navigate while recording: a screencast stopped with no paints in
        # between is a valid but frameless (few hundred byte) WebM.
        vibe.go(test_server)
        video = vibe.screencast.stop()
        assert len(video) > 1000
        assert video[:4] == b"\x1a\x45\xdf\xa3"  # WebM EBML magic
    finally:
        bro.stop()
