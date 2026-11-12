# Handling Downloads

Detect and save files downloaded by the browser.

## JavaScript
Downloads need a real HTTP server — the browser must receive a `Content-Disposition: attachment` header to trigger a download.

## Quick Example

First, save this server as `server.js` and start it:

```javascript
// server.js
const http = require('http');

const server = http.createServer((req, res) => {
  if (req.url === '/file') {
    res.writeHead(200, {
      'Content-Type': 'text/plain',
      'Content-Disposition': 'attachment; filename="hello.txt"',
    });
    return res.end('hello world');
  }
  res.writeHead(200, { 'Content-Type': 'text/html' });
  res.end('<a href="/file" id="dl-link">Download hello.txt</a>');
});

server.listen(3000, () => console.log('http://localhost:3000'));
```

```
node server.js
```

Then in another terminal, run the download script:

```javascript
const { browser } = require('vibium');

async function main() {
  const bro = await browser.start();
  const vibe = await bro.page();
  await vibe.go('http://localhost:3000');

  const download = await vibe.capture.download(async () => {
    await vibe.find({ css: '#dl-link' }).click();
  });

  console.log(download.suggestedFilename()); // hello.txt
  await download.saveAs('/tmp/hello.txt');

  await bro.stop();
}

main();
```

<details>
<summary>Sync version</summary>

```javascript
const { browser } = require('vibium/sync');

const bro = browser.start();
const vibe = bro.page();
vibe.go('http://localhost:3000');

const result = vibe.capture.download(() => {
  vibe.find({ css: '#dl-link' }).click();
});

console.log(result.suggestedFilename); // hello.txt
result.saveAs('/tmp/hello.txt');

bro.stop();
```

</details>

## capture.download

`capture.download()` sets up a listener, runs your action, and returns the download:

<!-- test: async "capture.download returns download with properties" -->
```javascript
const { browser } = require('vibium');
const fs = require('fs');
const os = require('os');
const path = require('path');

const bro = await browser.start({ headless: true });
const vibe = await bro.page();
await vibe.go(baseURL);

const download = await vibe.capture.download(async () => {
  await vibe.find({ css: '#dl-link' }).click();
});

assert.ok(download.url().includes('/file'));
assert.strictEqual(download.suggestedFilename(), 'hello.txt');

const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-dl-'));
try {
  await download.saveAs(path.join(tmpDir, 'saved.txt'));
  assert.strictEqual(fs.readFileSync(path.join(tmpDir, 'saved.txt'), 'utf-8'), 'hello world');
} finally {
  fs.rmSync(tmpDir, { recursive: true, force: true });
}

await bro.stop();
```

<details>
<summary>Sync test</summary>

<!-- test: sync "capture.download returns download with properties" -->
```javascript
const { browser } = require('vibium/sync');
const fs = require('fs');
const os = require('os');
const path = require('path');

const bro = browser.start({ headless: true });
const vibe = bro.page();
vibe.go(baseURL);

const result = vibe.capture.download(() => {
  vibe.find({ css: '#dl-link' }).click();
});

assert.ok(result.url.includes('/file'));
assert.strictEqual(result.suggestedFilename, 'hello.txt');
assert.ok(result.path, 'path should be set after download');

const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-dl-'));
try {
  result.saveAs(path.join(tmpDir, 'saved.txt'));
  assert.strictEqual(fs.readFileSync(path.join(tmpDir, 'saved.txt'), 'utf-8'), 'hello world');
} finally {
  fs.rmSync(tmpDir, { recursive: true, force: true });
}

bro.stop();
```

</details>

## onDownload

For ongoing monitoring, `onDownload()` fires on every download:

<!-- test: async "onDownload fires on download" -->
```javascript
const { browser } = require('vibium');

const bro = await browser.start({ headless: true });
const vibe = await bro.page();
await vibe.go(baseURL);

const downloads = [];
vibe.onDownload((dl) => downloads.push(dl));

await vibe.find({ css: '#dl-link' }).click();
await vibe.wait(1000);

assert.ok(downloads.length >= 1);
assert.strictEqual(downloads[0].suggestedFilename(), 'hello.txt');

await bro.stop();
```

<details>
<summary>Sync test</summary>

<!-- test: sync "onDownload fires on download" -->
```javascript
const { browser } = require('vibium/sync');

const bro = browser.start({ headless: true });
const vibe = bro.page();
vibe.go(baseURL);

const downloads = [];
vibe.onDownload((dl) => downloads.push(dl));

vibe.find({ css: '#dl-link' }).click();
vibe.wait(2000);

assert.ok(downloads.length >= 1);
assert.strictEqual(downloads[0].suggestedFilename, 'hello.txt');
assert.ok(downloads[0].path, 'download should have path');

bro.stop();
```

</details>

Call `removeAllListeners('download')` to stop listening.

## Python
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

bro = browser.start()
vibe = bro.page()
vibe.go("http://localhost:3000")

result = vibe.capture.download(lambda: vibe.find(css="#dl-link").click())

print(result["suggested_filename"])  # hello.txt
result.save_as("/tmp/hello.txt")

bro.stop()
```

<details>
<summary>Async version</summary>

```python
import asyncio
from vibium.async_api import browser

async def main():
    bro = await browser.start()
    vibe = await bro.page()
    await vibe.go("http://localhost:3000")

    async with vibe.capture.download() as cap:
        el = await vibe.find(css="#dl-link")
        await el.click()

    download = cap.value
    print(download.suggested_filename())  # hello.txt
    await download.save_as("/tmp/hello.txt")

    await bro.stop()

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

bro = browser.start(headless=True)
vibe = bro.page()
vibe.go(base_url)

result = vibe.capture.download(lambda: vibe.find(css="#dl-link").click())

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

bro.stop()
```

<details>
<summary>Async test</summary>

<!-- test: async "capture.download returns download with properties" -->
```python
import os
import tempfile
import shutil
from vibium.async_api import browser

bro = await browser.start(headless=True)
vibe = await bro.page()
await vibe.go(base_url)

async with vibe.capture.download() as cap:
    el = await vibe.find(css="#dl-link")
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

await bro.stop()
```

</details>

## on_download

For ongoing monitoring, `on_download()` fires on every download:

<!-- test: sync "on_download fires on download" -->
```python
import time
from vibium import browser

bro = browser.start(headless=True)
vibe = bro.page()
vibe.go(base_url)

downloads = []
vibe.on_download(lambda dl: downloads.append(dl))

vibe.find(css="#dl-link").click()

# The download event arrives asynchronously; poll briefly for it.
deadline = time.time() + 15
while not downloads and time.time() < deadline:
    time.sleep(0.1)

assert len(downloads) >= 1
assert downloads[0].suggested_filename() == "hello.txt"
assert downloads[0]["path"] is not None

bro.stop()
```

<details>
<summary>Async test</summary>

<!-- test: async "on_download fires on download" -->
```python
import asyncio
from vibium.async_api import browser

bro = await browser.start(headless=True)
vibe = await bro.page()
await vibe.go(base_url)

downloads = []
vibe.on_download(lambda dl: downloads.append(dl))

el = await vibe.find(css="#dl-link")
await el.click()
await asyncio.sleep(1)

assert len(downloads) >= 1
assert downloads[0].suggested_filename() == "hello.txt"

await bro.stop()
```

</details>

## Java

## Quick Example

First, save this server as `Server.java` and start it:

```java
// Server.java
import com.sun.net.httpserver.HttpServer;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;

public class Server {
    public static void main(String[] args) throws Exception {
        HttpServer server = HttpServer.create(new InetSocketAddress("localhost", 3000), 0);
        server.createContext("/", exchange -> {
            byte[] body;
            if (exchange.getRequestURI().getPath().equals("/file")) {
                exchange.getResponseHeaders().add("Content-Type", "text/plain");
                exchange.getResponseHeaders().add("Content-Disposition", "attachment; filename=\"hello.txt\"");
                body = "hello world".getBytes(StandardCharsets.UTF_8);
            } else {
                exchange.getResponseHeaders().add("Content-Type", "text/html");
                body = "<a href=\"/file\" id=\"dl-link\">Download hello.txt</a>".getBytes(StandardCharsets.UTF_8);
            }
            exchange.sendResponseHeaders(200, body.length);
            exchange.getResponseBody().write(body);
            exchange.close();
        });
        server.start();
        System.out.println("http://localhost:3000");
    }
}
```

```
java Server.java
```

Then in another terminal, run the download program:

```java
import com.vibium.Download;
import com.vibium.Vibium;
import com.vibium.types.SelectorOptions;

public class DownloadExample {
    public static void main(String[] args) {
        var bro = Vibium.start();
        var vibe = bro.page();
        vibe.go("http://localhost:3000");

        Download download = vibe.capture().download(
            () -> vibe.find(new SelectorOptions().css("#dl-link")).click());

        System.out.println(download.suggestedFilename()); // hello.txt
        download.saveAs("/tmp/hello.txt");

        bro.stop();
    }
}
```

## capture().download

`capture().download()` sets up a listener, runs your action, and returns the download:

```java
Download download = vibe.capture().download(
    () -> vibe.find(new SelectorOptions().css("#dl-link")).click());

download.url();                // ends with /file
download.suggestedFilename();  // "hello.txt"
download.saveAs("/tmp/hello.txt");
download.path();               // the browser's temp file, once the download completes
```

## onDownload

`onDownload()` registers a callback that fires for every download on the page:

```java
List<Download> downloads = new CopyOnWriteArrayList<>();
vibe.onDownload(downloads::add);

vibe.find(new SelectorOptions().css("#dl-link")).click();

// The download event arrives asynchronously; poll briefly for it.
for (int i = 0; i < 50 && downloads.isEmpty(); i++) {
    Thread.sleep(100);
}

downloads.get(0).suggestedFilename();  // "hello.txt"
```

---

## See also

- [Accessibility Tree](accessibility-tree.md)
- [Getting Started with JavaScript](../tutorials/getting-started-js.md)
- [Getting Started with Python](../tutorials/getting-started-python.md)
