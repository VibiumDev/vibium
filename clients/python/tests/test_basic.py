"""Basic tests for the Vibium Python client."""

from vibium import browser


def test_sync_api():
    """Test the synchronous API with new object model."""
    bro = browser.launch(headless=True)
    try:
        vibe = bro.new_page()
        vibe.go("https://example.com")

        # Test title
        title = vibe.title()
        assert title == "Example Domain", f"Expected 'Example Domain', got: {title}"

        # Test url
        url = vibe.url()
        assert "example.com" in url, f"Expected URL with 'example.com', got: {url}"

        # Test find (CSS) and text
        h1 = vibe.find("h1")
        h1_text = h1.text()
        assert h1_text == "Example Domain", f"Expected 'Example Domain', got: {h1_text}"

        # Test find (semantic kwargs) — uses info.text from find result
        heading = vibe.find(role="heading")
        assert heading.info.text == "Example Domain", f"Expected 'Example Domain', got: {heading.info.text}"

        # Test find link and click
        link = vibe.find("a")
        link_text = link.text()
        assert link_text, f"Expected link text, got: {link_text}"

        # Test screenshot
        png = vibe.screenshot()
        assert len(png) > 1000, f"Screenshot too small: {len(png)} bytes"

        # Test eval
        result = vibe.eval("2 + 2")
        assert result == 4, f"Expected 4, got: {result}"

        # Test eval string
        doc_title = vibe.eval("document.title")
        assert doc_title == "Example Domain", f"Expected 'Example Domain', got: {doc_title}"

        # Test click
        link.click()

    finally:
        bro.close()


if __name__ == "__main__":
    test_sync_api()
    print("Python client test passed!")
