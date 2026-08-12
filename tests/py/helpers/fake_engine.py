#!/usr/bin/env python3
"""A stand-in for the vibium binary, for tests that need to observe ordering
rather than a real browser.

It installs the WebSocket monitor slowly and reports a socket only once that
install has been answered — exactly like the router, which replies to
vibium:page.onWebSocket after subscribing, adding the preload script and
injecting the monitor. A client that fires the install and moves on therefore
loses the event its next command produces, which is the bug in #351.

Point a client at it with VIBIUM_BIN_PATH.
"""

import json
import os
import sys
import threading

SETUP_DELAY_S = float(os.environ.get("FAKE_ENGINE_SETUP_DELAY_MS", "300")) / 1000
FAIL_SETUP = os.environ.get("FAKE_ENGINE_FAIL_SETUP") == "1"

# The clients run `vibium is-installed` before launching.
if len(sys.argv) > 1 and sys.argv[1] == "is-installed":
    sys.exit(0)

_lock = threading.Lock()
_monitor_installed = False


def write(msg):
    with _lock:
        sys.stdout.write(json.dumps(msg) + "\n")
        sys.stdout.flush()


def ok(cmd_id, result=None):
    write({"id": cmd_id, "type": "success", "result": result or {}})


def install_monitor(cmd_id):
    global _monitor_installed
    if FAIL_SETUP:
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
        threading.Timer(SETUP_DELAY_S, install_monitor, [cmd_id]).start()
    elif method == "vibium:page.eval":
        # Stands in for `new WebSocket(...)`: only a live monitor sees it.
        if _monitor_installed and "openSocket" in str(params.get("expression", "")):
            write({"method": "vibium:ws.created",
                   "params": {"context": "ctx-1", "id": 1, "url": "ws://127.0.0.1:1/live"}})
        ok(cmd_id, {"value": None})
    else:
        ok(cmd_id)
