"""Say and listen: results, the wire commands, and the engine the forms with no browser share."""

from __future__ import annotations

import asyncio
import atexit
import os
import threading
from concurrent.futures import Future
from dataclasses import dataclass, field
from typing import Any, Dict, List, Optional, TYPE_CHECKING

from .errors import BiDiError, SpeechError

if TYPE_CHECKING:
    from .binary import VibiumProcess
    from .client import BiDiClient

SAY_TIMEOUT = 180.0
_LISTEN_TIMEOUT_MS = 10_000
_LISTEN_DURATION_MS = 30_000
# The sync API's wait outlasts the command's, so the command's own timeout
# is the error seen.
SYNC_SLACK = 5.0
# The first say or listen with no browser also starts the engine.
_ENGINE_START = 70.0


@dataclass
class SayResult:
    text: str
    target: str
    provider: str
    model: str
    voice: str
    duration_ms: float
    cached: bool
    # SHA-256 of the audio that was played.
    sha256: str
    # Set for file targets.
    path: Optional[str] = None
    # Set when a file was played in place of text.
    file: Optional[str] = None


@dataclass
class Segment:
    start_ms: float
    end_ms: float
    text: str


@dataclass
class Word:
    start_ms: float
    end_ms: float
    word: str


@dataclass
class ListenResult:
    text: str
    # False when the capture was not sent to a provider.
    transcribed: bool
    # False when no speech was detected (only with allow_silence).
    speech: bool
    # "silence", "duration", "eof" or "timeout".
    stopped_by: str
    # Length of the captured audio.
    duration_ms: float
    target: str
    provider: str
    model: str
    # Times are ms from the start of the capture.
    segments: List[Segment] = field(default_factory=list)
    # Set when the provider returns word timings.
    words: Optional[List[Word]] = None
    # When speech started and ended in the capture.
    onset_ms: Optional[float] = None
    speech_end_ms: Optional[float] = None
    # Speech onset minus the end of the preceding say.
    latency_ms: Optional[float] = None
    path: Optional[str] = None
    # Audio the target dropped on its way here, in ms. The transcript may miss words.
    lost_ms: Optional[float] = None


def listen_timeout(timeout: Optional[int], duration: Optional[int]) -> float:
    """Seconds a listen may take: its whole timeout, its whole duration, and the transcript."""
    wait = _LISTEN_TIMEOUT_MS if timeout is None else timeout
    capture = _LISTEN_DURATION_MS if duration is None else duration
    return (wait + capture + 60_000) / 1000


def _path(value: Optional[str]) -> Optional[str]:
    # The engine may run in another directory.
    return os.path.abspath(value) if value else value


def _target(value: Optional[str]) -> Optional[str]:
    return _path(value) if value and value.lower().endswith(".wav") else value


def say_params(text: str, context: Optional[str] = None, *, file: Optional[str] = None, target: Optional[str] = None,
               provider: Optional[str] = None, model: Optional[str] = None, voice: Optional[str] = None,
               language: Optional[str] = None, speed: Optional[float] = None, cache: bool = True,
               cache_dir: Optional[str] = None) -> Dict[str, Any]:
    params: Dict[str, Any] = {"text": text or None, "file": _path(file), "target": _target(target), "provider": provider,
                              "model": model, "voice": voice, "language": language, "speed": speed,
                              "cacheDir": _path(cache_dir), "context": context or None}
    if not cache:
        params["cache"] = False
    return {k: v for k, v in params.items() if v is not None}


def listen_params(context: Optional[str] = None, *, target: Optional[str] = None, provider: Optional[str] = None,
                  model: Optional[str] = None, language: Optional[str] = None, silence: Optional[int] = None,
                  timeout: Optional[int] = None, duration: Optional[int] = None, allow_silence: bool = False,
                  path: Optional[str] = None, transcribe: bool = True) -> Dict[str, Any]:
    params: Dict[str, Any] = {"target": _target(target), "provider": provider, "model": model, "language": language,
                              "silence": silence, "timeout": timeout, "duration": duration, "path": _path(path),
                              "context": context or None}
    if allow_silence:
        params["allowSilence"] = True
    if not transcribe:
        params["transcribe"] = False
    return {k: v for k, v in params.items() if v is not None}


def _say_result(r: Dict[str, Any]) -> SayResult:
    return SayResult(text=r.get("text", ""), target=r.get("target", ""), provider=r.get("provider", ""),
                     model=r.get("model", ""), voice=r.get("voice", ""), duration_ms=r.get("durationMs", 0),
                     cached=r.get("cached", False), sha256=r.get("sha256", ""), path=r.get("path"), file=r.get("file"))


def _listen_result(r: Dict[str, Any]) -> ListenResult:
    words = r.get("words")
    return ListenResult(
        text=r.get("text", ""), transcribed=r.get("transcribed", False), speech=r.get("speech", False),
        stopped_by=r.get("stoppedBy", ""), duration_ms=r.get("durationMs", 0), target=r.get("target", ""),
        provider=r.get("provider", ""), model=r.get("model", ""),
        segments=[Segment(s.get("startMs", 0), s.get("endMs", 0), s.get("text", "")) for s in r.get("segments") or []],
        words=None if words is None else [Word(w.get("startMs", 0), w.get("endMs", 0), w.get("word", "")) for w in words],
        onset_ms=r.get("onsetMs"), speech_end_ms=r.get("speechEndMs"), latency_ms=r.get("latencyMs"),
        path=r.get("path"), lost_ms=r.get("lostMs"))


async def _call(client: BiDiClient, method: str, params: Dict[str, Any], timeout: float) -> Dict[str, Any]:
    try:
        return await client.send(method, params, timeout=timeout)
    except BiDiError as err:
        # The engine answers a speech failure with its code in place of the
        # generic "error".
        if err.error != "error":
            raise SpeechError(err.error, err.message) from None
        raise


async def send_say(client: BiDiClient, params: Dict[str, Any]) -> SayResult:
    return _say_result(await _call(client, "vibium:speech.say", params, SAY_TIMEOUT))


async def send_listen(client: BiDiClient, params: Dict[str, Any]) -> ListenResult:
    return _listen_result(await _call(client, "vibium:speech.listen", params,
                                      listen_timeout(params.get("timeout"), params.get("duration"))))


class _SharedEngine:
    """One engine with no browser, for every say and listen that has none.

    It runs on a loop thread of its own, so it outlives the caller's event
    loop, and the capture window a say opens is still there for the next
    listen.
    """

    def __init__(self) -> None:
        self._guard = threading.Lock()
        self._thread: Any = None
        self._lock: Optional[asyncio.Lock] = None
        self._process: Optional[VibiumProcess] = None
        self._client: Optional[BiDiClient] = None

    def submit(self, coro_fn: Any, params: Dict[str, Any]) -> Future:
        """Run a send function against the engine. The result is a concurrent future."""
        with self._guard:
            if self._thread is None:
                from ._sync_base import _EventLoopThread
                self._thread = _EventLoopThread()
                self._thread.start()
                atexit.register(self.stop)
            loop = self._thread._loop
        return asyncio.run_coroutine_threadsafe(self._run(coro_fn, params), loop)

    async def _run(self, coro_fn: Any, params: Dict[str, Any]) -> Any:
        if self._lock is None:
            self._lock = asyncio.Lock()
        async with self._lock:
            if self._process is None or self._process._process.returncode is not None:
                from .binary import VibiumProcess
                from .client import BiDiClient
                self._process = await VibiumProcess.start(no_browser=True)
                self._client = await BiDiClient.connect(self._process)
                # Exit handlers run last in, first out. The process added
                # one that kills it; this one, which closes it, goes first.
                atexit.unregister(self.stop)
                atexit.register(self.stop)
            client = self._client
        return await coro_fn(client, params)

    async def _close(self) -> None:
        client, process, self._client, self._process = self._client, self._process, None, None
        try:
            if client:
                await client.close()
        finally:
            if process:
                await process.stop()

    def stop(self) -> None:
        with self._guard:
            thread, self._thread, self._lock = self._thread, None, None
        if thread is None:
            return
        atexit.unregister(self.stop)
        loop = thread._loop
        try:
            thread.run(self._close(), timeout=10)
        finally:
            thread.stop()
            if not loop.is_running():
                loop.close()


_engine = _SharedEngine()


async def shared_say(params: Dict[str, Any]) -> SayResult:
    return await asyncio.wrap_future(_engine.submit(send_say, params))


async def shared_listen(params: Dict[str, Any]) -> ListenResult:
    return await asyncio.wrap_future(_engine.submit(send_listen, params))


def shared_say_sync(params: Dict[str, Any]) -> SayResult:
    return _engine.submit(send_say, params).result(timeout=SAY_TIMEOUT + _ENGINE_START)


def shared_listen_sync(params: Dict[str, Any]) -> ListenResult:
    wait = listen_timeout(params.get("timeout"), params.get("duration"))
    return _engine.submit(send_listen, params).result(timeout=wait + _ENGINE_START)
