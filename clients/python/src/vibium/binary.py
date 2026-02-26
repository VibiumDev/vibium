"""Vibium binary management - finding, spawning, and stopping."""

import asyncio
import importlib.util
import os
import platform
import shutil
import subprocess
import sys
from pathlib import Path
from typing import Optional


class VibiumNotFoundError(Exception):
    """Raised when the vibium binary cannot be found."""
    pass


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

    # 3. Check PATH
    path_binary = shutil.which(binary_name)
    if path_binary:
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
    # Check if Chrome is installed by running 'vibium paths'
    try:
        result = subprocess.run(
            [vibium_path, "paths"],
            capture_output=True,
            text=True,
            timeout=10,
        )
        output = result.stdout

        # Check if Chrome path exists
        for line in output.split("\n"):
            if line.startswith("Chrome:"):
                chrome_path = line.split(":", 1)[1].strip()
                if os.path.isfile(chrome_path):
                    return  # Chrome is installed

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
    """Manages a vibium subprocess."""

    def __init__(self, process: subprocess.Popen, port: int):
        self._process = process
        self.port = port

    @classmethod
    async def start(
        cls,
        headless: bool = False,
        port: Optional[int] = None,
        executable_path: Optional[str] = None,
    ) -> "VibiumProcess":
        """Start a vibium process.

        Args:
            headless: Run browser in headless mode.
            port: WebSocket port (default: auto-assigned).
            executable_path: Path to vibium binary (default: auto-detect).

        Returns:
            A VibiumProcess instance.
        """
        binary = executable_path or find_vibium_bin()

        # Ensure Chrome is installed (auto-download if needed)
        ensure_browser_installed(binary)

        args = [binary, "serve"]
        if headless:
            args.append("--headless")
        # Use port 0 (OS-assigned random port) by default to avoid conflicts
        # when multiple browser instances run concurrently
        args.extend(["--port", str(port if port is not None else 0)])

        # Start the process
        process = subprocess.Popen(
            args,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
        )

        # Read the port from stdout
        # Vibium prints "Server listening on ws://localhost:PORT"
        actual_port = port or 0

        if process.stdout:
            # First line: "Starting Clicker proxy server on port ..."
            # Second line: "Server listening on ws://localhost:PORT"
            for _ in range(2):
                line = process.stdout.readline()
                if "listening on" in line.lower():
                    try:
                        actual_port = int(line.strip().split(":")[-1])
                    except (ValueError, IndexError):
                        pass
                    break

        # Give it a moment to start
        await asyncio.sleep(0.1)

        # Check if process is still running
        if process.poll() is not None:
            stderr = process.stderr.read() if process.stderr else ""
            raise RuntimeError(f"Vibium failed to start: {stderr}")

        return cls(process, actual_port)

    async def stop(self) -> None:
        """Stop the vibium process."""
        if self._process.poll() is None:
            self._process.terminate()
            try:
                self._process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                self._process.kill()
