"""Structured error types for the vibium Python client."""

from __future__ import annotations

import builtins


class VibiumError(Exception):
    """Base class for all vibium errors."""
    pass


class BiDiError(VibiumError):
    """Raised when a BiDi command fails."""

    def __init__(self, error: str, message: str):
        self.error = error
        self.message = message
        super().__init__(f"{error}: {message}")


class SpeechError(VibiumError):
    """Raised when say or listen fails. ``code`` names the failure, such as
    ``no_speech`` or ``not_configured``."""

    def __init__(self, code: str, message: str):
        self.code = code
        self.message = message
        super().__init__(f"{code}: {message}")


class VibiumNotFoundError(VibiumError):
    """Raised when the vibium binary cannot be found."""
    pass


class TimeoutError(VibiumError, builtins.TimeoutError):
    """Raised when an operation times out.

    Subclasses builtins.TimeoutError so ``except TimeoutError`` still catches it.
    """

    def __init__(self, message: str, selector: str | None = None, timeout_ms: int | None = None):
        self.selector = selector
        self.timeout_ms = timeout_ms
        super().__init__(message)


class ConnectionError(VibiumError, builtins.ConnectionError):
    """Raised when the connection to the browser is lost.

    Subclasses builtins.ConnectionError so ``except ConnectionError`` still catches it.
    """

    def __init__(self, url: str, cause: BaseException | None = None):
        self.url = url
        self.cause = cause
        super().__init__(url)


class ElementNotFoundError(VibiumError):
    """Raised when an element cannot be found on the page."""

    def __init__(self, selector: str):
        self.selector = selector
        super().__init__(f"Element not found: {selector}")


class BrowserCrashedError(VibiumError):
    """Raised when the browser process exits unexpectedly."""

    def __init__(self, message: str, exit_code: int | None = None, output: str | None = None):
        self.exit_code = exit_code
        self.output = output
        super().__init__(message)
