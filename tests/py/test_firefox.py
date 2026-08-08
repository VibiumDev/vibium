"""Firefox smoke tests (sync API).

Skip when Firefox is not installed, so the suite stays green on machines
that only have Chrome. Install with: vibium install --engine firefox

CI sets VIBIUM_REQUIRE_FIREFOX, which turns every skip into a failure: the
green check must prove Firefox and video recording actually ran.
"""

import io
import os
import subprocess
import zipfile

import pytest

from vibium import browser, firefox


def _firefox_installed() -> bool:
    bin_path = os.environ.get("VIBIUM_BIN_PATH")
    if not bin_path:
        return False
    try:
        result = subprocess.run(
            [bin_path, "is-installed", "--engine", "firefox"],
            capture_output=True,
            timeout=10,
        )
        return result.returncode == 0
    except (subprocess.TimeoutExpired, subprocess.SubprocessError):
        return False


def _skip_or_fail(reason: str) -> None:
    if os.environ.get("VIBIUM_REQUIRE_FIREFOX"):
        pytest.fail(f"{reason}, and VIBIUM_REQUIRE_FIREFOX is set")
    pytest.skip(reason)


@pytest.fixture(autouse=True)
def _require_firefox():
    if not _firefox_installed():
        _skip_or_fail("Firefox not installed")


def test_firefox_smoke(test_server):
    # Named launcher: firefox.start() == browser.start(engine="firefox")
    bro = firefox.start(headless=True)
    try:
        vibe = bro.page()
        vibe.go(test_server)
        assert vibe.title() == "Test App"
        assert len(vibe.screenshot()) > 1000
    finally:
        bro.stop()


def test_firefox_recording_video(test_server, tmp_path):
    bro = browser.start(engine="firefox", headless=True)
    try:
        vibe = bro.page()
        vibe.go(test_server)
        try:
            vibe.context.recording.start(video=True, path=str(tmp_path / "run.zip"))
        except Exception as err:
            # Firefox gains BiDi screencast in 154; self-skip on older builds
            # so this activates on its own once release catches up.
            if "not supported" in str(err):
                _skip_or_fail("this Firefox does not support video recording yet")
            raise
        # Force a series of paints. A navigation can finish without Firefox's
        # encoder observing a frame, yielding a valid but empty ~200-byte WebM.
        vibe.evaluate("""
            (() => {
                const box = document.createElement('div');
                Object.assign(box.style, {
                    position: 'fixed', width: '100px', height: '100px',
                    background: 'red', left: '0px', top: '0px'
                });
                document.body.appendChild(box);
                let frame = 0;
                const animate = () => {
                    box.style.left = `${frame++ % 200}px`;
                    if (frame < 60) requestAnimationFrame(animate);
                };
                requestAnimationFrame(animate);
            })()
        """)
        vibe.wait(1200)
        result = vibe.context.recording.stop()
        assert result.path == str(tmp_path / "run.zip")
        assert (tmp_path / "run.zip").exists()
        assert len(result.videos) == 1 and not result.videos[0].get("error")
        with zipfile.ZipFile(io.BytesIO((tmp_path / "run.zip").read_bytes())) as zf:
            videos = [n for n in zf.namelist() if n.startswith("video/") and n.endswith(".webm")]
            assert len(videos) == 1
            video = zf.read(videos[0])
            assert len(video) > 1000
            assert video[:4] == b"\x1a\x45\xdf\xa3"  # WebM EBML magic
            assert "video/index.json" in zf.namelist()
    finally:
        bro.stop()
