"""Tests extracted from docs/how-to-guides/accessibility-tree.md (async)."""

import pytest

pytestmark = pytest.mark.capability("core")
from helpers.tutorial_runner import get_tutorial_tests, run_async_block

TESTS = get_tutorial_tests("docs/how-to-guides/accessibility-tree.md", mode="async")


@pytest.mark.parametrize("name,helpers,code", TESTS, ids=[t[0] for t in TESTS])
async def test_tutorial(name, helpers, code, async_page):
    await run_async_block(helpers + "\n" + code, async_page)
