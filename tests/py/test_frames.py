"""Frame tests — frames, frame by URL/name, nested, mainFrame (7 async tests)."""

import pytest


async def test_frames_returns_children(async_page, test_server):
    await async_page.go(test_server + "/frames")
    frames = await async_page.frames()
    assert isinstance(frames, list)
    assert len(frames) >= 1


async def test_frame_by_url(async_page, test_server):
    await async_page.go(test_server + "/frames")
    await async_page.wait(500)  # Wait for frames to load
    frame = await async_page.frame("frame1")
    assert frame is not None


async def test_frame_by_name(async_page, test_server):
    await async_page.go(test_server + "/frames")
    await async_page.wait(500)
    frame = await async_page.frame("frame-one")
    assert frame is not None


async def test_frame_null_missing(async_page, test_server):
    await async_page.go(test_server + "/frames")
    frame = await async_page.frame("nonexistent-frame")
    assert frame is None


async def test_frame_full_page_api(async_page, test_server):
    await async_page.go(test_server + "/frames")
    await async_page.wait(500)
    frame = await async_page.frame("frame1")
    if frame:
        title = await frame.title()
        assert "Frame 1" in title


async def test_nested_frames(async_page, test_server):
    await async_page.go(test_server + "/frames")
    await async_page.wait(1000)  # Wait for nested frames
    frame1 = await async_page.frame("frame1")
    if frame1:
        inner_frames = await frame1.frames()
        assert len(inner_frames) >= 1


async def test_main_frame(async_page, test_server):
    await async_page.go(test_server + "/frames")
    main = async_page.main_frame()
    assert main.id == async_page.id
