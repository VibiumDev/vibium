# Java Client Specification

Design document for the Vibium Java client library.

---

## 1. Overview

The Java client provides a type-safe, idiomatic Java API for browser automation via the Vibium binary. Target audiences: Java developers, Android developers, enterprise automation teams.

**Key decisions:**
- Build tool: Gradle (Kotlin DSL)
- Minimum Java: 11 (broadest LTS compatibility)
- groupId: `com.vibium`, artifactId: `vibium`
- JSON library: Gson (lightweight, zero transitive deps)
- License: Apache 2.0
- No JNI — Go binary spawned as subprocess

---

## 2. Architecture

```
┌────────────────┐  stdin/stdout  ┌─────────────┐    BiDi/WS     ┌─────────┐
│  Java Client   │◄──────────────►│   vibium    │◄──────────────►│ Chrome  │
│  (this lib)    │ ndjson (pipes) │   binary    │ WebDriver BiDi │ browser │
└────────────────┘                └─────────────┘                └─────────┘
```

### JAR Strategy

Single fat JAR (~50MB) with all 5 platform binaries embedded in `natives/`:
- `natives/vibium-darwin-amd64`
- `natives/vibium-darwin-arm64`
- `natives/vibium-linux-amd64`
- `natives/vibium-linux-arm64`
- `natives/vibium-windows-amd64.exe`

At runtime, `BinaryResolver` extracts the correct binary to a temp directory.

### Subprocess Protocol

The client spawns `vibium pipe [--headless] [--connect URL]` and communicates via newline-delimited JSON (ndjson) over stdin/stdout.

**Request** (client → vibium):
```json
{"id": 1, "method": "vibium:page.navigate", "params": {"context": "ctx-1", "url": "https://example.com"}}
```

**Success response** (vibium → client):
```json
{"id": 1, "type": "success", "result": {}}
```

**Error response** (vibium → client):
```json
{"id": 1, "type": "error", "error": "timeout", "message": "Timeout after 30000ms waiting for '#btn'"}
```

**Event** (vibium → client, no `id`):
```json
{"method": "browsingContext.load", "params": {"context": "ctx-1", "url": "https://example.com"}}
```

---

## 3. Public API

### Class Diagram

```
Vibium                        ← static entry point: Vibium.start()
  └── returns Browser

Browser                       ← manages browser lifecycle
├── page()                    → Page
├── newPage()                 → Page
├── newContext()              → BrowserContext
├── pages()                   → List<Page>
├── onPage(Consumer<Page>)
├── onPopup(Consumer<Page>)
├── removeAllListeners(String?)
└── stop()

BrowserContext                ← isolation boundary
├── recording                 → Recording (property)
├── newPage()                 → Page
├── close()
├── cookies(String...?)       → List<Cookie>
├── setCookies(List<SetCookieParam>)
├── clearCookies()
├── storage()                 → StorageState
├── setStorage(StorageState)
├── clearStorage()
└── addInitScript(String)     → String

Page                          ← a browser tab (~60 methods)
├── keyboard                  → Keyboard (property)
├── mouse                     → Mouse (property)
├── touch                     → Touch (property)
├── clock                     → Clock (property)
├── context                   → BrowserContext (property)
├── id()                      → String (context ID)
│
├── go(String url)
├── back(), forward(), reload()
├── url(), title(), content()
│
├── find(String selector)               → Element
├── find(String selector, FindOptions)  → Element
├── find(SelectorOptions)               → Element
├── findAll(String selector)            → List<Element>
├── findAll(SelectorOptions)            → List<Element>
│
├── screenshot()              → byte[]
├── screenshot(ScreenshotOptions) → byte[]
├── pdf()                     → byte[]
│
├── evaluate(String expr)     → Object
├── addScript(String)
├── addStyle(String)
├── expose(String name, Function<Object[], Object>)
│
├── wait(long ms)
├── waitFor(String selector)
├── waitFor(String selector, FindOptions)
├── waitForFunction(String fn)
├── waitForFunction(String fn, WaitOptions)
├── waitForURL(String pattern)
├── waitForURL(String pattern, WaitOptions)
├── waitForLoad()
├── waitForLoad(WaitOptions)
│
├── setViewport(ViewportSize)
├── viewport()                → ViewportSize
├── emulateMedia(MediaOptions)
├── setContent(String html)
├── setGeolocation(GeoCoords)
├── setWindow(WindowOptions)
├── window()                  → WindowInfo
│
├── a11yTree()                → A11yNode
├── a11yTree(A11yOptions)     → A11yNode
├── frames()                  → List<Page>
├── frame(String nameOrUrl)   → Page
├── mainFrame()               → Page
│
├── scroll()
├── scroll(ScrollOptions)
├── bringToFront()
├── close()
├── setHeaders(Map<String,String>)
│
├── route(String pattern, Consumer<Route>)
├── unroute(String pattern)
├── onRequest(Consumer<Request>)
├── onResponse(Consumer<Response>)
├── onDialog(Consumer<Dialog>)
├── onConsole(Consumer<ConsoleMessage>)
├── onError(Consumer<String>)
├── onDownload(Consumer<Download>)
├── onWebSocket(Consumer<WebSocketInfo>)
├── removeAllListeners(String?)
├── consoleMessages()         → List<ConsoleMessage>
└── errors()                  → List<String>

Element                       ← resolved DOM element (~35 methods)
├── click(), dblclick()
├── fill(String), type(String), press(String)
├── clear(), check(), uncheck()
├── selectOption(String)
├── hover(), focus()
├── dragTo(Element target)
├── tap()
├── scrollIntoView()
├── dispatchEvent(String type)
├── setFiles(List<String>)
├── highlight()
│
├── text(), innerText(), html(), value()
├── attr(String name)  / getAttribute(String name)
├── bounds()           / boundingBox()
│
├── isVisible(), isHidden(), isEnabled(), isChecked(), isEditable()
├── role(), label()
├── screenshot()              → byte[]
│
├── waitUntil()
├── waitUntil(String state)
├── find(String), findAll(String)
└── find(SelectorOptions), findAll(SelectorOptions)

Keyboard                      ← page-level keyboard input
├── press(String key)
├── down(String key)
├── up(String key)
└── type(String text)

Mouse                         ← page-level mouse input
├── click(double x, double y)
├── click(double x, double y, MouseOptions)
├── move(double x, double y)
├── move(double x, double y, MouseOptions)
├── down(), down(MouseOptions)
├── up(), up(MouseOptions)
└── wheel(double deltaX, double deltaY)

Touch                         ← page-level touch input
└── tap(double x, double y)

Clock                         ← fake timer control
├── install(), install(ClockOptions)
├── fastForward(long ticks)
├── runFor(long ticks)
├── pauseAt(String time)
├── resume()
├── setFixedTime(String time)
├── setSystemTime(String time)
└── setTimezone(String tz)

Recording                     ← trace recording
├── start(), start(RecordingOptions)
├── stop()                    → byte[]
├── stop(String path)         → byte[]
├── startChunk(), startChunk(ChunkOptions)
├── stopChunk()               → byte[]
├── startGroup(String name)
├── startGroup(String name, String location)
└── stopGroup()

Route                         ← intercepted request
├── request()                 → Request
├── fulfill()
├── fulfill(FulfillOptions)
├── doContinue()
├── doContinue(ContinueOptions)
└── abort()

Dialog                        ← browser dialog
├── message()                 → String
├── type()                    → String
├── defaultValue()            → String
├── accept()
├── accept(String promptText)
└── dismiss()

Request                       ← network request
├── url()                     → String
├── method()                  → String
├── headers()                 → Map<String,String>
└── requestId()               → String

Response                      ← network response
├── url()                     → String
├── status()                  → int
├── headers()                 → Map<String,String>
├── requestId()               → String
├── body()                    → String
└── json()                    → Object

Download                      ← file download
├── url()                     → String
├── suggestedFilename()       → String
├── saveAs(String path)
└── path()                    → String

ConsoleMessage                ← console.log message
├── type()                    → String
├── text()                    → String
├── args()                    → List<Object>
└── location()                → SourceLocation

WebSocketInfo                 ← WebSocket connection
├── url()                     → String
├── isClosed()                → boolean
├── onMessage(BiConsumer<String, String>)
└── onClose(BiConsumer<Integer, String>)
```

---

## 4. Type Mapping

### Wire → Java

Wire protocol uses camelCase. Java also uses camelCase — direct mapping for most names.

| Wire | Java |
|---|---|
| `vibium:page.navigate` | `page.go()` |
| `vibium:page.setViewport` | `page.setViewport()` |
| `vibium:element.isVisible` | `element.isVisible()` |
| `vibium:page.a11yTree` | `page.a11yTree()` |
| `vibium:network.continue` | `route.doContinue()` |

### Reserved Word Handling

| Wire | Java method | Reason |
|---|---|---|
| `continue` | `doContinue()` | `continue` is reserved |
| `wait` (duration) | `sleep()` | `Object.wait()` is final |

### Aliases

| Primary | Alias |
|---|---|
| `attr(name)` | `getAttribute(name)` |
| `bounds()` | `boundingBox()` |

---

## 5. Error Hierarchy

```
RuntimeException
└── VibiumException                    ← base, all Vibium errors
    ├── VibiumTimeoutException         ← element wait / waitForFunction timed out
    ├── VibiumConnectionException      ← pipe connection to binary failed
    ├── ElementNotFoundException       ← selector matched no elements
    ├── BrowserCrashedException        ← browser process died
    └── VibiumNotFoundException        ← vibium binary not found on system
```

### Wire Error Mapping

| Wire `error` field | Java exception |
|---|---|
| `"timeout"` | `VibiumTimeoutException` |
| message contains `"not found"` or `"no elements"` | `ElementNotFoundException` |
| process exit without response | `BrowserCrashedException` |
| pipe connection failure | `VibiumConnectionException` |

---

## 6. Thread Safety

- `BiDiClient` is thread-safe — multiple threads can call `send()` concurrently
- `Browser` is thread-safe — `page()`, `newPage()`, `stop()` can be called from any thread
- `Page` and `Element` are NOT thread-safe — use from a single thread
- Event callbacks run on the BiDi reader thread

---

## 7. Binary Bundling

### Resource Layout in JAR

```
META-INF/
com/vibium/...
natives/
  vibium-darwin-amd64
  vibium-darwin-arm64
  vibium-linux-amd64
  vibium-linux-arm64
  vibium-windows-amd64.exe
```

### Extraction

At runtime, `BinaryResolver`:
1. Checks `VIBIUM_BIN_PATH` env var
2. Checks `PATH` for `vibium` binary
3. Extracts from JAR to `java.io.tmpdir/vibium-<version>/vibium[.exe]`
4. Caches — only extracts once per version

Step 3 is safe for several JVMs sharing one cold `java.io.tmpdir`, the normal
state of a CI agent running a parallel test runner. A lock file in the cache
directory serialises publication across processes — the OS releases it if the
holder is killed — and the binary is copied to a `.part` file in that same
directory, checked against the resource length, marked executable, then renamed
onto the cache path with `ATOMIC_MOVE`. So a reader sees either no file or a
complete, executable one, and a cached artifact that is zero-length, the wrong
length, not a regular file, or not executable is re-extracted rather than
trusted.

### Platform Detection

`PlatformDetector` maps `os.name` + `os.arch` to `{darwin,linux,windows}-{amd64,arm64}`:

| `os.name` | `os.arch` | Platform key |
|---|---|---|
| `Mac OS X` | `aarch64` | `darwin-arm64` |
| `Mac OS X` | `x86_64` | `darwin-amd64` |
| `Linux` | `amd64` | `linux-amd64` |
| `Linux` | `aarch64` | `linux-arm64` |
| `Windows *` | `amd64` | `windows-amd64` |

---

## 8. Build & Publish

### Gradle Configuration

- Reads version from `../../VERSION`
- `copyNativeBinaries` task copies from `clicker/bin/` into `src/main/resources/natives/`
- `maven-publish` + `signing` plugins for Maven Central
- POM metadata: name, description, Apache 2.0 license, SCM URL, developers

### Maven Central

- Portal: https://central.sonatype.com
- Namespace: `com.vibium`
- Required artifacts: JAR + sources JAR + javadoc JAR + GPG signatures

---

## 9. Usage Examples

### Getting Started

```java
import com.vibium.Vibium;
import com.vibium.Browser;
import com.vibium.Page;
import com.vibium.Element;

Browser bro = Vibium.start();                    // visible browser
Page vibe = bro.page();                          // default page
vibe.go("https://example.com");

Element heading = vibe.find("h1");
System.out.println(heading.text());              // "Example Domain"

vibe.screenshot();                               // saves PNG
bro.stop();
```

### Headless Mode

```java
Browser bro = Vibium.start(new StartOptions().headless(true));
Page vibe = bro.page();
vibe.go("https://example.com");
System.out.println(vibe.title());
bro.stop();
```

### Element Interaction

```java
Page vibe = bro.page();
vibe.go("https://example.com/login");

vibe.find("#username").fill("user@example.com");
vibe.find("#password").fill("secret");
vibe.find("button[type=submit]").click();

vibe.waitForURL("**/dashboard");
```

### Network Interception

```java
vibe.route("**/api/**", route -> {
    route.fulfill(new FulfillOptions()
        .status(200)
        .contentType("application/json")
        .body("{\"mock\": true}"));
});
```

### Dialog Handling

```java
vibe.onDialog(dialog -> {
    System.out.println("Dialog: " + dialog.message());
    dialog.accept();
});
```
