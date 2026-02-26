# Setting Up Local Vibium Dev (x86 Windows)
This doc covers Windows VM setup on an x86 Windows host for isolated development.

For other platforms, see:
- [macOS Setup](local-dev-setup-mac.md) — for macOS VM on Mac host
- [Linux x86 Setup](local-dev-setup-x86-linux.md) — for Linux VM on x86 Linux host

---

## Why Develop Inside a Virtual Machine?

When using AI-assisted tools like Claude Code, it's important to limit the "blast radius" of what the AI can access or modify. A VM provides hard boundaries:

- **Containment**: The AI can only see/modify files inside the VM — not your host machine, personal files, or other projects
- **Scoped credentials**: GitHub PATs and API keys are isolated to the VM and scoped to specific repos
- **Easy reset**: If something goes wrong, you can restore from a checkpoint or rebuild the VM from scratch
- **Reproducible environment**: Every developer starts from the same clean slate
- **Peace of mind**: You can let the AI operate more freely without worrying about unintended side effects

This isn't about distrust — it's defense in depth. The same reason you don't run untested code as admin.

---

## Hardware

See [Linux x86 Setup](local-dev-setup-x86-linux.md#hardware) for hardware recommendations.

---

## Host Setup

### Install Windows 11 Pro

Standard Windows 11 Pro install on the mini PC. Pro edition is required for Hyper-V.

### Enable Hyper-V

1. Open PowerShell as Administrator
2. Run:

```powershell
Enable-WindowsOptionalFeature -Online -FeatureName Microsoft-Hyper-V -All
```

3. Restart when prompted

### Enable SSH on Host (for remote access)

```powershell
# Run as Administrator
Add-WindowsCapability -Online -Name OpenSSH.Server~~~~0.0.1.0
Start-Service sshd
Set-Service -Name sshd -StartupType Automatic

# Allow SSH through Windows Firewall
New-NetFirewallRule -Name sshd -DisplayName 'OpenSSH Server' -Enabled True -Direction Inbound -Protocol TCP -Action Allow -LocalPort 22

# Get IP address
ipconfig
```

### Create External Virtual Switch (for LAN access to VM)

By default, Hyper-V's "Default Switch" gives VMs a NAT IP only reachable from the host. To access the VM from other machines on your network (e.g., your Mac), create an External Switch:

1. Open Hyper-V Manager
2. Right panel → Virtual Switch Manager
3. Select **External** → Create Virtual Switch
4. Name: `External Switch`
5. Under "External network", select your physical network adapter
6. Apply/OK (this will briefly drop the host's network connection)

---

## Create Windows VM (Guest)

### Download Windows ISO

Download Windows 11 ISO from Microsoft: https://www.microsoft.com/software-download/windows11

### Create VM with Hyper-V Manager

1. Open Hyper-V Manager
2. Action → New → Virtual Machine
3. Configure:
   - Name: `vibium-dev`
   - Generation: Generation 2
   - Memory: 4GB minimum (8GB recommended), enable Dynamic Memory
   - Network: External Switch (created above)
   - Virtual Hard Disk: 64GB minimum
4. Install Options: select Windows ISO
5. Finish and start installation

### VM Settings (before first boot)

Right-click VM → Settings:
- Security: Enable Trusted Platform Module (required for Windows 11)
- Security: Disable Secure Boot (or set to "Microsoft UEFI Certificate Authority")
- Processor: 4 virtual processors
- Checkpoints: Enable (for snapshots)

### Install Windows in VM

1. Start the VM and **immediately click inside the console window**
2. Spam the spacebar — you need to hit a key while the "Press any key to boot from CD or DVD..." prompt is visible (it disappears quickly)
3. Standard Windows 11 install. Use a local account for simplicity.

---

## Inside the VM

**All commands below are run inside the VM.**

---

## Enable OpenSSH Server

```powershell
# Run in PowerShell as Administrator
Add-WindowsCapability -Online -Name OpenSSH.Server~~~~0.0.1.0
Start-Service sshd
Set-Service -Name sshd -StartupType Automatic

# Allow SSH through Windows Firewall
New-NetFirewallRule -Name sshd -DisplayName 'OpenSSH Server' -Enabled True -Direction Inbound -Protocol TCP -Action Allow -LocalPort 22

# Set PowerShell as default SSH shell
New-ItemProperty -Path "HKLM:\SOFTWARE\OpenSSH" -Name DefaultShell -Value "C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe" -PropertyType String -Force
```

Get VM IP:
```powershell
ipconfig
```

---

## Allow PowerShell Scripts

PowerShell blocks `.ps1` scripts by default, which breaks tools like `npm`. Run this once to allow scripts you install:

```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

---

## Install Dev Tools (via winget)

Windows 11 includes `winget` by default. Open a regular PowerShell (not Administrator):

```powershell
winget install Git.Git
winget install GitHub.cli
winget install GnuWin32.Make
winget install BurntSushi.ripgrep.MSVC
winget install jqlang.jq
```

Restart terminal after installing Git, then add GnuWin32 Make and Git's Unix tools to your PATH:

```powershell
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";C:\Program Files (x86)\GnuWin32\bin;C:\Program Files\Git\usr\bin", "User")
```

This adds `make` plus Unix tools (`cp`, `rm`, `mkdir`, `cat`, `bash`, etc.) that the Makefile requires. GnuWin32 Make 3.81 doesn't use the Makefile's `SHELL` variable, so these tools must be directly on PATH.

Restart your terminal after updating PATH. Verify:

```powershell
make --version
bash --version
```

---

## Install Go

```powershell
winget install GoLang.Go
```

Restart terminal, then verify:
```powershell
go version
```

---

## Install Node.js

```powershell
winget install OpenJS.NodeJS.LTS
```

Restart terminal, then verify:
```powershell
node --version
npm --version
```

---

## Install Claude Code

```powershell
winget install Anthropic.ClaudeCode
```

---

## Git Config

```powershell
git config --global user.name "Your Name"
git config --global user.email "you@example.com"
git config --global core.autocrlf input
```

---

## Clone the Repo

```powershell
mkdir C:\Projects
cd C:\Projects
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
- Token name: `windows-vm` (or whatever identifies this VM)
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

```powershell
gh auth login
```

Follow the prompts:
- Account: GitHub.com
- Protocol: HTTPS
- Authenticate: Paste an authentication token

Paste your PAT when prompted. Credentials are stored automatically.

Verify it worked:

```powershell
gh auth status
```

---

## Connect to VM

With the External Switch, the VM has a LAN IP accessible from any machine on your network.

### Via Terminal

```sh
ssh yourusername@<vm-ip>
```

### Via Zed

1. Install Zed: https://zed.dev
2. Open Zed
3. `Ctrl+Shift+P` → "remote projects: Open Remote Project"
4. Enter: `yourusername@<vm-ip>`
5. Navigate to `C:\Projects\vibium`

---

## Build and Test

```powershell
cd C:\Projects\vibium
make build
make test
```

To verify manually:

```powershell
.\clicker\bin\vibium.exe --version
.\clicker\bin\vibium.exe paths
```

---

## Checkpoints (Snapshots)

Take VM checkpoints before risky operations:

1. Open Hyper-V Manager
2. Right-click VM → Checkpoint
3. Name it (e.g., "Fresh dev setup")

To restore:
1. Right-click checkpoint → Apply
2. Or right-click → Revert to restore to most recent

---

## Tips

### Path Length

Windows has a 260-character path limit by default. Enable long paths:

```powershell
# Run as Administrator
New-ItemProperty -Path "HKLM:\SYSTEM\CurrentControlSet\Control\FileSystem" -Name "LongPathsEnabled" -Value 1 -PropertyType DWORD -Force
```

### PowerShell Emacs Keybindings (optional)

If you're coming from Mac/Linux, PowerShell's default keybindings will feel wrong. This gives you familiar shortcuts (Ctrl+A, Ctrl+E, Ctrl+K, etc.):

```powershell
Set-PSReadLineOption -EditMode Emacs
```

To make it permanent, add it to your PowerShell profile:

```powershell
Add-Content $PROFILE "Set-PSReadLineOption -EditMode Emacs"
```

### Windows Defender Exclusions

Defender can slow down builds. Add exclusions:

1. Windows Security → Virus & threat protection → Manage settings
2. Exclusions → Add or remove exclusions
3. Add folder: `C:\Projects`
4. Add folder: `C:\Users\<you>\go`
