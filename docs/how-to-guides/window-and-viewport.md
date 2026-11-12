# Window Size vs. Viewport Size

Learn the difference between the browser window and the viewport, and how to control both.

---

## What's the Difference?

- **Window** = the actual OS browser window (title bar, toolbar, and all). Controlled by `page.setWindow()`.
- **Viewport** = the CSS content area inside the window where web pages render. Controlled by `page.setViewport()`.

Python uses snake_case for the same calls: `set_window()` and `set_viewport()`.

When you resize the window, the viewport changes too. But `setViewport()` only changes the content area — the window stays the same size and scrollbars may appear.

---

## Setting the Viewport

Use `setViewport()` to control the CSS layout size. This is what you want for responsive testing.

```javascript
const { browser } = require('vibium');

async function main() {
  const b = await browser.start();
  const page = await b.page();

  // Simulate a mobile screen
  await page.setViewport({ width: 375, height: 812 });
  const vp = await page.viewport();
  console.log(vp); // { width: 375, height: 812 }

  await b.stop();
}

main();
```

```python
from vibium import browser

b = browser.start()
page = b.page()

# Simulate a mobile screen
page.set_viewport({"width": 375, "height": 812})
print(page.viewport())  # {'height': 812, 'width': 375}

b.stop()
```

```java
import com.vibium.Vibium;
import com.vibium.types.ViewportSize;

public class Viewport {
    public static void main(String[] args) {
        var b = Vibium.start();
        var page = b.page();

        // Simulate a mobile screen
        page.setViewport(new ViewportSize(375, 812));
        System.out.println(page.viewport()); // ViewportSize{width=375, height=812}

        b.stop();
    }
}
```

---

## Setting the Window

Use `setWindow()` to control the OS browser window — resize it, move it, or change its state.

```javascript
const { browser } = require('vibium');

async function main() {
  const b = await browser.start();
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

  await b.stop();
}

main();
```

```python
from vibium import browser

b = browser.start()
page = b.page()

# Resize the window
page.set_window(width=1280, height=720)

# Move the window to the top-left corner
page.set_window(x=0, y=0)

# Resize and move at the same time
page.set_window(width=800, height=600, x=100, y=100)

# Read the current window state
print(page.window())
# {'height': 600, 'state': 'normal', 'width': 800, 'x': 100, 'y': 100}

b.stop()
```

```java
import com.vibium.Vibium;
import com.vibium.types.WindowOptions;

public class Window {
    public static void main(String[] args) {
        var b = Vibium.start();
        var page = b.page();

        // Resize the window
        page.setWindow(new WindowOptions().width(1280).height(720));

        // Move the window to the top-left corner
        page.setWindow(new WindowOptions().x(0).y(0));

        // Resize and move at the same time
        page.setWindow(new WindowOptions().width(800).height(600).x(100).y(100));

        // Read the current window state
        var win = page.window();
        System.out.println(win.state() + " " + win.width() + "x" + win.height());
        // normal 800x600

        b.stop();
    }
}
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

```python
# Maximize the window
page.set_window(state="maximized")

# Minimize it
page.set_window(state="minimized")

# Go fullscreen
page.set_window(state="fullscreen")

# Restore to a specific size
page.set_window(state="normal", width=1024, height=768)
```

```java
// Maximize the window
page.setWindow(new WindowOptions().state("maximized"));

// Minimize it
page.setWindow(new WindowOptions().state("minimized"));

// Go fullscreen
page.setWindow(new WindowOptions().state("fullscreen"));

// Restore to a specific size
page.setWindow(new WindowOptions().state("normal").width(1024).height(768));
```

A minimized window MUST be restored to `normal` before it can go fullscreen. The browser rejects the direct change.

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

```python
# Viewport (CSS content area)
page.set_viewport({"width": 1280, "height": 720})
vp = page.viewport()  # {"width", "height"}

# Window (OS browser window)
page.set_window(width=1280, height=720, x=0, y=0)
page.set_window(state="maximized")
win = page.window()   # {"state", "width", "height", "x", "y"}
```

```java
// Viewport (CSS content area)
page.setViewport(new ViewportSize(1280, 720));
ViewportSize vp = page.viewport();  // width(), height()

// Window (OS browser window)
page.setWindow(new WindowOptions().width(1280).height(720).x(0).y(0));
page.setWindow(new WindowOptions().state("maximized"));
WindowInfo win = page.window();     // state(), width(), height(), x(), y()
```
