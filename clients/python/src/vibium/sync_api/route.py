"""Sync Route wrapper using decision pattern."""

from __future__ import annotations

from typing import Any, Dict, Optional, TYPE_CHECKING

if TYPE_CHECKING:
    from ..async_api.route import Route as AsyncRoute


class Route:
    """Sync wrapper for an intercepted network request.

    The user's handler calls fulfill(), continue_(), or abort() to set the decision.
    If none is called, the default is 'continue'.
    """

    def __init__(self, async_route: AsyncRoute) -> None:
        self._async = async_route
        self._decision: Dict[str, Any] = {"action": "continue"}

    @property
    def request(self) -> Dict[str, Any]:
        """Request info as a dict."""
        req = self._async.request
        return {
            "url": req.url(),
            "method": req.method(),
            "headers": req.headers(),
        }

    def fulfill(
        self,
        status: Optional[int] = None,
        headers: Optional[Dict[str, str]] = None,
        content_type: Optional[str] = None,
        body: Optional[str] = None,
    ) -> None:
        self._decision = {
            "action": "fulfill",
            "status": status,
            "headers": headers,
            "content_type": content_type,
            "body": body,
        }

    def continue_(
        self,
        url: Optional[str] = None,
        method: Optional[str] = None,
        headers: Optional[Dict[str, str]] = None,
        post_data: Optional[str] = None,
    ) -> None:
        self._decision = {
            "action": "continue",
            "url": url,
            "method": method,
            "headers": headers,
            "post_data": post_data,
        }

    def abort(self) -> None:
        self._decision = {"action": "abort"}
