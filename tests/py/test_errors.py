"""Error type tests — verify structured error types are importable and raised correctly."""

import builtins

import pytest


# --- Import tests (no browser needed) ---


def test_errors_importable_from_vibium():
    """All error types should be importable from the top-level vibium package."""
    import vibium

    assert hasattr(vibium, "VibiumError")
    assert hasattr(vibium, "BiDiError")
    assert hasattr(vibium, "VibiumNotFoundError")
    assert hasattr(vibium, "TimeoutError")
    assert hasattr(vibium, "ConnectionError")
    assert hasattr(vibium, "ElementNotFoundError")
    assert hasattr(vibium, "BrowserCrashedError")


def test_errors_importable_from_async_api():
    """Error types should also be importable from vibium.async_api."""
    from vibium.async_api import (
        VibiumError,
        BiDiError,
        TimeoutError,
        ConnectionError,
        ElementNotFoundError,
        BrowserCrashedError,
    )

    assert VibiumError is not None
    assert BiDiError is not None


def test_errors_importable_from_sync_api():
    """Error types should also be importable from vibium.sync_api."""
    from vibium.sync_api import (
        VibiumError,
        BiDiError,
        TimeoutError,
        ConnectionError,
        ElementNotFoundError,
        BrowserCrashedError,
    )

    assert VibiumError is not None
    assert BiDiError is not None


def test_timeout_is_builtin_subclass():
    """vibium.TimeoutError should be a subclass of builtins.TimeoutError."""
    import vibium

    assert issubclass(vibium.TimeoutError, builtins.TimeoutError)
    err = vibium.TimeoutError("test timeout")
    assert isinstance(err, builtins.TimeoutError)


def test_connection_is_builtin_subclass():
    """vibium.ConnectionError should be a subclass of builtins.ConnectionError."""
    import vibium

    assert issubclass(vibium.ConnectionError, builtins.ConnectionError)
    err = vibium.ConnectionError("ws://localhost:1234")
    assert isinstance(err, builtins.ConnectionError)


def test_all_errors_subclass_vibium_error():
    """Every custom error type should be a subclass of VibiumError."""
    import vibium

    for cls in [
        vibium.BiDiError,
        vibium.VibiumNotFoundError,
        vibium.TimeoutError,
        vibium.ConnectionError,
        vibium.ElementNotFoundError,
        vibium.BrowserCrashedError,
    ]:
        assert issubclass(cls, vibium.VibiumError), f"{cls.__name__} is not a VibiumError subclass"


def test_element_not_found_attributes():
    """ElementNotFoundError should store the selector."""
    import vibium

    err = vibium.ElementNotFoundError("#missing")
    assert err.selector == "#missing"
    assert "#missing" in str(err)


def test_bidi_error_attributes():
    """BiDiError should store error and message."""
    import vibium

    err = vibium.BiDiError("invalid argument", "bad selector")
    assert err.error == "invalid argument"
    assert err.message == "bad selector"


def test_browser_crashed_attributes():
    """BrowserCrashedError should store exit_code and output."""
    import vibium

    err = vibium.BrowserCrashedError("crashed", exit_code=1, output="segfault")
    assert err.exit_code == 1
    assert err.output == "segfault"


# --- Integration tests (browser needed) ---


async def test_element_not_found_raised(async_page, test_server):
    """Finding a non-existent element should raise ElementNotFoundError."""
    import vibium

    await async_page.go(test_server)
    with pytest.raises(vibium.ElementNotFoundError) as exc_info:
        await async_page.find("#this-element-does-not-exist-at-all", timeout=1000)
    assert exc_info.value.selector is not None


def test_element_not_found_sync(sync_page, test_server):
    """Sync API should also raise ElementNotFoundError."""
    import vibium

    sync_page.go(test_server)
    with pytest.raises(vibium.ElementNotFoundError):
        sync_page.find("#this-element-does-not-exist-at-all", timeout=1000)


async def test_timeout_error_is_catchable_as_builtin(async_page, test_server):
    """vibium.TimeoutError should be catchable with except builtins.TimeoutError."""
    import vibium

    await async_page.go(test_server)
    # Use wait_for_function with an expression that never becomes truthy to get a pure timeout
    with pytest.raises(builtins.TimeoutError):
        await async_page.wait_for_function("() => false", timeout=500)
