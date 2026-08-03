"""Sync Element wrapper."""

from __future__ import annotations

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
        self.info = async_element.info

    def __repr__(self) -> str:
        text = self.info.text
        if len(text) > 50:
            text = text[:50] + "..."
        return f"Element(tag='{self.info.tag}', text='{text}')"

    # --- Interaction ---

    def click(self, timeout: Optional[int] = None) -> None:
        self._loop.run(self._async.click(timeout))

    def dblclick(self, timeout: Optional[int] = None) -> None:
        self._loop.run(self._async.dblclick(timeout))

    def fill(self, value: str, timeout: Optional[int] = None) -> None:
        self._loop.run(self._async.fill(value, timeout))

    def type(self, text: str, timeout: Optional[int] = None) -> None:
        self._loop.run(self._async.type(text, timeout))

    def press(self, key: str, timeout: Optional[int] = None) -> None:
        self._loop.run(self._async.press(key, timeout))

    def clear(self, timeout: Optional[int] = None) -> None:
        self._loop.run(self._async.clear(timeout))

    def check(self, timeout: Optional[int] = None) -> None:
        self._loop.run(self._async.check(timeout))

    def uncheck(self, timeout: Optional[int] = None) -> None:
        self._loop.run(self._async.uncheck(timeout))

    def select_option(self, value: str, timeout: Optional[int] = None) -> None:
        self._loop.run(self._async.select_option(value, timeout))

    def hover(self, timeout: Optional[int] = None) -> None:
        self._loop.run(self._async.hover(timeout))

    def focus(self, timeout: Optional[int] = None) -> None:
        self._loop.run(self._async.focus(timeout))

    def highlight(self, timeout: Optional[int] = None) -> None:
        """Briefly outline the element so a human watching can see it."""
        self._loop.run(self._async.highlight(timeout))

    def drag_to(self, target: Element, timeout: Optional[int] = None) -> None:
        self._loop.run(self._async.drag_to(target._async, timeout))

    def tap(self, timeout: Optional[int] = None) -> None:
        self._loop.run(self._async.tap(timeout))

    def scroll_into_view(self, timeout: Optional[int] = None) -> None:
        self._loop.run(self._async.scroll_into_view(timeout))

    def dispatch_event(
        self,
        event_type: str,
        event_init: Optional[Dict[str, Any]] = None,
        timeout: Optional[int] = None,
    ) -> None:
        self._loop.run(self._async.dispatch_event(event_type, event_init, timeout))

    def set_files(self, files: List[str], timeout: Optional[int] = None) -> None:
        self._loop.run(self._async.set_files(files, timeout))

    # --- State ---

    def text(self) -> str:
        return self._loop.run(self._async.text())

    def inner_text(self) -> str:
        return self._loop.run(self._async.inner_text())

    def html(self) -> str:
        return self._loop.run(self._async.html())

    def value(self) -> str:
        return self._loop.run(self._async.value())

    def attr(self, name: str) -> Optional[str]:
        return self._loop.run(self._async.attr(name))

    def get_attribute(self, name: str) -> Optional[str]:
        return self.attr(name)

    def bounds(self) -> BoundingBox:
        return self._loop.run(self._async.bounds())

    def bounding_box(self) -> BoundingBox:
        return self.bounds()

    def is_visible(self) -> bool:
        return self._loop.run(self._async.is_visible())

    def is_hidden(self) -> bool:
        return self._loop.run(self._async.is_hidden())

    def is_enabled(self) -> bool:
        return self._loop.run(self._async.is_enabled())

    def is_checked(self) -> bool:
        return self._loop.run(self._async.is_checked())

    def is_editable(self) -> bool:
        return self._loop.run(self._async.is_editable())

    def role(self) -> str:
        return self._loop.run(self._async.role())

    def label(self) -> str:
        return self._loop.run(self._async.label())

    def screenshot(self) -> bytes:
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
        async_el = self._loop.run(self._async.find(
            selector, role=role, text=text, label=label, placeholder=placeholder,
            alt=alt, title=title, testid=testid, xpath=xpath, near=near, timeout=timeout,
        ))
        return Element(async_el, self._loop)

    def find_all(
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
    ) -> List[Element]:
        async_elements = self._loop.run(self._async.find_all(
            selector, role=role, text=text, label=label, placeholder=placeholder,
            alt=alt, title=title, testid=testid, xpath=xpath, near=near, timeout=timeout,
        ))
        return [Element(el, self._loop) for el in async_elements]
