"""Tutorial HTTP server concurrency (#605).

The tutorial fixture server used single-threaded HTTPServer: one blocked
read starves the accept queue, so a browser connection that never sends a
request wedges the page load behind it until the client's 60s navigate
timeout. The server must keep answering while another connection sits
idle; the kernel accept queue is FIFO, so parking the idle connection
first deterministically reproduces the starvation on a serial server.
No browser, no binary.
"""

import socket
import urllib.request

from helpers.tutorial_runner import start_tutorial_server


def test_serves_request_while_another_connection_is_idle():
    server, base_url = start_tutorial_server({}, "tutorial body")
    idle = socket.create_connection(("127.0.0.1", server.server_address[1]))
    try:
        with urllib.request.urlopen(base_url + "/", timeout=10) as response:
            assert response.read() == b"tutorial body"
    finally:
        idle.close()
        server.shutdown()
