"""Shared pytest fixtures for the Vibium Python test suite."""

import asyncio
import pytest
import pytest_asyncio

from test_server import start_test_server


# ---------------------------------------------------------------------------
# Session-scoped: one test server for the whole test run
# ---------------------------------------------------------------------------

@pytest.fixture(scope="session")
def test_server():
    """Start the local HTTP test server. Returns base URL string."""
    server, base_url = start_test_server()
    yield base_url
    server.shutdown()


# ---------------------------------------------------------------------------
# Module-scoped: shared browser (one per test file)
# Uses loop_scope="module" so the async browser stays on the same event loop
# as all tests in the module.
# ---------------------------------------------------------------------------

@pytest.fixture(scope="module")
def sync_browser():
    """Launch a shared headless sync browser for a test module."""
    from vibium import browser
    bro = browser.launch(headless=True)
    yield bro
    bro.close()


@pytest_asyncio.fixture(scope="module", loop_scope="module")
async def async_browser():
    """Launch a shared headless async browser for a test module."""
    from vibium.async_api import browser
    bro = await browser.launch(headless=True)
    yield bro
    await bro.close()


# ---------------------------------------------------------------------------
# Function-scoped: fresh page per test (reuses module browser)
# async_page uses loop_scope="module" to share the browser's event loop.
# ---------------------------------------------------------------------------

@pytest.fixture
def sync_page(sync_browser):
    """Get a fresh page from the shared sync browser."""
    return sync_browser.page()


@pytest_asyncio.fixture(loop_scope="module")
async def async_page(async_browser):
    """Get a fresh page from the shared async browser."""
    return await async_browser.page()


# ---------------------------------------------------------------------------
# Function-scoped: fresh browser for lifecycle/process tests
# ---------------------------------------------------------------------------

@pytest.fixture
def fresh_sync_browser():
    """Launch a fresh headless sync browser for a single test."""
    from vibium import browser
    bro = browser.launch(headless=True)
    yield bro
    bro.close()


@pytest_asyncio.fixture(scope="module", loop_scope="module")
async def fresh_async_browser():
    """Launch a shared headless async browser for test modules needing isolation.

    Module-scoped to avoid port conflicts from launching too many processes.
    Each test should create its own page via ``await fresh_async_browser.page()``.
    """
    from vibium.async_api import browser
    bro = await browser.launch(headless=True)
    yield bro
    await bro.close()


# ---------------------------------------------------------------------------
# Module-scoped: WebSocket echo server (for test_websocket.py)
# ---------------------------------------------------------------------------

@pytest_asyncio.fixture(scope="module", loop_scope="module")
async def ws_echo_server():
    """Start a simple WebSocket echo server. Returns ws:// URL."""
    import websockets

    async def echo(websocket):
        async for message in websocket:
            await websocket.send(message)

    server = await websockets.serve(echo, "127.0.0.1", 0)
    port = server.sockets[0].getsockname()[1]
    yield f"ws://127.0.0.1:{port}"
    server.close()
    await server.wait_closed()
