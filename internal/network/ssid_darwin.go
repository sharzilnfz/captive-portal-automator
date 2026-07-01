//go:build darwin

package network

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// GetSSID returns the SSID of the currently connected Wi-Fi network on macOS.
// It tries three detection strategies in order:
//  1. networksetup -listallhardwareports to find the Wi-Fi device, then -getairportnetwork
//  2. Iterate en0–en9 interfaces trying -getairportnetwork on each
//  3. system_profiler SPAirPortDataType as a last resort
//
// Returns ErrNoWiFi if no Wi-Fi connection can be detected.
func GetSSID() (string, error) {
	// Strategy 1: Find Wi-Fi device via networksetup, then query it.
	if ssid, err := getSSIDViaNetworkSetup(); err == nil && ssid != "" {
		return ssid, nil
	}

	// Strategy 2: Brute-force en* interfaces.
	if ssid, err := getSSIDViaEnInterfaces(); err == nil && ssid != "" {
		return ssid, nil
	}

	// Strategy 3: system_profiler fallback.
	if ssid, err := getSSIDViaSystemProfiler(); err == nil && ssid != "" {
		return ssid, nil
	}

	return "", ErrNoWiFi
}

// getSSIDViaNetworkSetup finds the Wi-Fi hardware port device name, then
// queries its current network via networksetup -getairportnetwork.
func getSSIDViaNetworkSetup() (string, error) {
	out, err := exec.Command("networksetup", "-listallhardwareports").Output()
	if err != nil {
		return "", fmt.Errorf("listallhardwareports: %w", err)
	}

	device := parseWiFiDevice(string(out))
	if device == "" {
		return "", fmt.Errorf("no Wi-Fi device found in hardware ports")
	}

	return queryAirportNetwork(device)
}

// parseWiFiDevice extracts the Device name (e.g. "en0") for the Wi-Fi
// hardware port from the output of `networksetup -listallhardwareports`.
func parseWiFiDevice(output string) string {
	lines := strings.Split(output, "\n")
	foundWiFi := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.Contains(trimmed, "Wi-Fi") || strings.Contains(trimmed, "AirPort") {
			foundWiFi = true
			continue
		}

		if foundWiFi && strings.HasPrefix(trimmed, "Device:") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "Device:"))
		}

		// Reset if we hit another Hardware Port section without finding a device.
		if foundWiFi && strings.HasPrefix(trimmed, "Hardware Port:") {
			foundWiFi = false
		}
	}

	return ""
}

// queryAirportNetwork runs `networksetup -getairportnetwork <device>` and
// parses the SSID from the output.
func queryAirportNetwork(device string) (string, error) {
	out, err := exec.Command("networksetup", "-getairportnetwork", device).Output()
	if err != nil {
		return "", fmt.Errorf("getairportnetwork %s: %w", device, err)
	}

	output := strings.TrimSpace(string(out))

	// Output format: "Current Wi-Fi Network: <SSID>"
	if strings.Contains(output, "You are not associated with an AirPort network") {
		return "", fmt.Errorf("not associated with network on %s", device)
	}

	const prefix = "Current Wi-Fi Network: "
	if strings.HasPrefix(output, prefix) {
		ssid := strings.TrimPrefix(output, prefix)
		ssid = strings.TrimSpace(ssid)
		if ssid != "" {
			return ssid, nil
		}
	}

	return "", fmt.Errorf("unexpected getairportnetwork output: %s", output)
}

// getSSIDViaEnInterfaces iterates en0 through en9 trying -getairportnetwork
// on each interface.
func getSSIDViaEnInterfaces() (string, error) {
	for i := 0; i < 10; i++ {
		device := fmt.Sprintf("en%d", i)
		if ssid, err := queryAirportNetwork(device); err == nil && ssid != "" {
			return ssid, nil
		}
	}
	return "", fmt.Errorf("no SSID found on any en* interface")
}

// getSSIDViaSystemProfiler uses system_profiler SPAirPortDataType to extract
// the current SSID as a last resort.
func getSSIDViaSystemProfiler() (string, error) {
	out, err := exec.Command("system_profiler", "SPAirPortDataType").Output()
	if err != nil {
		return "", fmt.Errorf("system_profiler: %w", err)
	}

	// Look for "Current Network Information:" section followed by an SSID line.
	// The SSID appears as an indented key followed by a colon.
	output := string(out)

	// Pattern: find lines after "Current Network Information:" — the next
	// indented non-empty line that ends with ":" is the SSID.
	re := regexp.MustCompile(`(?m)Current Network Information:\s*\n\s+(\S[^:]+):`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		ssid := strings.TrimSpace(matches[1])
		if ssid != "" {
			return ssid, nil
		}
	}

	return "", fmt.Errorf("no SSID found in system_profiler output")
}
