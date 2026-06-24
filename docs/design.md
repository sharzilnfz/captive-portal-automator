# Design Specification: Dynamic Captive Portal Automator (CapAuto)

**Date:** 2026-06-24  
**Status:** Proposed  
**Author:** Antigravity (AI Partner)  

---

## 1. Objective

To build a lightweight, robust, and zero-dependency desktop CLI tool (**CapAuto**) for macOS and Windows that automatically detects when the device is behind a captive portal, extracts the login form structure, and logs in using saved credentials for the current Wi-Fi network (SSID).

---

## 2. Requirements & Constraints

*   **Zero External Dependencies:** Must run using native Node.js APIs (v18+) to eliminate `npm install` complexity and compile issues across platforms.
*   **Platform Support:** macOS (13+) and Windows (10/11).
*   **Secure Storage:** Store credentials locally in `~/.capauto/config.json` with user-only read/write permissions (`chmod 600` on macOS).
*   **Background Execution:** Run silently in the background whenever a network state change occurs.
*   **Resiliency:** Automatically handle changes to the captive portal URL or HTML form layout by dynamically parsing the form on every run.

---

## 3. Architecture & Key Modules

CapAuto is organized into four logical phases:

```
+------------------+     +-----------------------+     +----------------------+     +-----------------------+
|  1. Network Check| --> | 2. Connectivity Probe | --> |  3. Portal Analysis  | --> |   4. Form Submission  |
|  (SSID Detection) |     |  (captive.apple.com)  |     | (HTML Form Parsing)  |     |  (HTTP POST & Verify) |
+------------------+     +-----------------------+     +----------------------+     +-----------------------+
```

### 3.1 Network Check
The tool first detects the active Wi-Fi SSID to look up saved credentials.
*   **macOS:** Executes `/usr/sbin/wdutil info` or `networksetup -getmedia wifi` and parses the SSID.
*   **Windows:** Executes `netsh wlan show interfaces` and parses the `SSID` line.

### 3.2 Connectivity Probe
Checks if the device is currently intercepted by a captive portal:
*   Sends an HTTP GET request to `http://captive.apple.com/hotspot-detect.html`.
*   If the response status is `200 OK` and the body is exactly `<HTML><HEAD><TITLE>Success</TITLE></HEAD><BODY>Success</BODY></HTML>`, the device is fully online. The script exits silently.
*   If the response redirects (status `302`/`307` or `Location` header present) or returns different HTML, we are behind a portal. The script captures the redirect URL.

### 3.3 Portal Analysis
Fetches the HTML of the redirect URL and parses it to locate the login form:
*   Scans the HTML for `<form>` elements.
*   Identifies the target form `action` URL (resolving relative paths to absolute URLs).
*   Scans for input fields:
    *   **Password Field:** An `<input>` with `type="password"`.
    *   **Username Field:** An `<input>` with `type="text"`, `type="email"`, or name/id containing `user`, `login`, `username`, `phone`, `member`.
    *   **Hidden & Helper Fields:** Captures all `<input type="hidden">` and default values (like CSRF tokens or session IDs) to ensure the request is valid.

### 3.4 Form Submission
*   If credentials for this SSID do not exist, runs an interactive terminal prompt to collect the username and password, then saves them to `~/.capauto/config.json`.
*   Constructs a URL-encoded form payload containing the credentials and all hidden/helper fields.
*   Submits the payload via HTTP POST to the form `action` URL.
*   Probes `captive.apple.com` again to verify internet access is restored.

---

## 4. File Layout

```
/Users/sharzilnafis/Desktop/Project/captive-portal-automator/
├── index.js          # Main entrypoint containing detection, parsing, and submission logic
├── package.json      # Node.js project manifest (minimal metadata)
├── README.md         # Documentation, manual usage, and background setup instructions
└── docs/
    └── design.md     # This design specification
```

---

## 5. Background Execution Setup

To ensure the script runs automatically on network changes:

### 5.1 macOS Launch Agent (`~/Library/LaunchAgents/com.sharzil.capauto.plist`)
Monitors the network configuration path. Whenever a network change occurs, macOS triggers the script.
```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.sharzil.capauto</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/node</string>
        <string>/Users/sharzilnafis/Desktop/Project/captive-portal-automator/index.js</string>
    </array>
    <key>WatchPaths</key>
    <array>
        <string>/var/run/resolv.conf</string>
        <string>/Library/Preferences/SystemConfiguration/com.apple.wifi.message-tracer.plist</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/Users/sharzilnafis/Desktop/Project/captive-portal-automator/capauto.log</string>
    <key>StandardErrorPath</key>
    <string>/Users/sharzilnafis/Desktop/Project/captive-portal-automator/capauto.err</string>
</dict>
</plist>
```

### 5.2 Windows Task Scheduler
Creates a task triggered by Event Viewer:
*   **Log:** `Microsoft-Windows-NetworkProfile/Operational`
*   **Source:** `NetworkProfile`
*   **Event ID:** `10000` (Connected to a network)
*   **Action:** Start a Program -> `node.exe` with arguments `C:\Users\sharzilnafis\Desktop\Project\captive-portal-automator\index.js`.

---

## 6. Security & Error Handling

*   **Credential Protection:** Stored in a JSON file at `~/.capauto/config.json` with permissions restricted to the owner (`chmod 600`).
*   **Error Logging:** Appends detailed timestamps and failure reasons to `capauto.err` for troubleshooting.
*   **Network Timeouts:** All fetch requests use a 10-second timeout to prevent the script from hanging on bad networks.
