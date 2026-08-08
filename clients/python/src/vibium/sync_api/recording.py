"""Sync Recording wrapper."""

from __future__ import annotations

from typing import Any, Dict, Optional, Union, TYPE_CHECKING

from ..async_api.recording import _UNSET, RecordingResult

if TYPE_CHECKING:
    from .._sync_base import _EventLoopThread
    from ..async_api.recording import Recording as AsyncRecording


class Recording:
    """Synchronous wrapper for async Recording."""

    def __init__(self, async_recording: AsyncRecording, loop_thread: _EventLoopThread) -> None:
        self._async = async_recording
        self._loop = loop_thread

    def start(
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
        self._loop.run(self._async.start(name=name, screenshots=screenshots,
                                          snapshots=snapshots, sources=sources,
                                          title=title, bidi=bidi,
                                          format=format, quality=quality,
                                          video=video, path=path))

    def stop(self, path: Optional[str] = None) -> RecordingResult:
        return self._loop.run(self._async.stop(path=path))

    def start_chunk(self, name: Optional[str] = None, title: Optional[str] = None) -> None:
        self._loop.run(self._async.start_chunk(name=name, title=title))

    def stop_chunk(self, path: Optional[str] = None) -> RecordingResult:
        return self._loop.run(self._async.stop_chunk(path=path))

    def start_group(self, name: str, location: Optional[Dict[str, Any]] = None) -> None:
        self._loop.run(self._async.start_group(name, location=location))

    def stop_group(self) -> None:
        self._loop.run(self._async.stop_group())
