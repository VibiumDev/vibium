"""Async Browser class and launcher."""

from __future__ import annotations

import json
import os
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

    def __repr__(self) -> str:
        return "Browser(connected=True)"

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

    async def stop(self) -> None:
        """Stop the browser and clean up."""
        try:
            await self._client.send("vibium:browser.stop", {})
        except Exception:
            pass  # Browser or connection may already be closed
        await self._client.close()
        if self._process:
            await self._process.stop()


class _BrowserLauncher:
    """Module-level browser launcher object."""

    async def start(
        self,
        url: Optional[str] = None,
        *,
        engine: Optional[str] = None,
        channel: Optional[str] = None,
        headless: bool = False,
        headers: Optional[Dict[str, str]] = None,
        caps: Optional[Dict[str, Any]] = None,
        executable_path: Optional[str] = None,
    ) -> Browser:
        """Start a browser session.

        Args:
            url: Remote BiDi WebSocket URL, or an http(s) classic WebDriver
                endpoint (Selenium Grid, cloud grid). If not provided, checks
                VIBIUM_CONNECT_URL env var, then falls back to local launch.
            engine: Browser engine to launch: "chrome" (default) or "firefox"
                (local launch only).
            channel: Release channel of the engine to install and run, e.g.
                "beta". Currently honored by Firefox only (local launch only).
            headless: Run browser in headless mode (local launch only).
            headers: HTTP headers for remote connection (e.g. auth tokens).
            caps: Extra alwaysMatch capabilities for classic WebDriver
                endpoints (vendor-prefixed keys like vendor:options).
            executable_path: Path to vibium binary (default: auto-detect).
        """
        from ..binary import VibiumProcess
        from ..client import BiDiClient

        connect_url = url or os.environ.get("VIBIUM_CONNECT_URL")
        if connect_url:
            env_headers: Dict[str, str] = {}
            api_key = os.environ.get("VIBIUM_CONNECT_API_KEY")
            if api_key:
                env_headers["Authorization"] = f"Bearer {api_key}"
            merged = {**env_headers, **(headers or {})}
            # Raw JSON string either way — the binary validates it and owns
            # the error message, so no parsing here.
            caps_json = json.dumps(caps) if caps else os.environ.get("VIBIUM_CONNECT_CAPS")
            process = await VibiumProcess.start(
                connect_url=connect_url,
                connect_headers=merged or None,
                connect_caps=caps_json or None,
                executable_path=executable_path,
            )
        else:
            process = await VibiumProcess.start(
                headless=headless,
                engine=engine,
                channel=channel,
                executable_path=executable_path,
            )
        client = await BiDiClient.connect(process)
        return Browser(client, process)


browser = _BrowserLauncher()


class _EngineLauncher:
    """Named engine launcher, Playwright-style: firefox.start() is
    browser.start(engine="firefox")."""

    def __init__(self, engine: str) -> None:
        self._engine = engine

    async def start(
        self,
        url: Optional[str] = None,
        *,
        channel: Optional[str] = None,
        headless: bool = False,
        headers: Optional[Dict[str, str]] = None,
        executable_path: Optional[str] = None,
    ) -> Browser:
        return await browser.start(
            url,
            engine=self._engine,
            channel=channel,
            headless=headless,
            headers=headers,
            executable_path=executable_path,
        )


firefox = _EngineLauncher("firefox")
chrome = _EngineLauncher("chrome")
