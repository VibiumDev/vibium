"""Sync Page wrapper."""

from __future__ import annotations

import shutil
from typing import Any, Callable, Dict, List, Optional, Union, TYPE_CHECKING

from .._types import A11yNode
from .element import Element
from .element_list import ElementList
from .clock import Clock
from .route import Route

if TYPE_CHECKING:
    from .._sync_base import _EventLoopThread
    from ..async_api.page import Page as AsyncPage
    from .context import BrowserContext as BrowserContextType


class Keyboard:
    """Sync keyboard input."""

    def __init__(self, async_keyboard: Any, loop_thread: _EventLoopThread) -> None:
        self._async = async_keyboard
        self._loop = loop_thread

    def press(self, key: str) -> None:
        self._loop.run(self._async.press(key))

    def down(self, key: str) -> None:
        self._loop.run(self._async.down(key))

    def up(self, key: str) -> None:
        self._loop.run(self._async.up(key))

    def type(self, text: str) -> None:
        self._loop.run(self._async.type(text))


class Mouse:
    """Sync mouse input."""

    def __init__(self, async_mouse: Any, loop_thread: _EventLoopThread) -> None:
        self._async = async_mouse
        self._loop = loop_thread

    def click(self, x: float, y: float) -> None:
        self._loop.run(self._async.click(x, y))

    def move(self, x: float, y: float) -> None:
        self._loop.run(self._async.move(x, y))

    def down(self) -> None:
        self._loop.run(self._async.down())

    def up(self) -> None:
        self._loop.run(self._async.up())

    def wheel(self, delta_x: float, delta_y: float) -> None:
        self._loop.run(self._async.wheel(delta_x, delta_y))


class Touch:
    """Sync touch input."""

    def __init__(self, async_touch: Any, loop_thread: _EventLoopThread) -> None:
        self._async = async_touch
        self._loop = loop_thread

    def tap(self, x: float, y: float) -> None:
        self._loop.run(self._async.tap(x, y))


class Page:
    """Synchronous wrapper for async Page."""

    def __init__(self, async_page: AsyncPage, loop_thread: _EventLoopThread) -> None:
        self._async = async_page
        self._loop = loop_thread

        self.keyboard = Keyboard(async_page.keyboard, loop_thread)
        self.mouse = Mouse(async_page.mouse, loop_thread)
        self.touch = Touch(async_page.touch, loop_thread)
        self.clock = Clock(async_page.clock, loop_thread)

        # Sync event state
        self._console_messages: List[Dict[str, str]] = []
        self._errors: List[Dict[str, str]] = []
        self._cached_context: Optional[BrowserContextType] = None

    @property
    def id(self) -> str:
        return self._async.id

    @property
    def context(self) -> BrowserContextType:
        """The parent BrowserContext that owns this page."""
        if self._cached_context is None:
            from .context import BrowserContext
            self._cached_context = BrowserContext(self._async.context, self._loop)
        return self._cached_context

    # --- Navigation ---

    def go(self, url: str) -> None:
        self._loop.run(self._async.go(url))

    def back(self) -> None:
        self._loop.run(self._async.back())

    def forward(self) -> None:
        self._loop.run(self._async.forward())

    def reload(self) -> None:
        self._loop.run(self._async.reload())

    # --- Info ---

    def url(self) -> str:
        return self._loop.run(self._async.url())

    def title(self) -> str:
        return self._loop.run(self._async.title())

    def content(self) -> str:
        return self._loop.run(self._async.content())

    # --- Finding ---

    def find(
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
        async_el = self._loop.run(self._async.find(
            selector, role=role, text=text, label=label, placeholder=placeholder,
            alt=alt, title=title, testid=testid, xpath=xpath, near=near, timeout=timeout,
        ))
        return Element(async_el, self._loop)

    def find_all(
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
        async_list = self._loop.run(self._async.find_all(
            selector, role=role, text=text, label=label, placeholder=placeholder,
            alt=alt, title=title, testid=testid, xpath=xpath, near=near, timeout=timeout,
        ))
        return ElementList(async_list, self._loop)

    # --- Waiting ---

    @property
    def capture(self) -> _SyncCaptureNamespace:
        """Capture namespace — set up a listener before performing an action."""
        return _SyncCaptureNamespace(self)

    @property
    def wait_until(self) -> _SyncWaitUntilNamespace:
        """Wait until a condition is met. Callable or use .url() / .loaded() sub-methods."""
        return _SyncWaitUntilNamespace(self)

    def wait(self, ms: int) -> None:
        self._loop.run(self._async.wait(ms))

    # --- Screenshots & PDF ---

    def screenshot(
        self,
        full_page: Optional[bool] = None,
        clip: Optional[Dict[str, Any]] = None,
    ) -> bytes:
        return self._loop.run(self._async.screenshot(full_page=full_page, clip=clip))

    def pdf(self) -> bytes:
        return self._loop.run(self._async.pdf())

    # --- Evaluation ---

    def eval(self, expression: str) -> Any:
        return self._loop.run(self._async.eval(expression))

    def evaluate(self, script: str) -> Any:
        """Execute a JS script (multi-statement, use 'return' for values)."""
        return self._loop.run(self._async.evaluate(script))

    def eval_handle(self, expression: str) -> str:
        return self._loop.run(self._async.eval_handle(expression))

    def add_script(self, source: str) -> None:
        self._loop.run(self._async.add_script(source))

    def add_style(self, source: str) -> None:
        self._loop.run(self._async.add_style(source))

    def expose(self, name: str, fn: str) -> None:
        self._loop.run(self._async.expose(name, fn))

    # --- Emulation ---

    def set_viewport(self, size: Dict[str, int]) -> None:
        self._loop.run(self._async.set_viewport(size))

    def viewport(self) -> Dict[str, int]:
        return self._loop.run(self._async.viewport())

    def emulate_media(self, **opts: Any) -> None:
        self._loop.run(self._async.emulate_media(**opts))

    def set_content(self, html: str) -> None:
        self._loop.run(self._async.set_content(html))

    def set_geolocation(self, coords: Dict[str, float]) -> None:
        self._loop.run(self._async.set_geolocation(coords))

    def set_window(self, **options: Any) -> None:
        self._loop.run(self._async.set_window(**options))

    def window(self) -> Dict[str, Any]:
        return self._loop.run(self._async.window())

    # --- Accessibility ---

    def a11y_tree(
        self,
        everything: Optional[bool] = None,
        root: Optional[str] = None,
    ) -> A11yNode:
        return self._loop.run(self._async.a11y_tree(everything, root))

    # --- Frames ---

    def frames(self) -> List[Page]:
        async_frames = self._loop.run(self._async.frames())
        return [Page(f, self._loop) for f in async_frames]

    def frame(self, name_or_url: str) -> Optional[Page]:
        async_frame = self._loop.run(self._async.frame(name_or_url))
        if async_frame is None:
            return None
        return Page(async_frame, self._loop)

    def main_frame(self) -> Page:
        return self

    # --- Lifecycle ---

    def bring_to_front(self) -> None:
        self._loop.run(self._async.bring_to_front())

    def close(self) -> None:
        self._loop.run(self._async.close())

    # --- Network ---

    def route(
        self,
        pattern: str,
        action: Union[str, Dict[str, Any], Callable[[Route], None]],
    ) -> None:
        """Intercept network requests.

        action can be:
          - 'continue' — pass through
          - 'abort' — block the request
          - dict — static fulfill ({status, body, headers})
          - callable — handler function receiving Route
        """
        if isinstance(action, str):
            if action == "abort":
                async def _abort_handler(async_route: Any) -> None:
                    await async_route.abort()
                self._loop.run(self._async.route(pattern, _abort_handler))
            else:  # 'continue'
                async def _continue_handler(async_route: Any) -> None:
                    await async_route.continue_()
                self._loop.run(self._async.route(pattern, _continue_handler))
        elif isinstance(action, dict):
            fulfill_opts = action

            async def _fulfill_handler(async_route: Any) -> None:
                await async_route.fulfill(**fulfill_opts)
            self._loop.run(self._async.route(pattern, _fulfill_handler))
        else:
            # Callable handler — use sync decision pattern
            def _sync_callback(async_route: Any) -> None:
                sync_route = Route(async_route)
                action(sync_route)
                decision = sync_route._decision
                import asyncio
                # Safe: this callback fires from the background event loop thread,
                # so ensure_future schedules onto the already-running loop.
                if decision["action"] == "fulfill":
                    opts = {k: v for k, v in decision.items() if k != "action" and v is not None}
                    asyncio.ensure_future(async_route.fulfill(**opts))
                elif decision["action"] == "abort":
                    asyncio.ensure_future(async_route.abort())
                else:
                    opts = {k: v for k, v in decision.items() if k != "action" and v is not None}
                    asyncio.ensure_future(async_route.continue_(**opts))

            self._loop.run(self._async.route(pattern, _sync_callback))

    def unroute(self, pattern: str) -> None:
        self._loop.run(self._async.unroute(pattern))

    def set_headers(self, headers: Dict[str, str]) -> None:
        self._loop.run(self._async.set_headers(headers))

    # --- Events ---

    def on_dialog(
        self,
        action: Union[str, Callable],
    ) -> None:
        """Handle browser dialogs.

        action can be:
          - 'accept' — auto-accept
          - 'dismiss' — auto-dismiss
          - callable — handler function receiving Dialog (sync)
        """
        from .dialog import Dialog as SyncDialog

        if isinstance(action, str):
            async def _simple_handler(dialog: Any) -> None:
                if action == "accept":
                    await dialog.accept()
                else:
                    await dialog.dismiss()
            self._async.on_dialog(_simple_handler)
        else:
            def _sync_callback(dialog: Any) -> None:
                sync_dialog = SyncDialog(dialog)
                action(sync_dialog)
                decision = sync_dialog._decision
                import asyncio
                # Safe: this callback fires from the background event loop thread,
                # so ensure_future schedules onto the already-running loop.
                if decision["action"] == "accept":
                    asyncio.ensure_future(dialog.accept(decision.get("prompt_text")))
                else:
                    asyncio.ensure_future(dialog.dismiss())

            self._async.on_dialog(_sync_callback)

    def on_console(self, mode: str = "collect") -> None:
        """Start collecting console messages. Retrieve with console_messages()."""
        def _collector(msg: Any) -> None:
            self._console_messages.append({"type": msg.type(), "text": msg.text()})
        self._async.on_console(_collector)

    def console_messages(self) -> List[Dict[str, str]]:
        """Return collected console messages."""
        return list(self._console_messages)

    def on_error(self, mode: str = "collect") -> None:
        """Start collecting page errors. Retrieve with errors()."""
        def _collector(error: Exception) -> None:
            self._errors.append({"message": str(error)})
        self._async.on_error(_collector)

    def errors(self) -> List[Dict[str, str]]:
        """Return collected errors."""
        return list(self._errors)

    def on_request(self, fn: Callable) -> None:
        """Register a callback for every outgoing request.

        fn receives a request-like object with sync methods:
        url(), method(), headers(), post_data().
        """
        import asyncio

        async def _wrapper(req: Any) -> None:
            try:
                pd = await req.post_data()
                fn(_SyncRequestData(req, pd))
            except Exception as e:
                import sys
                print(f"vibium: error in on_request callback: {e}", file=sys.stderr)

        def _sync_wrapper(req: Any) -> None:
            # Safe: callbacks fire from the background event loop thread,
            # so ensure_future schedules onto the already-running loop.
            asyncio.ensure_future(_wrapper(req))

        # Must run via loop thread: on_request() calls _ensure_data_collector()
        # which uses asyncio.ensure_future() and needs a running event loop.
        async def _register() -> None:
            self._async.on_request(_sync_wrapper)
        self._loop.run(_register())

    def on_response(self, fn: Callable) -> None:
        """Register a callback for every completed response.

        fn receives a response-like object with sync methods:
        url(), status(), headers(), body(), json().
        """
        import asyncio

        async def _wrapper(resp: Any) -> None:
            try:
                b = await resp.body()
                fn(_SyncResponseData(resp, b))
            except Exception as e:
                import sys
                print(f"vibium: error in on_response callback: {e}", file=sys.stderr)

        def _sync_wrapper(resp: Any) -> None:
            # Safe: callbacks fire from the background event loop thread,
            # so ensure_future schedules onto the already-running loop.
            asyncio.ensure_future(_wrapper(resp))

        # Must run via loop thread: on_response() calls _ensure_data_collector()
        # which uses asyncio.ensure_future() and needs a running event loop.
        async def _register() -> None:
            self._async.on_response(_sync_wrapper)
        self._loop.run(_register())

    def on_download(self, fn: Callable) -> None:
        """Register a callback for file downloads.

        fn receives a SyncDownload object with methods: url(), suggested_filename(),
        path(), save_as(), and dict access for backward compatibility.
        """
        import asyncio

        async def _wrapper(dl: Any) -> None:
            try:
                file_path = await dl.path()
                sync_dl = SyncDownload({
                    "url": dl.url(),
                    "suggested_filename": dl.suggested_filename(),
                    "path": file_path,
                })
                fn(sync_dl)
            except Exception as e:
                import sys
                print(f"vibium: error in on_download callback: {e}", file=sys.stderr)

        def _sync_wrapper(dl: Any) -> None:
            asyncio.ensure_future(_wrapper(dl))

        async def _register() -> None:
            self._async.on_download(_sync_wrapper)
        self._loop.run(_register())

    def on_web_socket(self, fn: Callable) -> None:
        """Listen for WebSocket connections opened by the page.

        fn receives a WebSocketInfo object with sync methods: url(), on_message(), on_close(), is_closed().
        """
        # Must run via loop thread: on_web_socket() uses asyncio.ensure_future()
        # internally and needs a running event loop.
        async def _register() -> None:
            self._async.on_web_socket(fn)
        self._loop.run(_register())

    def remove_all_listeners(self, event: Optional[str] = None) -> None:
        self._async.remove_all_listeners(event)
        if not event or event == "console":
            self._console_messages.clear()
        if not event or event == "error":
            self._errors.clear()
        if not event or event == "navigation":
            pass  # navigation callbacks are on async page


class _SyncRequestData:
    """Lightweight wrapper exposing request data as sync methods."""

    def __init__(self, req: Any, post_data: Optional[str]) -> None:
        self._req = req
        self._post_data = post_data

    def url(self) -> str:
        return self._req.url()

    def method(self) -> str:
        return self._req.method()

    def headers(self) -> Dict[str, str]:
        return self._req.headers()

    def post_data(self) -> Optional[str]:
        return self._post_data


class _SyncResponseData:
    """Lightweight wrapper exposing response data as sync methods."""

    def __init__(self, resp: Any, body: Optional[str]) -> None:
        self._resp = resp
        self._body = body

    def url(self) -> str:
        return self._resp.url()

    def status(self) -> int:
        return self._resp.status()

    def headers(self) -> Dict[str, str]:
        return self._resp.headers()

    def body(self) -> Optional[str]:
        return self._body

    def json(self) -> Any:
        import json as _json
        if self._body is None:
            return None
        return _json.loads(self._body)


class SyncDownload(dict):
    """Download result with dict access (backward-compatible) plus save_as().

    Supports result["url"], result["suggested_filename"], result["path"],
    and result.save_as(dest).
    """

    def save_as(self, dest: str) -> None:
        """Copy the downloaded file to *dest*."""
        src = self.get("path")
        if not src:
            raise RuntimeError("Download failed or path not available")
        import os
        os.makedirs(os.path.dirname(os.path.abspath(dest)), exist_ok=True)
        shutil.copy2(src, dest)

    def url(self) -> str:
        return self["url"]

    def suggested_filename(self) -> str:
        return self["suggested_filename"]

    def path(self) -> Optional[str]:
        return self.get("path")


class _SyncCapturedResponse:
    """Returned by capture.response(). Usable as context manager or direct call."""

    def __init__(self, page: Page, pattern: str, timeout: Optional[int] = None) -> None:
        self._page = page
        self._pattern = pattern
        self._timeout = timeout
        self._wait_coro: Any = None
        self.value: Optional[Dict[str, Any]] = None

    def __enter__(self) -> _SyncCapturedResponse:
        self._wait_coro = self._page._loop.run(
            self._page._async._setup_capture_response(self._pattern, self._timeout)
        )
        return self

    def __exit__(self, exc_type: Any, exc_val: Any, exc_tb: Any) -> None:
        if exc_type is None and self._wait_coro:
            resp = self._page._loop.run(self._wait_coro)
            body = self._page._loop.run(resp.body())
            self.value = {"url": resp.url(), "status": resp.status(), "headers": resp.headers(), "body": body}


class _SyncCapturedRequest:
    """Returned by capture.request(). Usable as context manager or direct call."""

    def __init__(self, page: Page, pattern: str, timeout: Optional[int] = None) -> None:
        self._page = page
        self._pattern = pattern
        self._timeout = timeout
        self._wait_coro: Any = None
        self.value: Optional[Dict[str, Any]] = None

    def __enter__(self) -> _SyncCapturedRequest:
        self._wait_coro = self._page._loop.run(
            self._page._async._setup_capture_request(self._pattern, self._timeout)
        )
        return self

    def __exit__(self, exc_type: Any, exc_val: Any, exc_tb: Any) -> None:
        if exc_type is None and self._wait_coro:
            req = self._page._loop.run(self._wait_coro)
            post_data = self._page._loop.run(req.post_data())
            self.value = {"url": req.url(), "method": req.method(), "headers": req.headers(), "post_data": post_data}


class _SyncCapturedNavigation:
    """Returned by capture.navigation(). Usable as context manager or direct call."""

    def __init__(self, page: Page, timeout: Optional[int] = None) -> None:
        self._page = page
        self._timeout = timeout
        self._wait_coro: Any = None
        self.value: Optional[str] = None

    def __enter__(self) -> _SyncCapturedNavigation:
        self._wait_coro = self._page._loop.run(
            self._page._async._setup_capture_navigation(self._timeout)
        )
        return self

    def __exit__(self, exc_type: Any, exc_val: Any, exc_tb: Any) -> None:
        if exc_type is None and self._wait_coro:
            self.value = self._page._loop.run(self._wait_coro)


class _SyncCapturedDownload:
    """Returned by capture.download(). Usable as context manager or direct call."""

    def __init__(self, page: Page, timeout: Optional[int] = None) -> None:
        self._page = page
        self._timeout = timeout
        self._wait_coro: Any = None
        self.value: Optional[Dict[str, Any]] = None

    def __enter__(self) -> _SyncCapturedDownload:
        self._wait_coro = self._page._loop.run(
            self._page._async._setup_capture_download(self._timeout)
        )
        return self

    def __exit__(self, exc_type: Any, exc_val: Any, exc_tb: Any) -> None:
        if exc_type is None and self._wait_coro:
            dl = self._page._loop.run(self._wait_coro)
            file_path = self._page._loop.run(dl.path())
            self.value = SyncDownload({"url": dl.url(), "suggested_filename": dl.suggested_filename(), "path": file_path})


class _SyncCapturedDialog:
    """Returned by capture.dialog(). Usable as context manager or direct call."""

    def __init__(self, page: Page, timeout: Optional[int] = None) -> None:
        self._page = page
        self._timeout = timeout
        self._wait_coro: Any = None
        self.value: Optional[Dict[str, Any]] = None

    def __enter__(self) -> _SyncCapturedDialog:
        self._wait_coro = self._page._loop.run(
            self._page._async._setup_capture_dialog(self._timeout)
        )
        return self

    def __exit__(self, exc_type: Any, exc_val: Any, exc_tb: Any) -> None:
        if exc_type is None and self._wait_coro:
            dialog = self._page._loop.run(self._wait_coro)
            self.value = {
                "type": dialog.type(),
                "message": dialog.message(),
                "default_value": dialog.default_value(),
            }


class _SyncCapturedEvent:
    """Returned by capture.event(). Usable as context manager or direct call."""

    def __init__(self, page: Page, name: str, timeout: Optional[int] = None) -> None:
        self._page = page
        self._name = name
        self._timeout = timeout
        self._wait_coro: Any = None
        self.value: Any = None

    def __enter__(self) -> _SyncCapturedEvent:
        self._wait_coro = self._page._loop.run(
            self._page._async._setup_capture_event(self._name, self._timeout)
        )
        return self

    def __exit__(self, exc_type: Any, exc_val: Any, exc_tb: Any) -> None:
        if exc_type is None and self._wait_coro:
            self.value = self._page._loop.run(self._wait_coro)


class _SyncCaptureNamespace:
    """Namespace for capture methods on sync Page."""

    def __init__(self, page: Page) -> None:
        self._page = page

    def response(self, pattern: str, fn: Optional[Callable] = None, timeout: Optional[int] = None) -> Union[Dict[str, Any], _SyncCapturedResponse]:
        """Wait for a response matching a URL pattern.

        With fn: sets up listener, runs fn, waits, returns dict.
        Without fn: returns context manager. Use with-block and .value.
        """
        if fn is not None:
            wait_coro = self._page._loop.run(
                self._page._async._setup_capture_response(pattern, timeout)
            )
            fn()
            resp = self._page._loop.run(wait_coro)
            body = self._page._loop.run(resp.body())
            return {"url": resp.url(), "status": resp.status(), "headers": resp.headers(), "body": body}
        return _SyncCapturedResponse(self._page, pattern, timeout)

    def request(self, pattern: str, fn: Optional[Callable] = None, timeout: Optional[int] = None) -> Union[Dict[str, Any], _SyncCapturedRequest]:
        """Wait for a request matching a URL pattern.

        With fn: sets up listener, runs fn, waits, returns dict.
        Without fn: returns context manager. Use with-block and .value.
        """
        if fn is not None:
            wait_coro = self._page._loop.run(
                self._page._async._setup_capture_request(pattern, timeout)
            )
            fn()
            req = self._page._loop.run(wait_coro)
            post_data = self._page._loop.run(req.post_data())
            return {"url": req.url(), "method": req.method(), "headers": req.headers(), "post_data": post_data}
        return _SyncCapturedRequest(self._page, pattern, timeout)

    def navigation(self, fn: Optional[Callable] = None, timeout: Optional[int] = None) -> Union[str, _SyncCapturedNavigation]:
        """Wait for a navigation event. Resolves with URL string."""
        if fn is not None:
            wait_coro = self._page._loop.run(
                self._page._async._setup_capture_navigation(timeout)
            )
            fn()
            return self._page._loop.run(wait_coro)
        return _SyncCapturedNavigation(self._page, timeout)

    def download(self, fn: Optional[Callable] = None, timeout: Optional[int] = None) -> Union[SyncDownload, _SyncCapturedDownload]:
        """Wait for a download event."""
        if fn is not None:
            wait_coro = self._page._loop.run(
                self._page._async._setup_capture_download(timeout)
            )
            fn()
            dl = self._page._loop.run(wait_coro)
            file_path = self._page._loop.run(dl.path())
            return SyncDownload({"url": dl.url(), "suggested_filename": dl.suggested_filename(), "path": file_path})
        return _SyncCapturedDownload(self._page, timeout)

    def dialog(self, fn: Optional[Callable] = None, timeout: Optional[int] = None) -> Union[Dict[str, Any], _SyncCapturedDialog]:
        """Wait for a dialog event."""
        if fn is not None:
            wait_coro = self._page._loop.run(
                self._page._async._setup_capture_dialog(timeout)
            )
            fn()
            dialog = self._page._loop.run(wait_coro)
            return {
                "type": dialog.type(),
                "message": dialog.message(),
                "default_value": dialog.default_value(),
            }
        return _SyncCapturedDialog(self._page, timeout)

    def event(self, name: str, fn: Optional[Callable] = None, timeout: Optional[int] = None) -> Union[Any, _SyncCapturedEvent]:
        """Wait for a named event."""
        if fn is not None:
            wait_coro = self._page._loop.run(
                self._page._async._setup_capture_event(name, timeout)
            )
            fn()
            return self._page._loop.run(wait_coro)
        return _SyncCapturedEvent(self._page, name, timeout)


class _SyncWaitUntilNamespace:
    """Namespace for waitUntil methods on sync Page. Also callable for waitForFunction."""

    def __init__(self, page: Page) -> None:
        self._page = page

    def __call__(self, fn: str, timeout: Optional[int] = None) -> Any:
        """Wait until a function returns a truthy value."""
        return self._page._loop.run(self._page._async._wait_for_function(fn, timeout))

    def url(self, pattern: str, timeout: Optional[int] = None) -> None:
        """Wait until the page URL matches a pattern."""
        self._page._loop.run(self._page._async._wait_for_url(pattern, timeout))

    def loaded(self, state: Optional[str] = None, timeout: Optional[int] = None) -> None:
        """Wait until the page reaches a load state."""
        self._page._loop.run(self._page._async._wait_for_load(state, timeout))
