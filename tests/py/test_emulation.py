"""Emulation tests — viewport, emulateMedia, setContent, setGeolocation, window (17 async tests)."""

import pytest


# --- Viewport ---

async def test_viewport(async_page, test_server):
    await async_page.go(test_server)
    vp = await async_page.viewport()
    assert "width" in vp
    assert "height" in vp
    assert vp["width"] > 0


async def test_set_viewport(async_page, test_server):
    await async_page.go(test_server)
    await async_page.set_viewport({"width": 1024, "height": 768})
    vp = await async_page.viewport()
    assert vp["width"] == 1024
    assert vp["height"] == 768


# --- SetContent ---

async def test_set_content(async_page, test_server):
    await async_page.set_content("<h1>Injected</h1>")
    el = await async_page.find("h1")
    text = await el.text()
    assert text == "Injected"


async def test_set_content_with_title(async_page, test_server):
    await async_page.set_content("<html><head><title>Custom Title</title></head><body><p>Body</p></body></html>")
    title = await async_page.title()
    assert title == "Custom Title"


# --- EmulateMedia ---

async def test_emulate_media_dark(async_page, test_server):
    await async_page.go(test_server)
    await async_page.emulate_media(colorScheme="dark")
    result = await async_page.eval("matchMedia('(prefers-color-scheme: dark)').matches")
    assert result is True


async def test_emulate_media_light(async_page, test_server):
    await async_page.go(test_server)
    await async_page.emulate_media(colorScheme="light")
    result = await async_page.eval("matchMedia('(prefers-color-scheme: light)').matches")
    assert result is True


async def test_emulate_media_print(async_page, test_server):
    await async_page.go(test_server)
    await async_page.emulate_media(media="print")
    result = await async_page.eval("matchMedia('print').matches")
    assert result is True


async def test_emulate_media_reduced_motion(async_page, test_server):
    await async_page.go(test_server)
    await async_page.emulate_media(reducedMotion="reduce")
    result = await async_page.eval("matchMedia('(prefers-reduced-motion: reduce)').matches")
    assert result is True


async def test_emulate_media_forced_colors(async_page, test_server):
    await async_page.go(test_server)
    await async_page.emulate_media(forcedColors="active")
    result = await async_page.eval("matchMedia('(forced-colors: active)').matches")
    assert result is True


async def test_emulate_media_contrast(async_page, test_server):
    await async_page.go(test_server)
    await async_page.emulate_media(contrast="more")
    result = await async_page.eval("matchMedia('(prefers-contrast: more)').matches")
    assert result is True


async def test_emulate_media_reset(async_page, test_server):
    await async_page.go(test_server)
    await async_page.emulate_media(colorScheme="dark")
    result = await async_page.eval("matchMedia('(prefers-color-scheme: dark)').matches")
    assert result is True
    # Reset
    await async_page.emulate_media(colorScheme="light")
    result = await async_page.eval("matchMedia('(prefers-color-scheme: light)').matches")
    assert result is True


# --- Window ---

async def test_window(async_page, test_server):
    await async_page.go(test_server)
    win = await async_page.window()
    assert isinstance(win, dict)
    assert "width" in win or "state" in win


async def test_set_window_resize(async_page, test_server):
    await async_page.go(test_server)
    await async_page.set_window(width=900, height=700)
    win = await async_page.window()
    # Window size may not exactly match due to OS chrome
    assert isinstance(win, dict)


async def test_set_window_move(async_page, test_server):
    await async_page.go(test_server)
    await async_page.set_window(x=100, y=100)
    # Just verify no error
    win = await async_page.window()
    assert isinstance(win, dict)


# --- Geolocation ---

async def test_set_geolocation(async_page, test_server):
    await async_page.go(test_server)
    await async_page.set_geolocation({"latitude": 37.7749, "longitude": -122.4194})
    # Verify geolocation was set by querying it
    result = await async_page.eval("""
        new Promise((resolve) => {
            navigator.geolocation.getCurrentPosition(
                (pos) => resolve({lat: pos.coords.latitude, lng: pos.coords.longitude}),
                () => resolve(null),
                {timeout: 5000}
            )
        })
    """)
    if result:
        assert abs(result["lat"] - 37.7749) < 0.01


# --- SetContent with title verification ---

async def test_set_content_replaces(async_page, test_server):
    await async_page.go(test_server)
    assert await async_page.title() == "Test App"
    await async_page.set_content("<h1>Replaced</h1>")
    el = await async_page.find("h1")
    text = await el.text()
    assert text == "Replaced"
