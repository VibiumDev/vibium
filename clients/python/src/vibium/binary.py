"""Vibium binary management - finding, spawning, and stopping."""

import asyncio
import atexit
import importlib.util
import os
import platform
import shutil
import subprocess
import sys
from pathlib import Path
from typing import Optional

from .errors import VibiumNotFoundError, BrowserCrashedError

# Max size of a single newline-delimited stdout message the client will read.
# asyncio's StreamReader defaults to 64 KiB, which a base64 screenshot can blow
# past, raising LimitOverrunError and killing the receiver loop (issue #110).
_STREAM_LIMIT = 256 * 1024 * 1024  # 256 MiB


def get_platform_package_name() -> str:
    """Get the platform-specific package name."""
    system = sys.platform
    machine = platform.machine().lower()

    # Normalize platform
    if system == "darwin":
        plat = "darwin"
    elif system == "win32":
        plat = "win32"
    else:
        plat = "linux"

    # Normalize architecture
    if machine in ("x86_64", "amd64"):
        arch = "x64"
    elif machine in ("arm64", "aarch64"):
        arch = "arm64"
    else:
        arch = "x64"  # Default fallback

    return f"vibium_{plat}_{arch}"


def get_cache_dir() -> Path:
    """Get the platform-specific cache directory."""
    if sys.platform == "darwin":
        return Path.home() / "Library" / "Caches" / "vibium"
    elif sys.platform == "win32":
        local_app_data = os.environ.get("LOCALAPPDATA", Path.home() / "AppData" / "Local")
        return Path(local_app_data) / "vibium"
    else:
        xdg_cache = os.environ.get("XDG_CACHE_HOME", Path.home() / ".cache")
        return Path(xdg_cache) / "vibium"


def _is_python_script(path: str) -> bool:
    """Check if a file is a Python wrapper script (has a #!...python shebang)."""
    try:
        with open(path, "rb") as f:
            first_line = f.readline(128)
            return first_line.startswith(b"#!") and b"python" in first_line
    except (OSError, IOError):
        return False


def find_vibium_bin() -> str:
    """Find the vibium binary.

    Search order:
    1. VIBIUM_BIN_PATH environment variable
    2. Platform-specific package (vibium_darwin_arm64, etc.)
    3. PATH (via shutil.which)
    4. Platform cache directory

    Returns:
        Path to the vibium binary.

    Raises:
        VibiumNotFoundError: If the binary cannot be found.
    """
    binary_name = "vibium.exe" if sys.platform == "win32" else "vibium"

    # 1. Check environment variable
    env_path = os.environ.get("VIBIUM_BIN_PATH")
    if env_path and os.path.isfile(env_path):
        return env_path

    # 2. Check platform package
    package_name = get_platform_package_name()
    try:
        spec = importlib.util.find_spec(package_name)
        if spec and spec.origin:
            package_dir = Path(spec.origin).parent
            binary_path = package_dir / "bin" / binary_name
            if binary_path.is_file():
                return str(binary_path)
    except (ImportError, ModuleNotFoundError):
        pass

    # 3. Check PATH (skip Python wrapper scripts to avoid infinite recursion)
    path_binary = shutil.which(binary_name)
    if path_binary and not _is_python_script(path_binary):
        return path_binary

    # 4. Check cache directory
    cache_dir = get_cache_dir()
    cache_binary = cache_dir / binary_name
    if cache_binary.is_file():
        return str(cache_binary)

    raise VibiumNotFoundError(
        f"Could not find vibium binary. "
        f"Install the platform package: pip install {package_name}"
    )


def ensure_browser_installed(vibium_path: str) -> None:
    """Ensure Chrome for Testing is installed.

    Runs 'vibium install' if Chrome is not found.
    """
    # Check if Chrome and chromedriver are installed
    try:
        result = subprocess.run(
            [vibium_path, "is-installed"],
            capture_output=True,
            timeout=10,
        )
        if result.returncode == 0:
            return  # Already installed

    except (subprocess.TimeoutExpired, subprocess.SubprocessError):
        pass

    # Chrome not found, run install
    print("Downloading Chrome for Testing...", flush=True)
    try:
        subprocess.run(
            [vibium_path, "install"],
            check=True,
            timeout=300,  # 5 minute timeout for download
        )
        print("Chrome installed successfully.", flush=True)
    except subprocess.CalledProcessError as e:
        raise RuntimeError(f"Failed to install Chrome: {e}")
    except subprocess.TimeoutExpired:
        raise RuntimeError("Chrome installation timed out")


class VibiumProcess:
    """Manages a vibium subprocess communicating via stdin/stdout pipes."""

    def __init__(self, process: asyncio.subprocess.Process):
        self._process = process
        atexit.register(self._cleanup)

    @classmethod
    async def start(
        cls,
        headless: bool = False,
        executable_path: Optional[str] = None,
        connect_url: Optional[str] = None,
        connect_headers: Optional[dict] = None,
    ) -> "VibiumProcess":
        """Start a vibium pipe process.

        Args:
            headless: Run browser in headless mode.
            executable_path: Path to vibium binary (default: auto-detect).
            connect_url: Remote BiDi WebSocket URL to connect to instead of launching a local browser.
            connect_headers: HTTP headers for the WebSocket connection (e.g. auth tokens).

        Returns:
            A VibiumProcess instance with stdin/stdout streams ready.
        """
        binary = executable_path or find_vibium_bin()

        # Ensure Chrome is installed (auto-download if needed) — skip for remote connections
        if not connect_url:
            ensure_browser_installed(binary)

        args = [binary, "pipe"]
        if headless:
            args.append("--headless")
        if connect_url:
            args.extend(["--connect", connect_url])
        if connect_headers:
            for key, value in connect_headers.items():
                args.extend(["--connect-header", f"{key}: {value}"])

        # Read lines from stdout until we get the vibium:lifecycle.ready signal.
        # Startup is slow (~16s cold) and slower still when many browsers launch
        # at once (test suites, CI), where a cold launch can blow the ready
        # timeout or crash under resource pressure. Retry a timed-out or crashed
        # launch a couple of times with a short backoff so a single unlucky
        # launch doesn't fail hard. (Setup errors like a missing binary raise
        # before this point.)
        import json
        max_attempts = 2
        for attempt in range(1, max_attempts + 1):
            process = await asyncio.create_subprocess_exec(
                *args,
                stdin=asyncio.subprocess.PIPE,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
                start_new_session=(sys.platform != "win32"),
                # Raise the StreamReader buffer well above asyncio's 64 KiB default:
                # a single newline-delimited message (e.g. a base64 screenshot) can be
                # several MB, and readline() raises LimitOverrunError past the limit,
                # killing the receiver loop (issue #110).
                limit=_STREAM_LIMIT,
            )

            # Events (e.g. browsingContext.contextCreated) may arrive first.
            pre_ready_lines = []
            try:
                while True:
                    line_bytes = await asyncio.wait_for(
                        process.stdout.readline(),  # type: ignore[union-attr]
                        timeout=30,
                    )
                    if not line_bytes:
                        # EOF — process died
                        stderr_bytes = await process.stderr.read() if process.stderr else b""  # type: ignore[union-attr]
                        raise BrowserCrashedError(f"Vibium failed to start: {stderr_bytes.decode(errors='replace')}")
                    line = line_bytes.decode().strip()
                    if not line:
                        continue
                    try:
                        msg = json.loads(line)
                    except (json.JSONDecodeError, ValueError):
                        continue
                    if msg.get("method") == "vibium:lifecycle.ready":
                        break
                    # Buffer pre-ready events for later replay
                    pre_ready_lines.append(line)
            except (asyncio.TimeoutError, BrowserCrashedError) as err:
                try:
                    process.kill()
                except ProcessLookupError:
                    pass
                if attempt < max_attempts:
                    await asyncio.sleep(0.5)
                    continue
                if isinstance(err, asyncio.TimeoutError):
                    raise BrowserCrashedError(
                        "Vibium failed to start: timed out waiting for ready signal"
                    )
                raise

            instance = cls(process)
            instance._pre_ready_lines = pre_ready_lines
            return instance

        # Unreachable: the final attempt either returns or raises above.
        raise BrowserCrashedError("Vibium failed to start")

    def _cleanup(self) -> None:
        """Kill the subprocess if still running (called at exit)."""
        try:
            if self._process.returncode is None:
                self._process.kill()
        except ProcessLookupError:
            pass

    async def stop(self) -> None:
        """Stop the vibium process by closing stdin."""
        atexit.unregister(self._cleanup)
        try:
            if self._process.stdin:
                self._process.stdin.close()
            # Wait for graceful shutdown
            try:
                await asyncio.wait_for(self._process.wait(), timeout=5)
            except asyncio.TimeoutError:
                self._process.kill()
        except ProcessLookupError:
            pass
