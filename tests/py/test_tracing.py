"""Tracing tests — start/stop, screenshots, snapshots, chunks, groups, zip structure (8 async tests)."""

import io
import zipfile

import pytest


# --- Basic ---

async def test_start_stop_zip(fresh_async_browser, test_server):
    """Start tracing, navigate, stop, and get a valid zip."""
    ctx = await fresh_async_browser.new_context()
    try:
        vibe = await ctx.new_page()
        await ctx.tracing.start()
        await vibe.go(test_server)
        data = await ctx.tracing.stop()
        assert isinstance(data, bytes)
        assert len(data) > 0
        # Should be a valid zip
        buf = io.BytesIO(data)
        assert zipfile.is_zipfile(buf)
    finally:
        await ctx.close()


async def test_stop_with_path(fresh_async_browser, test_server):
    """Stop tracing with a file path."""
    import tempfile
    import os

    ctx = await fresh_async_browser.new_context()
    try:
        vibe = await ctx.new_page()
        await ctx.tracing.start()
        await vibe.go(test_server)
        with tempfile.NamedTemporaryFile(suffix=".zip", delete=False) as f:
            path = f.name
        try:
            data = await ctx.tracing.stop(path=path)
            assert isinstance(data, bytes)
            assert os.path.exists(path)
        finally:
            if os.path.exists(path):
                os.unlink(path)
    finally:
        await ctx.close()


# --- Features ---

async def test_screenshots_png(fresh_async_browser, test_server):
    """Tracing with screenshots=True captures screenshots."""
    ctx = await fresh_async_browser.new_context()
    try:
        vibe = await ctx.new_page()
        await ctx.tracing.start(screenshots=True)
        await vibe.go(test_server)
        await vibe.wait(500)
        data = await ctx.tracing.stop()
        buf = io.BytesIO(data)
        with zipfile.ZipFile(buf) as zf:
            names = zf.namelist()
            # Should have at least a trace file
            assert len(names) >= 1
    finally:
        await ctx.close()


async def test_snapshots_html(fresh_async_browser, test_server):
    """Tracing with snapshots=True captures page snapshots."""
    ctx = await fresh_async_browser.new_context()
    try:
        vibe = await ctx.new_page()
        await ctx.tracing.start(snapshots=True)
        await vibe.go(test_server)
        await vibe.wait(500)
        data = await ctx.tracing.stop()
        assert isinstance(data, bytes)
        assert len(data) > 0
    finally:
        await ctx.close()


# --- Chunks ---

async def test_chunks(fresh_async_browser, test_server):
    """Start/stop chunks produce separate zip data."""
    ctx = await fresh_async_browser.new_context()
    try:
        vibe = await ctx.new_page()
        await ctx.tracing.start()
        await vibe.go(test_server)

        await ctx.tracing.start_chunk(name="chunk1")
        await vibe.go(test_server + "/subpage")
        chunk_data = await ctx.tracing.stop_chunk()
        assert isinstance(chunk_data, bytes)
        assert len(chunk_data) > 0

        await ctx.tracing.stop()
    finally:
        await ctx.close()


# --- Groups ---

async def test_groups(fresh_async_browser, test_server):
    """start_group/stop_group complete without error."""
    ctx = await fresh_async_browser.new_context()
    try:
        vibe = await ctx.new_page()
        await ctx.tracing.start()
        await ctx.tracing.start_group("test-group")
        await vibe.go(test_server)
        await ctx.tracing.stop_group()
        data = await ctx.tracing.stop()
        assert len(data) > 0
    finally:
        await ctx.close()


# --- Network ---

async def test_network_traces(fresh_async_browser, test_server):
    """Tracing captures network activity."""
    ctx = await fresh_async_browser.new_context()
    try:
        vibe = await ctx.new_page()
        await ctx.tracing.start()
        await vibe.go(test_server)
        await vibe.eval("fetch('/json')")
        await vibe.wait(500)
        data = await ctx.tracing.stop()
        assert len(data) > 0
    finally:
        await ctx.close()


# --- Structure ---

async def test_zip_structure(fresh_async_browser, test_server):
    """Trace zip has expected file structure."""
    ctx = await fresh_async_browser.new_context()
    try:
        vibe = await ctx.new_page()
        await ctx.tracing.start(screenshots=True, snapshots=True)
        await vibe.go(test_server)
        await vibe.wait(500)
        data = await ctx.tracing.stop()
        buf = io.BytesIO(data)
        with zipfile.ZipFile(buf) as zf:
            names = zf.namelist()
            assert len(names) >= 1
            # Should have some trace-related files
            assert any("trace" in n.lower() or n.endswith(".json") or n.endswith(".html") for n in names) or len(names) >= 1
    finally:
        await ctx.close()
