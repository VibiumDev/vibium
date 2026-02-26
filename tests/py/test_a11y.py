"""Accessibility tests — el.role, el.label, a11yTree with options (16 async tests)."""

import pytest


# --- Role ---

async def test_role_link(async_page, test_server):
    await async_page.go(test_server + "/a11y")
    link = await async_page.find("a")
    role = await link.role()
    assert role == "link"


async def test_role_heading(async_page, test_server):
    await async_page.go(test_server + "/a11y")
    h1 = await async_page.find("h1")
    role = await h1.role()
    assert role == "heading"


async def test_role_explicit(async_page, test_server):
    await async_page.go(test_server + "/a11y")
    btn = await async_page.find("button[aria-label='Close dialog']")
    role = await btn.role()
    assert role == "button"


async def test_role_chain(async_page, test_server):
    await async_page.go(test_server + "/a11y")
    el = await async_page.find("input#username")
    role = await el.role()
    assert role == "textbox"


# --- Label ---

async def test_label_link(async_page, test_server):
    await async_page.go(test_server + "/a11y")
    link = await async_page.find("a")
    lbl = await link.label()
    assert "subpage" in lbl.lower()


async def test_label_aria_label(async_page, test_server):
    await async_page.go(test_server + "/a11y")
    btn = await async_page.find("button[aria-label='Close dialog']")
    lbl = await btn.label()
    assert lbl == "Close dialog"


async def test_label_aria_labelledby(async_page, test_server):
    await async_page.go(test_server + "/a11y")
    inp = await async_page.find("input[aria-labelledby='desc']")
    lbl = await inp.label()
    assert "Description" in lbl or "desc" in lbl.lower()


async def test_label_for(async_page, test_server):
    await async_page.go(test_server + "/a11y")
    inp = await async_page.find("#username")
    lbl = await inp.label()
    assert "Username" in lbl or "username" in lbl.lower()


async def test_label_chain(async_page, test_server):
    await async_page.go(test_server + "/a11y")
    nav = await async_page.find("nav")
    lbl = await nav.label()
    assert "Main navigation" in lbl


# --- Tree ---

async def test_tree_root(async_page, test_server):
    await async_page.go(test_server + "/a11y")
    tree = await async_page.a11y_tree()
    assert isinstance(tree, dict)
    assert "role" in tree


async def test_tree_roles(async_page, test_server):
    await async_page.go(test_server + "/a11y")
    tree = await async_page.a11y_tree()
    # Tree should contain the page structure
    tree_str = str(tree)
    assert "heading" in tree_str.lower() or "link" in tree_str.lower()


async def test_tree_everything_true(async_page, test_server):
    await async_page.go(test_server + "/a11y")
    tree = await async_page.a11y_tree(everything=True)
    assert isinstance(tree, dict)
    # With everything=True, should have more nodes
    tree_str = str(tree)
    assert len(tree_str) > 50


async def test_tree_everything_false(async_page, test_server):
    await async_page.go(test_server + "/a11y")
    tree_all = await async_page.a11y_tree(everything=True)
    tree_interesting = await async_page.a11y_tree(everything=False)
    # Filtered tree should be equal or smaller
    assert len(str(tree_interesting)) <= len(str(tree_all))


async def test_tree_checked(async_page, test_server):
    await async_page.go(test_server + "/a11y")
    tree = await async_page.a11y_tree(everything=True)
    tree_str = str(tree)
    assert "checked" in tree_str.lower() or "checkbox" in tree_str.lower()


async def test_tree_disabled(async_page, test_server):
    await async_page.go(test_server + "/a11y")
    tree = await async_page.a11y_tree(everything=True)
    tree_str = str(tree)
    assert "disabled" in tree_str.lower() or "Disabled" in tree_str


async def test_tree_heading_levels(async_page, test_server):
    await async_page.go(test_server + "/a11y")
    tree = await async_page.a11y_tree(everything=True)
    tree_str = str(tree)
    assert "Heading Level 1" in tree_str or "heading" in tree_str.lower()
