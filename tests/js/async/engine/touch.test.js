/**
 * JS Library Tests: page.touch gestures (tap, swipe, long-press).
 * Each gesture drives input.performActions with pointerType "touch", so the
 * page sees real pointer events with pointerType === "touch" (#682).
 */

const { test, describe, before, after } = require("../../../helpers/capabilities").suite("touch");
const assert = require('node:assert');

const { browser } = require('../../../../clients/javascript/dist');

let bro;

before(async () => {
  bro = await browser.start({ headless: true });
});

after(async () => {
  await bro.stop();
});

// A full-viewport recorder: every pointer event is pushed to window.__ev with
// its type, client coordinates, pointerType and a timestamp.
const RECORDER = `<body style="margin:0">
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
</body>`;

describe('Touch: page-level gestures', () => {
  test('tap() fires a touch pointerdown/up at the point', async () => {
    const vibe = await bro.newPage();
    await vibe.setContent(RECORDER);

    await vibe.touch.tap(60, 80);

    const ev = await vibe.evaluate('window.__ev');
    const down = ev.find((e) => e.type === 'pointerdown');
    const up = ev.find((e) => e.type === 'pointerup');
    assert.ok(down, 'expected a pointerdown');
    assert.ok(up, 'expected a pointerup');
    assert.strictEqual(down.pt, 'touch', 'pointerType should be touch');
    assert.strictEqual(down.x, 60);
    assert.strictEqual(down.y, 80);
  });

  test('swipe() drags from start to end with the finger down', async () => {
    const vibe = await bro.newPage();
    await vibe.setContent(RECORDER);

    await vibe.touch.swipe(40, 100, 240, 100, 200);

    const ev = await vibe.evaluate('window.__ev');
    assert.ok(ev.every((e) => e.pt === 'touch'), 'every event should be a touch pointer');

    const down = ev.find((e) => e.type === 'pointerdown');
    const up = ev[ev.length - 1];
    const moves = ev.filter((e) => e.type === 'pointermove');

    assert.ok(down, 'expected a pointerdown');
    assert.strictEqual(down.x, 40, 'press at start x');
    assert.ok(moves.length >= 1, 'expected at least one pointermove while held');
    assert.strictEqual(up.type, 'pointerup', 'gesture ends with pointerup');
    assert.strictEqual(up.x, 240, 'release at end x');
    assert.strictEqual(up.y, 100, 'release at end y');
  });

  test('longPress() holds the press for the requested duration', async () => {
    const vibe = await bro.newPage();
    await vibe.setContent(RECORDER);

    await vibe.touch.longPress(120, 120, 400);

    const ev = await vibe.evaluate('window.__ev');
    const down = ev.find((e) => e.type === 'pointerdown');
    const up = ev.find((e) => e.type === 'pointerup');
    assert.ok(down && up, 'expected a pointerdown and pointerup');
    assert.strictEqual(down.pt, 'touch');
    assert.strictEqual(down.x, 120);
    assert.strictEqual(down.y, 120);
    // Held at least most of the requested 400ms (allow scheduling slack).
    assert.ok(up.t - down.t >= 300, `press held ${Math.round(up.t - down.t)}ms, expected >= 300`);
  });
});
