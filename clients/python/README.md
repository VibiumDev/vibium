# Vibium Python Client

Browser automation for AI agents and humans.

## Installation

```bash
pip install vibium
```

Chrome downloads automatically on first use. To install Chrome ahead of time:

```bash
vibium install
```

## Quick Start

```python
from vibium import browser_sync

vibe = browser_sync.launch()
vibe.go("https://example.com")

# Take a screenshot
png = vibe.screenshot()
with open("screenshot.png", "wb") as f:
    f.write(png)

# Find and click a link
link = vibe.find("a")
print(link.text())
link.click()

vibe.quit()
```

### Custom Window Size

```python
from vibium import browser_sync

# Launch with specific resolution
vibe = browser_sync.launch(width=1920, height=1080)
vibe.go("https://example.com")
vibe.quit()
```

## Async API

```python
import asyncio
from vibium import browser

async def main():
    vibe = await browser.launch(width=1920, height=1080)
    await vibe.go("https://example.com")

    link = await vibe.find("a")
    await link.click()

    await vibe.quit()

asyncio.run(main())
```

## CLI

```bash
vibium install   # Download Chrome for Testing
vibium version   # Show version
```

## Requirements

- Python 3.9+

## License

Apache-2.0
