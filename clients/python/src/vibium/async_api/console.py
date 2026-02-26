"""Console message data object."""

from __future__ import annotations

from typing import Any, Dict, List, Optional


class ConsoleMessage:
    """Represents a single console message from the page."""

    def __init__(self, data: Dict[str, Any]) -> None:
        self._data = data

    def type(self) -> str:
        """The console method: 'log', 'warn', 'error', 'debug', 'info', etc."""
        return self._data.get("method", "log")

    def text(self) -> str:
        """The message text."""
        return self._data.get("text", "")

    def args(self) -> List[Any]:
        """The serialized arguments passed to the console call."""
        return self._data.get("args", [])

    def location(self) -> Optional[Dict[str, Any]]:
        """The source location of the console call, if available."""
        stack = self._data.get("stackTrace")
        if not stack:
            return None
        frames = stack.get("callFrames", [])
        if not frames:
            return None
        frame = frames[0]
        return {
            "url": frame.get("url", ""),
            "lineNumber": frame.get("lineNumber", 0),
            "columnNumber": frame.get("columnNumber", 0),
        }
