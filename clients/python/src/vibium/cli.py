"""Command-line interface for Vibium."""

import subprocess
import sys

from .binary import find_vibium_bin


def main():
    """Main CLI entry point."""
    if len(sys.argv) < 2:
        print("Usage: vibium <command>")
        print("")
        print("Commands:")
        print("  install    Download Chrome for Testing")
        print("  version    Show version")
        sys.exit(1)

    command = sys.argv[1]

    if command == "install":
        install_browser()
    elif command == "version":
        from . import __version__
        print(f"vibium {__version__}")
    else:
        print(f"Unknown command: {command}")
        print("Run 'vibium' for usage.")
        sys.exit(1)


def install_browser():
    """Download Chrome for Testing."""
    try:
        clicker = find_vibium_bin()
    except Exception as e:
        print(f"Error: {e}")
        sys.exit(1)

    print("Installing Chrome for Testing...")
    try:
        subprocess.run([clicker, "install"], check=True)
        print("Done.")
    except subprocess.CalledProcessError:
        print("Installation failed.")
        sys.exit(1)
