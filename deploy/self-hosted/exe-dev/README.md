# Self-hosted cloud browser on exe.dev

exe.dev VMs boot in ~2 seconds from an image that already includes
Chrome headless-shell and every Chrome library dependency — the least
setup of any platform here. The whole API is SSH.

## Prereqs (human)

1. An SSH keypair
2. `ssh exe.dev` — interactive signup (email + region)
3. Personal plan $20/mo (2 vCPU / 8 GB pooled across up to 50 VMs);
   trial terms vary — check on signup

## Two ways to use it

### A. Everything-cloud (recommended — no tunnel, no latency)

Run the agent, vibium, and Chrome all on the VM:

```bash
ssh exe.dev new --name vibium-box        # VM ready in ~2s
ssh vibium-box.exe.xyz
# on the VM: claude/codex are preinstalled; add vibium
npm install -g vibium
vibium go https://example.com            # everything local to the VM
```

There is no client↔browser network hop at all — BiDi round-trips are
loopback. This is the "no speed of light penalty" topology.

### B. Split (browser on the VM, vibium on your laptop)

```bash
ssh vibium-box.exe.xyz -- bash -s -- --service < deploy/self-hosted/install-chrome.sh
ssh -N -L 9515:127.0.0.1:9515 vibium-box.exe.xyz &
vibium start http://127.0.0.1:9515
```

## Teardown

```bash
ssh exe.dev rm vibium-box
```

Flat-plan VMs bill while idle (no scale-to-zero), so delete or
downsize boxes you're done with.

## Caveats

- Confirm VM arch on first login (`uname -m`): vibium ships linux-x64
  and linux-arm64 npm packages, but the exeuntu image is multi-arch —
  make sure the right one landed.
