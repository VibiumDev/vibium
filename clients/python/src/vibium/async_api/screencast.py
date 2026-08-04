"""Async Screencast class."""

from __future__ import annotations

import base64
from typing import Any, Dict, Optional, TYPE_CHECKING

if TYPE_CHECKING:
    from ..client import BiDiClient


class Screencast:
    """Native browser video recording (WebDriver BiDi screencast).

    Supported on Firefox 154+. Chrome has not implemented the BiDi screencast
    commands yet; start() fails there with an explanatory error.
    """

    def __init__(self, client: BiDiClient, context_id: str) -> None:
        self._client = client
        self._context_id = context_id

    async def start(
        self,
        mime_type: Optional[str] = None,
        width: Optional[int] = None,
        height: Optional[int] = None,
        frame_rate: Optional[int] = None,
        audio: Optional[bool] = None,
    ) -> None:
        """Start recording this page.

        Args:
            mime_type: Video MIME type (browser default if omitted, typically video/webm).
            width: Requested video width in pixels.
            height: Requested video height in pixels.
            frame_rate: Requested frame rate.
            audio: Record page audio as well (default False).
                Firefox 154 does not support this yet.
        """
        params: Dict[str, Any] = {"context": self._context_id}
        if mime_type is not None:
            params["mimeType"] = mime_type
        if width is not None:
            params["width"] = width
        if height is not None:
            params["height"] = height
        if frame_rate is not None:
            params["frameRate"] = frame_rate
        if audio is not None:
            params["audio"] = audio
        await self._client.send("vibium:screencast.start", params)

    async def stop(self, path: Optional[str] = None) -> bytes:
        """Stop recording and return the video as bytes.

        Args:
            path: Save the video to this path. Omit to only get the bytes back.
        """
        params: Dict[str, Any] = {}
        if path is not None:
            params["path"] = path
        result = await self._client.send("vibium:screencast.stop", params)

        if path and result.get("path"):
            with open(result["path"], "rb") as f:
                return f.read()

        return base64.b64decode(result.get("data", ""))
