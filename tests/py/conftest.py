"""Shared fixtures and capability selection for the Python test suite."""

import asyncio
import json
import os
from collections import Counter
from pathlib import Path
import pytest
import pytest_asyncio

from test_server import start_test_server


_CAPABILITIES_FILE = Path(__file__).parents[1] / "capabilities.json"
_CROSS_ENGINE_ROOT = Path(__file__).parent / "engine"


def _engine():
    engine = os.environ.get("VIBIUM_ENGINE") or "chrome"
    if engine not in {"chrome", "firefox"}:
        raise pytest.UsageError(f"unknown VIBIUM_ENGINE: {engine}")
    return engine


def _manifest():
    data = json.loads(_CAPABILITIES_FILE.read_text())
    if not isinstance(data, dict):
        raise pytest.UsageError("tests/capabilities.json must be an object")
    return data


def _requirements(item):
    requirements = []
    for marker in item.iter_markers("capability"):
        for name in marker.args:
            if not isinstance(name, str):
                raise pytest.UsageError(f"{item.nodeid}: capability names must be strings")
            if name not in requirements:
                requirements.append(name)
    return requirements


def pytest_collection_modifyitems(config, items):
    manifest = _manifest()
    engine = _engine()
    counts = Counter(collected=len(items))

    for item in items:
        requirements = _requirements(item)
        in_root = item.path.is_relative_to(_CROSS_ENGINE_ROOT)
        if in_root and not requirements:
            raise pytest.UsageError(f"{item.nodeid}: unmarked test in Python cross-engine root")

        unknown = [name for name in requirements if name not in manifest]
        if unknown:
            raise pytest.UsageError(f"{item.nodeid}: unknown capabilities: {', '.join(unknown)}")

        missing = [name for name in requirements if engine not in manifest[name]]
        if missing:
            counts["skipped"] += 1
            for name in missing:
                counts[f"skip:{name}"] += 1
            reason = f"{engine} lacks capabilities: {', '.join(missing)}"
            item.add_marker(pytest.mark.skip(reason=reason))
        else:
            counts["selected"] += 1

        # The manifest must not list an engine for a capability unless chrome
        # is also listed; empty entries are fine. Add an exemption mechanism
        # before introducing one.
        if config.getoption("capability_audit") and engine == "chrome":
            invalid = [name for name in missing if manifest[name]]
            if invalid:
                raise pytest.UsageError(
                    f"{item.nodeid}: Chrome audit rejected skips for: {', '.join(invalid)}"
                )

    config._vibium_capability_counts = counts


def pytest_addoption(parser):
    parser.addoption(
        "--capability-audit",
        action="store_true",
        help="fail if Chrome skips a capability supported by any engine",
    )


def pytest_terminal_summary(terminalreporter, exitstatus, config):
    counts = getattr(config, "_vibium_capability_counts", Counter())
    terminalreporter.write_sep(
        "-",
        "capabilities: "
        f"engine={_engine()} collected={counts['collected']} "
        f"selected={counts['selected']} skipped={counts['skipped']}",
    )
    for key in sorted(k for k in counts if k.startswith("skip:")):
        terminalreporter.write_line(f"capabilities: {key}={counts[key]}")


def pytest_sessionfinish(session, exitstatus):
    """Send one deterministic collection summary back from xdist workers."""
    if hasattr(session.config, "workeroutput"):
        counts = getattr(session.config, "_vibium_capability_counts", Counter())
        session.config.workeroutput["vibium_capability_counts"] = dict(counts)


def pytest_testnodedown(node, error):
    """All xdist workers collect the same items; retain (do not sum) one copy."""
    raw = node.workeroutput.get("vibium_capability_counts")
    if raw is not None and not hasattr(node.config, "_vibium_capability_counts"):
        node.config._vibium_capability_counts = Counter(raw)


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
    bro = browser.start(headless=True)
    # Firefox keeps the startup tab in the parent process until a
    # navigation, where script-backed commands are refused.
    bro.page().go("about:blank")
    yield bro
    bro.stop()


@pytest_asyncio.fixture(scope="module", loop_scope="module")
async def async_browser():
    """Launch a shared headless async browser for a test module."""
    from vibium.async_api import browser
    bro = await browser.start(headless=True)
    # Firefox keeps the startup tab in the parent process until a
    # navigation, where script-backed commands are refused.
    page = await bro.page()
    await page.go("about:blank")
    yield bro
    await bro.stop()


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
    bro = browser.start(headless=True)
    # Firefox keeps the startup tab in the parent process until a
    # navigation, where script-backed commands are refused.
    bro.page().go("about:blank")
    yield bro
    bro.stop()


@pytest_asyncio.fixture(scope="module", loop_scope="module")
async def fresh_async_browser():
    """Launch a shared headless async browser for test modules needing isolation.

    Module-scoped to avoid port conflicts from launching too many processes.
    Each test should create its own page via ``await fresh_async_browser.page()``.
    """
    from vibium.async_api import browser
    bro = await browser.start(headless=True)
    # Firefox keeps the startup tab in the parent process until a
    # navigation, where script-backed commands are refused.
    page = await bro.page()
    await page.go("about:blank")
    yield bro
    await bro.stop()


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
