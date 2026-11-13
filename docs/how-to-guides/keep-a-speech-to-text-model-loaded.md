# Keep a Speech-to-Text Model Loaded

A speech-to-text engine on your own machine needs no key and no network.
Started for every call, it is slow, because it loads its model each time:
on a Raspberry Pi 4, whisper.cpp takes about 10 seconds for a 6-second
reply. A [sherpa-onnx](https://github.com/k2-fsa/sherpa-onnx) streaming
server loads a model once and keeps it. Through it the same reply takes 2
to 3 seconds.

sherpa-onnx is a separate program (Apache-2.0). Vibium does not include
it or start it. You run the server, and `vibium listen` sends it the
audio.

## 1. Install sherpa-onnx

```bash
python3 -m venv ~/sherpa
~/sherpa/bin/pip install sherpa-onnx websockets
```

## 2. Get a model

```bash
mkdir -p ~/sherpa/models
curl -L https://github.com/k2-fsa/sherpa-onnx/releases/download/asr-models/sherpa-onnx-streaming-zipformer-en-2023-06-26.tar.bz2 \
  | tar xj -C ~/sherpa/models
```

This is an English streaming Zipformer model, under Apache-2.0. The
[say and listen spec](../specs/say-listen-spec.md#speech-to-text) compares
it with another one. sherpa-onnx's
[streaming Zipformer models](https://k2-fsa.github.io/sherpa/onnx/pretrained_models/online-transducer/index.html)
page lists more, for other languages too.

## 3. Start a server

[Station](https://github.com/VibiumDev/station) has a small server script
for this, which runs by itself:

```bash
curl -LO https://raw.githubusercontent.com/VibiumDev/station/main/cmd/station/sherpa_server.py
~/sherpa/bin/python sherpa_server.py \
  --model ~/sherpa/models/sherpa-onnx-streaming-zipformer-en-2023-06-26 --port 6006
```

- It listens on `127.0.0.1`, so only this machine reaches it, and it
  turns a browser's pages away.
- The first 10 to 15 seconds go to loading the model. After that it
  answers.

sherpa-onnx's own servers work too: `sherpa-onnx-online-websocket-server`
from its releases, and `streaming_server.py` from its Python examples.
They listen on every address and ask for no password, so run one only on
a network you trust.

## 4. Listen through it

```bash
export VIBIUM_SHERPA_URL=ws://127.0.0.1:6006
vibium listen --target reply.wav --silence 0
# come here watson i want to see you
```

`--provider sherpa-server` picks it for one call. Without
`VIBIUM_SHERPA_URL` the address is the servers' default,
`ws://127.0.0.1:6006`.

- With a session open, `vibium listen` hears the session, as with any
  provider.
- `--json` gives the words with their times.
- A model trained on upper-case text answers in capitals. Vibium gives
  that text in lower case.
- `--model` and `--language` change nothing here: the server has one
  model loaded.

## 5. Keep it running

A user service starts the server at login and restarts it if it stops.
Save this as `~/.config/systemd/user/sherpa.service`, with the script in
`~/sherpa`:

```ini
[Unit]
Description=sherpa-onnx speech-to-text server, with a model kept loaded

[Service]
ExecStart=%h/sherpa/bin/python %h/sherpa/sherpa_server.py --model %h/sherpa/models/sherpa-onnx-streaming-zipformer-en-2023-06-26 --port 6006
Restart=on-failure

[Install]
WantedBy=default.target
```

```bash
systemctl --user daemon-reload
systemctl --user enable --now sherpa
```

## When something is off

- `engine_failed: no sherpa-onnx server answered`: the server is not
  running, or `VIBIUM_SHERPA_URL` names another address.
  `curl http://127.0.0.1:6006/info` answers with the loaded model when
  Station's script is up.
- `engine_failed: … stopped before it finished`: the server closed the
  connection. Its own output says why.
- `vibium listen` uses another engine: a `--provider`,
  `VIBIUM_LISTEN_PROVIDER` or `VIBIUM_LISTEN_COMMAND` you set comes first.
