"""Locator plumbing shared by page and element find."""

from __future__ import annotations

import base64
import os
from typing import Any, Dict, Optional, Union

from ._types import BoundingBox, ElementInfo

ImageInput = Union[str, bytes, bytearray, memoryview, "os.PathLike[str]"]

_REUSE_DROP = ("context", "timeout", "image", "details", "strategy",
               "provider", "model", "aiBaseURL", "reasoningEffort", "limit", "bare")


def encode_image(image: ImageInput) -> str:
    """Base64 for the wire: bytes as they are, a path read, a data: URL kept."""
    if isinstance(image, (bytes, bytearray, memoryview)):
        return base64.b64encode(bytes(image)).decode("ascii")
    path = os.fspath(image)
    if path.startswith("data:"):
        return path
    with open(path, "rb") as f:
        return base64.b64encode(f.read()).decode("ascii")


def locator_params(
    selector: Optional[str],
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
) -> Dict[str, Any]:
    """Wire params for a locator. A bare positional string is a description.

    css travels as "selector", the name element commands read, so one set of
    params serves the find and the actions that follow it."""
    params: Dict[str, Any] = {}
    if selector is not None:
        params["description"] = selector
        params["bare"] = True
    if description is not None:
        params["description"] = description
        params.pop("bare", None)
    if image is not None:
        params["image"] = encode_image(image)
    if css is not None:
        params["selector"] = css
    for key, val in [("tag", tag), ("name", name), ("role", role), ("text", text), ("label", label),
                     ("placeholder", placeholder), ("alt", alt), ("title", title),
                     ("testid", testid), ("xpath", xpath), ("near", near),
                     ("strategy", strategy), ("details", details), ("provider", provider),
                     ("model", model), ("aiBaseURL", ai_base_url),
                     ("reasoningEffort", reasoning_effort), ("timeout", timeout)]:
        if val is not None:
            params[key] = val
    return params


def resolved_handle(params: Dict[str, Any]) -> bool:
    """Whether later commands address the match by what find resolved rather
    than by running the locator again."""
    return any(k in params for k in ("description", "image", "tag", "name"))


def handle_selector(result: Dict[str, Any]) -> str:
    """The DOM selector of a match, or its @ref for a visual region."""
    return result.get("selector") or result.get("ref") or ""


def info_from(result: Dict[str, Any]) -> ElementInfo:
    return ElementInfo(
        tag=result["tag"],
        text=result["text"],
        box=BoundingBox(**result["box"]),
        ref=result.get("ref"),
        role=result.get("role"),
        name=result.get("name"),
        source=result.get("source"),
        selector=result.get("selector"),
        details=result.get("details"),
    )


def reusable_params(params: Dict[str, Any]) -> Dict[str, Any]:
    return {k: v for k, v in params.items() if k not in _REUSE_DROP}
