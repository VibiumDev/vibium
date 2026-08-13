"""WebSocket monitoring tests — onWebSocket, url, onMessage, onClose, isClosed (8 async tests)."""

import pytest

pytestmark = pytest.mark.capability("core")


async def _wait_for(vibe, condition, timeout_ms=15000):
    """Poll for an event-driven condition; a fixed sleep flakes under load."""
    waited = 0
    while not condition() and waited < timeout_ms:
        await vibe.wait(100)
        waited += 100


async def _subscribed(vibe, ws_echo_server, ws_connections):
    """Prove the onWebSocket subscription is active before the real test runs.

    The client sends the subscribe command without awaiting it, so an early
    createWS can race it, and the one-shot created event is then lost for
    good. Probe with throwaway sockets until one is tracked, then drain
    stragglers and reset.
    """
    for _ in range(100):
        await vibe.evaluate(f"createWS('{ws_echo_server}').close()")
        await _wait_for(vibe, lambda: ws_connections, timeout_ms=200)
        if ws_connections:
            await vibe.wait(300)
            ws_connections.clear()
            return
    raise AssertionError("onWebSocket subscription never became active")


async def _ws_open(vibe):
    """Wait for window.__ws to reach OPEN; send() on CONNECTING throws."""
    for _ in range(150):
        if await vibe.evaluate("window.__ws && window.__ws.readyState === 1"):
            return
        await vibe.wait(100)
    raise AssertionError("websocket never reached OPEN")


async def test_fires(fresh_async_browser, test_server, ws_echo_server):
    """onWebSocket fires when a WebSocket connection is opened."""
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/ws-page")

    ws_connections = []
    vibe.on_web_socket(lambda ws: ws_connections.append(ws))
    await _subscribed(vibe, ws_echo_server, ws_connections)

    await vibe.evaluate(f"createWS('{ws_echo_server}')")
    await _wait_for(vibe, lambda: ws_connections)
    assert len(ws_connections) >= 1


async def test_url(fresh_async_browser, test_server, ws_echo_server):
    """WebSocket info has correct URL."""
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/ws-page")

    ws_connections = []
    vibe.on_web_socket(lambda ws: ws_connections.append(ws))
    await _subscribed(vibe, ws_echo_server, ws_connections)

    await vibe.evaluate(f"createWS('{ws_echo_server}')")
    await _wait_for(vibe, lambda: ws_connections)
    assert len(ws_connections) >= 1
    assert ws_echo_server.replace("ws://", "") in ws_connections[0].url()


async def test_on_message_sent(fresh_async_browser, test_server, ws_echo_server):
    """onMessage fires for sent messages."""
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/ws-page")

    ws_connections = []
    vibe.on_web_socket(lambda ws: ws_connections.append(ws))
    await _subscribed(vibe, ws_echo_server, ws_connections)

    await vibe.evaluate(f"window.__ws = createWS('{ws_echo_server}')")
    await _wait_for(vibe, lambda: ws_connections)
    assert len(ws_connections) >= 1

    messages = []
    ws_connections[0].on_message(lambda data, info: messages.append({"data": data, "direction": info["direction"]}))

    await _ws_open(vibe)
    await vibe.evaluate("window.__ws.send('hello')")
    await _wait_for(vibe, lambda: [m for m in messages if m["direction"] == "sent"])
    sent = [m for m in messages if m["direction"] == "sent"]
    assert len(sent) >= 1
    assert sent[0]["data"] == "hello"


async def test_on_message_received(fresh_async_browser, test_server, ws_echo_server):
    """onMessage fires for received (echoed) messages."""
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/ws-page")

    ws_connections = []
    vibe.on_web_socket(lambda ws: ws_connections.append(ws))
    await _subscribed(vibe, ws_echo_server, ws_connections)

    await vibe.evaluate(f"window.__ws = createWS('{ws_echo_server}')")
    await _wait_for(vibe, lambda: ws_connections)
    assert len(ws_connections) >= 1

    messages = []
    ws_connections[0].on_message(lambda data, info: messages.append({"data": data, "direction": info["direction"]}))

    await _ws_open(vibe)
    await vibe.evaluate("window.__ws.send('echo-me')")
    await _wait_for(vibe, lambda: [m for m in messages if m["direction"] == "received"])
    received = [m for m in messages if m["direction"] == "received"]
    assert len(received) >= 1
    assert received[0]["data"] == "echo-me"


async def test_on_close(fresh_async_browser, test_server, ws_echo_server):
    """onClose fires when WebSocket is closed."""
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/ws-page")

    ws_connections = []
    vibe.on_web_socket(lambda ws: ws_connections.append(ws))
    await _subscribed(vibe, ws_echo_server, ws_connections)

    await vibe.evaluate(f"window.__ws = createWS('{ws_echo_server}')")
    await _wait_for(vibe, lambda: ws_connections)
    assert len(ws_connections) >= 1

    closed = []
    ws_connections[0].on_close(lambda code, reason: closed.append({"code": code, "reason": reason}))

    await vibe.evaluate("window.__ws.close()")
    await _wait_for(vibe, lambda: closed)
    assert len(closed) >= 1


async def test_is_closed(fresh_async_browser, test_server, ws_echo_server):
    """isClosed returns True after WebSocket is closed."""
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/ws-page")

    ws_connections = []
    vibe.on_web_socket(lambda ws: ws_connections.append(ws))
    await _subscribed(vibe, ws_echo_server, ws_connections)

    await vibe.evaluate(f"window.__ws = createWS('{ws_echo_server}')")
    await _wait_for(vibe, lambda: ws_connections)
    assert len(ws_connections) >= 1
    assert not ws_connections[0].is_closed()

    await vibe.evaluate("window.__ws.close()")
    await _wait_for(vibe, lambda: ws_connections[0].is_closed())
    assert ws_connections[0].is_closed()


async def test_survives_navigation(fresh_async_browser, test_server, ws_echo_server):
    """WebSocket tracking survives page navigation."""
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/ws-page")

    ws_connections = []
    vibe.on_web_socket(lambda ws: ws_connections.append(ws))
    await _subscribed(vibe, ws_echo_server, ws_connections)

    await vibe.evaluate(f"createWS('{ws_echo_server}')")
    await _wait_for(vibe, lambda: ws_connections)
    count_before = len(ws_connections)

    await vibe.go(test_server + "/ws-page")
    await vibe.evaluate(f"createWS('{ws_echo_server}')")
    await vibe.wait(1000)
    assert len(ws_connections) >= count_before


async def test_remove_listeners(fresh_async_browser, test_server, ws_echo_server):
    """removeAllListeners('websocket') clears ws handlers."""
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/ws-page")

    ws_connections = []
    vibe.on_web_socket(lambda ws: ws_connections.append(ws))
    vibe.remove_all_listeners("websocket")

    await vibe.evaluate(f"createWS('{ws_echo_server}')")
    await vibe.wait(1000)
    assert len(ws_connections) == 0
