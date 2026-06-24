# Dynamic Captive Portal Automator (CapAuto) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a lightweight, zero-dependency Node.js CLI tool that automatically detects captive portals, parses their login forms dynamically, and automates form submission using saved SSID-specific credentials.

**Architecture:** A single-file Node.js script utilizing native `fetch` for HTTP requests, `readline` for terminal prompts, and child process execution (`execSync`) to fetch Wi-Fi SSIDs. A local JSON configuration file stores credentials securely.

**Tech Stack:** Node.js (v18+ using native ESM), Native Node APIs (`http`, `fs`, `path`, `child_process`, `readline`).

## Global Constraints

*   **Zero External Dependencies:** No external npm packages; only use Node.js built-in modules.
*   **Target Platforms:** macOS (13+) and Windows (10/11).
*   **Secure Storage:** Configuration saved at `~/.capauto/config.json` on macOS and `%USERPROFILE%\.capauto\config.json` on Windows with user-only permissions (`chmod 600`).
*   **Dynamic Parsing:** No hardcoded form layouts; forms must be parsed from the fetched HTML dynamically on every run.

---

### Task 1: Project Scaffolding & Mock Captive Portal Server

**Files:**
*   Create: `package.json`
*   Create: `test/mock-portal.js`
*   Create: `test/automator.test.js`

**Interfaces:**
*   Produces: `MockPortal` class which starts a local HTTP server simulating a redirecting captive portal.

- [ ] **Step 1: Create package.json**
  Write a minimal `package.json` to enable ES Modules (`"type": "module"`) and configure a test script.
  Create `/Users/sharzilnafis/Desktop/Project/captive-portal-automator/package.json`:
  ```json
  {
    "name": "captive-portal-automator",
    "version": "1.0.0",
    "type": "module",
    "scripts": {
      "test": "node test/automator.test.js"
    }
  }
  ```

- [ ] **Step 2: Write the Mock Portal Server**
  Create `/Users/sharzilnafis/Desktop/Project/captive-portal-automator/test/mock-portal.js`:
  ```javascript
  import http from 'http';

  export class MockPortal {
    constructor(port = 8080) {
      this.port = port;
      this.isOnline = false;
      this.server = null;
      this.lastRequest = null;
    }

    start() {
      return new Promise((resolve) => {
        this.server = http.createServer((req, res) => {
          this.lastRequest = {
            url: req.url,
            method: req.method,
            headers: req.headers
          };

          // 1. Simulate Apple's Connectivity Check
          if (req.url === '/hotspot-detect.html') {
            if (this.isOnline) {
              res.writeHead(200, { 'Content-Type': 'text/html' });
              res.end('<HTML><HEAD><TITLE>Success</TITLE></HEAD><BODY>Success</BODY></HTML>');
            } else {
              // Redirect to captive portal landing page
              res.writeHead(302, { 'Location': `http://localhost:${this.port}/login` });
              res.end();
            }
            return;
          }

          // 2. Serve the Captive Portal Login Page
          if (req.url === '/login' && req.method === 'GET') {
            res.writeHead(200, { 'Content-Type': 'text/html' });
            res.end(`
              <html>
                <body>
                  <form action="/auth" method="POST">
                    <input type="hidden" name="csrf" value="mock-csrf-token-123">
                    <input type="hidden" name="session" value="xyz987">
                    <label>Username: <input type="text" name="username_field"></label>
                    <label>Password: <input type="password" name="password_field"></label>
                    <input type="submit" value="Login">
                  </form>
                </body>
              </html>
            `);
            return;
          }

          // 3. Handle Authentication Submit
          if (req.url === '/auth' && req.method === 'POST') {
            let body = '';
            req.on('data', chunk => { body += chunk; });
            req.on('end', () => {
              const params = new URLSearchParams(body);
              if (
                params.get('username_field') === 'student123' &&
                params.get('password_field') === 'securepass' &&
                params.get('csrf') === 'mock-csrf-token-123'
              ) {
                this.isOnline = true;
                res.writeHead(200, { 'Content-Type': 'text/html' });
                res.end('<h1>Logged In Successfully!</h1>');
              } else {
                res.writeHead(401, { 'Content-Type': 'text/html' });
                res.end('<h1>Unauthorized</h1>');
              }
            });
            return;
          }

          // 4. Default Not Found
          res.writeHead(404);
          res.end();
        });

        this.server.listen(this.port, () => {
          resolve();
        });
      });
    }

    stop() {
      return new Promise((resolve) => {
        if (this.server) {
          this.server.close(() => resolve());
        } else {
          resolve();
        }
      });
    }
  }
  ```

- [ ] **Step 3: Create the Test Harness**
  Create `/Users/sharzilnafis/Desktop/Project/captive-portal-automator/test/automator.test.js`:
  ```javascript
  import { MockPortal } from './mock-portal.js';

  async function runTests() {
    console.log('Starting Mock Captive Portal Server...');
    const portal = new MockPortal(8080);
    await portal.start();
    console.log('Mock Portal running on port 8080.');

    try {
      // Basic assertion to prove server runs and handles Apple check
      const res = await fetch('http://localhost:8080/hotspot-detect.html', { redirect: 'manual' });
      if (res.status !== 302) {
        throw new Error(`Expected redirect status 302, got ${res.status}`);
      }
      console.log('T1 PASS: Mock Server starts and redirects connectivity probes.');
    } finally {
      await portal.stop();
      console.log('Mock Server stopped.');
    }
  }

  runTests().catch((err) => {
    console.error('Test Suite Failed:', err);
    process.exit(1);
  });
  ```

- [ ] **Step 4: Run Tests to Verify Scaffolding Works**
  Run: `node test/automator.test.js`
  Expected Output: `T1 PASS: Mock Server starts and redirects connectivity probes.`

- [ ] **Step 5: Commit changes**
  ```bash
  git add package.json test/
  git commit -m "test: scaffold project and implement mock captive portal server"
  ```

---

### Task 2: SSID Detection & Connectivity Check

**Files:**
*   Create: `index.js`
*   Modify: `test/automator.test.js`

**Interfaces:**
*   Produces: `getSSID()` returning current WiFi SSID as string.
*   Produces: `checkConnectivity(probeUrl)` returning `{ online: boolean, redirectUrl: string|null }`.

- [ ] **Step 1: Write the failing tests**
  Add tests to `/Users/sharzilnafis/Desktop/Project/captive-portal-automator/test/automator.test.js`:
  ```javascript
  // Import index.js functions once they exist
  import { getSSID, checkConnectivity } from '../index.js';
  
  // Inside runTests() try-block:
  const ssid = getSSID();
  console.log(`Detected SSID: "${ssid}"`);
  if (typeof ssid !== 'string') throw new Error('SSID must be a string');

  portal.isOnline = false;
  const probe1 = await checkConnectivity('http://localhost:8080/hotspot-detect.html');
  if (probe1.online !== false || probe1.redirectUrl !== 'http://localhost:8080/login') {
    throw new Error(`Probe 1 failed: Expected online=false, redirect='http://localhost:8080/login'. Got: ${JSON.stringify(probe1)}`);
  }
  console.log('T2.1 PASS: Correctly detects redirection behind captive portal.');

  portal.isOnline = true;
  const probe2 = await checkConnectivity('http://localhost:8080/hotspot-detect.html');
  if (probe2.online !== true || probe2.redirectUrl !== null) {
    throw new Error(`Probe 2 failed: Expected online=true, redirect=null. Got: ${JSON.stringify(probe2)}`);
  }
  console.log('T2.2 PASS: Correctly detects online status.');
  ```

- [ ] **Step 2: Run test to verify it fails**
  Run: `node test/automator.test.js`
  Expected: Error: Cannot find module '../index.js'

- [ ] **Step 3: Implement SSID Detection and Connectivity Probe**
  Create `/Users/sharzilnafis/Desktop/Project/captive-portal-automator/index.js`:
  ```javascript
  import { execSync } from 'child_process';

  export function getSSID() {
    try {
      if (process.platform === 'darwin') {
        // macOS SSID extraction
        const output = execSync('/usr/sbin/wdutil info').toString();
        const match = output.match(/SSID\s*:\s*(.+)/);
        return match ? match[1].trim() : 'Unknown_macOS_WiFi';
      } else if (process.platform === 'win32') {
        // Windows SSID extraction
        const output = execSync('netsh wlan show interfaces').toString();
        const match = output.match(/SSID\s*:\s*(.+)/);
        return match ? match[1].trim() : 'Unknown_Windows_WiFi';
      }
    } catch (e) {
      return 'Unknown_WiFi';
    }
    return 'Unknown_WiFi';
  }

  export async function checkConnectivity(probeUrl = 'http://captive.apple.com/hotspot-detect.html') {
    try {
      const res = await fetch(probeUrl, { redirect: 'manual', signal: AbortSignal.timeout(10000) });
      if (res.status === 200) {
        const text = await res.text();
        if (text.includes('<BODY>Success</BODY>')) {
          return { online: true, redirectUrl: null };
        }
      }
      if (res.status === 302 || res.status === 307 || res.status === 301) {
        return { online: false, redirectUrl: res.headers.get('location') };
      }
      // If we got hijacked but returned 200 without Success text, it's the captive portal page
      return { online: false, redirectUrl: probeUrl };
    } catch (e) {
      return { online: false, redirectUrl: null };
    }
  }
  ```

- [ ] **Step 4: Run test to verify it passes**
  Run: `node test/automator.test.js`
  Expected: Success logs showing `T2.1 PASS` and `T2.2 PASS`.

- [ ] **Step 5: Commit**
  ```bash
  git add index.js test/automator.test.js
  git commit -m "feat: implement SSID detection and connectivity probing"
  ```

---

### Task 3: Dynamic HTML Form Parsing

**Files:**
*   Modify: `index.js`
*   Modify: `test/automator.test.js`

**Interfaces:**
*   Produces: `parseLoginForm(html, baseUrl)` returning `{ action: string, method: string, fields: Record<string, string>, usernameField: string, passwordField: string }`.

- [ ] **Step 1: Write the failing tests**
  Add tests to `/Users/sharzilnafis/Desktop/Project/captive-portal-automator/test/automator.test.js`:
  ```javascript
  import { parseLoginForm } from '../index.js';

  // Inside runTests():
  const mockHtml = `
    <html>
      <form action="/auth" method="POST">
        <input type="hidden" name="csrf" value="token123">
        <input type="text" name="username_field">
        <input type="password" name="password_field">
      </form>
    </html>
  `;
  const parsed = parseLoginForm(mockHtml, 'http://localhost:8080/login');
  if (parsed.action !== 'http://localhost:8080/auth') {
    throw new Error(`Expected action 'http://localhost:8080/auth', got '${parsed.action}'`);
  }
  if (parsed.usernameField !== 'username_field' || parsed.passwordField !== 'password_field') {
    throw new Error(`Expected fields username_field/password_field, got ${parsed.usernameField}/${parsed.passwordField}`);
  }
  if (parsed.fields.csrf !== 'token123') {
    throw new Error(`Expected hidden csrf value 'token123', got '${parsed.fields.csrf}'`);
  }
  console.log('T3 PASS: Correctly parses HTML form elements and identifies inputs.');
  ```

- [ ] **Step 2: Run test to verify it fails**
  Run: `node test/automator.test.js`
  Expected: TypeError: parseLoginForm is not a function

- [ ] **Step 3: Implement Zero-Dependency HTML Form Parser**
  Modify `/Users/sharzilnafis/Desktop/Project/captive-portal-automator/index.js`:
  Add `parseLoginForm` function:
  ```javascript
  export function parseLoginForm(html, baseUrl) {
    // 1. Extract <form> tag and contents using regex
    const formRegex = /<form([^>]*?)>([\s\S]*?)<\/form>/i;
    const formMatch = html.match(formRegex);
    if (!formMatch) {
      throw new Error('No form element found on the login page');
    }

    const formAttributes = formMatch[1];
    const formContent = formMatch[2];

    // Extract action and method
    const actionMatch = formAttributes.match(/action=["']([^"']+)["']/i);
    let action = actionMatch ? actionMatch[1] : '';
    const methodMatch = formAttributes.match(/method=["']([^"']+)["']/i);
    const method = methodMatch ? methodMatch[1].toUpperCase() : 'POST';

    // Resolve relative action URLs
    if (action && !action.startsWith('http')) {
      const url = new URL(action, baseUrl);
      action = url.href;
    } else if (!action) {
      action = baseUrl;
    }

    // 2. Parse all <input> tags inside the form
    const inputRegex = /<input([^>]*?)>/gi;
    const fields = {};
    let usernameField = '';
    let passwordField = '';

    let match;
    while ((match = inputRegex.exec(formContent)) !== null) {
      const attrs = match[1];
      const nameMatch = attrs.match(/name=["']([^"']+)["']/i);
      const typeMatch = attrs.match(/type=["']([^"']+)["']/i);
      const valueMatch = attrs.match(/value=["']([^"']+)["']/i);

      if (!nameMatch) continue;

      const name = nameMatch[1];
      const type = typeMatch ? typeMatch[1].toLowerCase() : 'text';
      const value = valueMatch ? valueMatch[1] : '';

      fields[name] = value;

      if (type === 'password') {
        passwordField = name;
      } else if (type === 'text' || type === 'email') {
        const nameLower = name.toLowerCase();
        if (
          nameLower.includes('user') ||
          nameLower.includes('login') ||
          nameLower.includes('id') ||
          nameLower.includes('member')
        ) {
          usernameField = name;
        }
      }
    }

    // Fallback if username field not matched by keywords
    if (!usernameField && passwordField) {
      const remainingKeys = Object.keys(fields).filter(k => k !== passwordField && k !== 'csrf');
      if (remainingKeys.length > 0) {
        usernameField = remainingKeys[0];
      }
    }

    return { action, method, fields, usernameField, passwordField };
  }
  ```

- [ ] **Step 4: Run test to verify it passes**
  Run: `node test/automator.test.js`
  Expected: Success logs showing `T3 PASS`.

- [ ] **Step 5: Commit**
  ```bash
  git add index.js test/automator.test.js
  git commit -m "feat: implement dynamic HTML form parser for zero-dependency parsing"
  ```

---

### Task 4: Interactive Setup & Credentials Storage

**Files:**
*   Modify: `index.js`
*   Modify: `test/automator.test.js`

**Interfaces:**
*   Produces: `loadConfig(configPath)` returning the config object.
*   Produces: `saveCredentials(ssid, loginUrl, formDetails, username, password, configPath)` saving config safely.

- [ ] **Step 1: Write the failing tests**
  Add tests to `/Users/sharzilnafis/Desktop/Project/captive-portal-automator/test/automator.test.js`:
  ```javascript
  import fs from 'fs';
  import path from 'path';
  import { loadConfig, saveCredentials } from '../index.js';

  // Inside runTests():
  const testConfigPath = path.join(process.cwd(), 'test-config.json');
  if (fs.existsSync(testConfigPath)) fs.unlinkSync(testConfigPath);

  // Load missing config
  const config1 = loadConfig(testConfigPath);
  if (Object.keys(config1).length !== 0) throw new Error('Expected empty config');

  // Save config
  saveCredentials('Test_Uni_WiFi', 'http://localhost/login', { usernameField: 'user', passwordField: 'pass', fields: { csrf: '12' } }, 'stud', 'pass1', testConfigPath);
  
  const config2 = loadConfig(testConfigPath);
  if (!config2['Test_Uni_WiFi'] || config2['Test_Uni_WiFi'].username !== 'stud') {
    throw new Error('Config failed to save or load credentials correctly');
  }

  // Cleanup
  if (fs.existsSync(testConfigPath)) fs.unlinkSync(testConfigPath);
  console.log('T4 PASS: Correctly stores and loads network-specific credentials.');
  ```

- [ ] **Step 2: Run test to verify it fails**
  Run: `node test/automator.test.js`
  Expected: TypeError: loadConfig is not a function

- [ ] **Step 3: Implement Local Storage Management**
  Modify `/Users/sharzilnafis/Desktop/Project/captive-portal-automator/index.js`:
  Add imports and functions:
  ```javascript
  import fs from 'fs';
  import path from 'path';
  import os from 'os';
  import readline from 'readline';

  export function getConfigPath() {
    const home = os.homedir();
    return path.join(home, '.capauto', 'config.json');
  }

  export function loadConfig(configPath = getConfigPath()) {
    try {
      if (fs.existsSync(configPath)) {
        const data = fs.readFileSync(configPath, 'utf8');
        return JSON.parse(data);
      }
    } catch (e) {
      // Ignore reading errors, return empty config
    }
    return {};
  }

  export function saveCredentials(ssid, loginUrl, formDetails, username, password, configPath = getConfigPath()) {
    const dir = path.dirname(configPath);
    if (!fs.existsSync(dir)) {
      fs.mkdirSync(dir, { recursive: true });
    }

    const config = loadConfig(configPath);
    config[ssid] = {
      loginUrl,
      username,
      password,
      usernameField: formDetails.usernameField,
      passwordField: formDetails.passwordField,
      staticFields: formDetails.fields,
      action: formDetails.action
    };

    fs.writeFileSync(configPath, JSON.stringify(config, null, 2), 'utf8');
    
    // Set user-only read/write permissions on macOS/Linux
    if (process.platform !== 'win32') {
      try {
        fs.chmodSync(configPath, 0o600);
      } catch (e) {
        // Fallback
      }
    }
  }

  export function promptUser(query) {
    const rl = readline.createInterface({
      input: process.stdin,
      output: process.stdout,
    });
    return new Promise((resolve) => rl.question(query, (ans) => {
      rl.close();
      resolve(ans);
    }));
  }
  ```

- [ ] **Step 4: Run test to verify it passes**
  Run: `node test/automator.test.js`
  Expected: Success logs showing `T4 PASS`.

- [ ] **Step 5: Commit**
  ```bash
  git add index.js test/automator.test.js
  git commit -m "feat: implement configuration storage and interactive terminal prompts"
  ```

---

### Task 5: Form Submission & Verification

**Files:**
*   Modify: `index.js`
*   Modify: `test/automator.test.js`

**Interfaces:**
*   Produces: `submitLogin(action, method, payload)` performing the login request.
*   Produces: `runAutomator(configPath, probeUrl)` orchestrating the full flow.

- [ ] **Step 1: Write the failing tests**
  Add tests to `/Users/sharzilnafis/Desktop/Project/captive-portal-automator/test/automator.test.js`:
  ```javascript
  import { runAutomator } from '../index.js';

  // Inside runTests():
  const integrationConfigPath = path.join(process.cwd(), 'integration-config.json');
  if (fs.existsSync(integrationConfigPath)) fs.unlinkSync(integrationConfigPath);

  // Setup mock credentials for our mock server
  saveCredentials(
    'Mock_WiFi',
    'http://localhost:8080/login',
    { usernameField: 'username_field', passwordField: 'password_field', action: 'http://localhost:8080/auth', fields: { csrf: 'mock-csrf-token-123', session: 'xyz987' } },
    'student123',
    'securepass',
    integrationConfigPath
  );

  // Start with offline portal
  portal.isOnline = false;
  
  // Mock SSID detection in environment
  process.env.CAPAUTO_TEST_SSID = 'Mock_WiFi';

  console.log('Running integration automation test...');
  const success = await runAutomator(integrationConfigPath, 'http://localhost:8080/hotspot-detect.html');
  if (!success) {
    throw new Error('Integration runAutomator failed');
  }
  if (!portal.isOnline) {
    throw new Error('Portal did not record transition to online after automation');
  }

  // Cleanup
  if (fs.existsSync(integrationConfigPath)) fs.unlinkSync(integrationConfigPath);
  console.log('T5 PASS: Complete end-to-end flow connects and logs into portal successfully.');
  ```

- [ ] **Step 2: Run test to verify it fails**
  Run: `node test/automator.test.js`
  Expected: TypeError: runAutomator is not a function

- [ ] **Step 3: Implement Form Submission and Orchestrator**
  Modify `/Users/sharzilnafis/Desktop/Project/captive-portal-automator/index.js`:
  Add implementation:
  ```javascript
  export async function submitLogin(action, method, payload) {
    const body = new URLSearchParams(payload).toString();
    const options = {
      method: method,
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
        'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36'
      },
      signal: AbortSignal.timeout(10000)
    };
    if (method === 'POST') {
      options.body = body;
    }

    const res = await fetch(action, options);
    return res;
  }

  export async function runAutomator(configPath = getConfigPath(), probeUrl = 'http://captive.apple.com/hotspot-detect.html') {
    const ssid = process.env.CAPAUTO_TEST_SSID || getSSID();
    console.log(`[${new Date().toISOString()}] Checking network connection on Wi-Fi: "${ssid}"...`);

    const status = await checkConnectivity(probeUrl);
    if (status.online) {
      console.log('Already online. Exiting.');
      return true;
    }

    if (!status.redirectUrl) {
      console.log('Not online, but no captive portal redirection detected. Exiting.');
      return false;
    }

    console.log(`Captive portal detected! Redirected to: ${status.redirectUrl}`);

    // Load credentials
    const config = loadConfig(configPath);
    let credentials = config[ssid];

    let username, password;
    let formDetails;

    if (!credentials) {
      console.log(`No saved credentials found for SSID: "${ssid}". Initiating first-time setup...`);
      console.log('Fetching login page to analyze form...');
      const pageRes = await fetch(status.redirectUrl);
      const html = await pageRes.text();

      try {
        formDetails = parseLoginForm(html, status.redirectUrl);
      } catch (err) {
        console.error('Failed to parse login form automatically:', err.message);
        console.log('Please enter credentials, we will try to submit standard fields.');
        formDetails = {
          action: status.redirectUrl,
          method: 'POST',
          fields: {},
          usernameField: 'username',
          passwordField: 'password'
        };
      }

      console.log(`Detected username field: "${formDetails.usernameField}"`);
      console.log(`Detected password field: "${formDetails.passwordField}"`);

      username = await promptUser('Enter your Wi-Fi username/ID: ');
      password = await promptUser('Enter your Wi-Fi password: ');

      if (!username || !password) {
        console.error('Credentials cannot be empty. Aborting.');
        return false;
      }

      saveCredentials(ssid, status.redirectUrl, formDetails, username, password, configPath);
      console.log(`Credentials saved for SSID: "${ssid}"`);
      
      // Reload credentials
      credentials = loadConfig(configPath)[ssid];
    }

    // Prepare payload
    const payload = {
      ...credentials.staticFields,
      [credentials.usernameField]: credentials.username,
      [credentials.passwordField]: credentials.password
    };

    console.log(`Submitting login request to: ${credentials.action}...`);
    try {
      await submitLogin(credentials.action, 'POST', payload);
      console.log('Form submitted. Verifying internet connectivity...');
      
      // Wait 2 seconds for portal to register
      await new Promise(resolve => setTimeout(resolve, 2000));

      const doubleCheck = await checkConnectivity(probeUrl);
      if (doubleCheck.online) {
        console.log('Success! Internet connection established.');
        return true;
      } else {
        console.error('Form submitted but still redirected. Login might have failed or needs verification.');
        return false;
      }
    } catch (err) {
      console.error('Error submitting login form:', err.message);
      return false;
    }
  }

  // Self-execute if run directly (not imported by test)
  if (process.argv[1] === import.meta.url || process.argv[1]?.endsWith('index.js')) {
    runAutomator().then(success => {
      process.exit(success ? 0 : 1);
    }).catch(err => {
      console.error('Fatal execution error:', err);
      process.exit(1);
    });
  }
  ```

- [ ] **Step 4: Run test to verify it passes**
  Run: `node test/automator.test.js`
  Expected: All tests pass successfully!

- [ ] **Step 5: Commit**
  ```bash
  git add index.js test/automator.test.js
  git commit -m "feat: implement form submission and complete orchestrator with passing integration tests"
  ```

---

### Task 6: macOS Background Agent Setup & Integration

**Files:**
*   Create: `com.sharzil.capauto.plist`
*   Create: `README.md`

**Interfaces:**
*   Produces: LaunchAgent plist configuration allowing background execution on macOS Wi-Fi change.

- [ ] **Step 1: Create the launchd plist configuration**
  Create `/Users/sharzilnafis/Desktop/Project/captive-portal-automator/com.sharzil.capauto.plist`:
  (Reference content in Design Spec Section 5.1).

- [ ] **Step 2: Create a rich README.md**
  Create `/Users/sharzilnafis/Desktop/Project/captive-portal-automator/README.md` outlining manual usage, config security, and setup instructions for macOS background launchd and Windows Task Scheduler.

- [ ] **Step 3: Run validation to verify setup instructions**
  Run: `node index.js` (Run to ensure it prints SSID and exits or prompts correctly if not connected).

- [ ] **Step 4: Commit**
  ```bash
  git add com.sharzil.capauto.plist README.md
  git commit -m "docs: add macOS background launchd configuration and comprehensive README"
  ```
