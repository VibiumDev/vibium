/**
 * Shared local HTTP test server.
 *
 * Replaces the-internet.herokuapp.com for all tests.
 * Serves HTML pages that mirror the external site's structure.
 *
 * Usage (inline in async tests):
 *   const { createTestServer } = require('../helpers/test-server');
 *   let server, baseURL;
 *   before(async () => ({ server, baseURL } = await createTestServer()));
 *   after(() => server.close());
 *
 * Usage (subprocess for sync tests):
 *   node tests/helpers/test-server.js
 *   # prints base URL to stdout, serves until killed
 */

const http = require('http');
const zlib = require('zlib');

// A deterministic, feature-rich thumbnail for image-find tests: rectangles
// and discs at many gray levels give SIFT plenty of corners and blobs. The
// decoy uses another seed so it looks alike without corresponding.
function makePatternPNG(seed, size = 96) {
  let state = seed >>> 0;
  const rand = () => { state = (Math.imul(state, 1664525) + 1013904223) >>> 0; return state / 4294967296; };
  const px = new Uint8Array(size * size * 3).fill(128);
  const put = (x, y, v) => { if (x >= 0 && y >= 0 && x < size && y < size) { const i = (y * size + x) * 3; px[i] = v; px[i + 1] = v; px[i + 2] = v; } };
  for (let n = 0; n < 60; n++) {
    const w = 4 + Math.floor(rand() * 18), h = 4 + Math.floor(rand() * 18);
    const x0 = Math.floor(rand() * (size - w)), y0 = Math.floor(rand() * (size - h));
    const v = Math.floor(rand() * 256);
    if (rand() < 0.5) {
      for (let y = y0; y < y0 + h; y++) for (let x = x0; x < x0 + w; x++) put(x, y, v);
    } else {
      const r = Math.min(w, h) / 2, cx = x0 + r, cy = y0 + r;
      for (let y = y0; y < y0 + 2 * r; y++) for (let x = x0; x < x0 + 2 * r; x++) if ((x - cx) ** 2 + (y - cy) ** 2 <= r * r) put(x, y, v);
    }
  }
  const raw = Buffer.alloc((size * 3 + 1) * size);
  for (let y = 0; y < size; y++) { raw[y * (size * 3 + 1)] = 0; Buffer.from(px.buffer, y * size * 3, size * 3).copy(raw, y * (size * 3 + 1) + 1); }
  const crcTable = new Int32Array(256);
  for (let n = 0; n < 256; n++) { let c = n; for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1; crcTable[n] = c; }
  const crc = (buf) => { let c = -1; for (const b of buf) c = crcTable[(c ^ b) & 0xff] ^ (c >>> 8); return (c ^ -1) >>> 0; };
  const chunk = (type, data) => {
    const len = Buffer.alloc(4); len.writeUInt32BE(data.length);
    const td = Buffer.concat([Buffer.from(type), data]);
    const c = Buffer.alloc(4); c.writeUInt32BE(crc(td));
    return Buffer.concat([len, td, c]);
  };
  const ihdr = Buffer.alloc(13); ihdr.writeUInt32BE(size, 0); ihdr.writeUInt32BE(size, 4); ihdr[8] = 8; ihdr[9] = 2; ihdr[10] = 0; ihdr[11] = 0; ihdr[12] = 0;
  return Buffer.concat([Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]), chunk('IHDR', ihdr), chunk('IDAT', zlib.deflateSync(raw)), chunk('IEND', Buffer.alloc(0))]);
}

const PATTERN_PNG = makePatternPNG(20261106);
const DECOY_PNG = makePatternPNG(777);

// --- HTML Pages ---

const HOME_HTML = `<html><head><title>The Internet</title></head><body>
  <h1 class="heading">Welcome to the-internet</h1>
  <ul>
    <li><a href="/login">Form Authentication</a></li>
    <li><a href="/checkboxes">Checkboxes</a></li>
    <li><a href="/hovers">Hovers</a></li>
    <li><a href="/dropdown">Dropdown</a></li>
    <li><a href="/inputs">Inputs</a></li>
    <li><a href="/dynamic_loading/1">Dynamic Loading</a></li>
    <li><a href="/add_remove_elements/">Add/Remove Elements</a></li>
  </ul>
  <div id="page-footer" style="margin-top: 2000px;">
    <p>Powered by <a href="http://elementalselenium.com/">Elemental Selenium</a></p>
  </div>
</body></html>`;

const FORM_REDIRECT_HTML = `<html><head><title>Form Redirect</title></head><body>
  <h2>Form Redirect</h2>
  <form id="f" action="/form-redirect" method="post">
    <input type="text" name="q" id="q" value="hello" />
    <button type="submit" id="go">Submit</button>
  </form>
</body></html>`;

const LOGIN_HTML = `<html><head><title>The Internet - Login</title></head><body>
  <h2>Login Page</h2>
  <form id="login" action="/authenticate" method="post">
    <div><label for="username">Username</label>
    <input type="text" name="username" id="username" /></div>
    <div><label for="password">Password</label>
    <input type="password" name="password" id="password" /></div>
    <button class="radius" type="submit"><i class="fa fa-2x fa-sign-in"></i> Login</button>
  </form>
  <div id="flash"></div>
</body></html>`;

const SECURE_HTML = `<html><head><title>The Internet - Secure Area</title></head><body>
  <div id="flash" class="flash success">You logged into a secure area!</div>
  <h2>Secure Area</h2>
  <p>Welcome to the Secure Area.</p>
</body></html>`;

const CHECKBOXES_HTML = `<html><head><title>The Internet - Checkboxes</title></head><body>
  <h3>Checkboxes</h3>
  <form id="checkboxes">
    <input type="checkbox" /> checkbox 1<br/>
    <input type="checkbox" checked /> checkbox 2
  </form>
</body></html>`;

const HOVERS_HTML = `<html><head><title>The Internet - Hovers</title></head><body>
  <style>
    .figure { position: relative; width: 200px; height: 200px; background: #ccc; display: inline-block; margin: 10px; }
    .figure img { width: 200px; height: 200px; }
    .figure .figcaption { position: absolute; bottom: 0; left: 0; right: 0; background: rgba(0,0,0,0.5); color: white; opacity: 0; transition: opacity 0.3s; padding: 10px; }
    .figure:hover .figcaption { opacity: 1; }
  </style>
  <h3>Hovers</h3>
  <div class="figure"><img src="data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7" /><div class="figcaption"><h5>name: user1</h5></div></div>
  <div class="figure"><img src="data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7" /><div class="figcaption"><h5>name: user2</h5></div></div>
  <div class="figure"><img src="data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7" /><div class="figcaption"><h5>name: user3</h5></div></div>
</body></html>`;

const DROPDOWN_HTML = `<html><head><title>The Internet - Dropdown</title></head><body>
  <h3>Dropdown List</h3>
  <select id="dropdown">
    <option value="" disabled selected>Please select an option</option>
    <option value="1">Option 1</option>
    <option value="2">Option 2</option>
  </select>
</body></html>`;

const INPUTS_HTML = `<html><head><title>The Internet - Inputs</title></head><body>
  <h3>Inputs</h3>
  <input type="number" />
  <textarea id="multiline" rows="5" cols="30"></textarea>
</body></html>`;

const DYNAMIC_LOADING_HTML = `<html><head><title>The Internet - Dynamic Loading</title></head><body>
  <h3>Dynamically Loaded Page Elements</h3>
  <div id="start"><button>Start</button></div>
  <div id="loading" style="display:none;"><img src="data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7" /></div>
  <div id="finish" style="display:none;"><h4>Hello World!</h4></div>
  <script>
    document.querySelector('#start button').addEventListener('click', function() {
      document.getElementById('start').style.display = 'none';
      document.getElementById('loading').style.display = 'block';
      setTimeout(function() {
        document.getElementById('loading').style.display = 'none';
        document.getElementById('finish').style.display = 'block';
      }, 500);
    });
  </script>
</body></html>`;

const ADD_REMOVE_HTML = `<html><head><title>The Internet - Add/Remove</title></head><body>
  <h3>Add/Remove Elements</h3>
  <button onclick="addElement()">Add Element</button>
  <div id="elements"></div>
  <script>
    function addElement() {
      var btn = document.createElement('button');
      btn.className = 'added-manually';
      btn.textContent = 'Delete';
      btn.onclick = function() { this.remove(); };
      document.getElementById('elements').appendChild(btn);
    }
  </script>
</body></html>`;

const EXAMPLE_HTML = `<html><head><title>Example Domain</title></head><body>
  <div>
    <h1>Example Domain</h1>
    <p>This domain is for use in illustrative examples in documents. You may use this
    domain in literature without prior coordination or asking for permission.</p>
    <p><a href="/more-information">More information...</a></p>
  </div>
</body></html>`;

const FIND_IMAGES_HTML = `<html><head><title>Find images</title></head><body style="margin:0;padding:20px;font-family:sans-serif">
  <h1>Gallery</h1>
  <p>Three copies of one thumbnail and a decoy.</p>
  <div style="display:flex;gap:40px;align-items:flex-start">
    <img id="t1" src="/fixtures/pattern.png" width="96" height="96" alt="Product thumbnail">
    <img id="t2" src="/fixtures/pattern.png" width="96" height="96" alt="Product thumbnail">
    <img id="decoy" src="/fixtures/decoy.png" width="96" height="96" alt="Other thumbnail">
  </div>
  <div style="margin-top:40px"><img id="t3" src="/fixtures/pattern.png" width="96" height="96" alt="Product thumbnail"></div>
  <p><button id="add" onclick="window.__added=(window.__added||0)+1">Add to Cart</button> <a id="more" href="/more-information">Learn more</a></p>
  <p><input id="q" type="search" placeholder="Search products"></p>
</body></html>`;

// The pattern drawn into a canvas far larger than itself: the match cannot
// snap to an element of its own size, so it stays a coordinate-backed ref.
const FIND_CANVAS_HTML = `<html><head><title>Find canvas</title></head><body style="margin:0;padding:20px;font-family:sans-serif">
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
</body></html>`;

const MORE_INFORMATION_HTML = `<html><head><title>More information</title></head><body>
  <h1>More information</h1>
  <p>You followed the example link. This page is served locally.</p>
</body></html>`;

const SELECTORS_HTML = `<html><head><title>Selectors</title></head><body>
  <h1>Selector Strategies Test</h1>
  <input type="text" id="search" placeholder="Search..." data-testid="search-input" title="Search field" />
  <label for="username">Username</label>
  <input type="text" id="username" name="username" />
  <img src="data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7" alt="Logo image" />
  <button data-testid="submit-btn">Submit</button>
  <input type="submit" value="Login" />
  <div class="container"><span class="inner-text">Hello from span</span></div>
</body></html>`;

// Only selector + text reaches "Go now": text alone picks the shorter link,
// and the selector alone picks "Stop".
const COMBINED_LOCATOR_HTML = `<html><head><title>Combined locator</title></head><body>
  <a id="go" href="#">Go</a>
  <form onsubmit="return false"><button id="stop">Stop</button><button id="go-now">Go now</button></form>
  <div id="src">Drag me</div>
  <div id="out"></div>
  <script>document.addEventListener('mouseup', (e) => { document.getElementById('out').textContent = e.target.id; });</script>
</body></html>`;

// The textarea is the shortest "Notes" text match; the decoy is longer.
const TYPE_FILTER_HTML = `<html><head><title>Type filter</title></head><body>
  <div id="decoy">Notes are rendered here after save</div>
  <textarea id="notes">Notes</textarea>
</body></html>`;

// Two identical children; only a scoped re-resolve reaches the one in #b.
const NESTED_SCOPE_HTML = `<html><head><title>Nested scope</title></head><body>
  <div id="a"><button class="act">A</button></div>
  <div id="b"><button class="act">B</button></div>
  <div id="out"></div>
  <script>document.addEventListener('mouseup', (e) => { document.getElementById('out').textContent = e.target.textContent; });</script>
</body></html>`;

const FRAME_INNER_HTML = `<html><head><title>Inner Frame</title></head><body>
  <h1 id="inner">Inside the frame</h1>
</body></html>`;

const FRAMES_HTML = `<html><head><title>Frames</title></head><body>
  <h1 id="outer">Outer page</h1>
  <iframe src="/frame-inner" name="myframe"></iframe>
</body></html>`;

// Shadow buttons without ids: "Save" has an old path that resolves to the
// page's first button, "B" has a path that also matches the deeper "Y" first, "C" is in
// the second of two top-level divs, and "Deep" is in a nested shadow root.
// The file input is for upload, which resolves refs through its own script.
// x-spaced's id ends in a space and x-quoted's holds a quote; their refs must
// keep the escapes.
const SHADOW_REFS_HTML = `<html><head><title>Shadow refs</title></head><body>
  <button onclick="document.getElementById('out').textContent = 'light'">Delete account</button>
  <button onclick="document.getElementById('out').textContent = 'light'">Logout</button>
  <button aria-label=">>" onclick="document.getElementById('out').textContent = 'last'">Last</button>
  <x-panel></x-panel>
  <x-spaced id="sp "></x-spaced>
  <x-quoted id='q"x'></x-quoted>
  <div id="out"></div>
  <script>
    const report = (root) => root.querySelectorAll('button').forEach(b => b.onclick = () => {
      document.getElementById('out').textContent = 'shadow:' + b.textContent;
    });
    customElements.define('x-inner', class extends HTMLElement {
      constructor() {
        super();
        const r = this.attachShadow({ mode: 'open' });
        r.innerHTML = '<button>Deep</button>';
        report(r);
      }
    });
    const r = document.querySelector('x-panel').attachShadow({ mode: 'open' });
    r.innerHTML = '<button>Save</button><section><div><button>X</button><button>Y</button></div></section>' +
      '<div><button>A</button><button>B</button></div><div><button>C</button></div><x-inner></x-inner><input type="file">';
    report(r);
    const sp = document.querySelector('x-spaced').attachShadow({ mode: 'open' });
    sp.innerHTML = '<button>Spaced</button>';
    report(sp);
    const q = document.querySelector('x-quoted').attachShadow({ mode: 'open' });
    q.innerHTML = '<button>Quoted</button>';
    report(q);
  </script>
</body></html>`;

const SHADOW_HTML = `<html><head><title>Shadow</title></head><body>
  <my-card></my-card>
  <div id="out"></div>
  <script>
    class NestedEl extends HTMLElement {
      constructor() {
        super();
        this.attachShadow({ mode: 'open' }).innerHTML =
          '<button id="deep">Deep Button</button>';
      }
    }
    class MyCard extends HTMLElement {
      constructor() {
        super();
        const r = this.attachShadow({ mode: 'open' });
        r.innerHTML = '<p>shadow text</p><input id="i"><button id="b">Shadow Button</button><nested-el></nested-el>';
        r.getElementById('b').onclick = () => {
          document.getElementById('out').textContent = 'clicked:' + r.getElementById('i').value;
        };
      }
    }
    customElements.define('nested-el', NestedEl);
    customElements.define('my-card', MyCard);
  </script>
</body></html>`;

const routes = {
  '/': HOME_HTML,
  '/shadow': SHADOW_HTML,
  '/shadow-refs': SHADOW_REFS_HTML,
  '/frames': FRAMES_HTML,
  '/frame-inner': FRAME_INNER_HTML,
  '/login': LOGIN_HTML,
  '/secure': SECURE_HTML,
  '/checkboxes': CHECKBOXES_HTML,
  '/hovers': HOVERS_HTML,
  '/dropdown': DROPDOWN_HTML,
  '/inputs': INPUTS_HTML,
  '/dynamic_loading/1': DYNAMIC_LOADING_HTML,
  '/add_remove_elements/': ADD_REMOVE_HTML,
  '/example': EXAMPLE_HTML,
  '/more-information': MORE_INFORMATION_HTML,
  '/find-images': FIND_IMAGES_HTML,
  '/find-canvas': FIND_CANVAS_HTML,
  '/selectors': SELECTORS_HTML,
  '/combined-locator': COMBINED_LOCATOR_HTML,
  '/type-filter': TYPE_FILTER_HTML,
  '/nested-scope': NESTED_SCOPE_HTML,
  '/form-redirect': FORM_REDIRECT_HTML,
};

function handleRequest(req, res) {
  // Form POST that 302s back to the same page. The redirect leaves a navigation
  // in flight just as a filmstrip screenshot is taken, which used to stall the
  // action for the full capture timeout (#289).
  if (req.url === '/form-redirect' && req.method === 'POST') {
    res.writeHead(302, { Location: '/form-redirect' });
    res.end();
    return;
  }

  // Handle login form POST → redirect to /secure
  if (req.url === '/authenticate' && req.method === 'POST') {
    res.writeHead(302, { 'Location': '/secure' });
    res.end();
    return;
  }

  if (req.url === '/fixtures/pattern.png' || req.url === '/fixtures/decoy.png') {
    res.writeHead(200, { 'Content-Type': 'image/png', 'Cache-Control': 'no-store' });
    res.end(req.url.endsWith('decoy.png') ? DECOY_PNG : PATTERN_PNG);
    return;
  }

  const html = routes[req.url] || routes['/'];
  res.writeHead(200, { 'Content-Type': 'text/html' });
  res.end(html);
}

/**
 * Create and start a test server on a random port.
 * Returns { server, baseURL }.
 */
function createTestServer() {
  return new Promise((resolve) => {
    const server = http.createServer(handleRequest);
    server.listen(0, '127.0.0.1', () => {
      const { port } = server.address();
      resolve({ server, baseURL: `http://127.0.0.1:${port}` });
    });
  });
}

module.exports = { createTestServer, makePatternPNG, PATTERN_PNG, DECOY_PNG };

// Standalone mode: print URL and serve
if (require.main === module) {
  const server = http.createServer(handleRequest);
  server.listen(0, '127.0.0.1', () => {
    const { port } = server.address();
    process.stdout.write(`http://127.0.0.1:${port}\n`);
  });
}
