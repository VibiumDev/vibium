# Self-hosted cloud browser on Google Cloud

A Compute Engine VM behind an SSH tunnel — with the nicest tunnel
story of the platforms here, since `gcloud compute ssh` manages keys itself.

## Prereqs (human)

1. GCP project with billing enabled
2. `gcloud init` (the CLI drives everything, including the tunnel)

## Create

```bash
gcloud compute instances create vibium-browser \
  --machine-type=e2-medium --zone=us-central1-a \
  --image-family=ubuntu-2404-lts-amd64 --image-project=ubuntu-os-cloud \
  --metadata-from-file user-data=deploy/self-hosted/cloud-init.yml
```

First boot takes ~3–5 min while cloud-init installs Chrome. Bake an
image afterward for ~1 min starts. Spot VMs (`--provisioning-model=SPOT`)
are fine for disposable boxes.

## Connect

```bash
gcloud compute ssh vibium-browser --zone=us-central1-a -- -N -L 9515:127.0.0.1:9515 &
vibium start http://127.0.0.1:9515
vibium go https://example.com
vibium stop
```

`gcloud compute ssh` manages keys for you; chromedriver stays on
loopback and the tunnel is the only way in.

## Teardown

```bash
gcloud compute instances delete vibium-browser --zone=us-central1-a
```

Per-second billing.

