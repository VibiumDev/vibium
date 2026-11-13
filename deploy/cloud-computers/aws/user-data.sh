#!/bin/bash
# EC2 user-data: Chrome + chromedriver as a systemd service on
# 127.0.0.1:9515. Ubuntu AMIs. Reach it over an SSH tunnel.
#
#   aws ec2 run-instances --image-id <ubuntu-24.04-ami> \
#     --instance-type t3.medium --key-name <keypair> \
#     --security-group-ids <ssh-only-sg> \
#     --user-data file://deploy/cloud-computers/aws/user-data.sh
#
# Same note as the DigitalOcean recipe: until these land on the main
# branch, scp install-chrome.sh up and run it by hand instead.
set -euo pipefail
curl -fsSL https://raw.githubusercontent.com/VibiumDev/vibium/main/deploy/cloud-computers/install-chrome.sh -o /root/install-chrome.sh
bash /root/install-chrome.sh --service
