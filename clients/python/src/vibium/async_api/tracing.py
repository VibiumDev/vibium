"""Async Tracing class."""

from __future__ import annotations

import base64
from typing import Any, Dict, Optional, TYPE_CHECKING

if TYPE_CHECKING:
    from ..client import BiDiClient


class Tracing:
    """Context-scoped trace recording."""

    def __init__(self, client: BiDiClient, user_context_id: str) -> None:
        self._client = client
        self._user_context_id = user_context_id

    async def start(
        self,
        name: Optional[str] = None,
        screenshots: Optional[bool] = None,
        snapshots: Optional[bool] = None,
        sources: Optional[bool] = None,
        title: Optional[str] = None,
        bidi: Optional[bool] = None,
        format: Optional[str] = None,
        quality: Optional[float] = None,
    ) -> None:
        """Start trace recording.

        Args:
            format: Screenshot format — 'jpeg' (default, faster/smaller) or 'png' (lossless).
            quality: JPEG quality 0.0-1.0 (default 0.8). Ignored for PNG.
        """
        params: Dict[str, Any] = {"userContext": self._user_context_id}
        if name is not None:
            params["name"] = name
        if screenshots is not None:
            params["screenshots"] = screenshots
        if snapshots is not None:
            params["snapshots"] = snapshots
        if sources is not None:
            params["sources"] = sources
        if title is not None:
            params["title"] = title
        if bidi is not None:
            params["bidi"] = bidi
        if format is not None:
            params["format"] = format
        if quality is not None:
            params["quality"] = quality
        await self._client.send("vibium:tracing.start", params)

    async def stop(self, path: Optional[str] = None) -> bytes:
        """Stop trace recording and return the trace zip as bytes."""
        params: Dict[str, Any] = {"userContext": self._user_context_id}
        if path is not None:
            params["path"] = path
        result = await self._client.send("vibium:tracing.stop", params)

        if path and result.get("path"):
            with open(result["path"], "rb") as f:
                return f.read()

        return base64.b64decode(result.get("data", ""))

    async def start_chunk(
        self,
        name: Optional[str] = None,
        title: Optional[str] = None,
    ) -> None:
        """Start a new trace chunk."""
        params: Dict[str, Any] = {"userContext": self._user_context_id}
        if name is not None:
            params["name"] = name
        if title is not None:
            params["title"] = title
        await self._client.send("vibium:tracing.startChunk", params)

    async def stop_chunk(self, path: Optional[str] = None) -> bytes:
        """Stop the current chunk and return the trace zip as bytes."""
        params: Dict[str, Any] = {"userContext": self._user_context_id}
        if path is not None:
            params["path"] = path
        result = await self._client.send("vibium:tracing.stopChunk", params)

        if path and result.get("path"):
            with open(result["path"], "rb") as f:
                return f.read()

        return base64.b64decode(result.get("data", ""))

    async def start_group(
        self,
        name: str,
        location: Optional[Dict[str, Any]] = None,
    ) -> None:
        """Start a named group of actions in the trace."""
        params: Dict[str, Any] = {"userContext": self._user_context_id, "name": name}
        if location is not None:
            params["location"] = location
        await self._client.send("vibium:tracing.startGroup", params)

    async def stop_group(self) -> None:
        """End the current group."""
        await self._client.send("vibium:tracing.stopGroup", {
            "userContext": self._user_context_id,
        })
