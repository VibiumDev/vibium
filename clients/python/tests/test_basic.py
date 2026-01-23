"""Basic tests for the Vibium Python client."""

import os
from vibium import browser_sync


def test_sync_api():
    """Test the synchronous API."""
    # Use headless mode in CI environments
    headless = os.environ.get("CHROME_HEADLESS", "").lower() in ("true", "1")
    vibe = browser_sync.launch(headless=headless)
    try:
        vibe.go("https://example.com")

        # Test find and text
        link = vibe.find("a")
        text = link.text()
        assert text, f"Expected link text, got: {text}"

        # Test screenshot
        png = vibe.screenshot()
        assert len(png) > 1000, f"Screenshot too small: {len(png)} bytes"

        # Test click
        link.click()
    finally:
        vibe.quit()


if __name__ == "__main__":
    test_sync_api()
    print("Python client test passed!")
