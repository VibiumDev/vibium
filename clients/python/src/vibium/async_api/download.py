"""Download data object."""

from __future__ import annotations

import asyncio
from typing import Any, Optional, TYPE_CHECKING

if TYPE_CHECKING:
    from ..client import BiDiClient


class Download:
    """Represents a file download triggered by the page."""

    def __init__(self, client: BiDiClient, url: str, suggested_filename: str) -> None:
        self._client = client
        self._url = url
        self._suggested_filename = suggested_filename
        self._future: asyncio.Future = asyncio.get_event_loop().create_future()

    def url(self) -> str:
        return self._url

    def suggested_filename(self) -> str:
        return self._suggested_filename

    async def save_as(self, path: str) -> None:
        """Wait for the download to complete, then save to the specified path."""
        result = await self._future
        if result["status"] != "complete" or not result.get("filepath"):
            raise RuntimeError(f"Download failed with status: {result['status']}")
        await self._client.send("vibium:download.saveAs", {
            "sourcePath": result["filepath"],
            "destPath": path,
        })

    async def path(self) -> Optional[str]:
        """Wait for the download to complete and return the temp file path."""
        result = await self._future
        return result.get("filepath")

    def _complete(self, status: str, filepath: Optional[str]) -> None:
        """Internal: called by Page when downloadCompleted fires."""
        if not self._future.done():
            self._future.set_result({"status": status, "filepath": filepath})
