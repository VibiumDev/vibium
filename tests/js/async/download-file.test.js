/**
 * JS Library Tests: Downloads & Files
 * Tests page.onDownload, download.saveAs, download.url, download.suggestedFilename,
 * el.setFiles, and removeAllListeners('download').
 *
 * Uses a local HTTP server — no external network dependencies.
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const http = require('http');
const fs = require('fs');
const path = require('path');
const os = require('os');

const { browser } = require('../../../clients/javascript/dist');

// --- Local test server ---

let server;
let baseURL;

const HTML_PAGE = `
<html>
<body>
  <a id="download-link" href="/download">Download File</a>
  <input id="file-input" type="file" />
  <p id="file-name"></p>
  <script>
    document.getElementById('file-input').addEventListener('change', (e) => {
      const name = e.target.files[0] ? e.target.files[0].name : '';
      document.getElementById('file-name').textContent = name;
    });
  </script>
</body>
</html>
`;

const DOWNLOAD_CONTENT = 'Hello from downloaded file!';

before(async () => {
  server = http.createServer((req, res) => {
    if (req.url === '/download') {
      res.writeHead(200, {
        'Content-Type': 'application/octet-stream',
        'Content-Disposition': 'attachment; filename="test-file.txt"',
      });
      res.end(DOWNLOAD_CONTENT);
    } else {
      res.writeHead(200, { 'Content-Type': 'text/html' });
      res.end(HTML_PAGE);
    }
  });

  await new Promise((resolve) => {
    server.listen(0, '127.0.0.1', () => {
      const { port } = server.address();
      baseURL = `http://127.0.0.1:${port}`;
      resolve();
    });
  });
});

after(() => {
  if (server) server.close();
});

// --- Download Events ---

describe('Downloads: page.onDownload', () => {
  test('onDownload() fires when download link clicked', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      const downloads = [];
      vibe.onDownload((dl) => downloads.push(dl));

      await vibe.find('#download-link').click();
      await vibe.wait(1000);

      assert.ok(downloads.length >= 1, `Expected at least 1 download, got ${downloads.length}`);
      assert.strictEqual(downloads[0].suggestedFilename(), 'test-file.txt');
    } finally {
      await bro.close();
    }
  });

  test('download.url() returns the download URL', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      const downloads = [];
      vibe.onDownload((dl) => downloads.push(dl));

      await vibe.find('#download-link').click();
      await vibe.wait(1000);

      assert.ok(downloads.length >= 1);
      assert.ok(downloads[0].url().includes('/download'), `Expected URL to contain /download, got: ${downloads[0].url()}`);
    } finally {
      await bro.close();
    }
  });

  test('download.saveAs(path) saves file with correct content', async () => {
    const bro = await browser.launch({ headless: true });
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-dl-test-'));
    const savePath = path.join(tmpDir, 'saved-file.txt');
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      const downloads = [];
      vibe.onDownload((dl) => downloads.push(dl));

      await vibe.find('#download-link').click();
      await vibe.wait(1000);

      assert.ok(downloads.length >= 1, 'Should have received download event');
      await downloads[0].saveAs(savePath);

      assert.ok(fs.existsSync(savePath), 'Saved file should exist');
      const content = fs.readFileSync(savePath, 'utf-8');
      assert.strictEqual(content, DOWNLOAD_CONTENT);
    } finally {
      await bro.close();
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });
});

// --- el.setFiles ---

describe('Element: el.setFiles', () => {
  test('setFiles() sets file on input type=file', async () => {
    const bro = await browser.launch({ headless: true });
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-sf-test-'));
    const testFile = path.join(tmpDir, 'upload-test.txt');
    fs.writeFileSync(testFile, 'test upload content');
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      await vibe.find('#file-input').setFiles([testFile]);
      await vibe.wait(300);

      const fileName = await vibe.find('#file-name').text();
      assert.strictEqual(fileName, 'upload-test.txt');
    } finally {
      await bro.close();
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });
});

// --- removeAllListeners ---

describe('removeAllListeners for download', () => {
  test('removeAllListeners("download") clears download callbacks', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      const downloads = [];
      vibe.onDownload((dl) => downloads.push(dl));

      vibe.removeAllListeners('download');

      await vibe.find('#download-link').click();
      await vibe.wait(1000);

      assert.strictEqual(downloads.length, 0, 'Should not capture downloads after removeAllListeners');
    } finally {
      await bro.close();
    }
  });
});
