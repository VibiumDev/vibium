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

const routes = {
  '/': HOME_HTML,
  '/login': LOGIN_HTML,
  '/secure': SECURE_HTML,
  '/checkboxes': CHECKBOXES_HTML,
  '/hovers': HOVERS_HTML,
  '/dropdown': DROPDOWN_HTML,
  '/inputs': INPUTS_HTML,
  '/dynamic_loading/1': DYNAMIC_LOADING_HTML,
  '/add_remove_elements/': ADD_REMOVE_HTML,
};

function handleRequest(req, res) {
  // Handle login form POST → redirect to /secure
  if (req.url === '/authenticate' && req.method === 'POST') {
    res.writeHead(302, { 'Location': '/secure' });
    res.end();
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

module.exports = { createTestServer };

// Standalone mode: print URL and serve
if (require.main === module) {
  const server = http.createServer(handleRequest);
  server.listen(0, '127.0.0.1', () => {
    const { port } = server.address();
    process.stdout.write(`http://127.0.0.1:${port}\n`);
  });
}
