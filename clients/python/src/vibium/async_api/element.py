"""Async Element class for interacting with page elements."""

from __future__ import annotations

import base64
from typing import Any, Dict, List, Optional, TYPE_CHECKING

from .._types import BoundingBox, ElementInfo

if TYPE_CHECKING:
    from ..client import BiDiClient
    from .element_list import ElementList


class Element:
    """Represents a DOM element that can be interacted with."""

    def __init__(
        self,
        client: BiDiClient,
        context: str,
        selector: str,
        info: ElementInfo,
        index: Optional[int] = None,
        params: Optional[Dict[str, Any]] = None,
    ) -> None:
        self._client = client
        self._context = context
        self._selector = selector
        self.info = info
        self._index = index
        self._params = params or {}

    def _command_params(self, extra: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Build the common params sent to vibium: commands."""
        p: Dict[str, Any] = {
            **self._params,
            "context": self._context,
            "selector": self._selector,
            "index": self._index,
        }
        if extra:
            p.update(extra)
        return p

    def _to_params(self) -> Dict[str, Any]:
        """Return params that identify this element (for dragTo target)."""
        return {
            **self._params,
            "selector": self._selector,
            "index": self._index,
        }

    # --- Interaction ---

    async def click(self, timeout: Optional[int] = None) -> None:
        await self._client.send("vibium:click", self._command_params({"timeout": timeout}))

    async def dblclick(self, timeout: Optional[int] = None) -> None:
        await self._client.send("vibium:dblclick", self._command_params({"timeout": timeout}))

    async def fill(self, value: str, timeout: Optional[int] = None) -> None:
        await self._client.send("vibium:fill", self._command_params({"value": value, "timeout": timeout}))

    async def type(self, text: str, timeout: Optional[int] = None) -> None:
        await self._client.send("vibium:type", self._command_params({"text": text, "timeout": timeout}))

    async def press(self, key: str, timeout: Optional[int] = None) -> None:
        await self._client.send("vibium:press", self._command_params({"key": key, "timeout": timeout}))

    async def clear(self, timeout: Optional[int] = None) -> None:
        await self._client.send("vibium:clear", self._command_params({"timeout": timeout}))

    async def check(self, timeout: Optional[int] = None) -> None:
        await self._client.send("vibium:check", self._command_params({"timeout": timeout}))

    async def uncheck(self, timeout: Optional[int] = None) -> None:
        await self._client.send("vibium:uncheck", self._command_params({"timeout": timeout}))

    async def select_option(self, value: str, timeout: Optional[int] = None) -> None:
        await self._client.send("vibium:selectOption", self._command_params({"value": value, "timeout": timeout}))

    async def hover(self, timeout: Optional[int] = None) -> None:
        await self._client.send("vibium:hover", self._command_params({"timeout": timeout}))

    async def focus(self, timeout: Optional[int] = None) -> None:
        await self._client.send("vibium:focus", self._command_params({"timeout": timeout}))

    async def drag_to(self, target: Element, timeout: Optional[int] = None) -> None:
        await self._client.send("vibium:dragTo", self._command_params({
            "target": target._to_params(),
            "timeout": timeout,
        }))

    async def tap(self, timeout: Optional[int] = None) -> None:
        await self._client.send("vibium:tap", self._command_params({"timeout": timeout}))

    async def scroll_into_view(self, timeout: Optional[int] = None) -> None:
        await self._client.send("vibium:scrollIntoView", self._command_params({"timeout": timeout}))

    async def dispatch_event(
        self,
        event_type: str,
        event_init: Optional[Dict[str, Any]] = None,
        timeout: Optional[int] = None,
    ) -> None:
        await self._client.send("vibium:dispatchEvent", self._command_params({
            "eventType": event_type,
            "eventInit": event_init,
            "timeout": timeout,
        }))

    async def set_files(self, files: List[str], timeout: Optional[int] = None) -> None:
        await self._client.send("vibium:el.setFiles", self._command_params({
            "files": files,
            "timeout": timeout,
        }))

    # --- State ---

    async def text(self) -> str:
        result = await self._client.send("vibium:el.text", self._command_params())
        return result["text"]

    async def inner_text(self) -> str:
        result = await self._client.send("vibium:el.innerText", self._command_params())
        return result["text"]

    async def html(self) -> str:
        result = await self._client.send("vibium:el.html", self._command_params())
        return result["html"]

    async def value(self) -> str:
        result = await self._client.send("vibium:el.value", self._command_params())
        return result["value"]

    async def attr(self, name: str) -> Optional[str]:
        result = await self._client.send("vibium:el.attr", self._command_params({"name": name}))
        return result["value"]

    async def get_attribute(self, name: str) -> Optional[str]:
        """Alias for attr()."""
        return await self.attr(name)

    async def bounds(self) -> BoundingBox:
        result = await self._client.send("vibium:el.bounds", self._command_params())
        return BoundingBox(x=result["x"], y=result["y"], width=result["width"], height=result["height"])

    async def bounding_box(self) -> BoundingBox:
        """Alias for bounds()."""
        return await self.bounds()

    async def is_visible(self) -> bool:
        result = await self._client.send("vibium:el.isVisible", self._command_params())
        return result["visible"]

    async def is_hidden(self) -> bool:
        result = await self._client.send("vibium:el.isHidden", self._command_params())
        return result["hidden"]

    async def is_enabled(self) -> bool:
        result = await self._client.send("vibium:el.isEnabled", self._command_params())
        return result["enabled"]

    async def is_checked(self) -> bool:
        result = await self._client.send("vibium:el.isChecked", self._command_params())
        return result["checked"]

    async def is_editable(self) -> bool:
        result = await self._client.send("vibium:el.isEditable", self._command_params())
        return result["editable"]

    async def role(self) -> str:
        result = await self._client.send("vibium:el.role", self._command_params())
        return result["role"]

    async def label(self) -> str:
        result = await self._client.send("vibium:el.label", self._command_params())
        return result["label"]

    async def eval(self, fn: str) -> Any:
        result = await self._client.send("vibium:el.eval", self._command_params({"fn": fn}))
        return result["value"]

    async def screenshot(self) -> bytes:
        result = await self._client.send("vibium:el.screenshot", self._command_params())
        return base64.b64decode(result["data"])

    async def wait_until(self, state: Optional[str] = None, timeout: Optional[int] = None) -> None:
        """Wait until the element reaches a state: visible, hidden, attached, or detached."""
        await self._client.send("vibium:el.waitFor", self._command_params({
            "state": state,
            "timeout": timeout,
        }))

    # --- Finding (scoped) ---

    async def find(
        self,
        selector: Optional[str] = None,
        /,
        *,
        role: Optional[str] = None,
        text: Optional[str] = None,
        label: Optional[str] = None,
        placeholder: Optional[str] = None,
        alt: Optional[str] = None,
        title: Optional[str] = None,
        testid: Optional[str] = None,
        xpath: Optional[str] = None,
        near: Optional[str] = None,
        timeout: Optional[int] = None,
    ) -> Element:
        params: Dict[str, Any] = {
            "context": self._context,
            "scope": self._selector,
            "timeout": timeout,
        }
        if selector is not None:
            params["selector"] = selector
        else:
            for key, val in [("role", role), ("text", text), ("label", label),
                             ("placeholder", placeholder), ("alt", alt), ("title", title),
                             ("testid", testid), ("xpath", xpath), ("near", near)]:
                if val is not None:
                    params[key] = val

        result = await self._client.send("vibium:find", params)
        info = ElementInfo(
            tag=result["tag"],
            text=result["text"],
            box=BoundingBox(**result["box"]),
        )
        child_selector = selector or ""
        child_params = {"selector": selector} if selector else {
            k: v for k, v in params.items() if k not in ("context", "scope", "timeout")
        }
        return Element(self._client, self._context, child_selector, info, None, child_params)

    async def find_all(
        self,
        selector: Optional[str] = None,
        /,
        *,
        role: Optional[str] = None,
        text: Optional[str] = None,
        label: Optional[str] = None,
        placeholder: Optional[str] = None,
        alt: Optional[str] = None,
        title: Optional[str] = None,
        testid: Optional[str] = None,
        xpath: Optional[str] = None,
        near: Optional[str] = None,
        timeout: Optional[int] = None,
    ) -> ElementList:
        from .element_list import ElementList

        params: Dict[str, Any] = {
            "context": self._context,
            "scope": self._selector,
            "timeout": timeout,
        }
        if selector is not None:
            params["selector"] = selector
        else:
            for key, val in [("role", role), ("text", text), ("label", label),
                             ("placeholder", placeholder), ("alt", alt), ("title", title),
                             ("testid", testid), ("xpath", xpath), ("near", near)]:
                if val is not None:
                    params[key] = val

        result = await self._client.send("vibium:findAll", params)
        sel_str = selector or ""
        sel_params = {"selector": selector} if selector else {
            k: v for k, v in params.items() if k not in ("context", "scope", "timeout")
        }
        elements = []
        for el in result["elements"]:
            info = ElementInfo(tag=el["tag"], text=el["text"], box=BoundingBox(**el["box"]))
            elements.append(Element(self._client, self._context, sel_str, info, el.get("index"), sel_params))
        return ElementList(self._client, self._context, selector or params, elements)
