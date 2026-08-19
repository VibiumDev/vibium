# Vibium API Reference

Command reference for every Vibium surface: wire protocol, CLI, MCP tools, JS client, Python client, and Java client.

**Legend:** Filled cell = implemented. `⬜` = planned, not yet done. `—` = not applicable for this surface.

For what goes *in* the `<sel>` argument — CSS, shadow-DOM pierce combinators, and semantic locators — see [Selectors](selectors.md).

---

## Browser

| # | Description | Wire Command | CLI | MCP | JS | Python | Java |
|---|---|---|---|---|---|---|---|
| 1 | Launch browser and connect | *binary launch + WS connect* | `vibium start` | `browser_start` | `browser.start(opts?)` | `browser.start(opts?)` | `Vibium.start(options?)` |
| 2 | Get the default page | `vibium:browser.page` | — | — | `browser.page()` | `browser.page()` | `browser.page()` |
| 3 | Create a new page | `vibium:browser.newPage` | `vibium page new` | `browser_new_page` | `browser.newPage()` | `browser.new_page()` | `browser.newPage()` |
| 4 | Create a new browser context | `vibium:browser.newContext` | — | — | `browser.newContext()` | `browser.new_context()` | `browser.newContext()` |
| 5 | List all pages | `vibium:browser.pages` | `vibium pages` | `browser_list_pages` | `browser.pages()` | `browser.pages()` | `browser.pages()` |
| 6 | Stop the browser | `vibium:browser.stop` | `vibium stop` | `browser_stop` | `browser.stop()` | `browser.stop()` | `browser.stop()` |
| 7 | Listen for new page events | *client-side event listener* | — | — | `browser.onPage(cb)` | `browser.on_page(cb)` | `browser.onPage(cb)` |
| 8 | Listen for popup events | *client-side event listener* | — | — | `browser.onPopup(cb)` | `browser.on_popup(cb)` | `browser.onPopup(cb)` |
| 9 | Remove all event listeners | *client-side* | — | — | `browser.removeAllListeners(ev?)` | `browser.remove_all_listeners(ev?)` | `browser.removeAllListeners(ev?)` |

## Page

| # | Description | Wire Command | CLI | MCP | JS | Python | Java |
|---|---|---|---|---|---|---|---|
| 10 | Navigate to a URL | `vibium:page.navigate` | `vibium go <url>` | `browser_navigate` | `page.go(url)` | `page.go(url)` | `page.go(url)` |
| 11 | Go back | `vibium:page.back` | `vibium back` | `browser_back` | `page.back()` | `page.back()` | `page.back()` |
| 12 | Go forward | `vibium:page.forward` | `vibium forward` | `browser_forward` | `page.forward()` | `page.forward()` | `page.forward()` |
| 13 | Reload page | `vibium:page.reload` | `vibium reload` | `browser_reload` | `page.reload()` | `page.reload()` | `page.reload()` |
| 14 | Get current URL | `vibium:page.url` | `vibium url` | `browser_get_url` | `page.url()` | `page.url()` | `page.url()` |
| 15 | Get page title | `vibium:page.title` | `vibium title` | `browser_get_title` | `page.title()` | `page.title()` | `page.title()` |
| 16 | Get page HTML | `vibium:page.content` | — | — | `page.content()` | `page.content()` | `page.content()` |
| 17 | Find a single element | `vibium:page.find` | `vibium find <sel>` | `browser_find` | `page.find(sel, opts?)` | `page.find(sel, **opts)` | `page.find(sel, options?)` |
| 18 | Find all matching elements | `vibium:page.findAll` | `vibium find --all <sel>` | `browser_find_all` | `page.findAll(sel, opts?)` | `page.find_all(sel, **opts)` | `page.findAll(sel, options?)` |
| 19 | Take a page screenshot | `vibium:page.screenshot` | `vibium screenshot` | `browser_screenshot` | `page.screenshot(opts?)` | `page.screenshot(opts?)` | `page.screenshot(options?)` |
| 20 | Generate PDF | `vibium:page.pdf` | `vibium pdf` | `browser_pdf` | `page.pdf()` | `page.pdf()` | `page.pdf()` |
| 21 | Evaluate JavaScript | `vibium:page.eval` | `vibium eval <expr>` | `browser_evaluate` | `page.evaluate(expr)` | `page.evaluate(expr)` | `page.evaluate(expr)` |
| 22 | Add a script tag | `vibium:page.addScript` | ⬜ | ⬜ | `page.addScript(src)` | `page.add_script(src)` | `page.addScript(src)` |
| 23 | Add a style tag | `vibium:page.addStyle` | ⬜ | ⬜ | `page.addStyle(src)` | `page.add_style(src)` | `page.addStyle(src)` |
| 24 | Expose a function to the page | `vibium:page.expose` | — | — | `page.expose(name, fn)` | `page.expose(name, fn)` | `page.expose(name, fn)` |
| 25 | Wait for a duration | `vibium:page.wait` | `vibium sleep <ms>` | `browser_sleep` | `page.wait(ms)` | `page.wait(ms)` | `page.sleep(ms)` |
| 26 | Wait for a selector | `vibium:page.waitFor` | `vibium wait <sel>` | `browser_wait` | `page.waitFor(sel, opts?)` | `page.wait_for(sel, **opts)` | `page.waitFor(sel, options?)` |
| 27 | Wait for JS function to return truthy | `vibium:page.waitForFunction` | `vibium wait fn <expr>` | `browser_wait_for_fn` | `page.waitForFunction(fn, opts?)` | `page.wait_for_function(fn, **opts)` | `page.waitForFunction(fn, options?)` |
| 28 | Wait for URL to match | `vibium:page.waitForURL` | `vibium wait url <pat>` | `browser_wait_for_url` | `page.waitForURL(url, opts?)` | `page.wait_for_url(url, **opts)` | `page.waitForURL(url, options?)` |
| 29 | Wait for page load | `vibium:page.waitForLoad` | `vibium wait load` | `browser_wait_for_load` | `page.waitForLoad(opts?)` | `page.wait_for_load(**opts)` | `page.waitForLoad(options?)` |
| 30 | Scroll the page | `vibium:page.scroll` | `vibium scroll <dir> <amt>` | `browser_scroll` | `page.scroll(dir?, amt?, sel?)` | `page.scroll(dir?, amt?, sel?)` | `page.scroll(options?)` |
| 31 | Set viewport size | `vibium:page.setViewport` | `vibium viewport <w> <h>` | `browser_set_viewport` | `page.setViewport(size)` | `page.set_viewport(size)` | `page.setViewport(size)` |
| 32 | Get viewport size | `vibium:page.viewport` | `vibium viewport` | `browser_get_viewport` | `page.viewport()` | `page.viewport()` | `page.viewport()` |
| 33 | Override CSS media features | `vibium:page.emulateMedia` | `vibium media <scheme>` | `browser_emulate_media` | `page.emulateMedia(opts)` | `page.emulate_media(**opts)` | `page.emulateMedia(opts)` |
| 34 | Set page HTML | `vibium:page.setContent` | `vibium content <html>` | `browser_set_content` | `page.setContent(html)` | `page.set_content(html)` | `page.setContent(html)` |
| 35 | Override geolocation | `vibium:page.setGeolocation` | `vibium geolocation <lat> <lon>` | `browser_set_geolocation` | `page.setGeolocation(coords)` | `page.set_geolocation(coords)` | `page.setGeolocation(coords)` |
| 36 | Set window size/position | `vibium:page.setWindow` | `vibium window <opts>` | `browser_set_window` | `page.setWindow(opts)` | `page.set_window(**opts)` | `page.setWindow(opts)` |
| 37 | Get window info | `vibium:page.window` | `vibium window` | `browser_get_window` | `page.window()` | `page.window()` | `page.window()` |
| 38 | Get accessibility tree | `vibium:page.a11yTree` | `vibium a11y-tree` | `browser_a11y_tree` | `page.a11yTree(opts?)` | `page.a11y_tree(opts?)` | `page.a11yTree(options?)` |
| 39 | List all frames | `vibium:page.frames` | `vibium frames` | `browser_frames` | `page.frames()` | `page.frames()` | `page.frames()` |
| 40 | Get a frame by name/URL | `vibium:page.frame` | `vibium frame <ref>` | `browser_frame` | `page.frame(nameOrUrl)` | `page.frame(name_or_url)` | `page.frame(nameOrUrl)` |
| 41 | Get the main frame | *returns self (top frame)* | — | — | `page.mainFrame()` | `page.main_frame()` | `page.mainFrame()` |
| 42 | Bring page to front | `browsingContext.activate` | `vibium page switch <idx>` | `browser_switch_page` | `page.bringToFront()` | `page.bring_to_front()` | `page.bringToFront()` |
| 43 | Close the page | `browsingContext.close` | `vibium page close` | `browser_close_page` | `page.close()` | `page.close()` | `page.close()` |
| 44 | Register a route handler | `vibium:page.route` | — | — | `page.route(pattern, handler)` | `page.route(pattern, handler)` | `page.route(pattern, handler)` |
| 45 | Remove a route handler | `network.removeIntercept` | — | — | `page.unroute(pattern)` | `page.unroute(pattern)` | `page.unroute(pattern)` |
| 46 | Set extra HTTP headers | `vibium:page.setHeaders` | ⬜ | ⬜ | `page.setHeaders(headers)` | `page.set_headers(headers)` | `page.setHeaders(headers)` |
| 47 | Listen for requests | *client-side event listener* | — | — | `page.onRequest(fn)` | `page.on_request(fn)` | `page.onRequest(fn)` |
| 48 | Listen for responses | *client-side event listener* | — | — | `page.onResponse(fn)` | `page.on_response(fn)` | `page.onResponse(fn)` |
| 49 | Listen for dialogs | *client-side event listener* | — | — | `page.onDialog(fn)` | `page.on_dialog(fn)` | `page.onDialog(fn)` |
| 50 | Listen for console messages | *client-side event listener* | — | — | `page.onConsole(fn)` | `page.on_console(fn)` | `page.onConsole(fn)` |
| 51 | Listen for page errors | *client-side event listener* | — | — | `page.onError(fn)` | `page.on_error(fn)` | `page.onError(fn)` |
| 52 | Listen for downloads | *client-side event listener* | — | — | `page.onDownload(fn)` | `page.on_download(fn)` | `page.onDownload(fn)` |
| 53 | Subscribe to WebSocket events | `vibium:page.onWebSocket` | — | — | `page.onWebSocket(fn)` | `page.on_web_socket(fn)` | `page.onWebSocket(fn)` |
| 54 | Remove all event listeners | *client-side* | — | — | `page.removeAllListeners(ev?)` | `page.remove_all_listeners(ev?)` | `page.removeAllListeners(ev?)` |
| 55 | Capture response (before action) | *client-side* | — | — | `page.capture.response(pat, fn?)` | `page.capture.response(pat, fn?)` | ⬜ |
| 56 | Capture request (before action) | *client-side* | — | — | `page.capture.request(pat, fn?)` | `page.capture.request(pat, fn?)` | ⬜ |
| 57 | Capture navigation (before action) | *client-side* | — | — | `page.capture.navigation(fn?)` | `page.capture.navigation(fn?)` | ⬜ |
| 58 | Capture event (before action) | *client-side* | — | — | `page.capture.event(name, fn?)` | `page.capture.event(name, fn?)` | ⬜ |
| 59 | Capture download (before action) | *client-side* | — | — | `page.capture.download(fn?)` | `page.capture.download(fn?)` | ⬜ |
| 60 | Capture dialog (before action) | *client-side* | — | — | `page.capture.dialog(fn?)` | `page.capture.dialog(fn?)` | ⬜ |
| 61 | Get buffered console messages | *client-side* | — | — | `page.consoleMessages()` | `page.console_messages()` | `page.consoleMessages()` |
| 62 | Get buffered page errors | *client-side* | — | — | `page.errors()` | `page.errors()` | `page.errors()` |

## Element

| # | Description | Wire Command | CLI | MCP | JS | Python | Java |
|---|---|---|---|---|---|---|---|
| 63 | Click an element | `vibium:element.click` | `vibium click <sel>` | `browser_click` | `el.click(opts?)` | `el.click(timeout?)` | `el.click()` |
| 64 | Double-click an element | `vibium:element.dblclick` | `vibium dblclick <sel>` | `browser_dblclick` | `el.dblclick(opts?)` | `el.dblclick(timeout?)` | `el.dblclick()` |
| 65 | Fill an input field | `vibium:element.fill` | `vibium fill <sel> <val>` | `browser_fill` | `el.fill(value, opts?)` | `el.fill(value, timeout?)` | `el.fill(value)` |
| 66 | Type text character by character | `vibium:element.type` | `vibium type <text>` | `browser_type` | `el.type(text, opts?)` | `el.type(text, timeout?)` | `el.type(text)` |
| 67 | Press a key on focused element | `vibium:element.press` | `vibium press <key>` | `browser_press` | `el.press(key, opts?)` | `el.press(key, timeout?)` | `el.press(key)` |
| 68 | Clear an input field | `vibium:element.clear` | — | — | `el.clear(opts?)` | `el.clear(timeout?)` | `el.clear()` |
| 69 | Check a checkbox | `vibium:element.check` | `vibium check <sel>` | `browser_check` | `el.check(opts?)` | `el.check(timeout?)` | `el.check()` |
| 70 | Uncheck a checkbox | `vibium:element.uncheck` | `vibium uncheck <sel>` | `browser_uncheck` | `el.uncheck(opts?)` | `el.uncheck(timeout?)` | `el.uncheck()` |
| 71 | Select a dropdown option | `vibium:element.selectOption` | `vibium select <sel> <val>` | `browser_select` | `el.selectOption(val, opts?)` | `el.select_option(val, timeout?)` | `el.selectOption(val)` |
| 72 | Hover over an element | `vibium:element.hover` | `vibium hover <sel>` | `browser_hover` | `el.hover(opts?)` | `el.hover(timeout?)` | `el.hover()` |
| 73 | Focus an element | `vibium:element.focus` | `vibium focus <sel>` | `browser_focus` | `el.focus(opts?)` | `el.focus(timeout?)` | `el.focus()` |
| 74 | Drag an element to a target | `vibium:element.dragTo` | `vibium drag <sel> <x> <y>` | `browser_drag` | `el.dragTo(target, opts?)` | `el.drag_to(target, timeout?)` | `el.dragTo(target)` |
| 75 | Tap an element (touch) | `vibium:element.tap` | — | — | `el.tap(opts?)` | `el.tap(timeout?)` | `el.tap()` |
| 76 | Scroll element into view | `vibium:element.scrollIntoView` | `vibium scroll into-view <sel>` | `browser_scroll_into_view` | `el.scrollIntoView(opts?)` | `el.scroll_into_view(timeout?)` | `el.scrollIntoView()` |
| 77 | Dispatch a DOM event | `vibium:element.dispatchEvent` | — | — | `el.dispatchEvent(type, init?)` | `el.dispatch_event(type, init?)` | `el.dispatchEvent(type, init?)` |
| 78 | Set files on a file input | `vibium:element.setFiles` | `vibium upload <sel> <paths>` | `browser_upload` | `el.setFiles(files, opts?)` | `el.set_files(files, timeout?)` | `el.setFiles(files)` |
| 79 | Highlight an element | `vibium:element.highlight` | `vibium highlight <sel>` | `browser_highlight` | `el.highlight()` | `el.highlight()` | `el.highlight()` |
| 80 | Get element text content | `vibium:element.text` | `vibium text <sel>` | `browser_get_text` | `el.text()` | `el.text()` | `el.text()` |
| 81 | Get element inner text | `vibium:element.innerText` | `vibium text <sel>` | `browser_get_text` | `el.innerText()` | `el.inner_text()` | `el.innerText()` |
| 82 | Get element outer HTML | `vibium:element.html` | `vibium html <sel>` | `browser_get_html` | `el.html()` | `el.html()` | `el.html()` |
| 83 | Get input element value | `vibium:element.value` | `vibium value <sel>` | `browser_get_value` | `el.value()` | `el.value()` | `el.value()` |
| 84 | Get element attribute | `vibium:element.attr` | `vibium attr <sel> <name>` | `browser_get_attribute` | `el.attr(name)` | `el.attr(name)` | `el.attr(name)` |
| 85 | Get element bounding box | `vibium:element.bounds` | — | — | `el.bounds()` | `el.bounds()` | `el.bounds()` |
| 86 | Check if element is visible | `vibium:element.isVisible` | `vibium is visible <sel>` | `browser_is_visible` | `el.isVisible()` | `el.is_visible()` | `el.isVisible()` |
| 87 | Check if element is hidden | `vibium:element.isHidden` | — | — | `el.isHidden()` | `el.is_hidden()` | `el.isHidden()` |
| 88 | Check if element is enabled | `vibium:element.isEnabled` | `vibium is enabled <sel>` | `browser_is_enabled` | `el.isEnabled()` | `el.is_enabled()` | `el.isEnabled()` |
| 89 | Check if element is checked | `vibium:element.isChecked` | `vibium is checked <sel>` | `browser_is_checked` | `el.isChecked()` | `el.is_checked()` | `el.isChecked()` |
| 90 | Check if element is editable | `vibium:element.isEditable` | ⬜ | ⬜ | `el.isEditable()` | `el.is_editable()` | `el.isEditable()` |
| 91 | Get element ARIA role | `vibium:element.role` | — | — | `el.role()` | `el.role()` | `el.role()` |
| 92 | Get element accessible label | `vibium:element.label` | — | — | `el.label()` | `el.label()` | `el.label()` |
| 93 | Screenshot an element | `vibium:element.screenshot` | — | — | `el.screenshot()` | `el.screenshot()` | `el.screenshot()` |
| 94 | Wait for element state | `vibium:element.waitFor` | `vibium wait <sel> --state <st>` | `browser_wait` | `el.waitUntil(state?, opts?)` | `el.wait_until(state?, timeout?)` | `el.waitUntil(state?, options?)` |
| 95 | Find a child element (scoped) | `vibium:element.find` | — | — | `el.find(sel, opts?)` | `el.find(sel, **opts)` | `el.find(sel, options?)` |
| 96 | Find all child elements (scoped) | `vibium:element.findAll` | — | — | `el.findAll(sel, opts?)` | `el.find_all(sel, **opts)` | `el.findAll(sel, options?)` |

## BrowserContext

| # | Description | Wire Command | CLI | MCP | JS | Python | Java |
|---|---|---|---|---|---|---|---|
| 97 | Create a page in a context | `vibium:context.newPage` | — | — | `context.newPage()` | `context.new_page()` | `context.newPage()` |
| 98 | Close the context | `browser.removeUserContext` | — | — | `context.close()` | `context.close()` | `context.close()` |
| 99 | Get cookies | `vibium:context.cookies` | `vibium cookies` | `browser_get_cookies` | `context.cookies(urls?)` | `context.cookies(urls?)` | `context.cookies(urls?)` |
| 100 | Set cookies | `vibium:context.setCookies` | `vibium cookies <n> <v>` | `browser_set_cookie` | `context.setCookies(cookies)` | `context.set_cookies(cookies)` | `context.setCookies(cookies)` |
| 101 | Clear cookies | `vibium:context.clearCookies` | `vibium cookies clear` | `browser_delete_cookies` | `context.clearCookies()` | `context.clear_cookies()` | `context.clearCookies()` |
| 102 | Get storage state | `vibium:context.storage` | `vibium storage` | `browser_storage_state` | `context.storage()` | `context.storage()` | `context.storage()` |
| 103 | Set storage state | `vibium:context.setStorage` | — | `browser_restore_storage` | `context.setStorage(state)` | `context.set_storage(state)` | `context.setStorage(state)` |
| 104 | Clear all storage | `vibium:context.clearStorage` | — | — | `context.clearStorage()` | `context.clear_storage()` | `context.clearStorage()` |
| 105 | Add an init script | `vibium:context.addInitScript` | — | — | `context.addInitScript(script)` | `context.add_init_script(script)` | `context.addInitScript(script)` |

## Keyboard

| # | Description | Wire Command | CLI | MCP | JS | Python | Java |
|---|---|---|---|---|---|---|---|
| 106 | Press a key | `vibium:keyboard.press` | `vibium keys <keys>` | `browser_keys` | `keyboard.press(key)` | `keyboard.press(key)` | `keyboard.press(key)` |
| 107 | Key down | `vibium:keyboard.down` | — | — | `keyboard.down(key)` | `keyboard.down(key)` | `keyboard.down(key)` |
| 108 | Key up | `vibium:keyboard.up` | — | — | `keyboard.up(key)` | `keyboard.up(key)` | `keyboard.up(key)` |
| 109 | Type text | `vibium:keyboard.type` | — | — | `keyboard.type(text)` | `keyboard.type(text)` | `keyboard.type(text)` |

## Mouse

| # | Description | Wire Command | CLI | MCP | JS | Python | Java |
|---|---|---|---|---|---|---|---|
| 110 | Click at coordinates | `vibium:mouse.click` | `vibium mouse click <x> <y>` | `browser_mouse_click` | `mouse.click(x, y, opts?)` | `mouse.click(x, y, **opts)` | `mouse.click(x, y, options?)` |
| 111 | Move mouse | `vibium:mouse.move` | `vibium mouse move <x> <y>` | `browser_mouse_move` | `mouse.move(x, y, opts?)` | `mouse.move(x, y, **opts)` | `mouse.move(x, y, options?)` |
| 112 | Mouse button down | `vibium:mouse.down` | `vibium mouse down` | `browser_mouse_down` | `mouse.down(opts?)` | `mouse.down(**opts)` | `mouse.down(options?)` |
| 113 | Mouse button up | `vibium:mouse.up` | `vibium mouse up` | `browser_mouse_up` | `mouse.up(opts?)` | `mouse.up(**opts)` | `mouse.up(options?)` |
| 114 | Scroll mouse wheel | `vibium:mouse.wheel` | — | ⬜ | `mouse.wheel(dx, dy)` | `mouse.wheel(dx, dy)` | `mouse.wheel(dx, dy)` |

## Touch

| # | Description | Wire Command | CLI | MCP | JS | Python | Java |
|---|---|---|---|---|---|---|---|
| 115 | Tap at coordinates | `vibium:touch.tap` | — | — | `touch.tap(x, y)` | `touch.tap(x, y)` | `touch.tap(x, y)` |

## Clock

| # | Description | Wire Command | CLI | MCP | JS | Python | Java |
|---|---|---|---|---|---|---|---|
| 116 | Install fake timers | `vibium:clock.install` | — | `page_clock_install` | `clock.install(opts?)` | `clock.install(time?, timezone?)` | `clock.install(options?)` |
| 117 | Fast-forward time | `vibium:clock.fastForward` | — | `page_clock_fast_forward` | `clock.fastForward(ticks)` | `clock.fast_forward(ticks)` | `clock.fastForward(ticks)` |
| 118 | Run timers for a duration | `vibium:clock.runFor` | — | `page_clock_run_for` | `clock.runFor(ticks)` | `clock.run_for(ticks)` | `clock.runFor(ticks)` |
| 119 | Pause clock at a time | `vibium:clock.pauseAt` | — | `page_clock_pause_at` | `clock.pauseAt(time)` | `clock.pause_at(time)` | `clock.pauseAt(time)` |
| 120 | Resume clock | `vibium:clock.resume` | — | `page_clock_resume` | `clock.resume()` | `clock.resume()` | `clock.resume()` |
| 121 | Set fixed fake time | `vibium:clock.setFixedTime` | — | `page_clock_set_fixed_time` | `clock.setFixedTime(time)` | `clock.set_fixed_time(time)` | `clock.setFixedTime(time)` |
| 122 | Set system time | `vibium:clock.setSystemTime` | — | `page_clock_set_system_time` | `clock.setSystemTime(time)` | `clock.set_system_time(time)` | `clock.setSystemTime(time)` |
| 123 | Set timezone | `vibium:clock.setTimezone` | — | `page_clock_set_timezone` | `clock.setTimezone(tz)` | `clock.set_timezone(tz)` | `clock.setTimezone(tz)` |

## Recording

| # | Description | Wire Command | CLI | MCP | JS | Python | Java |
|---|---|---|---|---|---|---|---|
| 124 | Start recording | `vibium:recording.start` | `vibium record start` | `browser_record_start` | `recording.start(opts?)` | `recording.start(opts?)` | `recording.start(options?)` |
| 125 | Stop recording, return trace | `vibium:recording.stop` | `vibium record stop` | `browser_record_stop` | `recording.stop(opts?)` | `recording.stop(path?)` | `recording.stop(path?)` |
| 126 | Start a recording chunk | `vibium:recording.startChunk` | `vibium record chunk start` | `browser_record_start_chunk` | `recording.startChunk(opts?)` | `recording.start_chunk(opts?)` | `recording.startChunk(options?)` |
| 127 | Stop a recording chunk | `vibium:recording.stopChunk` | `vibium record chunk stop` | `browser_record_stop_chunk` | `recording.stopChunk(opts?)` | `recording.stop_chunk(path?)` | `recording.stopChunk(path?)` |
| 128 | Start a logical group | `vibium:recording.startGroup` | `vibium record group start <name>` | `browser_record_start_group` | `recording.startGroup(name, opts?)` | `recording.start_group(name, location?)` | `recording.startGroup(name, location?)` |
| 129 | Stop a logical group | `vibium:recording.stopGroup` | `vibium record group stop` | `browser_record_stop_group` | `recording.stopGroup()` | `recording.stop_group()` | `recording.stopGroup()` |

`recording.start` accepts a `video` option (`true`/`false`/`{width, height, frameRate}`) that adds a native browser video track to the recording zip (Firefox 154+, local browsers), and a `path` declaring where the zip lands at stop (default: timestamped `record-YYYYMMDD-HHMMSS.zip`; `null` for bytes-only). `recording.stop` returns `{path, steps, durationMs, videos | videoUnavailable}`, with the zip bytes included only for bytes-only recordings. See [Record Video](../how-to-guides/record-video.md).

## Route

| # | Description | Wire Command | CLI | MCP | JS | Python | Java |
|---|---|---|---|---|---|---|---|
| 130 | Access intercepted request | — | — | — | `route.request` (property) | *passed via callback args* | `route.request()` |
| 131 | Fulfill an intercepted request | `vibium:network.fulfill` | — | — | `route.fulfill(resp?)` | `route.fulfill(status?, headers?, ...)` | `route.fulfill(options?)` |
| 132 | Continue an intercepted request | `vibium:network.continue` | — | — | `route.continue(overrides?)` | `route.continue_(overrides?)` | `route.doContinue(options?)` |
| 133 | Abort an intercepted request | `vibium:network.abort` | — | — | `route.abort()` | `route.abort()` | `route.abort()` |

## Dialog

| # | Description | Wire Command | CLI | MCP | JS | Python | Java |
|---|---|---|---|---|---|---|---|
| 134 | Get dialog message | *from event data* | — | — | `dialog.message()` | `dialog.message()` | `dialog.message()` |
| 135 | Get dialog type | *from event data* | — | — | `dialog.type()` | `dialog.type()` | `dialog.type()` |
| 136 | Get dialog default value | *from event data* | — | — | `dialog.defaultValue()` | `dialog.default_value()` | `dialog.defaultValue()` |
| 137 | Accept the dialog | `browsingContext.handleUserPrompt` | `vibium dialog accept` | `browser_dialog_accept` | `dialog.accept(promptText?)` | `dialog.accept(prompt_text?)` | `dialog.accept(promptText?)` |
| 138 | Dismiss the dialog | `browsingContext.handleUserPrompt` | `vibium dialog dismiss` | `browser_dialog_dismiss` | `dialog.dismiss()` | `dialog.dismiss()` | `dialog.dismiss()` |

## Download

| # | Description | Wire Command | CLI | MCP | JS | Python | Java |
|---|---|---|---|---|---|---|---|
| 139 | Save a download to path | `vibium:download.saveAs` | ⬜ | ⬜ | `download.saveAs(path)` | `download.save_as(path)` | `download.saveAs(path)` |
| 140 | Get download URL | *from event data* | — | — | `download.url()` | `download.url()` | `download.url()` |
| 141 | Get download filename | *from event data* | — | — | `download.filename()` | `download.filename()` | `download.suggestedFilename()` |
| 142 | Get download path | *from event data* | — | — | `download.path()` | `download.path()` | `download.path()` |

## Request

| # | Description | Wire Command | CLI | MCP | JS | Python | Java |
|---|---|---|---|---|---|---|---|
| 143 | Get request URL | *from event data* | — | — | `request.url()` | `request.url()` | `request.url()` |
| 144 | Get HTTP method | *from event data* | — | — | `request.method()` | `request.method()` | `request.method()` |
| 145 | Get request headers | *from event data* | — | — | `request.headers()` | `request.headers()` | `request.headers()` |
| 146 | Get BiDi request ID | *from event data* | — | — | `request.requestId()` | `request.request_id()` | `request.requestId()` |
| 147 | Get request body | `network.getData` | — | — | `request.postData()` | `request.post_data()` | — |

## Response

| # | Description | Wire Command | CLI | MCP | JS | Python | Java |
|---|---|---|---|---|---|---|---|
| 148 | Get response URL | *from event data* | — | — | `response.url()` | `response.url()` | `response.url()` |
| 149 | Get HTTP status | *from event data* | — | — | `response.status()` | `response.status()` | `response.status()` |
| 150 | Get response headers | *from event data* | — | — | `response.headers()` | `response.headers()` | `response.headers()` |
| 151 | Get associated request ID | *from event data* | — | — | `response.requestId()` | `response.request_id()` | `response.requestId()` |
| 152 | Get response body | `network.getData` | — | — | `response.body()` | `response.body()` | `response.body()` |
| 153 | Parse response body as JSON | `network.getData` | — | — | `response.json()` | `response.json()` | `response.json()` |

## ConsoleMessage

| # | Description | Wire Command | CLI | MCP | JS | Python | Java |
|---|---|---|---|---|---|---|---|
| 154 | Get console message type | *from event data* | — | — | `message.type()` | `message.type()` | `message.type()` |
| 155 | Get formatted message text | *from event data* | — | — | `message.text()` | `message.text()` | `message.text()` |
| 156 | Get console arguments | *from event data* | — | — | `message.args()` | `message.args()` | `message.args()` |
| 157 | Get source location | *from event data* | — | — | `message.location()` | `message.location()` | `message.location()` |

## WebSocketInfo

| # | Description | Wire Command | CLI | MCP | JS | Python | Java |
|---|---|---|---|---|---|---|---|
| 158 | Get WebSocket URL | *from event data* | — | — | `socket.url()` | `socket.url()` | `socket.url()` |
| 159 | Listen for messages | *client-side event listener* | — | — | `socket.onMessage(fn)` | `socket.on_message(fn)` | `socket.onMessage(fn)` |
| 160 | Listen for close | *client-side event listener* | — | — | `socket.onClose(fn)` | `socket.on_close(fn)` | `socket.onClose(fn)` |
| 161 | Check whether socket is closed | *client-side* | — | — | `socket.isClosed()` | `socket.is_closed()` | `socket.isClosed()` |

## Agent & CLI Extras

MCP/CLI-only tools with no direct client API equivalent.

| # | Description | Wire Command | CLI | MCP | JS | Python | Java |
|---|---|---|---|---|---|---|---|
| 162 | Map interactive page elements with @refs | — | `vibium map` | `browser_map` | — | — | — |
| 163 | Diff page state vs last map | — | `vibium diff` | `browser_diff_map` | — | — | — |
| 164 | Count elements matching selector | — | `vibium count <sel>` | `browser_count` | — | — | — |
| 165 | Wait for text to appear on page | — | `vibium wait text <text>` | `browser_wait_for_text` | — | — | — |
| 166 | Set the download directory | — | `vibium download dir <path>` | `browser_download_set_dir` | — | — | — |

## AI-Native (Planned)

| # | Description | Wire Command | CLI | MCP | JS | Python | Java |
|---|---|---|---|---|---|---|---|
| 167 | Assert a visual claim | *TBD* | ⬜ | ⬜ | `page.check(claim)` | `page.check(claim)` | ⬜ |
| 168 | Perform a natural language action | *TBD* | ⬜ | ⬜ | `page.do(action)` | `page.do(action)` | ⬜ |
| 169 | NL action with data extraction | *TBD* | ⬜ | ⬜ | `page.do(action, {data})` | `page.do(action, data=...)` | ⬜ |

---

**Total: 169 commands**
