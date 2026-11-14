"""Async BrowserContext class."""

from __future__ import annotations

from typing import Any, Dict, List, Optional, TYPE_CHECKING

from .._types import Cookie, SetCookieParam, StorageState

if TYPE_CHECKING:
    from ..client import BiDiClient
    from .page import Page
    from .recording import Recording as RecordingType


class BrowserContext:
    """An isolated browser context (incognito-like)."""

    def __init__(self, client: BiDiClient, user_context_id: str) -> None:
        self._client = client
        self._user_context_id = user_context_id
        self._recording: Optional[RecordingType] = None

    @property
    def id(self) -> str:
        """The user context ID for this browser context."""
        return self._user_context_id

    @property
    def recording(self) -> RecordingType:
        """Get the Recording for this context."""
        if self._recording is None:
            from .recording import Recording
            self._recording = Recording(self._client, self._user_context_id)
        return self._recording

    async def new_page(self) -> Page:
        """Create a new page (tab) in this context."""
        from .page import Page
        result = await self._client.send("vibium:context.newPage", {
            "userContext": self._user_context_id,
        })
        return Page(self._client, result["context"], self._user_context_id)

    async def close(self) -> None:
        """Close this context and all its pages."""
        await self._client.send("browser.removeUserContext", {
            "userContext": self._user_context_id,
        })

    async def cookies(self, urls: Optional[List[str]] = None) -> List[Cookie]:
        """Get cookies for this context."""
        params: Dict[str, Any] = {"userContext": self._user_context_id}
        if urls:
            params["urls"] = urls
        result = await self._client.send("vibium:context.cookies", params)
        return result["cookies"]

    async def set_cookies(self, cookies: List[SetCookieParam]) -> None:
        """Set cookies in this context."""
        await self._client.send("vibium:context.setCookies", {
            "userContext": self._user_context_id,
            "cookies": cookies,
        })

    async def clear_cookies(self) -> None:
        """Clear all cookies in this context."""
        await self._client.send("vibium:context.clearCookies", {
            "userContext": self._user_context_id,
        })

    async def storage(self) -> StorageState:
        """Get the storage state (cookies + localStorage + sessionStorage)."""
        return await self._client.send("vibium:context.storage", {
            "userContext": self._user_context_id,
        })

    async def set_storage(self, state: StorageState) -> None:
        """Set the storage state (cookies + localStorage + sessionStorage)."""
        await self._client.send("vibium:context.setStorage", {
            "userContext": self._user_context_id,
            "state": state,
        })

    async def clear_storage(self) -> None:
        """Clear all storage (cookies + localStorage + sessionStorage)."""
        await self._client.send("vibium:context.clearStorage", {
            "userContext": self._user_context_id,
        })

    async def add_init_script(self, script: str) -> str:
        """Add an init script that runs before page scripts in this context."""
        result = await self._client.send("vibium:context.addInitScript", {
            "userContext": self._user_context_id,
            "script": script,
        })
        return result["script"]
