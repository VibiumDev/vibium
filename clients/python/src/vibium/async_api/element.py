"""Async Element class for interacting with page elements."""

from __future__ import annotations

import base64
from typing import Any, Dict, List, Optional, TYPE_CHECKING

from .._locator import ImageInput, handle_selector, info_from, locator_params, resolved_handle, reusable_params
from .._types import BoundingBox, ElementInfo

if TYPE_CHECKING:
    from ..client import BiDiClient


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

    def __repr__(self) -> str:
        text = self.info.text
        if len(text) > 50:
            text = text[:50] + "..."
        return f"Element(tag='{self.info.tag}', text='{text}')"

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
        """Click the element. Waits for the element to be visible, stable, receive events, and enabled."""
        await self._client.send("vibium:element.click", self._command_params({"timeout": timeout}))

    async def dblclick(self, timeout: Optional[int] = None) -> None:
        """Double-click the element."""
        await self._client.send("vibium:element.dblclick", self._command_params({"timeout": timeout}))

    async def fill(self, value: str, timeout: Optional[int] = None) -> None:
        """Fill the element with text (clears existing content first). For inputs and textareas."""
        await self._client.send("vibium:element.fill", self._command_params({"value": value, "timeout": timeout}))

    async def type(self, text: str, timeout: Optional[int] = None) -> None:
        # "value" is the payload key; "text" would collide with a text filter.
        """Type text into the element (appends to existing content). Waits for the element to be visible, stable, receive events, enabled, and editable."""
        await self._client.send("vibium:element.type", self._command_params({"value": text, "timeout": timeout}))

    async def press(self, key: str, timeout: Optional[int] = None) -> None:
        """Press a key while the element is focused. Supports key names ("Enter", "Tab") and combos ("Control+a")."""
        await self._client.send("vibium:element.press", self._command_params({"key": key, "timeout": timeout}))

    async def clear(self, timeout: Optional[int] = None) -> None:
        """Clear the element's content (select all + delete)."""
        await self._client.send("vibium:element.clear", self._command_params({"timeout": timeout}))

    async def set(self, value: bool = True, timeout: Optional[int] = None) -> None:
        """Check a checkbox (no-op if already checked)."""
        await self._client.send("vibium:element.set", self._command_params({"value": value, "timeout": timeout}))

    async def unset(self, timeout: Optional[int] = None) -> None:
        """Uncheck a checkbox (no-op if already unchecked)."""
        await self._client.send("vibium:element.unset", self._command_params({"timeout": timeout}))

    async def select_option(self, value: str, timeout: Optional[int] = None) -> None:
        """Select an option in a <select> element by value."""
        await self._client.send("vibium:element.selectOption", self._command_params({"value": value, "timeout": timeout}))

    async def hover(self, timeout: Optional[int] = None) -> None:
        """Hover over the element (move mouse to center, no click)."""
        await self._client.send("vibium:element.hover", self._command_params({"timeout": timeout}))

    async def focus(self, timeout: Optional[int] = None) -> None:
        """Focus the element."""
        await self._client.send("vibium:element.focus", self._command_params({"timeout": timeout}))

    async def highlight(self, timeout: Optional[int] = None) -> None:
        """Briefly outline the element so a human watching can see it."""
        await self._client.send("vibium:element.highlight", self._command_params({"timeout": timeout}))

    async def drag_to(self, target: Element, timeout: Optional[int] = None) -> None:
        """Drag this element to a target element."""
        await self._client.send("vibium:element.dragTo", self._command_params({
            "target": target._to_params(),
            "timeout": timeout,
        }))

    async def tap(self, timeout: Optional[int] = None) -> None:
        """Tap the element (touch action)."""
        await self._client.send("vibium:element.tap", self._command_params({"timeout": timeout}))

    async def scroll_into_view(self, timeout: Optional[int] = None) -> None:
        """Scroll the element into view."""
        await self._client.send("vibium:element.scrollIntoView", self._command_params({"timeout": timeout}))

    async def dispatch_event(
        self,
        event_type: str,
        event_init: Optional[Dict[str, Any]] = None,
        timeout: Optional[int] = None,
    ) -> None:
        """Dispatch a DOM event on the element."""
        await self._client.send("vibium:element.dispatchEvent", self._command_params({
            "eventType": event_type,
            "eventInit": event_init,
            "timeout": timeout,
        }))

    async def set_files(self, files: List[str], timeout: Optional[int] = None) -> None:
        """Set files on an <input type="file"> element."""
        await self._client.send("vibium:element.setFiles", self._command_params({
            "files": files,
            "timeout": timeout,
        }))

    # --- State ---

    async def text(self) -> str:
        """Get the element's textContent (trimmed)."""
        result = await self._client.send("vibium:element.text", self._command_params())
        return result["text"]

    async def inner_text(self) -> str:
        """Get the element's innerText (rendered text only)."""
        result = await self._client.send("vibium:element.innerText", self._command_params())
        return result["text"]

    async def html(self) -> str:
        """Get the element's innerHTML."""
        result = await self._client.send("vibium:element.html", self._command_params())
        return result["html"]

    async def value(self) -> str:
        """Get the element's value (for inputs, textareas, selects)."""
        result = await self._client.send("vibium:element.value", self._command_params())
        return result["value"]

    async def attr(self, name: str) -> Optional[str]:
        """Get an attribute value. Short name for get_attribute()."""
        result = await self._client.send("vibium:element.attr", self._command_params({"name": name}))
        return result["value"]

    async def get_attribute(self, name: str) -> Optional[str]:
        """Alias for attr()."""
        return await self.attr(name)

    async def bounds(self) -> BoundingBox:
        """Get the element's bounding box. Short name for bounding_box()."""
        result = await self._client.send("vibium:element.bounds", self._command_params())
        return BoundingBox(x=result["x"], y=result["y"], width=result["width"], height=result["height"])

    async def bounding_box(self) -> BoundingBox:
        """Alias for bounds()."""
        return await self.bounds()

    async def is_visible(self) -> bool:
        """Check if the element is visible (has dimensions, not display:none/visibility:hidden/opacity:0)."""
        result = await self._client.send("vibium:element.isVisible", self._command_params())
        return result["visible"]

    async def is_hidden(self) -> bool:
        """Check if the element is hidden (inverse of is_visible())."""
        result = await self._client.send("vibium:element.isHidden", self._command_params())
        return result["hidden"]

    async def is_enabled(self) -> bool:
        """Check if the element is enabled (not disabled)."""
        result = await self._client.send("vibium:element.isEnabled", self._command_params())
        return result["enabled"]

    async def is_set(self) -> bool:
        """Check if the element is checked (for checkboxes/radios)."""
        result = await self._client.send("vibium:element.isSet", self._command_params())
        return result["checked"]

    async def is_editable(self) -> bool:
        """Check if the element is editable (not disabled and not readonly)."""
        result = await self._client.send("vibium:element.isEditable", self._command_params())
        return result["editable"]

    async def role(self) -> str:
        """Get the element's computed ARIA role."""
        result = await self._client.send("vibium:element.role", self._command_params())
        return result["role"]

    async def label(self) -> str:
        """Get the element's accessible name (label)."""
        result = await self._client.send("vibium:element.label", self._command_params())
        return result["label"]

    async def screenshot(self) -> bytes:
        """Take a screenshot of just this element. Returns PNG bytes."""
        result = await self._client.send("vibium:element.screenshot", self._command_params())
        return base64.b64decode(result["data"])

    async def wait_until(self, state: Optional[str] = None, timeout: Optional[int] = None) -> None:
        """Wait until the element reaches a state: visible, hidden, attached, or detached."""
        await self._client.send("vibium:element.waitFor", self._command_params({
            "state": state,
            "timeout": timeout,
        }))

    # --- Finding (scoped) ---

    async def find(
        self,
        selector: Optional[str] = None,
        /,
        *,
        description: Optional[str] = None,
        image: Optional[ImageInput] = None,
        css: Optional[str] = None,
        tag: Optional[str] = None,
        name: Optional[str] = None,
        role: Optional[str] = None,
        text: Optional[str] = None,
        label: Optional[str] = None,
        placeholder: Optional[str] = None,
        alt: Optional[str] = None,
        title: Optional[str] = None,
        testid: Optional[str] = None,
        xpath: Optional[str] = None,
        near: Optional[str] = None,
        strategy: Optional[str] = None,
        details: Optional[bool] = None,
        provider: Optional[str] = None,
        model: Optional[str] = None,
        ai_base_url: Optional[str] = None,
        reasoning_effort: Optional[str] = None,
        timeout: Optional[int] = None,
    ) -> Element:
        """Find a child element; the forms are those of page.find, scoped to this element."""
        params = locator_params(selector, description=description, image=image, css=css, tag=tag, name=name, role=role, text=text, label=label,
            placeholder=placeholder, alt=alt, title=title, testid=testid, xpath=xpath, near=near,
            strategy=strategy, details=details, provider=provider, model=model,
            ai_base_url=ai_base_url, reasoning_effort=reasoning_effort, timeout=timeout)
        params["context"] = self._context
        params["scope"] = self._selector
        result = await self._client.send("vibium:element.find", params)
        if resolved_handle(params):
            return Element(self._client, self._context, handle_selector(result), info_from(result))
        # The child keeps the scope so re-resolution stays inside the parent (#779).
        return Element(self._client, self._context, params.get("selector", ""), info_from(result), None, reusable_params(params))

    async def find_all(
        self,
        selector: Optional[str] = None,
        /,
        *,
        description: Optional[str] = None,
        image: Optional[ImageInput] = None,
        css: Optional[str] = None,
        tag: Optional[str] = None,
        name: Optional[str] = None,
        role: Optional[str] = None,
        text: Optional[str] = None,
        label: Optional[str] = None,
        placeholder: Optional[str] = None,
        alt: Optional[str] = None,
        title: Optional[str] = None,
        testid: Optional[str] = None,
        xpath: Optional[str] = None,
        near: Optional[str] = None,
        strategy: Optional[str] = None,
        details: Optional[bool] = None,
        provider: Optional[str] = None,
        model: Optional[str] = None,
        ai_base_url: Optional[str] = None,
        reasoning_effort: Optional[str] = None,
        timeout: Optional[int] = None,
    ) -> List[Element]:
        """Find all child elements; the forms are those of page.find_all, scoped to this element."""
        params = locator_params(selector, description=description, image=image, css=css, tag=tag, name=name, role=role, text=text, label=label,
            placeholder=placeholder, alt=alt, title=title, testid=testid, xpath=xpath, near=near,
            strategy=strategy, details=details, provider=provider, model=model,
            ai_base_url=ai_base_url, reasoning_effort=reasoning_effort, timeout=timeout)
        params["context"] = self._context
        params["scope"] = self._selector
        result = await self._client.send("vibium:element.findAll", params)
        resolved = resolved_handle(params)
        reuse = reusable_params(params)
        elements = []
        for el in result["elements"]:
            if resolved:
                elements.append(Element(self._client, self._context, handle_selector(el), info_from(el)))
            else:
                elements.append(Element(self._client, self._context, params.get("selector", ""), info_from(el), el.get("index"), reuse))
        return elements
