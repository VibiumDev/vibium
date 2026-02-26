"""Sync Dialog wrapper using decision pattern."""

from __future__ import annotations

from typing import Any, Dict, Optional, TYPE_CHECKING

if TYPE_CHECKING:
    from ..async_api.dialog import Dialog as AsyncDialog


class Dialog:
    """Sync wrapper for a browser dialog.

    The user's handler calls accept() or dismiss() to set the decision.
    If none is called, the default is 'dismiss'.
    """

    def __init__(self, async_dialog: AsyncDialog) -> None:
        self._async = async_dialog
        self._decision: Dict[str, Any] = {"action": "dismiss"}

    def type(self) -> str:
        return self._async.type()

    def message(self) -> str:
        return self._async.message()

    def default_value(self) -> str:
        return self._async.default_value()

    def accept(self, prompt_text: Optional[str] = None) -> None:
        self._decision = {"action": "accept", "prompt_text": prompt_text}

    def dismiss(self) -> None:
        self._decision = {"action": "dismiss"}
