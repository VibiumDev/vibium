"""Element finding tests — findAll, first/last/nth/count, scoped find, semantic, iteration (10 async tests)."""

import pytest


async def test_find_all_multiple(async_page, test_server):
    await async_page.go(test_server + "/links")
    els = await async_page.find_all(".link")
    assert els.count() == 4


async def test_first(async_page, test_server):
    await async_page.go(test_server + "/links")
    els = await async_page.find_all(".link")
    first = els.first()
    text = await first.text()
    assert text == "Link 1"


async def test_last(async_page, test_server):
    await async_page.go(test_server + "/links")
    els = await async_page.find_all(".link")
    last = els.last()
    text = await last.text()
    assert text == "Link 4"


async def test_nth(async_page, test_server):
    await async_page.go(test_server + "/links")
    els = await async_page.find_all(".link")
    second = els.nth(1)
    text = await second.text()
    assert text == "Link 2"


async def test_count(async_page, test_server):
    await async_page.go(test_server + "/links")
    els = await async_page.find_all(".link")
    assert els.count() == 4
    assert len(els) == 4


async def test_scoped_find(async_page, test_server):
    await async_page.go(test_server + "/links")
    nested = await async_page.find("#nested")
    span = await nested.find(".inner")
    text = await span.text()
    assert "Nested span" in text or "span" in text.lower()


async def test_find_by_role(async_page, test_server):
    await async_page.go(test_server)
    el = await async_page.find(role="heading")
    assert el.info.tag == "h1"
    assert "Welcome" in el.info.text


async def test_find_by_text(async_page, test_server):
    await async_page.go(test_server)
    el = await async_page.find(text="Some info text")
    assert el.info.tag == "p"


async def test_find_role_text_combo(async_page, test_server):
    await async_page.go(test_server)
    el = await async_page.find(role="link", text="Go to subpage")
    assert el.info.tag == "a"
    assert "subpage" in el.info.text.lower()


async def test_iteration(async_page, test_server):
    await async_page.go(test_server + "/links")
    els = await async_page.find_all(".link")
    texts = []
    for el in els:
        t = await el.text()
        texts.append(t)
    assert len(texts) == 4
    assert texts[0] == "Link 1"
    assert texts[3] == "Link 4"
