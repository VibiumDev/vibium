"""Sync Clock wrapper."""

from __future__ import annotations

from typing import Optional, Union, TYPE_CHECKING

if TYPE_CHECKING:
    from .._sync_base import _EventLoopThread
    from ..async_api.clock import Clock as AsyncClock


class Clock:
    """Synchronous wrapper for async Clock."""

    def __init__(self, async_clock: AsyncClock, loop_thread: _EventLoopThread) -> None:
        self._async = async_clock
        self._loop = loop_thread

    def install(self, time: Optional[Union[int, str]] = None, timezone: Optional[str] = None) -> None:
        self._loop.run(self._async.install(time=time, timezone=timezone))

    def fast_forward(self, ticks: int) -> None:
        self._loop.run(self._async.fast_forward(ticks))

    def run_for(self, ticks: int) -> None:
        self._loop.run(self._async.run_for(ticks))

    def pause_at(self, time: Union[int, str]) -> None:
        self._loop.run(self._async.pause_at(time))

    def resume(self) -> None:
        self._loop.run(self._async.resume())

    def set_fixed_time(self, time: Union[int, str]) -> None:
        self._loop.run(self._async.set_fixed_time(time))

    def set_system_time(self, time: Union[int, str]) -> None:
        self._loop.run(self._async.set_system_time(time))

    def set_timezone(self, timezone: str) -> None:
        self._loop.run(self._async.set_timezone(timezone))
