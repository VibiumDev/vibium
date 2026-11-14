"""Network request and response data objects."""

from __future__ import annotations

import base64
from typing import Any, Dict, List, Optional, TYPE_CHECKING

if TYPE_CHECKING:
    from ..client import BiDiClient


def _convert_headers(entries: List[Dict[str, Any]]) -> Dict[str, str]:
    """Convert BiDi header entries to simple dict."""
    result: Dict[str, str] = {}
    for entry in entries:
        val = entry.get("value", {})
        result[entry.get("name", "")] = val.get("value", "") if isinstance(val, dict) else str(val)
    return result


class Request:
    """Wraps a BiDi network.beforeRequestSent event."""

    def __init__(self, data: Dict[str, Any], client: Optional[BiDiClient] = None) -> None:
        self._data = data
        self._client = client

    def url(self) -> str:
        """The request URL."""
        req = self._data.get("request", {})
        return req.get("url", "")

    def method(self) -> str:
        """The HTTP method (GET, POST, etc.)."""
        req = self._data.get("request", {})
        return req.get("method", "")

    def headers(self) -> Dict[str, str]:
        """Request headers as a dict."""
        req = self._data.get("request", {})
        return _convert_headers(req.get("headers", []))

    def request_id(self) -> str:
        """The request ID (BiDi network request identifier)."""
        req = self._data.get("request", {})
        return req.get("request", "")

    async def post_data(self) -> Optional[str]:
        """Request body / post data."""
        if not self._client:
            return None
        req_id = self.request_id()
        if not req_id:
            return None
        try:
            result = await self._client.send(
                "network.getData",
                {"dataType": "request", "request": req_id},
            )
            b = result.get("bytes", {})
            return b.get("value") if b else None
        except Exception:
            return None


class Response:
    """Wraps a BiDi network.responseCompleted event."""

    def __init__(self, data: Dict[str, Any], client: Optional[BiDiClient] = None) -> None:
        self._data = data
        self._client = client

    def url(self) -> str:
        """The response URL."""
        resp = self._data.get("response", {})
        return resp.get("url", "") or self._data.get("url", "")

    def status(self) -> int:
        """The HTTP status code."""
        resp = self._data.get("response", {})
        return resp.get("status", 0)

    def headers(self) -> Dict[str, str]:
        """Response headers as a dict."""
        resp = self._data.get("response", {})
        return _convert_headers(resp.get("headers", []))

    def request_id(self) -> str:
        """The request ID associated with this response."""
        req = self._data.get("request", {})
        return req.get("request", "")

    async def body(self) -> Optional[str]:
        """Response body as a string. Requires a data collector (set up by on_response or capture.response)."""
        if not self._client:
            return None
        req_id = self.request_id()
        if not req_id:
            return None
        try:
            result = await self._client.send(
                "network.getData",
                {"dataType": "response", "request": req_id},
            )
            b = result.get("bytes", {})
            if not b or not b.get("value"):
                return None
            if b.get("type") == "base64":
                return base64.b64decode(b["value"]).decode("utf-8")
            return b["value"]
        except Exception:
            return None

    async def json(self) -> Any:
        """Response body parsed as JSON. Requires a data collector (set up by on_response or capture.response)."""
        import json as _json
        text = await self.body()
        if text is None:
            return None
        return _json.loads(text)
