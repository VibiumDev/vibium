"""Touch gesture tests — tap, swipe, long-press (#682).

Each gesture drives input.performActions with pointerType "touch", so the page
sees real pointer events with pointerType == "touch". A fresh page per test
keeps each recorder's window.__ev isolated.
"""

import pytest

pytestmark = pytest.mark.capability("touch")

# A full-viewport recorder: every pointer event is pushed to window.__ev.
RECORDER = """<body style="margin:0">
  <div id="pad" style="width:100vw;height:100vh;touch-action:none"></div>
  <script>
    window.__ev = [];
    const pad = document.getElementById('pad');
    for (const type of ['pointerdown', 'pointermove', 'pointerup']) {
      pad.addEventListener(type, (e) => {
        window.__ev.push({ type, x: e.clientX, y: e.clientY, pt: e.pointerType, t: performance.now() });
      });
    }
  </script>
</body>"""


async def test_tap(async_browser):
    page = await async_browser.new_page()
    await page.set_content(RECORDER)

    await page.touch.tap(60, 80)

    ev = await page.evaluate("window.__ev")
    down = next(e for e in ev if e["type"] == "pointerdown")
    assert down["pt"] == "touch"
    assert down["x"] == 60
    assert down["y"] == 80


async def test_swipe(async_browser):
    page = await async_browser.new_page()
    await page.set_content(RECORDER)

    await page.touch.swipe(40, 100, 240, 100, 200)

    ev = await page.evaluate("window.__ev")
    assert all(e["pt"] == "touch" for e in ev)
    down = next(e for e in ev if e["type"] == "pointerdown")
    moves = [e for e in ev if e["type"] == "pointermove"]
    up = ev[-1]
    assert down["x"] == 40
    assert len(moves) >= 1
    assert up["type"] == "pointerup"
    assert up["x"] == 240
    assert up["y"] == 100


async def test_long_press(async_browser):
    page = await async_browser.new_page()
    await page.set_content(RECORDER)

    await page.touch.long_press(120, 120, 400)

    ev = await page.evaluate("window.__ev")
    down = next(e for e in ev if e["type"] == "pointerdown")
    up = next(e for e in ev if e["type"] == "pointerup")
    assert down["pt"] == "touch"
    assert down["x"] == 120
    assert down["y"] == 120
    # Held at least most of the requested 400ms (allow scheduling slack).
    assert up["t"] - down["t"] >= 300
