"""Parses annotated markdown tutorials and provides test runners.

Annotations (same format as the JS tutorial runner):
    <!-- helpers -->              — next code block defines shared helper functions
    <!-- server -->               — next code block defines a test server
    <!-- test: async "name" -->   — next code block is an async test
    <!-- test: sync "name" -->    — next code block is a sync test

Server blocks:
    Python code that defines ``ROUTES`` (dict mapping paths to
    ``(status, headers_dict, body_bytes)`` tuples) and ``DEFAULT_BODY``
    (HTML string served for unmatched paths).  ``create_tutorial_server()``
    starts an HTTP server from these definitions.
"""

import re
import textwrap
from pathlib import Path

# tests/py/helpers/ -> project root
PROJECT_ROOT = Path(__file__).resolve().parents[3]


def extract_blocks(md_path):
    """Parse markdown and extract annotated ``python`` code blocks."""
    content = (PROJECT_ROOT / md_path).read_text()
    lines = content.split("\n")
    blocks = []

    pending = None
    in_code_block = False
    is_annotated = False
    code_lines = []

    for line in lines:
        if not in_code_block:
            if re.match(r"<!--\s*helpers\s*-->", line):
                pending = {"type": "helpers"}
                continue

            if re.match(r"<!--\s*server\s*-->", line):
                pending = {"type": "server"}
                continue

            m = re.match(r'<!--\s*test:\s*(async|sync)\s+"([^"]+)"\s*-->', line)
            if m:
                pending = {"type": "test", "mode": m.group(1), "name": m.group(2)}
                continue

            if re.match(r"^```python\s*$", line):
                in_code_block = True
                if pending:
                    is_annotated = True
                    code_lines = []
                else:
                    is_annotated = False
                continue
        else:
            if re.match(r"^```\s*$", line):
                in_code_block = False
                if is_annotated and pending:
                    blocks.append({**pending, "code": "\n".join(code_lines)})
                    pending = None
                continue
            if is_annotated:
                code_lines.append(line)

    return blocks


def get_tutorial_tests(md_path, mode):
    """Return ``[(name, helpers, code), ...]`` for *pytest.mark.parametrize*."""
    blocks = extract_blocks(md_path)
    helpers = ""
    tests = []

    # Collect all helpers first (they may appear anywhere in the file)
    for block in blocks:
        if block["type"] == "helpers":
            helpers += block["code"] + "\n"

    for block in blocks:
        if block.get("mode") != mode:
            continue
        if block["type"] == "test":
            tests.append((block["name"], helpers, block["code"]))

    return tests


def get_server_code(md_path):
    """Return the ``<!-- server -->`` block code, or *None* if absent."""
    for block in extract_blocks(md_path):
        if block["type"] == "server":
            return block["code"]
    return None


def start_tutorial_server(routes, default_body=""):
    """Start an HTTP server from a *routes* dict.

    *routes* maps paths to ``(status, headers_dict, body_bytes)`` tuples.
    Unmatched paths return *default_body* as ``text/html``.

    Returns ``(server, base_url)``.  Caller must call ``server.shutdown()``.
    """
    import threading
    from http.server import HTTPServer, BaseHTTPRequestHandler

    class Handler(BaseHTTPRequestHandler):
        def do_GET(self):
            if self.path in routes:
                status, headers, body = routes[self.path]
                self.send_response(status)
                for k, v in headers.items():
                    self.send_header(k, v)
                self.end_headers()
                self.wfile.write(body if isinstance(body, bytes) else body.encode())
            else:
                self.send_response(200)
                self.send_header("Content-Type", "text/html")
                self.end_headers()
                self.wfile.write(
                    default_body.encode()
                    if isinstance(default_body, str)
                    else default_body
                )

        def log_message(self, format, *args):
            pass

    server = HTTPServer(("127.0.0.1", 0), Handler)
    port = server.server_address[1]
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    return server, f"http://127.0.0.1:{port}"


def create_tutorial_server(md_path):
    """Start an HTTP server defined by the tutorial's ``<!-- server -->`` block.

    Returns ``(server, base_url)``.  Caller must call ``server.shutdown()``.
    Returns ``(None, None)`` when no server block is present.
    """
    code = get_server_code(md_path)
    if not code:
        return None, None

    ns = {}
    exec(compile(code, "<server>", "exec"), ns)
    return start_tutorial_server(ns.get("ROUTES", {}), ns.get("DEFAULT_BODY", ""))


async def run_async_block(code, vibe, base_url=None):
    """Exec a block of async Python code with *vibe* as the page object."""
    indented = textwrap.indent(code, "    ")
    wrapped = f"async def _run(vibe, base_url):\n{indented}\n"
    ns = {}
    exec(compile(wrapped, "<tutorial>", "exec"), ns)
    await ns["_run"](vibe, base_url)


def run_sync_block(code, vibe, base_url=None):
    """Exec a block of sync Python code with *vibe* as the page object."""
    exec(compile(code, "<tutorial>", "exec"), {"vibe": vibe, "base_url": base_url})


async def run_async_standalone(code, base_url=None):
    """Exec a standalone async block that manages its own browser lifecycle."""
    indented = textwrap.indent(code, "    ")
    wrapped = f"async def _run(base_url):\n{indented}\n"
    ns = {}
    exec(compile(wrapped, "<tutorial>", "exec"), ns)
    await ns["_run"](base_url)


def run_sync_standalone(code, base_url=None):
    """Exec a standalone sync block that manages its own browser lifecycle."""
    exec(compile(code, "<tutorial>", "exec"), {"base_url": base_url})
