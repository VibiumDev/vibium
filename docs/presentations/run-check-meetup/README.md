# Run & Check — technical meetup

A 19-slide, 15–20 minute HTML talk for developers and testers. Includes real browser screenshots, two short silent videos, code examples, speaker notes, and original recordings.

## Present

Open **index.html** in a current desktop browser. Keep the whole folder together.
For reliable caption loading and video seeking, serve it locally with Node.js:

~~~sh
node serve.mjs
~~~

Open **http://127.0.0.1:4181**. No network connection or API credentials are needed to present. The server uses only Node built-ins. Use PORT to choose another port.

- Left / Right, Page Up / Page Down, or Space: navigate.
- N: speaker notes (visible on the presentation screen; not a separate presenter monitor).
- F: fullscreen.
- O: slide overview.
- B: black screen; Escape restores it.
- Home / End: first / last slide.
- Videos have native playback controls and English captions. They are silent, at normal speed.
- Browser Print: all slides, with still-image replacements for video. Enable background graphics.

Use the separate **speaker-notes.md** if you do not want notes visible to the audience. The deck scales a 1600 × 900 canvas to the window; a landscape display is recommended.

## What was actually captured

Captured September 7, 2026 (America/Chicago), using the local development build of Vibium, **OpenAI / gpt-5.6-sol**, and Firefox. Exact UTC time, prompts, evidence, parent spans, child actions, and public model settings are in **assets/demo-results.json**.

| Fixture | Run | Check | Video |
| --- | --- | --- | --- |
| Deliberately missing persistence | completed | failed after reload | broken.mp4 / broken.webm, ~12 seconds |
| Persistence enabled | completed | passed after reload | fixed.mp4 / fixed.webm, ~13 seconds |

These are real model invocations on synthetic test data. The default fixture deliberately shows Saved without persisting the field. Its fixed mode writes and reads localStorage. This is a controlled illustration, not a backend integration or an accuracy benchmark. A deterministic assertion is also a good fit for this particular invariant.

The native Firefox WebM files are unedited. MP4 files are H.264 transcodes for playback compatibility, without speed changes. Captions label operation windows using recording timestamps. The Record Player screenshot is from the actual broken ZIP opened in the local Record Player. The Pinthing screenshot is copied from the existing first-check tutorial and is labeled as such; it is not a new availability check.

Original archives: **assets/broken.zip**, **assets/fixed.zip**.
Open them in Vibium Record Player to inspect the groups, child actions, and evidence.

## Run the fixture yourself

This deck describes the current development build. Check that your installed build includes run, check, and ready; the public package version may differ.

~~~sh
# From this presentation folder, terminal 1:
node demo/server.mjs

# Terminal 2, after configuring the model:
vibium go http://127.0.0.1:4177
vibium run "Change the timezone to America/Chicago and save it. Confirm the page shows a Saved message. Stay on this page."
vibium check "The saved timezone is America/Chicago after reloading this page. Reload and inspect the value without editing it or clicking Save."
vibium stop
~~~

For the fixed page, use http://127.0.0.1:4177/?persist=1.
Start a fresh browser between runs to isolate localStorage. Use the same session throughout each workflow.

For recording with native video, open the page with Firefox 154+:

~~~sh
vibium --engine firefox go http://127.0.0.1:4177
vibium record start --video --snapshots -o flow.zip
vibium run "Change the timezone to America/Chicago and save it"
vibium check "The saved timezone is America/Chicago after reload. Do not edit the form."
vibium record stop
vibium stop
~~~

Choose a new output filename. Chrome records screenshots and actions, without continuous video; omit --video when using Chrome.

## Configure privately

Keep credentials outside the repository. In a private environment file (permissions 600), use exported assignments:

~~~sh
export OPENAI_API_KEY='your-key'
export VIBIUM_AI_PROVIDER=openai
export VIBIUM_AI_MODEL=gpt-5.6-sol
export VIBIUM_AI_REASONING_EFFORT=none
~~~

Load it in the shell running Vibium:

~~~sh
source ~/.config/vibium/ai.env
vibium ready ai
~~~

Use a model your account can access and settings it supports. AI readiness and live captures make provider requests and can incur API charges. Do not put real credentials in this folder, slide content, or shell transcripts.

## Regenerate the recordings

The capture script invokes the existing Vibium CLI and opens isolated Firefox sessions. It does not use another browser automation layer. It performs four real model operations, saves PNGs and native recordings, and writes the actual results manifest.

~~~sh
# Load your private exported provider configuration first.
# Use a new directory; bundled recordings will not be overwritten.
MEETUP_CAPTURE_DIR=/tmp/my-meetup-capture \
  node demo/capture.mjs
~~~

In the repository it defaults to clicker/bin/vibium. Outside the checkout, set VIBIUM_BIN_PATH to the absolute path of a compatible Vibium binary. The script also requires unzip. It uses the VIBIUM_AI provider/model settings explicitly for both roles for this comparison.

The script creates WebM files. To produce compatible MP4 copies, use ffmpeg:

~~~sh
ffmpeg -i broken.webm -c:v libx264 -crf 21 -pix_fmt yuv420p -movflags +faststart broken.mp4
ffmpeg -i fixed.webm -c:v libx264 -crf 21 -pix_fmt yuv420p -movflags +faststart fixed.mp4
~~~

If you replace the recordings, update screenshots, captions, timing, results, and any quoted evidence in the slides to match the new runs.

## Source material and boundaries

The **sources/** folder contains snapshots of the repository documents used for this talk; slide notes link to them. Their original locations are:

- docs/explanation/run-check-webdriver-bidi-architecture.md
- docs/how-to-guides/run.md
- docs/reference/check.md
- docs/reference/ready.md
- docs/reference/model-providers.md
- docs/tutorials/first-check.md
- docs/tutorials/check-with-a-coding-agent.md
- docs/tutorials/check-from-a-recording.md

Source snapshots are Markdown references; some links inside them point to the wider repository and require that checkout.

The talk distinguishes model judgments from proof. A fresh context does not guarantee independent errors, and a passing AI readiness does not test the application. Existing deterministic tests remain important. Live operations can change state. Known-secret redaction and recognized-sensitive-field suppression are not a guarantee that all unknown secrets are removed.

Real-provider acceptance in this development checkout has been exercised with OpenAI. Anthropic, Gemini, and llama.cpp adapters have deterministic coverage; their real-provider acceptance remains pending configuration.
