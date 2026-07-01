# CapAuto — Captive Portal Automator

**CapAuto** is a fast, cross-platform, single-binary tool that automatically detects captive portal WiFi networks, parses the login form, and logs you in using saved credentials.

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
capauto install
```

### First Use

1. Connect to a WiFi network with a captive portal
2. Run `capauto` — it will detect the portal and prompt for your credentials
3. Credentials are saved securely; future connections are automatic

### Commands

```bash
capauto                     # Run once: detect portal → login
capauto --debug             # Verbose output
capauto --dry-run           # Detect portal but don't submit

capauto creds list          # Show saved SSIDs
capauto creds add <SSID>    # Add credentials for a network
capauto creds remove <SSID> # Remove credentials

capauto install             # Install as background service
capauto uninstall           # Remove background service
capauto status              # Show Wi-Fi status
capauto migrate             # Migrate v1 plaintext config
capauto version             # Show version
```

## Upgrading from v1 (Node.js)

If you previously used the Node.js version, CapAuto v2 automatically migrates your saved credentials from `~/.capauto/config.json` to the OS keychain on first run.

You can also migrate manually:

```bash
capauto migrate
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

```bash
# Requirements: Go 1.21+

# Build for current platform
make build

# Run tests
make test

# Cross-compile for all platforms
make cross-compile
# Output: build/capauto-{darwin,linux,windows}-{amd64,arm64}
```

## Security

- Credentials stored in OS keychain by default
- Fallback: JSON file with `0600` permissions at `~/.capauto/credentials.json`
- Config directories created with `0700` permissions
- Passwords **never** appear in logs at any level
- TLS verification disabled only for captive portal connections (self-signed certs are common)
- Use `--insecure-store` flag to explicitly opt into plaintext storage

## Architecture

```
cmd/capauto/main.go          → CLI entry point & orchestrator
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
