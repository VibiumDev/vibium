#!/usr/bin/env python3
"""A stand-in for the vibium binary, for tests that assert command ordering
rather than browser behavior.

It answers vibium:page.onWebSocket after a delay and reports a socket only
once that install has been answered, like the real router, which replies
after subscribing and injecting the monitor. A client that fires the install
and moves on loses the event its next command produces (#351).

Point a client at it with browser.start(executable_path=...).
FAKE_ENGINE_SETUP_DELAY_MS adjusts the install delay (default 300).
FAKE_ENGINE_FAIL_SETUP=1 makes every install fail; =once fails only the
first, so tests can pin that a failed install is retried.
"""

import json
import os
import sys
import threading

SETUP_DELAY_S = float(os.environ.get("FAKE_ENGINE_SETUP_DELAY_MS", "300")) / 1000
FAIL_SETUP = os.environ.get("FAKE_ENGINE_FAIL_SETUP", "")

_lock = threading.Lock()
_monitor_installed = False
_install_count = 0


def write(msg):
    with _lock:
        sys.stdout.write(json.dumps(msg) + "\n")
        sys.stdout.flush()


def ok(cmd_id, result=None):
    write({"id": cmd_id, "type": "success", "result": result or {}})


def install_monitor(cmd_id, fail):
    global _monitor_installed
    if fail:
        write({"id": cmd_id, "type": "error", "error": "unsupported operation",
               "message": "no preload scripts here"})
        return
    _monitor_installed = True
    ok(cmd_id)


write({"method": "vibium:lifecycle.ready", "params": {}})

for line in sys.stdin:
    if not line.strip():
        continue
    cmd = json.loads(line)
    cmd_id, method, params = cmd.get("id"), cmd.get("method"), cmd.get("params") or {}

    if method in ("vibium:browser.page", "vibium:browser.newPage"):
        ok(cmd_id, {"context": "ctx-1", "userContext": "default"})
    elif method == "vibium:page.onWebSocket":
        # Counted before deciding failure, so __installCount below includes
        # failed attempts and a test can pin that a retry re-sent the install.
        _install_count += 1
        fail = FAIL_SETUP == "1" or (FAIL_SETUP == "once" and _install_count == 1)
        threading.Timer(SETUP_DELAY_S, install_monitor, [cmd_id, fail]).start()
    elif method == "vibium:page.eval":
        # Test back-channel, not protocol behavior: reports how many installs
        # this engine was sent. Race-free because the eval goes through the
        # gate, so any pending install is answered before this is.
        if str(params.get("expression", "")) == "__installCount":
            ok(cmd_id, {"value": _install_count})
        else:
            # Stands in for `new WebSocket(...)`: only a live monitor sees it.
            if _monitor_installed and "openSocket" in str(params.get("expression", "")):
                write({"method": "vibium:ws.created",
                       "params": {"context": "ctx-1", "id": 1, "url": "ws://127.0.0.1:1/live"}})
            ok(cmd_id, {"value": None})
    else:
        ok(cmd_id)
