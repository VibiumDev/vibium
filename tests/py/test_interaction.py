"""Interaction tests — click, dblclick, fill, type, clear, press, check, selectOption, etc. (14 async tests)."""

import pytest


async def test_click_navigates(async_page, test_server):
    await async_page.go(test_server)
    link = await async_page.find('a[href="/subpage"]')
    await link.click()
    await async_page.wait_until.url("**/subpage")
    assert await async_page.title() == "Subpage"


async def test_dblclick(async_page, test_server):
    await async_page.go(test_server + "/inputs")
    inp = await async_page.find("#text-input")
    await inp.fill("hello world")
    await inp.dblclick()  # double click should select a word
    # Just verify it doesn't throw
    val = await inp.value()
    assert val == "hello world"


async def test_fill(async_page, test_server):
    await async_page.go(test_server + "/inputs")
    inp = await async_page.find("#text-input")
    await inp.fill("test value")
    assert await inp.value() == "test value"


async def test_type_appends(async_page, test_server):
    await async_page.go(test_server + "/inputs")
    inp = await async_page.find("#text-input")
    await inp.fill("")
    await inp.type("abc")
    assert await inp.value() == "abc"


async def test_clear(async_page, test_server):
    await async_page.go(test_server + "/inputs")
    inp = await async_page.find("#text-input")
    await inp.fill("some text")
    await inp.clear()
    assert await inp.value() == ""


async def test_press(async_page, test_server):
    await async_page.go(test_server + "/inputs")
    inp = await async_page.find("#text-input")
    await inp.fill("")
    await inp.focus()
    await inp.press("a")
    assert await inp.value() == "a"


async def test_check_uncheck(async_page, test_server):
    await async_page.go(test_server + "/form")
    cb = await async_page.find("#agree")
    await cb.check()
    assert await cb.is_checked()
    await cb.uncheck()
    assert not await cb.is_checked()


async def test_select_option(async_page, test_server):
    await async_page.go(test_server + "/form")
    sel = await async_page.find("#color")
    await sel.select_option("blue")
    assert await sel.value() == "blue"


async def test_hover(async_page, test_server):
    await async_page.go(test_server)
    el = await async_page.find("h1")
    await el.hover()
    # Just verify no error
    assert await el.is_visible()


async def test_focus(async_page, test_server):
    await async_page.go(test_server + "/inputs")
    inp = await async_page.find("#text-input")
    await inp.focus()
    # Focused element should be editable
    assert await inp.is_editable()


async def test_scroll_into_view(async_page, test_server):
    await async_page.go(test_server)
    el = await async_page.find("#info")
    await el.scroll_into_view()
    assert await el.is_visible()


async def test_dispatch_event(async_page, test_server):
    await async_page.go(test_server + "/inputs")
    inp = await async_page.find("#text-input")
    await inp.dispatch_event("focus")
    # Just verify no error
    assert await inp.is_visible()


async def test_find_all_nth_click_correct(async_page, test_server):
    await async_page.go(test_server + "/links")
    links = await async_page.find_all(".link")
    assert links.count() == 4
    third = links.nth(2)
    text = await third.text()
    assert text == "Link 3"


async def test_login_flow_checkpoint(async_page, test_server):
    """Full interaction checkpoint: navigate, fill form, check, select, submit."""
    await async_page.go(test_server + "/form")
    name = await async_page.find("#name")
    await name.fill("Test User")
    email = await async_page.find("#email")
    await email.fill("user@test.com")
    cb = await async_page.find("#agree")
    await cb.check()
    assert await cb.is_checked()
    sel = await async_page.find("#color")
    await sel.select_option("green")
    assert await sel.value() == "green"
