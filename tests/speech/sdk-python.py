# The speech scenario of surfaces.test.js, through the Python client.
import asyncio
import dataclasses
import json
import os

TEXT = "Come here, Watson."
SYNC = os.environ["SPEECH_TEST_SYNC"] == "1"


def plain(result):
    return {k: v for k, v in dataclasses.asdict(result).items() if v is not None}


def failure(err):
    return {"type": type(err).__name__, "code": err.code, "message": err.message}


async def local_async():
    from vibium.async_api import browser, SpeechError
    out = {}
    out["said"] = plain(await browser.say(TEXT, target="said.wav"))
    out["again"] = plain(await browser.say(TEXT, target="again.wav"))
    out["fresh"] = plain(await browser.say(TEXT, target="fresh.wav", cache=False))
    out["heard"] = plain(await browser.listen(target="said.wav", silence=0))
    out["unheard"] = plain(await browser.listen(target="said.wav", transcribe=False, path="copy.wav"))
    try:
        await browser.listen(target="quiet.wav")
    except SpeechError as err:
        out["quiet"] = failure(err)
    out["allowed"] = plain(await browser.listen(target="quiet.wav", allow_silence=True))
    try:
        await browser.say(TEXT, target="session")
    except SpeechError as err:
        out["no_session"] = failure(err)
    return out


def local_sync():
    from vibium import browser, SpeechError
    out = {}
    out["said"] = plain(browser.say(TEXT, target="said.wav"))
    out["again"] = plain(browser.say(TEXT, target="again.wav"))
    out["fresh"] = plain(browser.say(TEXT, target="fresh.wav", cache=False))
    out["heard"] = plain(browser.listen(target="said.wav", silence=0))
    out["unheard"] = plain(browser.listen(target="said.wav", transcribe=False, path="copy.wav"))
    try:
        browser.listen(target="quiet.wav")
    except SpeechError as err:
        out["quiet"] = failure(err)
    out["allowed"] = plain(browser.listen(target="quiet.wav", allow_silence=True))
    try:
        browser.say(TEXT, target="session")
    except SpeechError as err:
        out["no_session"] = failure(err)
    return out


def start_options():
    return {"headless": True, "engine": os.environ["SPEECH_TEST_ENGINE"], "channel": os.environ.get("VIBIUM_ENGINE_CHANNEL") or None}


async def page_async():
    from vibium.async_api import browser, SpeechError
    out = {}
    bro = await browser.start(**start_options())
    try:
        page = await bro.page()
        await page.go(os.environ["SPEECH_TEST_URL"])
        await (await page.find(css="#probe")).click()
        out["clock"] = await page.evaluate(os.environ["SPEECH_TEST_CLOCK"])
        if not out["clock"]["advanced"] >= 300:
            return out
        try:
            await page.say("", file="tone.wav")
        except SpeechError as err:
            out["early"] = failure(err)
        await (await page.find(css="#talk")).click()
        out["said"] = plain(await page.say("", file="tone.wav"))
        out["heard"] = plain(await bro.listen(transcribe=False, timeout=9000, path="heard.wav"))
    finally:
        await bro.stop()
    return out


def page_sync():
    from vibium import browser, SpeechError
    out = {}
    bro = browser.start(**start_options())
    try:
        page = bro.page()
        page.go(os.environ["SPEECH_TEST_URL"])
        page.find(css="#probe").click()
        out["clock"] = page.evaluate(os.environ["SPEECH_TEST_CLOCK"])
        if not out["clock"]["advanced"] >= 300:
            return out
        try:
            page.say("", file="tone.wav")
        except SpeechError as err:
            out["early"] = failure(err)
        page.find(css="#talk").click()
        out["said"] = plain(page.say("", file="tone.wav"))
        out["heard"] = plain(bro.listen(transcribe=False, timeout=9000, path="heard.wav"))
    finally:
        bro.stop()
    return out


if os.environ["SPEECH_TEST_MODE"] == "page":
    result = page_sync() if SYNC else asyncio.run(page_async())
else:
    result = local_sync() if SYNC else asyncio.run(local_async())
print(json.dumps(result))
