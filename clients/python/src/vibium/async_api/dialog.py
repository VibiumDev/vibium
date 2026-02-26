"""Dialog data object."""

from __future__ import annotations

from typing import Any, Dict, Optional, TYPE_CHECKING

if TYPE_CHECKING:
    from ..client import BiDiClient


def _is_dialog_race_error(e: Exception) -> bool:
    msg = str(e)
    return ("Connection closed" in msg or
            "no such alert" in msg or
            "No dialog" in msg)


class Dialog:
    """Represents a browser dialog (alert, confirm, prompt, beforeunload)."""

    def __init__(self, client: BiDiClient, context_id: str, data: Dict[str, Any]) -> None:
        self._client = client
        self._context_id = context_id
        self._data = data

    def type(self) -> str:
        """The dialog type: 'alert', 'confirm', 'prompt', or 'beforeunload'."""
        return self._data.get("type", "alert")

    def message(self) -> str:
        """The dialog message text."""
        return self._data.get("message", "")

    def default_value(self) -> str:
        """The default value for prompt dialogs."""
        return self._data.get("defaultValue", "")

    async def accept(self, prompt_text: Optional[str] = None) -> None:
        """Accept the dialog. For prompt dialogs, optionally provide text."""
        try:
            params: Dict[str, Any] = {
                "context": self._context_id,
                "accept": True,
            }
            if prompt_text is not None:
                params["userText"] = prompt_text
            await self._client.send("browsingContext.handleUserPrompt", params)
        except Exception as e:
            if _is_dialog_race_error(e):
                return
            raise

    async def dismiss(self) -> None:
        """Dismiss the dialog (cancel/close)."""
        try:
            await self._client.send("browsingContext.handleUserPrompt", {
                "context": self._context_id,
                "accept": False,
            })
        except Exception as e:
            if _is_dialog_race_error(e):
                return
            raise
