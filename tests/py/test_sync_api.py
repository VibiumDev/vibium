"""Comprehensive sync API tests (79 tests).

Tests all sync-specific patterns: basic lifecycle, navigation, evaluation,
finding, interaction, state, keyboard/mouse, clock, viewport/emulation,
context, dialog handlers, route handlers, console/error collection,
expect request/response, and full checkpoint.
"""

import time

import pytest

from vibium import browser, Browser, Page, Element, ElementList, BrowserContext


# ---------------------------------------------------------------------------
# Module-scoped browser and server
# ---------------------------------------------------------------------------

@pytest.fixture(scope="module")
def bro():
    b = browser.launch(headless=True)
    yield b
    b.close()


@pytest.fixture
def vibe(bro, test_server):
    v = bro.page()
    v.go(test_server)
    return v


# ===========================================================================
# Browser lifecycle (2)
# ===========================================================================

def test_launch_and_close():
    b = browser.launch(headless=True)
    assert isinstance(b, Browser)
    b.close()


def test_page_returns_default(bro):
    vibe = bro.page()
    assert isinstance(vibe, Page)


# ===========================================================================
# Multi-page (1)
# ===========================================================================

def test_new_page_creates_tab(bro):
    p = bro.new_page()
    assert isinstance(p, Page)
    pages = bro.pages()
    assert len(pages) >= 2


# ===========================================================================
# Navigation (6)
# ===========================================================================

def test_go(vibe, test_server):
    assert test_server in vibe.url()


def test_url(vibe, test_server):
    url = vibe.url()
    assert url.startswith("http")


def test_title(vibe):
    assert vibe.title() == "Test App"


def test_content(vibe):
    html = vibe.content()
    assert "Welcome to test-app" in html


def test_back_forward(vibe, test_server):
    vibe.go(test_server + "/subpage")
    assert vibe.title() == "Subpage"
    vibe.back()
    assert vibe.title() == "Test App"
    vibe.forward()
    assert vibe.title() == "Subpage"


def test_reload(vibe):
    vibe.reload()
    assert vibe.title() == "Test App"


# ===========================================================================
# Screenshots (2)
# ===========================================================================

def test_screenshot_png(vibe):
    data = vibe.screenshot()
    assert isinstance(data, bytes)
    assert data[:4] == b"\x89PNG"


def test_pdf(bro, test_server):
    vibe = bro.page()
    vibe.go(test_server)
    data = vibe.pdf()
    assert isinstance(data, bytes)
    assert data[:2] == b"%P"


# ===========================================================================
# Evaluation (2)
# ===========================================================================

def test_eval_expression(bro, test_server):
    vibe = bro.page()
    vibe.go(test_server + "/eval")
    result = vibe.eval("window.testVal")
    assert result == 42


def test_eval_computed_value(vibe):
    result = vibe.eval("2 + 3")
    assert result == 5


# ===========================================================================
# Finding (4)
# ===========================================================================

def test_find_css(vibe):
    el = vibe.find("h1")
    assert isinstance(el, Element)


def test_find_semantic(vibe):
    el = vibe.find(role="heading")
    assert isinstance(el, Element)


def test_find_all(bro, test_server):
    vibe = bro.page()
    vibe.go(test_server + "/links")
    els = vibe.find_all(".link")
    assert isinstance(els, ElementList)
    assert els.count() == 4


def test_find_auto_waits(bro, test_server):
    vibe = bro.page()
    vibe.go(test_server + "/dynamic-loading")
    el = vibe.find("#loaded", timeout=5000)
    assert isinstance(el, Element)


# ===========================================================================
# ElementList (3)
# ===========================================================================

def test_count_first_last_nth(bro, test_server):
    vibe = bro.page()
    vibe.go(test_server + "/links")
    els = vibe.find_all(".link")
    assert els.count() == 4
    first = els.first()
    assert first.info.text == "Link 1"
    last = els.last()
    assert last.info.text == "Link 4"
    second = els.nth(1)
    assert second.info.text == "Link 2"


def test_iteration(bro, test_server):
    vibe = bro.page()
    vibe.go(test_server + "/links")
    els = vibe.find_all(".link")
    texts = [el.info.text for el in els]
    assert len(texts) == 4
    assert texts[0] == "Link 1"


def test_filter(bro, test_server):
    vibe = bro.page()
    vibe.go(test_server + "/links")
    els = vibe.find_all(".link")
    filtered = els.filter(has_text="Link 4")
    assert filtered.count() >= 1


# ===========================================================================
# Interaction (7)
# ===========================================================================

def test_click_navigates(bro, test_server):
    vibe = bro.page()
    vibe.go(test_server)
    link = vibe.find('a[href="/subpage"]')
    link.click()
    vibe.wait_until.url("**/subpage")
    assert vibe.title() == "Subpage"


def test_fill_and_value(bro, test_server):
    vibe = bro.page()
    vibe.go(test_server + "/inputs")
    inp = vibe.find("#text-input")
    inp.fill("hello world")
    assert inp.value() == "hello world"


def test_type_appends(bro, test_server):
    vibe = bro.page()
    vibe.go(test_server + "/inputs")
    inp = vibe.find("#text-input")
    inp.fill("")
    inp.type("abc")
    assert inp.value() == "abc"


def test_check_uncheck(bro, test_server):
    vibe = bro.page()
    vibe.go(test_server + "/form")
    cb = vibe.find("#agree")
    cb.check()
    assert cb.is_checked()
    cb.uncheck()
    assert not cb.is_checked()


def test_select_option(bro, test_server):
    vibe = bro.page()
    vibe.go(test_server + "/form")
    sel = vibe.find("#color")
    sel.select_option("blue")
    assert sel.value() == "blue"


def test_hover(bro, test_server):
    vibe = bro.page()
    vibe.go(test_server)
    el = vibe.find("h1")
    el.hover()  # should not throw


def test_press(bro, test_server):
    vibe = bro.page()
    vibe.go(test_server + "/inputs")
    inp = vibe.find("#text-input")
    inp.fill("")
    inp.focus()
    inp.press("a")
    assert inp.value() == "a"


# ===========================================================================
# State (8)
# ===========================================================================

def test_text(vibe):
    el = vibe.find("#info")
    assert el.text() == "Some info text"


def test_inner_text(vibe):
    el = vibe.find("#info")
    assert "Some info text" in el.inner_text()


def test_html(vibe):
    el = vibe.find("#info")
    assert el.html() == "Some info text"


def test_attr(bro, test_server):
    vibe = bro.page()
    vibe.go(test_server + "/links")
    link = vibe.find(".link")
    assert link.attr("class") == "link"


def test_bounds(vibe):
    el = vibe.find("h1")
    box = el.bounds()
    assert box.width > 0
    assert box.height > 0


def test_is_visible(vibe):
    el = vibe.find("h1")
    assert el.is_visible()


def test_is_enabled(bro, test_server):
    vibe = bro.page()
    vibe.go(test_server + "/form")
    btn = vibe.find("button")
    assert btn.is_enabled()


def test_is_editable(bro, test_server):
    vibe = bro.page()
    vibe.go(test_server + "/inputs")
    inp = vibe.find("#text-input")
    assert inp.is_editable()


# ===========================================================================
# Scoped find (2)
# ===========================================================================

def test_element_find_scoped(bro, test_server):
    vibe = bro.page()
    vibe.go(test_server + "/links")
    nested = vibe.find("#nested")
    span = nested.find(".inner")
    assert "span" in span.info.text.lower() or span.info.tag == "span"


def test_element_find_all_scoped(bro, test_server):
    vibe = bro.page()
    vibe.go(test_server + "/links")
    nested = vibe.find("#nested")
    spans = nested.find_all(".inner")
    assert spans.count() == 2


# ===========================================================================
# Keyboard / Mouse (3)
# ===========================================================================

def test_keyboard_type(bro, test_server):
    vibe = bro.page()
    vibe.go(test_server + "/inputs")
    inp = vibe.find("#text-input")
    inp.fill("")
    inp.focus()
    vibe.keyboard.type("typed")
    assert inp.value() == "typed"


def test_keyboard_press(bro, test_server):
    vibe = bro.page()
    vibe.go(test_server + "/inputs")
    inp = vibe.find("#text-input")
    inp.fill("")
    inp.focus()
    vibe.keyboard.press("a")
    assert inp.value() == "a"


def test_mouse_click(bro, test_server):
    vibe = bro.page()
    vibe.go(test_server + "/inputs")
    inp = vibe.find("#text-input")
    box = inp.bounds()
    vibe.mouse.click(box.x + box.width / 2, box.y + box.height / 2)
    vibe.keyboard.type("mouse")
    assert inp.value() == "mouse"


# ===========================================================================
# Clock (2)
# ===========================================================================

def test_install_set_fixed_time(bro, test_server):
    vibe = bro.page()
    vibe.go(test_server + "/clock")
    vibe.clock.install(time=0)
    vibe.clock.set_fixed_time(1000)
    result = vibe.eval("Date.now()")
    assert result == 1000


def test_fast_forward(bro, test_server):
    vibe = bro.page()
    vibe.go(test_server + "/clock")
    vibe.clock.install(time=0)
    vibe.clock.fast_forward(5000)
    result = vibe.eval("Date.now()")
    assert result >= 5000


# ===========================================================================
# Viewport / Emulation (3)
# ===========================================================================

def test_set_viewport_viewport(bro, test_server):
    vibe = bro.page()
    vibe.go(test_server)
    vibe.set_viewport({"width": 800, "height": 600})
    vp = vibe.viewport()
    assert vp["width"] == 800
    assert vp["height"] == 600


def test_set_content(bro, test_server):
    vibe = bro.page()
    vibe.set_content("<h1>Custom</h1>")
    el = vibe.find("h1")
    assert el.text() == "Custom"


def test_a11y_tree(bro, test_server):
    vibe = bro.page()
    vibe.go(test_server + "/a11y")
    tree = vibe.a11y_tree()
    assert isinstance(tree, dict)
    assert "role" in tree


# ===========================================================================
# Context (2)
# ===========================================================================

def test_new_context(bro):
    ctx = bro.new_context()
    assert isinstance(ctx, BrowserContext)
    assert ctx.id
    ctx.close()


def test_cookies_in_context(bro, test_server):
    ctx = bro.new_context()
    try:
        vibe = ctx.new_page()
        vibe.go(test_server + "/set-cookie")
        cookies = ctx.cookies()
        assert isinstance(cookies, list)
        assert len(cookies) >= 1
    finally:
        ctx.close()


# ===========================================================================
# Dialog auto-handling (1)
# ===========================================================================

def test_on_dialog_accept_string(bro, test_server):
    vibe = bro.new_page()
    vibe.go(test_server)
    vibe.on_dialog("accept")
    result = vibe.eval('confirm("Are you sure?")')
    assert result is True


# ===========================================================================
# Route handler callback (6)
# ===========================================================================

def test_route_fulfill(bro, test_server):
    vibe = bro.new_page()
    vibe.go(test_server + "/fetch")

    def handler(route):
        route.fulfill(status=200, body='{"message":"mocked"}', content_type="application/json")

    vibe.route("**/api/data", handler)
    vibe.eval("doFetch()")
    vibe.wait(500)
    result = vibe.find("#result").text()
    assert "mocked" in result
    vibe.unroute("**/api/data")


def test_route_inspect_request(bro, test_server):
    vibe = bro.new_page()
    vibe.go(test_server + "/fetch")
    captured = {}

    def handler(route):
        captured["url"] = route.request["url"]
        captured["method"] = route.request["method"]
        route.continue_()

    vibe.route("**/api/data", handler)
    vibe.eval("doFetch()")
    vibe.wait(500)
    assert "api/data" in captured.get("url", "")
    assert captured.get("method") == "GET"
    vibe.unroute("**/api/data")


def test_route_abort(bro, test_server):
    vibe = bro.new_page()
    vibe.go(test_server + "/fetch")
    vibe.route("**/api/data", "abort")
    vibe.eval("doFetch().catch(() => document.getElementById('result').textContent = 'aborted')")
    vibe.wait(500)
    result = vibe.find("#result").text()
    assert "aborted" in result or result == ""  # abort may not populate result
    vibe.unroute("**/api/data")


def test_route_default_continue(bro, test_server):
    vibe = bro.new_page()
    vibe.go(test_server + "/fetch")
    vibe.route("**/api/data", "continue")
    vibe.eval("doFetch()")
    vibe.wait(500)
    result = vibe.find("#result").text()
    assert "real data" in result
    vibe.unroute("**/api/data")


def test_static_route(bro, test_server):
    vibe = bro.new_page()
    vibe.go(test_server + "/fetch")
    vibe.route("**/api/data", {"status": 200, "body": '{"static":true}', "content_type": "application/json"})
    vibe.eval("doFetch()")
    vibe.wait(500)
    result = vibe.find("#result").text()
    assert "static" in result
    vibe.unroute("**/api/data")


def test_unroute(bro, test_server):
    vibe = bro.new_page()
    vibe.go(test_server + "/fetch")
    vibe.route("**/api/data", "abort")
    vibe.unroute("**/api/data")
    vibe.eval("doFetch()")
    vibe.wait(500)
    result = vibe.find("#result").text()
    assert "real data" in result


# ===========================================================================
# Dialog handler callback (7)
# ===========================================================================

def test_dialog_accept_alert(bro, test_server):
    vibe = bro.new_page()
    vibe.go(test_server)
    messages = []

    def handler(dialog):
        messages.append(dialog.message())
        dialog.accept()

    vibe.on_dialog(handler)
    vibe.eval('alert("Hello from test")')
    assert len(messages) == 1
    assert messages[0] == "Hello from test"


def test_dialog_dismiss_confirm(bro, test_server):
    vibe = bro.new_page()
    vibe.go(test_server)

    def handler(dialog):
        dialog.dismiss()

    vibe.on_dialog(handler)
    result = vibe.eval('confirm("Are you sure?")')
    assert result is False


def test_dialog_accept_confirm(bro, test_server):
    vibe = bro.new_page()
    vibe.go(test_server)

    def handler(dialog):
        dialog.accept()

    vibe.on_dialog(handler)
    result = vibe.eval('confirm("Are you sure?")')
    assert result is True


def test_dialog_prompt_text(bro, test_server):
    vibe = bro.new_page()
    vibe.go(test_server)

    def handler(dialog):
        assert dialog.type() == "prompt"
        dialog.accept("my answer")

    vibe.on_dialog(handler)
    result = vibe.eval('prompt("Enter name:")')
    assert result == "my answer"


def test_dialog_default_dismiss(bro, test_server):
    vibe = bro.new_page()
    vibe.go(test_server)
    # No handler — should auto-dismiss
    result = vibe.eval('confirm("Auto dismiss?")')
    assert result is False


def test_dialog_default_value(bro, test_server):
    vibe = bro.new_page()
    vibe.go(test_server)

    captured_default = []

    def handler(dialog):
        captured_default.append(dialog.default_value())
        dialog.accept()

    vibe.on_dialog(handler)
    vibe.eval('prompt("Name?", "default-val")')
    assert len(captured_default) == 1
    assert captured_default[0] == "default-val"


def test_static_on_dialog(bro, test_server):
    vibe = bro.new_page()
    vibe.go(test_server)
    vibe.on_dialog("dismiss")
    result = vibe.eval('confirm("test")')
    assert result is False


# ===========================================================================
# Console collect (2)
# ===========================================================================

def test_on_console_collect(bro, test_server):
    vibe = bro.new_page()
    vibe.go(test_server)
    vibe.on_console()
    vibe.eval('console.log("hello from console")')
    vibe.wait(300)
    msgs = vibe.console_messages()
    assert any("hello from console" in m["text"] for m in msgs)


def test_on_error_collect(bro, test_server):
    vibe = bro.new_page()
    vibe.go(test_server)
    vibe.on_error()
    vibe.eval('setTimeout(() => { throw new Error("test error") }, 0)')
    vibe.wait(500)
    errs = vibe.errors()
    assert any("test error" in e["message"] for e in errs)


# ===========================================================================
# Capture request/response (2)
# ===========================================================================

def test_capture_request_returns_dict(bro, test_server):
    vibe = bro.new_page()
    vibe.go(test_server + "/fetch")
    req = vibe.capture.request("**/api/data", fn=lambda: vibe.eval("setTimeout(() => doFetch(), 100)"), timeout=5000)
    assert isinstance(req, dict)
    assert "url" in req
    assert "api/data" in req["url"]


def test_capture_response_returns_dict(bro, test_server):
    vibe = bro.new_page()
    vibe.go(test_server + "/fetch")
    resp = vibe.capture.response("**/api/data", fn=lambda: vibe.eval("setTimeout(() => doFetch(), 100)"), timeout=5000)
    assert isinstance(resp, dict)
    assert "url" in resp
    assert resp["status"] == 200


# ===========================================================================
# Capture navigation (1)
# ===========================================================================

def test_capture_navigation(bro, test_server):
    vibe = bro.new_page()
    vibe.go(test_server + "/nav-test")
    with vibe.capture.navigation() as info:
        vibe.find("#link").click()
    assert info.value is not None
    assert "/page2" in info.value


# ===========================================================================
# Capture download (1)
# ===========================================================================

def test_capture_download(bro, test_server):
    vibe = bro.new_page()
    vibe.go(test_server + "/download")
    with vibe.capture.download() as info:
        vibe.find("#download-link").click()
    assert info.value is not None
    assert "/download-file" in info.value["url"]
    assert info.value["suggested_filename"] == "test.txt"
    assert info.value["path"] is not None


def test_capture_download_save_as(bro, test_server):
    import os
    import tempfile
    import shutil
    vibe = bro.new_page()
    vibe.go(test_server + "/download")
    result = vibe.capture.download(lambda: vibe.find("#download-link").click())
    assert result["path"] is not None
    tmp_dir = tempfile.mkdtemp(prefix="vibium-dl-")
    try:
        result.save_as(os.path.join(tmp_dir, "saved.txt"))
        with open(os.path.join(tmp_dir, "saved.txt")) as f:
            assert f.read() == "download content"
    finally:
        shutil.rmtree(tmp_dir)


# ===========================================================================
# Capture dialog (1)
# ===========================================================================

def test_capture_dialog(bro, test_server):
    vibe = bro.new_page()
    vibe.go(test_server)
    with vibe.capture.dialog() as info:
        vibe.eval('setTimeout(() => alert("Hello from expect"), 50)')
    assert info.value is not None
    assert info.value["type"] == "alert"
    assert info.value["message"] == "Hello from expect"


# ===========================================================================
# Capture event (1)
# ===========================================================================

def test_capture_event_navigation(bro, test_server):
    vibe = bro.new_page()
    vibe.go(test_server + "/nav-test")
    with vibe.capture.event("navigation") as info:
        vibe.find("#link").click()
    assert info.value is not None


# ===========================================================================
# Checkpoint (1)
# ===========================================================================

def test_full_checkpoint(bro, test_server):
    """End-to-end: navigate, find, interact, eval, screenshot — all sync."""
    vibe = bro.page()
    vibe.go(test_server + "/form")
    assert vibe.title() == "Form"

    name_input = vibe.find("#name")
    name_input.fill("Test User")
    assert name_input.value() == "Test User"

    email_input = vibe.find("#email")
    email_input.fill("test@example.com")

    cb = vibe.find("#agree")
    cb.check()
    assert cb.is_checked()

    sel = vibe.find("#color")
    sel.select_option("green")
    assert sel.value() == "green"

    result = vibe.eval("document.getElementById('name').value")
    assert result == "Test User"

    data = vibe.screenshot()
    assert data[:4] == b"\x89PNG"
