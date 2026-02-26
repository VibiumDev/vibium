"""Async Browser class and launcher."""

from __future__ import annotations

from typing import Any, Callable, Dict, List, Optional, Set, TYPE_CHECKING

from .page import Page
from .context import BrowserContext

if TYPE_CHECKING:
    from ..client import BiDiClient
    from ..binary import VibiumProcess


class Browser:
    """Async browser automation entry point."""

    def __init__(self, client: BiDiClient, process: Optional[VibiumProcess]) -> None:
        self._client = client
        self._process = process
        self._page_callbacks: List[Callable[[Page], None]] = []
        self._popup_callbacks: List[Callable[[Page], None]] = []
        self._seen_context_ids: Set[str] = set()

        # Listen for browsingContext.contextCreated events
        self._client.on_event(self._handle_event)

    def _handle_event(self, event: Dict[str, Any]) -> None:
        if event.get("method") != "browsingContext.contextCreated":
            return
        params = event.get("params", {})
        context_id = params.get("context")
        if not context_id or context_id in self._seen_context_ids:
            return
        self._seen_context_ids.add(context_id)
        callbacks = self._popup_callbacks if params.get("originalOpener") else self._page_callbacks
        if callbacks:
            page = Page(self._client, params["context"], params.get("userContext", "default"))
            for cb in callbacks:
                cb(page)

    async def page(self) -> Page:
        """Get the default page (first browsing context)."""
        result = await self._client.send("vibium:browser.page", {})
        return Page(self._client, result["context"], result.get("userContext", "default"))

    async def new_page(self) -> Page:
        """Create a new page (tab) in the default context."""
        result = await self._client.send("vibium:browser.newPage", {})
        return Page(self._client, result["context"], result.get("userContext", "default"))

    async def new_context(self) -> BrowserContext:
        """Create a new browser context (isolated, incognito-like)."""
        result = await self._client.send("vibium:browser.newContext", {})
        return BrowserContext(self._client, result["userContext"])

    async def pages(self) -> List[Page]:
        """Get all open pages."""
        result = await self._client.send("vibium:browser.pages", {})
        return [Page(self._client, p["context"], p.get("userContext", "default")) for p in result["pages"]]

    def on_page(self, callback: Callable[[Page], None]) -> None:
        """Register a callback for when a new page is created."""
        self._page_callbacks.append(callback)

    def on_popup(self, callback: Callable[[Page], None]) -> None:
        """Register a callback for when a popup is opened."""
        self._popup_callbacks.append(callback)

    def remove_all_listeners(self, event: Optional[str] = None) -> None:
        """Remove all listeners for 'page', 'popup', or all."""
        if not event or event == "page":
            self._page_callbacks.clear()
        if not event or event == "popup":
            self._popup_callbacks.clear()

    async def close(self) -> None:
        """Close the browser and clean up."""
        await self._client.send("vibium:browser.close", {})
        await self._client.close()
        if self._process:
            await self._process.stop()


class _BrowserLauncher:
    """Module-level browser launcher object."""

    async def launch(
        self,
        headless: bool = False,
        port: Optional[int] = None,
        executable_path: Optional[str] = None,
    ) -> Browser:
        """Launch a new browser instance."""
        from ..binary import VibiumProcess
        from ..client import BiDiClient

        process = await VibiumProcess.start(
            headless=headless,
            port=port,
            executable_path=executable_path,
        )
        client = await BiDiClient.connect(f"ws://localhost:{process.port}")
        return Browser(client, process)


browser = _BrowserLauncher()
