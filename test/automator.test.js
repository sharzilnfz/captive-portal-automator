import fs from 'fs';
import path from 'path';
import { MockPortal } from './mock-portal.js';
import { getSSID, checkConnectivity, parseLoginForm, loadConfig, saveCredentials, runAutomator } from '../index.js';

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

    // Task 2: SSID Detection & Connectivity Check
    const ssid = getSSID();
    console.log(`Detected SSID: "${ssid}"`);
    if (typeof ssid !== 'string') throw new Error('SSID must be a string');

    portal.isOnline = false;
    const probe1 = await checkConnectivity('http://localhost:8080/hotspot-detect.html');
    if (probe1.online !== false || probe1.redirectUrl !== 'http://localhost:8080/login') {
      throw new Error(`Probe 1 failed: Expected online=false, redirect='http://localhost:8080/login'. Got: ${JSON.stringify(probe1)}`);
    }
    console.log('T2.1 PASS: Correctly detects redirection behind captive portal and resolves relative redirect URL.');

    portal.isOnline = true;
    const probe2 = await checkConnectivity('http://localhost:8080/hotspot-detect.html');
    if (probe2.online !== true || probe2.redirectUrl !== null) {
      throw new Error(`Probe 2 failed: Expected online=true, redirect=null. Got: ${JSON.stringify(probe2)}`);
    }
    console.log('T2.2 PASS: Correctly detects online status.');

    // Task 3: Dynamic HTML Form Parsing
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

    // Additional case: form with no action, method="GET", and fallback username detection
    const mockHtml2 = `
      <form method="GET">
        <input type="text" name="email_addr">
        <input type="password" name="pass">
      </form>
    `;
    const parsed2 = parseLoginForm(mockHtml2, 'http://localhost:8080/login');
    if (parsed2.action !== 'http://localhost:8080/login') {
      throw new Error(`Expected action 'http://localhost:8080/login' (baseUrl), got '${parsed2.action}'`);
    }
    if (parsed2.method !== 'GET') {
      throw new Error(`Expected method 'GET', got '${parsed2.method}'`);
    }
    if (parsed2.usernameField !== 'email_addr') {
      throw new Error(`Expected username field 'email_addr', got '${parsed2.usernameField}'`);
    }
    if (parsed2.passwordField !== 'pass') {
      throw new Error(`Expected password field 'pass', got '${parsed2.passwordField}'`);
    }

    // New Test Case 1: Unquoted attributes and spaces around equals signs
    const unquotedHtml = `
      <form action = /auth method = POST>
        <input type = hidden name = csrf value = token123>
        <input type = text name = username_field>
        <input type = password name = password_field>
      </form>
    `;
    const parsedUnquoted = parseLoginForm(unquotedHtml, 'http://localhost:8080/login');
    if (parsedUnquoted.action !== 'http://localhost:8080/auth') {
      throw new Error(`Expected action 'http://localhost:8080/auth', got '${parsedUnquoted.action}'`);
    }
    if (parsedUnquoted.usernameField !== 'username_field' || parsedUnquoted.passwordField !== 'password_field') {
      throw new Error(`Expected username_field/password_field, got ${parsedUnquoted.usernameField}/${parsedUnquoted.passwordField}`);
    }
    if (parsedUnquoted.fields.csrf !== 'token123') {
      throw new Error(`Expected csrf 'token123', got '${parsedUnquoted.fields.csrf}'`);
    }

    // New Test Case 2: Multiple forms on a page, prioritizing the one with a password input
    const multiFormHtml = `
      <html>
        <!-- Form 1: Language selection (no password field) -->
        <form action="/lang" method="GET">
          <input type="submit" name="lang" value="en">
          <input type="submit" name="lang" value="es">
        </form>
        <!-- Form 2: Actual login form (has password field) -->
        <form action="/login-submit" method="POST">
          <input type="text" name="email">
          <input type="password" name="pass">
        </form>
      </html>
    `;
    const parsedMulti = parseLoginForm(multiFormHtml, 'http://localhost:8080/login');
    if (parsedMulti.action !== 'http://localhost:8080/login-submit') {
      throw new Error(`Expected action 'http://localhost:8080/login-submit' from prioritized login form, got '${parsedMulti.action}'`);
    }
    if (parsedMulti.usernameField !== 'email' || parsedMulti.passwordField !== 'pass') {
      throw new Error(`Expected email/pass from prioritized login form, got ${parsedMulti.usernameField}/${parsedMulti.passwordField}`);
    }

    // New Test Case 3: Fallback username detection ignoring hidden/submit inputs
    const fallbackHtml = `
      <form action="/login" method="POST">
        <input type="hidden" name="csrf" value="xyz">
        <input type="submit" name="submit-btn" value="Login">
        <input type="text" name="weird_username_field">
        <input type="password" name="pass">
      </form>
    `;
    const parsedFallback = parseLoginForm(fallbackHtml, 'http://localhost:8080/login');
    if (parsedFallback.usernameField !== 'weird_username_field') {
      throw new Error(`Expected fallback username field 'weird_username_field', got '${parsedFallback.usernameField}'`);
    }

    // New Test Case 4: Guest-portal keywords (phone, mobile, telephone)
    const keywordsHtml = `
      <form action="/login" method="POST">
        <input type="text" name="phone">
        <input type="password" name="pass">
      </form>
    `;
    const parsedKeywords = parseLoginForm(keywordsHtml, 'http://localhost:8080/login');
    if (parsedKeywords.usernameField !== 'phone') {
      throw new Error(`Expected username field 'phone', got '${parsedKeywords.usernameField}'`);
    }

    console.log('T3 PASS: Correctly parses HTML form elements and identifies inputs (including multi-form, unquoted attributes, and robust fallback username detection).');

    // Task 4: Interactive Setup & Credentials Storage
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

    // Verify permission restrictions on the config file (0o600 on Unix)
    if (process.platform !== 'win32') {
      const stat = fs.statSync(testConfigPath);
      const mode = stat.mode & 0o777;
      if (mode !== 0o600) {
        throw new Error(`Expected file mode 0600, got ${mode.toString(8)}`);
      }
    }

    // Cleanup
    if (fs.existsSync(testConfigPath)) fs.unlinkSync(testConfigPath);
    console.log('T4 PASS: Correctly stores and loads network-specific credentials with safe permissions.');

    // Task 5: Integration test
    const integrationConfigPath = path.join(process.cwd(), 'integration-config.json');
    if (fs.existsSync(integrationConfigPath)) fs.unlinkSync(integrationConfigPath);

    // Setup mock credentials for our mock server
    saveCredentials(
      'Mock_WiFi',
      'http://localhost:8080/login',
      { usernameField: 'username_field', passwordField: 'password_field', action: 'http://localhost:8080/auth', method: 'POST', fields: { csrf: 'mock-csrf-token-123', session: 'xyz987' } },
      'student123',
      'securepass',
      integrationConfigPath
    );

    // Start with offline portal
    portal.isOnline = false;
    
    // Mock SSID detection in environment
    process.env.CAPAUTO_TEST_SSID = 'Mock_WiFi';

    console.log('Running integration automation test (POST)...');
    const success = await runAutomator(integrationConfigPath, 'http://localhost:8080/hotspot-detect.html');
    if (!success) {
      throw new Error('Integration runAutomator failed');
    }
    if (!portal.isOnline) {
      throw new Error('Portal did not record transition to online after automation');
    }

    // Cleanup
    if (fs.existsSync(integrationConfigPath)) fs.unlinkSync(integrationConfigPath);
    console.log('T5 PASS: Complete end-to-end flow connects and logs into portal successfully (POST).');

    // Task 5.2: Integration test with GET method
    const getIntegrationConfigPath = path.join(process.cwd(), 'get-integration-config.json');
    if (fs.existsSync(getIntegrationConfigPath)) fs.unlinkSync(getIntegrationConfigPath);

    saveCredentials(
      'Mock_WiFi_GET',
      'http://localhost:8080/login',
      { usernameField: 'username_field', passwordField: 'password_field', action: 'http://localhost:8080/auth', method: 'GET', fields: { csrf: 'mock-csrf-token-123', session: 'xyz987' } },
      'student123',
      'securepass',
      getIntegrationConfigPath
    );

    portal.isOnline = false;
    process.env.CAPAUTO_TEST_SSID = 'Mock_WiFi_GET';

    console.log('Running integration automation test (GET)...');
    const getSuccess = await runAutomator(getIntegrationConfigPath, 'http://localhost:8080/hotspot-detect.html');
    if (!getSuccess) {
      throw new Error('Integration runAutomator with GET failed');
    }
    if (!portal.isOnline) {
      throw new Error('Portal did not record transition to online after GET automation');
    }

    // Cleanup
    if (fs.existsSync(getIntegrationConfigPath)) fs.unlinkSync(getIntegrationConfigPath);
    console.log('T5.2 PASS: Complete end-to-end flow connects and logs into portal successfully (GET).');
  } finally {
    await portal.stop();
    console.log('Mock Server stopped.');
  }
}

runTests().catch((err) => {
  console.error('Test Suite Failed:', err);
  process.exit(1);
});
