#!/usr/bin/env bash
# Install Chrome for Testing + matching chromedriver on a Debian/Ubuntu
# linux64 box, and run chromedriver as a non-root user (Chrome refuses to
# start as root).
#
# Used by the DigitalOcean cloud-init, the AWS user-data script, the Fly.io
# Dockerfile, and the exe.dev recipe — one script, every DIY target.
#
#   sudo ./setup-chrome.sh            # install only
#   sudo ./setup-chrome.sh --service  # install + systemd unit on :9515
#
# Connect from a laptop through a tunnel (never expose 9515 publicly):
#   ssh -N -L 9515:127.0.0.1:9515 <user>@<box>
#   vibium start http://127.0.0.1:9515

set -euo pipefail

CFT_JSON="https://googlechromelabs.github.io/chrome-for-testing/last-known-good-versions-with-downloads.json"
INSTALL_DIR="/opt/chrome-for-testing"
RUN_USER="vibium"

if [ "$(uname -m)" != "x86_64" ]; then
    echo "error: this script installs the linux64 build; $(uname -m) is not supported" >&2
    exit 1
fi

echo "Installing Chrome dependencies..."
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq --no-install-recommends \
    curl unzip jq ca-certificates fonts-liberation \
    libasound2t64 libatk-bridge2.0-0 libatk1.0-0 libcairo2 libcups2 \
    libdbus-1-3 libdrm2 libgbm1 libglib2.0-0 libgtk-3-0 libnss3 \
    libpango-1.0-0 libx11-6 libxcb1 libxcomposite1 libxdamage1 \
    libxext6 libxfixes3 libxkbcommon0 libxrandr2 xdg-utils \
    2>/dev/null || apt-get install -y -qq --no-install-recommends \
    curl unzip jq ca-certificates fonts-liberation \
    libasound2 libatk-bridge2.0-0 libatk1.0-0 libcairo2 libcups2 \
    libdbus-1-3 libdrm2 libgbm1 libglib2.0-0 libgtk-3-0 libnss3 \
    libpango-1.0-0 libx11-6 libxcb1 libxcomposite1 libxdamage1 \
    libxext6 libxfixes3 libxkbcommon0 libxrandr2 xdg-utils

echo "Resolving latest stable Chrome for Testing..."
CHROME_URL=$(curl -fsSL "$CFT_JSON" | jq -r '.channels.Stable.downloads.chrome[] | select(.platform=="linux64") | .url')
DRIVER_URL=$(curl -fsSL "$CFT_JSON" | jq -r '.channels.Stable.downloads.chromedriver[] | select(.platform=="linux64") | .url')
VERSION=$(curl -fsSL "$CFT_JSON" | jq -r '.channels.Stable.version')
echo "Version: $VERSION"

mkdir -p "$INSTALL_DIR"
cd "$INSTALL_DIR"
curl -fsSL -o chrome.zip "$CHROME_URL"
curl -fsSL -o chromedriver.zip "$DRIVER_URL"
unzip -oq chrome.zip
unzip -oq chromedriver.zip
rm chrome.zip chromedriver.zip
ln -sf "$INSTALL_DIR/chrome-linux64/chrome" /usr/local/bin/chrome
ln -sf "$INSTALL_DIR/chromedriver-linux64/chromedriver" /usr/local/bin/chromedriver

id -u "$RUN_USER" >/dev/null 2>&1 || useradd --system --create-home --shell /usr/sbin/nologin "$RUN_USER"
chown -R "$RUN_USER":"$RUN_USER" "$INSTALL_DIR"

echo "Installed: $(chromedriver --version)"

if [ "${1:-}" = "--service" ]; then
    cat > /etc/systemd/system/chromedriver.service <<EOF
[Unit]
Description=chromedriver (WebDriver BiDi endpoint for vibium)
After=network.target

[Service]
User=$RUN_USER
# Loopback-only: reach it over an SSH tunnel. To serve the private network
# instead (e.g. Fly private networking), add --allowed-ips="" and front it
# with your platform's network isolation.
ExecStart=/usr/local/bin/chromedriver --port=9515
Restart=always
RestartSec=2

[Install]
WantedBy=multi-user.target
EOF
    systemctl daemon-reload
    systemctl enable --now chromedriver
    echo "chromedriver listening on 127.0.0.1:9515 (systemd unit: chromedriver.service)"
fi
