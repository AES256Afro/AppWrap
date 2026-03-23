# AppWrap

**Containerize any Windows application.** AppWrap scans an installed app, discovers all its dependencies, and wraps it into a Docker container — complete with encryption, firewall rules, and VPN support baked in.

```
appwrap scan "C:\Program Files\Mozilla Firefox\firefox.exe" --encrypt --firewall deny
appwrap build firefox-profile.yaml -t firefox:latest
appwrap run firefox:latest --display vnc --age-key ./keys/appwrap-age-key.txt
```

## Features

- **Automatic dependency discovery** — PE analysis, DLL imports, COM objects, VC++ redistributables, .NET, DirectX, registry keys, fonts, services
- **Installed app browser** — scans Windows registry, Start Menu shortcuts, and App Paths to list all installed applications
- **Multiple build strategies** — Wine (Linux), Windows Server Core, Windows Nano Server
- **Container encryption** — Age-based file encryption with runtime decryption to tmpfs (files never touch disk)
- **Built-in firewall** — iptables rules with default-deny/allow policies, per-port/IP whitelisting
- **VPN integration** — WireGuard configs baked into containers with kill switch support
- **Three interfaces** — CLI, interactive terminal UI (TUI), and browser-based Web UI
- **Real-time streaming** — live progress, build logs, and events across all interfaces
- **Self-bootstrapping** — `appwrap setup` installs Docker, WSL2, Age, and WireGuard automatically

## Quick Start

### Prerequisites

- Windows 10/11 with WSL2
- Docker Desktop
- Go 1.22+ (to build from source)

### Install

```bash
git clone https://github.com/AES256Afro/appwrap.git
cd appwrap
go build -o appwrap.exe .
```

### Scan an App

```bash
# Manual path
.\appwrap.exe scan "C:\Program Files\Mozilla Firefox\firefox.exe" -o firefox-profile.yaml

# Or browse installed apps interactively
.\appwrap.exe ui
# Press [S] for Scan, then [Tab] to browse installed apps
```

### Build a Container

```bash
.\appwrap.exe build firefox-profile.yaml -t firefox:latest
```

### Run It

```bash
.\appwrap.exe run firefox:latest --display vnc
# Connect via VNC viewer on port 5901
```

## Commands

| Command | Description |
|---------|-------------|
| `appwrap scan <exe>` | Discover dependencies and generate a container profile |
| `appwrap build <profile>` | Build a Docker image from a profile |
| `appwrap run <image>` | Start a containerized application |
| `appwrap inspect <exe>` | Analyze a PE binary (architecture, imports) |
| `appwrap keygen` | Generate Age encryption keypair |
| `appwrap setup` | Check and install dependencies (Docker, WSL2, Age, WireGuard) |
| `appwrap ui` | Launch the interactive terminal UI |
| `appwrap serve` | Start the browser-based Web UI |
| `appwrap version` | Print version |

## Scan Options

```bash
appwrap scan <exe-or-lnk> [flags]

Flags:
  -o, --output <path>      Output profile path (default: <app>-profile.yaml)
      --format <fmt>       Output format: yaml, json (default: yaml)
      --strategy <name>    Build strategy: wine, windows-servercore, windows-nanoserver
      --encrypt            Enable Age encryption in the container
      --firewall <policy>  Firewall default policy: deny or allow
      --vpn <path>         Path to WireGuard .conf file
  -v, --verbose            Verbose output
```

## Build Options

```bash
appwrap build <profile> [flags]

Flags:
  -t, --tag <tag>          Image tag (default: <app>:latest)
      --no-cache           Build without Docker cache
      --generate <dir>     Generate Dockerfile + context without building
  -v, --verbose            Verbose output
```

## Run Options

```bash
appwrap run <image> [flags]

Flags:
      --display <mode>     Display: none, vnc, novnc, rdp (default: none)
  -d, --detach             Run in background
      --rm                 Remove container after exit (default: true)
      --name <name>        Container name
      --profile <path>     Profile path for security features
      --age-key <path>     Path to Age identity file
      --passphrase <pass>  Passphrase for encrypted containers
  -v, --verbose            Verbose output
```

## Security

### Encryption

AppWrap uses [Age](https://github.com/FiloSottile/age) for file-level encryption. App files are encrypted at build time and decrypted at container startup into a tmpfs mount (RAM-only — files never touch disk).

```bash
# Generate keys
appwrap keygen -o ./keys

# Scan with encryption enabled
appwrap scan app.exe --encrypt -o app-profile.yaml
# Edit profile to add recipient: security.encryption.recipient = "age1..."

# Run with decryption key
appwrap run myapp:latest --age-key ./keys/appwrap-age-key.txt
```

### Firewall

Built-in iptables firewall with default-deny or default-allow policies. Rules are baked into the container at build time.

```yaml
# In profile
security:
  firewall:
    enabled: true
    defaultPolicy: "deny"
    allowRules:
      - ip: "1.1.1.1"
        port: 443
        protocol: "tcp"
    allowDNS: true
    allowLoopback: true
```

### VPN (WireGuard)

Bake a WireGuard configuration into the container. All app traffic routes through the VPN tunnel.

```bash
appwrap scan app.exe --vpn /path/to/wg0.conf
```

With kill switch enabled, traffic is blocked if the VPN connection drops:

```yaml
security:
  vpn:
    enabled: true
    provider: "wireguard"
    configFile: "/path/to/wg0.conf"
    killSwitch: true
```

### Security Execution Order

At container startup, security features execute in this order:
1. **VPN** — WireGuard tunnel established
2. **Firewall** — iptables rules applied
3. **Decrypt** — Files decrypted to tmpfs
4. **Application** — App launches

## Interfaces

### Terminal UI

```bash
appwrap ui
```

Full-featured interactive interface with keyboard navigation:

- **Dashboard** — Docker status, quick actions, recent profiles
- **Scan** — Multi-step wizard with app browser (Tab to browse installed apps)
- **Build** — Live Docker build log streaming
- **Run** — Display mode selector, security options
- **Inspect** — PE binary analysis
- **Keygen** — Age key generation
- **Profiles** — Browse, view, delete profiles
- **Containers** — List, stop, remove, view logs
- **Setup** — Check/install dependencies

Keyboard: `S`can `B`uild `R`un `I`nspect `K`eygen `P`rofiles `C`ontainers | `q` quit | `esc` back

### Web UI

```bash
appwrap serve -p 8080
# Open http://localhost:8080
```

Browser-based interface with:

- Dark theme with Docker blue accent
- Real-time WebSocket event streaming
- Browse Installed Apps modal with search/filter
- Log panels with color-coded event types
- Responsive layout

### REST API

All operations are available via REST:

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/status` | Docker status and version |
| `GET` | `/api/apps` | List installed Windows applications |
| `GET` | `/api/profiles` | List profiles |
| `GET` | `/api/profiles/{name}` | Get profile details |
| `DELETE` | `/api/profiles/{name}` | Delete profile |
| `POST` | `/api/scan` | Start scan (returns operationId) |
| `POST` | `/api/build` | Start build (returns operationId) |
| `POST` | `/api/run` | Run container |
| `GET` | `/api/inspect?target=<path>` | Inspect binary |
| `POST` | `/api/keygen` | Generate keys |
| `GET` | `/api/containers` | List containers |
| `POST` | `/api/containers/{id}/stop` | Stop container |

Long-running operations (scan, build) return an `operationId`. Connect to `ws://host/ws/events/{operationId}` for real-time event streaming.

## Profile Format

AppWrap profiles are YAML (or JSON) files describing everything needed to containerize an app:

```yaml
schemaVersion: "1.0"
app:
  name: "Firefox"
  version: "128.0"
  publisher: "Mozilla"

binary:
  path: "firefox.exe"
  arch: "x64"
  subsystem: "gui"

dependencies:
  dlls:
    - name: "mozglue.dll"
      isSystem: false
    - name: "kernel32.dll"
      isSystem: true
  vcredist: ["vc2015-2022"]

network:
  ports: [80, 443]
  protocols: ["tcp"]

display:
  width: 1920
  height: 1080
  gpu: false
  audio: true

security:
  encryption:
    enabled: true
    recipient: "age1abc123..."
  firewall:
    enabled: true
    defaultPolicy: "deny"
    allowDNS: true
  vpn:
    enabled: true
    provider: "wireguard"
    killSwitch: true

build:
  strategy: "wine"
```

## App Discovery

AppWrap discovers installed applications from multiple sources:

- **Windows Registry** — HKLM and HKCU Uninstall keys + WOW6432Node
- **App Paths** — `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths`
- **Start Menu** — Resolves `.lnk` shortcuts to exe paths
- **Well-known paths** — Checks Program Files, Program Files (x86), and LocalAppData

Access via:
- CLI: `appwrap scan` then Tab in TUI, or `appwrap ui` > Scan > Tab
- Web UI: "Browse Installed Apps" button on Scan page
- API: `GET /api/apps`

## Architecture

```
appwrap/
  cmd/              CLI commands (Cobra)
  internal/
    builder/        Dockerfile generation + file staging
    discovery/      App dependency scanning
      static/       PE analyzer, manifest parser, shortcut resolver
    profile/        Profile serialization (YAML/JSON)
    runtime/        Docker CLI wrapper
    security/       Encryption, firewall, VPN generators
    service/        Shared backend (CLI, TUI, Web all use this)
    tui/            Terminal UI (Bubble Tea)
    web/            Web server + REST API + WebSocket hub
      static/       Embedded SPA (HTML/CSS/JS)
    util/           System DLL lists, helpers
  configs/          Known system DLLs list
```

## Tech Stack

- **Go 1.22+** with embedded static files
- **Cobra** — CLI framework
- **Bubble Tea + Lipgloss** — Terminal UI
- **nhooyr.io/websocket** — WebSocket for real-time events
- **saferwall/pe** — PE file parsing
- **golang.org/x/sys** — Windows registry access
- **Docker CLI** — Container builds and execution
- **Age** — File encryption
- **WireGuard** — VPN tunneling
- **iptables** — Container firewall

## License

MIT

## Author

[@TheEncryptedAFro](https://github.com/theencryptedafro)
