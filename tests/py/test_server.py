"""Local HTTP test server for Python tests.

All routes match the JS sync-test-server.js so tests are equivalent.
"""

import json
import struct
import threading
import zlib
from http.server import ThreadingHTTPServer, BaseHTTPRequestHandler
from urllib.parse import parse_qs, urlparse


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

SELECTORS_HTML = """<html><head><title>Selectors</title></head><body>
  <h1>Selector Strategies Test</h1>
  <input type="text" id="search" placeholder="Search..." data-testid="search-input" title="Search field" />
  <label for="username">Username</label>
  <input type="text" id="username" name="username" />
  <img src="data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7" alt="Logo image" />
  <button data-testid="submit-btn">Submit</button>
  <div class="container"><span class="inner-text">Hello from span</span></div>
</body></html>"""

# Only selector + text reaches "Go now": text alone picks the shorter link,
# and the selector alone picks "Stop". Mirrors the JS test server page.
COMBINED_LOCATOR_HTML = """<html><head><title>Combined locator</title></head><body>
  <a id="go" href="#">Go</a>
  <form onsubmit="return false"><button id="stop">Stop</button><button id="go-now">Go now</button></form>
  <div id="src">Drag me</div>
  <div id="out"></div>
  <script>document.addEventListener('mouseup', (e) => { document.getElementById('out').textContent = e.target.id; });</script>
</body></html>"""

# The textarea is the shortest "Notes" text match; the decoy is longer.
TYPE_FILTER_HTML = """<html><head><title>Type filter</title></head><body>
  <div id="decoy">Notes are rendered here after save</div>
  <textarea id="notes">Notes</textarea>
</body></html>"""

# Two identical children; only a scoped re-resolve reaches the one in #b.
NESTED_SCOPE_HTML = """<html><head><title>Nested scope</title></head><body>
  <div id="a"><button class="act">A</button></div>
  <div id="b"><button class="act">B</button></div>
  <div id="out"></div>
  <script>document.addEventListener('mouseup', (e) => { document.getElementById('out').textContent = e.target.textContent; });</script>
</body></html>"""

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
    "/selectors": SELECTORS_HTML,
    "/combined-locator": COMBINED_LOCATOR_HTML,
    "/type-filter": TYPE_FILTER_HTML,
    "/nested-scope": NESTED_SCOPE_HTML,
    "/find-images": None,
    "/find-canvas": None,
}


def make_pattern_png(seed: int, size: int = 96) -> bytes:
    """A deterministic, feature-rich thumbnail for image-find tests: the same
    drawing as tests/helpers/test-server.js makePatternPNG."""
    state = seed & 0xFFFFFFFF
    px = bytearray([128] * (size * size * 3))

    def rand():
        nonlocal state
        state = (state * 1664525 + 1013904223) & 0xFFFFFFFF
        return state / 4294967296

    def put(x, y, v):
        if 0 <= x < size and 0 <= y < size:
            i = (y * size + x) * 3
            px[i] = px[i + 1] = px[i + 2] = v

    for _ in range(60):
        w, h = 4 + int(rand() * 18), 4 + int(rand() * 18)
        x0, y0 = int(rand() * (size - w)), int(rand() * (size - h))
        v = int(rand() * 256)
        if rand() < 0.5:
            for y in range(y0, y0 + h):
                for x in range(x0, x0 + w):
                    put(x, y, v)
        else:
            r = min(w, h) / 2
            cx, cy = x0 + r, y0 + r
            for y in range(y0, int(y0 + 2 * r)):
                for x in range(x0, int(x0 + 2 * r)):
                    if (x - cx) ** 2 + (y - cy) ** 2 <= r * r:
                        put(x, y, v)
    raw = b"".join(b"\x00" + bytes(px[y * size * 3:(y + 1) * size * 3]) for y in range(size))

    def chunk(kind, data):
        return struct.pack(">I", len(data)) + kind + data + struct.pack(">I", zlib.crc32(kind + data) & 0xFFFFFFFF)

    return (b"\x89PNG\r\n\x1a\n"
            + chunk(b"IHDR", struct.pack(">IIBBBBB", size, size, 8, 2, 0, 0, 0))
            + chunk(b"IDAT", zlib.compress(raw))
            + chunk(b"IEND", b""))


PATTERN_PNG = make_pattern_png(20261106)
DECOY_PNG = make_pattern_png(777)

FIND_IMAGES_HTML = """<html><head><title>Find images</title></head><body style="margin:0;padding:20px;font-family:sans-serif">
  <h1>Gallery</h1>
  <p>Three copies of one thumbnail and a decoy.</p>
  <div style="display:flex;gap:40px;align-items:flex-start">
    <img id="t1" src="/fixtures/pattern.png" width="96" height="96" alt="Product thumbnail">
    <img id="t2" src="/fixtures/pattern.png" width="96" height="96" alt="Product thumbnail">
    <img id="decoy" src="/fixtures/decoy.png" width="96" height="96" alt="Other thumbnail">
  </div>
  <div style="margin-top:40px"><img id="t3" src="/fixtures/pattern.png" width="96" height="96" alt="Product thumbnail"></div>
  <p><button id="add" onclick="window.__added=(window.__added||0)+1">Add to Cart</button> <a id="more" href="/subpage">Learn more</a></p>
  <p><input id="q" type="search" placeholder="Search products"></p>
</body></html>"""

FIND_CANVAS_HTML = """<html><head><title>Find canvas</title></head><body style="margin:0;padding:20px;font-family:sans-serif">
  <h1>Canvas</h1>
  <div style="height:40px"></div>
  <canvas id="c" width="700" height="500" style="display:block;background:#dde"></canvas>
  <div style="height:900px"></div>
  <script>
    window.__clicks = [];
    const c = document.getElementById('c');
    c.addEventListener('click', (e) => { const r = c.getBoundingClientRect(); window.__clicks.push({ x: e.clientX - r.left, y: e.clientY - r.top }); });
    const img = new Image();
    img.onload = () => { c.getContext('2d').drawImage(img, 300, 150); window.__drawn = true; };
    img.src = '/fixtures/pattern.png';
  </script>
</body></html>"""

HTML_ROUTES["/find-images"] = FIND_IMAGES_HTML
HTML_ROUTES["/find-canvas"] = FIND_CANVAS_HTML


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

        if path in ("/fixtures/pattern.png", "/fixtures/decoy.png"):
            self.send_response(200)
            self.send_header("Content-Type", "image/png")
            self.send_header("Cache-Control", "no-store")
            self.end_headers()
            self.wfile.write(DECOY_PNG if path.endswith("decoy.png") else PATTERN_PNG)
            return

        if path == "/set-cookie":
            self.send_response(200)
            self.send_header("Content-Type", "text/html")
            self.send_header("Set-Cookie", "test_cookie=hello; Path=/")
            self.send_header("Set-Cookie", "another=world; Path=/")
            self.end_headers()
            self.wfile.write(SET_COOKIE_HTML.encode())
            return

        # ?name= gives a test its own filename. Repeating one name within a
        # browser session makes Firefox report the deduplicated name
        # ("test(1).txt") in downloadWillBegin, so name-asserting tests must
        # not share a filename.
        if path == "/download-file":
            query = parse_qs(urlparse(self.path).query)
            name = query.get("name", ["test.txt"])[0]
            self.send_response(200)
            self.send_header("Content-Type", "application/octet-stream")
            self.send_header("Content-Disposition", f'attachment; filename="{name}"')
            self.end_headers()
            self.wfile.write(b"download content")
            return

        if path == "/download":
            query = parse_qs(urlparse(self.path).query)
            if "name" in query:
                name = query["name"][0]
                self.send_response(200)
                self.send_header("Content-Type", "text/html")
                self.end_headers()
                self.wfile.write(
                    f"""<html><head><title>Download</title></head><body>
  <a href="/download-file?name={name}" id="download-link" download="{name}">Download</a>
</body></html>""".encode()
                )
                return

        html = HTML_ROUTES.get(path, HOME_HTML)
        self.send_response(200)
        self.send_header("Content-Type", "text/html")
        self.end_headers()
        self.wfile.write(html.encode())

    def log_message(self, format, *args):
        pass  # Suppress request logging


def start_test_server() -> tuple:
    """Start a test server on a random port. Returns (server, base_url).

    Uses ThreadingHTTPServer so each connection is handled on its own thread.
    A single-threaded HTTPServer serializes requests, so while it's busy
    handling/draining one of the several parallel connections Chrome opens to
    load a page, the main-document request waits behind it — under the
    parallel test suite's CPU load that stall reached the 60s navigate
    timeout and flaked (unlike the JS suite, whose server is a separate process).
    """
    server = ThreadingHTTPServer(("127.0.0.1", 0), VibiumTestHandler)
    port = server.server_address[1]
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    return server, f"http://127.0.0.1:{port}"
