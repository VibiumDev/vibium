# Spec: `say` and `listen`

> **Status: Draft.** This document describes proposed behavior. It does
> not reflect what is currently implemented in vibium. Do not rely on it
> as documentation of existing functionality.

## Summary

Vibium gains two commands.

- `vibium say "<text>"` turns text into speech and plays it into a target.
- `vibium listen` captures audio from a target and returns a transcript.

Both exist in the CLI, the MCP server, and the JS/TS, Python and Java
clients. Together they let a test talk to a voice interface and check
what it answers and how fast.

```bash
vibium say "What's the weather like in Chicago tomorrow?"
vibium listen --json
# {"ok":true,"result":{"text":"It's 72 degrees and sunny in Chicago.","latencyMs":840,...}}
```

## Goals

- Speak a phrase into a phone on a Roadie, a page in a browser, the local speakers, or a WAV file.
- Hear the reply from the same kinds of target and return its text with timestamps.
- Follow the session. With none open, use this machine's speakers and microphone.
- Measure reply latency from the moment speech starts in the captured audio.
- Run fully offline on a Raspberry Pi 4 with open-source engines.
- Replay identical audio on repeat runs.
- Put what was said and heard into record.zip, on the same timeline as actions and screenshots.

## Non-goals

These are future work. See [Future work](#future-work).

- A plugin system or plugin install.
- Streaming partial transcripts.
- Continuous synced audio playback in the Record Player.
- Audio compression.
- Barge-in testing.

## Concepts

### Targets

A target is where audio goes or comes from. Both commands take the same
`target` option.

| Target | `say` | `listen` |
|--------|-------|----------|
| `session` | Plays into whatever the current session drives. | Captures what it plays. |
| `local` | Plays on this machine's default speakers. | Captures from this machine's default microphone. |
| a path ending in `.wav` | Writes the rendered audio to the file. | Reads the file as if it were live audio. |

`session` depends on what the session drives:

| Session | `say` reaches | `listen` hears |
|---------|---------------|----------------|
| A phone on a Roadie, directly or through a station | The phone's microphone | The phone's speaker output |
| A browser, local or remote | The page's microphone | What the page plays |

The default follows the session.

- A session is open: the default is `session`.
- No session is open: the default is `local`.

```bash
vibium say "Hello"                     # no session: this machine's speakers
vibium go https://voice.example.com
vibium click "#talk"                   # the page opens its microphone
vibium say "What's on my calendar?"    # the page's microphone
vibium listen                          # what the page says back
```

The default MUST NOT fall back. With a session open, a `say` the session
cannot carry fails. It does not play on the speakers instead. The error
names `--target local`.

`local` and file targets MUST NOT launch or require a browser.

### Providers

A provider turns text into audio or audio into text. Text-to-speech (TTS)
and speech-to-text (STT) are configured separately.

- **Hosted providers** call an HTTP API. They are built into the core.
- **The `command` provider** runs a CLI the user installed, as a
  subprocess. Local engines run this way. Presets cover the recommended
  engines.

Speech engines stay out of the Vibium binary. Vibium links no speech
model runtime. GPL code such as espeak-ng runs only in the user's own
subprocess.

### The capture window

A reply can start before the client gets around to calling `listen`. So
capture does not wait for the call.

- When `say` plays into a target that can be heard, the engine captures
  from that target during playback. It keeps the audio from the moment
  playback ends, up to 30 seconds. That is the capture window.
- The next `listen` on that target starts at the end of that `say`, not
  at the time of the call.
- A new `say`, or the end of the session, discards the window.
- While a recording with audio is running, the continuous capture track
  serves as the window.

A `listen` with no `say` before it starts capturing when it is called.

One `say` or `listen` runs on a target at a time. A second call while
one is running fails with `busy`.

### Where the commands run

- `session` needs an open session: a browser, or a device reached with
  `vibium start <roadie-url>` or `browser.start(url)`. `say` and
  `listen` never open one.
- CLI commands go to the daemon and start it when needed. The daemon
  handles `say` and `listen` without launching a browser. The capture
  window lives in the daemon, so `vibium say` followed by
  `vibium listen` measures latency.
- The no-browser SDK forms share one engine process for the life of the
  client process. `say` then `listen` on `local` measures latency there too.

## CLI

### `vibium say`

```
vibium say <text> [flags]
vibium say -            # read the text from stdin
vibium say --file <clip.wav> [flags]
```

| Flag | Default | Meaning |
|------|---------|---------|
| `--target` | `session` or `local` (see Targets) | `session`, `local`, or a `.wav` path |
| `--provider` | from config | TTS provider or preset name |
| `--model` | provider default | Provider's model id |
| `--voice` | provider default | Provider's voice id |
| `--language` | provider default | BCP 47 tag, for providers that take one |
| `--speed` | `1.0` | Speaking rate multiplier, where the provider supports it |
| `--no-cache` | off | Render again and replace the cached audio |
| `--cache-dir` | see [Cache](#cache) | Where rendered audio is kept |
| `--file` | none | A WAV file to play as it is, in place of text |
| `--json` | off | Print the full result as JSON |

`say` returns when playback has finished.

With `--file` nothing is rendered. The file is played as it is: not
trimmed, not checked, not cached. No provider or key is needed. Use it
for a recorded voice, a tone, or noise.

```bash
vibium say "Set a timer for ten minutes"
# Said "Set a timer for ten minutes" (1.9 s, openai gpt-4o-mini-tts alloy, cached)

vibium say "Hello there" --target hello.wav
# Wrote hello.wav (0.9 s, openai gpt-4o-mini-tts alloy)

echo "What time is it?" | vibium say - --target local --provider openai
# Said "What time is it?" (1.2 s, openai gpt-4o-mini-tts alloy)
```

### `vibium listen`

```
vibium listen [flags]
```

| Flag | Default | Meaning |
|------|---------|---------|
| `--target` | `session` or `local` (see Targets) | `session`, `local`, or a `.wav` path |
| `--provider` | from config | STT provider or preset name |
| `--model` | provider default | Provider's model id |
| `--language` | `en` | BCP 47 tag |
| `--silence` | `1000` | Milliseconds of trailing silence that end the reply. `0` turns endpointing off |
| `--timeout` | `10000` | Milliseconds to wait for speech to start |
| `--duration` | `30000` | Maximum milliseconds of capture. With `--silence 0` the capture runs exactly this long |
| `--allow-silence` | off | Return an empty transcript instead of failing when nothing is heard |
| `-o, --output` | none | Also save the captured audio as a WAV file |
| `--no-transcribe` | off | Capture without asking a provider for the words |
| `--json` | off | Print the full result as JSON |

Plain output is the transcript alone, so shell scripts can capture it.

With `--no-transcribe` no provider is called and none is needed. Speech
is still detected on the machine, so the result has the onset, the end
and the reply latency. Only `text` is missing. This is the fast path
during a run: save the audio with `-o`, and transcribe it afterwards
with `vibium listen --target reply.wav`.

```bash
vibium listen
# It's currently 72 degrees and sunny in Chicago.

vibium listen --timeout 5000 --json
# {"ok":true,"result":{"text":"Okay, your timer is set for ten minutes.","speech":true,
#  "onsetMs":640,"speechEndMs":2910,"latencyMs":640,"stoppedBy":"silence",
#  "segments":[{"startMs":640,"endMs":2910,"text":"Okay, your timer is set for ten minutes."}],
#  "provider":"openai","model":"gpt-transcribe","durationMs":3910}}

vibium listen --target reply.wav --silence 0
# Transcribes the whole file.

vibium listen --silence 0 --duration 5000 -o five-seconds.wav
# Captures exactly five seconds.

vibium say --file question.wav
# Played question.wav (2.3 s)
vibium listen --no-transcribe -o reply.wav
# Heard 2.1 s of speech after 640 ms (reply.wav)
```

Both commands MUST be built so a text or path that begins with `-` is not
read as a flag where that matters (`say -- "-5 degrees"`).

## SDKs

`say` and `listen` live on `Page` and `Browser`, as `run` and `check` do.
On a `Page` they act on that page. On a `Browser` they act on the
current page. Both default to `session`.

Module-level forms run without a browser, for `local` and file targets.
They follow the pattern of module-level `browser.check`.

### JavaScript / TypeScript

```ts
interface SpeechOptions {
  /** 'session', 'local', or a path ending in .wav. */
  target?: string;
  provider?: string;
  model?: string;
  language?: string;
}

interface SayOptions extends SpeechOptions {
  /** A WAV file to play as it is. Pass an empty text with it. */
  file?: string;
  voice?: string;
  speed?: number;
  /** false renders again and replaces the cached audio. Default true. */
  cache?: boolean;
  cacheDir?: string;
}

interface SayResult {
  text: string;
  target: string;
  provider: string;
  model: string;
  voice: string;
  durationMs: number;
  cached: boolean;
  /** SHA-256 of the audio that was played. */
  sha256: string;
  /** Present for file targets. */
  path?: string;
}

interface ListenOptions extends SpeechOptions {
  /** Trailing silence that ends the reply, in ms. 0 turns endpointing off. Default 1000. */
  silence?: number;
  /** How long to wait for speech to start, in ms. Default 10000. */
  timeout?: number;
  /** Maximum capture length in ms. Default 30000. */
  duration?: number;
  /** Return an empty transcript instead of throwing when nothing is heard. */
  allowSilence?: boolean;
  /** Also save the captured audio as a WAV file. */
  path?: string;
  /** false captures without a transcript. Default true. */
  transcribe?: boolean;
}

interface ListenResult {
  text: string;
  /** False when the capture was not sent to a provider. */
  transcribed: boolean;
  /** False when no speech was detected (only with allowSilence). */
  speech: boolean;
  /** Times are ms from the start of the capture. */
  segments: { startMs: number; endMs: number; text: string }[];
  /** Present when the provider returns word timings. */
  words?: { startMs: number; endMs: number; word: string }[];
  /** When speech started and ended in the capture. */
  onsetMs?: number;
  speechEndMs?: number;
  /** Speech onset minus the end of the preceding say. See Reply latency. */
  latencyMs?: number;
  stoppedBy: 'silence' | 'duration' | 'eof' | 'timeout';
  /** Length of the captured audio. */
  durationMs: number;
  provider: string;
  model: string;
  path?: string;
}

class Page {
  say(text: string, options?: SayOptions): Promise<SayResult>;
  listen(options?: ListenOptions): Promise<ListenResult>;
}
class Browser {
  say(text: string, options?: SayOptions): Promise<SayResult>;
  listen(options?: ListenOptions): Promise<ListenResult>;
}
// No browser: local and file targets only.
browser.say(text: string, options?: SayOptions): Promise<SayResult>;
browser.listen(options?: ListenOptions): Promise<ListenResult>;
```

The sync API mirrors these without promises.

### Python

```python
# async_api; sync_api is the same without await
async def say(self, text: str, *, target: Optional[str] = None,
              provider: Optional[str] = None, model: Optional[str] = None,
              voice: Optional[str] = None, language: Optional[str] = None,
              speed: Optional[float] = None, cache: bool = True,
              cache_dir: Optional[str] = None) -> SayResult: ...

async def listen(self, *, target: Optional[str] = None,
                 provider: Optional[str] = None, model: Optional[str] = None,
                 language: Optional[str] = None, silence: Optional[int] = None,
                 timeout: Optional[int] = None, duration: Optional[int] = None,
                 allow_silence: bool = False,
                 path: Optional[str] = None) -> ListenResult: ...
```

`SayResult` and `ListenResult` are dataclasses with snake_case fields:
`duration_ms`, `onset_ms`, `speech_end_ms`, `latency_ms`, `stopped_by`.

### Java

```java
public SayResult say(String text);
public SayResult say(String text, SayOptions options);
public ListenResult listen();
public ListenResult listen(ListenOptions options);

page.say("What time is it?", SayOptions.builder().voice("en_US-ljspeech-medium").build());
ListenResult reply = page.listen(ListenOptions.builder().timeout(5000).build());
```

`SayOptions`, `ListenOptions`, `SayResult` and `ListenResult` live in
`com.vibium.types`. The options follow the existing builder style.
`Vibium.say` and `Vibium.listen` are the no-browser forms.

### Errors in the SDKs

Each client raises a `SpeechError` with a `code` from
[Errors](#errors). `listen` raises it with code `no_speech` when nothing
is heard and `allowSilence` is off.

## MCP tools

Two tools join `vibium mcp`. Both take the optional `page` argument
every page tool takes. It matters in a browser with several pages.

```json
{
  "name": "vibium_say",
  "description": "Speak text into the target. The default is the current session: the phone on a Roadie, or the page's microphone in a browser. With no session open it is this machine's speakers. Returns when playback ends.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "text": {"type": "string", "description": "What to say. Give text or file"},
      "file": {"type": "string", "description": "A WAV file to play as it is, in place of text"},
      "target": {"type": "string", "description": "session, local, or a path ending in .wav"},
      "provider": {"type": "string"},
      "model": {"type": "string"},
      "voice": {"type": "string"},
      "language": {"type": "string"},
      "speed": {"type": "number"},
      "cache": {"type": "boolean", "description": "false renders again instead of replaying cached audio"}
    },
    "additionalProperties": false
  }
}
```

```json
{
  "name": "vibium_listen",
  "description": "Listen to the target and return what was said. The default is the current session: what the phone or the page plays. With no session open it is this machine's microphone. Starts at the end of the last vibium_say. Stops after trailing silence.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "target": {"type": "string", "description": "session, local, or a path ending in .wav"},
      "provider": {"type": "string"},
      "model": {"type": "string"},
      "language": {"type": "string"},
      "silence": {"type": "number", "description": "Trailing silence that ends the reply, in ms (default 1000)"},
      "timeout": {"type": "number", "description": "How long to wait for speech to start, in ms (default 10000)"},
      "duration": {"type": "number", "description": "Maximum capture length in ms (default 30000)"},
      "allowSilence": {"type": "boolean", "description": "Return an empty transcript instead of an error when nothing is heard"},
      "transcribe": {"type": "boolean", "description": "false captures without a transcript; onset and latency are still measured"}
    },
    "additionalProperties": false
  }
}
```

Tool results are the result objects above as JSON text. A model reads
`text` and `latencyMs` and never sees audio.

The CLI commands call these tools through the daemon, as other commands do.

## Wire protocol

| Method | Params | Result |
|--------|--------|--------|
| `vibium:speech.say` | `text`, plus the `SayOptions` fields | `SayResult` |
| `vibium:speech.listen` | the `ListenOptions` fields | `ListenResult` |

The client timeout for `listen` MUST exceed `timeout + duration` plus
transcription time. Clients use `timeout + duration + 60000` ms.

## Behavior

### say

1. Resolve the provider, model and voice from options, environment and defaults.
2. Compute the cache key. On a hit, load the cached audio.
3. On a miss, call the provider, check the audio, and store it in the cache.
4. Convert to the target's format. Roadie takes 16 kHz mono 16-bit PCM.
5. Open a capture on the target, if it can be heard and none is open.
6. Play into the target and wait for playback to end.
7. Mark the end of playback as the start of the capture window.
8. If a recording is running, add the clip and the step to it.

Rendered audio is trimmed before it is cached. Each end is cut to the
speech and given exactly 50 ms of silence. So the end of playback is the
end of speech, which is what reply latency counts from. Providers pad
their audio by varying amounts: up to 2.9 seconds was measured.

The audio played is exactly the cached audio after conversion. Conversion
is deterministic.

A text longer than the provider accepts is an error. Vibium does not
split text in this version.

### listen

1. Take audio from the capture window if one is open. Otherwise open a capture.
2. Run voice activity detection on the incoming audio.
3. Stop on the first of: trailing silence, the `duration` cap, the end of a file, or no speech by `timeout`.
4. Send the audio from 300 ms before speech onset to 300 ms after speech end to the provider.
5. Map the provider's timestamps back to capture time.
6. Compute reply latency.
7. If a recording is running, add the transcript and the step to it.

Stop rules:

| Rule | Condition | `stoppedBy` |
|------|-----------|-------------|
| Silence | Speech started, then `silence` ms passed without speech | `silence` |
| Duration | `duration` ms of audio captured | `duration` |
| End of file | A WAV target ran out | `eof` |
| Timeout | No speech started within `timeout` ms | error `no_speech`, or `timeout` with `allowSilence` |

All rules apply in audio time. `timeout` and `duration` count from the
start of the capture window, not from the call.

A WAV target is read as fast as possible. A file gives the same result
on every machine.

A reply with a pause longer than `silence` ends at the pause. Raise
`silence` for a slow talker.

Speech that the provider cannot turn into words is not an error. The
result has `speech: true` and an empty `text`.

### Voice activity detection

Vibium detects speech with a small built-in detector. It has no model
and no dependency, so it works with every provider.

- Audio is 16 kHz mono. A one-pole high-pass at 100 Hz removes hum and DC.
- Frames are 20 ms with a 10 ms hop. The level is RMS in dBFS.
- The noise floor is the median level of the first 300 ms. It keeps
  tracking the level while no speech is active. It is clamped to −75 to −45 dBFS.
- Speech starts when 3 of 5 consecutive frames exceed the floor by 12 dB
  and are above −50 dBFS. The start is dated to the first of those frames.
- Speech ends when the level stays below the floor plus 8 dB for `silence` ms.
- On `local`, the detector ignores the first 200 ms after playback. The
  room's echo of the question is not a reply.

Onset is accurate to about one hop. [Measurements](#voice-activity-detection-1)
will compare it with Silero VAD.

A provider MAY return its own speech boundaries. Vibium still reports
its own onset, so latency means the same thing across providers.

### Reply latency

`latencyMs` is the time from the end of the question to the start of the answer.

```
latencyMs = (time speech starts in the captured audio) − (time the last say finished playing)
```

- Both times are taken on the target's audio clock. Through Roadie both
  are Roadie timestamps. In a browser both are page timestamps. Network
  delay and transcription time do not enter.
- The start of speech comes from voice activity detection, not from the provider.
- `latencyMs` is present when a `say` to the same target opened the capture window.
- Without a preceding `say`, `latencyMs` is absent and `onsetMs` counts from the start of the call.
- If the other side is already speaking when `say` ends, `latencyMs` is `0`.
  Telling an early reply from an overlapping one is barge-in testing, which is future work.

### Cache

Rendered speech is cached so repeat runs replay identical audio.

- The key is the SHA-256 of a canonical JSON object: cache version, text,
  provider, model, voice, language, speed, and any provider settings that
  change the audio. For the `command` provider the command line is part of the key.
- The text is hashed as given. Vibium does not normalize it.
- The value is the provider's audio as a WAV file, plus a JSON sidecar
  with the key's fields and the creation time.
- The default directory is `speech/` under Vibium's cache directory.
  `--cache-dir` or `cacheDir` moves it.
- Writes are atomic. Concurrent processes MAY share a directory.
- `--no-cache` or `cache: false` renders again and replaces the entry.
- Nothing expires. Deleting the directory clears the cache.

A project MAY commit a cache directory. Tests then run offline and every
run plays the same bytes.

The sidecar holds the text in the clear. Treat a cache directory like test fixtures.

Transcripts are never cached.

## Providers

### The provider interface

One internal Go interface serves both directions. A provider MAY
implement only one of them.

```go
package speech

// Audio is 16-bit little-endian PCM.
type Audio struct {
	SampleRate int
	Channels   int
	PCM        []byte
}

type SynthesisRequest struct {
	Text     string
	Model    string
	Voice    string
	Language string
	Speed    float64
}

type TranscriptionRequest struct {
	Audio    Audio // 16 kHz mono
	Model    string
	Language string
}

type Transcript struct {
	Text     string
	Language string
	Segments []Segment // may be empty
	Words    []Word    // may be empty
}

type Segment struct {
	Start, End time.Duration // from the start of the audio given
	Text       string
}

type Word struct {
	Start, End time.Duration
	Word       string
}

// Provider turns text into audio and audio into text.
type Provider interface {
	Name() string
	// Synthesize returns ErrUnsupported when the provider has no TTS.
	Synthesize(ctx context.Context, req SynthesisRequest) (Audio, error)
	// Transcribe returns ErrUnsupported when the provider has no STT.
	Transcribe(ctx context.Context, req TranscriptionRequest) (Transcript, error)
	// CacheSettings names what else changes the audio, for the cache key.
	CacheSettings(req SynthesisRequest) map[string]string
}
```

Targets, voice activity detection, the cache, resampling and the
recording are not the provider's business. A provider sees text or
audio and nothing else.

Vibium guarantees one segment per reply from its own detector. Word
timings and finer segments appear when the provider returns them.

### Hosted providers

Hosted providers are HTTP clients in the core. They use the standard
library only.

| Provider | TTS | STT | Key |
|----------|-----|-----|-----|
| `openai` | `POST /v1/audio/speech`, model `gpt-4o-mini-tts`, voice `alloy`, 24 kHz WAV. A model named `gpt-realtime…` speaks through the Realtime API instead. | `POST /v1/audio/transcriptions`, model `gpt-transcribe`. No word timings. | `OPENAI_API_KEY` |
| `deepgram` | `POST /v1/speak`, linear16 WAV at 16 kHz. Default voice `aura-2-thalia-en`. | `POST /v1/listen`, model `nova-3`. Word timings. | `DEEPGRAM_API_KEY` |
| `elevenlabs` | `POST /v1/text-to-speech/{voice}`, model `eleven_flash_v2_5`, `wav_16000` output. | `POST /v1/speech-to-text`, model `scribe_v2`, `timestamps_granularity=word`. Word timings. | `ELEVENLABS_API_KEY` |

`--model` and `--voice` override the defaults.

OpenAI retires `/v1/audio/speech` and its three models on 2027-01-06.
Its replacement is a realtime model on the Realtime API. Vibium speaks
through both. The speech endpoint is the default while it exists,
because it fails far less often. See
[Hosted providers compared](#hosted-providers-compared).

### Checking the audio

A speech model's output is not trusted. Models sometimes render silence
in place of words, most often the last ones. A realtime model does this
while its own transcript claims it said them.

So `say` checks every render before caching it, when the provider can
also transcribe:

1. Render the text.
2. Transcribe the audio with the same provider.
3. Compare the transcript with the text.
4. On a mismatch, render again. After three renders that fail, `say`
   fails with `provider_error` and quotes what the audio said.

A clip that fails its check is never cached and never played.

The comparison ignores formatting. A recognizer writes numbers, money
and abbreviations its own way: "three thirty" comes back as "3:30".

- The first and last words of the text must be heard at the start and
  end of the audio. A number may be heard as any number.
- Most of the text's letters must be in the transcript, and the
  transcript must not be much longer than the text.

The check costs one transcription per new phrase, about a second. A
cached phrase is not checked again. A local engine with no recognizer
configured for the same provider is not checked.

A word dropped from the middle of a long text can pass. The check is
built for the failure that was measured.

### The `command` provider

The `command` provider runs a program the user installed. This is how
local engines run.

The contract is deliberately small. It can become a plugin protocol later.

**Common rules**

- The command line is split into arguments once, shell-style. No shell runs it.
- Placeholders in braces are replaced inside arguments.
- A placeholder that stands alone as an argument and has no value is
  dropped, along with the option before it. `-v {voice}` disappears when
  no voice is set.
- Exit status 0 means success. Anything else fails the call.
- On failure the last 2 KB of stderr go into the error message.
- The environment is the engine's own, plus `VIBIUM_SPEECH_PROTOCOL=1`
  and `VIBIUM_SPEECH_OP=synthesize` or `transcribe`.
- A call is killed after 120 seconds.

**Synthesize: text in, WAV out**

| Placeholder | Replaced with |
|-------------|---------------|
| `{text}` | The text, as one argument. Without it the text arrives on stdin as UTF-8. |
| `{output}` | A temporary `.wav` path the command writes. Without it the WAV is read from stdout. |
| `{voice}` `{model}` `{language}` `{speed}` | The request's values. Empty when unset. |

The WAV MUST be PCM, 16-bit or 32-bit float, mono or stereo, at any
sample rate. Vibium converts it.

```bash
VIBIUM_SAY_COMMAND='piper -m {voice} -f {output} -- {text}'
```

**Transcribe: audio in, JSON out**

| Placeholder | Replaced with |
|-------------|---------------|
| `{input}` | A temporary `.wav` path: PCM, 16-bit, mono, 16 kHz. Without it the WAV arrives on stdin. |
| `{output}` | A temporary path the command writes its result to. Without it the result is read from stdout. |
| `{model}` `{language}` | The request's values. Empty when unset. |

The result is JSON:

```json
{
  "text": "Okay, your timer is set for ten minutes.",
  "language": "en",
  "segments": [{"start": 0.31, "end": 2.58, "text": "Okay, your timer is set for ten minutes."}],
  "words": [{"start": 0.31, "end": 0.62, "word": "Okay"}]
}
```

- Times are seconds from the start of the audio given.
- `text` is required. Everything else is optional.
- Output that is not JSON is taken as the transcript text alone.

```bash
VIBIUM_LISTEN_COMMAND='my-stt --wav {input} --lang {language}'
```

### Presets

A preset is a named `command` configuration built into Vibium: the
command line, the default model or voice, an adapter for the engine's
native output, and an install hint. A preset is data. It contains no
engine code.

`--provider piper` selects a preset. The engine must be on `PATH`, or
`VIBIUM_SAY_COMMAND` / `VIBIUM_LISTEN_COMMAND` names the full command.
Vibium does not install engines or download models.

| Preset | Direction | Command line | Output adapter |
|--------|-----------|--------------|----------------|
| `system` on macOS | TTS | `say -v {voice} -o {output} --data-format=LEI16@22050`, text on stdin | WAV file |
| `piper` | TTS | `piper -m {voice} -f {output} -- {text}` | WAV file |
| `kitten` | TTS | `python3 -c <snippet> {model} {voice} {output}`, text on stdin | WAV file |
| `whisper.cpp` | STT | `whisper-cli -m {model} -f {input} -l {language} -ojf -of {output} -np` | whisper.cpp JSON: segments and token times |
| `moonshine` | STT | `moonshine-voice transcribe --wav-path {input} --model-arch {model} --language {language} --quiet` | One line of text per utterance |
| `sherpa-onnx` | STT | `sherpa-onnx --tokens={model}/tokens.txt --encoder=… --decoder=… --joiner=… {input}` | sherpa-onnx JSON: tokens and token times |

Notes:

- Piper 1.x ships as a Python module. Where no `piper` program exists the
  command is `python3 -m piper …`, set through `VIBIUM_SAY_COMMAND`.
- Kitten TTS has no CLI. The preset's snippet is a few lines of Python
  that call its API. The snippet is Vibium's own code.
- For `sherpa-onnx`, `{model}` is a model directory. The preset finds the
  encoder, decoder and joiner files in it.

Defaults for each preset come from [Measurements](#measurements-on-a-raspberry-pi-4).

### Choosing a provider

Each direction resolves on its own.

An explicit choice always wins: the call's `provider` option or
`--provider` flag, then `VIBIUM_SAY_PROVIDER` or
`VIBIUM_LISTEN_PROVIDER`, then a `VIBIUM_SAY_COMMAND` or
`VIBIUM_LISTEN_COMMAND` on its own.

With nothing chosen, the default goes through three levels. It takes the
first one that exists on the machine.

| Level | What | Why it comes here |
|-------|------|-------------------|
| 1. System | Speech the operating system has built in | Nothing to install, nothing to pay, nothing leaves the machine |
| 2. Local | An open-source engine the user installed, found on `PATH` | Nothing to pay, nothing leaves the machine |
| 3. Hosted | An API whose key is set: `deepgram`, then `elevenlabs`, then `openai` | Works anywhere a key is present |

With no level available the call fails with `not_configured`, which
names the ways to set one up.

What each level holds today:

| | `say` | `listen` |
|---|-------|----------|
| System, macOS | `say`, the system voice | none: macOS has no transcription command |
| System, Windows | planned: the built-in synthesizer through PowerShell | none: the built-in recognizer is not accurate enough to assert on |
| System, Linux | none | none |
| Local | `piper`, then `kitten` | `whisper.cpp`, then `moonshine`, then `sherpa-onnx` |
| Hosted | OpenAI `gpt-4o-mini-tts` | OpenAI `gpt-transcribe` |

So on a Mac `vibium say "hello"` speaks with the system voice and needs
no setup at all. On a machine with only `OPENAI_API_KEY` set, both
commands use OpenAI.

`--provider system`, a preset name, or a hosted name picks a level
directly.

A hosted provider chosen by default receives the audio `listen`
captures. That happens only when no local way to transcribe exists.
Every result names the provider that was used.

The macOS `say` line was written from its manual and has not been run:
the development machine is a Pi. A test runs it on macOS.

## Configuration

Most machines need none. See [Choosing a provider](#choosing-a-provider).

Precedence is: call option or flag, then environment variable, then the
config file, then the built-in default.

The config file is the existing `~/.config/vibium/ai.env`. It fills
variables that are empty in the process environment. Its allow-list
gains the variables below.

| Variable | Meaning |
|----------|---------|
| `VIBIUM_SAY_PROVIDER` | TTS provider or preset |
| `VIBIUM_SAY_MODEL` | TTS model |
| `VIBIUM_SAY_VOICE` | Voice |
| `VIBIUM_SAY_COMMAND` | Command line for the `command` provider or a preset |
| `VIBIUM_LISTEN_PROVIDER` | STT provider or preset |
| `VIBIUM_LISTEN_MODEL` | STT model |
| `VIBIUM_LISTEN_COMMAND` | Command line for the `command` provider or a preset |
| `OPENAI_API_KEY`, `DEEPGRAM_API_KEY`, `ELEVENLABS_API_KEY` | Credentials |

That is the whole list. Language, speed, silence, timeout and the cache
directory are per-call options only.

API keys MUST come from the environment only. There is no key flag, SDK
option or MCP argument. Keys MUST NOT appear in results, logs, the cache
or recordings. Recorded steps carry the provider and model, never a key.

The CLI reads these on every call and sends them with it, as it does
for Run and Check. A daemon started earlier does not pin them. The MCP
server and the SDK engines read their own process environment.

A station sets these variables for the Vibium process it starts. That is
how a jam's listen and say models reach Vibium.

### The `local` target

Vibium stays a static binary. It plays and captures local audio by
running a system program, the same way it runs local engines.

| OS | Play | Capture |
|----|------|---------|
| Linux | `aplay -q -` | `arecord -q -t raw -f S16_LE -r 16000 -c 1 -` |
| macOS | `afplay {input}` | `sox -q -d -t raw -r 16000 -e signed -b 16 -c 1 -` |
| Windows | PowerShell `Media.SoundPlayer` | none built in |

- Play receives a WAV on stdin, or at `{input}`.
- Capture writes raw 16 kHz mono 16-bit PCM to stdout until it is stopped.
- Both use the system's default device.
- A missing program is error `no_audio`, with the name of the package to install.

## Errors

Every failure carries a code. The CLI prints the message and exits 1;
with `--json` the envelope's `error` starts with the code. SDKs raise
`SpeechError` with `code` and `message`. MCP returns a tool error.

| Code | When | The message says |
|------|------|------------------|
| `not_configured` | No provider resolves for this direction | How to set a hosted key, and which engine to install |
| `engine_not_installed` | A preset's or command's program is not on `PATH` | The program's name and the install hint |
| `engine_failed` | The command exited non-zero, timed out, or wrote no usable output | The exit status and the end of stderr |
| `no_api_key` | A hosted provider's key variable is empty | The variable's name |
| `provider_error` | The hosted API answered with an error | The HTTP status and the provider's message |
| `no_audio` | The target cannot play or capture: the session has no audio, no page has the microphone open, no local program, an unreadable file, or no samples arrived within 3 seconds | Which of these it was, and `--target local` when a session was the default |
| `no_speech` | Audio arrived but no speech started within `timeout` | How long it listened and the peak level it saw |
| `invalid_target` | `session` with no session open, or a path that is not `.wav` | What targets are available |
| `busy` | A `say` or `listen` is already running on this target | Which call holds it |
| `text_too_long` | The provider refused the text for its length | The provider's limit |

`no_audio` and `no_speech` are different failures. `no_audio` means the
pipe is broken. `no_speech` means the pipe worked and the other side
stayed quiet. `--allow-silence` turns `no_speech` into a result.

Errors do not write to the cache. A failed `say` or `listen` is still
recorded as a step, with its error.

## What this asks of Roadie

Roadie stays a raw audio pipe. It gains no speech code.

Roadie also carries audio over two WebSockets: `/audio` for the device's
sound and `/api/mic` for voice into the device. The commands below add
the same pipe to its BiDi socket. They are implemented on Roadie's
`feat/audio-bidi` branch and documented in its `API.md`.

Vibium uses the BiDi extension because it rides the session socket.
It needs no second connection and no second address, and a station
relays it without change.

### `roadie:audio`

A session's capabilities announce it:

```json
"roadie:audio": {"play": true, "capture": true}
```

**`roadie:audio.play`** plays a clip into the device's microphone.

```json
{"method": "roadie:audio.play", "params": {
  "format": "s16le", "sampleRate": 16000, "channels": 1,
  "data": "<base64 PCM, at most 60 seconds>"}}
```

Roadie queues the whole clip and plays it at its own pace, without gaps.
It answers when playback has ended:

```json
{"result": {"startTime": 81234567.25, "endTime": 81236877.25, "samples": 36960}}
```

`startTime` and `endTime` are when the first and last sample left Roadie
for the device, in milliseconds on Roadie's monotonic clock.

**`roadie:audio.capture.start`** starts sending what the device plays.

```json
{"method": "roadie:audio.capture.start", "params": {
  "format": "s16le", "sampleRate": 16000, "channels": 1, "chunkMs": 100}}
```

```json
{"result": {"capture": "c1", "sampleRate": 16000, "channels": 1, "startTime": 81236880.0}}
```

Roadie then emits one event per chunk:

```json
{"method": "roadie:audio.chunk", "params": {
  "capture": "c1", "sample": 1600, "time": 81236980.0, "data": "<base64 PCM>"}}
```

- `sample` counts the samples sent before this chunk.
- `time` is when the chunk's first sample was captured, on Roadie's monotonic clock.
- When Roadie drops audio, `time` jumps and `sample` does not. Vibium
  places chunks by `time` and fills the gap with silence.

**`roadie:audio.capture.stop`** takes `{"capture": "c1"}` and ends the events.

**`roadie:audio.clock`** takes no params and answers
`{"now": 81236990.5}` on the same clock.

Through a station the whole BiDi socket needs the `bidi` scope, which a
watch-level token does not have. The station relays the socket byte for
byte, so these commands need nothing more from it.

Roadie's existing `/audio` and `/api/mic` endpoints are unchanged. They
remain the raw pipe for viewers and other tools.

## Browser sessions

A browser session carries audio through the page itself. Vibium adds a
preload script to every page of the session, as it already does for the
clock and for WebSocket monitoring. No browser flag, extension or audio
device is involved. So it works the same in a visible browser, a
headless one and a remote one.

This path is built and tested in Chrome and Firefox against a page that
records its microphone and plays the recording back, and against
soundcheck.vibium.dev by hand. Media elements and frames are written and
not yet exercised.

### Into the page

The script gives the page a virtual microphone.

- A page that asks `getUserMedia` for audio gets a track that Vibium
  feeds. The machine's real microphone is not opened.
- A request for video passes to the browser unchanged.
- `enumerateDevices` lists one audio input, named `Vibium microphone`.
- Vibium grants the microphone permission for the session. No prompt appears.
- `say` sends the clip to the page. The script plays it into the track.
  It reports when the first and last sample played, on the page's clock.
- Between clips the track carries silence.

`say` reaches every frame of the page that has the microphone open. If
none has, `say` fails with `no_audio`. The message says no page is
listening.

The virtual microphone is on by default, so a first `say` works without
setup. An opt-out, `vibium start --audio=false` or `audio: false` in a
client, is specified and not built yet.

### Out of the page

The script taps what the page plays.

- Web Audio: everything a page connects to an `AudioContext` destination.
- Media elements: `<audio>` and `<video>`, including one that plays a WebRTC stream.

The script mixes these, converts them to 16 kHz mono, and sends chunks
to Vibium over a BiDi channel. Each chunk carries the page-clock time of
its first sample. The page still plays through the machine's speakers.

The tap cannot hear everything.

- `speechSynthesis`. The browser hands the text to the operating
  system. No audio returns to the page.
- Media from another origin served without CORS headers. The browser
  gives a script silence for it.

When `listen` hears nothing and the page used `speechSynthesis`, the
`no_speech` message says so.

The hooks are in place from page load. They copy audio only after the
first `say` or `listen` of the session, or when a recording asks for
audio. A page that never uses speech pays nothing.

### Pages and frames

- `page.say` and `page.listen` act on that page and its frames.
- `Browser.say`, the CLI and MCP act on the current page unless one is named.
- Audio from several frames of one page is mixed.

## Clock sync

Several clocks are involved: Vibium's, the target's, and the audio
sample clock. This section uses Roadie as the target. A browser follows
the same steps.

**Latency never crosses clocks.** `play` returns its end time and each
chunk carries its capture time, both on Roadie's clock. Reply latency is
their difference. It is exact to the accuracy of Roadie's own
timestamps, whatever the network does.

**Placing audio on Vibium's timeline** needs the offset between the two clocks.

1. Vibium sends `roadie:audio.clock` and notes its own monotonic time
   at send (`t0`) and at receipt (`t1`). Roadie answers `now`.
2. One sample gives `offset = now − (t0 + t1) / 2` and `rtt = t1 − t0`.
3. Vibium takes five samples and keeps the one with the smallest `rtt`.
4. It repeats this when capture starts and every 10 seconds after.
5. A Roadie time `r` falls at `r − offset` on Vibium's clock.

The error of the offset is at most `rtt / 2`. That is under 2 ms on a
LAN and typically 10 to 50 ms through a station. The recording stores
every offset sample, so a reader sees the bound.

**The sample clock** drifts against Roadie's system clock. Chunk
timestamps anchor it: the recording maps sample index to time through
the chunks' own times, not through the nominal sample rate.

**Fixed path delays.** Roadie's timestamps are when samples cross its
own audio interface. The phone hears the question slightly later and
its answer is captured slightly later. Both are a few tens of
milliseconds and constant for a given rig. Roadie SHOULD report them in
`roadie:audio` as `playDelayMs` and `captureDelayMs`. Vibium adds them
when present.

**In a browser** the target's clock is the page's:
`performance.timeOrigin + performance.now()`. The script maps audio time
to it with `AudioContext.getOutputTimestamp()`. A script call takes the
place of `roadie:audio.clock`. Latency is computed on the page's clock alone.

For `local`, both directions use Vibium's own clock, stamped when bytes
are written and read. That is accurate to the audio buffers, about 50 ms.

For a WAV file, time is audio time. No clock is involved.

## Recordings

When a recording is running, `say` and `listen` are recorded as steps
and their audio goes into record.zip.

### What is recorded

- **Audio sent.** The exact samples played for each `say`, with its
  text, provider, model and voice.
- **Audio received.** One continuous track per target, with
  speech-onset markers. It runs to the end of the recording.
- **Transcripts.** One per `listen`, with segments, word timings when
  available, onset and reply latency.

Everything is on the recording timeline: relative milliseconds since the
recording started, as in `trace.trace`.

### Video

A recording on a station or a Roadie session also holds the device's
screen as `video/<context>.mp4`. Vibium takes it from the same stream a
viewer watches. The audio above and the video share the recording
timeline, so a player can run them together.

The file also has an audio track: every clip sent and everything heard,
mixed into one channel as AAC. A person can play or post the file by
itself and hear the recording. The track is absent when nothing sounded
while the video ran.
[Recording format](../reference/recording-format.md#video-of-a-device)
describes the entry.

Measured on the phone, with `say` at 27.9 s and the page's playback heard
at 38.2 s: the frames at those times show the page's level meter moving
and its "Playing back" line.

### The `audio` option

`recording.start` gains `audio`, shaped like `video`.

| Value | Behavior |
|-------|----------|
| omitted | Audio is recorded from the first `say` or `listen` of the recording to its end. A recording that never uses speech has no audio. |
| `true` | The session's audio is recorded from the start of the recording. The start fails if the session cannot be captured. |
| `false` | Steps are recorded without any audio files. |

Use `audio: true` when the other side may speak before the first
`listen`.

CLI: `vibium record start --audio` and `--audio=false`.

The local microphone is recorded only while a `listen` on `local` is
running. A recording never listens to the room on its own.

### Zip layout

```
record.zip
├── trace.trace
├── trace.network
├── resources/…
├── audio/index.json        # manifest
├── audio/in-session.wav    # everything heard from the session
├── audio/out-0001.wav      # one file per distinct clip sent
├── video/index.json        # video manifest, on a station or a Roadie session
└── video/screen.mp4        # the device's screen, with the sound mixed in
```

- Audio is WAV: PCM, 16-bit, mono, 16 kHz.
- A received track is one file per target: `in-session.wav`, `in-local.wav`.
- Sent clips are numbered in order of first use. A clip played twice is stored once.
- Audio lives under `audio/`, never under `resources/`. Existing readers
  load every file in `resources/` as an image.

The additions are additive. No existing entry changes.

### Trace events

`say` and `listen` are ordinary actions: a `before`/`after` pair with
the shared `call@N` counter. No new event type is added to
`trace.trace`.

```json
{"type":"before","callId":"call@7","startTime":3010,"class":"Speech","method":"vibium:speech.say","title":"Speech.say","params":{"text":"What's the weather like in Chicago tomorrow?","target":"session","provider":"openai","model":"gpt-4o-mini-tts","voice":"alloy"}}
{"type":"after","callId":"call@7","endTime":5335,"result":{"clip":"out-0001","durationMs":2310,"cached":true,"playStartMs":3012.5,"playEndMs":5322.5}}
{"type":"before","callId":"call@8","startTime":5340,"class":"Speech","method":"vibium:speech.listen","title":"Speech.listen","params":{"target":"session","provider":"openai","model":"gpt-transcribe","silence":1000,"timeout":10000,"duration":30000}}
{"type":"after","callId":"call@8","endTime":9480,"result":{"text":"It's currently 72 degrees and sunny in Chicago.","speech":true,"onsetMs":6014,"speechEndMs":8105,"latencyMs":691.5,"latencyFrom":"call@7","stoppedBy":"silence","track":"in-session","audioStartMs":5322.5,"audioEndMs":9105}}
```

- `result` and `error` on `after` follow
  [recording-error-result-spec.md](recording-error-result-spec.md). This
  spec needs them for these two methods at least.
- `params` are what the caller passed. The `result` names the provider,
  model and voice that were used.
- Times inside `result` are on the recording timeline, not relative to the capture.
- A failed step has `error: {"message": …}` and no `result`. The message
  starts with the error's code.
- `params` never holds a key, a base URL or a command line.

### The manifest

`audio/index.json` describes the files and holds what does not fit a step.

```json
{
  "version": 1,
  "tracks": [{
    "id": "in-session",
    "target": "session",
    "file": "audio/in-session.wav",
    "sampleRate": 16000, "channels": 1, "encoding": "pcm_s16le",
    "startMs": 412.0,
    "durationMs": 61230,
    "clock": {
      "source": "roadie",
      "sync": [{"atMs": 400, "offsetMs": 81231156.2, "rttMs": 1.8}],
      "anchors": [{"sample": 0, "timeMs": 412.0}, {"sample": 160000, "timeMs": 10412.6}],
      "playDelayMs": 0, "captureDelayMs": 0
    },
    "gaps": [{"sample": 48000, "samples": 1600}],
    "speech": [{"startMs": 6014, "endMs": 8105}]
  }],
  "clips": [{
    "id": "out-0001",
    "target": "session",
    "file": "audio/out-0001.wav",
    "sha256": "…",
    "sampleRate": 16000, "channels": 1, "encoding": "pcm_s16le",
    "durationMs": 2310,
    "text": "What's the weather like in Chicago tomorrow?",
    "provider": "openai", "model": "gpt-4o-mini-tts", "voice": "alloy",
    "plays": [{"callId": "call@7", "startMs": 3012.5, "endMs": 5322.5}]
  }],
  "transcripts": [{
    "callId": "call@8",
    "track": "in-session",
    "startMs": 5322.5, "endMs": 9105,
    "onsetMs": 6014, "speechEndMs": 8105,
    "latencyMs": 691.5, "latencyFrom": "call@7",
    "text": "It's currently 72 degrees and sunny in Chicago.",
    "provider": "openai", "model": "gpt-transcribe",
    "segments": [{"startMs": 6014, "endMs": 8105, "text": "It's currently 72 degrees and sunny in Chicago."}],
    "words": []
  }]
}
```

| Field | Meaning |
|-------|---------|
| `tracks[].startMs` | Timeline time of the track's first sample |
| `tracks[].clock.sync` | Every clock-offset sample taken, with its round trip. `source` is `roadie`, `page`, `local` or `file` |
| `tracks[].clock.anchors` | Sample index to timeline time, one anchor every 10 seconds. Readers interpolate between anchors |
| `tracks[].gaps` | Runs of samples Vibium filled with silence because the source dropped them |
| `tracks[].speech` | Speech-onset markers: every stretch of speech the detector found on the track, whether or not a `listen` was running |
| `clips[].plays` | Each time the clip was played, when it started and ended at the target, and for which step |
| `transcripts[]` | One per `listen`, with the same fields as the step's `result` plus segments and words |

Speech markers cover the whole track. Transcripts exist only where a
`listen` ran. Vibium does not transcribe audio nobody asked about.

### Not built yet

- Audio in chunk zips, and in the spool directory. Audio goes into the
  recording's final zip only.
- The `audio` option on `recording.start`. Audio is recorded from the
  first `say` or `listen`, as if the option were omitted.
- The local microphone's audio in a track. A `listen` on `local` records
  its transcript and no audio.
- A new clock reading every 10 seconds. One reading is taken when the
  track starts.

### Chunks, spooling, redaction

- **Chunks.** Each chunk zip carries its own slice of every track, the
  clips played in it, and a manifest on that chunk's timeline.
- **Spooling.** With a path declared, tracks append to
  `<path>.parts/audio/` as they arrive. `vibium record recover` rebuilds
  the WAV headers.
- **Size.** A track is 1.9 MB per minute before zip compression. Silence compresses well.
- **Redaction.** Audio is not redacted. A spoken secret stays in the
  WAV. `audio: false` keeps audio out of the zip.
- **Upload.** `recording.stop({ upload })` sends the zip as it is, audio included.

## The Record Player

The player at `VibiumDev/record-player` plays these recordings.

A player that does not know `audio/` or `video/` still opens them. It
reads the trace files and `resources/`, and shows `say` and `listen` as
steps.

### Audio

- The player reads `audio/index.json`. It ignores audio when the file is
  absent or its `version` is not `1`.
- While the recording plays, each clip sounds at the time it was played.
  The track sounds at the time it was heard.
- The caption under the picture shows the words:
  - `Say “…”` for a `say` step;
  - `Listening…` and then `Heard “…”` for a `listen` step.
- A Sound button turns the audio off.

### Video

- The player reads `video/index.json`. It ignores video when the file is
  absent or its `version` is not `1`.
- When a recording has a video, the player shows it in place of the
  screenshots.
- The video follows the playhead. It plays, pauses, seeks and changes
  speed with it.
- The player mutes the video. Its sound comes from `audio/`.
- `offsetMs` places the video. The player's timeline starts at the first
  event. The video's first frame is at `offsetMs` minus that event's time.
- The player shows the screenshots in three cases:
  - the recording has no video;
  - the browser cannot play the video;
  - the playhead is more than a second outside the video.
- The video has no sound. Sound comes from `audio/`.

### Not built

- A play button on each `say` and `listen` step.
- Marks on the timeline for each play and each stretch of speech.

## Measurements on a Raspberry Pi 4

**Status: partial.** The run was stopped to free the Pi for
implementation. Everything in the tables was measured. The list at the
end of this section says what is still to run.

### Conditions

- Raspberry Pi 4 Model B Rev 1.4, 8 GB, 64-bit Debian 13, kernel 6.18, four cores at 1.8 GHz.
- The Pi was busy. Roadie was capturing a phone's screen, and the load average was 3 to 5 before each run.
- The Pi was hot: 81 to 85 °C, with the firmware's temperature limit active.

These numbers are a floor for a hot, busy Pi. A cool, idle Pi 4 will be
faster. That was not measured. The ranking between engines holds either way.

### Method

- Each engine ran as the `command` provider runs it: one process per
  utterance, four threads, model files already in the page cache.
- Times are wall time. They include process start and model load.
- TTS: three phrases of 44, 79 and 157 characters, three runs each, median.
- STT: 21 clips of 2 to 17 seconds. Three are human speech (the JFK
  sample and two LibriSpeech utterances). Eighteen are assistant-style
  replies with numbers, names and dates, spoken by three TTS voices.
- Errors are word errors after Whisper's English text normalizer, with
  times and ordinals made equal ("3:30 p.m." and "three thirty PM").
- Hosted APIs were called from the same Pi, one request per clip.

### Text to speech

| Engine | Voice or model | Short phrase | Long phrase | Model load | Synthesis time per second of audio |
|--------|----------------|--------------|-------------|------------|-------------------------------------|
| Piper | `lessac-low` (16 kHz) | 13.7 s for 2.3 s | 22.2 s for 8.8 s | 8.3 s | 1.5 s |
| Piper | `lessac-medium` (22 kHz) | 13.8 s for 2.1 s | 24.8 s for 8.3 s | 8.2 s | 2.0 s |
| Piper | `amy-low` (16 kHz) | 15.0 s for 2.8 s | 21.6 s for 9.6 s | | |
| Piper | `amy-medium` (22 kHz) | 12.3 s for 2.7 s | 23.6 s for 9.6 s | 6.0 s | 1.9 s |
| Piper | `ryan-medium` (22 kHz) | 13.1 s for 1.7 s | 25.5 s for 8.1 s | | |
| Kitten TTS | nano, 15M (24 kHz) | 16.5 s for 3.5 s | 40.1 s for 10.6 s | 4.0 s | 3.4 s |
| Kitten TTS | nano, int8 | 16.8 s for 3.5 s | 40.9 s for 10.6 s | 5.6 s | 3.0 s |
| Kitten TTS | micro, 40M | 19.0 s for 3.4 s | 36.8 s for 10.1 s | 5.6 s | 3.4 s |
| Kitten TTS | mini, 80M | 29.7 s for 3.1 s | 69.1 s for 9.2 s | 5.3 s | 7.2 s |
| OpenAI | `gpt-realtime-2.1-mini`, Realtime API | 0.7 to 0.8 s for 2.1 s | 1.5 to 2.2 s for 9.5 s | | first audio in 0.5 to 1.1 s |
| OpenAI | `gpt-4o-mini-tts`, retiring | 1.0 to 1.6 s for 2.9 s | 1.8 s for 9.3 s | | first byte in 0.4 to 0.6 s |
| OpenAI | `tts-1`, retiring | 1.0 s for 2.7 s | 1.2 s for 9.8 s | | first byte in 0.8 to 0.9 s |
| OpenAI | `tts-1-hd`, retiring | 1.4 to 2.5 s for 2.7 s | 2.1 to 3.2 s for 9.8 s | | first byte in 1.4 to 2.4 s |

What this shows:

- **No local engine is real time here.** Piper needs 12 to 15 seconds
  for a short phrase. More than half of that is loading the model.
- **Piper is faster than Kitten TTS at every size.** Kitten's smallest
  model synthesizes at half Piper's speed. Its largest is four times slower.
- **Low and medium Piper voices cost the same from a cold start.** Low
  saves about a quarter of the synthesis time and nothing of the load.
- **The cache carries local TTS.** A phrase costs its render once.
  Every later `say` of it plays at once.
- **All three voices are understood.** OpenAI's `gpt-transcribe` made no
  word errors on 219 words spoken by Piper `lessac`, Piper `ryan` and
  Kitten `Bella`.
- **Hosted TTS gives different audio every time.** The same text sent
  twice comes back as two different recordings. The words match. The
  timing, the intonation and the padding do not. Without the cache, each
  test run would play the device slightly different audio. With it,
  every run plays the first render.
- **No hosted model can be trusted to say the text.** Four short
  phrases were rendered 22 times with each OpenAI model, and every clip
  was transcribed. `gpt-4o-mini-tts` returned one empty clip. The
  realtime model dropped words in six: "Come here, Watson" came back as
  "Come here" four times out of eight. Its own transcript claimed the
  full text every time. So `say` checks the audio.
- **The check works.** With it, 10 of 10 fresh renders through the
  speech endpoint and 8 of 8 through the realtime model said the text.
- **A final period helps a realtime model.** Text without one lost its
  last word in 4 of 8 renders. With one, in none of 6. The adapter adds it.
- **Providers pad the audio.** The same phrase came back with 0.1 to
  2.9 seconds of trailing silence. The speech itself kept its length.
  So Vibium trims.
- sherpa-onnx 2.0.1's prebuilt binary could not speak Piper voices. Its
  build has no phonemizer.

Time to first audio equals the whole render time. Vibium plays a clip
after it is complete.

### Speech to text

| Engine | Model | Word errors, replies | Word errors, human | Median time for a reply clip | Time per second of audio |
|--------|-------|----------------------|--------------------|------------------------------|--------------------------|
| whisper.cpp | `tiny.en` | 5 of 219 | 2 of 88 | 13.5 s | 2.2 s |
| whisper.cpp | `tiny.en`, q5_1 | 5 of 219 | 2 of 88 | 10.4 s | 1.7 s |
| whisper.cpp | `base.en` | 3 of 219 | 2 of 88 | 33.2 s | 5.3 s |
| whisper.cpp | `base.en`, q5_1 | 3 of 219 | 2 of 88 | 21.5 s | 3.4 s |
| OpenAI | `gpt-transcribe` | 0 of 219 | 2 of 88 | 0.8 s | 0.14 s |
| OpenAI | `gpt-4o-mini-transcribe` | 1 of 219 | 2 of 88 | 0.7 s | 0.12 s |
| OpenAI | `gpt-4o-transcribe` | 1 of 219 | 2 of 88 | 0.7 s | 0.11 s |
| OpenAI | `whisper-1`, with word timings | 4 of 219 | 2 of 88 | 1.5 s | 0.22 s |

What this shows:

- **Every engine gets the words.** Every model misses the same two
  human words. The worst result is 5 errors in 219 words.
- **Quantized models are faster and no less accurate.** q5_1 cuts
  whisper.cpp's time by a quarter to a third.
- **`tiny.en` is the only local model near usable here.** It still
  takes 10 seconds to transcribe a 6-second reply on this Pi.
- **Hosted transcription is fifteen times faster** from the same machine.
- **`gpt-transcribe` returns no timestamps.** It refuses the timestamped
  format with HTTP 400. Vibium's own detector supplies onset and end.
- The OpenAI Whisper weights are what whisper.cpp runs. OpenAI's hosted
  `whisper-1` and the local `base.en` land within two words of each other.

Transcription time does not enter reply latency. Latency comes from the
detector, on the target's clock.

### Voice activity detection

Not measured yet. The built-in detector is to be compared with Silero
VAD on the same 21 clips, for onset and end error.

### Chosen defaults

| | Default | Why |
|---|---------|-----|
| System `say`, macOS | `say`, the system voice | Built in. Not measured: no Mac was at hand. |
| Hosted `say` | OpenAI `gpt-4o-mini-tts`, voice `alloy` | Measured. One bad clip in 22, against six for the realtime model. First byte in half a second. Uses the key most users already have. Retired on 2027-01-06. |
| Hosted `listen` | OpenAI `gpt-transcribe` | Measured. No word errors on replies. Under a second per reply. |
| Local `say` | Piper, a medium voice | Fastest local engine measured. The default voice is `en_US-ljspeech-medium`, for its public-domain dataset. That voice is still to be measured. |
| Local `listen` | whisper.cpp, `tiny.en` q5_1 | Fastest local model measured, with 5 errors in 219 words. |

The local defaults are provisional until the remaining engines are measured.

### Still to measure

- Moonshine, in its tiny, base and streaming sizes.
- sherpa-onnx streaming Zipformer with Silero VAD, including time to first word.
- faster-whisper, the reference `openai-whisper` package, and whisper.cpp `small.en`.
- A warm whisper.cpp server, to price the model load.
- Piper voices with clean dataset terms: `ljspeech`, `joe`, `norman`, `kristin`, `libritts_r`.
- The built-in detector against Silero VAD.
- A cool, idle Pi 4.
- Deepgram and ElevenLabs.

## License notes

This is an engineering reading, not legal advice.

Vibium's binary contains none of the components below. The `command`
provider starts a program the user installed and exchanges files with
it. A preset is a command line.

| Component | Role | License | Note |
|-----------|------|---------|------|
| Piper 1.x (`OHF-Voice/piper1-gpl`) | TTS engine | GPL-3.0 | The older MIT-licensed `rhasspy/piper` is archived. |
| espeak-ng | Phonemizer inside Piper and Kitten TTS | GPL-3.0-or-later | The reason engines stay outside the core. |
| Kitten TTS (`KittenML/KittenTTS`) | TTS engine and models | Apache-2.0 | Needs `phonemizer` (GPL-3.0) and espeak-ng at run time. |
| whisper.cpp | STT engine | MIT | |
| Whisper weights | STT models | MIT | |
| `openai-whisper` | Reference STT engine | MIT | Needs PyTorch. |
| faster-whisper, CTranslate2 | STT engine | MIT | |
| Moonshine | STT engine and models | MIT for code and English models | Older non-English models carry a non-commercial license. |
| sherpa-onnx | STT engine | Apache-2.0 | |
| Streaming Zipformer English models | STT models | Apache-2.0 | |
| Silero VAD | Detector model | MIT | Not used by default. |
| ONNX Runtime | Runtime under most of the above | MIT | |

Piper voices follow the terms of the recordings they were trained on.
The voice repository's own license is MIT.

| Voice | Dataset terms | Usable as the default |
|-------|---------------|-----------------------|
| `ljspeech` | Public domain | Yes |
| `joe` | CC0 | Yes |
| `norman`, `john`, `kristin`, `bryce` | LibriVox recordings, public domain | Yes |
| `libritts_r` | CC BY 4.0 | Yes, with attribution |
| `amy`, `danny`, `kusal` | From Mycroft's Mimic 3 voices, CC BY-SA 4.0 | With attribution and share-alike |
| `lessac` | Blizzard Challenge 2013 license, research use on application | No |
| `ryan`, `hfc_female` | CC BY-NC-SA 4.0 | No, non-commercial |
| `kathleen` | Not checked | Unknown |

The measured voices were `lessac`, `amy` and `ryan`. None of them is a
good default on these terms. That is why the default is `ljspeech`.

## Hosted providers compared

OpenAI was measured for this spec, from the Pi, on 2026-11-09. See
[Measurements](#measurements-on-a-raspberry-pi-4). Deepgram and
ElevenLabs were not measured. Their rows are from their documentation,
read the same day.

| | OpenAI | Deepgram | ElevenLabs |
|---|--------|----------|------------|
| TTS model | `gpt-4o-mini-tts`; `gpt-realtime-2.1-mini` after it | `aura-2` voices | `eleven_flash_v2_5` |
| TTS output | WAV or PCM, 24 kHz | WAV, 8 to 48 kHz | WAV or raw PCM, 16 kHz and up |
| TTS price | by token; under a cent per phrase | $0.030 per 1,000 characters | about $0.05 per 1,000 characters |
| TTS status | default retires 2027-01-06; the realtime model is current | current | current |
| STT model | `gpt-transcribe` | `nova-3` | `scribe_v2` |
| STT price | $0.0045 per minute | $0.0043 per minute | $0.22 per hour ($0.0037 per minute) |
| Word timings | no | yes | yes |
| Measured here | yes | no | no |
| Key | `OPENAI_API_KEY` | `DEEPGRAM_API_KEY` | `ELEVENLABS_API_KEY` |

Notes:

- **OpenAI's speech endpoint is being retired.** `tts-1`, `tts-1-hd`
  and `gpt-4o-mini-tts` on `/v1/audio/speech` shut down on 2027-01-06.
  OpenAI's replacement is a realtime model on its Realtime API. That
  model is a conversational model, not a speech engine, and it dropped
  words in 27% of test renders. Vibium uses the speech endpoint while it
  exists and has the Realtime path ready behind the same audio check.
- `gpt-transcribe` is OpenAI's current transcription model. The older
  `whisper-1` and `gpt-4o-transcribe` models shut down on 2027-02-26.
- At these prices a 5-second reply costs about $0.0004 to transcribe. A
  cached phrase costs nothing to say again.

### Recommended hosted defaults

**OpenAI for both directions, today.**

- It is measured and it works: no word errors on replies, and under a
  second per reply.
- One key covers `say` and `listen`. Most Vibium users have it already,
  for Run and Check.
- Nothing else needs configuring.

Deepgram and ElevenLabs are the alternatives. A user picks one with
`--provider`, or by setting its key. Both return word timings for `listen`.

## Examples

### 1. Ask a phone a question and time the answer

A phone sits on a Roadie. The test speaks into the phone's microphone,
hears the phone's reply, and checks the words and the latency.

```js
const assert = require('node:assert/strict');
const { browser } = require('vibium');

const bro = await browser.start('ws://roadie.local:8080/session');
const page = await bro.page();
await page.context.recording.start({ path: 'weather.zip' });

// Open the assistant the way your app needs: a wake phrase, or a tap on its microphone button.
await page.say("Hey Google, what's the weather in Chicago tomorrow?");
const reply = await page.listen({ timeout: 8000 });

assert.match(reply.text, /chicago/i);
assert.match(reply.text, /degrees/i);
assert.ok(reply.latencyMs < 2500, `the reply took ${reply.latencyMs} ms to start`);

await page.context.recording.stop();
await bro.stop();
```

What happens:

1. `say` renders the phrase, or loads it from the cache. Roadie plays it into the phone.
2. Capture starts the moment playback ends.
3. `listen` waits for the phone to speak, then for one second of silence.
4. The reply goes to the STT provider. The transcript comes back.
5. `latencyMs` is the gap between the end of the question and the first
   sound of the answer, on Roadie's clock.
6. `weather.zip` holds both steps, the clip sent, and what the phone played from the question onward.

The same test from a shell:

```bash
vibium start ws://roadie.local:8080/session
vibium record start -o weather.zip

vibium say "Hey Google, what's the weather in Chicago tomorrow?"
reply=$(vibium listen --timeout 8000 --json)

echo "$reply" | jq -e '.result.text | test("chicago"; "i")'
echo "$reply" | jq -e '.result.latencyMs < 2500'

vibium record stop
vibium stop
```

Assert on keywords, not whole sentences. Engines differ on punctuation,
and on digits versus words: "72" or "seventy-two".

### 2. An offline round trip, for CI

No device, no sound card, no network. `say` writes a WAV file and
`listen` reads it back. This proves the speech setup works before a
device is involved.

```python
from vibium import browser

said = browser.say("Set a timer for ten minutes",
                   target="timer.wav", cache_dir="tests/speech-cache")
heard = browser.listen(target="timer.wav", silence=0)

print(said.cached, said.duration_ms, heard.text)
assert "timer" in heard.text.lower()
assert "ten" in heard.text.lower() or "10" in heard.text
```

```bash
vibium say "Set a timer for ten minutes" --target timer.wav --cache-dir tests/speech-cache
# Wrote timer.wav (2.1 s, openai gpt-4o-mini-tts alloy, cached)

vibium listen --target timer.wav --silence 0
# Set a timer for ten minutes.
```

With `tests/speech-cache` committed, CI needs no TTS engine or key for `say`. Every run
plays the same bytes into the device.

A recorded reply works the same way as a fixture:

```bash
vibium listen --target fixtures/weather-reply.wav --json
# onsetMs and segments count from the start of the file. latencyMs is absent.
```

## Future work

These are out of scope here. The design leaves room for each.

- **Plugins.** The `command` contract becomes a plugin protocol.
  Plugin install fetches an engine and its models. Presets become plugins.
- **Streaming partial transcripts.** `listen` returns once, at the end.
  A streaming form would emit words as they are recognized. The
  provider interface would gain a streaming method.
- **Continuous audio playback in the Record Player.** Playing the
  received track in step with the screenshots, with scrubbing.
- **Audio compression.** Opus or FLAC inside record.zip. The manifest's
  `encoding` field is where a reader learns of it.
- **Barge-in testing.** Speaking while the device is speaking, and
  measuring how fast it stops. This needs overlapping `say` and `listen`
  and echo handling.

Smaller items this spec deferred:

- Warm local engines. A long-lived engine process, or an OpenAI-compatible local server, would remove the model load from every call. On a Pi 4 that load is most of the wait.
- Hearing `speechSynthesis` and cross-origin media in a browser. That needs capture below the page.
- Splitting long text across several provider calls.
- A model-based detector, such as Silero VAD, as an option for noisy rooms.

## Open questions

Each has a proposed answer. The spec is written as if that answer holds.

1. **Virtual microphone by default.** Every browser session would give
   pages a virtual microphone in place of the real one. That makes a
   first `say` work with no setup. It also changes `getUserMedia` for
   every existing user, including tests that use a browser's own
   fake-device flags. `audio: false` opts out. Is default-on right, or
   should it be opt-in at `vibium start`?
2. **Order of delivery.** The browser path is the largest piece and is
   not prototyped. Roadie, `local` and files could ship first, with the
   browser path next. Or all four ship together.
3. **Tool names.** `vibium_say` and `vibium_listen` follow `vibium_run`
   and `vibium_check`. They work without a browser, so `browser_say`
   would mislead. Is that the right family?
4. **The Roadie contract.** `roadie:audio` is new. Roadie has no audio
   commands on its BiDi socket today. Does Roadie take the contract as
   written? Two points need a decision there: base64 audio inside JSON
   or binary frames, and the 60-second limit on a clip.
5. **Path delays.** Should Roadie measure `playDelayMs` and
   `captureDelayMs` for each rig, with a loopback calibration? Without
   them latency carries a constant error of a few tens of milliseconds.
6. **Audio in recordings by default.** The spec records audio from the
   first `say` or `listen` onward, and from the start only with
   `audio: true`. The alternative is from the start whenever the
   session has audio. That catches a phone that speaks first. It also
   adds 1.9 MB per minute to recordings that never use speech.
7. **Local audio through system programs.** `aplay`, `arecord`,
   `afplay` and `sox` keep Vibium free of cgo. macOS capture then needs
   `sox`, and Windows has no capture. Is that enough for a first version?
8. **OpenAI `say` after 2027-01-06.** The default model is retired
   then. The realtime model works behind the audio check, at about one
   retry in four renders. The other choice is Deepgram or ElevenLabs as
   the hosted default for `say`. Which one, and by when?
9. **Measuring Deepgram and ElevenLabs.** Both are specified from
   documentation alone. Should they be measured with keys of yours
   before their adapters are written?
10. **Variable names for stations.** A jam's listen and say models
    reach Vibium as `VIBIUM_LISTEN_*` and `VIBIUM_SAY_*`. The station
    spec says it uses the names Vibium gives them. Are these the names?
11. **`after.result` in recordings.** `say` and `listen` need `result`
    and `error` on `after` events. That spec is still a draft. May
    these two methods land the fields first?
12. **Readiness check.** `vibium ready speech` would report the chosen
    providers, find the engines, and do one round trip through a WAV
    file. It is small and it helps a first-time user. Is it in the
    first version?
13. **System speech on Windows.** Its built-in synthesizer can be run
    through PowerShell. The command line is not written yet because it
    could not be tried. Is Windows `say` wanted in the first version?
14. **System transcription.** macOS can transcribe on the device, but
    only through an API, not a command. Using it means Vibium ships a
    small native helper for macOS. Is that wanted, or do local and
    hosted `listen` cover it?
