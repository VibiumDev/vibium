"""Vibium - Browser automation for AI agents and humans.

Usage (sync, default):
    from vibium import browser
    bro = browser.launch()
    vibe = bro.new_page()
    vibe.go("https://example.com")
    bro.close()

Usage (async):
    from vibium.async_api import browser
    bro = await browser.launch()
    vibe = await bro.new_page()
    await vibe.go("https://example.com")
    await bro.close()
"""

from .sync_api.browser import browser, Browser
from .sync_api.page import Page
from .sync_api.element import Element
from .sync_api.element_list import ElementList
from .sync_api.context import BrowserContext

__version__ = "26.2.0"
__all__ = ["browser", "Browser", "Page", "Element", "ElementList", "BrowserContext"]
