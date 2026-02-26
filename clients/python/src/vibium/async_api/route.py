"""Route class for network interception."""

from __future__ import annotations

from typing import Any, Dict, Optional, TYPE_CHECKING

if TYPE_CHECKING:
    from ..client import BiDiClient
    from .network import Request


def _is_race_error(e: Exception) -> bool:
    msg = str(e)
    return ("Connection closed" in msg or
            "Invalid state" in msg or
            "No blocked request" in msg or
            "no such request" in msg)


class Route:
    """Represents an intercepted network request."""

    def __init__(self, client: BiDiClient, request_id: str, request: Request) -> None:
        self._client = client
        self._request_id = request_id
        self.request = request

    async def fulfill(
        self,
        status: Optional[int] = None,
        headers: Optional[Dict[str, str]] = None,
        content_type: Optional[str] = None,
        body: Optional[str] = None,
    ) -> None:
        """Fulfill the request with a custom response."""
        try:
            params: Dict[str, Any] = {"request": self._request_id}
            if status is not None:
                params["statusCode"] = status
            if headers is not None:
                params["headers"] = headers
            if content_type is not None:
                params["contentType"] = content_type
            if body is not None:
                params["body"] = body
            await self._client.send("vibium:network.fulfill", params)
        except Exception as e:
            if _is_race_error(e):
                return
            raise

    async def continue_(
        self,
        url: Optional[str] = None,
        method: Optional[str] = None,
        headers: Optional[Dict[str, str]] = None,
        post_data: Optional[str] = None,
    ) -> None:
        """Continue the request with optional overrides."""
        try:
            params: Dict[str, Any] = {"request": self._request_id}
            if url is not None:
                params["url"] = url
            if method is not None:
                params["method"] = method
            if headers is not None:
                params["headers"] = headers
            if post_data is not None:
                params["postData"] = post_data
            await self._client.send("vibium:network.continue", params)
        except Exception as e:
            if _is_race_error(e):
                return
            raise

    async def abort(self) -> None:
        """Abort the request."""
        try:
            await self._client.send("network.failRequest", {
                "request": self._request_id,
            })
        except Exception as e:
            if _is_race_error(e):
                return
            raise
