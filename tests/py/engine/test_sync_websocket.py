"""Sync WebSocket monitoring tests — on_web_socket (6 sync tests)."""

import asyncio
import threading

import pytest

pytestmark = pytest.mark.capability("core")


@pytest.fixture(scope="module")
def sync_ws_echo_server():
    """Start a WebSocket echo server in a background thread. Returns ws:// URL."""
    import websockets

    loop = asyncio.new_event_loop()
    server = None
    port = None
    error = None
    # An Event instead of polling a variable: the old poll fell through
    # silently after its timeout (or when the thread died on an exception)
    # and yielded ws://127.0.0.1:None (#468).
    ready = threading.Event()

    async def _start():
        nonlocal server, port

        async def echo(websocket):
            async for message in websocket:
                await websocket.send(message)

        server = await websockets.serve(echo, "127.0.0.1", 0)
        port = server.sockets[0].getsockname()[1]

    def _run():
        nonlocal error
        asyncio.set_event_loop(loop)
        try:
            loop.run_until_complete(_start())
            ready.set()
            loop.run_forever()
        except Exception as exc:
            error = exc
            ready.set()
        finally:
            loop.close()

    t = threading.Thread(target=_run, daemon=True)
    t.start()

    if not ready.wait(30):
        pytest.fail("WebSocket echo server did not start within 30s")
    if error is not None:
        pytest.fail(f"WebSocket echo server failed to start: {error!r}")

    yield f"ws://127.0.0.1:{port}"

    async def _stop():
        server.close()
        # Without this, connection handlers still pending when the loop
        # stops surface later as "coroutine ignored GeneratorExit" noise.
        await server.wait_closed()

    # The loop belongs to the server thread; closing from here raced it
    # and left "Event loop is closed" noise in teardown.
    asyncio.run_coroutine_threadsafe(_stop(), loop).result(timeout=10)
    loop.call_soon_threadsafe(loop.stop)
    t.join(timeout=10)


def test_fires(sync_browser, test_server, sync_ws_echo_server):
    """on_web_socket fires when a WebSocket is opened."""
    vibe = sync_browser.new_page()
    vibe.go(test_server + "/ws-page")

    ws_connections = []
    vibe.on_web_socket(lambda ws: ws_connections.append(ws))

    vibe.evaluate(f"createWS('{sync_ws_echo_server}')")
    vibe.wait(1000)
    assert len(ws_connections) >= 1


def test_url(sync_browser, test_server, sync_ws_echo_server):
    """WebSocket info has correct URL."""
    vibe = sync_browser.new_page()
    vibe.go(test_server + "/ws-page")

    ws_connections = []
    vibe.on_web_socket(lambda ws: ws_connections.append(ws))

    vibe.evaluate(f"createWS('{sync_ws_echo_server}')")
    vibe.wait(1000)
    assert len(ws_connections) >= 1
    assert sync_ws_echo_server.replace("ws://", "") in ws_connections[0].url()


def test_on_message_sent(sync_browser, test_server, sync_ws_echo_server):
    """on_message fires for sent messages."""
    vibe = sync_browser.new_page()
    vibe.go(test_server + "/ws-page")

    ws_connections = []
    vibe.on_web_socket(lambda ws: ws_connections.append(ws))

    vibe.evaluate(f"window.__ws = createWS('{sync_ws_echo_server}')")
    vibe.wait(1000)

    messages = []
    if ws_connections:
        ws_connections[0].on_message(
            lambda data, info: messages.append({"data": data, "direction": info["direction"]})
        )

    vibe.evaluate("window.__ws.send('hello')")
    vibe.wait(500)
    sent = [m for m in messages if m["direction"] == "sent"]
    assert len(sent) >= 1
    assert sent[0]["data"] == "hello"


def test_on_close(sync_browser, test_server, sync_ws_echo_server):
    """on_close fires when WebSocket is closed."""
    vibe = sync_browser.new_page()
    vibe.go(test_server + "/ws-page")

    ws_connections = []
    vibe.on_web_socket(lambda ws: ws_connections.append(ws))

    vibe.evaluate(f"window.__ws = createWS('{sync_ws_echo_server}')")
    vibe.wait(1000)

    closed = []
    if ws_connections:
        ws_connections[0].on_close(lambda code, reason: closed.append({"code": code}))

    vibe.evaluate("window.__ws.close()")
    vibe.wait(500)
    assert len(closed) >= 1


def test_is_closed(sync_browser, test_server, sync_ws_echo_server):
    """is_closed returns True after WebSocket is closed."""
    vibe = sync_browser.new_page()
    vibe.go(test_server + "/ws-page")

    ws_connections = []
    vibe.on_web_socket(lambda ws: ws_connections.append(ws))

    vibe.evaluate(f"window.__ws = createWS('{sync_ws_echo_server}')")
    vibe.wait(1000)
    assert len(ws_connections) >= 1
    assert not ws_connections[0].is_closed()

    vibe.evaluate("window.__ws.close()")
    vibe.wait(500)
    assert ws_connections[0].is_closed()


def test_remove_listeners(sync_browser, test_server, sync_ws_echo_server):
    """remove_all_listeners('websocket') clears ws handlers."""
    vibe = sync_browser.new_page()
    vibe.go(test_server + "/ws-page")

    ws_connections = []
    vibe.on_web_socket(lambda ws: ws_connections.append(ws))
    vibe.remove_all_listeners("websocket")

    vibe.evaluate(f"createWS('{sync_ws_echo_server}')")
    vibe.wait(1000)
    assert len(ws_connections) == 0
