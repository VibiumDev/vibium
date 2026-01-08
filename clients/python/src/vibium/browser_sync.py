"""Synchronous browser launcher using a background event loop."""

import asyncio
import threading
from typing import Literal, Optional

from .browser import browser
from .element import Element, ElementInfo
from .vibe import Vibe


class _EventLoopThread:
    """Manages a background thread running an asyncio event loop."""

    def __init__(self):
        self._loop: Optional[asyncio.AbstractEventLoop] = None
        self._thread: Optional[threading.Thread] = None

    def start(self) -> asyncio.AbstractEventLoop:
        """Start the background event loop thread."""
        if self._loop is not None:
            return self._loop

        self._loop = asyncio.new_event_loop()
        self._thread = threading.Thread(target=self._run_loop, daemon=True)
        self._thread.start()
        return self._loop

    def _run_loop(self) -> None:
        """Run the event loop in the background thread."""
        asyncio.set_event_loop(self._loop)
        self._loop.run_forever()

    def run(self, coro) -> any:
        """Run a coroutine in the background loop and wait for result."""
        if self._loop is None:
            raise RuntimeError("Event loop not started")
        future = asyncio.run_coroutine_threadsafe(coro, self._loop)
        return future.result()

    def stop(self) -> None:
        """Stop the event loop and thread."""
        if self._loop:
            self._loop.call_soon_threadsafe(self._loop.stop)
        if self._thread:
            self._thread.join(timeout=5)
        self._loop = None
        self._thread = None


class ElementSync:
    """Synchronous wrapper for Element."""

    def __init__(self, element: Element, loop_thread: _EventLoopThread):
        self._element = element
        self._loop_thread = loop_thread
        self.info = element.info

    def click(self, timeout: Optional[int] = None) -> None:
        """Click the element."""
        self._loop_thread.run(self._element.click(timeout))

    def type(self, text: str, timeout: Optional[int] = None) -> None:
        """Type text into the element."""
        self._loop_thread.run(self._element.type(text, timeout))

    def text(self) -> str:
        """Get the text content of the element."""
        return self._loop_thread.run(self._element.text())

    def get_attribute(self, name: str) -> Optional[str]:
        """Get an attribute value from the element."""
        return self._loop_thread.run(self._element.get_attribute(name))


class VibeSync:
    """Synchronous wrapper for Vibe."""

    def __init__(self, vibe: Vibe, loop_thread: _EventLoopThread):
        self._vibe = vibe
        self._loop_thread = loop_thread

    def go(self, url: str) -> None:
        """Navigate to a URL."""
        self._loop_thread.run(self._vibe.go(url))

    def screenshot(self) -> bytes:
        """Capture a screenshot of the viewport."""
        return self._loop_thread.run(self._vibe.screenshot())

    def find(self, selector: str, timeout: Optional[int] = None) -> ElementSync:
        """Find an element by CSS selector."""
        element = self._loop_thread.run(self._vibe.find(selector, timeout))
        return ElementSync(element, self._loop_thread)

    def start_recording(
        self,
        fps: int = 10,
        format: Literal["mp4", "webm"] = "mp4",
        output_path: Optional[str] = None,
    ) -> None:
        """Start recording the browser session as a video.

        Requires FFmpeg to be installed on the system.

        Args:
            fps: Frames per second. Default: 10
            format: Output format ('mp4' or 'webm'). Default: 'mp4'
            output_path: Output file path. If not provided, uses temp directory.
        """
        self._loop_thread.run(
            self._vibe.start_recording(fps=fps, format=format, output_path=output_path)
        )

    def stop_recording(self) -> str:
        """Stop recording and save the video file.

        Returns:
            Path to the saved video file.
        """
        return self._loop_thread.run(self._vibe.stop_recording())

    def quit(self) -> None:
        """Close the browser and clean up resources."""
        self._loop_thread.run(self._vibe.quit())
        self._loop_thread.stop()


class browser_sync:
    """Synchronous browser launcher.

    Usage:
        vibe = browser_sync.launch()
        vibe.go("https://example.com")
        vibe.quit()
    """

    @staticmethod
    def launch(
        headless: bool = False,
        port: Optional[int] = None,
        executable_path: Optional[str] = None,
    ) -> VibeSync:
        """Launch a new browser instance.

        Args:
            headless: Run browser in headless mode (default: visible).
            port: WebSocket port (default: auto-assigned).
            executable_path: Path to clicker binary (default: auto-detect).

        Returns:
            A VibeSync instance for browser automation.
        """
        loop_thread = _EventLoopThread()
        loop_thread.start()

        vibe = loop_thread.run(
            browser.launch(
                headless=headless,
                port=port,
                executable_path=executable_path,
            )
        )

        return VibeSync(vibe, loop_thread)
