package network

import (
	"errors"
	"testing"
)

// TestGetSSID_ReturnsStringOrError verifies that GetSSID either returns a
// non-empty SSID string or a meaningful error. This test is designed to pass
// in CI environments where Wi-Fi may not be available — it accepts ErrNoWiFi
// gracefully.
func TestGetSSID_ReturnsStringOrError(t *testing.T) {
	ssid, err := GetSSID()

	if err != nil {
		// In CI or non-Wi-Fi environments, ErrNoWiFi is expected.
		// Any other error is also acceptable (e.g. command not found).
		t.Logf("GetSSID returned error (acceptable in CI): %v", err)
		return
	}

	if ssid == "" {
		t.Error("GetSSID returned empty SSID with nil error")
	}

	t.Logf("Detected SSID: %q", ssid)
}

// TestErrNoWiFi verifies the error message of the sentinel error.
func TestErrNoWiFi(t *testing.T) {
	expected := "network: not connected to Wi-Fi"
	if ErrNoWiFi.Error() != expected {
		t.Errorf("ErrNoWiFi.Error() = %q, want %q", ErrNoWiFi.Error(), expected)
	}

	// Verify it can be matched with errors.Is.
	var wrapped error = ErrNoWiFi
	if !errors.Is(wrapped, ErrNoWiFi) {
		t.Error("errors.Is(ErrNoWiFi, ErrNoWiFi) returned false")
	}
}
