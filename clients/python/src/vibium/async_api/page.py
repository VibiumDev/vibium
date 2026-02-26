"""Async Page class — the main browser automation interface."""

from __future__ import annotations

import asyncio
import base64
import fnmatch
import re
from typing import Any, Callable, Dict, List, Optional, Union, TYPE_CHECKING

from .._types import A11yNode, BoundingBox, ElementInfo
from .element import Element
from .element_list import ElementList
from .clock import Clock
from .route import Route
from .network import Request, Response
from .dialog import Dialog
from .console import ConsoleMessage
from .download import Download
from .websocket_info import WebSocketInfo

if TYPE_CHECKING:
    from ..client import BiDiClient
    from .context import BrowserContext as BrowserContextType


def _match_pattern(pattern: str, url: str) -> bool:
    """Match a URL against a glob-like pattern."""
    if pattern == "**":
        return True
    if "*" in pattern:
        return fnmatch.fnmatch(url, pattern)
    return pattern in url


class Keyboard:
    """Page-level keyboard input."""

    def __init__(self, client: BiDiClient, context_id: str) -> None:
        self._client = client
        self._context_id = context_id

    async def press(self, key: str) -> None:
        await self._client.send("vibium:keyboard.press", {"context": self._context_id, "key": key})

    async def down(self, key: str) -> None:
        await self._client.send("vibium:keyboard.down", {"context": self._context_id, "key": key})

    async def up(self, key: str) -> None:
        await self._client.send("vibium:keyboard.up", {"context": self._context_id, "key": key})

    async def type(self, text: str) -> None:
        await self._client.send("vibium:keyboard.type", {"context": self._context_id, "text": text})


class Mouse:
    """Page-level mouse input."""

    def __init__(self, client: BiDiClient, context_id: str) -> None:
        self._client = client
        self._context_id = context_id

    async def click(self, x: float, y: float) -> None:
        await self._client.send("vibium:mouse.click", {"context": self._context_id, "x": x, "y": y})

    async def move(self, x: float, y: float) -> None:
        await self._client.send("vibium:mouse.move", {"context": self._context_id, "x": x, "y": y})

    async def down(self) -> None:
        await self._client.send("vibium:mouse.down", {"context": self._context_id})

    async def up(self) -> None:
        await self._client.send("vibium:mouse.up", {"context": self._context_id})

    async def wheel(self, delta_x: float, delta_y: float) -> None:
        await self._client.send("vibium:mouse.wheel", {
            "context": self._context_id, "x": 0, "y": 0,
            "deltaX": delta_x, "deltaY": delta_y,
        })


class Touch:
    """Page-level touch input."""

    def __init__(self, client: BiDiClient, context_id: str) -> None:
        self._client = client
        self._context_id = context_id

    async def tap(self, x: float, y: float) -> None:
        await self._client.send("vibium:touch.tap", {"context": self._context_id, "x": x, "y": y})


class Page:
    """Async page automation interface."""

    def __init__(self, client: BiDiClient, context_id: str, user_context_id: str = "default") -> None:
        self._client = client
        self._context_id = context_id
        self._user_context_id = user_context_id
        self._context: Optional[BrowserContextType] = None

        self.keyboard = Keyboard(client, context_id)
        self.mouse = Mouse(client, context_id)
        self.touch = Touch(client, context_id)
        self.clock = Clock(client, context_id)

        # Event state
        self._routes: List[Dict[str, Any]] = []
        self._request_callbacks: List[Callable] = []
        self._response_callbacks: List[Callable] = []
        self._dialog_callbacks: List[Callable] = []
        self._console_callbacks: List[Callable] = []
        self._error_callbacks: List[Callable] = []
        self._download_callbacks: List[Callable] = []
        self._navigation_callbacks: List[Callable] = []
        self._pending_downloads: Dict[str, Download] = {}
        self._ws_callbacks: List[Callable] = []
        self._ws_connections: Dict[int, WebSocketInfo] = {}
        self._intercept_id: Optional[str] = None
        self._data_collector_id: Optional[str] = None

        # Console/error collect-mode buffers (None = not collecting)
        self._console_buffer: Optional[List[Dict[str, str]]] = None
        self._error_buffer: Optional[List[Dict[str, str]]] = None

        # Register event handler
        self._event_handler = self._handle_event
        self._client.on_event(self._event_handler)

    @property
    def id(self) -> str:
        return self._context_id

    @property
    def context(self) -> BrowserContextType:
        """The parent BrowserContext that owns this page."""
        if self._context is None:
            from .context import BrowserContext
            self._context = BrowserContext(self._client, self._user_context_id)
        return self._context

    # --- Navigation ---

    async def go(self, url: str) -> None:
        """Navigate to a URL."""
        await self._client.send("vibium:page.navigate", {"context": self._context_id, "url": url})

    async def back(self) -> None:
        await self._client.send("vibium:page.back", {"context": self._context_id})

    async def forward(self) -> None:
        await self._client.send("vibium:page.forward", {"context": self._context_id})

    async def reload(self) -> None:
        await self._client.send("vibium:page.reload", {"context": self._context_id})

    # --- Info ---

    async def url(self) -> str:
        result = await self._client.send("vibium:page.url", {"context": self._context_id})
        return result["url"]

    async def title(self) -> str:
        result = await self._client.send("vibium:page.title", {"context": self._context_id})
        return result["title"]

    async def content(self) -> str:
        result = await self._client.send("vibium:page.content", {"context": self._context_id})
        return result["content"]

    # --- Finding ---

    async def find(
        self,
        selector: Optional[str] = None,
        /,
        *,
        role: Optional[str] = None,
        text: Optional[str] = None,
        label: Optional[str] = None,
        placeholder: Optional[str] = None,
        alt: Optional[str] = None,
        title: Optional[str] = None,
        testid: Optional[str] = None,
        xpath: Optional[str] = None,
        near: Optional[str] = None,
        timeout: Optional[int] = None,
    ) -> Element:
        """Find an element by CSS selector or semantic options."""
        params: Dict[str, Any] = {"context": self._context_id, "timeout": timeout}
        if selector is not None:
            params["selector"] = selector
        else:
            for key, val in [("role", role), ("text", text), ("label", label),
                             ("placeholder", placeholder), ("alt", alt), ("title", title),
                             ("testid", testid), ("xpath", xpath), ("near", near)]:
                if val is not None:
                    params[key] = val

        result = await self._client.send("vibium:find", params)
        info = ElementInfo(tag=result["tag"], text=result["text"], box=BoundingBox(**result["box"]))
        sel_str = selector or ""
        sel_params = {"selector": selector} if selector else {
            k: v for k, v in params.items() if k not in ("context", "timeout")
        }
        return Element(self._client, self._context_id, sel_str, info, None, sel_params)

    async def find_all(
        self,
        selector: Optional[str] = None,
        /,
        *,
        role: Optional[str] = None,
        text: Optional[str] = None,
        label: Optional[str] = None,
        placeholder: Optional[str] = None,
        alt: Optional[str] = None,
        title: Optional[str] = None,
        testid: Optional[str] = None,
        xpath: Optional[str] = None,
        near: Optional[str] = None,
        timeout: Optional[int] = None,
    ) -> ElementList:
        """Find all elements matching a selector or semantic options."""
        params: Dict[str, Any] = {"context": self._context_id, "timeout": timeout}
        if selector is not None:
            params["selector"] = selector
        else:
            for key, val in [("role", role), ("text", text), ("label", label),
                             ("placeholder", placeholder), ("alt", alt), ("title", title),
                             ("testid", testid), ("xpath", xpath), ("near", near)]:
                if val is not None:
                    params[key] = val

        result = await self._client.send("vibium:findAll", params)
        sel_str = selector or ""
        sel_params = {"selector": selector} if selector else {
            k: v for k, v in params.items() if k not in ("context", "timeout")
        }
        elements = []
        for el in result["elements"]:
            info = ElementInfo(tag=el["tag"], text=el["text"], box=BoundingBox(**el["box"]))
            elements.append(Element(self._client, self._context_id, sel_str, info, el.get("index"), sel_params))
        return ElementList(self._client, self._context_id, selector or params, elements)

    # --- Waiting ---

    @property
    def capture(self) -> _CaptureNamespace:
        """Capture namespace — set up a listener before performing an action."""
        return _CaptureNamespace(self)

    @property
    def wait_until(self) -> _WaitUntilNamespace:
        """Wait until a condition is met. Callable or use .url() / .loaded() sub-methods."""
        return _WaitUntilNamespace(self)

    async def wait(self, ms: int) -> None:
        """Wait for a fixed amount of time (milliseconds)."""
        await self._client.send("vibium:page.wait", {"context": self._context_id, "ms": ms})

    async def _wait_for_url(self, pattern: str, timeout: Optional[int] = None) -> None:
        """Internal: wait until the page URL matches a pattern."""
        await self._client.send("vibium:page.waitForURL", {
            "context": self._context_id, "pattern": pattern, "timeout": timeout,
        })

    async def _wait_for_load(self, state: Optional[str] = None, timeout: Optional[int] = None) -> None:
        """Internal: wait until the page reaches a load state."""
        await self._client.send("vibium:page.waitForLoad", {
            "context": self._context_id, "state": state, "timeout": timeout,
        })

    async def _wait_for_function(self, fn: str, timeout: Optional[int] = None) -> Any:
        """Internal: wait until a function returns a truthy value."""
        result = await self._client.send("vibium:page.waitForFunction", {
            "context": self._context_id, "fn": fn, "timeout": timeout,
        })
        return result["value"]

    async def _capture_response(self, pattern: str, timeout: Optional[int] = None) -> Response:
        """Internal: wait for a response matching a URL pattern."""
        timeout_ms = timeout or 10000
        future: asyncio.Future = asyncio.get_running_loop().create_future()

        def handler(response: Response) -> None:
            if _match_pattern(pattern, response.url()):
                self._response_callbacks.remove(handler)
                if not future.done():
                    future.set_result(response)

        self._ensure_data_collector()
        self._response_callbacks.append(handler)

        try:
            return await asyncio.wait_for(future, timeout=timeout_ms / 1000)
        except asyncio.TimeoutError:
            if handler in self._response_callbacks:
                self._response_callbacks.remove(handler)
            raise TimeoutError(f"Timeout waiting for response matching '{pattern}'")

    async def _setup_capture_response(self, pattern: str, timeout: Optional[int] = None) -> Any:
        """Internal: set up response listener and return a coroutine to await later."""
        timeout_ms = timeout or 10000
        future: asyncio.Future = asyncio.get_running_loop().create_future()

        def handler(response: Response) -> None:
            if _match_pattern(pattern, response.url()):
                self._response_callbacks.remove(handler)
                if not future.done():
                    future.set_result(response)

        self._ensure_data_collector()
        self._response_callbacks.append(handler)

        async def _wait() -> Response:
            try:
                return await asyncio.wait_for(future, timeout=timeout_ms / 1000)
            except asyncio.TimeoutError:
                if handler in self._response_callbacks:
                    self._response_callbacks.remove(handler)
                raise TimeoutError(f"Timeout waiting for response matching '{pattern}'")

        return _wait()

    async def _capture_request(self, pattern: str, timeout: Optional[int] = None) -> Request:
        """Internal: wait for a request matching a URL pattern."""
        timeout_ms = timeout or 10000
        future: asyncio.Future = asyncio.get_running_loop().create_future()

        def handler(request: Request) -> None:
            if _match_pattern(pattern, request.url()):
                self._request_callbacks.remove(handler)
                if not future.done():
                    future.set_result(request)

        self._ensure_data_collector()
        self._request_callbacks.append(handler)

        try:
            return await asyncio.wait_for(future, timeout=timeout_ms / 1000)
        except asyncio.TimeoutError:
            if handler in self._request_callbacks:
                self._request_callbacks.remove(handler)
            raise TimeoutError(f"Timeout waiting for request matching '{pattern}'")

    async def _setup_capture_request(self, pattern: str, timeout: Optional[int] = None) -> Any:
        """Internal: set up request listener and return a coroutine to await later."""
        timeout_ms = timeout or 10000
        future: asyncio.Future = asyncio.get_running_loop().create_future()

        def handler(request: Request) -> None:
            if _match_pattern(pattern, request.url()):
                self._request_callbacks.remove(handler)
                if not future.done():
                    future.set_result(request)

        self._ensure_data_collector()
        self._request_callbacks.append(handler)

        async def _wait() -> Request:
            try:
                return await asyncio.wait_for(future, timeout=timeout_ms / 1000)
            except asyncio.TimeoutError:
                if handler in self._request_callbacks:
                    self._request_callbacks.remove(handler)
                raise TimeoutError(f"Timeout waiting for request matching '{pattern}'")

        return _wait()

    async def _capture_navigation(self, timeout: Optional[int] = None) -> str:
        """Internal: wait for a navigation event. Resolves with URL."""
        timeout_ms = timeout or 10000
        future: asyncio.Future = asyncio.get_running_loop().create_future()

        def handler(url: str) -> None:
            self._navigation_callbacks.remove(handler)
            if not future.done():
                future.set_result(url)

        self._navigation_callbacks.append(handler)

        try:
            return await asyncio.wait_for(future, timeout=timeout_ms / 1000)
        except asyncio.TimeoutError:
            if handler in self._navigation_callbacks:
                self._navigation_callbacks.remove(handler)
            raise TimeoutError("Timeout waiting for navigation")

    async def _setup_capture_navigation(self, timeout: Optional[int] = None) -> Any:
        """Internal: set up navigation listener and return a coroutine to await later."""
        timeout_ms = timeout or 10000
        future: asyncio.Future = asyncio.get_running_loop().create_future()

        def handler(url: str) -> None:
            self._navigation_callbacks.remove(handler)
            if not future.done():
                future.set_result(url)

        self._navigation_callbacks.append(handler)

        async def _wait() -> str:
            try:
                return await asyncio.wait_for(future, timeout=timeout_ms / 1000)
            except asyncio.TimeoutError:
                if handler in self._navigation_callbacks:
                    self._navigation_callbacks.remove(handler)
                raise TimeoutError("Timeout waiting for navigation")

        return _wait()

    async def _capture_download(self, timeout: Optional[int] = None) -> Download:
        """Internal: wait for a download event."""
        timeout_ms = timeout or 10000
        future: asyncio.Future = asyncio.get_running_loop().create_future()

        def handler(download: Download) -> None:
            self._download_callbacks.remove(handler)
            if not future.done():
                future.set_result(download)

        self._download_callbacks.append(handler)

        try:
            return await asyncio.wait_for(future, timeout=timeout_ms / 1000)
        except asyncio.TimeoutError:
            if handler in self._download_callbacks:
                self._download_callbacks.remove(handler)
            raise TimeoutError("Timeout waiting for download")

    async def _setup_capture_download(self, timeout: Optional[int] = None) -> Any:
        """Internal: set up download listener and return a coroutine to await later."""
        timeout_ms = timeout or 10000
        future: asyncio.Future = asyncio.get_running_loop().create_future()

        def handler(download: Download) -> None:
            self._download_callbacks.remove(handler)
            if not future.done():
                future.set_result(download)

        self._download_callbacks.append(handler)

        async def _wait() -> Download:
            try:
                return await asyncio.wait_for(future, timeout=timeout_ms / 1000)
            except asyncio.TimeoutError:
                if handler in self._download_callbacks:
                    self._download_callbacks.remove(handler)
                raise TimeoutError("Timeout waiting for download")

        return _wait()

    async def _capture_dialog(self, timeout: Optional[int] = None) -> Dialog:
        """Internal: wait for a dialog event. Callback presence prevents auto-dismiss."""
        timeout_ms = timeout or 10000
        future: asyncio.Future = asyncio.get_running_loop().create_future()

        def handler(dialog: Dialog) -> None:
            self._dialog_callbacks.remove(handler)
            if not future.done():
                future.set_result(dialog)

        self._dialog_callbacks.append(handler)

        try:
            return await asyncio.wait_for(future, timeout=timeout_ms / 1000)
        except asyncio.TimeoutError:
            if handler in self._dialog_callbacks:
                self._dialog_callbacks.remove(handler)
            raise TimeoutError("Timeout waiting for dialog")

    async def _setup_capture_dialog(self, timeout: Optional[int] = None) -> Any:
        """Internal: set up dialog listener and return a coroutine to await later."""
        timeout_ms = timeout or 10000
        future: asyncio.Future = asyncio.get_running_loop().create_future()

        def handler(dialog: Dialog) -> None:
            self._dialog_callbacks.remove(handler)
            if not future.done():
                future.set_result(dialog)

        self._dialog_callbacks.append(handler)

        async def _wait() -> Dialog:
            try:
                return await asyncio.wait_for(future, timeout=timeout_ms / 1000)
            except asyncio.TimeoutError:
                if handler in self._dialog_callbacks:
                    self._dialog_callbacks.remove(handler)
                raise TimeoutError("Timeout waiting for dialog")

        return _wait()

    async def _capture_event(self, name: str, timeout: Optional[int] = None) -> Any:
        """Internal: wait for a named event."""
        timeout_ms = timeout or 10000
        future: asyncio.Future = asyncio.get_running_loop().create_future()

        callback_list = self._get_callback_list(name)
        if callback_list is None:
            raise ValueError(f"Unknown event name: '{name}'")

        def handler(data: Any) -> None:
            callback_list.remove(handler)
            if not future.done():
                future.set_result(data)

        if name in ("request", "response"):
            self._ensure_data_collector()
        callback_list.append(handler)

        try:
            return await asyncio.wait_for(future, timeout=timeout_ms / 1000)
        except asyncio.TimeoutError:
            if handler in callback_list:
                callback_list.remove(handler)
            raise TimeoutError(f"Timeout waiting for event '{name}'")

    async def _setup_capture_event(self, name: str, timeout: Optional[int] = None) -> Any:
        """Internal: set up event listener and return a coroutine to await later."""
        timeout_ms = timeout or 10000
        future: asyncio.Future = asyncio.get_running_loop().create_future()

        callback_list = self._get_callback_list(name)
        if callback_list is None:
            raise ValueError(f"Unknown event name: '{name}'")

        def handler(data: Any) -> None:
            callback_list.remove(handler)
            if not future.done():
                future.set_result(data)

        if name in ("request", "response"):
            self._ensure_data_collector()
        callback_list.append(handler)

        async def _wait() -> Any:
            try:
                return await asyncio.wait_for(future, timeout=timeout_ms / 1000)
            except asyncio.TimeoutError:
                if handler in callback_list:
                    callback_list.remove(handler)
                raise TimeoutError(f"Timeout waiting for event '{name}'")

        return _wait()

    def _get_callback_list(self, name: str) -> Optional[List[Callable]]:
        """Map event name to callback list."""
        mapping = {
            "request": self._request_callbacks,
            "response": self._response_callbacks,
            "dialog": self._dialog_callbacks,
            "download": self._download_callbacks,
            "navigation": self._navigation_callbacks,
            "console": self._console_callbacks,
            "error": self._error_callbacks,
        }
        return mapping.get(name)

    # --- Screenshots & PDF ---

    async def screenshot(
        self,
        full_page: Optional[bool] = None,
        clip: Optional[Dict[str, Any]] = None,
    ) -> bytes:
        """Take a screenshot. Returns PNG bytes."""
        result = await self._client.send("vibium:page.screenshot", {
            "context": self._context_id,
            "fullPage": full_page,
            "clip": clip,
        })
        return base64.b64decode(result["data"])

    async def pdf(self) -> bytes:
        """Print the page to PDF. Returns PDF bytes. Only works in headless mode."""
        result = await self._client.send("vibium:page.pdf", {"context": self._context_id})
        return base64.b64decode(result["data"])

    # --- Evaluation ---

    async def eval(self, expression: str) -> Any:
        """Evaluate a JS expression and return the deserialized value."""
        result = await self._client.send("vibium:page.eval", {
            "context": self._context_id, "expression": expression,
        })
        return result["value"]

    async def evaluate(self, script: str) -> Any:
        """Execute a JS script (multi-statement, use 'return' for values)."""
        result = await self._client.send("script.callFunction", {
            "functionDeclaration": f"() => {{ {script} }}",
            "target": {"context": self._context_id},
            "arguments": [],
            "awaitPromise": True,
            "resultOwnership": "root",
        })
        return result.get("result", {}).get("value")

    async def eval_handle(self, expression: str) -> str:
        """Evaluate a JS expression and return a handle ID."""
        result = await self._client.send("vibium:page.evalHandle", {
            "context": self._context_id, "expression": expression,
        })
        return result["handle"]

    async def add_script(self, source: str) -> None:
        """Inject a script into the page. Pass a URL or inline JavaScript."""
        is_url = source.startswith("http://") or source.startswith("https://") or source.startswith("//")
        params: Dict[str, Any] = {"context": self._context_id}
        if is_url:
            params["url"] = source
        else:
            params["content"] = source
        await self._client.send("vibium:page.addScript", params)

    async def add_style(self, source: str) -> None:
        """Inject a stylesheet into the page. Pass a URL or inline CSS."""
        is_url = source.startswith("http://") or source.startswith("https://") or source.startswith("//")
        params: Dict[str, Any] = {"context": self._context_id}
        if is_url:
            params["url"] = source
        else:
            params["content"] = source
        await self._client.send("vibium:page.addStyle", params)

    async def expose(self, name: str, fn: str) -> None:
        """Expose a function on window."""
        await self._client.send("vibium:page.expose", {
            "context": self._context_id, "name": name, "fn": fn,
        })

    # --- Emulation ---

    async def set_viewport(self, size: Dict[str, int]) -> None:
        """Set the viewport size. size: {width, height}."""
        await self._client.send("vibium:page.setViewport", {
            "context": self._context_id, "width": size["width"], "height": size["height"],
        })

    async def viewport(self) -> Dict[str, int]:
        """Get the current viewport size."""
        return await self._client.send("vibium:page.viewport", {"context": self._context_id})

    async def emulate_media(self, **opts: Any) -> None:
        """Override CSS media features."""
        await self._client.send("vibium:page.emulateMedia", {
            "context": self._context_id, **opts,
        })

    async def set_content(self, html: str) -> None:
        """Replace the page HTML content."""
        await self._client.send("vibium:page.setContent", {"context": self._context_id, "html": html})

    async def set_geolocation(self, coords: Dict[str, float]) -> None:
        """Override the browser's geolocation."""
        await self._client.send("vibium:page.setGeolocation", {
            "context": self._context_id, **coords,
        })

    async def set_window(self, **options: Any) -> None:
        """Set the OS browser window size, position, or state."""
        await self._client.send("vibium:page.setWindow", options)

    async def window(self) -> Dict[str, Any]:
        """Get the current OS browser window state and dimensions."""
        return await self._client.send("vibium:page.window", {})

    # --- Accessibility ---

    async def a11y_tree(
        self,
        everything: Optional[bool] = None,
        root: Optional[str] = None,
    ) -> A11yNode:
        """Get the accessibility tree for the page."""
        params: Dict[str, Any] = {"context": self._context_id}
        if everything is not None:
            params["everything"] = everything
        if root is not None:
            params["root"] = root
        result = await self._client.send("vibium:page.a11yTree", params)
        return result["tree"]

    # --- Frames ---

    async def frames(self) -> List[Page]:
        """Get all child frames of this page."""
        result = await self._client.send("vibium:page.frames", {"context": self._context_id})
        return [Page(self._client, f["context"]) for f in result["frames"]]

    async def frame(self, name_or_url: str) -> Optional[Page]:
        """Find a frame by name or URL substring."""
        result = await self._client.send("vibium:page.frame", {
            "context": self._context_id, "nameOrUrl": name_or_url,
        })
        if not result or not result.get("context"):
            return None
        return Page(self._client, result["context"])

    def main_frame(self) -> Page:
        """Returns this page — the page IS its own main frame."""
        return self

    # --- Lifecycle ---

    async def bring_to_front(self) -> None:
        await self._client.send("browsingContext.activate", {"context": self._context_id})

    async def close(self) -> None:
        await self._client.send("browsingContext.close", {"context": self._context_id})

    # --- Network Interception ---

    async def route(self, pattern: str, handler: Callable[[Route], Any]) -> None:
        """Intercept network requests matching a URL pattern."""
        if self._intercept_id is None:
            result = await self._client.send("vibium:page.route", {"context": self._context_id})
            self._intercept_id = result["intercept"]

        self._ensure_data_collector()
        self._routes.append({"pattern": pattern, "handler": handler, "interceptId": self._intercept_id})

    async def unroute(self, pattern: str) -> None:
        """Remove a previously registered route."""
        self._routes = [r for r in self._routes if r["pattern"] != pattern]
        if not self._routes and self._intercept_id:
            await self._client.send("network.removeIntercept", {"intercept": self._intercept_id})
            self._intercept_id = None

    def on_request(self, fn: Callable[[Request], None]) -> None:
        """Register a callback for every outgoing request."""
        self._ensure_data_collector()
        self._request_callbacks.append(fn)

    def on_response(self, fn: Callable[[Response], None]) -> None:
        """Register a callback for every completed response."""
        self._ensure_data_collector()
        self._response_callbacks.append(fn)

    async def set_headers(self, headers: Dict[str, str]) -> None:
        """Set extra HTTP headers for all requests in this page."""
        result = await self._client.send("vibium:page.setHeaders", {
            "context": self._context_id, "headers": headers,
        })

        def _header_handler(route: Route) -> None:
            merged = {**route.request.headers(), **headers}
            import asyncio
            asyncio.ensure_future(route.continue_(headers=merged))

        self._routes.append({
            "pattern": "**",
            "handler": _header_handler,
            "interceptId": result["intercept"],
        })

    def on_web_socket(self, fn: Callable[[WebSocketInfo], None]) -> None:
        """Listen for WebSocket connections opened by the page."""
        is_first = len(self._ws_callbacks) == 0
        self._ws_callbacks.append(fn)
        if is_first:
            import asyncio
            asyncio.ensure_future(
                self._client.send("vibium:page.onWebSocket", {"context": self._context_id})
            )


    # --- Dialog Handling ---

    def on_dialog(self, handler: Callable[[Dialog], Any]) -> None:
        self._dialog_callbacks.append(handler)

    def on_console(self, handler: Union[Callable[[ConsoleMessage], None], str]) -> None:
        """Register a handler for console messages, or pass 'collect' to buffer them."""
        if handler == "collect":
            if self._console_buffer is None:
                self._console_buffer = []

                def _collector(msg: ConsoleMessage) -> None:
                    if self._console_buffer is not None:
                        self._console_buffer.append({"type": msg.type(), "text": msg.text()})

                self._console_callbacks.append(_collector)
        else:
            self._console_callbacks.append(handler)

    def console_messages(self) -> List[Dict[str, str]]:
        """Return collected console messages and clear the buffer. Returns [] if not collecting."""
        msgs = list(self._console_buffer) if self._console_buffer is not None else []
        if self._console_buffer is not None:
            self._console_buffer.clear()
        return msgs

    def on_error(self, handler: Union[Callable[[Exception], None], str]) -> None:
        """Register a handler for uncaught page errors, or pass 'collect' to buffer them."""
        if handler == "collect":
            if self._error_buffer is None:
                self._error_buffer = []

                def _collector(error: Exception) -> None:
                    if self._error_buffer is not None:
                        self._error_buffer.append({"message": str(error)})

                self._error_callbacks.append(_collector)
        else:
            self._error_callbacks.append(handler)

    def errors(self) -> List[Dict[str, str]]:
        """Return collected errors and clear the buffer. Returns [] if not collecting."""
        errs = list(self._error_buffer) if self._error_buffer is not None else []
        if self._error_buffer is not None:
            self._error_buffer.clear()
        return errs

    def on_download(self, handler: Callable[[Download], None]) -> None:
        self._download_callbacks.append(handler)

    def remove_all_listeners(self, event: Optional[str] = None) -> None:
        """Remove all listeners for a given event, or all events."""
        if not event or event == "request":
            self._request_callbacks.clear()
        if not event or event == "response":
            self._response_callbacks.clear()
        if not event or event == "dialog":
            self._dialog_callbacks.clear()
        if not event or event == "console":
            self._console_callbacks.clear()
            self._console_buffer = None
        if not event or event == "error":
            self._error_callbacks.clear()
            self._error_buffer = None
        if not event or event == "download":
            self._download_callbacks.clear()
        if not event or event == "navigation":
            self._navigation_callbacks.clear()
        if not event or event == "websocket":
            self._ws_callbacks.clear()
        if (not self._request_callbacks and not self._response_callbacks and not self._routes):
            self._teardown_data_collector()

    # --- Internal Event Handling ---

    def _ensure_data_collector(self) -> None:
        if self._data_collector_id is not None:
            return
        self._data_collector_id = "pending"
        import asyncio

        async def _setup() -> None:
            try:
                result = await self._client.send(
                    "network.addDataCollector",
                    {"dataTypes": ["request", "response"], "maxEncodedDataSize": 10 * 1024 * 1024},
                )
                self._data_collector_id = result["collector"]
            except Exception:
                self._data_collector_id = None

        asyncio.ensure_future(_setup())

    def _teardown_data_collector(self) -> None:
        cid = self._data_collector_id
        if not cid or cid == "pending":
            self._data_collector_id = None
            return
        self._data_collector_id = None
        import asyncio
        asyncio.ensure_future(
            self._client.send("network.removeDataCollector", {"collector": cid})
        )

    def _handle_event(self, event: Dict[str, Any]) -> None:
        """Dispatch a BiDi event to the appropriate handler."""
        params = event.get("params", {})
        event_context = params.get("context")

        # Filter events to this page's context
        if event_context and event_context != self._context_id:
            # log.entryAdded uses source.context
            method = event.get("method", "")
            if method == "log.entryAdded":
                source = params.get("source", {})
                if source.get("context") != self._context_id:
                    return
            else:
                return

        method = event.get("method", "")

        if method == "network.beforeRequestSent":
            self._handle_before_request_sent(params)
        elif method == "network.responseCompleted":
            self._handle_response_completed(params)
        elif method == "browsingContext.userPromptOpened":
            self._handle_user_prompt_opened(params)
        elif method == "browsingContext.downloadWillBegin":
            self._handle_download_will_begin(params)
        elif method == "browsingContext.downloadEnd":
            self._handle_download_completed(params)
        elif method == "log.entryAdded":
            self._handle_log_entry_added(params)
        elif method == "browsingContext.load":
            url = params.get("url", "")
            if url:
                for cb in self._navigation_callbacks:
                    cb(url)
        elif method == "browsingContext.fragmentNavigated":
            url = params.get("url", "")
            if url:
                for cb in self._navigation_callbacks:
                    cb(url)
        elif method == "vibium:ws.created":
            self._handle_ws_created(params)
        elif method == "vibium:ws.message":
            self._handle_ws_message(params)
        elif method == "vibium:ws.closed":
            self._handle_ws_closed(params)

    def _handle_before_request_sent(self, params: Dict[str, Any]) -> None:
        is_blocked = params.get("isBlocked", False)
        request_data = params.get("request", {})
        request_id = request_data.get("request", "")

        if is_blocked and request_id:
            request_url = request_data.get("url", "")
            req = Request(params, self._client)

            for route_entry in self._routes:
                if _match_pattern(route_entry["pattern"], request_url):
                    route = Route(self._client, request_id, req)
                    try:
                        result = route_entry["handler"](route)
                        if hasattr(result, "__await__"):
                            import asyncio
                            asyncio.ensure_future(result)
                    except Exception:
                        pass
                    return

            # No matching route — auto-continue
            import asyncio
            asyncio.ensure_future(
                self._client.send("network.continueRequest", {"request": request_id})
            )
        else:
            req = Request(params, self._client)
            for cb in self._request_callbacks:
                cb(req)

    def _handle_response_completed(self, params: Dict[str, Any]) -> None:
        resp = Response(params, self._client)
        for cb in self._response_callbacks:
            cb(resp)

    def _handle_user_prompt_opened(self, params: Dict[str, Any]) -> None:
        dialog = Dialog(self._client, self._context_id, params)

        if self._dialog_callbacks:
            for cb in self._dialog_callbacks:
                try:
                    result = cb(dialog)
                    if hasattr(result, "__await__"):
                        import asyncio
                        asyncio.ensure_future(result)
                except Exception:
                    pass
        else:
            # Auto-dismiss if no handler registered
            import asyncio
            asyncio.ensure_future(dialog.dismiss())

    def _handle_log_entry_added(self, params: Dict[str, Any]) -> None:
        entry_type = params.get("type", "")
        if entry_type == "console":
            msg = ConsoleMessage(params)
            for cb in self._console_callbacks:
                cb(msg)
        elif entry_type == "javascript":
            text = params.get("text", "Unknown error")
            error = RuntimeError(text)
            for cb in self._error_callbacks:
                cb(error)

    def _handle_download_will_begin(self, params: Dict[str, Any]) -> None:
        url = params.get("url", "")
        filename = params.get("suggestedFilename", "")
        navigation = params.get("navigation", "")

        download = Download(self._client, url, filename)
        if navigation:
            self._pending_downloads[navigation] = download

        for cb in self._download_callbacks:
            cb(download)

    def _handle_download_completed(self, params: Dict[str, Any]) -> None:
        navigation = params.get("navigation", "")
        status = params.get("status", "complete")
        filepath = params.get("filepath")

        download = self._pending_downloads.pop(navigation, None)
        if download:
            download._complete(status, filepath)

    def _handle_ws_created(self, params: Dict[str, Any]) -> None:
        ws_id = params.get("id", 0)
        url = params.get("url", "")
        ws = WebSocketInfo(url)
        self._ws_connections[ws_id] = ws
        for cb in self._ws_callbacks:
            cb(ws)

    def _handle_ws_message(self, params: Dict[str, Any]) -> None:
        ws_id = params.get("id", 0)
        data = params.get("data", "")
        direction = params.get("direction", "received")
        ws = self._ws_connections.get(ws_id)
        if ws:
            ws._emit_message(data, direction)

    def _handle_ws_closed(self, params: Dict[str, Any]) -> None:
        ws_id = params.get("id", 0)
        code = params.get("code")
        reason = params.get("reason")
        ws = self._ws_connections.pop(ws_id, None)
        if ws:
            ws._emit_close(code, reason)


class _CapturedResponse:
    """Returned by capture.response(). Awaitable and usable as async context manager."""

    def __init__(self, page: Page, pattern: str, timeout: Optional[int] = None) -> None:
        self._page = page
        self._pattern = pattern
        self._timeout = timeout
        self._wait_coro: Any = None
        self.value: Optional[Response] = None

    def __await__(self):
        return self._page._capture_response(self._pattern, self._timeout).__await__()

    async def __aenter__(self):
        self._wait_coro = await self._page._setup_capture_response(self._pattern, self._timeout)
        return self

    async def __aexit__(self, exc_type: Any, exc_val: Any, exc_tb: Any) -> None:
        if exc_type is None and self._wait_coro:
            self.value = await self._wait_coro


class _CapturedRequest:
    """Returned by capture.request(). Awaitable and usable as async context manager."""

    def __init__(self, page: Page, pattern: str, timeout: Optional[int] = None) -> None:
        self._page = page
        self._pattern = pattern
        self._timeout = timeout
        self._wait_coro: Any = None
        self.value: Optional[Request] = None

    def __await__(self):
        return self._page._capture_request(self._pattern, self._timeout).__await__()

    async def __aenter__(self):
        self._wait_coro = await self._page._setup_capture_request(self._pattern, self._timeout)
        return self

    async def __aexit__(self, exc_type: Any, exc_val: Any, exc_tb: Any) -> None:
        if exc_type is None and self._wait_coro:
            self.value = await self._wait_coro


class _CapturedNavigation:
    """Returned by capture.navigation(). Awaitable and usable as async context manager."""

    def __init__(self, page: Page, timeout: Optional[int] = None) -> None:
        self._page = page
        self._timeout = timeout
        self._wait_coro: Any = None
        self.value: Optional[str] = None

    def __await__(self):
        return self._page._capture_navigation(self._timeout).__await__()

    async def __aenter__(self):
        self._wait_coro = await self._page._setup_capture_navigation(self._timeout)
        return self

    async def __aexit__(self, exc_type: Any, exc_val: Any, exc_tb: Any) -> None:
        if exc_type is None and self._wait_coro:
            self.value = await self._wait_coro


class _CapturedDownload:
    """Returned by capture.download(). Awaitable and usable as async context manager."""

    def __init__(self, page: Page, timeout: Optional[int] = None) -> None:
        self._page = page
        self._timeout = timeout
        self._wait_coro: Any = None
        self.value: Optional[Download] = None

    def __await__(self):
        return self._page._capture_download(self._timeout).__await__()

    async def __aenter__(self):
        self._wait_coro = await self._page._setup_capture_download(self._timeout)
        return self

    async def __aexit__(self, exc_type: Any, exc_val: Any, exc_tb: Any) -> None:
        if exc_type is None and self._wait_coro:
            self.value = await self._wait_coro


class _CapturedDialog:
    """Returned by capture.dialog(). Awaitable and usable as async context manager."""

    def __init__(self, page: Page, timeout: Optional[int] = None) -> None:
        self._page = page
        self._timeout = timeout
        self._wait_coro: Any = None
        self.value: Optional[Dialog] = None

    def __await__(self):
        return self._page._capture_dialog(self._timeout).__await__()

    async def __aenter__(self):
        self._wait_coro = await self._page._setup_capture_dialog(self._timeout)
        return self

    async def __aexit__(self, exc_type: Any, exc_val: Any, exc_tb: Any) -> None:
        if exc_type is None and self._wait_coro:
            self.value = await self._wait_coro


class _CapturedEvent:
    """Returned by capture.event(). Awaitable and usable as async context manager."""

    def __init__(self, page: Page, name: str, timeout: Optional[int] = None) -> None:
        self._page = page
        self._name = name
        self._timeout = timeout
        self._wait_coro: Any = None
        self.value: Any = None

    def __await__(self):
        return self._page._capture_event(self._name, self._timeout).__await__()

    async def __aenter__(self):
        self._wait_coro = await self._page._setup_capture_event(self._name, self._timeout)
        return self

    async def __aexit__(self, exc_type: Any, exc_val: Any, exc_tb: Any) -> None:
        if exc_type is None and self._wait_coro:
            self.value = await self._wait_coro


class _CaptureNamespace:
    """Namespace for capture methods on Page."""

    def __init__(self, page: Page) -> None:
        self._page = page

    def response(self, pattern: str, timeout: Optional[int] = None) -> _CapturedResponse:
        """Wait for a response matching a URL pattern.

        Can be awaited directly or used as async context manager:
            resp = await page.capture.response("**/api")
            async with page.capture.response("**/api") as info:
                await el.click()
            resp = info.value
        """
        return _CapturedResponse(self._page, pattern, timeout)

    def request(self, pattern: str, timeout: Optional[int] = None) -> _CapturedRequest:
        """Wait for a request matching a URL pattern.

        Can be awaited directly or used as async context manager:
            req = await page.capture.request("**/api")
            async with page.capture.request("**/api") as info:
                await el.click()
            req = info.value
        """
        return _CapturedRequest(self._page, pattern, timeout)

    def navigation(self, timeout: Optional[int] = None) -> _CapturedNavigation:
        """Wait for a navigation event. Resolves with URL string."""
        return _CapturedNavigation(self._page, timeout)

    def download(self, timeout: Optional[int] = None) -> _CapturedDownload:
        """Wait for a download event."""
        return _CapturedDownload(self._page, timeout)

    def dialog(self, timeout: Optional[int] = None) -> _CapturedDialog:
        """Wait for a dialog event."""
        return _CapturedDialog(self._page, timeout)

    def event(self, name: str, timeout: Optional[int] = None) -> _CapturedEvent:
        """Wait for a named event."""
        return _CapturedEvent(self._page, name, timeout)


class _WaitUntilNamespace:
    """Namespace for waitUntil methods on Page. Also callable for waitForFunction."""

    def __init__(self, page: Page) -> None:
        self._page = page

    async def __call__(self, fn: str, timeout: Optional[int] = None) -> Any:
        """Wait until a function returns a truthy value."""
        return await self._page._wait_for_function(fn, timeout)

    async def url(self, pattern: str, timeout: Optional[int] = None) -> None:
        """Wait until the page URL matches a pattern."""
        await self._page._wait_for_url(pattern, timeout)

    async def loaded(self, state: Optional[str] = None, timeout: Optional[int] = None) -> None:
        """Wait until the page reaches a load state."""
        await self._page._wait_for_load(state, timeout)
