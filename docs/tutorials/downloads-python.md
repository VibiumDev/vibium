# Handling Downloads (Python)

Detect and save files downloaded by the browser.

Downloads need a real HTTP server — the browser must receive a `Content-Disposition: attachment` header to trigger a download.

## Quick Example

First, save this server as `server.py` and start it:

```python
# server.py
from http.server import HTTPServer, BaseHTTPRequestHandler

class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/file":
            self.send_response(200)
            self.send_header("Content-Type", "text/plain")
            self.send_header("Content-Disposition", 'attachment; filename="hello.txt"')
            self.end_headers()
            self.wfile.write(b"hello world")
        else:
            self.send_response(200)
            self.send_header("Content-Type", "text/html")
            self.end_headers()
            self.wfile.write(b'<a href="/file" id="dl-link">Download hello.txt</a>')

server = HTTPServer(("localhost", 3000), Handler)
print("http://localhost:3000")
server.serve_forever()
```

```
python server.py
```

Then in another terminal, run the download script:

```python
from vibium import browser

bro = browser.launch()
vibe = bro.page()
vibe.go("http://localhost:3000")

result = vibe.capture.download(lambda: vibe.find("#dl-link").click())

print(result["suggested_filename"])  # hello.txt
result.save_as("/tmp/hello.txt")

bro.close()
```

<details>
<summary>Async version</summary>

```python
import asyncio
from vibium.async_api import browser

async def main():
    bro = await browser.launch()
    vibe = await bro.page()
    await vibe.go("http://localhost:3000")

    async with vibe.capture.download() as cap:
        el = await vibe.find("#dl-link")
        await el.click()

    download = cap.value
    print(download.suggested_filename())  # hello.txt
    await download.save_as("/tmp/hello.txt")

    await bro.close()

asyncio.run(main())
```

In async mode you get a `Download` object with methods and `save_as()`. In sync mode you get a `SyncDownload` (dict subclass) with both dict access and `save_as()`.

</details>

## capture.download

`capture.download()` sets up a listener, runs your action, and returns the download:

<!-- test: sync "capture.download returns download with properties" -->
```python
import os
import tempfile
import shutil
from vibium import browser

bro = browser.launch(headless=True)
vibe = bro.page()
vibe.go(base_url)

result = vibe.capture.download(lambda: vibe.find("#dl-link").click())

assert "/file" in result["url"]
assert result["suggested_filename"] == "hello.txt"
assert result["path"] is not None

tmp_dir = tempfile.mkdtemp(prefix="vibium-dl-")
try:
    result.save_as(os.path.join(tmp_dir, "saved.txt"))
    with open(os.path.join(tmp_dir, "saved.txt")) as f:
        assert f.read() == "hello world"
finally:
    shutil.rmtree(tmp_dir)

bro.close()
```

<details>
<summary>Async test</summary>

<!-- test: async "capture.download returns download with properties" -->
```python
import os
import tempfile
import shutil
from vibium.async_api import browser

bro = await browser.launch(headless=True)
vibe = await bro.page()
await vibe.go(base_url)

async with vibe.capture.download() as cap:
    el = await vibe.find("#dl-link")
    await el.click()

download = cap.value
assert "/file" in download.url()
assert download.suggested_filename() == "hello.txt"

tmp_dir = tempfile.mkdtemp(prefix="vibium-dl-")
try:
    await download.save_as(os.path.join(tmp_dir, "saved.txt"))
    with open(os.path.join(tmp_dir, "saved.txt")) as f:
        assert f.read() == "hello world"
finally:
    shutil.rmtree(tmp_dir)

await bro.close()
```

</details>

## on_download

For ongoing monitoring, `on_download()` fires on every download:

<!-- test: sync "on_download fires on download" -->
```python
import time
from vibium import browser

bro = browser.launch(headless=True)
vibe = bro.page()
vibe.go(base_url)

downloads = []
vibe.on_download(lambda dl: downloads.append(dl))

vibe.find("#dl-link").click()
time.sleep(2)

assert len(downloads) >= 1
assert downloads[0].suggested_filename() == "hello.txt"
assert downloads[0]["path"] is not None

bro.close()
```

<details>
<summary>Async test</summary>

<!-- test: async "on_download fires on download" -->
```python
import asyncio
from vibium.async_api import browser

bro = await browser.launch(headless=True)
vibe = await bro.page()
await vibe.go(base_url)

downloads = []
vibe.on_download(lambda dl: downloads.append(dl))

el = await vibe.find("#dl-link")
await el.click()
await asyncio.sleep(1)

assert len(downloads) >= 1
assert downloads[0].suggested_filename() == "hello.txt"

await bro.close()
```

</details>

---

- [Downloads (JavaScript)](downloads-js.md)
- [Accessibility Tree (Python)](a11y-tree-python.md)
