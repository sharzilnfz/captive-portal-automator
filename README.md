# Dynamic Captive Portal Automator (CapAuto)

CapAuto is a lightweight, robust, and **zero-dependency** Node.js script that automatically detects when your computer is behind a captive portal WiFi network, dynamically parses the login form, and logs you in using saved credentials for that specific Wi-Fi network (SSID).

It works seamlessly on **macOS** and **Windows**, running silently in the background whenever your network connection changes.

---

## Features

*   **Zero Dependencies:** Runs using pure Node.js built-in APIs (v18+). No `npm install` required.
*   **Dynamic Parsing:** Scans the captive portal HTML on-the-fly to detect form fields and submit values (resilient to university portal changes).
*   **Secure Storage:** Saves credentials locally in `~/.capauto/config.json` (macOS) or `%USERPROFILE%\.capauto\config.json` (Windows) with secure owner-only read/write permissions (`0o600`).
*   **Network-Aware:** Stores separate credentials for different Wi-Fi networks (SSIDs) and automatically selects the correct one.
*   **Robust Connectivity Probe:** Probes Apple's connectivity check server (`captive.apple.com`) to determine if it is online or intercepted.

---

## Requirements

*   **Node.js** (version 18 or higher) installed on your machine.

---

## Manual Usage & First-Time Setup

1.  Open your terminal and navigate to the project directory:
    ```bash
    cd "/Users/sharzilnafis/Desktop/Project/captive-portal-automator"
    ```

2.  Run the script:
    ```bash
    node index.js
    ```

3.  **Interactive Setup (First Time on a network):**
    *   If the script detects a captive portal and you haven't configured credentials for this Wi-Fi network (SSID) yet, it will automatically analyze the form fields and prompt you in the terminal:
        ```
        No saved credentials found for SSID: "Uni-WiFi". Initiating first-time setup...
        Enter your Wi-Fi username/ID: student123
        Enter your Wi-Fi password: **********
        ```
    *   It will save your credentials and form metadata securely.
    *   It will then submit the login request and verify you are online.

4.  **Subsequent Connections:**
    *   Running `node index.js` again will check your status. If you are already online, it exits silently. If you are offline behind the portal, it automatically logs you in using your saved credentials without prompting.

---

## Run Automatically in the Background

You can configure your operating system to run the script silently in the background whenever you connect to a Wi-Fi network.

### 1. macOS Setup (Launch Agent)

We use `launchd` to watch network changes. Whenever your Wi-Fi connection updates, macOS will run the script.

1.  Copy the provided `com.sharzil.capauto.plist` configuration to your user's LaunchAgents directory:
    ```bash
    cp com.sharzil.capauto.plist ~/Library/LaunchAgents/com.sharzil.capauto.plist
    ```

2.  Load and activate the Launch Agent:
    ```bash
    launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/com.sharzil.capauto.plist
    ```

3.  **Verify it is running:**
    *   Any output or errors will be written to `capauto.log` and `capauto.err` in the project directory.
    *   To stop/unload the agent in the future:
        ```bash
        launchctl bootout gui/$(id -u) ~/Library/LaunchAgents/com.sharzil.capauto.plist
        ```

*Note: The plist defaults to `/usr/local/bin/node` for the Node.js path. If your Node.js path is different (e.g., using Volta, NVM, or Homebrew), modify the first `<string>` under `<key>ProgramArguments</key>` in `~/Library/LaunchAgents/com.sharzil.capauto.plist` to point to your node executable (e.g., `/Users/sharzilnafis/.volta/bin/node` or `/opt/homebrew/bin/node`).*

---

## 2. Windows Setup (Task Scheduler)

We use Windows Task Scheduler to run the script when a network connection event is logged.

1.  Open **Task Scheduler** (search for it in the Start Menu).
2.  Click **Create Basic Task...** in the Actions panel on the right.
3.  **Name:** `CapAuto Captive Portal Login`
4.  **Trigger:** Select **When a specific event is logged**.
5.  **Event Trigger Details:**
    *   **Log:** `Microsoft-Windows-NetworkProfile/Operational`
    *   **Source:** `NetworkProfile`
    *   **Event ID:** `10000` (This event is logged whenever you successfully connect to a network).
6.  **Action:** Select **Start a program**.
7.  **Program/script:** Enter `node` (or the absolute path to your `node.exe`, e.g., `C:\Program Files\nodejs\node.exe`).
8.  **Add arguments:** Enter the absolute path to your script, e.g.:
    `C:\Users\sharzilnafis\Desktop\Project\captive-portal-automator\index.js`
9.  Click **Finish**.
10. **Run Silently (Optional):** Double-click the newly created task, and under the *General* tab, select **Run whether user is logged on or not** and check **Hidden** to prevent a command prompt window from popping up when you connect to Wi-Fi.

---

## Security & Permissions

*   Your credentials configuration is saved at `~/.capauto/config.json` (macOS) or `%USERPROFILE%\.capauto\config.json` (Windows).
*   On macOS, the script automatically restricts access to this file using `0o600` permissions (read/write by owner only), protecting it from other users on the machine.

---

## Running Tests

To run the local TDD and integration test suite:
```bash
npm test
```
