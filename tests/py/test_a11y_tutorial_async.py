"""Tests extracted from docs/tutorials/a11y-tree-python.md (async)."""

import pytest
from helpers.tutorial_runner import get_tutorial_tests, run_async_block

TESTS = get_tutorial_tests("docs/tutorials/a11y-tree-python.md", mode="async")


@pytest.mark.parametrize("name,helpers,code", TESTS, ids=[t[0] for t in TESTS])
async def test_tutorial(name, helpers, code, async_page):
    await run_async_block(helpers + "\n" + code, async_page)
