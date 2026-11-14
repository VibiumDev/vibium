# Keep a Piper Voice Loaded

Piper speaks on your own machine, with no key and no network. Started
for every phrase, it is slow: on a Raspberry Pi 4 a short phrase takes 10
to 15 seconds, and most of that is loading the voice. Piper also ships a
small server that loads a voice once and keeps it. Through it the same
phrase takes about a second.

Piper is a separate program under its own license (GPL-3.0). Vibium does
not include it or start it. You run the server, and `vibium say` sends it
text.

## 1. Install Piper with its server

```bash
python3 -m venv ~/piper
~/piper/bin/pip install "piper-tts[http]"
```

## 2. Get a voice

```bash
mkdir -p ~/piper/voices
~/piper/bin/python -m piper.download_voices --download-dir ~/piper/voices en_US-bryce-medium
```

Each voice has its own license, and some are for research only. The
[say and listen spec](../specs/say-listen-spec.md) has a table.

The [Piper voices](https://huggingface.co/rhasspy/piper-voices) page lists
the others. A low voice renders about a third faster than a medium one.
A medium one has a higher sample rate.

## 3. Start the server

```bash
~/piper/bin/python -m piper.http_server \
  -m ~/piper/voices/en_US-bryce-medium.onnx \
  --data-dir ~/piper/voices --host 127.0.0.1 --port 5000
```

- `--host 127.0.0.1` keeps it to this machine. Without it the server
  listens on every address, and it asks for no password.
- The first seconds go to loading the voice. After that it answers.

## 4. Speak through it

```bash
export VIBIUM_PIPER_URL=http://127.0.0.1:5000
vibium say "Come here, Watson"
# Said "Come here, Watson" (1.2 s, piper-server)
```

`--provider piper-server` picks it for one call. Without
`VIBIUM_PIPER_URL` the address is Piper's default, `http://127.0.0.1:5000`.

- `--voice en_US-amy-low` speaks with another voice from the server's
  `--data-dir`. The server loads it on first use and keeps it too.
- `--speed 1.2` works as with any provider.
- Rendered audio is cached as usual. The cache knows which voice the
  server has loaded, so a server started with another voice renders again.

## 5. Keep it running

A user service starts the server at login and restarts it if it stops.
Save this as `~/.config/systemd/user/piper.service`:

```ini
[Unit]
Description=Piper speech server, with a voice kept loaded

[Service]
ExecStart=%h/piper/bin/python -m piper.http_server -m %h/piper/voices/en_US-bryce-medium.onnx --data-dir %h/piper/voices --host 127.0.0.1 --port 5000
Restart=on-failure

[Install]
WantedBy=default.target
```

```bash
systemctl --user daemon-reload
systemctl --user enable --now piper
```

## When something is off

- `engine_failed: no Piper server answered`: the server is not running,
  or `VIBIUM_PIPER_URL` names another address. `curl http://127.0.0.1:5000/info`
  answers with the loaded voice when it is up.
- A phrase is slow once after a start: the voice is still loading.
- `vibium say` uses another engine: a `--provider`, `VIBIUM_SAY_PROVIDER`
  or `VIBIUM_SAY_COMMAND` you set comes first.
