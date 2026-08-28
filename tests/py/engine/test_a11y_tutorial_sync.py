"""Tests extracted from docs/how-to-guides/accessibility-tree.md (sync)."""

import pytest

pytestmark = pytest.mark.capability("core")
from helpers.tutorial_runner import get_tutorial_tests, run_sync_block

TESTS = get_tutorial_tests("docs/how-to-guides/accessibility-tree.md", mode="sync")


@pytest.mark.parametrize("name,helpers,code", TESTS, ids=[t[0] for t in TESTS])
def test_tutorial(name, helpers, code, sync_page):
    run_sync_block(helpers + "\n" + code, sync_page)
