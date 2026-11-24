"""The sync download wait must outlast the async wait it wraps (#856)."""

import asyncio

from vibium.async_api.download import Download
from vibium.sync_api.page import _SyncCaptureNamespace


class _Client:
    async def send(self, method, params, timeout=60):
        self.timeout = timeout
        return {"status": "complete", "filepath": "/tmp/file"}


class _Download:
    def url(self):
        return "http://example.test/file"

    def suggested_filename(self):
        return "file"

    def path(self):
        return "path-wait"


class _Loop:
    """Stands in for the event loop thread and records each wait's timeout."""

    def __init__(self):
        self.timeouts = {}

    def run(self, work, timeout=120):
        self.timeouts[work] = timeout
        return {"setup": "event-wait", "event-wait": _Download(), "path-wait": "/tmp/file"}[work]


class _Async:
    def _setup_capture_download(self, timeout):
        return "setup"


class _Page:
    def __init__(self):
        self._loop = _Loop()
        self._async = _Async()


def _async_path_timeout():
    client = _Client()
    asyncio.run(Download(client, {"navigation": "nav"}).path())
    return client.timeout


def test_capture_download_waits_as_long_as_the_async_path():
    page = _Page()
    result = _SyncCaptureNamespace(page).download(lambda: None)
    assert result["path"] == "/tmp/file"
    assert page._loop.timeouts["path-wait"] > _async_path_timeout()


def test_capture_download_block_waits_as_long_as_the_async_path():
    page = _Page()
    with _SyncCaptureNamespace(page).download() as captured:
        pass
    assert captured.value["path"] == "/tmp/file"
    assert page._loop.timeouts["path-wait"] > _async_path_timeout()
