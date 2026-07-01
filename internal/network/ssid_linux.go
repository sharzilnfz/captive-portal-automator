//go:build linux

package network

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetSSID returns the SSID of the currently connected Wi-Fi network on Linux.
// It tries two detection strategies:
//  1. nmcli -t -f active,ssid dev wifi (NetworkManager)
//  2. iwgetid -r (wireless-tools fallback)
//
// Returns ErrNoWiFi if no Wi-Fi connection can be detected.
func GetSSID() (string, error) {
	// Strategy 1: nmcli (NetworkManager).
	if ssid, err := getSSIDViaNmcli(); err == nil && ssid != "" {
		return ssid, nil
	}

	// Strategy 2: iwgetid fallback.
	if ssid, err := getSSIDViaIwgetid(); err == nil && ssid != "" {
		return ssid, nil
	}

	return "", ErrNoWiFi
}

// getSSIDViaNmcli queries NetworkManager for the active Wi-Fi SSID.
// The output format is colon-separated: "yes:MyNetwork" for active connections.
func getSSIDViaNmcli() (string, error) {
	out, err := exec.Command("nmcli", "-t", "-f", "active,ssid", "dev", "wifi").Output()
	if err != nil {
		return "", fmt.Errorf("nmcli: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[0]) == "yes" {
			ssid := strings.TrimSpace(parts[1])
			if ssid != "" {
				return ssid, nil
			}
		}
	}

	return "", fmt.Errorf("no active SSID found in nmcli output")
}

// getSSIDViaIwgetid uses iwgetid -r to get the current SSID.
// This is a simpler fallback for systems without NetworkManager.
func getSSIDViaIwgetid() (string, error) {
	out, err := exec.Command("iwgetid", "-r").Output()
	if err != nil {
		return "", fmt.Errorf("iwgetid: %w", err)
	}

	ssid := strings.TrimSpace(string(out))
	if ssid == "" {
		return "", fmt.Errorf("iwgetid returned empty SSID")
	}

	return ssid, nil
}
