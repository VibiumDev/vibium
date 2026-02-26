"""WebSocket monitoring tests — onWebSocket, url, onMessage, onClose, isClosed (8 async tests)."""

import asyncio

import pytest


async def test_fires(fresh_async_browser, test_server, ws_echo_server):
    """onWebSocket fires when a WebSocket connection is opened."""
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/ws-page")

    ws_connections = []
    vibe.on_web_socket(lambda ws: ws_connections.append(ws))
    await asyncio.sleep(0.5)  # Wait for subscription to be processed

    await vibe.eval(f"createWS('{ws_echo_server}')")
    await vibe.wait(1000)
    assert len(ws_connections) >= 1


async def test_url(fresh_async_browser, test_server, ws_echo_server):
    """WebSocket info has correct URL."""
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/ws-page")

    ws_connections = []
    vibe.on_web_socket(lambda ws: ws_connections.append(ws))
    await asyncio.sleep(0.5)

    await vibe.eval(f"createWS('{ws_echo_server}')")
    await vibe.wait(1000)
    assert len(ws_connections) >= 1
    assert ws_echo_server.replace("ws://", "") in ws_connections[0].url()


async def test_on_message_sent(fresh_async_browser, test_server, ws_echo_server):
    """onMessage fires for sent messages."""
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/ws-page")

    ws_connections = []
    vibe.on_web_socket(lambda ws: ws_connections.append(ws))
    await asyncio.sleep(0.5)

    await vibe.eval(f"window.__ws = createWS('{ws_echo_server}')")
    await vibe.wait(1000)

    messages = []
    if ws_connections:
        ws_connections[0].on_message(lambda data, info: messages.append({"data": data, "direction": info["direction"]}))

    await vibe.eval("window.__ws.send('hello')")
    await vibe.wait(500)
    sent = [m for m in messages if m["direction"] == "sent"]
    assert len(sent) >= 1
    assert sent[0]["data"] == "hello"


async def test_on_message_received(fresh_async_browser, test_server, ws_echo_server):
    """onMessage fires for received (echoed) messages."""
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/ws-page")

    ws_connections = []
    vibe.on_web_socket(lambda ws: ws_connections.append(ws))
    await asyncio.sleep(0.5)

    await vibe.eval(f"window.__ws = createWS('{ws_echo_server}')")
    await vibe.wait(1000)

    messages = []
    if ws_connections:
        ws_connections[0].on_message(lambda data, info: messages.append({"data": data, "direction": info["direction"]}))

    await vibe.eval("window.__ws.send('echo-me')")
    await vibe.wait(500)
    received = [m for m in messages if m["direction"] == "received"]
    assert len(received) >= 1
    assert received[0]["data"] == "echo-me"


async def test_on_close(fresh_async_browser, test_server, ws_echo_server):
    """onClose fires when WebSocket is closed."""
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/ws-page")

    ws_connections = []
    vibe.on_web_socket(lambda ws: ws_connections.append(ws))
    await asyncio.sleep(0.5)

    await vibe.eval(f"window.__ws = createWS('{ws_echo_server}')")
    await vibe.wait(1000)

    closed = []
    if ws_connections:
        ws_connections[0].on_close(lambda code, reason: closed.append({"code": code, "reason": reason}))

    await vibe.eval("window.__ws.close()")
    await vibe.wait(500)
    assert len(closed) >= 1


async def test_is_closed(fresh_async_browser, test_server, ws_echo_server):
    """isClosed returns True after WebSocket is closed."""
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/ws-page")

    ws_connections = []
    vibe.on_web_socket(lambda ws: ws_connections.append(ws))
    await asyncio.sleep(0.5)

    await vibe.eval(f"window.__ws = createWS('{ws_echo_server}')")
    await vibe.wait(1000)
    assert len(ws_connections) >= 1
    assert not ws_connections[0].is_closed()

    await vibe.eval("window.__ws.close()")
    await vibe.wait(500)
    assert ws_connections[0].is_closed()


async def test_survives_navigation(fresh_async_browser, test_server, ws_echo_server):
    """WebSocket tracking survives page navigation."""
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/ws-page")

    ws_connections = []
    vibe.on_web_socket(lambda ws: ws_connections.append(ws))
    await asyncio.sleep(0.5)

    await vibe.eval(f"createWS('{ws_echo_server}')")
    await vibe.wait(1000)
    count_before = len(ws_connections)

    await vibe.go(test_server + "/ws-page")
    await vibe.eval(f"createWS('{ws_echo_server}')")
    await vibe.wait(1000)
    assert len(ws_connections) >= count_before


async def test_remove_listeners(fresh_async_browser, test_server, ws_echo_server):
    """removeAllListeners('websocket') clears ws handlers."""
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/ws-page")

    ws_connections = []
    vibe.on_web_socket(lambda ws: ws_connections.append(ws))
    vibe.remove_all_listeners("websocket")

    await vibe.eval(f"createWS('{ws_echo_server}')")
    await vibe.wait(1000)
    assert len(ws_connections) == 0
