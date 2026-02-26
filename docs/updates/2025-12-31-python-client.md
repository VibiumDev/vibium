# pip install vibium 🥂

happy new year! we shipped v1 by christmas. now we're ringing in 2026 by publishing the Python client to PyPI.

```bash
pip install vibium
```

that's it. chrome downloads automatically on first use.

## what got published

five packages on PyPI:

- `vibium` - main package with sync + async APIs
- `vibium-darwin-arm64` - macOS Apple Silicon
- `vibium-darwin-x64` - macOS Intel
- `vibium-linux-x64` - Linux x64
- `vibium-win32-x64` - Windows x64

the right binary installs automatically based on your platform.

## quick start (sync)

```python
from vibium import browser_sync

vibe = browser_sync.launch()
vibe.go("https://example.com")

link = vibe.find("a")
print(link.text())
link.click()

vibe.quit()
```

## quick start (async version):

```python
import asyncio
from vibium import browser

async def main():
    vibe = await browser.launch()
    await vibe.go("https://example.com")
    await vibe.quit()

asyncio.run(main())
```

## CLI

```bash
vibium install   # pre-download chrome for testing
vibium version   # show version
```

## new make targets

```bash
make package-python     # build all Python wheels
make test-python        # run Python client tests
make clean-python-packages  # clean built wheels
```

## docs

- [Getting Started (Python)](../tutorials/getting-started-python.md)
- [PyPI Publishing Guide](https://github.com/VibiumDev/vibium/blob/c858092/docs/how-to-guides/pypi-publishing.md)

## the stack

v1 shipped 🚢:
- Go (clicker binary)
- TypeScript (JS client) - `npm install vibium`
- Python (Python client) - `pip install vibium`
- MCP server for Claude Code

## what's next

people are using vibium! we've got a few bug reports coming in - we'll be working on those next.

cheers to 2026. 🥂🎆

*december 31, 2025*
