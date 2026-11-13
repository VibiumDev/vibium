"""A failed sync browser.start() must stop its event-loop thread (#722).

No browser: the launch fails at spawn time on a nonexistent binary, which is
exactly the path that used to leak the loop thread.
"""

import threading

import pytest

from vibium.sync_api import browser


def test_failed_start_stops_loop_thread():
    before = set(threading.enumerate())

    with pytest.raises(Exception):
        browser.start(executable_path="/nonexistent/vibium-722")

    leaked = [t for t in threading.enumerate() if t not in before and t.is_alive()]
    assert not leaked, f"failed start leaked event-loop threads: {leaked}"
