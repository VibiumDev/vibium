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
        req = self._data.get("request", {})
        return req.get("url", "")

    def method(self) -> str:
        req = self._data.get("request", {})
        return req.get("method", "")

    def headers(self) -> Dict[str, str]:
        req = self._data.get("request", {})
        return _convert_headers(req.get("headers", []))

    def request_id(self) -> str:
        req = self._data.get("request", {})
        return req.get("request", "")

    async def post_data(self) -> Optional[str]:
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
        resp = self._data.get("response", {})
        return resp.get("url", "") or self._data.get("url", "")

    def status(self) -> int:
        resp = self._data.get("response", {})
        return resp.get("status", 0)

    def headers(self) -> Dict[str, str]:
        resp = self._data.get("response", {})
        return _convert_headers(resp.get("headers", []))

    def request_id(self) -> str:
        req = self._data.get("request", {})
        return req.get("request", "")

    async def body(self) -> Optional[str]:
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
        import json as _json
        text = await self.body()
        if text is None:
            return None
        return _json.loads(text)
