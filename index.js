import { execSync } from 'child_process';
import fs from 'fs';
import path from 'path';
import os from 'os';
import readline from 'readline';
import { fileURLToPath } from 'url';

// Captive portals frequently use self-signed or local-only certificates.
process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';

// ────────────────────────────────────────────────────────────────────────────
// SSID detection
// ────────────────────────────────────────────────────────────────────────────
export function getSSID() {
  try {
    if (process.platform === 'darwin') {
      // 1. Find which device is the Wi-Fi interface (not always en0)
      let wifiDevice = null;
      try {
        const ports = execSync('networksetup -listallhardwareports 2>/dev/null', { encoding: 'utf8' });
        const m = ports.match(/Hardware Port:\s*Wi-Fi[\s\S]*?Device:\s*(\w+)/i);
        if (m) wifiDevice = m[1].trim();
      } catch (_) { /* ignore */ }

      // 2. Query the Wi-Fi device
      if (wifiDevice) {
        try {
          const out = execSync(`networksetup -getairportnetwork ${wifiDevice} 2>/dev/null`, { encoding: 'utf8' });
          const m = out.match(/Current Wi-Fi Network\s*:\s*(.+)/i);
          if (m) return m[1].trim();
        } catch (_) { /* fall through */ }
      }

      // 3. Try every en* interface
      try {
        const list = execSync('ifconfig -l 2>/dev/null', { encoding: 'utf8' })
          .trim().split(/\s+/).filter(d => /^en\d+$/.test(d));
        for (const dev of list) {
          try {
            const out = execSync(`networksetup -getairportnetwork ${dev} 2>/dev/null`, { encoding: 'utf8' });
            const m = out.match(/Current Wi-Fi Network\s*:\s*(.+)/i);
            if (m) return m[1].trim();
          } catch (_) { /* try next */ }
        }
      } catch (_) { /* ignore */ }

      // 4. system_profiler — slow but reliable on modern macOS
      try {
        const out = execSync('system_profiler SPAirPortDataType 2>/dev/null', { encoding: 'utf8', timeout: 15000 });
        // On Sonoma/Sequoia: "Current Network: SSID" under the active interface
        const m = out.match(/Current Network\s*:\s*(.+)/i)
          || out.match(/Current Network Information:[\s\S]*?\n\s+([^\n:]+):/m);
        if (m) return m[1].trim();
      } catch (_) { /* ignore */ }

      return 'Unknown_macOS_WiFi';
    }

    if (process.platform === 'win32') {
      const out = execSync('netsh wlan show interfaces', { encoding: 'utf8' });
      const m = out.match(/^\s*SSID\s*:\s*(.+)/m);
      return m ? m[1].trim() : 'Unknown_Windows_WiFi';
    }

    if (process.platform === 'linux') {
      try {
        const out = execSync('nmcli -t -f active,ssid dev wifi 2>/dev/null', { encoding: 'utf8' });
        const line = out.split('\n').find(l => l.startsWith('yes:'));
        if (line) return line.substring(4).trim();
      } catch (_) { /* ignore */ }
      try {
        const out = execSync('iwgetid -r 2>/dev/null', { encoding: 'utf8' });
        if (out.trim()) return out.trim();
      } catch (_) { /* ignore */ }
      return 'Unknown_Linux_WiFi';
    }
  } catch (_) { /* ignore */ }
  return 'Unknown_WiFi';
}

// ────────────────────────────────────────────────────────────────────────────
// Cookie jar — many captive portals require a session cookie set by the
// initial GET before the POST will be accepted.
// ────────────────────────────────────────────────────────────────────────────
function createCookieJar() {
  const cookies = new Map();
  return {
    capture(res) {
      const setCookies = res.headers.getSetCookie?.() ?? [];
      for (const c of setCookies) {
        const [pair] = c.split(';');
        const eq = pair.indexOf('=');
        if (eq > 0) {
          cookies.set(pair.slice(0, eq).trim(), pair.slice(eq + 1).trim());
        }
      }
    },
    header() {
      if (cookies.size === 0) return undefined;
      return Array.from(cookies.entries()).map(([k, v]) => `${k}=${v}`).join('; ');
    }
  };
}

// ────────────────────────────────────────────────────────────────────────────
// Connectivity / captive-portal detection
// ────────────────────────────────────────────────────────────────────────────
async function probe(probeUrl) {
  try {
    // redirect: 'follow' (default) — then inspect res.redirected + res.url.
    // Using 'manual' returns an opaque-redirect response whose headers are
    // inaccessible, which was the root cause of the original bug.
    const res = await fetch(probeUrl, {
      redirect: 'follow',
      signal: AbortSignal.timeout(10000),
    });
    const text = await res.text();
    const finalUrl = res.url || probeUrl;

    // Apple success page: <HTML><HEAD><TITLE>Success</TITLE></HEAD><BODY>Success</BODY></HTML>
    const isAppleSuccess =
      /<body>\s*success\s*<\/body>/i.test(text) && !res.redirected;

    // Google / MSFT 204 probes
    const is204 = res.status === 204;

    if (isAppleSuccess || is204) {
      return { online: true, redirectUrl: null, html: text, finalUrl };
    }

    // If fetch followed a redirect, the final URL *is* the portal login page.
    if (res.redirected && finalUrl !== probeUrl) {
      return { online: false, redirectUrl: finalUrl, html: text, finalUrl };
    }

    // 200 at the probe URL but content isn't "Success" — the portal
    // hijacked the response.  Try to dig the real login URL out of the HTML.
    const extracted = extractPortalUrl(text, probeUrl);
    if (extracted && extracted !== probeUrl) {
      return { online: false, redirectUrl: extracted, html: text, finalUrl };
    }

    // Couldn't extract a URL — return the probe URL but mark as hijacked
    // so the caller can try alternate probes.
    return { online: false, redirectUrl: null, html: text, finalUrl };
  } catch (e) {
    return { online: false, redirectUrl: null, html: null, finalUrl: probeUrl };
  }
}

export async function checkConnectivity(probeUrl = 'http://captive.apple.com/hotspot-detect.html') {
  let r = await probe(probeUrl);
  if (r.online) return { online: true, redirectUrl: null };

  // If the first probe was hijacked but we couldn't extract a portal URL,
  // try alternate probes — a different probe may trigger a 302 to the
  // actual login page.
  if (!r.redirectUrl) {
    const alternates = [
      'http://neverssl.com/',
      'http://msftconnecttest.com/redirect',
      'http://connectivitycheck.gstatic.com/generate_204',
      'http://detectportal.firefox.com/canonical.html',
    ];
    for (const alt of alternates) {
      r = await probe(alt);
      if (r.online) return { online: true, redirectUrl: null };
      if (r.redirectUrl) return { online: false, redirectUrl: r.redirectUrl };
    }
  }

  return { online: false, redirectUrl: r.redirectUrl };
}

export function extractPortalUrl(html, fallbackUrl) {
  if (!html) return null;
  const probeHost = new URL(fallbackUrl).hostname;

  // 1. <meta http-equiv="refresh" content="0;url=…">
  const metaRefresh =
    html.match(/<meta[^>]+http-equiv\s*=\s*["']?refresh["']?[^>]+content\s*=\s*["'][^"']*url\s*=\s*([^"'\s>]+)/i) ||
    html.match(/<meta[^>]+content\s*=\s*["'][^"']*url\s*=\s*([^"'\s>]+)[^>]*http-equiv\s*=\s*["']?refresh["']?/i);
  if (metaRefresh) {
    try {
      const u = new URL(metaRefresh[1], fallbackUrl);
      return u.href;
    } catch (_) { /* ignore */ }
  }

  // 2. <form action="…">
  const formAction = html.match(/<form[^>]+action\s*=\s*["']([^"']+)["']/i);
  if (formAction) {
    try {
      return new URL(formAction[1], fallbackUrl).href;
    } catch (_) { /* ignore */ }
  }

  // 3. First <a href> pointing to a non-probe host
  const linkRe = /<a[^>]+href\s*=\s*["']([^"'#][^"']*)["']/gi;
  let lm;
  while ((lm = linkRe.exec(html)) !== null) {
    try {
      const u = new URL(lm[1], fallbackUrl);
      if (u.protocol.startsWith('http') && u.hostname !== probeHost) return u.href;
    } catch (_) { /* ignore */ }
  }

  // 4. JS-style redirect: location.href = "…" or location.replace("…")
  const jsRedirect = html.match(/location(?:\.href|\.replace)\s*=\s*["']([^"']+)["']/i);
  if (jsRedirect) {
    try {
      return new URL(jsRedirect[1], fallbackUrl).href;
    } catch (_) { /* ignore */ }
  }

  return null;   // ← was: return fallbackUrl  (that was the trap)
}

// ────────────────────────────────────────────────────────────────────────────
// Form parsing helpers
// ────────────────────────────────────────────────────────────────────────────
function getAttr(attrs, name) {
  const re = new RegExp('(?:^|\\s)' + name + '\\s*=\\s*(?:["\']([^\'"]*)["\']|([^\\s>]+))', 'i');
  const m = attrs.match(re);
  return m ? (m[1] !== undefined ? m[1] : m[2]) : null;
}

export function parseLoginForm(html, baseUrl) {
  const formRegex = /<form([^>]*?)>([\s\S]*?)<\/form>/gi;
  const forms = [];
  let fm;
  while ((fm = formRegex.exec(html)) !== null) forms.push(fm);

  if (forms.length === 0) {
    throw new Error('No form element found on the login page');
  }

  const parsed = forms.map(match => {
    const formAttrs = match[1];
    const formContent = match[2];

    let action = getAttr(formAttrs, 'action') || '';
    const method = (getAttr(formAttrs, 'method') || 'POST').toUpperCase();

    if (action && !/^https?:/i.test(action)) {
      action = new URL(action, baseUrl).href;
    } else if (!action) {
      action = baseUrl;
    }

    const inputRe = /<input([^>]*?)\/?>/gi;
    const fields = {};
    let usernameField = '';
    let passwordField = '';
    const textInputs = [];

    let im;
    while ((im = inputRe.exec(formContent)) !== null) {
      const attrs = im[1];
      const name = getAttr(attrs, 'name');
      if (!name) continue;

      const type = (getAttr(attrs, 'type') || 'text').toLowerCase();
      const value = getAttr(attrs, 'value') || '';

      fields[name] = value;

      if (type === 'password') {
        passwordField = name;
      } else if (type === 'text' || type === 'email') {
        textInputs.push(name);
        const lower = name.toLowerCase();
        if (
          lower.includes('user') || lower.includes('login') ||
          lower.includes('id')    || lower.includes('member') ||
          lower.includes('email') || lower.includes('phone') ||
          lower.includes('mobile')|| lower.includes('telephone') ||
          lower.includes('account')
        ) {
          if (!usernameField) usernameField = name;
        }
      }
    }

    if (!usernameField && passwordField) {
      const cands = textInputs.filter(n => n !== passwordField);
      if (cands.length > 0) usernameField = cands[0];
    }

    return { action, method, fields, usernameField, passwordField };
  });

  return parsed.find(f => f.passwordField) || parsed[0];
}

// ────────────────────────────────────────────────────────────────────────────
// Config persistence
// ────────────────────────────────────────────────────────────────────────────
export function getConfigPath() {
  return path.join(os.homedir(), '.capauto', 'config.json');
}

export function loadConfig(configPath = getConfigPath()) {
  try {
    if (fs.existsSync(configPath)) {
      return JSON.parse(fs.readFileSync(configPath, 'utf8'));
    }
  } catch (e) {
    console.warn(`[Warning] Failed to parse config at "${configPath}":`, e.message);
  }
  return {};
}

export function saveCredentials(ssid, loginUrl, formDetails, username, password, configPath = getConfigPath()) {
  const dir = path.dirname(configPath);
  if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true, mode: 0o700 });

  const config = loadConfig(configPath);
  config[ssid] = {
    loginUrl,
    username,
    password,
    usernameField: formDetails.usernameField,
    passwordField: formDetails.passwordField,
    staticFields: formDetails.fields,
    action: formDetails.action,
    method: formDetails.method || 'POST',
  };

  fs.writeFileSync(configPath, JSON.stringify(config, null, 2), { mode: 0o600, encoding: 'utf8' });
  if (process.platform !== 'win32') {
    try { fs.chmodSync(configPath, 0o600); } catch (_) { /* ignore */ }
  }
}

export function promptUser(query) {
  if (!process.stdin.isTTY) {
    return Promise.reject(new Error('Terminal is non-interactive — cannot prompt for credentials'));
  }
  const rl = readline.createInterface({ input: process.stdin, output: process.stdout });
  return new Promise(resolve =>
    rl.question(query, ans => { rl.close(); resolve(ans); })
  );
}

// ────────────────────────────────────────────────────────────────────────────
// HTTP submission (with cookie support)
// ────────────────────────────────────────────────────────────────────────────
export async function submitLogin(action, method, payload, cookieJar) {
  const m = (method || 'POST').toUpperCase();
  let targetUrl = action;
  const options = {
    method: m,
    headers: {
      'User-Agent':
        'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
    },
    redirect: 'follow',
    signal: AbortSignal.timeout(15000),
  };

  const cookieHeader = cookieJar?.header();
  if (cookieHeader) options.headers['Cookie'] = cookieHeader;

  if (m === 'GET') {
    const u = new URL(action);
    const params = new URLSearchParams(payload);
    for (const [k, v] of params) u.searchParams.set(k, v);
    targetUrl = u.href;
  } else if (m === 'POST') {
    options.headers['Content-Type'] = 'application/x-www-form-urlencoded';
    options.body = new URLSearchParams(payload).toString();
  }

  const res = await fetch(targetUrl, options);
  if (cookieJar) cookieJar.capture(res);
  return res;
}

// ────────────────────────────────────────────────────────────────────────────
// Main orchestrator
// ────────────────────────────────────────────────────────────────────────────
export async function runAutomator(
  configPath = getConfigPath(),
  probeUrl = 'http://captive.apple.com/hotspot-detect.html'
) {
  let ssid = process.env.CAPAUTO_TEST_SSID;
  if (!ssid) {
    try { ssid = getSSID(); } catch (_) { ssid = 'Unknown_WiFi'; }
  }
  console.log(`[${new Date().toISOString()}] Checking network connection on Wi-Fi: "${ssid}"...`);

  // --- 1. Are we online / behind a captive portal? ------------------------
  let status = await checkConnectivity(probeUrl);
  if (status.online) {
    console.log('Already online. Exiting.');
    return true;
  }

  // Retry a few times in case the interface is still coming up.
  let attempts = 1;
  while (!status.online && !status.redirectUrl && attempts < 3) {
    console.log(`Offline but no portal detected yet (attempt ${attempts}). Retrying in 2 s…`);
    await new Promise(r => setTimeout(r, 2000));
    status = await checkConnectivity(probeUrl);
    attempts++;
  }

  if (status.online) {
    console.log('Already online. Exiting.');
    return true;
  }
  if (!status.redirectUrl) {
    console.log('Not online, and no captive-portal redirection detected. Exiting.');
    return false;
  }

  console.log(`Captive portal detected! Login page: ${status.redirectUrl}`);

  // --- 2. Fetch the *actual* login page (always, never trust saved URL) --
  const cookieJar = createCookieJar();
  let html = '';
  let finalLoginUrl = status.redirectUrl;
  try {
    const pageRes = await fetch(status.redirectUrl, {
      redirect: 'follow',
      signal: AbortSignal.timeout(15000),
    });
    cookieJar.capture(pageRes);
    html = await pageRes.text();
    finalLoginUrl = pageRes.url || status.redirectUrl;
  } catch (err) {
    console.error('Failed to fetch captive-portal login page:', err.message);
    return false;
  }

  if (process.argv.includes('--debug')) {
    console.log('\n--- DEBUG: Portal page HTML (first 2000 chars) ---');
    console.log(html.substring(0, 2000));
    console.log('--- END DEBUG ---\n');
  }

  // --- 3. Parse the form (always re-parse; fall back to saved fields) ----
  const config = loadConfig(configPath);
  const saved = config[ssid];

  let formDetails;
  try {
    formDetails = parseLoginForm(html, finalLoginUrl);
    console.log(`Found form action: "${formDetails.action}"`);
  } catch (err) {
    console.error('Failed to parse login form automatically:', err.message);
    if (saved) {
      console.log('Falling back to previously saved form details.');
      formDetails = {
        action: saved.action,
        method: saved.method,
        fields: saved.staticFields || {},
        usernameField: saved.usernameField,
        passwordField: saved.passwordField,
      };
    } else {
      console.log('Using standard username/password fields. Run with --debug to inspect the page.');
      formDetails = {
        action: finalLoginUrl,
        method: 'POST',
        fields: {},
        usernameField: 'username',
        passwordField: 'password',
      };
    }
  }

  console.log(`Detected username field: "${formDetails.usernameField}"`);
  console.log(`Detected password field: "${formDetails.passwordField}"`);

  // --- 4. Obtain credentials ---------------------------------------------
  let username, password;
  if (saved && saved.username && saved.password) {
    username = saved.username;
    password = saved.password;
    console.log('Using saved credentials.');
  } else {
    if (!saved) console.log(`No saved credentials for SSID: "${ssid}". First-time setup…`);
    username = await promptUser('Enter your Wi-Fi username/ID: ');
    password = await promptUser('Enter your Wi-Fi password: ');
    if (!username || !password) {
      console.error('Credentials cannot be empty. Aborting.');
      return false;
    }
  }

  // Persist / refresh saved credentials so future runs are seamless.
  saveCredentials(ssid, finalLoginUrl, formDetails, username, password, configPath);
  console.log(`Credentials saved for SSID: "${ssid}"`);

  // --- 5. Submit the login ------------------------------------------------
  const payload = {
    ...formDetails.fields,
    [formDetails.usernameField]: username,
    [formDetails.passwordField]: password,
  };

  console.log(`Submitting login request to: ${formDetails.action}…`);
  try {
    await submitLogin(formDetails.action, formDetails.method, payload, cookieJar);
    console.log('Form submitted. Verifying internet connectivity…');

    // Give the portal a moment to register the session.
    await new Promise(r => setTimeout(r, 2500));

    const recheck = await checkConnectivity(probeUrl);
    if (recheck.online) {
      console.log('Success! Internet connection established.');
      return true;
    }
    console.error('Still redirected after login. Login may have failed or the portal needs extra steps.');
    console.error('Tip: run with --debug to inspect the portal HTML.');
    return false;
  } catch (err) {
    console.error('Error submitting login form:', err.message);
    return false;
  }
}

// ────────────────────────────────────────────────────────────────────────────
// Self-execute when run directly
// ────────────────────────────────────────────────────────────────────────────
const isMain = (() => {
  try {
    return process.argv[1] && fs.realpathSync(process.argv[1]) === fileURLToPath(import.meta.url);
  } catch (_) {
    return false;
  }
})();

if (isMain) {
  runAutomator()
    .then(ok => process.exit(ok ? 0 : 1))
    .catch(err => { console.error('Fatal execution error:', err); process.exit(1); });
}
