# Hetzner browser box

Budget cloud instances, and the one kit with a **bare-metal path** —
Hetzner's dedicated servers give you real KVM, which is what Android
emulators need (see the note at the bottom).

## Prereqs (human)

1. Hetzner Cloud account (console.hetzner.cloud; payment method required)
2. `brew install hcloud && hcloud context create vibium` (API token from
   the console)
3. An SSH key uploaded: `hcloud ssh-key list`

## Create

```bash
hcloud server create --name vibium-browser \
  --type cx32 --image ubuntu-24.04 --location fsn1 \
  --ssh-key <name> \
  --user-data-from-file deploy/browser-box/cloud-init.yml
hcloud server ip vibium-browser
```

First boot takes ~3–5 min while cloud-init installs Chrome. Snapshot
afterward and future boxes start in ~1 min. Locations: Falkenstein,
Nuremberg, Helsinki (EU), Ashburn, Hillsboro (US), Singapore.

## Connect

```bash
ssh -N -L 9515:127.0.0.1:9515 root@<server-ip> &
vibium start http://127.0.0.1:9515
vibium go https://example.com
vibium stop
```

chromedriver listens on loopback only; the tunnel is the only way in.

## Teardown

```bash
hcloud server delete vibium-browser
```

Hourly billing, capped at the monthly price.

## Bare metal (the Android-emulator path)

Hetzner's dedicated servers (Robot console, including the server
auction) are real hardware with native KVM — the requirement Android
emulators have that microVM platforms can't meet. The same
`setup-chrome.sh --service` works there over SSH, and Google's
android-emulator-container-scripts run alongside on the same box.
Provisioning is a manual order rather than an API call, so treat it as
a standing fleet, not burst capacity.
