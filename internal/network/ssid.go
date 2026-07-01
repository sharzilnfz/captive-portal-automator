package network

import "errors"

// ErrNoWiFi is returned when the system is not connected to any Wi-Fi network
// or when Wi-Fi detection is not possible on the current platform.
var ErrNoWiFi = errors.New("network: not connected to Wi-Fi")
