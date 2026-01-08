"""Vibium - Browser automation for AI agents and humans."""

from .browser import browser
from .browser_sync import browser_sync
from .vibe import RecordingOptions

__version__ = "0.1.0"
__all__ = ["browser", "browser_sync", "RecordingOptions"]
