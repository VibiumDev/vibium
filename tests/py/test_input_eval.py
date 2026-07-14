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
    result = await async_page.evaluate("window.testVal")
    assert result == 42


async def test_eval_string(async_page, test_server):
    await async_page.go(test_server)
    result = await async_page.evaluate("'hello'")
    assert result == "hello"


async def test_eval_null_undefined(async_page, test_server):
    await async_page.go(test_server)
    result = await async_page.evaluate("null")
    assert result is None


async def test_add_script(async_page, test_server):
    await async_page.go(test_server)
    await async_page.add_script("window.__injected = 'yes'")
    result = await async_page.evaluate("window.__injected")
    assert result == "yes"


async def test_add_style(async_page, test_server):
    await async_page.go(test_server)
    await async_page.add_style("body { background: rgb(255, 0, 0); }")
    bg = await async_page.evaluate("getComputedStyle(document.body).backgroundColor")
    assert "255" in bg


async def test_expose(async_page, test_server):
    await async_page.go(test_server)
    await async_page.expose("myFunc", "(...args) => args.join('-')")
    result = await async_page.evaluate("window.myFunc('a', 'b')")
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
    result = await async_page.evaluate("1 + 1")
    assert result == 2


async def test_eval_is_alias_for_evaluate(async_page, test_server):
    # Regression #144: page.eval() did not exist (only evaluate()).
    await async_page.go(test_server)
    assert await async_page.eval("1 + 1") == 2


async def test_large_message_does_not_crash_reader(async_page, test_server):
    # Regression #110: a single pipe message larger than asyncio's 64 KiB default
    # StreamReader limit (e.g. a base64 screenshot) raised LimitOverrunError and
    # killed the receiver loop. A ~5 MB result must round-trip intact.
    await async_page.go(test_server)
    n = 5_000_000
    big = await async_page.evaluate(f"'x'.repeat({n})")
    assert isinstance(big, str) and len(big) == n
    # the receiver loop must still be alive for follow-up commands
    assert await async_page.evaluate("2 + 2") == 4
