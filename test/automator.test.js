import { MockPortal } from './mock-portal.js';
import { getSSID, checkConnectivity } from '../index.js';

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
  } finally {
    await portal.stop();
    console.log('Mock Server stopped.');
  }
}

runTests().catch((err) => {
  console.error('Test Suite Failed:', err);
  process.exit(1);
});
