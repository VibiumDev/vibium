"""Local HTTP test server for Python tests.

All routes match the JS sync-test-server.js so tests are equivalent.
"""

import json
import threading
from http.server import HTTPServer, BaseHTTPRequestHandler


HOME_HTML = """<html><head><title>Test App</title></head><body>
  <h1 class="heading">Welcome to test-app</h1>
  <a href="/subpage">Go to subpage</a>
  <a href="/inputs">Inputs</a>
  <a href="/form">Form</a>
  <p id="info">Some info text</p>
</body></html>"""

SUBPAGE_HTML = """<html><head><title>Subpage</title></head><body>
  <h3>Subpage Title</h3>
  <a href="/">Back home</a>
</body></html>"""

INPUTS_HTML = """<html><head><title>Inputs</title></head><body>
  <input type="text" id="text-input" />
  <input type="number" id="num-input" />
  <textarea id="textarea"></textarea>
</body></html>"""

FORM_HTML = """<html><head><title>Form</title></head><body>
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
</body></html>"""

LINKS_HTML = """<html><head><title>Links</title></head><body>
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
</body></html>"""

EVAL_HTML = """<html><head><title>Eval</title></head><body>
  <div id="result"></div>
  <script>window.testVal = 42;</script>
</body></html>"""

DIALOG_HTML = """<html><head><title>Dialog</title></head><body>
  <button id="alert-btn" onclick="alert('hello')">Alert</button>
  <button id="confirm-btn" onclick="document.getElementById('result').textContent = confirm('sure?')">Confirm</button>
  <div id="result"></div>
</body></html>"""

CLOCK_HTML = """<html><head><title>Clock</title></head><body>
  <div id="time"></div>
</body></html>"""

PROMPT_HTML = """<html><head><title>Prompt</title></head><body>
  <button id="prompt-btn" onclick="document.getElementById('result').textContent = prompt('Enter name:')">Prompt</button>
  <button id="confirm-btn" onclick="document.getElementById('result').textContent = confirm('sure?')">Confirm</button>
  <button id="alert-btn" onclick="alert('hello')">Alert</button>
  <div id="result"></div>
</body></html>"""

FETCH_HTML = """<html><head><title>Fetch</title></head><body>
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
</body></html>"""

FRAMES_HTML = """<html><head><title>Frames</title></head><body>
  <h1>Frames Page</h1>
  <iframe src="/frame1" name="frame-one"></iframe>
</body></html>"""

FRAME1_HTML = """<html><head><title>Frame 1</title></head><body>
  <h2>Frame 1 Content</h2>
  <iframe src="/frame2" name="inner"></iframe>
</body></html>"""

FRAME2_HTML = """<html><head><title>Frame 2</title></head><body>
  <h3>Frame 2 Nested</h3>
</body></html>"""

DYNAMIC_LOADING_HTML = """<html><head><title>Dynamic</title></head><body>
  <div id="container"></div>
  <script>
    setTimeout(() => {
      const el = document.createElement('div');
      el.id = 'loaded';
      el.textContent = 'Loaded!';
      document.getElementById('container').appendChild(el);
    }, 500);
  </script>
</body></html>"""

WS_PAGE_HTML = """<html><head><title>WebSocket</title></head><body>
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
</body></html>"""

NAV_TEST_HTML = """<html><head><title>Nav Test</title></head><body>
  <a id="link" href="/page2">Go to page 2</a>
</body></html>"""

PAGE2_HTML = """<html><head><title>Page 2</title></head><body>
  <h1>Page 2</h1>
</body></html>"""

DOWNLOAD_HTML = """<html><head><title>Download</title></head><body>
  <a href="/download-file" id="download-link" download="test.txt">Download</a>
</body></html>"""

A11Y_HTML = """<html><head><title>A11y</title></head><body>
  <nav aria-label="Main navigation">
    <a href="/subpage">Go to subpage</a>
  </nav>
  <main>
    <h1>Heading Level 1</h1>
    <h2>Heading Level 2</h2>
    <button aria-label="Close dialog">X</button>
    <p id="desc">Description text</p>
    <input type="text" aria-labelledby="desc" />
    <label for="username">Username</label>
    <input type="text" id="username" />
    <input type="checkbox" id="cb" checked />
    <button disabled>Disabled Button</button>
  </main>
</body></html>"""

SET_COOKIE_HTML = """<html><head><title>Cookies Set</title></head><body>
  <p>Cookies set!</p>
</body></html>"""

TEXT_PAGE = "hello world"

JSON_DATA = {"name": "vibium", "version": 1}
API_DATA = {"message": "real data", "count": 42}

HTML_ROUTES = {
    "/": HOME_HTML,
    "/subpage": SUBPAGE_HTML,
    "/inputs": INPUTS_HTML,
    "/form": FORM_HTML,
    "/links": LINKS_HTML,
    "/eval": EVAL_HTML,
    "/dialog": DIALOG_HTML,
    "/clock": CLOCK_HTML,
    "/prompt": PROMPT_HTML,
    "/fetch": FETCH_HTML,
    "/frames": FRAMES_HTML,
    "/frame1": FRAME1_HTML,
    "/frame2": FRAME2_HTML,
    "/dynamic-loading": DYNAMIC_LOADING_HTML,
    "/ws-page": WS_PAGE_HTML,
    "/download": DOWNLOAD_HTML,
    "/nav-test": NAV_TEST_HTML,
    "/page2": PAGE2_HTML,
    "/a11y": A11Y_HTML,
}


class VibiumTestHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        path = self.path.split("?")[0]
        if path == "/api/echo":
            length = int(self.headers.get("Content-Length", 0))
            body = self.rfile.read(length)
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(json.dumps({"echo": json.loads(body)}).encode())
            return
        self.send_response(404)
        self.end_headers()

    def do_GET(self):
        path = self.path.split("?")[0]

        if path == "/api/data":
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(json.dumps(API_DATA).encode())
            return

        if path == "/json":
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(json.dumps(JSON_DATA).encode())
            return

        if path == "/text":
            self.send_response(200)
            self.send_header("Content-Type", "text/plain")
            self.end_headers()
            self.wfile.write(TEXT_PAGE.encode())
            return

        if path == "/set-cookie":
            self.send_response(200)
            self.send_header("Content-Type", "text/html")
            self.send_header("Set-Cookie", "test_cookie=hello; Path=/")
            self.send_header("Set-Cookie", "another=world; Path=/")
            self.end_headers()
            self.wfile.write(SET_COOKIE_HTML.encode())
            return

        if path == "/download-file":
            self.send_response(200)
            self.send_header("Content-Type", "application/octet-stream")
            self.send_header("Content-Disposition", 'attachment; filename="test.txt"')
            self.end_headers()
            self.wfile.write(b"download content")
            return

        html = HTML_ROUTES.get(path, HOME_HTML)
        self.send_response(200)
        self.send_header("Content-Type", "text/html")
        self.end_headers()
        self.wfile.write(html.encode())

    def log_message(self, format, *args):
        pass  # Suppress request logging


def start_test_server() -> tuple:
    """Start a test server on a random port. Returns (server, base_url)."""
    server = HTTPServer(("127.0.0.1", 0), VibiumTestHandler)
    port = server.server_address[1]
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    return server, f"http://127.0.0.1:{port}"
