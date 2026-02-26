"""Sync ElementList wrapper."""

from __future__ import annotations

from typing import Iterator, List, Optional, TYPE_CHECKING

if TYPE_CHECKING:
    from .._sync_base import _EventLoopThread
    from ..async_api.element_list import ElementList as AsyncElementList
    from .element import Element


class ElementList:
    """Synchronous wrapper for async ElementList."""

    def __init__(self, async_list: AsyncElementList, loop_thread: _EventLoopThread) -> None:
        self._async = async_list
        self._loop = loop_thread

    def count(self) -> int:
        return self._async.count()

    def first(self) -> Element:
        from .element import Element
        return Element(self._async.first(), self._loop)

    def last(self) -> Element:
        from .element import Element
        return Element(self._async.last(), self._loop)

    def nth(self, index: int) -> Element:
        from .element import Element
        return Element(self._async.nth(index), self._loop)

    def filter(self, has_text: Optional[str] = None, has: Optional[str] = None) -> ElementList:
        async_filtered = self._loop.run(self._async.filter(has_text=has_text, has=has))
        return ElementList(async_filtered, self._loop)

    def to_array(self) -> List[Element]:
        from .element import Element
        return [Element(el, self._loop) for el in self._async.to_array()]

    def __iter__(self) -> Iterator[Element]:
        from .element import Element
        return iter(Element(el, self._loop) for el in self._async)

    def __len__(self) -> int:
        return len(self._async)
