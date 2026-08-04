# Getting Started with Vibium (Python)

A complete beginner's guide. No prior experience required.

---

## What You'll Build

A script that opens a browser, visits a website, takes a screenshot, and clicks a link. All in about 10 lines of code.

---

## Step 1: Install Python

Vibium requires Python 3.9 or higher. Check if you have it:

```bash
python3 --version
```

If you see a version number (like `Python 3.9.0` or higher), skip to Step 2.

### macOS

```bash
# Install Homebrew (if you don't have it)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install Python
brew install python
```

### Windows

Download and run the installer from [python.org](https://python.org). Choose the latest version.

### Linux

```bash
# Ubuntu/Debian
sudo apt-get install python3 python3-pip python3-venv

# Or use your distro's package manager
```

---

## Step 2: Create a Project Folder

```bash
mkdir my-first-bot
cd my-first-bot
python3 -m venv .venv
source .venv/bin/activate  # On Windows: .venv\Scripts\activate
```

---

## Step 3: Install Vibium

```bash
pip install vibium
```

This downloads Vibium and the vibium binary for your platform. Chrome downloads automatically on first run (or run `vibium install` to pre-download it).

| Platform | Cache path |
|----------|------------|
| Linux | `~/.cache/vibium/` |
| macOS | `~/Library/Caches/vibium/` |
| Windows | `%LOCALAPPDATA%\vibium\` |

Chrome downloads automatically on first `browser.start()`. To skip this (if you manage Chrome separately), set the env var before running your script:
```bash
export VIBIUM_SKIP_BROWSER_DOWNLOAD=1
```

---

## Step 4: Write Your First Script

Create a file called `hello.py`:

```python
from vibium import browser

# Launch a browser (you'll see it open!)
bro = browser.start()
vibe = bro.page()

# Go to a website
vibe.go("https://example.com")
print("Loaded example.com")

# Take a screenshot
png = vibe.screenshot()
with open("screenshot.png", "wb") as f:
    f.write(png)
print("Saved screenshot.png")

# Find and click the link
link = vibe.find("a")
print("Found link:", link.text())
link.click()
print("Clicked!")

# Close the browser
bro.stop()
print("Done!")
```

---

## Step 5: Run It

```bash
python3 hello.py
```

You should see:
1. A Chrome window open
2. example.com load
3. The browser click "Learn more"
4. The browser close

Check your folder - there's now a `screenshot.png` file!

---

## What Just Happened?

| Line | What It Does |
|------|--------------|
| `browser.start()` | Opens Chrome, returns a Browser |
| `bro.page()` | Gets the default page (tab) |
| `vibe.go(url)` | Navigates to a URL |
| `vibe.screenshot()` | Captures the page as PNG bytes |
| `vibe.find(selector)` | Finds an element by CSS selector |
| `link.text()` | Gets the element's text content |
| `link.click()` | Clicks the element |
| `bro.stop()` | Closes the browser |

---

## Next Steps

**Hide the browser** (run headless):
```python
bro = browser.start(headless=True)
```

**Use async/await** (for more complex scripts):
```python
import asyncio
from vibium.async_api import browser

async def main():
    bro = await browser.start()
    vibe = await bro.page()
    await vibe.go("https://example.com")
    # ...
    await bro.stop()

asyncio.run(main())
```

**Use JavaScript instead:**
See [Getting Started (JavaScript)](getting-started-js.md) for the JS version.

**Let AI control the browser:**
See [Agent Setup](../../README.md#agent-setup) for CLI setup and [Getting Started with MCP](getting-started-mcp.md) for MCP server setup.

---

## Troubleshooting

### "command not found: python3"

Python isn't installed or isn't in your PATH. Reinstall from [python.org](https://python.org).

### "No module named 'vibium'"

Make sure you activated the virtual environment:
```bash
source .venv/bin/activate
```

Then install vibium:
```bash
pip install vibium
```

### Browser doesn't open

Try running with headless mode disabled (it's disabled by default, but just in case):
```python
bro = browser.start(headless=False)
```

### Custom cache directory

By default, Chrome for Testing installs to the platform-specific cache path shown in [Step 3](#step-3-install-vibium). To change this (e.g. if your IT policy restricts writes to `%LOCALAPPDATA%`), set `VIBIUM_CACHE_DIR`:

```bash
# macOS/Linux
export VIBIUM_CACHE_DIR=/path/to/allowed/dir

# Windows (PowerShell)
$env:VIBIUM_CACHE_DIR = "C:\path\to\allowed\dir"
```

Then re-run your script — Chrome for Testing will download to the new location.

### Permission denied (Linux)

You might need to install dependencies for Chrome:
```bash
sudo apt-get install -y libgbm1 libnss3 libatk-bridge2.0-0 libdrm2 libxkbcommon0 libxcomposite1 libxdamage1 libxfixes3 libxrandr2 libasound2
```

---

## You Did It!

You just automated a browser with Python. The same techniques work for:
- Web scraping
- Testing websites
- Automating repetitive tasks
- Building AI agents that can browse the web

Questions? [Open an issue](https://github.com/VibiumDev/vibium/issues).
