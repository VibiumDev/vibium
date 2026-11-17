# Introducing Say and Listen

Vibium can talk now. And listen.

**Say** turns text into speech and plays it into a page's microphone, or a
phone's. **Listen** hears what comes back and turns it into text, with how
long it took to answer. Text-to-speech in, speech-to-text out.

That means you can test the things screen-only tools can't: voice
assistants, voice agents, calls, and any app where sound matters.

```bash
vibium say "What's the weather like in Chicago tomorrow?"
vibium listen
# It's 72 degrees and sunny in Chicago.
```

Say and Listen shipped in Vibium 26.11, in the CLI, the MCP server, and the
JS, Python and Java clients.

---

## Try it in 30 seconds

No browser needed. Say something into a WAV file, then listen to it:

```bash
vibium say "Come here, Watson" --target watson.wav
# Wrote watson.wav (1.2 s, system)

vibium listen --target watson.wav
# Come here, Watson.
```

On a Mac, `say` works with no setup: it uses the system voice. `listen`
needs a speech-to-text provider. Set `OPENAI_API_KEY`, or install
[whisper.cpp](https://github.com/ggml-org/whisper.cpp) or
[sherpa-onnx](https://github.com/k2-fsa/sherpa-onnx) to keep everything on
your machine. Then check both directions:

```bash
vibium ready speech
```

---

## Talk to a page

With a browser open, Say speaks into the page's microphone and Listen hears
what the page plays back. The page doesn't know it's talking to a script:
it gets a microphone like any other.

```bash
vibium go https://voice.example.com
vibium click "#talk"                  # the page opens its microphone
vibium say "Come here, Watson"        # the page hears it
vibium listen --json                  # what the page says back
# {"ok":true,"result":{"text":"Come here, Watson.","latencyMs":5550,"stoppedBy":"silence",...}}
```

`latencyMs` is the reply latency: from the end of what you said to the
start of the answer. Listen starts hearing where Say left off, so a reply
that begins before you call `listen` isn't missed.

---

## Talk to a real phone

The same two commands work on a real phone through
[Roadie](https://github.com/VibiumDev/roadie). Say speaks into the phone's
microphone. Listen hears the phone's speaker. That works for native apps,
the OS's own voice assistant, and live phone calls.

Through a [Vibium Station](https://github.com/VibiumDev/station), the phone
can be anywhere. Speech can run on the station itself.

```bash
export VIBIUM_CONNECT_API_KEY=...     # the station's API key
vibium start wss://station.example/session
vibium say "Set a timer for ten minutes"
vibium listen
# Okay, your timer is set for ten minutes.
```

---

## From code

JavaScript:

```js
import { browser } from 'vibium';

await browser.say('Set a timer for ten minutes', { target: 'timer.wav' });
const heard = await browser.listen({ target: 'timer.wav' });
console.log(heard.text); // Set a timer for ten minutes.
```

Python:

```python
from vibium import browser

browser.say("What time is it?", target="time.wav")
heard = browser.listen(target="time.wav")
print(heard.text)  # What time is it?
```

With a browser started, the same calls go to the page. `page.say()` and
`page.listen()` pin a specific page. Java has `browser.say()`,
`browser.listen()`, and `Vibium.say()` / `Vibium.listen()` with no browser.
See the [API reference](../reference/api.md#say-and-listen).

Coding agents get it through MCP as `vibium_say` and `vibium_listen`.

---

## Your speech, your choice

Say and Listen pick a provider on their own. They try the operating system
first, then an open-source engine you installed, then a hosted API with a
key set.

| | Say (text-to-speech) | Listen (speech-to-text) |
|---|---|---|
| Built into the OS | macOS `say` | none |
| On your machine | Piper, Kitten | whisper.cpp, Moonshine, sherpa-onnx |
| Hosted | OpenAI | OpenAI |

Pick one explicitly with `--provider`, or set `VIBIUM_SAY_PROVIDER` and
`VIBIUM_LISTEN_PROVIDER`. Everything can run offline, even on a Raspberry Pi.

Starting a model on every call is slow. Keep one loaded instead:

- [Keep a Piper voice loaded](../how-to-guides/keep-a-piper-voice-loaded.md)
- [Keep a speech-to-text model loaded](../how-to-guides/keep-a-speech-to-text-model-loaded.md)

Rendered speech is cached, so the same text plays the same audio on every
run. Tests stay repeatable.

---

## Recordings with sound

Recordings keep both sides of the conversation: what was said and what was
heard, on the same timeline as the clicks and screenshots. Device recordings
include video with sound.

The [Record Player](https://player.vibium.dev) plays it all back and
captions what was said and heard.

---

## Learn more

- [Say and Listen API](../reference/api.md#say-and-listen)
- `vibium say --help` and `vibium listen --help`
- [How Say and Listen work](../specs/say-listen-spec.md): targets, providers, the capture window, and errors
- [Record video](../how-to-guides/record-video.md)
