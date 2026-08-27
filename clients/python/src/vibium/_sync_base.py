"""Background event loop thread for synchronous wrappers."""

from __future__ import annotations

import asyncio
import threading
from typing import Any, Optional

from . import errors


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

    def run(self, coro: Any, timeout: float = 120) -> Any:
        """Run a coroutine in the background loop and wait for result.

        Args:
            coro: The coroutine to run.
            timeout: Maximum seconds to wait for the result (default 120s).

        Raises:
            TimeoutError: If the coroutine does not complete within the timeout.
        """
        if self._loop is None:
            raise RuntimeError("Event loop not started")
        if asyncio.isfuture(coro):
            # Tasks already running on the loop (capture setups start their
            # wire command eagerly) cannot be resubmitted; await them instead.
            task = coro

            async def _await():
                return await task

            coro = _await()
        future = asyncio.run_coroutine_threadsafe(coro, self._loop)
        try:
            return future.result(timeout=timeout)
        except BaseException:
            # In Python 3.11+, concurrent.futures.TimeoutError is an alias for
            # the built-in TimeoutError, so we can't distinguish between "the
            # future.result() call timed out" and "the coroutine itself raised
            # TimeoutError" using exception type alone.  Check future.done()
            # instead: if the future completed, re-raise its stored exception;
            # if it's still pending, the wait itself timed out.
            if future.done():
                raise  # coroutine raised — preserve the original exception
            future.cancel()
            raise errors.TimeoutError(
                f"Synchronous call did not complete within {timeout}s — "
                "the background event loop may be stuck or the operation is taking too long"
            )

    def stop(self) -> None:
        """Stop the event loop and thread."""
        if self._loop:
            self._loop.call_soon_threadsafe(self._loop.stop)
        if self._thread:
            self._thread.join(timeout=5)
        self._loop = None
        self._thread = None
