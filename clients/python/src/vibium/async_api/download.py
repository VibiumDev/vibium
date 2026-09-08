"""Download data object."""

from __future__ import annotations

from typing import Any, Dict, Optional, TYPE_CHECKING

if TYPE_CHECKING:
    from ..client import BiDiClient

# Default timeout for download completion (5 minutes).
_DOWNLOAD_TIMEOUT = 300


class Download:
    """Represents a file download triggered by the page."""

    def __init__(self, client: BiDiClient, params: Dict[str, Any]) -> None:
        self._client = client
        self._url = params.get("url", "")
        self._suggested_filename = params.get("suggestedFilename", "")
        self._navigation = params.get("navigation", "")

    def url(self) -> str:
        return self._url

    def suggested_filename(self) -> str:
        return self._suggested_filename

    async def _wait_for_completion(self) -> dict:
        """Completion is awaited in the engine (vibium:download.await), which
        tracks every download by its navigation id, so no client keeps a
        pending-downloads map. Finished downloads answer immediately, so
        asking repeatedly is fine."""
        return await self._client.send(
            "vibium:download.await",
            {"navigation": self._navigation, "timeout": _DOWNLOAD_TIMEOUT * 1000},
            timeout=_DOWNLOAD_TIMEOUT + 10,
        )

    async def save_as(self, path: str) -> None:
        """Wait for the download to complete, then save to the specified path."""
        result = await self._wait_for_completion()
        if result["status"] != "complete" or not result.get("filepath"):
            raise RuntimeError(f"Download failed with status: {result['status']}")
        await self._client.send("vibium:download.saveAs", {
            "sourcePath": result["filepath"],
            "destPath": path,
        })

    async def path(self) -> Optional[str]:
        """Wait for the download to complete and return the temp file path."""
        result = await self._wait_for_completion()
        return result.get("filepath") or None
