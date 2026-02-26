"""Element state tests — text, innerText, html, value, attr, bounds, visibility, eval, screenshot (23 async tests)."""

import pytest

from vibium._types import BoundingBox


# --- Text / content ---

async def test_text(async_page, test_server):
    await async_page.go(test_server)
    el = await async_page.find("#info")
    assert await el.text() == "Some info text"


async def test_inner_text(async_page, test_server):
    await async_page.go(test_server)
    el = await async_page.find("#info")
    inner = await el.inner_text()
    assert "Some info text" in inner


async def test_html(async_page, test_server):
    await async_page.go(test_server)
    el = await async_page.find("#info")
    html = await el.html()
    assert html == "Some info text"


async def test_value(async_page, test_server):
    await async_page.go(test_server + "/inputs")
    inp = await async_page.find("#text-input")
    await inp.fill("test123")
    assert await inp.value() == "test123"


# --- Attributes ---

async def test_attr(async_page, test_server):
    await async_page.go(test_server + "/links")
    link = await async_page.find(".link")
    cls = await link.attr("class")
    assert cls == "link"


async def test_attr_null_missing(async_page, test_server):
    await async_page.go(test_server)
    el = await async_page.find("h1")
    val = await el.attr("data-nonexistent")
    assert val is None


async def test_get_attribute_alias(async_page, test_server):
    await async_page.go(test_server + "/links")
    link = await async_page.find(".link")
    cls = await link.get_attribute("class")
    assert cls == "link"


# --- Bounds ---

async def test_bounds(async_page, test_server):
    await async_page.go(test_server)
    el = await async_page.find("h1")
    box = await el.bounds()
    assert isinstance(box, BoundingBox)
    assert box.width > 0
    assert box.height > 0
    assert box.x >= 0
    assert box.y >= 0


async def test_bounding_box_alias(async_page, test_server):
    await async_page.go(test_server)
    el = await async_page.find("h1")
    box = await el.bounding_box()
    assert isinstance(box, BoundingBox)
    assert box.width > 0


# --- Visibility ---

async def test_is_visible(async_page, test_server):
    await async_page.go(test_server)
    el = await async_page.find("h1")
    assert await el.is_visible()


async def test_is_hidden(async_page, test_server):
    await async_page.go(test_server)
    # Set content with a hidden element
    await async_page.set_content('<div style="display:none" id="hidden">Hidden</div><div id="shown">Shown</div>')
    hidden = await async_page.find("#hidden")
    assert await hidden.is_hidden()


# --- Enabled / checked / editable ---

async def test_is_enabled(async_page, test_server):
    await async_page.go(test_server + "/form")
    btn = await async_page.find("button")
    assert await btn.is_enabled()


async def test_is_checked(async_page, test_server):
    await async_page.go(test_server + "/form")
    cb = await async_page.find("#agree")
    assert not await cb.is_checked()
    await cb.check()
    assert await cb.is_checked()


async def test_is_editable(async_page, test_server):
    await async_page.go(test_server + "/inputs")
    inp = await async_page.find("#text-input")
    assert await inp.is_editable()


# --- Eval ---

async def test_element_eval(async_page, test_server):
    await async_page.go(test_server)
    el = await async_page.find("h1")
    result = await el.eval("return el.tagName")
    assert result == "H1"


# --- Screenshot ---

async def test_element_screenshot(async_page, test_server):
    await async_page.go(test_server)
    el = await async_page.find("h1")
    data = await el.screenshot()
    assert isinstance(data, bytes)
    assert data[:4] == b"\x89PNG"


# --- Waiting ---

async def test_find_auto_waits(async_page, test_server):
    await async_page.go(test_server + "/dynamic-loading")
    el = await async_page.find("#loaded", timeout=5000)
    text = await el.text()
    assert text == "Loaded!"


async def test_wait_ms(async_page, test_server):
    await async_page.go(test_server)
    await async_page.wait(100)  # should not throw


async def test_wait_until_function(async_page, test_server):
    await async_page.go(test_server)
    await async_page.eval("setTimeout(() => { window.__ready = true }, 200)")
    result = await async_page.wait_until("() => window.__ready", timeout=5000)
    assert result is True


# --- Fluent chains ---

async def test_find_text_chain(async_page, test_server):
    await async_page.go(test_server)
    text = await (await async_page.find("#info")).text()
    assert text == "Some info text"


async def test_find_attr_chain(async_page, test_server):
    await async_page.go(test_server + "/links")
    href = await (await async_page.find(".link")).attr("href")
    assert href == "/subpage"


async def test_find_is_visible_chain(async_page, test_server):
    await async_page.go(test_server)
    visible = await (await async_page.find("h1")).is_visible()
    assert visible is True


# --- Checkpoint ---

async def test_full_checkpoint(async_page, test_server):
    """End-to-end state inspection checkpoint."""
    await async_page.go(test_server + "/form")
    name = await async_page.find("#name")
    await name.fill("John")
    assert await name.value() == "John"
    assert await name.is_visible()
    assert await name.is_editable()
    box = await name.bounds()
    assert box.width > 0
    html = await name.html()
    assert html == ""  # input has no innerHTML
    tag_result = await name.eval("return el.tagName")
    assert tag_result == "INPUT"
