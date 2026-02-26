"""Background event loop thread for synchronous wrappers."""

from __future__ import annotations

import asyncio
import threading
from typing import Any, Optional


class _EventLoopThread:
    """Manages a background thread running an asyncio event loop."""

    def __init__(self) -> None:
        self._loop: Optional[asyncio.AbstractEventLoop] = None
        self._thread: Optional[threading.Thread] = None

    def start(self) -> asyncio.AbstractEventLoop:
        """Start the background event loop thread."""
        if self._loop is not None:
            return self._loop

        self._loop = asyncio.new_event_loop()
        self._thread = threading.Thread(target=self._run_loop, daemon=True)
        self._thread.start()
        return self._loop

    def _run_loop(self) -> None:
        """Run the event loop in the background thread."""
        asyncio.set_event_loop(self._loop)
        self._loop.run_forever()  # type: ignore[union-attr]

    def run(self, coro: Any) -> Any:
        """Run a coroutine in the background loop and wait for result."""
        if self._loop is None:
            raise RuntimeError("Event loop not started")
        future = asyncio.run_coroutine_threadsafe(coro, self._loop)
        return future.result()

    def stop(self) -> None:
        """Stop the event loop and thread."""
        if self._loop:
            self._loop.call_soon_threadsafe(self._loop.stop)
        if self._thread:
            self._thread.join(timeout=5)
        self._loop = None
        self._thread = None
