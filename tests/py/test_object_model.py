"""Object model tests — verify Browser/Page/Context isinstance and API shape (8 tests)."""

from vibium import browser, Browser, Page, BrowserContext


def test_launch_returns_browser():
    bro = browser.launch(headless=True)
    try:
        assert isinstance(bro, Browser)
    finally:
        bro.close()


def test_page_returns_page():
    bro = browser.launch(headless=True)
    try:
        vibe = bro.page()
        assert isinstance(vibe, Page)
    finally:
        bro.close()


def test_new_page():
    bro = browser.launch(headless=True)
    try:
        vibe = bro.new_page()
        assert isinstance(vibe, Page)
        assert vibe.id
    finally:
        bro.close()


def test_pages_returns_all():
    bro = browser.launch(headless=True)
    try:
        bro.new_page()
        pages = bro.pages()
        assert isinstance(pages, list)
        assert len(pages) >= 2
        for p in pages:
            assert isinstance(p, Page)
    finally:
        bro.close()


def test_new_context():
    bro = browser.launch(headless=True)
    try:
        ctx = bro.new_context()
        assert isinstance(ctx, BrowserContext)
        assert ctx.id
        ctx.close()
    finally:
        bro.close()


def test_go_url_roundtrip(test_server):
    bro = browser.launch(headless=True)
    try:
        vibe = bro.page()
        vibe.go(test_server)
        url = vibe.url()
        assert test_server in url
    finally:
        bro.close()


def test_title(test_server):
    bro = browser.launch(headless=True)
    try:
        vibe = bro.page()
        vibe.go(test_server)
        assert vibe.title() == "Test App"
    finally:
        bro.close()


def test_page_close(test_server):
    bro = browser.launch(headless=True)
    try:
        vibe = bro.new_page()
        vibe.go(test_server)
        vibe.close()
        # After close, remaining pages should not include the closed one
        pages = bro.pages()
        assert all(p.id != vibe.id for p in pages)
    finally:
        bro.close()
