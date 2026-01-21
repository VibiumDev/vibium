"""Async browser launcher."""

from typing import Optional

from .client import BiDiClient
from .clicker import ClickerProcess
from .vibe import Vibe


class browser:
    """Async browser launcher.

    Usage:
        vibe = await browser.launch()
        await vibe.go("https://example.com")
        await vibe.quit()
    """

    @staticmethod
    async def launch(
        headless: bool = False,
        port: Optional[int] = None,
        executable_path: Optional[str] = None,
        width: Optional[int] = None,
        height: Optional[int] = None,
    ) -> Vibe:
        """Launch a new browser instance.

        Args:
            headless: Run browser in headless mode (default: visible).
            port: WebSocket port (default: auto-assigned).
            executable_path: Path to clicker binary (default: auto-detect).
            width: Browser window width in pixels (default: 1280).
            height: Browser window height in pixels (default: 720).

        Returns:
            A Vibe instance for browser automation.
        """
        process = await ClickerProcess.start(
            headless=headless,
            port=port,
            executable_path=executable_path,
            width=width,
            height=height,
        )

        client = await BiDiClient.connect(f"ws://localhost:{process.port}")

        return Vibe(client, process)
