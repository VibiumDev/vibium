"""Sync download event tests — on_download (3 sync tests)."""


def test_download_fires(sync_browser, test_server):
    """on_download fires when download link clicked."""
    vibe = sync_browser.new_page()
    vibe.go(test_server + "/download")
    downloads = []
    vibe.on_download(lambda d: downloads.append(d))
    link = vibe.find("#download-link")
    link.click()
    vibe.wait(1000)
    assert len(downloads) >= 1


def test_download_url_filename(sync_browser, test_server):
    """Download object has url and suggested_filename."""
    vibe = sync_browser.new_page()
    vibe.go(test_server + "/download")
    downloads = []
    vibe.on_download(lambda d: downloads.append(d))
    link = vibe.find("#download-link")
    link.click()
    vibe.wait(1000)
    assert len(downloads) >= 1
    assert "download-file" in downloads[0].url()
    assert downloads[0].suggested_filename() == "test.txt"


def test_remove_download_listeners(sync_browser, test_server):
    """remove_all_listeners('download') stops on_download callbacks."""
    vibe = sync_browser.new_page()
    vibe.go(test_server + "/download")
    downloads = []
    vibe.on_download(lambda d: downloads.append(d))
    vibe.remove_all_listeners("download")
    link = vibe.find("#download-link")
    link.click()
    vibe.wait(1000)
    assert len(downloads) == 0
