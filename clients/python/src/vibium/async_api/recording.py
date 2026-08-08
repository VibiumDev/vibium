"""Async Recording class."""

from __future__ import annotations

import base64
import os
import re
import time
from dataclasses import dataclass, field
from typing import Any, Dict, List, Optional, Union, TYPE_CHECKING

if TYPE_CHECKING:
    from ..client import BiDiClient

# Distinguishes "path not given" (timestamped default) from an explicit
# path=None (bytes-only capture).
_UNSET: Any = object()


@dataclass
class RecordingResult:
    """What recording.stop() / stop_chunk() delivered."""

    path: Optional[str] = None
    """Where the zip landed. None for bytes-only recordings."""
    data: Optional[bytes] = None
    """The zip itself. Present only for bytes-only recordings (path=None at start)."""
    steps: Optional[int] = None
    duration_ms: Optional[int] = None
    videos: List[Dict[str, Any]] = field(default_factory=list)
    video_unavailable: Optional[str] = None
    """Why no video was recorded (video omitted on an engine without support)."""


def _result_from_wire(result: Dict[str, Any]) -> RecordingResult:
    data = result.get("data")
    return RecordingResult(
        path=result.get("path"),
        data=base64.b64decode(data) if data else None,
        steps=result.get("steps"),
        duration_ms=result.get("durationMs"),
        videos=result.get("videos") or [],
        video_unavailable=result.get("videoUnavailable"),
    )


def _default_record_path(name: Optional[str] = None) -> str:
    """Timestamped default destination so a rerun never clobbers the previous artifact.

    The recording's name, sanitized, seeds the stem: name "login" yields
    login-20260808-094123.zip.
    """
    stem = re.sub(r"[^A-Za-z0-9._-]", "-", name or "").strip("-.") or "record"
    stamp = time.strftime("%Y%m%d-%H%M%S")
    path = f"{stem}-{stamp}.zip"
    n = 2
    while os.path.exists(path):
        path = f"{stem}-{stamp}-{n}.zip"
        n += 1
    return path


class Recording:
    """Context-scoped recording."""

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
        video: Optional[Union[bool, Dict[str, Any]]] = None,
        path: Optional[str] = _UNSET,
    ) -> None:
        """Start recording.

        Args:
            format: Screenshot format — 'jpeg' (default, faster/smaller) or 'png' (lossless).
            quality: JPEG quality 0.0-1.0 (default 0.5). Ignored for PNG.
            video: Video track (Firefox 154+, local browsers). Omitted: record
                video if the engine supports it; the stop result reports
                videoUnavailable otherwise. True (or a dict with "width",
                "height", "frame_rate"): start fails if the engine can't
                deliver. False: off.
            path: Where the recording zip lands at stop. Defaults to a
                timestamped record-YYYYMMDD-HHMMSS.zip in the working
                directory so a rerun never clobbers the previous one. None
                selects bytes-only capture: no file is written and the
                recording is lost if the session closes before stop().
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
        if video is not None:
            if isinstance(video, dict):
                # snake_case keys map to the wire's camelCase
                params["video"] = {
                    {"frame_rate": "frameRate"}.get(k, k): v for k, v in video.items()
                }
            else:
                params["video"] = video
        if path is _UNSET:
            path = _default_record_path(name)
        if path is not None:
            # The binary's working directory is not necessarily ours, so
            # relative paths resolve here before going over the wire.
            params["path"] = os.path.abspath(path)
        await self._client.send("vibium:recording.start", params)

    async def stop(self, path: Optional[str] = None) -> RecordingResult:
        """Stop recording and deliver the zip to the declared path.

        path overrides the path declared at start. The result carries where
        the zip landed and what it holds; the bytes are included only for
        bytes-only recordings (path=None at start).
        """
        params: Dict[str, Any] = {"userContext": self._user_context_id}
        if path is not None:
            params["path"] = os.path.abspath(path)
        result = await self._client.send("vibium:recording.stop", params)
        return _result_from_wire(result)

    async def start_chunk(
        self,
        name: Optional[str] = None,
        title: Optional[str] = None,
    ) -> None:
        """Start a new recording chunk."""
        params: Dict[str, Any] = {"userContext": self._user_context_id}
        if name is not None:
            params["name"] = name
        if title is not None:
            params["title"] = title
        await self._client.send("vibium:recording.startChunk", params)

    async def stop_chunk(self, path: Optional[str] = None) -> RecordingResult:
        """Stop the current chunk.

        The result carries the path it was written to, or the chunk's bytes
        when no path was given.
        """
        params: Dict[str, Any] = {"userContext": self._user_context_id}
        if path is not None:
            params["path"] = os.path.abspath(path)
        result = await self._client.send("vibium:recording.stopChunk", params)
        return _result_from_wire(result)

    async def start_group(
        self,
        name: str,
        location: Optional[Dict[str, Any]] = None,
    ) -> None:
        """Start a named group of actions in the recording."""
        params: Dict[str, Any] = {"userContext": self._user_context_id, "name": name}
        if location is not None:
            params["location"] = location
        await self._client.send("vibium:recording.startGroup", params)

    async def stop_group(self) -> None:
        """End the current group."""
        await self._client.send("vibium:recording.stopGroup", {
            "userContext": self._user_context_id,
        })
