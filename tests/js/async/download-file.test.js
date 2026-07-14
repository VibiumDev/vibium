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
const { withTimeout } = require('../helpers/wait');

// --- Local test server ---

let server;
let baseURL;
let bro;

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

  bro = await browser.start({ headless: true });
});

after(async () => {
  if (bro) await bro.stop();
  if (server) server.close();
});

// Each test uses a fresh page so onDownload listeners don't leak.
async function freshPage() {
  const vibe = await bro.newPage();
  await vibe.go(baseURL);
  return vibe;
}

// --- Download Events ---

describe('Downloads: page.onDownload', () => {
  test('onDownload() fires when download link clicked', async () => {
    const vibe = await freshPage();
    try {
      const downloads = [];
      const firstDownload = new Promise((resolve) =>
        vibe.onDownload((dl) => {
          downloads.push(dl);
          resolve();
        }),
      );

      await vibe.find('#download-link').click();
      await withTimeout(firstDownload, 5000, 'onDownload to fire');

      assert.strictEqual(downloads[0].suggestedFilename(), 'test-file.txt');
    } finally {
      await vibe.close();
    }
  });

  test('download.url() returns the download URL', async () => {
    const vibe = await freshPage();
    try {
      const downloads = [];
      const firstDownload = new Promise((resolve) =>
        vibe.onDownload((dl) => {
          downloads.push(dl);
          resolve();
        }),
      );

      await vibe.find('#download-link').click();
      await withTimeout(firstDownload, 5000, 'onDownload to fire');

      assert.ok(downloads[0].url().includes('/download'), `Expected URL to contain /download, got: ${downloads[0].url()}`);
    } finally {
      await vibe.close();
    }
  });

  test('download.saveAs(path) saves file with correct content', async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-dl-test-'));
    const savePath = path.join(tmpDir, 'saved-file.txt');
    const vibe = await freshPage();
    try {
      const downloads = [];
      const firstDownload = new Promise((resolve) =>
        vibe.onDownload((dl) => {
          downloads.push(dl);
          resolve();
        }),
      );

      await vibe.find('#download-link').click();
      await withTimeout(firstDownload, 5000, 'onDownload to fire');

      await downloads[0].saveAs(savePath);

      assert.ok(fs.existsSync(savePath), 'Saved file should exist');
      const content = fs.readFileSync(savePath, 'utf-8');
      assert.strictEqual(content, DOWNLOAD_CONTENT);
    } finally {
      await vibe.close();
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });
});

// --- el.setFiles ---

describe('Element: el.setFiles', () => {
  test('setFiles() sets file on input type=file', async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-sf-test-'));
    const testFile = path.join(tmpDir, 'upload-test.txt');
    fs.writeFileSync(testFile, 'test upload content');
    const vibe = await freshPage();
    try {
      await vibe.find('#file-input').setFiles([testFile]);
      // setFiles may resolve before the change-event handler runs in Chrome
      // and updates #file-name. Poll the DOM instead of sleeping.
      await vibe.waitUntil(
        `() => document.getElementById('file-name').textContent === 'upload-test.txt'`,
        { timeout: 5000 },
      );

      const fileName = await vibe.find('#file-name').text();
      assert.strictEqual(fileName, 'upload-test.txt');
    } finally {
      await vibe.close();
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });
});

// --- removeAllListeners ---

describe('removeAllListeners for download', () => {
  test('removeAllListeners("download") clears download callbacks', async () => {
    const vibe = await freshPage();
    try {
      const downloads = [];
      vibe.onDownload((dl) => downloads.push(dl));
      vibe.removeAllListeners('download');

      // Use the server hitting /download as a barrier: the request only fires
      // after Chrome processes the click, which is the same point onDownload
      // would have triggered. A drain eval afterward gives any in-flight
      // server→client events time to arrive before we assert absence.
      let resolveDownloadHit;
      const downloadHit = new Promise((resolve) => { resolveDownloadHit = resolve; });
      const probe = (req) => { if (req.url === '/download') resolveDownloadHit(); };
      server.on('request', probe);
      try {
        await vibe.find('#download-link').click();
        await withTimeout(downloadHit, 5000, 'download request to reach server');
      } finally {
        server.off('request', probe);
      }
      await vibe.evaluate('1');

      assert.strictEqual(downloads.length, 0, 'Should not capture downloads after removeAllListeners');
    } finally {
      await vibe.close();
    }
  });
});
