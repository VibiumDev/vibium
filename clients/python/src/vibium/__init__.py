"""Vibium - Browser automation for AI agents and humans.

Usage (sync, default):
    from vibium import browser
    bro = browser.start()
    vibe = bro.new_page()
    vibe.go("https://example.com")
    bro.stop()

Usage (async):
    from vibium.async_api import browser
    bro = await browser.start()
    vibe = await bro.new_page()
    await vibe.go("https://example.com")
    await bro.stop()
"""

from .sync_api.browser import browser, Browser
from .sync_api.page import Page, Keyboard, Mouse, Touch, SyncDownload as Download
from .sync_api.element import Element
from .sync_api.context import BrowserContext
from .sync_api.clock import Clock
from .sync_api.recording import Recording
from .sync_api.dialog import Dialog
from .sync_api.route import Route
from .errors import (
    VibiumError,
    BiDiError,
    VibiumNotFoundError,
    TimeoutError,
    ConnectionError,
    ElementNotFoundError,
    BrowserCrashedError,
)

__version__ = "26.5.31"
__all__ = [
    "browser",
    "Browser",
    "Page",
    "Keyboard",
    "Mouse",
    "Touch",
    "Element",
    "BrowserContext",
    "Clock",
    "Recording",
    "Dialog",
    "Route",
    "Download",
    "VibiumError",
    "BiDiError",
    "VibiumNotFoundError",
    "TimeoutError",
    "ConnectionError",
    "ElementNotFoundError",
    "BrowserCrashedError",
]
