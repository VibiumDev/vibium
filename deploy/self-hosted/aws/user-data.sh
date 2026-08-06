#!/bin/bash
# EC2 user-data: Chrome + chromedriver as a systemd service on
# 127.0.0.1:9515. Ubuntu AMIs. Reach it over an SSH tunnel.
#
#   aws ec2 run-instances --image-id <ubuntu-24.04-ami> \
#     --instance-type t3.medium --key-name <keypair> \
#     --security-group-ids <ssh-only-sg> \
#     --user-data file://deploy/self-hosted/aws/user-data.sh
#
# Same note as the DigitalOcean recipe: until these land on the main
# branch, scp setup-chrome.sh up and run it by hand instead.
set -euo pipefail
curl -fsSL https://raw.githubusercontent.com/VibiumDev/vibium/main/deploy/self-hosted/setup-chrome.sh -o /root/setup-chrome.sh
bash /root/setup-chrome.sh --service
