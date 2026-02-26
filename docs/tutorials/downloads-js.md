# Handling Downloads

Detect and save files downloaded by the browser.

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
  const bro = await browser.launch();
  const vibe = await bro.page();
  await vibe.go('http://localhost:3000');

  const download = await vibe.capture.download(async () => {
    await vibe.find('#dl-link').click();
  });

  console.log(download.suggestedFilename()); // hello.txt
  await download.saveAs('/tmp/hello.txt');

  await bro.close();
}

main();
```

<details>
<summary>Sync version</summary>

```javascript
const { browser } = require('vibium/sync');

const bro = browser.launch();
const vibe = bro.page();
vibe.go('http://localhost:3000');

const result = vibe.capture.download(() => {
  vibe.find('#dl-link').click();
});

console.log(result.suggestedFilename); // hello.txt
result.saveAs('/tmp/hello.txt');

bro.close();
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

const bro = await browser.launch({ headless: true });
const vibe = await bro.page();
await vibe.go(baseURL);

const download = await vibe.capture.download(async () => {
  await vibe.find('#dl-link').click();
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

await bro.close();
```

<details>
<summary>Sync test</summary>

<!-- test: sync "capture.download returns download with properties" -->
```javascript
const { browser } = require('vibium/sync');
const fs = require('fs');
const os = require('os');
const path = require('path');

const bro = browser.launch({ headless: true });
const vibe = bro.page();
vibe.go(baseURL);

const result = vibe.capture.download(() => {
  vibe.find('#dl-link').click();
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

bro.close();
```

</details>

## onDownload

For ongoing monitoring, `onDownload()` fires on every download:

<!-- test: async "onDownload fires on download" -->
```javascript
const { browser } = require('vibium');

const bro = await browser.launch({ headless: true });
const vibe = await bro.page();
await vibe.go(baseURL);

const downloads = [];
vibe.onDownload((dl) => downloads.push(dl));

await vibe.find('#dl-link').click();
await vibe.wait(1000);

assert.ok(downloads.length >= 1);
assert.strictEqual(downloads[0].suggestedFilename(), 'hello.txt');

await bro.close();
```

<details>
<summary>Sync test</summary>

<!-- test: sync "onDownload fires on download" -->
```javascript
const { browser } = require('vibium/sync');

const bro = browser.launch({ headless: true });
const vibe = bro.page();
vibe.go(baseURL);

const downloads = [];
vibe.onDownload((dl) => downloads.push(dl));

vibe.find('#dl-link').click();
vibe.wait(2000);

assert.ok(downloads.length >= 1);
assert.strictEqual(downloads[0].suggestedFilename, 'hello.txt');
assert.ok(downloads[0].path, 'download should have path');

bro.close();
```

</details>

Call `removeAllListeners('download')` to stop listening.

---

- [Downloads (Python)](downloads-python.md)
- [Accessibility Tree](a11y-tree-js.md)
