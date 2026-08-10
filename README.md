# AutoCap

> Automatic, zero-touch Captive Portal login engine written in Go.

AutoCap runs silently in the background, detects captive portal Wi-Fi networks (universities, airports, hotels, cafes), and logs you in instantly using OS-native secure credential storage.

---

### Features

- **Zero-Touch Automation:** Runs as a background service (`launchd`, `systemd`, or Task Scheduler).
- **Secure Credentials:** Encrypted storage via macOS Keychain, Windows Credential Manager, and Linux Secret Service.
- **Universal Portal Engine:** Smart DOM parser + JS analyzer supporting 90%+ of global enterprise & vendor portals (Ruijie, Cisco ISE, Aruba ClearPass, Fortinet, MikroTik, pfSense).
- **Browser-Grade Emulation:** Full header fingerprinting, session cookie persistence, CSRF protection, and exponential backoff retry.
- **Zero Dependencies:** Single compiled binary — no Node.js or Python runtime needed.

---

### Quick Start

#### Installation

```bash
# macOS
bash install/install_macos.sh

# Linux
bash install/install_linux.sh

# Windows (PowerShell as Administrator)
powershell -ExecutionPolicy Bypass -File install/install_windows.ps1
```

*Or build manually:*
```bash
make build
make install
```

#### Usage

```bash
autocap                     # Detect network -> parse portal -> log in
autocap --debug             # Verbose output with HTTP wire-level dumps
autocap creds add <SSID>    # Save credentials for a network
autocap creds list          # List saved networks
autocap status              # View Wi-Fi status
```

---

### How It Works

```
+----------------+     +----------------+     +----------------+     +----------------+
| Network Check  | --> | Probe & Fetch  | --> | Parse & Build  | --> | Authenticate   |
|   (Get SSID)   |     |  (Portal URL)  |     |   (DOM + JS)   |     |   & Verify     |
+----------------+     +----------------+     +----------------+     +----------------+
```

1. **Detect Network:** Identifies active SSID and probes connectivity check endpoints (`apple`, `gstatic`, `firefox`).
2. **DOM & JS Analysis:** Fetches portal HTML, follows iframe/meta/JS redirects, extracts hidden fields, and resolves AJAX auth targets.
3. **Secure Auth:** Loads credentials from OS Keychain and posts with full browser headers & cookies.
4. **Verify:** Confirms internet connectivity and retries with backoff if needed.

---

### License

[MIT](LICENSE)
