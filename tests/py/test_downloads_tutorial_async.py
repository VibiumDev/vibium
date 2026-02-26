"""Tests extracted from docs/tutorials/downloads-python.md (async)."""

import pytest
from helpers.tutorial_runner import get_tutorial_tests, start_tutorial_server, run_async_standalone

MD_PATH = "docs/tutorials/downloads-python.md"
TESTS = get_tutorial_tests(MD_PATH, mode="async")

ROUTES = {
    "/file": (
        200,
        {
            "Content-Type": "text/plain",
            "Content-Disposition": 'attachment; filename="hello.txt"',
        },
        b"hello world",
    ),
}
DEFAULT_BODY = '<a href="/file" id="dl-link">Download hello.txt</a>'


@pytest.fixture(scope="module")
def download_server():
    server, base_url = start_tutorial_server(ROUTES, DEFAULT_BODY)
    yield base_url
    server.shutdown()


@pytest.mark.parametrize("name,helpers,code", TESTS, ids=[t[0] for t in TESTS])
async def test_tutorial(name, helpers, code, download_server):
    await run_async_standalone(helpers + "\n" + code, base_url=download_server)
