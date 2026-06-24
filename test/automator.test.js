import { MockPortal } from './mock-portal.js';
import { getSSID, checkConnectivity, parseLoginForm } from '../index.js';

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
      throw new Error(`Expected fallback username field 'email_addr', got '${parsed2.usernameField}'`);
    }
    if (parsed2.passwordField !== 'pass') {
      throw new Error(`Expected password field 'pass', got '${parsed2.passwordField}'`);
    }

    console.log('T3 PASS: Correctly parses HTML form elements and identifies inputs.');
  } finally {
    await portal.stop();
    console.log('Mock Server stopped.');
  }
}

runTests().catch((err) => {
  console.error('Test Suite Failed:', err);
  process.exit(1);
});
