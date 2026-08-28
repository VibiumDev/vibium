"""Vibium binary for Linux ARM64."""

from pathlib import Path

__version__ = "26.8.21"

def get_binary_path() -> str:
    """Get the path to the vibium binary."""
    return str(Path(__file__).parent / "bin" / "vibium")
