"""Tests extracted from docs/tutorials/a11y-tree-python.md (sync)."""

import pytest
from helpers.tutorial_runner import get_tutorial_tests, run_sync_block

TESTS = get_tutorial_tests("docs/tutorials/a11y-tree-python.md", mode="sync")


@pytest.mark.parametrize("name,helpers,code", TESTS, ids=[t[0] for t in TESTS])
def test_tutorial(name, helpers, code, sync_page):
    run_sync_block(helpers + "\n" + code, sync_page)
