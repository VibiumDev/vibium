"""Async ElementList class."""

from __future__ import annotations

from typing import Any, Dict, Iterator, List, Optional, Union, TYPE_CHECKING

from .._types import BoundingBox, ElementInfo

if TYPE_CHECKING:
    from ..client import BiDiClient
    from .element import Element


class ElementList:
    """A collection of elements returned by findAll."""

    def __init__(
        self,
        client: BiDiClient,
        context: str,
        selector: Any,
        elements: List[Element],
    ) -> None:
        self._client = client
        self._context = context
        self._selector = selector
        self._elements = elements

    def count(self) -> int:
        return len(self._elements)

    def first(self) -> Element:
        return self.nth(0)

    def last(self) -> Element:
        return self.nth(len(self._elements) - 1)

    def nth(self, index: int) -> Element:
        if index < 0 or index >= len(self._elements):
            raise IndexError(f"Index {index} out of bounds (0..{len(self._elements) - 1})")
        return self._elements[index]

    async def filter(
        self,
        has_text: Optional[str] = None,
        has: Optional[str] = None,
    ) -> ElementList:
        """Filter elements by re-querying with filter params."""
        from .element import Element

        params: Dict[str, Any] = {
            "context": self._context,
            "timeout": 5000,
        }
        if isinstance(self._selector, str):
            params["selector"] = self._selector
        elif isinstance(self._selector, dict):
            params.update(self._selector)

        if has_text:
            params["hasText"] = has_text
        if has:
            params["has"] = has

        result = await self._client.send("vibium:findAll", params)

        sel_str = self._selector if isinstance(self._selector, str) else ""
        sel_params = ({"selector": self._selector} if isinstance(self._selector, str)
                      else dict(self._selector) if isinstance(self._selector, dict)
                      else {})
        elements = []
        for el in result["elements"]:
            info = ElementInfo(tag=el["tag"], text=el["text"], box=BoundingBox(**el["box"]))
            elements.append(Element(self._client, self._context, sel_str, info, el.get("index"), sel_params))
        return ElementList(self._client, self._context, self._selector, elements)

    def to_array(self) -> List[Element]:
        return list(self._elements)

    def __iter__(self) -> Iterator[Element]:
        return iter(self._elements)

    def __len__(self) -> int:
        return len(self._elements)
