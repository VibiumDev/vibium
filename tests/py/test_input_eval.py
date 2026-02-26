"""Input & eval tests — keyboard, mouse, screenshot options, pdf, eval, addScript, expose (18 async tests)."""

import pytest


# --- Keyboard ---

async def test_keyboard_type(async_page, test_server):
    await async_page.go(test_server + "/inputs")
    inp = await async_page.find("#text-input")
    await inp.fill("")
    await inp.focus()
    await async_page.keyboard.type("typed via keyboard")
    assert await inp.value() == "typed via keyboard"


async def test_keyboard_press(async_page, test_server):
    await async_page.go(test_server + "/inputs")
    inp = await async_page.find("#text-input")
    await inp.fill("")
    await inp.focus()
    await async_page.keyboard.press("a")
    assert await inp.value() == "a"


async def test_keyboard_down_up(async_page, test_server):
    await async_page.go(test_server + "/inputs")
    inp = await async_page.find("#text-input")
    await inp.fill("")
    await inp.focus()
    await async_page.keyboard.down("Shift")
    await async_page.keyboard.press("a")
    await async_page.keyboard.up("Shift")
    assert await inp.value() == "A"


# --- Mouse ---

async def test_mouse_click(async_page, test_server):
    await async_page.go(test_server + "/inputs")
    inp = await async_page.find("#text-input")
    box = await inp.bounds()
    await async_page.mouse.click(box.x + box.width / 2, box.y + box.height / 2)
    await async_page.keyboard.type("mouse click")
    assert await inp.value() == "mouse click"


async def test_mouse_move(async_page, test_server):
    await async_page.go(test_server)
    await async_page.mouse.move(100, 100)  # should not throw


async def test_mouse_wheel(async_page, test_server):
    await async_page.go(test_server)
    await async_page.mouse.wheel(0, 100)  # scroll down, should not throw


# --- Screenshots ---

async def test_screenshot_png(async_page, test_server):
    await async_page.go(test_server)
    data = await async_page.screenshot()
    assert data[:4] == b"\x89PNG"


async def test_screenshot_full_page(async_page, test_server):
    await async_page.go(test_server)
    data = await async_page.screenshot(full_page=True)
    assert data[:4] == b"\x89PNG"
    assert len(data) > 100


async def test_screenshot_clip(async_page, test_server):
    await async_page.go(test_server)
    data = await async_page.screenshot(clip={"x": 0, "y": 0, "width": 100, "height": 100})
    assert data[:4] == b"\x89PNG"


async def test_pdf(async_page, test_server):
    await async_page.go(test_server)
    data = await async_page.pdf()
    assert isinstance(data, bytes)
    assert data[:2] == b"%P"


# --- Eval ---

async def test_eval_expression(async_page, test_server):
    await async_page.go(test_server + "/eval")
    result = await async_page.eval("window.testVal")
    assert result == 42


async def test_eval_string(async_page, test_server):
    await async_page.go(test_server)
    result = await async_page.eval("'hello'")
    assert result == "hello"


async def test_eval_null_undefined(async_page, test_server):
    await async_page.go(test_server)
    result = await async_page.eval("null")
    assert result is None


async def test_eval_handle(async_page, test_server):
    await async_page.go(test_server)
    handle = await async_page.eval_handle("document.body")
    assert isinstance(handle, str)
    assert len(handle) > 0


async def test_add_script(async_page, test_server):
    await async_page.go(test_server)
    await async_page.add_script("window.__injected = 'yes'")
    result = await async_page.eval("window.__injected")
    assert result == "yes"


async def test_add_style(async_page, test_server):
    await async_page.go(test_server)
    await async_page.add_style("body { background: rgb(255, 0, 0); }")
    bg = await async_page.eval("getComputedStyle(document.body).backgroundColor")
    assert "255" in bg


async def test_expose(async_page, test_server):
    await async_page.go(test_server)
    await async_page.expose("myFunc", "(...args) => args.join('-')")
    result = await async_page.eval("window.myFunc('a', 'b')")
    assert result == "a-b"


# --- Checkpoint ---

async def test_checkpoint(async_page, test_server):
    """End-to-end: keyboard, mouse, eval, screenshot."""
    await async_page.go(test_server + "/inputs")
    inp = await async_page.find("#text-input")
    await inp.fill("")
    await inp.focus()
    await async_page.keyboard.type("checkpoint")
    assert await inp.value() == "checkpoint"
    data = await async_page.screenshot()
    assert data[:4] == b"\x89PNG"
    result = await async_page.eval("1 + 1")
    assert result == 2
