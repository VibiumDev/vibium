# #vibium week 1 progress: we're building!

here's where we're at.

---

## what's working

the go binary (clicker) can now:

- auto-install chrome for testing + chromedriver (no manual setup)
- launch chrome via chromedriver with webdriver bidi enabled
- navigate to any url
- take screenshots (headless or headed)
- find elements by css selector
- click elements
- type into inputs

all the core browser primitives are in place. the foundation is solid.

---

## human review checkpoint #1: passed

completed the first human review checkpoint in the roadmap doc. tested everything manually (ironically?):

- chrome launches and exits cleanly
- no zombie processes (even on ctrl+c mid-operation)
- screenshots render correctly
- click navigates pages
- keyboard input works on real sites

tested on example.com, the-internet.herokuapp.com, and vibium.com. everything works. (macos for now, will test linux and windows later)

---

## cli in action:

```bash
# take a screenshot
clicker screenshot https://example.com -o shot.png

# find an element
clicker find https://example.com "a"
→ tag=A, text="Learn more", box={x:151, y:151, w:82, h:18}

# click a link
clicker click https://example.com "a"
→ navigates to iana.org

# type into an input
clicker type https://the-internet.herokuapp.com/inputs "input" "12345"
```

flags for debugging:
- `--headed` shows the browser window (headless by default)
- `--wait-open 5` (waits 5 second for page load)
- `--wait-close 3` (keeps browser visible for 3 seconds before closing)

---

## building with claude code:

it's been a joy using claude code to build vibium. each milestone is a prompt. claude reads the roadmap, implements the feature, runs the checkpoint test, and we move on. so much fun. it helps that the webdriver bidi spec has been out for a few years. it's pretty straightforward to implement now. it's "just websockets and json", just like the chrome devtools protocol that playwright depends on.

bootstrapping clicker as a go-based command line utility has also made testing dead simple. i still secretly wish i made this back-end utility in nim, but i'm not sure the world's ready for that, yet.

---

## what's next

- day 6: bidi proxy server (websocket server that routes messages between clients and chrome)
- day 7-8: javascript/typescript client with async and sync apis
- day 9: auto-wait for elements
- day 10: mcp server for claude code integration

we're on track for christmas. ✨🎅🎄🎁✨

follow the vibium github repo for more. (better yet, give it a ⭐ )

---

*december 16, 2025*
