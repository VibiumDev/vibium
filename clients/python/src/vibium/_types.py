"""Shared types for the Vibium Python client."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any, Dict, List, Optional, Union


@dataclass
class BoundingBox:
    """Bounding box of an element."""

    x: float
    y: float
    width: float
    height: float


@dataclass
class ElementInfo:
    """Information about an element returned from find commands."""

    tag: str
    text: str
    box: BoundingBox


# --- TypedDicts as plain dicts (for Python 3.9 compat) ---

# Cookie: {name, value, domain, path, size, httpOnly, secure, sameSite, expiry?}
Cookie = Dict[str, Any]

# SetCookieParam: {name, value, domain?, url?, path?, httpOnly?, secure?, sameSite?, expiry?}
SetCookieParam = Dict[str, Any]

# OriginState: {origin, localStorage, sessionStorage}
OriginState = Dict[str, Any]

# StorageState: {cookies: list[Cookie], origins: list[OriginState]}
StorageState = Dict[str, Any]

# A11yNode: {role, name?, value?, description?, children?, ...}
A11yNode = Dict[str, Any]

# ViewportSize: {width, height}
ViewportSize = Dict[str, Union[int, float]]

# WindowInfo: {state, width, height, x, y}
WindowInfo = Dict[str, Any]
