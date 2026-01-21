"""Tests for browser resolution support."""

import asyncio
from vibium import browser, browser_sync


def test_sync_default_resolution():
    """Test the synchronous API with default resolution."""
    vibe = browser_sync.launch()
    try:
        vibe.go("https://example.com")

        # Take screenshot and verify it's not empty
        png = vibe.screenshot()
        assert len(png) > 1000, f"Screenshot too small: {len(png)} bytes"

        print("✅ Default resolution works (1280x720)")
    finally:
        vibe.quit()


def test_sync_custom_resolution_1920x1080():
    """Test the synchronous API with 1920x1080 resolution."""
    vibe = browser_sync.launch(width=1920, height=1080)
    try:
        vibe.go("https://example.com")

        # Take screenshot - should be larger than default
        png = vibe.screenshot()
        assert len(png) > 1000, f"Screenshot too small: {len(png)} bytes"

        print("✅ 1920x1080 resolution works")
    finally:
        vibe.quit()


def test_sync_custom_resolution_800x600():
    """Test the synchronous API with 800x600 resolution."""
    vibe = browser_sync.launch(width=800, height=600)
    try:
        vibe.go("https://example.com")

        # Take screenshot - should be smaller than default
        png = vibe.screenshot()
        assert len(png) > 500, f"Screenshot too small: {len(png)} bytes"

        print("✅ 800x600 resolution works")
    finally:
        vibe.quit()


def test_sync_only_width():
    """Test the synchronous API with only width specified."""
    vibe = browser_sync.launch(width=1600)
    try:
        vibe.go("https://example.com")

        # Should use default height (720)
        png = vibe.screenshot()
        assert len(png) > 1000, f"Screenshot too small: {len(png)} bytes"

        print("✅ Width-only (1600x720) resolution works")
    finally:
        vibe.quit()


def test_sync_only_height():
    """Test the synchronous API with only height specified."""
    vibe = browser_sync.launch(height=900)
    try:
        vibe.go("https://example.com")

        # Should use default width (1280)
        png = vibe.screenshot()
        assert len(png) > 1000, f"Screenshot too small: {len(png)} bytes"

        print("✅ Height-only (1280x900) resolution works")
    finally:
        vibe.quit()


async def test_async_custom_resolution():
    """Test the async API with custom resolution."""
    vibe = await browser.launch(width=1920, height=1080)
    try:
        await vibe.go("https://example.com")

        # Take screenshot
        png = await vibe.screenshot()
        assert len(png) > 1000, f"Screenshot too small: {len(png)} bytes"

        # Find an element to ensure browser is working
        link = await vibe.find("a")
        text = await link.text()
        assert text, f"Expected link text, got: {text}"

        print("✅ Async API with 1920x1080 resolution works")
    finally:
        await vibe.quit()


def test_sync_multiple_resolutions():
    """Test launching browsers with different resolutions sequentially."""
    resolutions = [
        (800, 600),
        (1280, 720),
        (1920, 1080),
        (2560, 1440),
    ]

    for width, height in resolutions:
        vibe = browser_sync.launch(width=width, height=height)
        try:
            vibe.go("https://example.com")
            png = vibe.screenshot()
            assert (
                len(png) > 500
            ), f"Screenshot too small for {width}x{height}: {len(png)} bytes"
            print(
                f"✅ {width}x{height} resolution works - screenshot: {len(png)} bytes"
            )
        finally:
            vibe.quit()


def test_sync_headless_with_resolution():
    """Test headless mode with custom resolution."""
    vibe = browser_sync.launch(headless=True, width=1920, height=1080)
    try:
        vibe.go("https://example.com")

        # Take screenshot in headless mode
        png = vibe.screenshot()
        assert len(png) > 1000, f"Screenshot too small: {len(png)} bytes"

        print("✅ Headless mode with 1920x1080 resolution works")
    finally:
        vibe.quit()


def run_all_tests():
    """Run all resolution tests."""
    print("=" * 60)
    print("Running Python Client Resolution Tests")
    print("=" * 60)
    print()

    # Sync tests
    print("🔍 Testing sync API...")
    test_sync_default_resolution()
    test_sync_custom_resolution_1920x1080()
    test_sync_custom_resolution_800x600()
    test_sync_only_width()
    test_sync_only_height()
    test_sync_headless_with_resolution()
    test_sync_multiple_resolutions()

    # Async test
    print()
    print("🔍 Testing async API...")
    asyncio.run(test_async_custom_resolution())

    print()
    print("=" * 60)
    print("✅ All resolution tests passed!")
    print("=" * 60)


if __name__ == "__main__":
    run_all_tests()
