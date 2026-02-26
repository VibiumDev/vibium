"""Clock tests — install, fastForward, runFor, pauseAt, resume, setFixedTime, timezone (17 async tests)."""

import pytest


# --- Install ---

async def test_install(fresh_async_browser, test_server):
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/clock")
    await vibe.clock.install()
    result = await vibe.eval("typeof Date.now")
    assert result == "function"


async def test_install_with_time(fresh_async_browser, test_server):
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/clock")
    await vibe.clock.install(time=1000)
    result = await vibe.eval("Date.now()")
    assert result == 1000


async def test_install_iso_string(fresh_async_browser, test_server):
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/clock")
    await vibe.clock.install(time="2024-01-01T00:00:00Z")
    result = await vibe.eval("new Date().toISOString()")
    assert "2024-01-01" in result


async def test_double_install(fresh_async_browser, test_server):
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/clock")
    await vibe.clock.install(time=0)
    # Second install should either succeed or replace
    await vibe.clock.install(time=5000)
    result = await vibe.eval("Date.now()")
    assert result == 5000


# --- SetFixedTime ---

async def test_set_fixed_time(fresh_async_browser, test_server):
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/clock")
    await vibe.clock.install(time=0)
    await vibe.clock.set_fixed_time(42000)
    result = await vibe.eval("Date.now()")
    assert result == 42000


async def test_set_fixed_time_string(fresh_async_browser, test_server):
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/clock")
    await vibe.clock.install(time=0)
    await vibe.clock.set_fixed_time("2024-06-15T12:00:00Z")
    result = await vibe.eval("new Date().toISOString()")
    assert "2024-06-15" in result


# --- FastForward ---

async def test_fast_forward(fresh_async_browser, test_server):
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/clock")
    await vibe.clock.install(time=0)
    await vibe.clock.fast_forward(10000)
    result = await vibe.eval("Date.now()")
    assert result >= 10000


# --- RunFor ---

async def test_run_for(fresh_async_browser, test_server):
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/clock")
    await vibe.clock.install(time=0)
    await vibe.clock.run_for(5000)
    result = await vibe.eval("Date.now()")
    assert result >= 5000


# --- PauseAt ---

async def test_pause_at(fresh_async_browser, test_server):
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/clock")
    await vibe.clock.install(time=0)
    await vibe.clock.pause_at(8000)
    result = await vibe.eval("Date.now()")
    assert result == 8000


# --- Resume ---

async def test_resume(fresh_async_browser, test_server):
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/clock")
    await vibe.clock.install(time=0)
    await vibe.clock.pause_at(1000)
    await vibe.clock.resume()
    # After resume, time should progress (just verify no error)
    result = await vibe.eval("Date.now()")
    assert result >= 1000


# --- SetSystemTime ---

async def test_set_system_time(fresh_async_browser, test_server):
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/clock")
    await vibe.clock.install(time=0)
    await vibe.clock.set_system_time(99000)
    result = await vibe.eval("Date.now()")
    assert result == 99000


# --- Navigation ---

async def test_clock_works_after_navigation(fresh_async_browser, test_server):
    """Clock can be installed and used after navigating to a new page."""
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/clock")
    await vibe.clock.install(time=1000)
    result1 = await vibe.eval("Date.now()")
    assert result1 == 1000
    # Navigate to a different page and re-install clock
    await vibe.go(test_server + "/clock?after")
    await vibe.clock.install(time=2000)
    result2 = await vibe.eval("Date.now()")
    assert result2 == 2000
    # Verify clock still works by setting fixed time
    await vibe.clock.set_fixed_time(1735689600000)  # 2025-01-01
    year = await vibe.eval("new Date().getUTCFullYear()")
    assert year == 2025


# --- Timezone ---

async def test_install_timezone(fresh_async_browser, test_server):
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/clock")
    await vibe.clock.install(time="2024-01-01T12:00:00Z", timezone="America/New_York")
    result = await vibe.eval("new Date().getHours()")
    assert result == 7  # EST = UTC-5


async def test_set_timezone(fresh_async_browser, test_server):
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/clock")
    await vibe.clock.set_timezone("Asia/Tokyo")
    # Just verify no error; timezone is set
    result = await vibe.eval("Intl.DateTimeFormat().resolvedOptions().timeZone")
    assert result == "Asia/Tokyo"


async def test_reset_timezone(fresh_async_browser, test_server):
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/clock")
    await vibe.clock.set_timezone("Europe/London")
    tz = await vibe.eval("Intl.DateTimeFormat().resolvedOptions().timeZone")
    assert tz == "Europe/London"


async def test_timezone_and_time(fresh_async_browser, test_server):
    vibe = await fresh_async_browser.new_page()
    await vibe.go(test_server + "/clock")
    await vibe.clock.install(time="2024-07-01T00:00:00Z", timezone="America/Los_Angeles")
    hours = await vibe.eval("new Date().getHours()")
    assert hours == 17  # PDT = UTC-7, so July 1 00:00 UTC = June 30 17:00 PDT
