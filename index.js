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
