import { execSync } from 'child_process';
import fs from 'fs';
import path from 'path';
import os from 'os';
import readline from 'readline';
import { fileURLToPath } from 'url';

// Bypass SSL verification by default for captive portal requests.
// Captive portal controllers frequently use self-signed or local-only certificates.
process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';

export function getSSID() {
  try {
    if (process.platform === 'darwin') {
      // macOS SSID extraction
      try {
        const output = execSync('/System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport -I').toString();
        const match = output.match(/^\s*SSID\s*:\s*(.+)/m);
        if (match) return match[1].trim();
      } catch (e) {
        // Fall back if airport command fails or isn't found
      }

      try {
        const output = execSync('networksetup -getairportnetwork en0').toString();
        const match = output.match(/Current Wi-Fi Network\s*:\s*(.+)/i);
        if (match) return match[1].trim();
      } catch (e) {
        // Fall back if networksetup fails
      }

      return 'Unknown_macOS_WiFi';
    } else if (process.platform === 'win32') {
      // Windows SSID extraction
      const output = execSync('netsh wlan show interfaces').toString();
      const match = output.match(/^\s*SSID\s*:\s*(.+)/m);
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
      if (/<body>\s*success\s*<\/body>/i.test(text)) {
        return { online: true, redirectUrl: null };
      }
    }
    if (res.status >= 300 && res.status < 400) {
      const redirectLocation = res.headers.get('location');
      const redirectUrl = redirectLocation ? new URL(redirectLocation, probeUrl).href : null;
      return { online: false, redirectUrl };
    }
    // If we got hijacked but returned 200 without Success text, it's the captive portal page
    return { online: false, redirectUrl: probeUrl };
  } catch (e) {
    return { online: false, redirectUrl: null };
  }
}

function getAttr(attrs, attrName) {
  const regex = new RegExp('(?:^|\\s)' + attrName + '\\s*=\\s*(?:["\']([^\'"]*)["\']|([^\\s>]+))', 'i');
  const match = attrs.match(regex);
  return match ? (match[1] !== undefined ? match[1] : match[2]) : null;
}

export function parseLoginForm(html, baseUrl) {
  // Find all <form> elements on the page
  const formRegex = /<form([^>]*?)>([\s\S]*?)<\/form>/gi;
  const formMatches = [];
  let formMatch;
  while ((formMatch = formRegex.exec(html)) !== null) {
    formMatches.push(formMatch);
  }

  if (formMatches.length === 0) {
    throw new Error('No form element found on the login page');
  }

  // Parse each form
  const parsedForms = formMatches.map(match => {
    const formAttributes = match[1];
    const formContent = match[2];

    // Extract action and method using getAttr
    let action = getAttr(formAttributes, 'action') || '';
    const method = (getAttr(formAttributes, 'method') || 'POST').toUpperCase();

    // Resolve relative action URLs
    if (action && !action.startsWith('http')) {
      const url = new URL(action, baseUrl);
      action = url.href;
    } else if (!action) {
      action = baseUrl;
    }

    // Parse all <input> tags inside the form
    const inputRegex = /<input([^>]*?)>/gi;
    const fields = {};
    let usernameField = '';
    let passwordField = '';
    const textInputNames = [];

    let inputMatch;
    while ((inputMatch = inputRegex.exec(formContent)) !== null) {
      const attrs = inputMatch[1];
      const name = getAttr(attrs, 'name');
      if (!name) continue;

      const type = (getAttr(attrs, 'type') || 'text').toLowerCase();
      const value = getAttr(attrs, 'value') || '';

      fields[name] = value;

      if (type === 'password') {
        passwordField = name;
      } else if (type === 'text' || type === 'email') {
        textInputNames.push(name);

        const nameLower = name.toLowerCase();
        if (
          nameLower.includes('user') ||
          nameLower.includes('login') ||
          nameLower.includes('id') ||
          nameLower.includes('member') ||
          nameLower.includes('email') ||
          nameLower.includes('phone') ||
          nameLower.includes('mobile') ||
          nameLower.includes('telephone')
        ) {
          if (!usernameField) {
            usernameField = name;
          }
        }
      }
    }

    // Fallback if username field not matched by keywords
    if (!usernameField && passwordField) {
      const candidates = textInputNames.filter(name => name !== passwordField);
      if (candidates.length > 0) {
        usernameField = candidates[0];
      }
    }

    return { action, method, fields, usernameField, passwordField };
  });

  // Prioritize forms containing a password input
  const loginForm = parsedForms.find(f => f.passwordField);
  if (loginForm) {
    return loginForm;
  }

  // Fallback to the first form on the page
  return parsedForms[0];
}

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
    console.warn(`[Warning] Failed to parse config file at "${configPath}":`, e.message);
  }
  return {};
}

export function saveCredentials(ssid, loginUrl, formDetails, username, password, configPath = getConfigPath()) {
  const dir = path.dirname(configPath);
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true, mode: 0o700 });
  }

  const config = loadConfig(configPath);
  config[ssid] = {
    loginUrl,
    username,
    password,
    usernameField: formDetails.usernameField,
    passwordField: formDetails.passwordField,
    staticFields: formDetails.fields,
    action: formDetails.action,
    method: formDetails.method || 'POST'
  };

  fs.writeFileSync(configPath, JSON.stringify(config, null, 2), { mode: 0o600, encoding: 'utf8' });
  
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
  if (!process.stdin.isTTY) {
    return Promise.reject(new Error('Terminal is non-interactive, cannot prompt for credentials in background mode'));
  }
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
  });
  return new Promise((resolve) => rl.question(query, (ans) => {
    rl.close();
    resolve(ans);
  }));
}

export async function submitLogin(action, method, payload) {
  const normalizedMethod = (method || 'POST').toUpperCase();
  let targetUrl = action;
  const options = {
    method: normalizedMethod,
    headers: {
      'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36'
    },
    signal: AbortSignal.timeout(10000)
  };

  if (normalizedMethod === 'GET') {
    const urlObj = new URL(action);
    const params = new URLSearchParams(payload);
    for (const [key, value] of params) {
      urlObj.searchParams.set(key, value);
    }
    targetUrl = urlObj.href;
  } else if (normalizedMethod === 'POST') {
    options.headers['Content-Type'] = 'application/x-www-form-urlencoded';
    options.body = new URLSearchParams(payload).toString();
  }

  const res = await fetch(targetUrl, options);
  return res;
}

export async function runAutomator(configPath = getConfigPath(), probeUrl = 'http://captive.apple.com/hotspot-detect.html') {
  let ssid = process.env.CAPAUTO_TEST_SSID;
  if (!ssid) {
    try {
      ssid = getSSID();
    } catch (err) {
      ssid = 'Unknown_WiFi';
    }
  }
  console.log(`[${new Date().toISOString()}] Checking network connection on Wi-Fi: "${ssid}"...`);

  let status = await checkConnectivity(probeUrl);
  if (status.online) {
    console.log('Already online. Exiting.');
    return true;
  }

  // If not online and no redirect URL found immediately, retry up to 3 times with a 2-second delay.
  // This gives the network interface time to stabilize and receive a DHCP lease when triggered by network changes.
  let attempts = 1;
  while (!status.online && !status.redirectUrl && attempts < 3) {
    console.log(`Connection offline but no captive portal detected (attempt ${attempts}). Retrying in 2 seconds...`);
    await new Promise(resolve => setTimeout(resolve, 2000));
    status = await checkConnectivity(probeUrl);
    attempts++;
  }

  if (status.online) {
    console.log('Already online. Exiting.');
    return true;
  }

  if (!status.redirectUrl) {
    console.log('Not online, and no captive portal redirection detected. Exiting.');
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
    
    let html = '';
    try {
      const pageRes = await fetch(status.redirectUrl, { signal: AbortSignal.timeout(10000) });
      html = await pageRes.text();
    } catch (err) {
      console.error('Failed to fetch captive portal login page:', err.message);
      return false;
    }

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
    await submitLogin(credentials.action, credentials.method || 'POST', payload);
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
let nodePath = null;
try {
  nodePath = process.argv[1] ? fs.realpathSync(process.argv[1]) : null;
} catch (e) {
  // Ignore realpath exceptions for virtual/non-existent paths
}
const currentPath = fileURLToPath(import.meta.url);
const isMain = nodePath === currentPath || (nodePath && nodePath === fs.realpathSync(currentPath));

if (isMain) {
  runAutomator().then(success => {
    process.exit(success ? 0 : 1);
  }).catch(err => {
    console.error('Fatal execution error:', err);
    process.exit(1);
  });
}

