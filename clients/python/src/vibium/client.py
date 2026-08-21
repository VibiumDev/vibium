"""Pipe client for communicating with the vibium binary via stdin/stdout."""

from __future__ import annotations

import asyncio
import json
from typing import Any, Callable, Dict, List, Optional, Set, TYPE_CHECKING

from . import errors
from .errors import BiDiError

if TYPE_CHECKING:
    from .binary import VibiumProcess


# Waiting for event setup would deadlock this one: an open dialog stops the
# browser answering anything else for that context, including the
# script.callFunction that installs a WebSocket monitor, and this is the only
# command that closes the dialog. Mirrors unblocksAnotherCommand in the Go
# router.
_NEVER_WAITS = "browsingContext.handleUserPrompt"


class BiDiClient:
    """Pipe client for BiDi protocol with event dispatch."""

    def __init__(
        self,
        process: VibiumProcess,
    ):
        self._process = process
        self._stdin = process._process.stdin
        self._stdout = process._process.stdout
        self._next_id = 1
        self._pending: Dict[int, asyncio.Future] = {}
        self._receiver_task: Optional[asyncio.Task] = None
        self._event_handlers: List[Callable[[Dict[str, Any]], None]] = []
        # Once the connection is gone nothing will ever resolve a response
        # future, so a command must fail rather than wait out its timeout.
        # The setup gate below makes this reachable: a command parked behind
        # setup can resume after close() has already drained the pending map.
        self._closed = False

        # Event registration (page.on_web_socket, network.addDataCollector) is
        # triggered from synchronous callback APIs with nothing for the caller
        # to await, so its command used to be handed to ensure_future and
        # forgotten. The next command, an evaluate that opens a socket, could
        # reach the engine first, and the one-shot event was lost (#351).
        #
        # send_setup() registers a task here that send() waits for.
        # Settle-only: this sequences commands and nothing more. A failed
        # setup is surfaced by whoever owns that registration, not injected
        # into an unrelated command.
        self._pending_setups: Set[asyncio.Future] = set()

    @classmethod
    async def connect(cls, process: VibiumProcess) -> BiDiClient:
        """Create a BiDiClient from a VibiumProcess with pipe streams."""
        client = cls(process)
        client._receiver_task = asyncio.create_task(client._receive_loop())
        # Replay any events that arrived before the ready signal
        if hasattr(process, "_pre_ready_lines"):
            for line in process._pre_ready_lines:
                client._dispatch_message(line)
            del process._pre_ready_lines
        return client

    def _dispatch_message(self, line: str) -> None:
        """Parse and dispatch a single message line."""
        try:
            data = json.loads(line)
        except (json.JSONDecodeError, ValueError):
            return
        msg_id = data.get("id")
        if msg_id is not None and msg_id in self._pending:
            future = self._pending[msg_id]
            if not future.done():
                future.set_result(data)
        elif msg_id is None and "method" in data:
            for handler in self._event_handlers:
                try:
                    handler(data)
                except Exception:
                    pass

    def on_event(self, handler: Callable[[Dict[str, Any]], None]) -> None:
        """Register an event handler for messages without an id (events)."""
        self._event_handlers.append(handler)

    def remove_event_handler(self, handler: Callable[[Dict[str, Any]], None]) -> None:
        """Remove a previously registered event handler."""
        try:
            self._event_handlers.remove(handler)
        except ValueError:
            pass

    async def _receive_loop(self) -> None:
        """Background task to receive and dispatch messages from stdout."""
        try:
            while True:
                line_bytes = await self._stdout.readline()  # type: ignore[union-attr]
                if not line_bytes:
                    break  # EOF — process exited
                line = line_bytes.decode().strip()
                if not line:
                    continue
                self._dispatch_message(line)
        except (asyncio.CancelledError, OSError):
            pass
        finally:
            self._closed = True
            for future in self._pending.values():
                if not future.done():
                    future.set_exception(errors.ConnectionError("Connection closed"))

    async def send(self, method: str, params: Optional[Dict[str, Any]] = None, timeout: float = 60) -> Any:
        """Send a command and wait for the response.

        Waits first for any event registration still being acknowledged, so a
        command cannot overtake the setup it depends on (#351).
        """
        if self._pending_setups and method != _NEVER_WAITS:
            # wait(), not gather(): a failed registration is reported by its
            # owner, not raised out of an unrelated command.
            await asyncio.wait(set(self._pending_setups))
        return await self._send(method, params, timeout)

    def send_setup(
        self,
        method: str,
        params: Optional[Dict[str, Any]] = None,
        timeout: float = 60,
    ) -> asyncio.Future:
        """Send a command whose acknowledgement the next command depends on.

        For event registration issued from a synchronous callback API, where
        the caller has no coroutine to await (#351). Returns the task, so
        whoever owns the registration can await it and surface a failure.
        """
        task = asyncio.ensure_future(self._send(method, params, timeout))
        self._pending_setups.add(task)

        def _done(finished: asyncio.Future) -> None:
            self._pending_setups.discard(finished)
            if not finished.cancelled():
                # Retrieve it here so a failure the async caller cannot see is
                # not reported as "exception was never retrieved". Awaiting
                # the task later still raises, which is how the sync wrapper
                # reports.
                finished.exception()

        task.add_done_callback(_done)
        return task

    async def _send(self, method: str, params: Optional[Dict[str, Any]] = None, timeout: float = 60) -> Any:
        """Write a command and wait for its response, bypassing the setup gate."""
        if self._closed:
            raise errors.ConnectionError("Connection closed")

        msg_id = self._next_id
        self._next_id += 1

        command = {
            "id": msg_id,
            "method": method,
            "params": params or {},
        }

        future: asyncio.Future = asyncio.get_running_loop().create_future()
        self._pending[msg_id] = future

        try:
            line = json.dumps(command) + "\n"
            self._stdin.write(line.encode())  # type: ignore[union-attr]
            # drain() carries no deadline of its own. If the vibium process
            # stops reading stdin, it blocks before the response timer below
            # ever starts, turning a wedged pipe into a hang that outlives
            # every client timeout (#397).
            try:
                await asyncio.wait_for(self._stdin.drain(), timeout=timeout)  # type: ignore[union-attr]
            except asyncio.TimeoutError:
                raise errors.TimeoutError(
                    f"Command '{method}' was not accepted by the vibium process after {timeout}s"
                )
            try:
                response = await asyncio.wait_for(future, timeout=timeout)
            except asyncio.TimeoutError:
                raise errors.TimeoutError(f"Command '{method}' timed out after {timeout}s")

            if response.get("type") == "error":
                error_code = response.get("error", "unknown")
                error_message = response.get("message", "Unknown error")
                if "element not found" in error_message:
                    raise errors.ElementNotFoundError(error_message)
                if error_code == "timeout":
                    raise errors.TimeoutError(error_message)
                raise BiDiError(error_code, error_message)

            return response.get("result")
        finally:
            self._pending.pop(msg_id, None)

    async def close(self) -> None:
        """Close the pipe connection."""
        self._closed = True

        # Cancelling the receiver leaves the setup tasks unresolved; drop
        # them so a command racing close() fails on the closed connection
        # rather than waiting for setup nobody will finish.
        for task in list(self._pending_setups):
            task.cancel()
        self._pending_setups.clear()

        if self._receiver_task:
            self._receiver_task.cancel()
            try:
                await self._receiver_task
            except asyncio.CancelledError:
                pass

        # Close stdin to signal the pipe process
        if self._stdin and not self._stdin.is_closing():
            self._stdin.close()
