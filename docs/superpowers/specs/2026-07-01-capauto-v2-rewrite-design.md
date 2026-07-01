# Design Specification: CapAuto v2 — Ground-Up Rewrite in Go

**Date:** 2026-07-01
**Status:** Approved
**Author:** Antigravity (AI Partner)

---

## 1. Objective

Rewrite the Dynamic Captive Portal Automator (CapAuto) from a monolithic 542-line
Node.js script into a modular, cross-platform Go application that compiles to a
single native binary. The rewrite addresses every major weakness of v1:

| v1 Problem | v2 Solution |
|---|---|
| 542-line monolith | 7 focused packages, ~80-120 lines each |
| Regex HTML parsing | Proper DOM tree parser (`golang.org/x/net/html`) |
| Plaintext credentials | OS keychain (macOS/Windows/Linux) + encrypted fallback |
| Node.js runtime dependency | Single compiled binary — no runtime required |
| "Find your node path" launchd friction | `capauto install` handles everything |
| `console.log` everywhere | Structured logging with levels and rotation |
| Hardcoded 2s retry delay | Exponential backoff with jitter |
| macOS + Windows only | macOS + Windows + Linux |

---

## 2. Requirements & Constraints

- **Single binary deployment:** `go build` produces one executable. No runtime, no `npm install`, no PATH issues.
- **Zero-config for end users:** First run detects portal, prompts for credentials, saves to keychain, done.
- **Cross-platform:** macOS (13+), Windows (10/11), Linux (any modern distro with NetworkManager or systemd-networkd).
- **Portal coverage:** All common HTML-based captive portals — universities, hotels, cafes, airports. Not 802.1X/WPA-Enterprise.
- **Backward compatible:** Migrates existing `~/.capauto/config.json` credentials to keychain on first run.
- **Secure by default:** Credentials in OS keychain; encrypted JSON fallback for headless/SSH; plaintext only via explicit `--insecure-store` flag.

---

## 3. Language Choice: Go

| Factor | Go | Rust | Python | Swift |
|---|---|---|---|---|
| Single binary | ✅ | ✅ | ❌ | ❌ macOS only |
| Cross-platform | ✅ trivial cross-compile | ✅ | ✅ | ❌ |
| HTTP stdlib | ✅ best-in-class | 🟡 needs crates | ✅ | 🟡 |
| Keychain libs | ✅ go-keyring | ✅ | 🟡 | ✅ native |
| Daemon suitability | ✅ tiny memory, fast start | ✅ | ❌ heavy | ✅ |
| HTML parsing | ✅ `x/net/html` | ✅ | ✅ | 🟡 |
| Maintainability | ✅ readable, small lang | 🟡 steep curve | ✅ | 🟡 |

---

## 4. Architecture

```
┌───────────────────────────────────────────────────────────────────────┐
│                           capauto CLI                                 │
│                                                                       │
│  cmd/capauto/main.go           ← entry point, flag parsing           │
│                                                                       │
│  ┌──────────┐  ┌───────────┐  ┌───────────┐  ┌──────────┐  ┌───────┐│
│  │ network   │→│  prober    │→│  portal    │→│  auth     │→│ verify ││
│  │           │  │           │  │           │  │          │  │       ││
│  │ SSID      │  │ Parallel  │  │ DOM-based │  │ Cookie-  │  │ Re-   ││
│  │ detect    │  │ multi-    │  │ form      │  │ aware    │  │ probe ││
│  │ per-OS    │  │ endpoint  │  │ parser    │  │ submit   │  │ to    ││
│  │           │  │ probing   │  │           │  │          │  │ verify││
│  └──────────┘  └───────────┘  └───────────┘  └──────────┘  └───────┘│
│                                                                       │
│  ┌──────────────────────────────┐  ┌──────────────────────────────┐  │
│  │ internal/credential          │  │ internal/log                 │  │
│  │                              │  │                              │  │
│  │ • Keychain read/write        │  │ • Structured JSON logging    │  │
│  │ • Encrypted JSON fallback    │  │ • Levels: debug/info/warn    │  │
│  │ • Per-SSID credential store  │  │ • File rotation support      │  │
│  └──────────────────────────────┘  └──────────────────────────────┘  │
└───────────────────────────────────────────────────────────────────────┘
```

### 4.1 Pipeline Flow

1. **Network** → Detect current SSID via OS-specific commands
2. **Prober** → Fire parallel HTTP probes to determine online/portal status
3. **Portal** → Fetch portal page, parse DOM to extract form structure
4. **Auth** → Load credentials (keychain → encrypted file → prompt), build payload, submit
5. **Verify** → Re-probe to confirm internet access is restored

---

## 5. Project Layout

```
captive-portal-automator/
├── cmd/
│   └── capauto/
│       └── main.go                  # CLI entry, flags, orchestrator
├── internal/
│   ├── network/
│   │   ├── ssid.go                  # SSID detection (macOS/Windows/Linux)
│   │   └── ssid_test.go
│   ├── prober/
│   │   ├── prober.go                # Connectivity probing (multi-endpoint)
│   │   ├── endpoints.go             # Probe endpoint definitions
│   │   └── prober_test.go
│   ├── portal/
│   │   ├── parser.go                # DOM-based HTML form parser
│   │   ├── extractor.go             # Portal URL extraction (redirect/meta/JS)
│   │   └── parser_test.go
│   ├── auth/
│   │   ├── submitter.go             # HTTP form submission with cookie jar
│   │   └── submitter_test.go
│   ├── credential/
│   │   ├── store.go                 # Store interface definition
│   │   ├── keychain.go              # macOS Keychain + Windows Cred Mgr
│   │   ├── secret_service.go        # Linux Secret Service (D-Bus)
│   │   ├── file.go                  # AES-256-GCM encrypted JSON fallback
│   │   ├── migrate.go               # v1 plaintext → v2 keychain migration
│   │   └── store_test.go
│   └── log/
│       ├── logger.go                # Structured logger (JSON + human)
│       └── logger_test.go
├── install/
│   ├── com.sharzil.capauto.plist    # macOS LaunchAgent template
│   ├── capauto.service              # Linux systemd user unit
│   ├── 99-capauto                   # Linux NetworkManager dispatcher
│   ├── install_macos.sh             # macOS install script
│   ├── install_linux.sh             # Linux install script
│   └── install_windows.ps1          # Windows Task Scheduler setup
├── docs/
│   └── ...
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 6. Package Designs

### 6.1 `internal/network` — SSID Detection

```go
// GetSSID returns the current Wi-Fi network SSID.
// Returns ("", ErrNoWiFi) if not connected to Wi-Fi.
func GetSSID() (string, error)
```

**Per-platform implementation:**

- **macOS:**
  1. `networksetup -listallhardwareports` → find Wi-Fi device name
  2. `networksetup -getairportnetwork <device>` → parse SSID
  3. Fallback: iterate all `en*` interfaces
  4. Fallback: `system_profiler SPAirPortDataType` (slow but reliable on Sonoma+)

- **Windows:**
  1. `netsh wlan show interfaces` → parse SSID line

- **Linux:**
  1. `nmcli -t -f active,ssid dev wifi` → find active SSID
  2. Fallback: `iwgetid -r`

Uses Go build tags (`//go:build darwin`, `//go:build windows`, `//go:build linux`)
for clean per-platform files instead of runtime `if/else`.

### 6.2 `internal/prober` — Connectivity Probing

```go
type ProbeResult struct {
    Online    bool
    PortalURL string   // empty if online
    HTML      string   // raw portal page HTML
    FinalURL  string   // URL after following all redirects
}

type Prober struct {
    client    *http.Client
    endpoints []Endpoint
    logger    *log.Logger
}

// Check fires probes in parallel. First conclusive result wins.
func (p *Prober) Check(ctx context.Context) (ProbeResult, error)
```

**Probe endpoints (default, configurable):**

| Endpoint | Success Signal |
|---|---|
| `http://captive.apple.com/hotspot-detect.html` | Body contains `<BODY>Success</BODY>` |
| `http://connectivitycheck.gstatic.com/generate_204` | Status 204 |
| `http://www.msftconnecttest.com/connecttest.txt` | Body equals `Microsoft Connect Test` |
| `http://detectportal.firefox.com/canonical.html` | Body equals `success\n` |
| `http://neverssl.com/` | Not redirected |

**Probing strategy:**
1. Fire the first 2 endpoints concurrently
2. If both return "online" → done
3. If either returns a portal redirect URL → done (use that URL)
4. If inconclusive (offline but no redirect) → retry with exponential backoff:
   `500ms → 1s → 2s → 4s` with ±25% jitter, max 3 retries
5. On each retry, add 1 more endpoint from the list

### 6.3 `internal/portal` — DOM-Based Form Parser

```go
type FormData struct {
    Action        string
    Method        string
    Fields        map[string]string  // all fields with default values
    UsernameField string
    PasswordField string
    FormIndex     int
}

func ParseLoginForm(body io.Reader, baseURL string) (*FormData, error)
func ExtractPortalURL(body io.Reader, baseURL string) (string, error)
```

**Parser algorithm:**
1. Parse HTML via `golang.org/x/net/html` into a node tree
2. Walk tree to find all `<form>` elements
3. For each form, collect all `<input>`, `<select>`, `<textarea>` children
4. Score each form:
   - Has `<input type="password">` → +100
   - Has text/email input with username-like name → +50
   - Has `method="POST"` → +10
   - Is first form with inputs → +5
5. Pick highest-scoring form
6. Username field detection (priority order):
   - Name/id contains: `user`, `login`, `email`, `phone`, `mobile`, `member`, `account`, `id`
   - First text/email input that isn't the password field
7. Resolve form `action` against `baseURL` to get absolute URL

**Portal URL extraction** (for pages that don't redirect but inject the portal URL):
1. `<meta http-equiv="refresh" content="0;url=...">` → parse content attr
2. `<form action="...">` → resolve against base
3. `<a href="...">` → first link to a different host
4. `<script>` text → regex for `location.href = "..."` or `location.replace("...")`

### 6.4 `internal/auth` — Form Submission

```go
type Submitter struct {
    client *http.Client  // shared with prober (same cookie jar)
    logger *log.Logger
}

// Submit sends the login form with credentials.
func (s *Submitter) Submit(ctx context.Context, form *portal.FormData, creds *credential.Credentials) error

// Verify re-probes to confirm internet access.
func (s *Submitter) Verify(ctx context.Context, prober *prober.Prober) (bool, error)
```

**Key design point:** The `http.Client` is created once in `main.go` and shared across
prober, portal fetcher, and submitter. Go's `http.Client` has a built-in `CookieJar`
(`net/http/cookiejar`), so session cookies from the initial probe/portal-fetch
automatically carry through to the login submission.

**Submit flow:**
1. Build payload: merge `form.Fields` + `creds.Username` + `creds.Password`
2. If POST: URL-encode body, set `Content-Type: application/x-www-form-urlencoded`
3. If GET: append params to URL query string
4. Send with shared cookie jar, follow redirects
5. Wait 2.5s, then call `Verify()` to re-probe

### 6.5 `internal/credential` — Secure Storage

```go
// Credentials holds login info for one SSID.
type Credentials struct {
    SSID          string
    Username      string
    Password      string
    UsernameField string
    PasswordField string
    FormAction    string
    FormMethod    string
    StaticFields  map[string]string
    UpdatedAt     time.Time
}

// Store is the credential storage interface.
type Store interface {
    Load(ssid string) (*Credentials, error)
    Save(creds *Credentials) error
    Delete(ssid string) error
    List() ([]string, error)
}
```

**Backend priority:**

| # | Backend | When Used | Implementation |
|---|---------|-----------|----------------|
| 1 | OS Keychain | Default on desktop | `go-keyring` library |
| 2 | Encrypted JSON | Headless/SSH, keychain unavail | AES-256-GCM, key from machine-ID + user-ID via PBKDF2 |
| 3 | Plaintext JSON | `--insecure-store` flag only | Warns on every use |

**Keychain storage format:**
- Service: `capauto`
- Account: `<SSID>`
- Password: JSON-encoded `{username, password}`

**Form metadata** (field names, static fields, action URL) stored separately in
`~/.capauto/config.json` as non-secret data. Only actual credentials go to keychain.

**v1 Migration (`migrate.go`):**
1. On first run, check `~/.capauto/config.json` for old format
2. Old format detection: SSID entries contain `username`/`password` keys directly
3. For each entry: save credentials to keychain, strip passwords from JSON
4. Log migration action

### 6.6 `internal/log` — Structured Logging

```go
type Level int
const (
    Debug Level = iota
    Info
    Warn
    Error
)

type Logger struct {
    level  Level
    output io.Writer
    format Format   // Human (colored) or JSON
}

func (l *Logger) Debug(msg string, fields ...Field)
func (l *Logger) Info(msg string, fields ...Field)
func (l *Logger) Warn(msg string, fields ...Field)
func (l *Logger) Error(msg string, fields ...Field)
```

- **CLI mode:** Human-readable colored output to stdout. Default level: Info.
  `--debug` flag sets level to Debug.
- **Daemon mode:** JSON lines to `~/.capauto/capauto.log`. Automatic rotation:
  keep last 5 files, 1MB max each.
- All log entries include: timestamp, SSID, operation phase.

---

## 7. CLI Interface

```bash
# Core operations
capauto                       # Detect portal → login (primary use case)
capauto --debug               # Verbose output with portal HTML dump
capauto --dry-run             # Detect and report but don't submit
capauto --insecure-store      # Force plaintext credential storage

# Credential management
capauto creds list            # Show all stored SSIDs (never shows passwords)
capauto creds add <SSID>      # Interactively add credentials for an SSID
capauto creds remove <SSID>   # Remove credentials for an SSID
capauto creds test <SSID>     # Test credentials against current portal

# Daemon lifecycle
capauto install               # Install as background service (auto-detects OS)
capauto uninstall             # Remove background service
capauto status                # Show daemon status + last run log

# Migration
capauto migrate               # Migrate v1 plaintext config → keychain
```

---

## 8. Daemon Setup per OS

### 8.1 macOS — launchd

`capauto install` on macOS:
1. Copy binary to `/usr/local/bin/capauto`
2. Generate plist from template (using actual binary path)
3. Copy to `~/Library/LaunchAgents/com.sharzil.capauto.plist`
4. Run `launchctl bootstrap gui/$(id -u) <plist_path>`

Plist watches `/var/run/resolv.conf` and
`/Library/Preferences/SystemConfiguration/com.apple.wifi.message-tracer.plist`
for network state changes. RunAtLoad = true.

### 8.2 Linux — systemd + NetworkManager

`capauto install` on Linux:
1. Copy binary to `/usr/local/bin/capauto`
2. Install systemd user unit to `~/.config/systemd/user/capauto.service`
3. Install NetworkManager dispatcher to `/etc/NetworkManager/dispatcher.d/99-capauto`
4. Run `systemctl --user enable capauto.service`

### 8.3 Windows — Task Scheduler

`capauto install` on Windows:
1. Copy binary to `%LOCALAPPDATA%\capauto\capauto.exe`
2. Create scheduled task via `schtasks.exe`:
   - Trigger: Event ID 10000 from `Microsoft-Windows-NetworkProfile/Operational`
   - Action: Start program → `capauto.exe`
   - Run hidden, no console window

---

## 9. Testing Strategy

### 9.1 Unit Tests (per package)

- `network/ssid_test.go` — Mock `exec.Command` to simulate OS-specific output
- `prober/prober_test.go` — Use `httptest.Server` to simulate probe responses
- `portal/parser_test.go` — Test with real-world portal HTML fixtures (saved from actual portals)
- `auth/submitter_test.go` — Use `httptest.Server` to verify form submission
- `credential/store_test.go` — Test against in-memory Store mock

### 9.2 Integration Tests

- `MockPortal` test server (ported from current `mock-portal.js` to Go's `httptest`)
- Full pipeline test: offline → detect portal → parse form → submit → verify online
- Tests both POST and GET form methods
- Tests cookie jar carry-through
- Tests credential migration from v1 format

### 9.3 Coverage Target

- 80%+ line coverage across all packages
- Critical paths (prober, parser, credential store) at 90%+

---

## 10. Build & Distribution

### 10.1 Makefile Targets

```makefile
build:          # Build for current OS/arch
test:           # Run all tests with coverage
lint:           # golangci-lint
install:        # Build + install to /usr/local/bin
cross-compile:  # Build for darwin/amd64, darwin/arm64, linux/amd64, windows/amd64
release:        # Cross-compile + create archives
```

### 10.2 Dependencies (minimal)

| Dependency | Purpose |
|---|---|
| `golang.org/x/net/html` | DOM-based HTML parser |
| `github.com/zalando/go-keyring` | OS keychain access (macOS/Windows/Linux) |
| Standard library only | Everything else: HTTP, JSON, crypto, logging |

---

## 11. Error Handling

- All functions return `(result, error)` — no panics, no silent swallowing
- Errors are wrapped with context: `fmt.Errorf("prober: check endpoint %s: %w", url, err)`
- User-facing errors are logged at Error level with actionable messages
- Debug-level logging captures full HTTP responses, headers, and timing

---

## 12. Security Considerations

- **No hardcoded secrets** — all credentials go through the Store interface
- **TLS verification** — disabled only for captive portal connections (portals use self-signed certs); re-enabled for verification probes
- **Input validation** — SSID sanitized before use as keychain key; form fields validated before submission
- **File permissions** — config JSON created with 0600; config directory with 0700
- **No credential logging** — passwords never appear in logs at any level
