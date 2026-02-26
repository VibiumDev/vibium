"""WebSocket client for communicating with the vibium binary."""

from __future__ import annotations

import asyncio
import json
from typing import Any, Callable, Dict, List, Optional

from websockets.asyncio.client import ClientConnection, connect as ws_connect
from websockets.exceptions import ConnectionClosed


class BiDiError(Exception):
    """Raised when a BiDi command fails."""

    def __init__(self, error: str, message: str):
        self.error = error
        self.message = message
        super().__init__(f"{error}: {message}")


class BiDiClient:
    """WebSocket client for BiDi protocol with event dispatch."""

    def __init__(self, ws: ClientConnection):
        self._ws = ws
        self._next_id = 1
        self._pending: Dict[int, asyncio.Future] = {}
        self._receiver_task: Optional[asyncio.Task] = None
        self._event_handlers: List[Callable[[Dict[str, Any]], None]] = []

    @classmethod
    async def connect(cls, url: str) -> BiDiClient:
        """Connect to a BiDi WebSocket server."""
        ws = await ws_connect(url)
        client = cls(ws)
        client._receiver_task = asyncio.create_task(client._receive_loop())
        return client

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
        """Background task to receive and dispatch messages."""
        try:
            async for message in self._ws:
                data = json.loads(message)
                msg_id = data.get("id")
                if msg_id is not None and msg_id in self._pending:
                    self._pending[msg_id].set_result(data)
                elif msg_id is None and "method" in data:
                    # Event message — dispatch to handlers
                    for handler in self._event_handlers:
                        try:
                            handler(data)
                        except Exception:
                            pass
        except ConnectionClosed:
            for future in self._pending.values():
                if not future.done():
                    future.set_exception(ConnectionError("Connection closed"))

    async def send(self, method: str, params: Optional[Dict[str, Any]] = None) -> Any:
        """Send a command and wait for the response."""
        msg_id = self._next_id
        self._next_id += 1

        command = {
            "id": msg_id,
            "method": method,
            "params": params or {},
        }

        future: asyncio.Future = asyncio.get_event_loop().create_future()
        self._pending[msg_id] = future

        try:
            await self._ws.send(json.dumps(command))
            response = await future

            if response.get("type") == "error":
                raise BiDiError(
                    response.get("error", "unknown"),
                    response.get("message", "Unknown error"),
                )

            return response.get("result")
        finally:
            self._pending.pop(msg_id, None)

    async def close(self) -> None:
        """Close the WebSocket connection."""
        if self._receiver_task:
            self._receiver_task.cancel()
            try:
                await self._receiver_task
            except asyncio.CancelledError:
                pass

        await self._ws.close()
