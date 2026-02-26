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

    def close(self) -> None:
        """Close the browser and clean up."""
        self._loop.run(self._async.close())
        self._loop.stop()


class _BrowserLauncher:
    """Module-level sync browser launcher object."""

    def launch(
        self,
        headless: bool = False,
        port: Optional[int] = None,
        executable_path: Optional[str] = None,
    ) -> Browser:
        """Launch a new browser instance."""
        from .._sync_base import _EventLoopThread
        from ..async_api.browser import browser as async_browser_launcher

        loop_thread = _EventLoopThread()
        loop_thread.start()

        async_browser = loop_thread.run(
            async_browser_launcher.launch(
                headless=headless,
                port=port,
                executable_path=executable_path,
            )
        )
        return Browser(async_browser, loop_thread)


browser = _BrowserLauncher()
