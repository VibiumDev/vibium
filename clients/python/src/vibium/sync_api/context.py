"""Sync BrowserContext wrapper."""

from __future__ import annotations

from typing import Any, Dict, List, Optional, TYPE_CHECKING

from .._types import Cookie, SetCookieParam, StorageState
from .tracing import Tracing

if TYPE_CHECKING:
    from .._sync_base import _EventLoopThread
    from ..async_api.context import BrowserContext as AsyncBrowserContext
    from .page import Page


class BrowserContext:
    """Synchronous wrapper for async BrowserContext."""

    def __init__(self, async_context: AsyncBrowserContext, loop_thread: _EventLoopThread) -> None:
        self._async = async_context
        self._loop = loop_thread
        self._tracing: Optional[Tracing] = None

    @property
    def id(self) -> str:
        return self._async.id

    @property
    def tracing(self) -> Tracing:
        if self._tracing is None:
            self._tracing = Tracing(self._async.tracing, self._loop)
        return self._tracing

    def new_page(self) -> Page:
        from .page import Page
        async_page = self._loop.run(self._async.new_page())
        return Page(async_page, self._loop)

    def close(self) -> None:
        self._loop.run(self._async.close())

    def cookies(self, urls: Optional[List[str]] = None) -> List[Cookie]:
        return self._loop.run(self._async.cookies(urls))

    def set_cookies(self, cookies: List[SetCookieParam]) -> None:
        self._loop.run(self._async.set_cookies(cookies))

    def clear_cookies(self) -> None:
        self._loop.run(self._async.clear_cookies())

    def storage_state(self) -> StorageState:
        return self._loop.run(self._async.storage_state())

    def add_init_script(self, script: str) -> str:
        return self._loop.run(self._async.add_init_script(script))
