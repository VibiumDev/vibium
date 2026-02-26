/**
 * HTTP server for sync API tests.
 * Runs in a separate process because the sync API blocks the main thread
 * with Atomics.wait(), which would prevent an in-process server from responding.
 *
 * Usage: node sync-test-server.js
 * Prints the base URL and WS URL to stdout, then serves until killed.
 */
const http = require('http');
const { WebSocketServer } = require('ws');

const HOME_HTML = `<html><head><title>Test App</title></head><body>
  <h1 class="heading">Welcome to test-app</h1>
  <a href="/subpage">Go to subpage</a>
  <a href="/inputs">Inputs</a>
  <a href="/form">Form</a>
  <p id="info">Some info text</p>
</body></html>`;

const SUBPAGE_HTML = `<html><head><title>Subpage</title></head><body>
  <h3>Subpage Title</h3>
  <a href="/">Back home</a>
</body></html>`;

const INPUTS_HTML = `<html><head><title>Inputs</title></head><body>
  <input type="text" id="text-input" />
  <input type="number" id="num-input" />
  <textarea id="textarea"></textarea>
</body></html>`;

const FORM_HTML = `<html><head><title>Form</title></head><body>
  <form>
    <label for="name">Name</label>
    <input type="text" id="name" name="name" />

    <label for="email">Email</label>
    <input type="email" id="email" name="email" />

    <label for="agree"><input type="checkbox" id="agree" name="agree" /> I agree</label>

    <select id="color" name="color">
      <option value="red">Red</option>
      <option value="green">Green</option>
      <option value="blue">Blue</option>
    </select>

    <button type="submit">Submit</button>
  </form>
</body></html>`;

const LINKS_HTML = `<html><head><title>Links</title></head><body>
  <ul>
    <li><a href="/subpage" class="link">Link 1</a></li>
    <li><a href="/subpage" class="link">Link 2</a></li>
    <li><a href="/subpage" class="link">Link 3</a></li>
    <li><a href="/subpage" class="link special">Link 4</a></li>
  </ul>
  <div id="nested">
    <span class="inner">Nested span</span>
    <span class="inner">Another span</span>
  </div>
</body></html>`;

const EVAL_HTML = `<html><head><title>Eval</title></head><body>
  <div id="result"></div>
  <script>window.testVal = 42;</script>
</body></html>`;

const DIALOG_HTML = `<html><head><title>Dialog</title></head><body>
  <button id="alert-btn" onclick="alert('hello')">Alert</button>
  <button id="confirm-btn" onclick="document.getElementById('result').textContent = confirm('sure?')">Confirm</button>
  <div id="result"></div>
</body></html>`;

const CLOCK_HTML = `<html><head><title>Clock</title></head><body>
  <div id="time"></div>
</body></html>`;

const PROMPT_HTML = `<html><head><title>Prompt</title></head><body>
  <button id="prompt-btn" onclick="document.getElementById('result').textContent = prompt('Enter name:')">Prompt</button>
  <button id="confirm-btn" onclick="document.getElementById('result').textContent = confirm('sure?')">Confirm</button>
  <button id="alert-btn" onclick="alert('hello')">Alert</button>
  <div id="result"></div>
</body></html>`;

const NAV_TEST_HTML = `<html><head><title>Nav Test</title></head><body>
  <a id="link" href="/page2">Go to page 2</a>
</body></html>`;

const PAGE2_HTML = `<html><head><title>Page 2</title></head><body>
  <h1>Page 2</h1>
</body></html>`;

const DOWNLOAD_HTML = `<html><head><title>Download</title></head><body>
  <a href="/download-file" id="download-link" download="test.txt">Download</a>
</body></html>`;

const FETCH_HTML = `<html><head><title>Fetch</title></head><body>
  <div id="result"></div>
  <script>
    async function doFetch() {
      const res = await fetch('/api/data');
      const json = await res.json();
      document.getElementById('result').textContent = JSON.stringify(json);
    }
    async function doPostFetch() {
      const res = await fetch('/api/echo', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ hello: 'world' }),
      });
      const json = await res.json();
      document.getElementById('result').textContent = JSON.stringify(json);
    }
  </script>
</body></html>`;

const WS_PAGE_HTML = `<html><head><title>WebSocket</title></head><body>
  <div id="ws-status">disconnected</div>
  <div id="ws-messages"></div>
  <script>
    function createWS(url) {
      const ws = new WebSocket(url);
      ws.onopen = () => {
        document.getElementById('ws-status').textContent = 'connected';
      };
      ws.onmessage = (e) => {
        const div = document.getElementById('ws-messages');
        div.textContent += e.data + '\\n';
      };
      ws.onclose = () => {
        document.getElementById('ws-status').textContent = 'closed';
      };
      return ws;
    }
  </script>
</body></html>`;

const routes = {
  '/': HOME_HTML,
  '/subpage': SUBPAGE_HTML,
  '/inputs': INPUTS_HTML,
  '/form': FORM_HTML,
  '/links': LINKS_HTML,
  '/eval': EVAL_HTML,
  '/dialog': DIALOG_HTML,
  '/clock': CLOCK_HTML,
  '/prompt': PROMPT_HTML,
  '/fetch': FETCH_HTML,
  '/nav-test': NAV_TEST_HTML,
  '/page2': PAGE2_HTML,
  '/download': DOWNLOAD_HTML,
  '/ws-page': WS_PAGE_HTML,
};

const server = http.createServer((req, res) => {
  if (req.url === '/api/data') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ message: 'real data', count: 42 }));
    return;
  }
  if (req.url === '/api/echo' && req.method === 'POST') {
    let body = '';
    req.on('data', (chunk) => { body += chunk; });
    req.on('end', () => {
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ echo: JSON.parse(body) }));
    });
    return;
  }
  if (req.url === '/download-file') {
    res.writeHead(200, {
      'Content-Type': 'application/octet-stream',
      'Content-Disposition': 'attachment; filename="test.txt"',
    });
    res.end('download content');
    return;
  }
  res.writeHead(200, { 'Content-Type': 'text/html' });
  res.end(routes[req.url] || HOME_HTML);
});

// WebSocket echo server
const wsServer = new WebSocketServer({ port: 0, host: '127.0.0.1' });
wsServer.on('connection', (ws) => {
  ws.on('message', (data) => {
    ws.send(data.toString());
  });
});

server.listen(0, '127.0.0.1', () => {
  const { port } = server.address();
  const wsAddr = wsServer.address();
  // Print HTTP base URL and WS URL to stdout so parent process can read them
  process.stdout.write(`http://127.0.0.1:${port}\n`);
  process.stdout.write(`ws://127.0.0.1:${wsAddr.port}\n`);
});
