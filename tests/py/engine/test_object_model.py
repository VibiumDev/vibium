"""Object model tests — verify Browser/Page/Context isinstance and API shape (8 tests)."""

import pytest

pytestmark = pytest.mark.capability("core")

from vibium import browser, Browser, Page, BrowserContext


@pytest.fixture(scope="module")
def bro():
    b = browser.start(headless=True)
    yield b
    b.stop()


def test_launch_returns_browser(bro):
    assert isinstance(bro, Browser)


def test_page_returns_page(bro):
    vibe = bro.page()
    assert isinstance(vibe, Page)


def test_new_page(bro):
    vibe = bro.new_page()
    assert isinstance(vibe, Page)
    assert vibe.id


def test_pages_returns_all(bro):
    pages = bro.pages()
    assert isinstance(pages, list)
    assert len(pages) >= 2
    for p in pages:
        assert isinstance(p, Page)


def test_new_context(bro):
    ctx = bro.new_context()
    assert isinstance(ctx, BrowserContext)
    assert ctx.id
    ctx.close()


def test_go_url_roundtrip(bro, test_server):
    vibe = bro.page()
    vibe.go(test_server)
    url = vibe.url()
    assert test_server in url


def test_title(bro, test_server):
    vibe = bro.page()
    vibe.go(test_server)
    assert vibe.title() == "Test App"


def test_page_close(bro, test_server):
    vibe = bro.new_page()
    vibe.go(test_server)
    vibe.close()
    # After close, remaining pages should not include the closed one
    pages = bro.pages()
    assert all(p.id != vibe.id for p in pages)
