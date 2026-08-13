"""Selector strategy tests — xpath, testid, placeholder, alt, title, label."""

import pytest

pytestmark = pytest.mark.capability("core")


async def test_find_by_xpath(async_page, test_server):
    await async_page.go(test_server + "/selectors")
    el = await async_page.find(xpath="//button")
    assert el.info.tag == "button"
    assert "Submit" in el.info.text


async def test_find_by_testid(async_page, test_server):
    await async_page.go(test_server + "/selectors")
    el = await async_page.find(testid="search-input")
    assert el.info.tag == "input"


async def test_find_by_placeholder(async_page, test_server):
    await async_page.go(test_server + "/selectors")
    el = await async_page.find(placeholder="Search...")
    assert el.info.tag == "input"


async def test_find_by_alt(async_page, test_server):
    await async_page.go(test_server + "/selectors")
    el = await async_page.find(alt="Logo image")
    assert el.info.tag == "img"


async def test_find_by_title(async_page, test_server):
    await async_page.go(test_server + "/selectors")
    el = await async_page.find(title="Search field")
    assert el.info.tag == "input"


async def test_find_by_label(async_page, test_server):
    await async_page.go(test_server + "/selectors")
    el = await async_page.find(label="Username")
    assert el.info.tag == "input"


async def test_xpath_nested(async_page, test_server):
    await async_page.go(test_server + "/selectors")
    el = await async_page.find(xpath='//div[@class="container"]/span')
    assert el.info.tag == "span"
    assert "Hello from span" in el.info.text
