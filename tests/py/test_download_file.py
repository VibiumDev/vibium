"""Download & file tests — onDownload, saveAs, setFiles (5 async tests)."""

import os
import tempfile

import pytest


async def test_download_fires(fresh_async_browser, test_server):
    """onDownload fires when download starts."""
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/download")
    downloads = []
    vibe.on_download(lambda d: downloads.append(d))
    link = await vibe.find("#download-link")
    await link.click()
    await vibe.wait(1000)
    assert len(downloads) >= 1


async def test_download_url(fresh_async_browser, test_server):
    """Download object has correct URL."""
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/download")
    downloads = []
    vibe.on_download(lambda d: downloads.append(d))
    link = await vibe.find("#download-link")
    await link.click()
    await vibe.wait(1000)
    assert len(downloads) >= 1
    assert "download-file" in downloads[0].url()


async def test_download_save_as(fresh_async_browser, test_server):
    """Download can be saved to a file."""
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/download")
    downloads = []
    vibe.on_download(lambda d: downloads.append(d))
    link = await vibe.find("#download-link")
    await link.click()
    await vibe.wait(1000)
    assert len(downloads) >= 1

    with tempfile.NamedTemporaryFile(suffix=".txt", delete=False) as f:
        dest = f.name
    try:
        await downloads[0].save_as(dest)
        assert os.path.exists(dest)
        with open(dest, "rb") as f:
            content = f.read()
        assert b"download content" in content
    finally:
        if os.path.exists(dest):
            os.unlink(dest)


async def test_set_files(fresh_async_browser, test_server):
    """setFiles sets file input."""
    vibe = await fresh_async_browser.new_page()
    await vibe.set_content('<input type="file" id="file-input" />')

    with tempfile.NamedTemporaryFile(suffix=".txt", delete=False, mode="w") as f:
        f.write("test content")
        temp_path = f.name

    try:
        inp = await vibe.find("#file-input")
        await inp.set_files([temp_path])
        # Verify file was set
        name = await vibe.eval("document.getElementById('file-input').files[0]?.name")
        assert name is not None
        assert ".txt" in name
    finally:
        os.unlink(temp_path)


async def test_remove_download_listeners(fresh_async_browser, test_server):
    """removeAllListeners('download') clears download handlers."""
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/download")
    downloads = []
    vibe.on_download(lambda d: downloads.append(d))
    vibe.remove_all_listeners("download")
    link = await vibe.find("#download-link")
    await link.click()
    await vibe.wait(1000)
    assert len(downloads) == 0
