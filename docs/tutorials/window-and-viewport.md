# Window Size vs. Viewport Size

Learn the difference between the browser window and the viewport, and how to control both.

---

## What's the Difference?

- **Window** = the actual OS browser window (title bar, toolbar, and all). Controlled by `page.setWindow()`.
- **Viewport** = the CSS content area inside the window where web pages render. Controlled by `page.setViewport()`.

When you resize the window, the viewport changes too. But `setViewport()` only changes the content area — the window stays the same size and scrollbars may appear.

---

## Setting the Viewport

Use `setViewport()` to control the CSS layout size. This is what you want for responsive testing.

```javascript
const { browser } = require('vibium');

async function main() {
  const b = await browser.launch();
  const page = await b.page();

  // Simulate a mobile screen
  await page.setViewport({ width: 375, height: 812 });
  const vp = await page.viewport();
  console.log(vp); // { width: 375, height: 812 }

  await b.close();
}

main();
```

---

## Setting the Window

Use `setWindow()` to control the OS browser window — resize it, move it, or change its state.

```javascript
const { browser } = require('vibium');

async function main() {
  const b = await browser.launch();
  const page = await b.page();

  // Resize the window
  await page.setWindow({ width: 1280, height: 720 });

  // Move the window to the top-left corner
  await page.setWindow({ x: 0, y: 0 });

  // Resize and move at the same time
  await page.setWindow({ width: 800, height: 600, x: 100, y: 100 });

  // Read the current window state
  const win = await page.window();
  console.log(win);
  // { state: 'normal', width: 800, height: 600, x: 100, y: 100 }

  await b.close();
}

main();
```

---

## Window States

The browser window can be in one of four states:

| State | Description |
|-------|-------------|
| `normal` | Default windowed mode |
| `maximized` | Fills the screen (with taskbar visible) |
| `minimized` | Hidden in the taskbar/dock |
| `fullscreen` | Fills the entire screen |

```javascript
// Maximize the window
await page.setWindow({ state: 'maximized' });

// Minimize it
await page.setWindow({ state: 'minimized' });

// Go fullscreen
await page.setWindow({ state: 'fullscreen' });

// Restore to a specific size
await page.setWindow({ state: 'normal', width: 1024, height: 768 });
```

---

## When to Use Which

| Goal | Use |
|------|-----|
| Test responsive layouts at specific breakpoints | `setViewport()` |
| Simulate a device screen size | `setViewport()` |
| Position the browser on your monitor | `setWindow()` |
| Maximize or fullscreen the browser | `setWindow()` |
| Get the CSS content area dimensions | `viewport()` |
| Get the OS window dimensions | `window()` |

---

## Quick Reference

```javascript
// Viewport (CSS content area)
await page.setViewport({ width: 1280, height: 720 });
const vp = await page.viewport();  // { width, height }

// Window (OS browser window)
await page.setWindow({ width: 1280, height: 720, x: 0, y: 0 });
await page.setWindow({ state: 'maximized' });
const win = await page.window();   // { state, width, height, x, y }
```
