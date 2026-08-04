"""Sync Screencast wrapper."""

from __future__ import annotations

from typing import Optional, TYPE_CHECKING

if TYPE_CHECKING:
    from .._sync_base import _EventLoopThread
    from ..async_api.screencast import Screencast as AsyncScreencast


class Screencast:
    """Synchronous wrapper for async Screencast."""

    def __init__(self, async_screencast: AsyncScreencast, loop_thread: _EventLoopThread) -> None:
        self._async = async_screencast
        self._loop = loop_thread

    def start(
        self,
        mime_type: Optional[str] = None,
        width: Optional[int] = None,
        height: Optional[int] = None,
        frame_rate: Optional[int] = None,
        audio: Optional[bool] = None,
    ) -> None:
        self._loop.run(self._async.start(mime_type=mime_type, width=width,
                                         height=height, frame_rate=frame_rate,
                                         audio=audio))

    def stop(self, path: Optional[str] = None) -> bytes:
        return self._loop.run(self._async.stop(path=path))
