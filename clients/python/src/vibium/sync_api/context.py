"""Sync BrowserContext wrapper."""

from __future__ import annotations

from typing import Any, Dict, List, Optional, TYPE_CHECKING

from .._types import Cookie, SetCookieParam, StorageState
from .recording import Recording

if TYPE_CHECKING:
    from .._sync_base import _EventLoopThread
    from ..async_api.context import BrowserContext as AsyncBrowserContext
    from .page import Page


class BrowserContext:
    """Synchronous wrapper for async BrowserContext."""

    def __init__(self, async_context: AsyncBrowserContext, loop_thread: _EventLoopThread) -> None:
        self._async = async_context
        self._loop = loop_thread
        self._recording: Optional[Recording] = None

    @property
    def id(self) -> str:
        """The user context ID for this browser context."""
        return self._async.id

    @property
    def recording(self) -> Recording:
        """Get the Recording for this context."""
        if self._recording is None:
            self._recording = Recording(self._async.recording, self._loop)
        return self._recording

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

    def storage(self) -> StorageState:
        return self._loop.run(self._async.storage())

    def set_storage(self, state: StorageState) -> None:
        self._loop.run(self._async.set_storage(state))

    def clear_storage(self) -> None:
        self._loop.run(self._async.clear_storage())

    def add_init_script(self, script: str) -> str:
        return self._loop.run(self._async.add_init_script(script))
