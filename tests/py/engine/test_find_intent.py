"""Find by intent: a string is a description, css= is explicit, image= matches
locally, and the element find returns is addressed by what find resolved. No
AI provider is configured here."""

import os
import urllib.request

import pytest

from vibium import errors

from test_server import make_pattern_png

ABSENT_PNG = make_pattern_png(4242)

pytestmark = pytest.mark.capability("core")


@pytest.fixture(autouse=True)
def no_ai(monkeypatch):
    for key in list(os.environ):
        if key.startswith("VIBIUM_AI_") or key.endswith("_API_KEY"):
            monkeypatch.delenv(key, raising=False)
    monkeypatch.setenv("VIBIUM_LOAD_AI_ENV", "0")


@pytest.fixture(scope="module")
def pattern(test_server):
    with urllib.request.urlopen(test_server + "/fixtures/pattern.png") as r:
        return r.read()


async def test_description_resolves_in_the_tree(async_page, test_server):
    await async_page.go(test_server + "/find-images")
    el = await async_page.find("the Add to Cart button")
    assert el.info.tag == "button"
    assert el.info.source == "tree"
    assert el.info.ref.startswith("@e")
    assert el.info.selector
    await el.click()
    assert await async_page.evaluate("window.__added") == 1


async def test_css_looking_description_names_the_css_form(async_page, test_server):
    await async_page.go(test_server + "/find-images")
    with pytest.raises(errors.VibiumError, match=r'reads like a CSS selector.*find\(css="h1"\)'):
        await async_page.find("h1")
    with pytest.raises(errors.VibiumError, match="none is configured"):
        await async_page.find(description="h1", timeout=300)


async def test_explicit_locators(async_page, test_server):
    await async_page.go(test_server + "/find-images")
    assert await (await async_page.find(css="h1")).text() == "Gallery"
    assert (await async_page.find(tag="h1")).info.tag == "h1"
    assert len(await async_page.find_all(css="img")) == 4
    assert len(await async_page.find_all("Product thumbnail")) == 3
    body = await async_page.find(css="body")
    assert (await body.find("Add to Cart")).info.tag == "button"


async def test_strategy_validation(async_page, test_server):
    await async_page.go(test_server + "/find-images")
    with pytest.raises(errors.VibiumError, match="strategy cv needs an image"):
        await async_page.find(css="h1", strategy="cv")
    with pytest.raises(errors.VibiumError, match="never by a model"):
        await async_page.find(role="button", strategy="vlm")
    with pytest.raises(errors.VibiumError, match="configured vision model"):
        await async_page.find("the red button", strategy="vlm")


async def test_image_finds_every_copy(async_page, test_server, pattern):
    await async_page.go(test_server + "/find-images")
    els = await async_page.find_all(image=pattern)
    assert len(els) == 3
    assert all(e.info.source == "image" and e.info.tag == "img" for e in els)
    one = await async_page.find(image=pattern, details=True, strategy="cv")
    assert one.info.details["method"] == "sift"
    await one.click()
    with pytest.raises(errors.VibiumError, match="configured vision model"):
        await async_page.find(image=pattern, strategy="vlm")


async def test_image_without_match_is_empty_not_a_config_error(async_page, test_server):
    await async_page.go(test_server + "/find-images")
    with urllib.request.urlopen(test_server + "/fixtures/decoy.png") as r:
        decoy = r.read()
    assert await async_page.find_all(image=decoy) != []
    assert await async_page.find_all(image=ABSENT_PNG, timeout=300) == []
    with pytest.raises(errors.VibiumError) as exc:
        await async_page.find(image=ABSENT_PNG, timeout=300)
    assert "no verified match" in str(exc.value)
    assert "VIBIUM_AI_PROVIDER" not in str(exc.value)


async def test_visual_ref_is_driven_by_coordinates(async_page, test_server, pattern):
    await async_page.go(test_server + "/find-canvas")
    await async_page.wait_for_function("window.__drawn === true")
    el = await async_page.find(image=pattern)
    assert el.info.selector is None
    assert el.info.ref.startswith("@e")
    await el.click()
    clicks = await async_page.evaluate("window.__clicks")
    assert len(clicks) == 1
    assert abs(clicks[0]["x"] - 348) <= 2 and abs(clicks[0]["y"] - 198) <= 2
    with pytest.raises(errors.VibiumError, match="visual match with no DOM element"):
        await el.fill("x")
    await async_page.evaluate("window.scrollTo(0, 40)")
    await el.click()
    assert len(await async_page.evaluate("window.__clicks")) == 2
    await async_page.go(test_server + "/example")
    with pytest.raises(errors.VibiumError, match="stale: the page navigated"):
        await el.click()


def test_sync_find_forms(sync_page, test_server, pattern):
    sync_page.go(test_server + "/find-images")
    assert sync_page.find("Add to Cart").info.tag == "button"
    assert sync_page.find(css="h1").text() == "Gallery"
    assert len(sync_page.find_all(image=pattern)) == 3
    with pytest.raises(errors.VibiumError, match="strategy cv needs an image"):
        sync_page.find(css="h1", strategy="cv")
