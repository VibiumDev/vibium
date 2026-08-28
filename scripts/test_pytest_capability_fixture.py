"""Assert the pytest capability adapter's selection and skip counts."""

import os
import re
import subprocess
import sys
from pathlib import Path

root = Path(__file__).resolve().parents[1]
env = dict(os.environ, VIBIUM_ENGINE="chrome")

result = subprocess.run(
    [
        sys.executable, "-m", "pytest",
        str(root / "tests" / "capability-fixtures" / "py"),
        "--collect-only", "-q", "--capability-audit",
    ],
    capture_output=True,
    text=True,
    env=env,
    cwd=root,
)
output = result.stdout + result.stderr
assert result.returncode == 0, output
assert re.search(r"engine=chrome collected=3 selected=1 skipped=2", output), output
assert re.search(r"skip:audio=2", output), output
print("pytest capability fixture counts ok")
