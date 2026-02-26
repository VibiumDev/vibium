# Getting Started with Vibium

A complete beginner's guide. No prior experience required.

---

## What You'll Build

A script that opens a browser, visits a website, takes a screenshot, and clicks a link. All in about 10 lines of code.

---

## Step 1: Install Node.js

Vibium runs on Node.js. Check if you have it:

```bash
node --version
```

If you see a version number (like `v18.0.0` or higher), skip to Step 2.

### macOS

```bash
# Install Homebrew (if you don't have it)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install Node
brew install node
```

### Windows

Download and run the installer from [nodejs.org](https://nodejs.org). Choose the LTS version.

### Linux

```bash
# Ubuntu/Debian
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt-get install -y nodejs

# Or use your distro's package manager
```

Verify it worked:

```bash
node --version   # Should show v18.0.0 or higher
npm --version    # Should show 9.0.0 or higher
```

---

## Step 2: Create a Project Folder

```bash
mkdir my-first-bot
cd my-first-bot
npm init -y
```

This creates a folder and a `package.json` file.

---

## Step 3: Install Vibium

```bash
npm install vibium
```

This downloads Vibium and Chrome for Testing. Might take a minute.

---

## Step 4: Write Your First Script

Create a file called `hello.js`:

```bash
# macOS/Linux
touch hello.js

# Windows (PowerShell)
New-Item hello.js
```

Open `hello.js` in any text editor and paste this:

```javascript
const fs = require('fs')
const { browser } = require('vibium/sync')

// Launch a browser (you'll see it open!)
const bro = browser.launch()
const vibe = bro.page()

// Go to a website
vibe.go('https://example.com')
console.log('Loaded example.com')

// Take a screenshot
const png = vibe.screenshot()
fs.writeFileSync('screenshot.png', png)
console.log('Saved screenshot.png')

// Find and click the link
const link = vibe.find('a')
console.log('Found link:', link.text())
link.click()
console.log('Clicked!')

// Close the browser
bro.close()
console.log('Done!')
```

---

## Step 5: Run It

```bash
node hello.js
```

You should see:
1. A Chrome window open
2. example.com load
3. The browser click "Learn more"
4. The browser close

Check your folder - there's now a `screenshot.png` file!

---

## What Just Happened?

| Line | What It Does |
|------|--------------|
| `browser.launch()` | Opens Chrome, returns a Browser |
| `bro.page()` | Gets the default page (tab) |
| `vibe.go(url)` | Navigates to a URL |
| `vibe.screenshot()` | Captures the page as PNG |
| `vibe.find(selector)` | Finds an element by CSS selector |
| `link.click()` | Clicks the element |
| `bro.close()` | Closes the browser |

---

## Next Steps

**Hide the browser** (run headless):
```javascript
const bro = browser.launch({ headless: true })
```

**Use async/await** (for more complex scripts):
```javascript
const { browser } = require('vibium')

async function main() {
  const bro = await browser.launch()
  const vibe = await bro.page()
  await vibe.go('https://example.com')
  // ...
  await bro.close()
}

main()
```

**Use Python instead:**
See [Getting Started (Python)](getting-started-python.md) for the Python version.

**Let AI control the browser:**
See [Getting Started with MCP](getting-started-mcp.md) for Claude Code and Gemini CLI setup.

---

## Troubleshooting

### "command not found: node"

Node.js isn't installed or isn't in your PATH. Reinstall from [nodejs.org](https://nodejs.org).

### "Cannot find module 'vibium'"

Run `npm install vibium` in your project folder.

### Browser doesn't open

Try running with headless mode disabled (it's disabled by default, but just in case):
```javascript
const bro = browser.launch({ headless: false })
```

### Permission denied (Linux)

You might need to install dependencies for Chrome:
```bash
sudo apt-get install -y libgbm1 libnss3 libatk-bridge2.0-0 libdrm2 libxkbcommon0 libxcomposite1 libxdamage1 libxfixes3 libxrandr2 libgbm1 libasound2
```

---

## You Did It! 🎉

You just automated a browser. The same techniques work for:
- Web scraping
- Testing websites
- Automating repetitive tasks
- Building AI agents that can browse the web

Questions? [Open an issue](https://github.com/VibiumDev/vibium/issues).
