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
        proxy: Optional[str] = None,
    ) -> Vibe:
        """Launch a new browser instance.

        Args:
            headless: Run browser in headless mode (default: visible).
            port: WebSocket port (default: auto-assigned).
            executable_path: Path to clicker binary (default: auto-detect).
            proxy: Proxy server URL (e.g., http://proxy:8080, socks5://proxy:1080).

        Returns:
            A Vibe instance for browser automation.
        """
        process = await ClickerProcess.start(
            headless=headless,
            port=port,
            executable_path=executable_path,
            proxy=proxy,
        )

        client = await BiDiClient.connect(f"ws://localhost:{process.port}")

        return Vibe(client, process)
