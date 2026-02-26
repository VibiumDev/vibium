# Setting Up Local Vibium Dev (macOS)

This doc covers macOS VM setup on a Mac host. For other platforms, see:
- [Linux x86 Setup](local-dev-setup-x86-linux.md) — for Linux VM on x86 Linux host
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

Assuming macOS VM dev on macOS (via UTM).

## Install UTM (on host)

- Install UTM on your host Mac
- Create a macOS VM
- Install Guest Tools (so clipboard can be shared between guest and host)

---

## Install Zed (on host)

- Download from https://zed.dev or `brew install --cask zed`
- Use Zed's remote development feature to edit files inside the VM via SSH

---

## Inside the VM

**All commands below are run inside the VM unless noted otherwise.**
Everything below happens inside the VM.

---

## Create and Edit ~/.zshrc

```bash
export PS1="$ "
WORDCHARS=${WORDCHARS//\/}

# Increase the number of commands kept in session memory
export HISTSIZE=1000000

# Increase the number of commands saved to the history file
export SAVEHIST=1000000

# Specify the history file location (optional, typically default)
export HISTFILE=~/.zsh_history

# Fix Homebrew
export HOMEBREW_NO_AUTO_UPDATE=1
```

---

## Remove "Last login" Message for New Tabs in Terminal

```bash
touch ~/.hushlogin
```

---

## Install Xcode Command Line Developer Tools

```bash
xcode-select --install
```

---

## Install Homebrew

```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

---

## Install Dev Tools

### Languages

```bash
brew install nvm node python go
```

### Tools

```bash
brew install git gh ripgrep jq websocat
```

---

## Install Claude Code

```bash
brew install --cask claude-code
```

---

## Git Config

```bash
git config --global user.name "Your Name"
git config --global user.email "you@example.com"
```

---

## Clone or Fork the Repo

### For Team Members (direct push access)

If you have push access to `VibiumDev/vibium`:

```bash
mkdir -p ~/Projects
cd ~/Projects
git clone https://github.com/VibiumDev/vibium.git
cd vibium
```

Verify it worked:

```bash
git remote -v
```

You should see:

```
origin  https://github.com/VibiumDev/vibium.git (fetch)
origin  https://github.com/VibiumDev/vibium.git (push)
```

### For External Contributors (fork-based workflow)

If you're contributing via pull request:

1. **Fork the repo** in browser: go to `https://github.com/VibiumDev/vibium` and click "Fork"

2. **Clone your fork**:

```bash
mkdir -p ~/Projects
cd ~/Projects
git clone https://github.com/yourusername/vibium.git
cd vibium
git remote add upstream https://github.com/VibiumDev/vibium.git
```

3. **Verify remotes are set up correctly**:

```bash
git remote -v
```

You should see:

```
origin    https://github.com/yourusername/vibium.git (fetch)
origin    https://github.com/yourusername/vibium.git (push)
upstream  https://github.com/VibiumDev/vibium.git (fetch)
upstream  https://github.com/VibiumDev/vibium.git (push)
```

---

## Create a GitHub Personal Access Token (PAT)

Now that you know which repo you're working with, create a PAT scoped to it.

In a browser (on host or VM — wherever you're logged into GitHub):

1. GitHub → Settings → Developer settings
2. Personal access tokens → Fine-grained tokens
3. Generate new token

Token settings:
- Token name: `utm-vm` (or whatever identifies this VM)
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

## You're Ready

To submit changes:
- **Team members**: push directly to `VibiumDev/vibium`
- **External contributors**: push to your fork, then open a PR to `VibiumDev/vibium`

---

## Enable SSH on the macOS Guest (VM)

### Enable Remote Login

On the guest VM:

1. System Settings
2. General → Sharing
3. Turn on Remote Login
4. Allow access for:
   - Your VM user (recommended)
   - Or "All users" if you don't care

This starts sshd.

### Verify SSH is Running

On the VM:

```bash
ssh localhost
```

If it connects, SSH is ready.

### Get the VM's IP Address

On the VM:

```bash
ipconfig getifaddr en0
```

This prints something like `192.168.64.4`. Note this address.

---

## Connect from Host to VM

### Via Terminal (on host)

```bash
ssh yourusername@192.168.64.4
```

Replace `yourusername` with your VM username and `192.168.64.4` with the IP from above.

### Via Zed (on host)

1. Open Zed
2. `Cmd+Shift+P` → "remote projects: Open Remote Project"
3. Enter: `yourusername@192.168.64.4`
4. Navigate to `~/Projects/vibium`

Now you can edit files on the VM from Zed on your host.

---

## Linux ARM VM (for testing linux/arm64 builds)

If you're on Apple Silicon and need to test linux/arm64 builds, you can run an Ubuntu ARM VM alongside your macOS VM.

### Create Linux VM in UTM

1. Open UTM
2. Create New → Virtualize → Linux
3. Download Ubuntu 24.04 LTS ARM64 ISO:
   - https://ubuntu.com/download/server (choose ARM64)
4. Configure VM:
   - Memory: 4GB minimum (8GB recommended)
   - CPU: 4 cores
   - Storage: 32GB minimum
5. Boot and install Ubuntu Server (minimal install is fine)

### Post-Install Setup

SSH into the Linux VM and run:

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install dev tools
sudo apt install -y git curl wget build-essential

# Install Go
wget https://go.dev/dl/go1.23.4.linux-arm64.tar.gz
sudo tar -C /usr/local -xzf go1.23.4.linux-arm64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Install Node.js (via nvm)
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash
source ~/.bashrc
nvm install --lts

# Install other tools
sudo apt install -y ripgrep jq
```

### Test linux/arm64 Build

```bash
# Clone repo (or copy build artifacts)
git clone https://github.com/VibiumDev/vibium.git
cd vibium/clicker

# Build and test
go build -o bin/vibium ./cmd/clicker
./bin/vibium --version
./bin/vibium launch-test
```

### Get VM IP for SSH Access

```bash
ip addr show | grep inet
```

From your Mac host:
```bash
ssh yourusername@<vm-ip>
```
