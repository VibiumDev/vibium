"""Network & dialog tests — route fulfill/continue/abort, onRequest/onResponse, dialogs (21 async tests).

Uses a module-scoped browser with new_page() per test for isolation.
"""

import asyncio
import pytest

pytestmark = pytest.mark.capability("core")
import pytest_asyncio


# Module-scoped browser for all network/dialog tests
@pytest_asyncio.fixture(scope="module", loop_scope="module")
async def net_browser():
    from vibium.async_api import browser
    bro = await browser.start(headless=True)
    yield bro
    await bro.stop()


# Helper: fire-and-forget an async route action on the running event loop
def _fire(coro):
    asyncio.get_running_loop().create_task(coro)


# --- Route ---

@pytest.mark.capability("network")
async def test_route_abort(net_browser, test_server):
    vibe = await net_browser.new_page()
    await vibe.go(test_server + "/fetch")

    def handler(route):
        _fire(route.abort())

    await vibe.route("**/api/data", handler)
    await vibe.evaluate("doFetch().catch(() => document.getElementById('result').textContent = 'aborted')")
    await vibe.wait(500)
    result = await (await vibe.find("#result")).text()
    assert "aborted" in result or result == ""
    await vibe.close()


@pytest.mark.capability("network")
async def test_route_fulfill(net_browser, test_server):
    vibe = await net_browser.new_page()
    await vibe.go(test_server)

    def handler(route):
        _fire(route.fulfill(status=200, body='{"mocked":true}', content_type="application/json"))

    await vibe.route("**/json", handler)
    result = await vibe.evaluate(f"fetch('{test_server}/json').then(r => r.json())")
    assert result["mocked"] is True
    await vibe.close()


@pytest.mark.capability("network")
async def test_route_fulfill_headers(net_browser, test_server):
    vibe = await net_browser.new_page()
    await vibe.go(test_server)

    def handler(route):
        _fire(route.fulfill(
            status=201,
            body="custom body",
            content_type="text/plain",
            headers={"X-Custom": "test-value"},
        ))

    await vibe.route("**/text", handler)
    result = await vibe.evaluate(
        f"fetch('{test_server}/text')"
        ".then(r => r.text().then(body => ({ status: r.status, body, custom: r.headers.get('X-Custom') })))"
    )
    assert result["status"] == 201
    assert result["body"] == "custom body"
    assert result["custom"] == "test-value"
    await vibe.close()


@pytest.mark.capability("network")
async def test_route_continue(net_browser, test_server):
    vibe = await net_browser.new_page()
    await vibe.go(test_server)

    def handler(route):
        _fire(route.continue_())

    await vibe.route("**/json", handler)
    result = await vibe.evaluate(f"fetch('{test_server}/json').then(r => r.json())")
    assert result["name"] == "vibium"
    await vibe.close()


@pytest.mark.capability("network")
async def test_unroute(net_browser, test_server):
    vibe = await net_browser.new_page()
    await vibe.go(test_server)

    def handler(route):
        _fire(route.abort())

    await vibe.route("**/json", handler)
    await vibe.unroute("**/json")
    result = await vibe.evaluate(f"fetch('{test_server}/json').then(r => r.json())")
    assert result["name"] == "vibium"
    await vibe.close()


# --- Events ---

@pytest.mark.capability("network")
async def test_on_request(net_browser, test_server):
    vibe = await net_browser.new_page()
    await vibe.go(test_server + "/fetch")
    urls = []
    vibe.on_request(lambda req: urls.append(req.url()))
    await vibe.evaluate("doFetch()")
    await vibe.wait(500)
    assert any("api/data" in u for u in urls)
    await vibe.close()


@pytest.mark.capability("network")
async def test_on_response(net_browser, test_server):
    vibe = await net_browser.new_page()
    await vibe.go(test_server + "/fetch")
    statuses = []
    vibe.on_response(lambda resp: statuses.append(resp.status()))
    await vibe.evaluate("doFetch()")
    await vibe.wait(500)
    assert 200 in statuses
    await vibe.close()


@pytest.mark.capability("network")
async def test_request_method_headers(net_browser, test_server):
    vibe = await net_browser.new_page()
    await vibe.go(test_server + "/fetch")
    captured = {}

    def handler(req):
        if "api/data" in req.url():
            captured["method"] = req.method()
            captured["headers"] = req.headers()

    vibe.on_request(handler)
    await vibe.evaluate("doFetch()")
    await vibe.wait(500)
    assert captured.get("method") == "GET"
    assert isinstance(captured.get("headers"), dict)
    await vibe.close()


@pytest.mark.capability("network")
async def test_response_url_status(net_browser, test_server):
    vibe = await net_browser.new_page()
    await vibe.go(test_server + "/fetch")
    captured = {}

    def handler(resp):
        if "api/data" in resp.url():
            captured["url"] = resp.url()
            captured["status"] = resp.status()

    vibe.on_response(handler)
    await vibe.evaluate("doFetch()")
    await vibe.wait(500)
    assert "api/data" in captured.get("url", "")
    assert captured.get("status") == 200
    await vibe.close()


# --- Waiters ---

@pytest.mark.capability("network")
async def test_capture_response(net_browser, test_server):
    vibe = await net_browser.new_page()
    await vibe.go(test_server + "/fetch")
    await vibe.evaluate("setTimeout(() => doFetch(), 100)")
    resp = await vibe.capture.response("**/api/data", timeout=5000)
    assert resp.status() == 200
    await vibe.close()


@pytest.mark.capability("network")
async def test_capture_request(net_browser, test_server):
    vibe = await net_browser.new_page()
    await vibe.go(test_server + "/fetch")
    await vibe.evaluate("setTimeout(() => doFetch(), 100)")
    req = await vibe.capture.request("**/api/data", timeout=5000)
    assert "api/data" in req.url()
    await vibe.close()


# --- Response body ---

@pytest.mark.capability("network")
async def test_body_text(net_browser, test_server):
    vibe = await net_browser.new_page()
    await vibe.go(test_server)
    await vibe.evaluate("setTimeout(() => fetch('/text'), 100)")
    resp = await vibe.capture.response("**/text", timeout=5000)
    body = await resp.body()
    assert body is not None
    assert "hello world" in body
    await vibe.close()


@pytest.mark.capability("network")
async def test_body_json(net_browser, test_server):
    vibe = await net_browser.new_page()
    await vibe.go(test_server)
    await vibe.evaluate("setTimeout(() => fetch('/json'), 100)")
    resp = await vibe.capture.response("**/json", timeout=5000)
    data = await resp.json()
    assert data is not None
    assert data["name"] == "vibium"
    await vibe.close()


@pytest.mark.capability("network")
async def test_body_via_expect(net_browser, test_server):
    vibe = await net_browser.new_page()
    await vibe.go(test_server + "/fetch")
    await vibe.evaluate("setTimeout(() => doFetch(), 100)")
    resp = await vibe.capture.response("**/api/data", timeout=5000)
    body = await resp.body()
    assert body is not None
    assert "real data" in body
    await vibe.close()


@pytest.mark.capability("network")
async def test_json_via_expect(net_browser, test_server):
    vibe = await net_browser.new_page()
    await vibe.go(test_server + "/fetch")
    await vibe.evaluate("setTimeout(() => doFetch(), 100)")
    resp = await vibe.capture.response("**/api/data", timeout=5000)
    data = await resp.json()
    assert data["count"] == 42
    await vibe.close()


# --- Dialogs ---

@pytest.mark.capability("dialogs")
async def test_alert(net_browser, test_server):
    vibe = await net_browser.new_page()
    await vibe.go(test_server)
    messages = []

    def handler(dialog):
        messages.append(dialog.message())
        _fire(dialog.accept())

    vibe.on_dialog(handler)
    await vibe.evaluate('alert("Hello from test")')
    assert len(messages) == 1
    assert messages[0] == "Hello from test"
    await vibe.close()


@pytest.mark.capability("dialogs")
async def test_confirm_accept(net_browser, test_server):
    vibe = await net_browser.new_page()
    await vibe.go(test_server)
    vibe.on_dialog(lambda d: _fire(d.accept()))
    result = await vibe.evaluate('confirm("Are you sure?")')
    assert result is True
    await vibe.close()


@pytest.mark.capability("dialogs")
async def test_confirm_dismiss(net_browser, test_server):
    vibe = await net_browser.new_page()
    await vibe.go(test_server)
    vibe.on_dialog(lambda d: _fire(d.dismiss()))
    result = await vibe.evaluate('confirm("Are you sure?")')
    assert result is False
    await vibe.close()


@pytest.mark.capability("dialogs")
async def test_prompt(net_browser, test_server):
    vibe = await net_browser.new_page()
    await vibe.go(test_server)

    def handler(dialog):
        assert dialog.type() == "prompt"
        _fire(dialog.accept("my answer"))

    vibe.on_dialog(handler)
    result = await vibe.evaluate('prompt("Enter name:")')
    assert result == "my answer"
    await vibe.close()


@pytest.mark.capability("dialogs")
async def test_auto_dismiss(net_browser, test_server):
    vibe = await net_browser.new_page()
    await vibe.go(test_server)
    # No handler — should auto-dismiss
    result = await vibe.evaluate('confirm("Auto dismiss?")')
    assert result is False
    await vibe.close()


# --- Expect navigation ---

@pytest.mark.capability("navigation-capture")
async def test_capture_navigation(net_browser, test_server):
    vibe = await net_browser.new_page()
    await vibe.go(test_server + "/nav-test")
    link = await vibe.find("#link")
    async with vibe.capture.navigation() as info:
        await link.click()
    assert info.value is not None
    assert "/page2" in info.value
    await vibe.close()


# --- Expect download ---

@pytest.mark.capability("downloads")
async def test_capture_download(net_browser, test_server):
    vibe = await net_browser.new_page()
    await vibe.go(test_server + "/download")
    link = await vibe.find("#download-link")
    async with vibe.capture.download() as info:
        await link.click()
    assert info.value is not None
    assert info.value.url().endswith("/download-file") or "/download-file" in info.value.url()
    assert info.value.suggested_filename() == "test.txt"
    await vibe.close()


# --- Expect dialog ---

@pytest.mark.capability("dialogs")
async def test_capture_dialog(net_browser, test_server):
    vibe = await net_browser.new_page()
    await vibe.go(test_server)
    async with vibe.capture.dialog() as info:
        await vibe.evaluate('setTimeout(() => alert("Hello from expect"), 50)')
    assert info.value is not None
    assert info.value.type() == "alert"
    assert info.value.message() == "Hello from expect"
    await vibe.close()


# --- Expect event ---

@pytest.mark.capability("network")
async def test_capture_event_response(net_browser, test_server):
    vibe = await net_browser.new_page()
    await vibe.go(test_server + "/fetch")
    await vibe.evaluate("setTimeout(() => doFetch(), 100)")
    result = await vibe.capture.event("response", timeout=5000)
    assert result is not None
    await vibe.close()


# --- Checkpoint ---

@pytest.mark.capability("network", "dialogs")
async def test_checkpoint(net_browser, test_server):
    """Route + response + dialog together."""
    vibe = await net_browser.new_page()

    # Route: continue all requests
    def route_handler(route):
        _fire(route.continue_())

    await vibe.route("**", route_handler)

    # Track responses
    response_urls = []
    vibe.on_response(lambda resp: response_urls.append(resp.url()))

    # Navigate
    await vibe.go(test_server)
    assert len(response_urls) >= 1

    # Dialog
    vibe.on_dialog(lambda d: _fire(d.accept()))
    result = await vibe.evaluate('confirm("checkpoint?")')
    assert result is True

    await vibe.unroute("**")
    await vibe.close()
