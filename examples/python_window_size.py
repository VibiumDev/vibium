"""Example: Launch browser with custom window size."""

from vibium import browser_sync

# Launch with 1920x1080 resolution
vibe = browser_sync.launch(width=1920, height=1080)

vibe.go("https://example.com")

# Take a screenshot at the specified resolution
png = vibe.screenshot()
with open("screenshot_1920x1080.png", "wb") as f:
    f.write(png)
    print(f"Screenshot saved: {len(png)} bytes")

vibe.quit()

print("Done! Browser launched with 1920x1080 resolution")
