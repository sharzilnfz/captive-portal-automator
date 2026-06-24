import { execSync } from 'child_process';
import fs from 'fs';
import path from 'path';
import os from 'os';
import readline from 'readline';

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
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
  });
  return new Promise((resolve) => rl.question(query, (ans) => {
    rl.close();
    resolve(ans);
  }));
}

