# WebDriver Classic Protocol Reference

All HTTP endpoints from the [WebDriver spec](https://w3c.github.io/webdriver/) (editor's draft).

61 commands total.

---

## Sessions

| Method | URI Template | Command | Description |
|--------|-------------|---------|-------------|
| POST | `/session` | New Session | Creates a new WebDriver session |
| DELETE | `/session/{session id}` | Delete Session | Ends the current session |
| GET | `/status` | Status | Returns whether the remote end can create new sessions |

## Timeouts

| Method | URI Template | Command | Description |
|--------|-------------|---------|-------------|
| GET | `/session/{session id}/timeouts` | Get Timeouts | Returns the current session timeouts configuration |
| POST | `/session/{session id}/timeouts` | Set Timeouts | Sets the session timeout durations (script, pageLoad, implicit) |

## Navigation

| Method | URI Template | Command | Description |
|--------|-------------|---------|-------------|
| POST | `/session/{session id}/url` | Navigate To | Navigates the current top-level browsing context to a URL |
| GET | `/session/{session id}/url` | Get Current URL | Returns the URL of the current page |
| POST | `/session/{session id}/back` | Back | Traverses one step backward in the joint session history |
| POST | `/session/{session id}/forward` | Forward | Traverses one step forward in the joint session history |
| POST | `/session/{session id}/refresh` | Refresh | Reloads the current page |
| GET | `/session/{session id}/title` | Get Title | Returns the document title of the current page |

## Contexts

| Method | URI Template | Command | Description |
|--------|-------------|---------|-------------|
| GET | `/session/{session id}/window` | Get Window Handle | Returns the handle of the current top-level browsing context |
| DELETE | `/session/{session id}/window` | Close Window | Closes the current top-level browsing context |
| POST | `/session/{session id}/window` | Switch To Window | Switches the current top-level browsing context by handle |
| GET | `/session/{session id}/window/handles` | Get Window Handles | Returns all window handles for the session |
| POST | `/session/{session id}/window/new` | New Window | Creates a new top-level browsing context (tab or window) |
| POST | `/session/{session id}/frame` | Switch To Frame | Switches the current browsing context to a frame (by index, element, or null for top) |
| POST | `/session/{session id}/frame/parent` | Switch To Parent Frame | Sets the current browsing context to the parent of the current context |

## Window Rect

| Method | URI Template | Command | Description |
|--------|-------------|---------|-------------|
| GET | `/session/{session id}/window/rect` | Get Window Rect | Returns the size and position of the OS window |
| POST | `/session/{session id}/window/rect` | Set Window Rect | Sets the size and position of the OS window |
| POST | `/session/{session id}/window/maximize` | Maximize Window | Maximizes the OS window |
| POST | `/session/{session id}/window/minimize` | Minimize Window | Iconifies/minimizes the OS window |
| POST | `/session/{session id}/window/fullscreen` | Fullscreen Window | Makes the OS window fullscreen |

## Elements — Retrieval

| Method | URI Template | Command | Description |
|--------|-------------|---------|-------------|
| POST | `/session/{session id}/element` | Find Element | Finds a single element in the current browsing context |
| POST | `/session/{session id}/elements` | Find Elements | Finds all elements matching a selector |
| POST | `/session/{session id}/element/{element id}/element` | Find Element From Element | Finds a single element starting from a given element |
| POST | `/session/{session id}/element/{element id}/elements` | Find Elements From Element | Finds all matching elements starting from a given element |
| POST | `/session/{session id}/shadow/{shadow id}/element` | Find Element From Shadow Root | Finds a single element within a shadow root |
| POST | `/session/{session id}/shadow/{shadow id}/elements` | Find Elements From Shadow Root | Finds all matching elements within a shadow root |
| GET | `/session/{session id}/element/active` | Get Active Element | Returns the currently focused element |
| GET | `/session/{session id}/element/{element id}/shadow` | Get Element Shadow Root | Returns the shadow root of an element |

## Elements — State

| Method | URI Template | Command | Description |
|--------|-------------|---------|-------------|
| GET | `/session/{session id}/element/{element id}/selected` | Is Element Selected | Returns whether an input/option element is selected |
| GET | `/session/{session id}/element/{element id}/attribute/{name}` | Get Element Attribute | Returns the value of an element's attribute |
| GET | `/session/{session id}/element/{element id}/property/{name}` | Get Element Property | Returns the value of an element's JS property |
| GET | `/session/{session id}/element/{element id}/css/{property name}` | Get Element CSS Value | Returns the computed value of a CSS property |
| GET | `/session/{session id}/element/{element id}/text` | Get Element Text | Returns the element's rendered text |
| GET | `/session/{session id}/element/{element id}/name` | Get Element Tag Name | Returns the element's tag name |
| GET | `/session/{session id}/element/{element id}/rect` | Get Element Rect | Returns the element's bounding rectangle (x, y, width, height) |
| GET | `/session/{session id}/element/{element id}/enabled` | Is Element Enabled | Returns whether a form control element is enabled |
| GET | `/session/{session id}/element/{element id}/computedrole` | Get Computed Role | Returns the WAI-ARIA role of the element |
| GET | `/session/{session id}/element/{element id}/computedlabel` | Get Computed Label | Returns the accessible name of the element |

## Elements — Interaction

| Method | URI Template | Command | Description |
|--------|-------------|---------|-------------|
| POST | `/session/{session id}/element/{element id}/click` | Element Click | Scrolls into view and clicks the element's center point |
| POST | `/session/{session id}/element/{element id}/clear` | Element Clear | Clears a content-editable or resettable form element |
| POST | `/session/{session id}/element/{element id}/value` | Element Send Keys | Sends keystrokes to an element (or sets files on an input[type=file]) |

## Document

| Method | URI Template | Command | Description |
|--------|-------------|---------|-------------|
| GET | `/session/{session id}/source` | Get Page Source | Returns the serialized DOM of the active document |
| POST | `/session/{session id}/execute/sync` | Execute Script | Executes a JavaScript function body synchronously |
| POST | `/session/{session id}/execute/async` | Execute Async Script | Executes a JavaScript function body with an async completion callback |

## Cookies

| Method | URI Template | Command | Description |
|--------|-------------|---------|-------------|
| GET | `/session/{session id}/cookie` | Get All Cookies | Returns all cookies visible to the current page |
| GET | `/session/{session id}/cookie/{name}` | Get Named Cookie | Returns a cookie by name |
| POST | `/session/{session id}/cookie` | Add Cookie | Creates a cookie for the current page |
| DELETE | `/session/{session id}/cookie/{name}` | Delete Cookie | Deletes a cookie by name |
| DELETE | `/session/{session id}/cookie` | Delete All Cookies | Deletes all cookies for the current page |

## Actions

| Method | URI Template | Command | Description |
|--------|-------------|---------|-------------|
| POST | `/session/{session id}/actions` | Perform Actions | Dispatches tick-based input actions (key, pointer, wheel) |
| DELETE | `/session/{session id}/actions` | Release Actions | Releases all keys and pointer buttons, resets input state |

## User Prompts

| Method | URI Template | Command | Description |
|--------|-------------|---------|-------------|
| POST | `/session/{session id}/alert/dismiss` | Dismiss Alert | Dismisses (cancels) the current user prompt |
| POST | `/session/{session id}/alert/accept` | Accept Alert | Accepts (OKs) the current user prompt |
| GET | `/session/{session id}/alert/text` | Get Alert Text | Returns the message of the current user prompt |
| POST | `/session/{session id}/alert/text` | Send Alert Text | Sets the text field of a window.prompt dialog |

## Screen Capture

| Method | URI Template | Command | Description |
|--------|-------------|---------|-------------|
| GET | `/session/{session id}/screenshot` | Take Screenshot | Captures the viewport as a Base64 PNG |
| GET | `/session/{session id}/element/{element id}/screenshot` | Take Element Screenshot | Captures an element's bounding rect as a Base64 PNG |

## Print

| Method | URI Template | Command | Description |
|--------|-------------|---------|-------------|
| POST | `/session/{session id}/print` | Print Page | Renders the page to PDF, returned as Base64 |
