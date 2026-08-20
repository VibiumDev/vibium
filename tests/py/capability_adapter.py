"""Capability selection for pytest.

Lives outside conftest.py so the synthetic fixture suite in
tests/capability-fixtures/py can load the same hooks through its own conftest.
"""

import json
import os
from collections import Counter
from pathlib import Path
import pytest


_CAPABILITIES_FILE = Path(__file__).parents[1] / "capabilities.json"
_CROSS_ENGINE_ROOT = Path(__file__).parent / "engine"


def _engine():
    engine = os.environ.get("VIBIUM_ENGINE") or "chrome"
    if engine not in {"chrome", "firefox"}:
        raise pytest.UsageError(f"unknown VIBIUM_ENGINE: {engine}")
    return engine


def _manifest():
    data = json.loads(_CAPABILITIES_FILE.read_text())
    if not isinstance(data, dict):
        raise pytest.UsageError("tests/capabilities.json must be an object")
    return data


def _requirements(item):
    requirements = []
    for marker in item.iter_markers("capability"):
        for name in marker.args:
            if not isinstance(name, str):
                raise pytest.UsageError(f"{item.nodeid}: capability names must be strings")
            if name not in requirements:
                requirements.append(name)
    return requirements


def pytest_configure(config):
    config.addinivalue_line(
        "markers",
        "capability(*names): browser capabilities required by a test "
        "(requirements use AND semantics)",
    )


def pytest_collection_modifyitems(config, items):
    manifest = _manifest()
    engine = _engine()
    counts = Counter(collected=len(items))

    for item in items:
        requirements = _requirements(item)
        in_root = item.path.is_relative_to(_CROSS_ENGINE_ROOT)
        if in_root and not requirements:
            raise pytest.UsageError(f"{item.nodeid}: unmarked test in Python cross-engine root")

        if requirements:
            counts["marked"] += 1

        unknown = [name for name in requirements if name not in manifest]
        if unknown:
            raise pytest.UsageError(f"{item.nodeid}: unknown capabilities: {', '.join(unknown)}")

        missing = [name for name in requirements if engine not in manifest[name]]
        if missing:
            counts["skipped"] += 1
            for name in missing:
                counts[f"skip:{name}"] += 1
            reason = f"{engine} lacks capabilities: {', '.join(missing)}"
            item.add_marker(pytest.mark.skip(reason=reason))
        else:
            counts["selected"] += 1

        # The manifest must not list an engine for a capability unless chrome
        # is also listed; empty entries are fine. Add an exemption mechanism
        # before introducing one.
        if config.getoption("capability_audit") and engine == "chrome":
            invalid = [name for name in missing if manifest[name]]
            if invalid:
                raise pytest.UsageError(
                    f"{item.nodeid}: Chrome audit rejected skips for: {', '.join(invalid)}"
                )

    config._vibium_capability_counts = counts


def pytest_addoption(parser):
    parser.addoption(
        "--capability-audit",
        action="store_true",
        help="fail if Chrome skips a capability supported by any engine",
    )


def pytest_terminal_summary(terminalreporter, exitstatus, config):
    counts = getattr(config, "_vibium_capability_counts", Counter())
    # Suites without capability markers (unit tests, browser-mode tests) have
    # nothing to report; printing there is noise.
    if not counts["marked"] and not config.getoption("capability_audit"):
        return
    terminalreporter.write_sep(
        "-",
        "capabilities: "
        f"engine={_engine()} collected={counts['collected']} "
        f"selected={counts['selected']} skipped={counts['skipped']}",
    )
    for key in sorted(k for k in counts if k.startswith("skip:")):
        terminalreporter.write_line(f"capabilities: {key}={counts[key]}")


def pytest_sessionfinish(session, exitstatus):
    """Send one deterministic collection summary back from xdist workers."""
    if hasattr(session.config, "workeroutput"):
        counts = getattr(session.config, "_vibium_capability_counts", Counter())
        session.config.workeroutput["vibium_capability_counts"] = dict(counts)


def pytest_testnodedown(node, error):
    """All xdist workers collect the same items; retain (do not sum) one copy."""
    raw = node.workeroutput.get("vibium_capability_counts")
    if raw is not None and not hasattr(node.config, "_vibium_capability_counts"):
        node.config._vibium_capability_counts = Counter(raw)
