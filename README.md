# AutoCap — Captive Portal Automator

**AutoCap** is a fast, cross-platform, single-binary tool that automatically detects captive portal WiFi networks, parses the login form, and logs you in using saved credentials.

It runs silently in the background and handles university, hotel, cafe, and airport portals — no more clicking through login pages every time you reconnect.

## Features

- **Single binary** — no runtime dependencies (Node.js, Python, etc.)
- **Cross-platform** — macOS, Windows, Linux
- **Secure credentials** — OS keychain (macOS Keychain, Windows Credential Manager, Linux Secret Service)
- **Smart form parsing** — proper HTML DOM parser (`golang.org/x/net/html`), not fragile regex
- **Auto-detect portals** — probes multiple connectivity endpoints (Apple, Google, Microsoft, Firefox)
- **Background daemon** — installs as a system service triggered by network changes
- **Exponential backoff** — retries with jitter instead of fixed delays
- **Credential management** — CLI subcommands to list, add, and remove saved credentials
- **V1 migration** — automatically migrates from the old Node.js version

## Quick Start

### Install

```bash
# Build from source (requires Go 1.21+)
make build

# Install to /usr/local/bin and set up background service
make install
autocap install
```

### First Use

1. Connect to a WiFi network with a captive portal
2. Run `autocap` — it will detect the portal and prompt for your credentials
3. Credentials are saved securely; future connections are automatic

### Commands

```bash
autocap                     # Run once: detect portal → login
autocap --debug             # Verbose output
autocap --dry-run           # Detect portal but don't submit

autocap creds list          # Show saved SSIDs
autocap creds add <SSID>    # Add credentials for a network
autocap creds remove <SSID> # Remove credentials

autocap install             # Install as background service
autocap uninstall           # Remove background service
autocap status              # Show Wi-Fi status
autocap migrate             # Migrate v1 plaintext config
autocap version             # Show version
```

## Upgrading from v1 (Node.js)

If you previously used the Node.js version, AutoCap v2 automatically migrates your saved credentials from `~/.autocap/config.json` to the OS keychain on first run.

You can also migrate manually:

```bash
autocap migrate
```

After migration, you can safely remove the old Node.js files (`index.js`, `package.json`, `node_modules/`).

## Background Service

### macOS

```bash
bash install/install_macos.sh
```

Uses `launchd` to trigger on network changes via `WatchPaths`.

### Linux

```bash
bash install/install_linux.sh
```

Uses `systemd` user service + optional NetworkManager dispatcher hook.

### Windows

Use Task Scheduler with a "Log on" trigger or network event trigger (Event ID 10000).

## Building from Source

### ⚡ Automated Installation (Recommended)

If you don't have Go installed, our platform-specific installation scripts will **automatically** attempt to detect, download, and install the correct Go version for you:

- **macOS:** Installs via Homebrew (`brew install go`)
- **Linux:** Installs via system package manager (`apt`, `dnf`, `yum`, or `pacman`)

```bash
# macOS
bash install/install_macos.sh

# Linux
bash install/install_linux.sh
```

---

### 🛠️ Manual Go Installation Guide

If you prefer to install Go manually, here is how to get **Go 1.21+** set up:

#### 🍎 macOS
```bash
# Option 1: Via Homebrew
brew install go

# Option 2: Installer package (.pkg)
# Download from: https://go.dev/dl/
```

#### 🐧 Linux
```bash
# Ubuntu / Debian
sudo apt-get update && sudo apt-get install -y golang-go

# Fedora / RHEL
sudo dnf install -y golang

# Arch Linux
sudo pacman -S --noconfirm go
```

#### 🪟 Windows
```bash
# Option 1: Via winget
winget install GoLang.Go

# Option 2: MSI installer
# Download and run the installer from: https://go.dev/dl/
```

Verify Go is installed correctly by running:
```bash
go version
```

---

### 📦 Manual Compilation & Testing

Once Go is installed, compile and test using the Makefile:

```bash
# Build for current platform
make build

# Run tests
make test

# Cross-compile for all platforms
make cross-compile
# Output: build/autocap-{darwin,linux,windows}-{amd64,arm64}
```

## Security

- Credentials stored in OS keychain by default
- Fallback: JSON file with `0600` permissions at `~/.autocap/credentials.json`
- Config directories created with `0700` permissions
- Passwords **never** appear in logs at any level
- TLS verification disabled only for captive portal connections (self-signed certs are common)
- Use `--insecure-store` flag to explicitly opt into plaintext storage

## Architecture

```
cmd/autocap/main.go          → CLI entry point & orchestrator
internal/
  log/                       → Structured logging with rotation (wraps slog)
  network/                   → SSID detection (macOS/Windows/Linux)
  prober/                    → Multi-endpoint connectivity probing
  portal/                    → DOM-based HTML form parser
  auth/                      → Form submission & verification
  credential/                → Keychain + file credential store + v1 migration
install/                     → Platform-specific service configs & install scripts
```

### How It Works

```
┌─────────────┐    ┌─────────┐    ┌──────────┐    ┌──────────┐    ┌────────┐
│  Network    │───►│ Prober  │───►│  Portal  │───►│  Auth    │───►│ Verify │
│  (SSID)     │    │ (probe) │    │ (parse)  │    │ (submit) │    │(re-probe)│
└─────────────┘    └─────────┘    └──────────┘    └──────────┘    └────────┘
                                       │
                                       ▼
                                 ┌──────────┐
                                 │ Credential│
                                 │  Store   │
                                 └──────────┘
```

1. **Detect SSID** — reads current Wi-Fi network name
2. **Probe** — hits connectivity check endpoints to detect captive portal
3. **Parse** — fetches portal page, parses login form using DOM parser
4. **Load creds** — retrieves saved credentials from keychain (or prompts first time)
5. **Submit** — sends login form with credentials
6. **Verify** — re-probes to confirm internet access

## License

MIT
