"""Vibium binary management - finding, spawning, and stopping."""

import asyncio
import atexit
import importlib.util
import os
import platform
import shutil
import sys
from pathlib import Path
from typing import Optional

from .errors import VibiumNotFoundError, BrowserCrashedError

# Max size of a single newline-delimited stdout message the client will read.
# asyncio's StreamReader defaults to 64 KiB, which a base64 screenshot can blow
# past, raising LimitOverrunError and killing the receiver loop (issue #110).
_STREAM_LIMIT = 256 * 1024 * 1024  # 256 MiB

# Overall wall-clock budget (seconds) for one launch attempt's ready signal.
# The per-read timeout alone can be reset forever by dribbled pre-ready output,
# so a stuck launch could hang far past it; this bounds the whole attempt.
_READY_TIMEOUT = 60

# Ready budget once vibium reports it is downloading the browser (first run).
# Matches the 5-minute install budget the old client-side installer had. The
# deadline is extended once, not per read, so a hang still fails in bounded
# time.
_INSTALL_READY_TIMEOUT = 300

# Printed by `vibium pipe` on stderr right before it downloads the browser.
# Must match the installingMarker constant in the binary's pipe.go.
_INSTALLING_MARKER = b"[pipe] installing browser"

# Bytes of trailing stderr kept for error messages.
_STDERR_TAIL_LIMIT = 8192


class _StderrWatcher:
    """Drains the subprocess's stderr from spawn time.

    An unread stderr pipe blocks vibium once the OS buffer (~64 KiB) fills.
    Keeps a bounded tail for error messages, sets ``installing`` when vibium
    reports it is downloading the browser, and forwards diagnostics to our
    stderr when VIBIUM_STDERR is set.
    """

    def __init__(self, process):
        self.tail = b""
        self.installing = asyncio.Event()
        self._stderr = process.stderr
        self.task = asyncio.create_task(self._drain()) if process.stderr else None

    async def _drain(self) -> None:
        forward = bool(os.environ.get("VIBIUM_STDERR"))
        try:
            while True:
                chunk = await self._stderr.read(65536)
                if not chunk:
                    return
                if forward:
                    sys.stderr.write(chunk.decode(errors="replace"))
                    sys.stderr.flush()
                self.tail = (self.tail + chunk)[-_STDERR_TAIL_LIMIT:]
                if not self.installing.is_set() and _INSTALLING_MARKER in self.tail:
                    self.installing.set()
        except (asyncio.CancelledError, OSError):
            pass

    def tail_text(self) -> str:
        return self.tail.decode(errors="replace")


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


class VibiumProcess:
    """Manages a vibium subprocess communicating via stdin/stdout pipes."""

    def __init__(self, process: asyncio.subprocess.Process):
        self._process = process
        atexit.register(self._cleanup)

    @classmethod
    async def start(
        cls,
        headless: bool = False,
        engine: Optional[str] = None,
        channel: Optional[str] = None,
        executable_path: Optional[str] = None,
        connect_url: Optional[str] = None,
        connect_headers: Optional[dict] = None,
    ) -> "VibiumProcess":
        """Start a vibium pipe process.

        Args:
            headless: Run browser in headless mode.
            engine: Browser engine to launch: "chrome" (default) or "firefox".
            channel: Release channel of the engine to install and run, e.g.
                "beta". Currently honored by Firefox only.
            executable_path: Path to vibium binary (default: auto-detect).
            connect_url: Remote BiDi WebSocket URL to connect to instead of launching a local browser.
            connect_headers: HTTP headers for the WebSocket connection (e.g. auth tokens).

        Returns:
            A VibiumProcess instance with stdin/stdout streams ready.
        """
        binary = executable_path or find_vibium_bin()

        args = [binary, "pipe"]
        if engine:
            args.extend(["--engine", engine])
        if channel:
            args.extend(["--firefox-channel", channel])
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

            # Drain stderr from spawn time: vibium prints its install marker
            # and download progress there before the ready signal, and the
            # pipe would otherwise fill and block vibium.
            watcher = _StderrWatcher(process)

            # Events (e.g. browsingContext.contextCreated) may arrive first.
            # Bound the whole wait with a wall-clock deadline, not just per-read:
            # vibium forwards pre-ready events, so a per-read timeout can be
            # reset indefinitely by dribbled output while `ready` never arrives.
            pre_ready_lines = []
            now = asyncio.get_running_loop().time
            deadline = now() + _READY_TIMEOUT
            extended = False
            read_task = None
            try:
                while True:
                    # First run: vibium is downloading the browser, which can
                    # legitimately take minutes. Extend the deadline once —
                    # still a hard bound, not a per-read reset.
                    if not extended and watcher.installing.is_set():
                        deadline = now() + _INSTALL_READY_TIMEOUT
                        extended = True
                    remaining = deadline - now()
                    if remaining <= 0:
                        raise asyncio.TimeoutError
                    if read_task is None:
                        read_task = asyncio.ensure_future(
                            process.stdout.readline()  # type: ignore[union-attr]
                        )
                    # asyncio.wait (unlike wait_for) leaves the read task
                    # running on timeout, so no partial line is lost. The cap
                    # keeps marker-driven deadline extension prompt.
                    done, _ = await asyncio.wait({read_task}, timeout=min(remaining, 0.5))
                    if not done:
                        continue
                    line_bytes = read_task.result()
                    read_task = None
                    if not line_bytes:
                        # EOF — process died. Let the stderr drain finish so
                        # the tail holds the failure message.
                        if watcher.task:
                            try:
                                await asyncio.wait_for(asyncio.shield(watcher.task), 1.0)
                            except asyncio.TimeoutError:
                                pass
                        raise BrowserCrashedError(f"Vibium failed to start: {watcher.tail_text()}")
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
                if read_task is not None:
                    read_task.cancel()
                if watcher.task:
                    watcher.task.cancel()
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
            # The watcher keeps draining stderr for the process's lifetime.
            instance._stderr_task = watcher.task
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
