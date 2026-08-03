"""Basic tests for the Vibium Python client."""

import threading
from http.server import BaseHTTPRequestHandler, HTTPServer

from vibium import browser

EXAMPLE_HTML = b"""<html><head><title>Example Domain</title></head><body>
  <div>
    <h1>Example Domain</h1>
    <p>This domain is for use in illustrative examples in documents.</p>
    <p><a href="/more-information">More information...</a></p>
  </div>
</body></html>"""

MORE_INFORMATION_HTML = b"""<html><head><title>More information</title></head><body>
  <h1>More information</h1>
</body></html>"""


class _Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        body = MORE_INFORMATION_HTML if self.path == "/more-information" else EXAMPLE_HTML
        self.send_response(200)
        self.send_header("Content-Type", "text/html")
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *args):
        pass


def test_sync_api():
    """Test the synchronous API with new object model."""
    server = HTTPServer(("127.0.0.1", 0), _Handler)
    threading.Thread(target=server.serve_forever, daemon=True).start()
    base_url = f"http://127.0.0.1:{server.server_port}"

    bro = browser.start(headless=True)
    try:
        vibe = bro.new_page()
        vibe.go(f"{base_url}/example")

        # Test title
        title = vibe.title()
        assert title == "Example Domain", f"Expected 'Example Domain', got: {title}"

        # Test url
        url = vibe.url()
        assert "/example" in url, f"Expected URL with '/example', got: {url}"

        # Test find (CSS) and text
        h1 = vibe.find("h1")
        h1_text = h1.text()
        assert h1_text == "Example Domain", f"Expected 'Example Domain', got: {h1_text}"

        # Test find (semantic kwargs) — uses info.text from find result
        heading = vibe.find(role="heading")
        assert heading.info.text == "Example Domain", f"Expected 'Example Domain', got: {heading.info.text}"

        # Test find link and click
        link = vibe.find("a")
        link_text = link.text()
        assert link_text, f"Expected link text, got: {link_text}"

        # Test screenshot
        png = vibe.screenshot()
        assert len(png) > 1000, f"Screenshot too small: {len(png)} bytes"

        # Test eval
        result = vibe.eval("2 + 2")
        assert result == 4, f"Expected 4, got: {result}"

        # Test eval string
        doc_title = vibe.eval("document.title")
        assert doc_title == "Example Domain", f"Expected 'Example Domain', got: {doc_title}"

        # Test click
        link.click()

    finally:
        bro.stop()
        server.shutdown()


if __name__ == "__main__":
    test_sync_api()
    print("Python client test passed!")
