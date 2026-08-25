"""Dump the Python client's public surface as JSON for the apidrift checker.

Keys are the receiver names api.md uses in its Python column; values are the
public members the sync client actually exports. Run from an environment
where vibium is importable:

    python scripts/apidrift_python.py | apidrift check -surface python -spec docs/reference/api.md -actual -
"""

import inspect
import json
import sys


def members(obj):
    out = set()
    for name, value in inspect.getmembers(obj):
        if name.startswith("_"):
            continue
        if not (callable(value) or isinstance(value, property)):
            continue
        if inspect.isclass(obj) and not defined_by_vibium(obj, name):
            continue  # inherited from dict or another stdlib base
        out.add(name)
    return sorted(out)


def defined_by_vibium(cls, name):
    for klass in inspect.getmro(cls):
        if name in vars(klass):
            return getattr(klass, "__module__", "").startswith("vibium")
    return False


def main():
    from vibium import browser
    from vibium.sync_api import page as page_mod
    from vibium.sync_api.page import (
        Page,
        Keyboard,
        Mouse,
        Touch,
        _SyncCaptureNamespace,
    )
    from vibium.sync_api.element import Element
    from vibium.sync_api.context import BrowserContext
    from vibium.sync_api.clock import Clock
    from vibium.sync_api.route import Route
    from vibium.sync_api.browser import Browser

    surface = {
        "browser": sorted(set(members(browser)) | set(members(Browser))),
        "page": members(Page),
        "page.capture": members(_SyncCaptureNamespace),
        "el": members(Element),
        "context": members(BrowserContext),
        "keyboard": members(Keyboard),
        "mouse": members(Mouse),
        "touch": members(Touch),
        "clock": members(Clock),
        "route": members(Route),
    }

    # Wrapper classes for events and recordings live beside Page or in their
    # own modules depending on the client's layout; resolve what exists and
    # report the rest as absent receivers so the checker flags the gap
    # instead of this script crashing on a refactor.
    optional = {
        "dialog": ("vibium.sync_api.dialog", "Dialog"),
        # The sync API hands back these wrapper objects from the async
        # modules (and a dict-backed SyncDownload defined beside Page).
        "download": ("vibium.sync_api.page", "SyncDownload"),
        "request": ("vibium.async_api.network", "Request"),
        "response": ("vibium.async_api.network", "Response"),
        "message": ("vibium.async_api.console", "ConsoleMessage"),
        "socket": ("vibium.async_api.websocket_info", "WebSocketInfo"),
        "recording": ("vibium.sync_api.recording", "Recording"),
    }
    for key, (mod_name, cls_name) in optional.items():
        try:
            mod = __import__(mod_name, fromlist=[cls_name])
            surface[key] = members(getattr(mod, cls_name))
        except (ImportError, AttributeError):
            pass  # absent receiver -> checker reports it as unmapped

    json.dump(surface, sys.stdout, indent=1, sort_keys=True)


if __name__ == "__main__":
    main()
