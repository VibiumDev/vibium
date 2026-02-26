"""Console & error event tests — onConsole, onError, removeAllListeners (8 async tests)."""

import pytest


# --- Console ---

async def test_console_log(fresh_async_browser, test_server):
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server)
    messages = []
    vibe.on_console(lambda msg: messages.append({"type": msg.type(), "text": msg.text()}))
    await vibe.eval('console.log("hello")')
    await vibe.wait(300)
    assert any(m["text"] == "hello" and m["type"] == "log" for m in messages)


async def test_console_warn(fresh_async_browser, test_server):
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server)
    messages = []
    vibe.on_console(lambda msg: messages.append({"type": msg.type(), "text": msg.text()}))
    await vibe.eval('console.warn("warning!")')
    await vibe.wait(300)
    assert any(m["type"] == "warn" for m in messages)


async def test_console_error_not_on_error(fresh_async_browser, test_server):
    """console.error fires onConsole, not onError."""
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server)
    console_msgs = []
    error_msgs = []
    vibe.on_console(lambda msg: console_msgs.append(msg.text()))
    vibe.on_error(lambda err: error_msgs.append(str(err)))
    await vibe.eval('console.error("console error")')
    await vibe.wait(300)
    assert any("console error" in m for m in console_msgs)
    # console.error should NOT trigger onError
    assert not any("console error" in m for m in error_msgs)


async def test_console_args(fresh_async_browser, test_server):
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server)
    texts = []
    vibe.on_console(lambda msg: texts.append(msg.text()))
    await vibe.eval('console.log("a", "b", "c")')
    await vibe.wait(300)
    # The text should contain all args joined
    assert any("a" in t for t in texts)


# --- Error ---

async def test_uncaught_error(fresh_async_browser, test_server):
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server)
    errors = []
    vibe.on_error(lambda err: errors.append(str(err)))
    await vibe.eval('setTimeout(() => { throw new Error("uncaught!") }, 0)')
    await vibe.wait(500)
    assert any("uncaught!" in e for e in errors)


async def test_error_not_for_console_error(fresh_async_browser, test_server):
    """onError should NOT fire for console.error."""
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server)
    errors = []
    vibe.on_error(lambda err: errors.append(str(err)))
    await vibe.eval('console.error("not a real error")')
    await vibe.wait(300)
    assert not any("not a real error" in e for e in errors)


# --- Remove ---

async def test_remove_console(fresh_async_browser, test_server):
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server)
    messages = []
    vibe.on_console(lambda msg: messages.append(msg.text()))
    await vibe.eval('console.log("before")')
    await vibe.wait(200)
    vibe.remove_all_listeners("console")
    await vibe.eval('console.log("after")')
    await vibe.wait(200)
    assert any("before" in m for m in messages)
    # "after" should not be captured because listener was removed
    assert not any("after" in m for m in messages)


async def test_remove_error(fresh_async_browser, test_server):
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server)
    errors = []
    vibe.on_error(lambda err: errors.append(str(err)))
    vibe.remove_all_listeners("error")
    await vibe.eval('setTimeout(() => { throw new Error("removed") }, 0)')
    await vibe.wait(300)
    assert not any("removed" in e for e in errors)


# --- Collect Mode ---

async def test_collect_console_log(fresh_async_browser, test_server):
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server)
    vibe.on_console("collect")
    await vibe.eval('console.log("collect hello")')
    await vibe.wait(300)
    messages = vibe.console_messages()
    assert any(m["text"] == "collect hello" and m["type"] == "log" for m in messages)


async def test_collect_console_warn(fresh_async_browser, test_server):
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server)
    vibe.on_console("collect")
    await vibe.eval('console.warn("collect warning")')
    await vibe.wait(300)
    messages = vibe.console_messages()
    assert any(m["type"] == "warn" and "collect warning" in m["text"] for m in messages)


async def test_collect_console_clears_buffer(fresh_async_browser, test_server):
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server)
    vibe.on_console("collect")
    await vibe.eval('console.log("first")')
    await vibe.wait(300)
    first = vibe.console_messages()
    assert len(first) >= 1
    second = vibe.console_messages()
    assert len(second) == 0, "Buffer should be empty after retrieval"


async def test_collect_console_returns_empty_when_not_collecting(fresh_async_browser, test_server):
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server)
    messages = vibe.console_messages()
    assert messages == []


async def test_collect_error(fresh_async_browser, test_server):
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server)
    vibe.on_error("collect")
    await vibe.eval('setTimeout(() => { throw new Error("collect boom") }, 0)')
    await vibe.wait(500)
    errs = vibe.errors()
    assert any("collect boom" in e["message"] for e in errs)


async def test_collect_error_clears_buffer(fresh_async_browser, test_server):
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server)
    vibe.on_error("collect")
    await vibe.eval('setTimeout(() => { throw new Error("err1") }, 0)')
    await vibe.wait(500)
    first = vibe.errors()
    assert len(first) >= 1
    second = vibe.errors()
    assert len(second) == 0, "Buffer should be empty after retrieval"


async def test_collect_error_returns_empty_when_not_collecting(fresh_async_browser, test_server):
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server)
    errs = vibe.errors()
    assert errs == []


async def test_remove_all_listeners_stops_console_collect(fresh_async_browser, test_server):
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server)
    vibe.on_console("collect")
    await vibe.eval('console.log("before remove")')
    await vibe.wait(300)
    msgs = vibe.console_messages()
    assert len(msgs) >= 1
    vibe.remove_all_listeners("console")
    await vibe.eval('console.log("after remove")')
    await vibe.wait(300)
    assert vibe.console_messages() == [], "Should return [] after removeAllListeners"


async def test_remove_all_listeners_stops_error_collect(fresh_async_browser, test_server):
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server)
    vibe.on_error("collect")
    vibe.remove_all_listeners("error")
    await vibe.eval('setTimeout(() => { throw new Error("should not collect") }, 0)')
    await vibe.wait(500)
    assert vibe.errors() == [], "Should return [] after removeAllListeners"
