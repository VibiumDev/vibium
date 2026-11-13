"""What the Python client sends for say and listen.

A fake client stands in for the engine, so no browser, engine or audio is
needed.
"""

import os
from unittest.mock import MagicMock

import pytest

from vibium import errors
from vibium.async_api.browser import Browser
from vibium.async_api.page import Page
from vibium.speech import ListenResult, SayResult, Segment, Word


class FakeClient:
    def __init__(self, answer=None):
        self.calls = []
        self.answer = answer or (lambda method: {})

    def on_event(self, handler):
        pass

    async def send(self, method, params=None, timeout=60):
        self.calls.append((method, params, timeout))
        return self.answer(method)


def raises(error):
    def answer(method):
        raise error
    return answer


async def test_say_and_listen_go_out_as_the_speech_wire_commands():
    client = FakeClient()
    browser, page = Browser(client, None), Page(client, "page-7")

    await page.say("What time is it?", voice="cedar", speed=1.5, cache=False)
    assert client.calls.pop() == (
        "vibium:speech.say",
        {"text": "What time is it?", "voice": "cedar", "speed": 1.5, "cache": False, "context": "page-7"},
        180,
    )

    await browser.say("Hello")
    assert client.calls.pop()[1] == {"text": "Hello"}, "a Browser names no page: the engine uses the current one"

    await page.listen(silence=0, allow_silence=True, transcribe=False)
    assert client.calls.pop() == (
        "vibium:speech.listen",
        {"silence": 0, "allowSilence": True, "transcribe": False, "context": "page-7"},
        100,
    )

    await browser.listen(timeout=5000, duration=20000)
    method, params, timeout = client.calls.pop()
    assert params == {"timeout": 5000, "duration": 20000}
    assert timeout == 85, "the client waits timeout + duration + 60000 ms"


async def test_paths_go_to_the_engine_absolute_and_named_targets_as_they_are():
    client = FakeClient()
    page = Page(client, "page-7")

    await page.say("", file="clip.wav", target="out/said.WAV", cache_dir="cache")
    assert client.calls.pop()[1] == {
        "file": os.path.abspath("clip.wav"), "target": os.path.abspath("out/said.WAV"),
        "cacheDir": os.path.abspath("cache"), "context": "page-7",
    }

    await page.listen(target="reply.wav", path="heard.wav")
    assert client.calls.pop()[1] == {
        "target": os.path.abspath("reply.wav"), "path": os.path.abspath("heard.wav"), "context": "page-7",
    }

    for target in ("session", "local", "speakers"):
        await page.listen(target=target)
        assert client.calls.pop()[1]["target"] == target


async def test_results_are_dataclasses_with_snake_case_fields():
    said = await Page(FakeClient(lambda m: {
        "text": "Hello", "target": "session", "provider": "openai", "model": "gpt-4o-mini-tts", "voice": "cedar",
        "durationMs": 900, "cached": True, "sha256": "ab12",
    }), "page-7").say("Hello")
    assert said == SayResult(text="Hello", target="session", provider="openai", model="gpt-4o-mini-tts",
                             voice="cedar", duration_ms=900, cached=True, sha256="ab12")
    assert said.path is None and said.file is None

    heard = await Page(FakeClient(lambda m: {
        "text": "Okay.", "speech": True, "transcribed": True, "stoppedBy": "silence", "durationMs": 3910,
        "target": "session", "provider": "openai", "model": "gpt-transcribe", "onsetMs": 640, "speechEndMs": 2910,
        "latencyMs": 640, "segments": [{"startMs": 640, "endMs": 2910, "text": "Okay."}],
        "words": [{"startMs": 640, "endMs": 900, "word": "Okay."}], "lostMs": 20,
    }), "page-7").listen()
    assert isinstance(heard, ListenResult)
    assert (heard.text, heard.speech, heard.transcribed, heard.stopped_by) == ("Okay.", True, True, "silence")
    assert (heard.duration_ms, heard.onset_ms, heard.speech_end_ms, heard.latency_ms, heard.lost_ms) == (3910, 640, 2910, 640, 20)
    assert heard.segments == [Segment(640, 2910, "Okay.")]
    assert heard.words == [Word(640, 900, "Okay.")]

    quiet = await Page(FakeClient(lambda m: {"text": "", "speech": False, "segments": [], "stoppedBy": "timeout"}), "page-7").listen(allow_silence=True)
    assert quiet.segments == [] and quiet.words is None and quiet.latency_ms is None


async def test_a_failure_with_a_speech_code_is_a_speech_error_and_any_other_failure_is_not():
    for call in (lambda p: p.say("Hello"), lambda p: p.listen()):
        with pytest.raises(errors.SpeechError) as caught:
            await call(Page(FakeClient(raises(errors.BiDiError("no_speech", "listened for 10.0 s and heard no speech"))), "page-7"))
        assert caught.value.code == "no_speech"
        assert caught.value.message == "listened for 10.0 s and heard no speech"
        assert str(caught.value) == "no_speech: listened for 10.0 s and heard no speech"
        assert isinstance(caught.value, errors.VibiumError)

        with pytest.raises(errors.BiDiError):
            await call(Page(FakeClient(raises(errors.BiDiError("error", "unsupported argument pitch"))), "page-7"))
        with pytest.raises(errors.TimeoutError):
            await call(Page(FakeClient(raises(errors.TimeoutError("Command timed out"))), "page-7"))


def test_the_sync_client_passes_say_and_listen_on_and_outwaits_them():
    from vibium.sync_api.browser import Browser as SyncBrowser
    from vibium.sync_api.page import Page as SyncPage

    # The wrappers reach for a page's keyboard and the like when built.
    fake = MagicMock()
    fake.say = lambda text, **options: ("say", text, options)
    fake.listen = lambda **options: ("listen", options)

    class FakeLoop:
        def __init__(self):
            self.calls = []

        def run(self, coro, timeout=120):
            self.calls.append((coro, timeout))
            return coro

    for cls in (SyncBrowser, SyncPage):
        loop = FakeLoop()
        obj = cls(fake, loop)
        obj.say("Hello", voice="cedar")
        (kind, text, options), timeout = loop.calls.pop()
        assert (kind, text, options["voice"], options["cache"], timeout) == ("say", "Hello", "cedar", True, 185)
        obj.listen(timeout=5000, duration=20000)
        (kind, options), timeout = loop.calls.pop()
        assert (kind, options["timeout"], options["duration"], options["transcribe"], timeout) == ("listen", 5000, 20000, True, 90)
        obj.listen()
        assert loop.calls.pop()[1] == 105


def test_every_entry_point_exports_the_speech_surface():
    import vibium
    import vibium.async_api
    import vibium.sync_api

    for module in (vibium, vibium.async_api, vibium.sync_api):
        for name in ("SpeechError", "SayResult", "ListenResult"):
            assert hasattr(module, name), f"{module.__name__} has no {name}"
        assert callable(module.browser.say) and callable(module.browser.listen)
