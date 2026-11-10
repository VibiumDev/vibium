"""Persistent profile tests (#715) — user_data_dir plumbing and reuse."""

import os
import tempfile

from vibium import browser


def test_user_data_dir_persists_across_sessions(test_server):
    with tempfile.TemporaryDirectory(prefix="py-profile-") as parent:
        profile = os.path.join(parent, "profile")

        bro = browser.start(headless=True, user_data_dir=profile)
        try:
            vibe = bro.page()
            vibe.go(test_server)
            vibe.evaluate('localStorage.setItem("persisted_key", "yes")')
            # Let the write reach disk before stopping: storage commits are
            # asynchronous, and a host where the browser cannot complete a
            # clean shutdown never gets the exit-time flush.
            vibe.wait(2000)
        finally:
            bro.stop()

        assert os.path.isdir(profile), "profile dir must survive browser close"
        assert os.listdir(profile), "profile dir must keep its contents"

        bro = browser.start(headless=True, user_data_dir=profile)
        try:
            vibe = bro.page()
            vibe.go(test_server)
            assert vibe.evaluate('localStorage.getItem("persisted_key")') == "yes"
        finally:
            bro.stop()
