"""Element finding tests — findAll, first/last/nth/count, scoped find, semantic, iteration (10 async tests)."""

import pytest

pytestmark = pytest.mark.capability("core")


async def test_find_all_multiple(async_page, test_server):
    await async_page.go(test_server + "/links")
    els = await async_page.find_all(css=".link")
    assert len(els) == 4


async def test_first(async_page, test_server):
    await async_page.go(test_server + "/links")
    els = await async_page.find_all(css=".link")
    first = els[0]
    text = await first.text()
    assert text == "Link 1"


async def test_last(async_page, test_server):
    await async_page.go(test_server + "/links")
    els = await async_page.find_all(css=".link")
    last = els[-1]
    text = await last.text()
    assert text == "Link 4"


async def test_nth(async_page, test_server):
    await async_page.go(test_server + "/links")
    els = await async_page.find_all(css=".link")
    second = els[1]
    text = await second.text()
    assert text == "Link 2"


async def test_count(async_page, test_server):
    await async_page.go(test_server + "/links")
    els = await async_page.find_all(css=".link")
    assert len(els) == 4


async def test_scoped_find(async_page, test_server):
    await async_page.go(test_server + "/links")
    nested = await async_page.find(css="#nested")
    span = await nested.find(css=".inner")
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
    els = await async_page.find_all(css=".link")
    texts = []
    for el in els:
        t = await el.text()
        texts.append(t)
    assert len(texts) == 4
    assert texts[0] == "Link 1"
    assert texts[3] == "Link 4"


async def test_scoped_child_reresolves_inside_parent(async_page, test_server):
    """A child of a scoped find must act inside its parent (#779)."""
    await async_page.go(test_server + "/nested-scope")
    parent = await async_page.find(css="#b")
    child = await parent.find(css=".act")
    await child.click()
    out = await async_page.evaluate("document.getElementById('out').textContent")
    assert out == "B"
