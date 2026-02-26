"""WebSocket info data object."""

from __future__ import annotations

from typing import Callable, List, Optional


class WebSocketInfo:
    """Represents a WebSocket connection opened by the page."""

    def __init__(self, url: str) -> None:
        self._url = url
        self._is_closed = False
        self._message_handlers: List[Callable[[str, dict], None]] = []
        self._close_handlers: List[Callable[[Optional[int], Optional[str]], None]] = []

    def url(self) -> str:
        return self._url

    def on_message(self, fn: Callable[[str, dict], None]) -> None:
        """Register handler for messages. fn(data, info) where info has 'direction'."""
        self._message_handlers.append(fn)

    def on_close(self, fn: Callable[[Optional[int], Optional[str]], None]) -> None:
        """Register handler for close. fn(code, reason)."""
        self._close_handlers.append(fn)

    def is_closed(self) -> bool:
        return self._is_closed

    def _emit_message(self, data: str, direction: str) -> None:
        """Internal: called by Page when a ws message event fires."""
        for fn in self._message_handlers:
            fn(data, {"direction": direction})

    def _emit_close(self, code: Optional[int] = None, reason: Optional[str] = None) -> None:
        """Internal: called by Page when a ws close event fires."""
        self._is_closed = True
        for fn in self._close_handlers:
            fn(code, reason)
