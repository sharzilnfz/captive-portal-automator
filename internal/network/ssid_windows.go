//go:build windows

package network

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetSSID returns the SSID of the currently connected Wi-Fi network on Windows.
// It parses the output of `netsh wlan show interfaces` to find the SSID line.
// Returns ErrNoWiFi if no Wi-Fi connection can be detected.
func GetSSID() (string, error) {
	out, err := exec.Command("netsh", "wlan", "show", "interfaces").Output()
	if err != nil {
		return "", fmt.Errorf("netsh wlan show interfaces: %w", err)
	}

	ssid := parseNetshSSID(string(out))
	if ssid == "" {
		return "", ErrNoWiFi
	}

	return ssid, nil
}

// parseNetshSSID extracts the SSID from the output of
// `netsh wlan show interfaces`. The relevant line looks like:
//
//	SSID                   : MyNetwork
//
// It specifically matches "SSID" but not "BSSID" to avoid confusion.
func parseNetshSSID(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip BSSID lines — they start with "BSSID".
		if strings.HasPrefix(trimmed, "BSSID") {
			continue
		}

		if strings.HasPrefix(trimmed, "SSID") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				// Verify the key is exactly "SSID" (not a substring match).
				key := strings.TrimSpace(parts[0])
				if key == "SSID" {
					ssid := strings.TrimSpace(parts[1])
					if ssid != "" {
						return ssid
					}
				}
			}
		}
	}

	return ""
}
