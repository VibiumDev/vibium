# Spec: Audio Tracks from a Roadie Source

Resolves #797.

## Context

No browser delivers audio. The BiDi spec defines an audio track on
`browsingContext.startScreencast`, but Firefox rejects it and Chromium
has no screencast at all, so the video spec reserves `audio`. Roadie
sessions have both audio directions as plain WebSocket PCM, independent
of the engine: `GET /audio` streams the target's speaker output and
`GET /audio-in/monitor` mirrors the mixed mic input (VibiumDev/roadie#30).
This spec binds those streams to record.zip.

## API

```ts
await recording.start({ audio: { source: 'http://roadie.local:8000' } });
```

- `audio?: { source: string }` — base URL of a roadie host. Audio is
  always explicit; there is no auto mode. `start` MUST fail when the
  source cannot deliver (unreachable, 503, bad params).
- The binary subscribes to `<source>/audio` ("output") and
  `<source>/audio-in/monitor` ("input") and spools PCM to temp files.
- A stream lost mid-recording MUST NOT fail `stop()`: the zip delivers,
  the track is absent or partial, and its index entry records the error.

## Zip layout

Additive to the trace format, like video; trace tooling ignores it.

```
record.zip
├── audio/output.wav
├── audio/input.wav
└── audio/index.json
```

```json
{ "version": 1, "audio": [ {
  "file": "audio/output.wav",
  "direction": "output",
  "startedAt": 1762700000123,
  "offsetMs": 412,
  "sampleRate": 48000,
  "channels": 2,
  "mimeType": "audio/wav",
  "durationMs": 14000
} ] }
```

`offsetMs` aligns the track to the recording's t0, same as
`video/index.json`. A track with no frames (silence on the input
monitor) has no `file` entry.

## Format

WAV over the raw stream: format 3 (IEEE float) for roadie's float32
output, format 1 (integer PCM) for the s16 mic monitor. Capture never
invokes an encoder; compression is an export concern, like MP4.

## Stop result

`audio: [{direction, durationMs, error?}]` next to `videos`. The CLI
sentence gains the audio length: `Saved record.zip (23 steps, 14s
video, 14s audio)`.

## Chunks

Chunk zips carry no audio; the tracks belong to the session zip.

## Surfaces

The logic lives once in the binary.

- JS: `audio?: { source: string }`
- Python: `audio={"source": ...}`
- Java: `new RecordingOptions().audioSource(...)`
- CLI: `vibium record start --audio-source <url>`
- MCP: `audio_source` (string)

## Future

An engine screencast audio track, if one ever ships, slots in as a
second source behind the same option and layout.
