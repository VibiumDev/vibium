"""Dump the Java client's public surface as JSON for the apidrift checker.

Reads the compiled classes with javap, so what gets compared is what the
class files actually expose. Requires a JDK and a prior gradle compileJava:

    python scripts/apidrift_java.py | apidrift check -surface java -spec docs/reference/api.md -actual -
"""

import json
import pathlib
import re
import subprocess
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent
CLASSES = ROOT / "clients" / "java" / "build" / "classes" / "java" / "main"

# Receiver names api.md uses in its Java column, to the classes behind them.
RECEIVERS = {
    "Vibium": "com.vibium.Vibium",
    "browser": "com.vibium.Browser",
    "page": "com.vibium.Page",
    "page.capture": "com.vibium.Capture",
    "el": "com.vibium.Element",
    "context": "com.vibium.BrowserContext",
    "keyboard": "com.vibium.Keyboard",
    "mouse": "com.vibium.Mouse",
    "touch": "com.vibium.Touch",
    "clock": "com.vibium.Clock",
    "route": "com.vibium.Route",
    "dialog": "com.vibium.Dialog",
    "download": "com.vibium.Download",
    "request": "com.vibium.Request",
    "response": "com.vibium.Response",
    "message": "com.vibium.ConsoleMessage",
    "socket": "com.vibium.WebSocketInfo",
    "recording": "com.vibium.Recording",
}

# javap member lines look like "  public void go(java.lang.String);" or
# "  public byte[] pdf();". Constructors carry the class name and no return
# type, so requiring a space-separated word before the member name skips them.
MEMBER = re.compile(r"^\s+public\s+(?:static\s+|final\s+)*\S+\s+(\w+)\(")


def members(class_name):
    out = subprocess.run(
        ["javap", "-public", "-classpath", str(CLASSES), class_name],
        capture_output=True,
        text=True,
    )
    if out.returncode != 0:
        sys.stderr.write(f"javap {class_name}: {out.stderr}")
        sys.exit(1)
    names = set()
    for line in out.stdout.splitlines():
        # Generic return types contain spaces (Map<String, String>), which
        # would split the type across the regex's word boundaries.
        while re.search(r"<[^<>]*>", line):
            line = re.sub(r"<[^<>]*>", "", line)
        m = MEMBER.match(line)
        if m and m.group(1) not in ("toString", "equals", "hashCode"):
            names.add(m.group(1))
    return sorted(names)


def main():
    if not CLASSES.is_dir():
        sys.stderr.write(f"{CLASSES} missing: run gradle compileJava first\n")
        sys.exit(1)
    surface = {recv: members(cls) for recv, cls in RECEIVERS.items()}
    json.dump(surface, sys.stdout, indent=1, sort_keys=True)


if __name__ == "__main__":
    main()
