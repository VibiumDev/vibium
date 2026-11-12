"""Sync Element wrapper."""

from __future__ import annotations

from .._locator import ImageInput
from typing import Any, Dict, List, Optional, TYPE_CHECKING

from .._types import BoundingBox

if TYPE_CHECKING:
    from .._sync_base import _EventLoopThread
    from ..async_api.element import Element as AsyncElement


class Element:
    """Synchronous wrapper for async Element."""

    def __init__(self, async_element: AsyncElement, loop_thread: _EventLoopThread) -> None:
        self._async = async_element
        self._loop = loop_thread

    @property
    def info(self):
        """Snapshot of tag, text, and box captured when the element was found."""
        return self._async.info

    def __repr__(self) -> str:
        text = self.info.text
        if len(text) > 50:
            text = text[:50] + "..."
        return f"Element(tag='{self.info.tag}', text='{text}')"

    # --- Interaction ---

    def click(self, timeout: Optional[int] = None) -> None:
        """Click the element. Waits for the element to be visible, stable, receive events, and enabled."""
        self._loop.run(self._async.click(timeout))

    def dblclick(self, timeout: Optional[int] = None) -> None:
        """Double-click the element."""
        self._loop.run(self._async.dblclick(timeout))

    def fill(self, value: str, timeout: Optional[int] = None) -> None:
        """Fill the element with text (clears existing content first). For inputs and textareas."""
        self._loop.run(self._async.fill(value, timeout))

    def type(self, text: str, timeout: Optional[int] = None) -> None:
        """Type text into the element (appends to existing content). Waits for the element to be visible, stable, receive events, enabled, and editable."""
        self._loop.run(self._async.type(text, timeout))

    def press(self, key: str, timeout: Optional[int] = None) -> None:
        """Press a key while the element is focused. Supports key names ("Enter", "Tab") and combos ("Control+a")."""
        self._loop.run(self._async.press(key, timeout))

    def clear(self, timeout: Optional[int] = None) -> None:
        """Clear the element's content (select all + delete)."""
        self._loop.run(self._async.clear(timeout))

    def set(self, value: bool = True, timeout: Optional[int] = None) -> None:
        """Check a checkbox (no-op if already checked)."""
        self._loop.run(self._async.set(value, timeout))

    def unset(self, timeout: Optional[int] = None) -> None:
        """Uncheck a checkbox (no-op if already unchecked)."""
        self._loop.run(self._async.unset(timeout))

    def select_option(self, value: str, timeout: Optional[int] = None) -> None:
        """Select an option in a <select> element by value."""
        self._loop.run(self._async.select_option(value, timeout))

    def hover(self, timeout: Optional[int] = None) -> None:
        """Hover over the element (move mouse to center, no click)."""
        self._loop.run(self._async.hover(timeout))

    def focus(self, timeout: Optional[int] = None) -> None:
        """Focus the element."""
        self._loop.run(self._async.focus(timeout))

    def highlight(self, timeout: Optional[int] = None) -> None:
        """Briefly outline the element so a human watching can see it."""
        self._loop.run(self._async.highlight(timeout))

    def drag_to(self, target: Element, timeout: Optional[int] = None) -> None:
        """Drag this element to a target element."""
        self._loop.run(self._async.drag_to(target._async, timeout))

    def tap(self, timeout: Optional[int] = None) -> None:
        """Tap the element (touch action)."""
        self._loop.run(self._async.tap(timeout))

    def scroll_into_view(self, timeout: Optional[int] = None) -> None:
        """Scroll the element into view."""
        self._loop.run(self._async.scroll_into_view(timeout))

    def dispatch_event(
        self,
        event_type: str,
        event_init: Optional[Dict[str, Any]] = None,
        timeout: Optional[int] = None,
    ) -> None:
        """Dispatch a DOM event on the element."""
        self._loop.run(self._async.dispatch_event(event_type, event_init, timeout))

    def set_files(self, files: List[str], timeout: Optional[int] = None) -> None:
        """Set files on an <input type="file"> element."""
        self._loop.run(self._async.set_files(files, timeout))

    # --- State ---

    def text(self) -> str:
        """Get the element's textContent (trimmed)."""
        return self._loop.run(self._async.text())

    def inner_text(self) -> str:
        """Get the element's innerText (rendered text only)."""
        return self._loop.run(self._async.inner_text())

    def html(self) -> str:
        """Get the element's innerHTML."""
        return self._loop.run(self._async.html())

    def value(self) -> str:
        """Get the element's value (for inputs, textareas, selects)."""
        return self._loop.run(self._async.value())

    def attr(self, name: str) -> Optional[str]:
        """Get an attribute value. Short name for get_attribute()."""
        return self._loop.run(self._async.attr(name))

    def get_attribute(self, name: str) -> Optional[str]:
        return self.attr(name)

    def bounds(self) -> BoundingBox:
        """Get the element's bounding box. Short name for bounding_box()."""
        return self._loop.run(self._async.bounds())

    def bounding_box(self) -> BoundingBox:
        return self.bounds()

    def is_visible(self) -> bool:
        """Check if the element is visible (has dimensions, not display:none/visibility:hidden/opacity:0)."""
        return self._loop.run(self._async.is_visible())

    def is_hidden(self) -> bool:
        """Check if the element is hidden (inverse of is_visible())."""
        return self._loop.run(self._async.is_hidden())

    def is_enabled(self) -> bool:
        """Check if the element is enabled (not disabled)."""
        return self._loop.run(self._async.is_enabled())

    def is_set(self) -> bool:
        """Check if the element is checked (for checkboxes/radios)."""
        return self._loop.run(self._async.is_set())

    def is_editable(self) -> bool:
        """Check if the element is editable (not disabled and not readonly)."""
        return self._loop.run(self._async.is_editable())

    def role(self) -> str:
        """Get the element's computed ARIA role."""
        return self._loop.run(self._async.role())

    def label(self) -> str:
        """Get the element's accessible name (label)."""
        return self._loop.run(self._async.label())

    def screenshot(self) -> bytes:
        """Take a screenshot of just this element. Returns PNG bytes."""
        return self._loop.run(self._async.screenshot())

    def wait_until(self, state: Optional[str] = None, timeout: Optional[int] = None) -> None:
        """Wait until the element reaches a state: visible, hidden, attached, or detached."""
        self._loop.run(self._async.wait_until(state, timeout))

    # --- Finding (scoped) ---

    def find(
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
        """Find a child element, scoped to this element; see the async page.find for the forms."""
        async_el = self._loop.run(self._async.find(
            selector, description=description, image=image, css=css, tag=tag, name=name, role=role, text=text, label=label,
            placeholder=placeholder, alt=alt, title=title, testid=testid, xpath=xpath, near=near,
            strategy=strategy, details=details, provider=provider, model=model,
            ai_base_url=ai_base_url, reasoning_effort=reasoning_effort, timeout=timeout,
        ))
        return Element(async_el, self._loop)

    def find_all(
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
        """Find all child elements, scoped to this element."""
        async_els = self._loop.run(self._async.find_all(
            selector, description=description, image=image, css=css, tag=tag, name=name, role=role, text=text, label=label,
            placeholder=placeholder, alt=alt, title=title, testid=testid, xpath=xpath, near=near,
            strategy=strategy, details=details, provider=provider, model=model,
            ai_base_url=ai_base_url, reasoning_effort=reasoning_effort, timeout=timeout,
        ))
        return [Element(e, self._loop) for e in async_els]
