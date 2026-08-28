"""Async Page class — the main browser automation interface."""

from __future__ import annotations

import asyncio
import base64
import logging
import re
import warnings
from typing import Any, Callable, Dict, List, Optional, Union, TYPE_CHECKING

from .. import errors
from .._types import A11yNode, BoundingBox, ElementInfo
from .element import Element
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

logger = logging.getLogger("vibium")


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
        self._dialog_policy_manual = False
        self._console_callbacks: List[Callable] = []
        self._error_callbacks: List[Callable] = []
        self._download_callbacks: List[Callable] = []
        self._navigation_callbacks: List[Callable] = []
        self._pending_downloads: Dict[str, Download] = {}
        self._ws_callbacks: List[Callable] = []
        self._ws_connections: Dict[int, WebSocketInfo] = {}
        self._ws_setup: Optional[asyncio.Future] = None
        self._intercept_id: Optional[str] = None
        self._data_collector_id: Optional[str] = None

        # Console/error collect-mode buffers (None = not collecting)
        self._console_buffer: Optional[List[Dict[str, str]]] = None
        self._error_buffer: Optional[List[Dict[str, str]]] = None

        # Register event handler
        self._event_handler = self._handle_event
        self._client.on_event(self._event_handler)

    def __repr__(self) -> str:
        return f"Page(context='{self._context_id}')"

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

        result = await self._client.send("vibium:page.find", params)
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
    ) -> List[Element]:
        """Find all elements matching a selector or semantic options.

        Waits up to the timeout for at least one match, then returns an empty
        list if there is none. A timeout of 0 checks once without waiting.
        """
        params: Dict[str, Any] = {"context": self._context_id, "timeout": timeout}
        if selector is not None:
            params["selector"] = selector
        else:
            for key, val in [("role", role), ("text", text), ("label", label),
                             ("placeholder", placeholder), ("alt", alt), ("title", title),
                             ("testid", testid), ("xpath", xpath), ("near", near)]:
                if val is not None:
                    params[key] = val

        result = await self._client.send("vibium:page.findAll", params)
        sel_str = selector or ""
        sel_params = {"selector": selector} if selector else {
            k: v for k, v in params.items() if k not in ("context", "timeout")
        }
        elements = []
        for el in result["elements"]:
            info = ElementInfo(tag=el["tag"], text=el["text"], box=BoundingBox(**el["box"]))
            elements.append(Element(self._client, self._context_id, sel_str, info, el.get("index"), sel_params))
        return elements

    # --- Waiting ---

    @property
    def capture(self) -> _CaptureNamespace:
        """Capture namespace — set up a listener before performing an action."""
        return _CaptureNamespace(self)

    @property
    def wait_until(self) -> _WaitUntilNamespace:
        """Deprecated alias — use wait_for_function / wait_for_url / wait_for_load."""
        warnings.warn(
            "wait_until is deprecated; use wait_for_function, wait_for_url, or wait_for_load",
            DeprecationWarning,
            stacklevel=2,
        )
        return _WaitUntilNamespace(self)

    async def wait_for_function(self, fn: str, timeout: Optional[int] = None) -> Any:
        """Wait until a function returns a truthy value."""
        return await self._wait_for_function(fn, timeout)

    async def wait_for_url(self, pattern: str, timeout: Optional[int] = None) -> None:
        """Wait until the page URL matches a pattern."""
        await self._wait_for_url(pattern, timeout)

    async def wait_for_load(self, state: Optional[str] = None, timeout: Optional[int] = None) -> None:
        """Wait until the page reaches a load state."""
        await self._wait_for_load(state, timeout)

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
        """Internal: wait for a response matching a URL pattern.

        The binary matches the pattern and waits for the event; this just
        awaits the command.
        """
        self._ensure_data_collector()
        result = await self._client.send("vibium:page.captureResponse", {
            "context": self._context_id, "pattern": pattern, "timeout": timeout or 10000,
        })
        return Response(result["event"], self._client)

    async def _setup_capture_response(self, pattern: str, timeout: Optional[int] = None) -> Any:
        """Internal: start a response capture now, return a task to await later."""
        return asyncio.get_running_loop().create_task(self._capture_response(pattern, timeout))

    async def _capture_request(self, pattern: str, timeout: Optional[int] = None) -> Request:
        """Internal: wait for a request matching a URL pattern."""
        self._ensure_data_collector()
        result = await self._client.send("vibium:page.captureRequest", {
            "context": self._context_id, "pattern": pattern, "timeout": timeout or 10000,
        })
        return Request(result["event"], self._client)

    async def _setup_capture_request(self, pattern: str, timeout: Optional[int] = None) -> Any:
        """Internal: start a request capture now, return a task to await later."""
        return asyncio.get_running_loop().create_task(self._capture_request(pattern, timeout))
    async def _capture_event_params(self, kind: str, timeout: Optional[int] = None) -> Dict[str, Any]:
        """Internal: one-shot capture, waited out in the engine
        (vibium:page.captureEvent), so no client keeps its own listener and
        timeout machinery (#446). Returns the raw event params."""
        result = await self._client.send("vibium:page.captureEvent", {
            "context": self._context_id, "kind": kind, "timeout": timeout or 10000,
        })
        return result["event"]

    async def _capture_navigation(self, timeout: Optional[int] = None) -> str:
        """Internal: wait for a navigation event. Resolves with URL."""
        params = await self._capture_event_params("navigation", timeout)
        return params.get("url", "")

    async def _setup_capture_navigation(self, timeout: Optional[int] = None) -> Any:
        """Internal: start a navigation capture now, return a task to await later."""
        return asyncio.get_running_loop().create_task(self._capture_navigation(timeout))

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
            raise errors.TimeoutError("Timeout waiting for download")

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
                raise errors.TimeoutError("Timeout waiting for download")

        return _wait()

    def _sync_dialog_policy(self) -> None:
        """Tell the engine whether dialogs are handled here.

        Without a handler the engine dismisses each dialog itself (#446).
        send_setup, so the policy is acknowledged before any later command
        can trigger a dialog.
        """
        manual = bool(self._dialog_callbacks)
        if manual == self._dialog_policy_manual:
            return
        self._dialog_policy_manual = manual
        params = {"context": self._context_id, "policy": "manual" if manual else "dismiss"}
        try:
            asyncio.get_running_loop()
        except RuntimeError:
            # The sync API registers handlers from its caller thread; the
            # policy command has to be scheduled onto the client's loop.
            self._client.send_setup_threadsafe("vibium:dialog.setPolicy", params)
            return
        self._client.send_setup("vibium:dialog.setPolicy", params)

    async def _capture_dialog(self, timeout: Optional[int] = None) -> Dialog:
        """Internal: wait for a dialog event. The pending engine capture keeps
        the dialog from being auto-dismissed."""
        params = await self._capture_event_params("dialog", timeout)
        return Dialog(self._client, self._context_id, params)

    async def _setup_capture_dialog(self, timeout: Optional[int] = None, auto_dismiss: bool = False) -> Any:
        """Internal: start a dialog capture now, return a task to await later.

        auto_dismiss dismisses the dialog the moment it is captured, not when
        the task is awaited. The sync wrappers use it: they return only the
        dialog's data, so the caller has no handle to close the dialog with,
        and a dialog left open blocks the page, including a trigger fn stuck
        inside evaluate("alert(...)"), which otherwise deadlocks the capture
        (#146).
        """
        task = asyncio.get_running_loop().create_task(self._capture_dialog(timeout))
        if auto_dismiss:
            def _dismiss(finished: Any) -> None:
                if not finished.cancelled() and finished.exception() is None:
                    asyncio.ensure_future(finished.result().dismiss())
            task.add_done_callback(_dismiss)
        return task

    _CAPTURE_EVENT_NAMES = ("request", "response", "download", "navigation", "dialog", "console", "error")

    async def _capture_event(self, name: str, timeout: Optional[int] = None) -> Any:
        """Internal: wait for a named event."""
        if name == "request":
            return await self._capture_request("**", timeout)
        if name == "response":
            return await self._capture_response("**", timeout)
        if name == "download":
            return await self._capture_download(timeout)
        if name == "navigation":
            return await self._capture_navigation(timeout)
        if name == "dialog":
            return await self._capture_dialog(timeout)
        if name == "console":
            return ConsoleMessage(await self._capture_event_params("console", timeout))
        if name == "error":
            params = await self._capture_event_params("error", timeout)
            return Exception(params.get("text", "Unknown error"))
        raise ValueError(f"Unknown event name: '{name}'")

    async def _setup_capture_event(self, name: str, timeout: Optional[int] = None) -> Any:
        """Internal: start an event capture now, return a task to await later."""
        if name not in self._CAPTURE_EVENT_NAMES:
            raise ValueError(f"Unknown event name: '{name}'")
        return asyncio.get_running_loop().create_task(self._capture_event(name, timeout))

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

    async def pdf(
        self,
        *,
        landscape: Optional[bool] = None,
        scale: Optional[float] = None,
        background: Optional[bool] = None,
        margin_top: Optional[float] = None,
        margin_bottom: Optional[float] = None,
        margin_left: Optional[float] = None,
        margin_right: Optional[float] = None,
        page_width: Optional[float] = None,
        page_height: Optional[float] = None,
        page_ranges: Optional[List[Union[int, str]]] = None,
        shrink_to_fit: Optional[bool] = None,
    ) -> bytes:
        """Print the page to PDF. Returns PDF bytes. Only works in headless mode.

        Unset options keep the browser's print defaults (portrait, scale 1,
        1cm margins, no background, letter-size page, all pages). Margins and
        page size are in cm; page_ranges takes ints and range strings, e.g.
        [1, "3-5"].
        """
        params: Dict[str, Any] = {"context": self._context_id}
        for key, val in [
            ("landscape", landscape), ("scale", scale), ("background", background),
            ("marginTop", margin_top), ("marginBottom", margin_bottom),
            ("marginLeft", margin_left), ("marginRight", margin_right),
            ("pageWidth", page_width), ("pageHeight", page_height),
            ("pageRanges", page_ranges), ("shrinkToFit", shrink_to_fit),
        ]:
            if val is not None:
                params[key] = val
        result = await self._client.send("vibium:page.pdf", params)
        return base64.b64decode(result["data"])

    # --- Evaluation ---

    async def evaluate(self, expression: str) -> Any:
        """Evaluate a JS expression and return the deserialized value."""
        result = await self._client.send("vibium:page.eval", {
            "context": self._context_id, "expression": expression,
        })
        return result["value"]

    async def eval(self, expression: str) -> Any:
        """Alias for :meth:`evaluate` (issue #144)."""
        return await self.evaluate(expression)

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

    async def emulate_media(
        self,
        *,
        media: Optional[str] = None,
        color_scheme: Optional[str] = None,
        reduced_motion: Optional[str] = None,
        forced_colors: Optional[str] = None,
        contrast: Optional[str] = None,
    ) -> None:
        """Override CSS media features."""
        params: Dict[str, Any] = {"context": self._context_id}
        if media is not None:
            params["media"] = media
        if color_scheme is not None:
            params["colorScheme"] = color_scheme
        if reduced_motion is not None:
            params["reducedMotion"] = reduced_motion
        if forced_colors is not None:
            params["forcedColors"] = forced_colors
        if contrast is not None:
            params["contrast"] = contrast
        await self._client.send("vibium:page.emulateMedia", params)

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
        if self._event_handler is not None:
            self._client.remove_event_handler(self._event_handler)
        await self._client.send("browsingContext.close", {"context": self._context_id})

    # --- Scrolling ---

    async def scroll(self, direction: str = "down", amount: int = 3, selector: Optional[str] = None) -> None:
        """Scroll the page in a direction (up/down/left/right)."""
        await self._client.send("vibium:page.scroll", {
            "context": self._context_id,
            "direction": direction,
            "amount": amount,
            "selector": selector,
        })

    # --- Network Interception ---

    async def route(self, pattern: str, handler: Callable[[Route], Any]) -> None:
        """Intercept network requests matching a URL pattern.

        The binary compiles the pattern, owns the intercept lifecycle, and
        annotates blocked request events with the patterns that matched, so
        dispatch never interprets the glob client-side.
        """
        result = await self._client.send(
            "vibium:page.route", {"context": self._context_id, "pattern": pattern}
        )
        self._intercept_id = result["intercept"]

        self._ensure_data_collector()
        self._routes.append({"pattern": pattern, "handler": handler, "interceptId": self._intercept_id})

    async def unroute(self, pattern: str) -> None:
        """Remove a previously registered route."""
        removed = sum(1 for r in self._routes if r["pattern"] == pattern)
        self._routes = [r for r in self._routes if r["pattern"] != pattern]
        # The binary refcounts pattern registrations and tears the intercept
        # down when the last one goes.
        for _ in range(removed):
            await self._client.send(
                "vibium:page.unroute", {"context": self._context_id, "pattern": pattern}
            )
        if not self._routes:
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
        """Listen for WebSocket connections opened by the page.

        Monitoring is installed in the engine before the next command on this
        connection is sent, so a socket opened by the very next call cannot be
        missed (#351).
        """
        self._ws_callbacks.append(fn)
        # Keyed on the setup state, not the callback count: after a failed
        # install the callbacks are still registered, and the next
        # registration must retry the install or they can never fire.
        if self._ws_setup is None:
            setup = self._client.send_setup(
                "vibium:page.onWebSocket", {"context": self._context_id}
            )
            self._ws_setup = setup

            def _reset(finished: asyncio.Future) -> None:
                if finished.cancelled():
                    if self._ws_setup is setup:
                        self._ws_setup = None
                elif finished.exception() is not None:
                    # Reset so a later listener retries; sockets are
                    # unmonitored until then. Guarded: a retry made in the
                    # meantime owns the state.
                    if self._ws_setup is setup:
                        self._ws_setup = None
                    logger.debug(
                        "page.on_web_socket setup failed: %s", finished.exception()
                    )

            setup.add_done_callback(_reset)

    async def _when_web_socket_setup(self) -> None:
        """Wait until this page's WebSocket monitor is installed.

        Raises if the install failed. The sync wrapper awaits this so its
        blocking on_web_socket() reports a failure the async caller cannot
        see.
        """
        # Captured before awaiting: a failed install resets _ws_setup to
        # None, and the raise must come from the setup this caller
        # registered under.
        setup = self._ws_setup
        if setup is not None:
            await setup


    # --- Dialog Handling ---

    def on_dialog(self, handler: Callable[[Dialog], Any]) -> None:
        self._dialog_callbacks.append(handler)
        self._sync_dialog_policy()

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
            self._sync_dialog_policy()
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

        # send_setup, not ensure_future(send): the collector must exist before
        # the request whose body a route/on_response handler is about to read
        # (#351).
        task = self._client.send_setup(
            "network.addDataCollector",
            {"dataTypes": ["request", "response"], "maxEncodedDataSize": 10 * 1024 * 1024},
        )

        def _store(finished: Any) -> None:
            if finished.cancelled() or finished.exception() is not None:
                # Reset so a later listener retries; bodies are unavailable
                # until then.
                self._data_collector_id = None
                if not finished.cancelled():
                    logger.debug(
                        "page._ensure_data_collector failed: %s", finished.exception()
                    )
                return
            self._data_collector_id = (finished.result() or {}).get("collector")

        task.add_done_callback(_store)

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
        elif method in ("browsingContext.fragmentNavigated", "browsingContext.historyUpdated"):
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
            # The binary already matched the URL against every registered
            # pattern (vibiumMatchedPatterns), so dispatch is a membership
            # check, not a glob evaluation.
            matched = params.get("vibiumMatchedPatterns") or []
            req = Request(params, self._client)

            for route_entry in self._routes:
                if route_entry["pattern"] in matched:
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
        # With no handler registered the engine dismisses the dialog itself
        # (#446), so there is nothing to do here but deliver.
        dialog = Dialog(self._client, self._context_id, params)

        for cb in list(self._dialog_callbacks):
            try:
                result = cb(dialog)
                if hasattr(result, "__await__"):
                    import asyncio
                    asyncio.ensure_future(result)
            except Exception:
                pass

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
