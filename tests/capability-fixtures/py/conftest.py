"""Load the capability adapter for the synthetic fixture suite."""

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / "py"))

from capability_adapter import *  # noqa: E402,F401,F403
