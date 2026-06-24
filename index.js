import { execSync } from 'child_process';

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

