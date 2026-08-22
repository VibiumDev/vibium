"""BiDiClient deadline tests (#397).

CI hangs outlived every client timeout: captures cap at 10s, download
completion at 300s, commands at 60s, yet suites sat for 8+ minutes until
the 600s watchdog killed them. The one await in the command path with no
deadline was stdin drain(): when the vibium process stops reading its
pipe, _send blocked before the response timer ever started, and the
setup gate in send() inherited the hang. These tests pin _send to a
deadline using in-process fakes; no browser, no binary.
"""

import asyncio

import pytest

from vibium import errors
from vibium.client import BiDiClient

pytestmark = pytest.mark.asyncio(loop_scope="function")


class _WedgedStdin:
    """A pipe whose buffer never drains, like a binary that stopped reading."""

    def write(self, data: bytes) -> None:
        pass

    async def drain(self) -> None:
        await asyncio.Event().wait()


class _SilentStdout:
    """A pipe that never produces a line."""

    async def readline(self) -> bytes:
        await asyncio.Event().wait()
        return b""


class _FakePipes:
    def __init__(self):
        self.stdin = _WedgedStdin()
        self.stdout = _SilentStdout()


class _FakeProcess:
    def __init__(self):
        self._process = _FakePipes()


async def test_send_fails_fast_when_stdin_never_drains():
    client = BiDiClient(_FakeProcess())
    # The outer wait_for is the test's own tripwire: without the drain
    # deadline the send hangs and this raises asyncio.TimeoutError, which
    # is not the vibium TimeoutError the assertion demands.
    with pytest.raises(errors.TimeoutError, match="not accepted"):
        await asyncio.wait_for(client._send("session.status", timeout=0.2), timeout=5)


async def test_setup_gate_cannot_outlive_its_setups():
    client = BiDiClient(_FakeProcess())
    # A registration wedged on the same dead pipe must not park every
    # later command forever: the setup fails at its own deadline, the
    # gate opens, and the command reports its own drain timeout.
    client.send_setup("script.addPreloadScript", timeout=0.2)
    with pytest.raises(errors.TimeoutError, match="not accepted"):
        await asyncio.wait_for(client.send("browsingContext.navigate", timeout=0.2), timeout=5)
