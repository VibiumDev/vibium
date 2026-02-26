# Setting Up Local Vibium Dev (x86 Linux)

> **Draft**: This doc has not been tested yet. Instructions may need adjustment.

This doc covers Linux VM setup on an x86 Linux host for isolated development.

For other platforms, see:
- [macOS Setup](local-dev-setup-mac.md) — for macOS VM on Mac host
- [Windows x86 Setup](local-dev-setup-x86-windows.md) — for Windows VM on x86 Windows host

---

## Why Develop Inside a Virtual Machine?

When using AI-assisted tools like Claude Code, it's important to limit the "blast radius" of what the AI can access or modify. A VM provides hard boundaries:

- **Containment**: The AI can only see/modify files inside the VM — not your host machine, personal files, or other projects
- **Scoped credentials**: GitHub PATs and API keys are isolated to the VM and scoped to specific repos
- **Easy reset**: If something goes wrong, you can restore from a snapshot or rebuild the VM from scratch
- **Reproducible environment**: Every developer starts from the same clean slate
- **Peace of mind**: You can let the AI operate more freely without worrying about unintended side effects

This isn't about distrust — it's defense in depth. The same reason you don't run untested code as root.

---

## Hardware

### Why x86 Hardware?

On Apple Silicon, you can run x86 Windows via Parallels (paid license required), but for testing both Linux x86 and Windows x86 without ongoing costs, a dedicated x86 mini PC is simpler.

### Recommended Specs

| Spec | Minimum | Recommended |
|------|---------|-------------|
| CPU | Any modern x86_64 | Ryzen 5 / Intel i5 |
| RAM | 8GB | 16GB+ (for VMs) |
| Storage | 256GB SSD | 512GB+ SSD |

Any mini PC in the $150-350 range with these specs will work fine.

---

## Host Setup

### Install Ubuntu 24.04 LTS (Host)

1. Download Ubuntu 24.04 Desktop ISO: https://ubuntu.com/download/desktop
2. Create bootable USB with Balena Etcher or `dd`
3. Boot from USB, install Ubuntu

### Install Virtualization Tools

```bash
# Install QEMU/KVM and virt-manager
sudo apt update
sudo apt install -y qemu-kvm libvirt-daemon-system libvirt-clients bridge-utils virt-manager

# Add your user to libvirt group
sudo usermod -aG libvirt $USER
sudo usermod -aG kvm $USER

# Log out and back in for group changes to take effect
```

### Enable SSH on Host (for remote access)

```bash
sudo apt install -y openssh-server
sudo systemctl enable ssh
sudo systemctl start ssh

# Get IP address
ip addr show | grep inet
```

---

## Create Linux VM (Guest)

### Download Ubuntu ISO

Download Ubuntu 24.04 Server or Desktop ISO for the guest VM.

### Create VM with virt-manager

1. Open virt-manager (Virtual Machine Manager)
2. File → New Virtual Machine
3. Select "Local install media" → Browse to Ubuntu ISO
4. Configure:
   - Memory: 4GB minimum (8GB recommended)
   - CPUs: 4 cores
   - Storage: 64GB minimum
5. Name it something like `vibium-dev`
6. Finish and start installation

### Install Ubuntu in VM

Standard Ubuntu install. Minimal/Server install is fine.

---

## Inside the VM

**All commands below are run inside the VM.**

---

## Create and Edit ~/.bashrc

Add to `~/.bashrc`:

```bash
# Increase history
export HISTSIZE=1000000
export SAVEHIST=1000000
export HISTFILE=~/.bash_history
```

---

## Install Dev Tools

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install essentials
sudo apt install -y git curl wget build-essential openssh-server
```

---

## Install Go

```bash
wget https://go.dev/dl/go1.23.4.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.23.4.linux-amd64.tar.gz
rm go1.23.4.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
go version
```

---

## Install Node.js (via nvm)

```bash
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash
source ~/.bashrc
nvm install --lts
node --version
npm --version
```

---

## Install Tools

```bash
sudo apt install -y ripgrep jq

# GitHub CLI
curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | sudo dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | sudo tee /etc/apt/sources.list.d/github-cli.list > /dev/null
sudo apt update
sudo apt install -y gh
```

---

## Install Claude Code

```bash
# Install via npm
npm install -g @anthropic-ai/claude-code
```

---

## Git Config

```bash
git config --global user.name "Your Name"
git config --global user.email "you@example.com"
```

---

## Clone the Repo

```bash
mkdir -p ~/Projects
cd ~/Projects
git clone https://github.com/VibiumDev/vibium.git
cd vibium
```

---

## Create a GitHub Personal Access Token (PAT)

Now that you know which repo you're working with, create a PAT scoped to it.

In a browser (on host or VM — wherever you're logged into GitHub):

1. GitHub → Settings → Developer settings
2. Personal access tokens → Fine-grained tokens
3. Generate new token

Token settings:
- Token name: `linux-vm` (or whatever identifies this VM)
- Expiration: 7 days (or 30 if you hate rotating)
- Resource owner: your username
- Repository access: Only select repositories
  - **Team members**: select `VibiumDev/vibium`
  - **External contributors**: select `yourusername/vibium` (your fork)

Permissions:
- Contents: Read and write
- Issues: Read and write
- Metadata: Read-only (required, auto-selected)
- Pull requests: Read and write
- Everything else: No access

Click "Generate token" and copy it (you won't see it again).

### Why PAT instead of browser auth?

Browser auth gives full account access. A fine-grained PAT limits blast radius:
- Scoped to specific repos
- Expires automatically
- Contained inside the VM

---

## Authenticate with GitHub

```bash
gh auth login
```

Follow the prompts:
- Account: GitHub.com
- Protocol: HTTPS
- Authenticate: Paste an authentication token

Paste your PAT when prompted. Credentials are stored automatically.

Verify it worked:

```bash
gh auth status
```

---

## Enable SSH in the VM

```bash
sudo systemctl enable ssh
sudo systemctl start ssh

# Get VM IP
ip addr show | grep inet
```

---

## Connect from Host to VM

### Via Terminal (on host)

```bash
ssh yourusername@<vm-ip>
```

### Via Zed (on host)

1. Install Zed: https://zed.dev or `curl -fsSL https://zed.dev/install.sh | sh`
2. Open Zed
3. `Ctrl+Shift+P` → "remote projects: Open Remote Project"
4. Enter: `yourusername@<vm-ip>`
5. Navigate to `~/Projects/vibium`

---

## Build and Test

```bash
cd ~/Projects/vibium/clicker
go build -o bin/vibium ./cmd/clicker
./bin/vibium --version
./bin/vibium paths
./bin/vibium install
./bin/vibium launch-test
```

---

## Snapshots

Take VM snapshots before risky operations:

```bash
# On host, with VM shut down
virsh snapshot-create-as vibium-dev clean-slate "Fresh dev setup"

# List snapshots
virsh snapshot-list vibium-dev

# Restore snapshot
virsh snapshot-revert vibium-dev clean-slate
```

Or use virt-manager GUI: right-click VM → Snapshots.
