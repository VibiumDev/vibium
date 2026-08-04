"""Sync Browser wrapper and launcher."""

from __future__ import annotations

from typing import Callable, List, Optional, TYPE_CHECKING

from .page import Page
from .context import BrowserContext

if TYPE_CHECKING:
    from typing import Any
    from .._sync_base import _EventLoopThread
    from ..async_api.browser import Browser as AsyncBrowser


class Browser:
    """Synchronous wrapper for async Browser."""

    def __init__(self, async_browser: AsyncBrowser, loop_thread: _EventLoopThread) -> None:
        self._async = async_browser
        self._loop = loop_thread

    def __repr__(self) -> str:
        return "Browser(connected=True)"

    def page(self) -> Page:
        """Get the default page (first browsing context)."""
        async_page = self._loop.run(self._async.page())
        return Page(async_page, self._loop)

    def new_page(self) -> Page:
        """Create a new page (tab) in the default context."""
        async_page = self._loop.run(self._async.new_page())
        return Page(async_page, self._loop)

    def new_context(self) -> BrowserContext:
        """Create a new browser context (isolated, incognito-like)."""
        async_ctx = self._loop.run(self._async.new_context())
        return BrowserContext(async_ctx, self._loop)

    def pages(self) -> List[Page]:
        """Get all open pages."""
        async_pages = self._loop.run(self._async.pages())
        return [Page(p, self._loop) for p in async_pages]

    def on_page(self, callback: Callable[[Page], None]) -> None:
        """Register a callback for when a new page is created."""
        def _wrapper(async_page: Any) -> None:
            sync_page = Page(async_page, self._loop)
            callback(sync_page)
        self._async.on_page(_wrapper)

    def on_popup(self, callback: Callable[[Page], None]) -> None:
        """Register a callback for when a popup is opened."""
        def _wrapper(async_page: Any) -> None:
            sync_page = Page(async_page, self._loop)
            callback(sync_page)
        self._async.on_popup(_wrapper)

    def remove_all_listeners(self, event: Optional[str] = None) -> None:
        """Remove all listeners for 'page', 'popup', or all."""
        self._async.remove_all_listeners(event)

    def stop(self) -> None:
        """Stop the browser and clean up."""
        self._loop.run(self._async.stop())
        self._loop.stop()


class _BrowserLauncher:
    """Module-level sync browser launcher object."""

    def start(
        self,
        url: Optional[str] = None,
        *,
        engine: Optional[str] = None,
        headless: bool = False,
        headers: Optional[dict] = None,
        executable_path: Optional[str] = None,
    ) -> Browser:
        """Start a browser session.

        Args:
            url: Remote BiDi WebSocket URL. If not provided, checks
                VIBIUM_CONNECT_URL env var, then falls back to local launch.
            engine: Browser engine to launch: "chrome" (default) or "firefox"
                (local launch only).
            headless: Run browser in headless mode (local launch only).
            headers: HTTP headers for remote connection (e.g. auth tokens).
            executable_path: Path to vibium binary (default: auto-detect).
        """
        from .._sync_base import _EventLoopThread
        from ..async_api.browser import browser as async_browser_launcher

        loop_thread = _EventLoopThread()
        loop_thread.start()

        async_browser = loop_thread.run(
            async_browser_launcher.start(
                url,
                engine=engine,
                headless=headless,
                headers=headers,
                executable_path=executable_path,
            )
        )
        return Browser(async_browser, loop_thread)


browser = _BrowserLauncher()


class _EngineLauncher:
    """Named engine launcher, Playwright-style: firefox.start() is
    browser.start(engine="firefox")."""

    def __init__(self, engine: str) -> None:
        self._engine = engine

    def start(
        self,
        url: Optional[str] = None,
        *,
        headless: bool = False,
        headers: Optional[dict] = None,
        executable_path: Optional[str] = None,
    ) -> Browser:
        return browser.start(
            url,
            engine=self._engine,
            headless=headless,
            headers=headers,
            executable_path=executable_path,
        )


firefox = _EngineLauncher("firefox")
chrome = _EngineLauncher("chrome")
