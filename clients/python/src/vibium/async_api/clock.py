"""Async Clock class for faking timers and Date."""

from __future__ import annotations

from datetime import datetime, timezone
from typing import Any, Dict, Optional, Union, TYPE_CHECKING

if TYPE_CHECKING:
    from ..client import BiDiClient


def _normalize_time(time: Union[int, str]) -> int:
    """Convert a time value to epoch milliseconds.

    Accepts an integer (epoch ms) or an ISO 8601 date string.
    """
    if isinstance(time, int):
        return time
    if isinstance(time, float):
        return int(time)
    # Parse ISO string → epoch ms
    dt = datetime.fromisoformat(time.replace("Z", "+00:00"))
    return int(dt.timestamp() * 1000)


class Clock:
    """Page-scoped clock control for faking timers and Date."""

    def __init__(self, client: BiDiClient, context_id: str) -> None:
        self._client = client
        self._context_id = context_id

    async def install(
        self,
        time: Optional[Union[int, str]] = None,
        timezone: Optional[str] = None,
    ) -> None:
        """Install fake clock, overriding Date, setTimeout, setInterval, etc."""
        params: Dict[str, Any] = {"context": self._context_id}
        if time is not None:
            params["time"] = _normalize_time(time)
        if timezone is not None:
            params["timezone"] = timezone
        await self._client.send("vibium:clock.install", params)

    async def fast_forward(self, ticks: int) -> None:
        """Jump forward by ticks ms, fire each due timer at most once."""
        await self._client.send("vibium:clock.fastForward", {
            "context": self._context_id,
            "ticks": ticks,
        })

    async def run_for(self, ticks: int) -> None:
        """Advance ticks ms, firing all callbacks systematically."""
        await self._client.send("vibium:clock.runFor", {
            "context": self._context_id,
            "ticks": ticks,
        })

    async def pause_at(self, time: Union[int, str]) -> None:
        """Jump to a specific time and pause."""
        await self._client.send("vibium:clock.pauseAt", {
            "context": self._context_id,
            "time": _normalize_time(time),
        })

    async def resume(self) -> None:
        """Resume real-time progression from current fake time."""
        await self._client.send("vibium:clock.resume", {
            "context": self._context_id,
        })

    async def set_fixed_time(self, time: Union[int, str]) -> None:
        """Freeze Date.now() at a value permanently. Timers still run."""
        await self._client.send("vibium:clock.setFixedTime", {
            "context": self._context_id,
            "time": _normalize_time(time),
        })

    async def set_system_time(self, time: Union[int, str]) -> None:
        """Set Date.now() without triggering timers."""
        await self._client.send("vibium:clock.setSystemTime", {
            "context": self._context_id,
            "time": _normalize_time(time),
        })

    async def set_timezone(self, timezone: str) -> None:
        """Override the browser timezone."""
        await self._client.send("vibium:clock.setTimezone", {
            "context": self._context_id,
            "timezone": timezone,
        })
