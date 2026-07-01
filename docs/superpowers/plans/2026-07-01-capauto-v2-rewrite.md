# CapAuto v2 Go Rewrite — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite the captive-portal-automator from a monolithic Node.js script into a modular, cross-platform Go binary with real HTML parsing, OS keychain credential storage, structured logging, and self-installing daemon support.

**Architecture:** Pipeline of 7 focused packages: network → prober → portal → auth (with credential store and logger as cross-cutting concerns). Single CLI entry point orchestrates the pipeline. Each package has its own tests.

**Tech Stack:** Go 1.21+, `golang.org/x/net/html` (DOM parser), `github.com/zalando/go-keyring` (OS keychain)

## Global Constraints

- Go module path: `github.com/sharzilnafis/capauto`
- Minimum Go version: 1.21 (for `slices`, `slog`, etc.)
- All packages under `internal/` — no exported API surface
- Every package has `_test.go` files with table-driven tests
- All functions return `(result, error)` — no panics except truly unrecoverable situations
- Errors wrapped with context: `fmt.Errorf("package: operation: %w", err)`
- No external dependencies beyond `golang.org/x/net/html` and `github.com/zalando/go-keyring`
- Use Go's `log/slog` (stdlib since Go 1.21) instead of custom logger
- File permissions: config dirs 0700, config files 0600
- Credentials NEVER appear in logs at any level
- Build tags for platform-specific files: `//go:build darwin`, `//go:build windows`, `//go:build linux`

---

### Task 1: Project Scaffold, Go Module & Logger

**Files:**
- Create: `go.mod`
- Create: `go.sum` (auto-generated)
- Create: `internal/log/logger.go`
- Create: `internal/log/logger_test.go`
- Create: `cmd/capauto/main.go` (minimal placeholder)
- Create: `Makefile`

**Interfaces:**
- Consumes: nothing (foundation task)
- Produces:
  - `internal/log.New(level slog.Level, format string) *slog.Logger` — creates a configured logger
  - `internal/log.SetupFileLogging(logPath string, maxBytes int64, maxFiles int) (*slog.Logger, error)` — creates a file-backed logger with rotation
  - Go module initialized with dependencies fetched

- [ ] **Step 1: Initialize Go module and fetch dependencies**

```bash
cd /Users/sharzilnafis/Desktop/Project/captive-portal-automator
go mod init github.com/sharzilnafis/capauto
go get golang.org/x/net/html
go get github.com/zalando/go-keyring
```

- [ ] **Step 2: Create the logger package**

Create `internal/log/logger.go`:

```go
package log

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"fmt"
)

// New creates a configured slog.Logger.
// format can be "text" (human-readable) or "json" (structured).
func New(level slog.Level, format string, w io.Writer) *slog.Logger {
	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}
	return slog.New(handler)
}

// SetupFileLogging creates a file-backed logger with simple rotation.
// When the log file exceeds maxBytes, it is rotated. Up to maxFiles old logs are kept.
func SetupFileLogging(logPath string, maxBytes int64, maxFiles int, level slog.Level) (*slog.Logger, error) {
	dir := filepath.Dir(logPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("log: create dir %s: %w", dir, err)
	}

	if err := rotateIfNeeded(logPath, maxBytes, maxFiles); err != nil {
		return nil, fmt.Errorf("log: rotate: %w", err)
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("log: open %s: %w", logPath, err)
	}

	return New(level, "json", f), nil
}

func rotateIfNeeded(logPath string, maxBytes int64, maxFiles int) error {
	info, err := os.Stat(logPath)
	if err != nil {
		return nil // file doesn't exist yet, nothing to rotate
	}
	if info.Size() < maxBytes {
		return nil
	}

	// Rotate: rename current to .1, .1 to .2, etc.
	// Delete oldest if exceeding maxFiles
	for i := maxFiles; i >= 1; i-- {
		old := fmt.Sprintf("%s.%d", logPath, i)
		newer := logPath
		if i > 1 {
			newer = fmt.Sprintf("%s.%d", logPath, i-1)
		}
		if i == maxFiles {
			os.Remove(old)
		}
		if _, err := os.Stat(newer); err == nil {
			os.Rename(newer, old)
		}
	}
	return nil
}

// CleanOldLogs removes rotated log files beyond maxFiles count.
func CleanOldLogs(logPath string, maxFiles int) error {
	dir := filepath.Dir(logPath)
	base := filepath.Base(logPath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	var logFiles []string
	for _, e := range entries {
		if matched, _ := filepath.Match(base+".*", e.Name()); matched {
			logFiles = append(logFiles, filepath.Join(dir, e.Name()))
		}
	}

	sort.Sort(sort.Reverse(sort.StringSlice(logFiles)))
	for i := maxFiles; i < len(logFiles); i++ {
		os.Remove(logFiles[i])
	}
	return nil
}
```

- [ ] **Step 3: Write logger tests**

Create `internal/log/logger_test.go`:

```go
package log

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew_TextFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := New(slog.LevelInfo, "text", &buf)
	logger.Info("hello", "key", "value")
	output := buf.String()
	if !strings.Contains(output, "hello") {
		t.Errorf("expected 'hello' in output, got: %s", output)
	}
	if !strings.Contains(output, "key=value") {
		t.Errorf("expected 'key=value' in output, got: %s", output)
	}
}

func TestNew_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := New(slog.LevelInfo, "json", &buf)
	logger.Info("test-msg", "foo", "bar")
	output := buf.String()
	if !strings.Contains(output, `"msg":"test-msg"`) {
		t.Errorf("expected JSON msg field, got: %s", output)
	}
}

func TestNew_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := New(slog.LevelWarn, "text", &buf)
	logger.Info("should-not-appear")
	logger.Warn("should-appear")
	output := buf.String()
	if strings.Contains(output, "should-not-appear") {
		t.Error("info message should be filtered at warn level")
	}
	if !strings.Contains(output, "should-appear") {
		t.Error("warn message should appear at warn level")
	}
}

func TestSetupFileLogging(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")

	logger, err := SetupFileLogging(logPath, 1024*1024, 3, slog.LevelInfo)
	if err != nil {
		t.Fatalf("SetupFileLogging failed: %v", err)
	}
	logger.Info("file-log-test")

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	if !strings.Contains(string(data), "file-log-test") {
		t.Errorf("expected log content in file, got: %s", string(data))
	}
}

func TestRotateIfNeeded(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "rotate.log")

	// Create a file that exceeds the threshold
	os.WriteFile(logPath, bytes.Repeat([]byte("x"), 100), 0600)

	err := rotateIfNeeded(logPath, 50, 3)
	if err != nil {
		t.Fatalf("rotateIfNeeded failed: %v", err)
	}

	// Original file should have been renamed to .1
	if _, err := os.Stat(logPath + ".1"); os.IsNotExist(err) {
		t.Error("expected rotated file .1 to exist")
	}
	// Original path should no longer exist (it was renamed)
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Error("expected original file to be renamed")
	}
}
```

- [ ] **Step 4: Create minimal main.go placeholder**

Create `cmd/capauto/main.go`:

```go
package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "capauto: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	fmt.Println("capauto v2 - captive portal automator")
	fmt.Println("(under construction)")
	return nil
}
```

- [ ] **Step 5: Create Makefile**

Create `Makefile`:

```makefile
.PHONY: build test lint clean install cross-compile

BINARY := capauto
MODULE := github.com/sharzilnafis/capauto
VERSION := 2.0.0
BUILD_DIR := build

build:
	go build -o $(BINARY) ./cmd/capauto

test:
	go test -v -race -cover ./...

lint:
	go vet ./...

clean:
	rm -f $(BINARY)
	rm -rf $(BUILD_DIR)

install: build
	cp $(BINARY) /usr/local/bin/$(BINARY)

cross-compile:
	mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 ./cmd/capauto
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 ./cmd/capauto
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY)-linux-amd64 ./cmd/capauto
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe ./cmd/capauto
```

- [ ] **Step 6: Run tests and verify**

```bash
go test -v -race ./internal/log/
```

Expected: all tests pass.

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "feat: project scaffold with Go module, logger package, and Makefile"
```

---

### Task 2: Network SSID Detection

**Files:**
- Create: `internal/network/ssid.go` (shared types + dispatcher)
- Create: `internal/network/ssid_darwin.go` (macOS implementation)
- Create: `internal/network/ssid_windows.go` (Windows implementation)
- Create: `internal/network/ssid_linux.go` (Linux implementation)
- Create: `internal/network/ssid_test.go`

**Interfaces:**
- Consumes: nothing
- Produces:
  - `func GetSSID() (string, error)` — returns current Wi-Fi SSID
  - `var ErrNoWiFi = errors.New("network: not connected to Wi-Fi")`

- [ ] **Step 1: Create shared types and error**

Create `internal/network/ssid.go`:

```go
package network

import "errors"

// ErrNoWiFi is returned when the device is not connected to Wi-Fi.
var ErrNoWiFi = errors.New("network: not connected to Wi-Fi")
```

- [ ] **Step 2: Create macOS SSID detection**

Create `internal/network/ssid_darwin.go`:

```go
//go:build darwin

package network

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// GetSSID returns the current Wi-Fi SSID on macOS.
func GetSSID() (string, error) {
	// Strategy 1: Find Wi-Fi device via networksetup, then query it
	if ssid, err := getSSIDViaNetworkSetup(); err == nil && ssid != "" {
		return ssid, nil
	}

	// Strategy 2: Try all en* interfaces
	if ssid, err := getSSIDViaInterfaces(); err == nil && ssid != "" {
		return ssid, nil
	}

	// Strategy 3: system_profiler (slow but reliable on Sonoma+)
	if ssid, err := getSSIDViaProfiler(); err == nil && ssid != "" {
		return ssid, nil
	}

	return "", ErrNoWiFi
}

func getSSIDViaNetworkSetup() (string, error) {
	ports, err := exec.Command("networksetup", "-listallhardwareports").Output()
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile(`(?i)Hardware Port:\s*Wi-Fi[\s\S]*?Device:\s*(\w+)`)
	m := re.FindSubmatch(ports)
	if m == nil {
		return "", fmt.Errorf("no Wi-Fi device found")
	}
	device := strings.TrimSpace(string(m[1]))

	out, err := exec.Command("networksetup", "-getairportnetwork", device).Output()
	if err != nil {
		return "", err
	}

	ssidRe := regexp.MustCompile(`(?i)Current Wi-Fi Network\s*:\s*(.+)`)
	sm := ssidRe.FindSubmatch(out)
	if sm == nil {
		return "", fmt.Errorf("no current network")
	}
	return strings.TrimSpace(string(sm[1])), nil
}

func getSSIDViaInterfaces() (string, error) {
	out, err := exec.Command("ifconfig", "-l").Output()
	if err != nil {
		return "", err
	}

	devRe := regexp.MustCompile(`en\d+`)
	devices := devRe.FindAllString(string(out), -1)

	ssidRe := regexp.MustCompile(`(?i)Current Wi-Fi Network\s*:\s*(.+)`)
	for _, dev := range devices {
		out, err := exec.Command("networksetup", "-getairportnetwork", dev).Output()
		if err != nil {
			continue
		}
		m := ssidRe.FindSubmatch(out)
		if m != nil {
			return strings.TrimSpace(string(m[1])), nil
		}
	}
	return "", fmt.Errorf("no SSID found on any interface")
}

func getSSIDViaProfiler() (string, error) {
	out, err := exec.Command("system_profiler", "SPAirPortDataType").Output()
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile(`(?i)Current Network\s*:\s*(.+)`)
	m := re.FindSubmatch(out)
	if m != nil {
		return strings.TrimSpace(string(m[1])), nil
	}

	// Sonoma variant
	re2 := regexp.MustCompile(`(?m)Current Network Information:[\s\S]*?\n\s+([^\n:]+):`)
	m2 := re2.FindSubmatch(out)
	if m2 != nil {
		return strings.TrimSpace(string(m2[1])), nil
	}

	return "", fmt.Errorf("no SSID in profiler output")
}
```

- [ ] **Step 3: Create Windows SSID detection**

Create `internal/network/ssid_windows.go`:

```go
//go:build windows

package network

import (
	"os/exec"
	"regexp"
	"strings"
)

// GetSSID returns the current Wi-Fi SSID on Windows.
func GetSSID() (string, error) {
	out, err := exec.Command("netsh", "wlan", "show", "interfaces").Output()
	if err != nil {
		return "", ErrNoWiFi
	}

	re := regexp.MustCompile(`(?m)^\s*SSID\s*:\s*(.+)`)
	m := re.FindSubmatch(out)
	if m == nil {
		return "", ErrNoWiFi
	}
	return strings.TrimSpace(string(m[1])), nil
}
```

- [ ] **Step 4: Create Linux SSID detection**

Create `internal/network/ssid_linux.go`:

```go
//go:build linux

package network

import (
	"os/exec"
	"strings"
)

// GetSSID returns the current Wi-Fi SSID on Linux.
func GetSSID() (string, error) {
	// Strategy 1: nmcli (NetworkManager)
	out, err := exec.Command("nmcli", "-t", "-f", "active,ssid", "dev", "wifi").Output()
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(line, "yes:") {
				ssid := strings.TrimPrefix(line, "yes:")
				ssid = strings.TrimSpace(ssid)
				if ssid != "" {
					return ssid, nil
				}
			}
		}
	}

	// Strategy 2: iwgetid
	out, err = exec.Command("iwgetid", "-r").Output()
	if err == nil {
		ssid := strings.TrimSpace(string(out))
		if ssid != "" {
			return ssid, nil
		}
	}

	return "", ErrNoWiFi
}
```

- [ ] **Step 5: Create SSID tests**

Create `internal/network/ssid_test.go`:

```go
package network

import (
	"errors"
	"testing"
)

func TestGetSSID_ReturnsStringOrError(t *testing.T) {
	ssid, err := GetSSID()

	if err != nil {
		// If error, it should be ErrNoWiFi or wrapped
		if !errors.Is(err, ErrNoWiFi) {
			// Other errors are acceptable (command not found, etc.)
			t.Logf("GetSSID returned non-WiFi error (acceptable in CI): %v", err)
		} else {
			t.Logf("GetSSID returned ErrNoWiFi (acceptable if no WiFi)")
		}
		return
	}

	if ssid == "" {
		t.Error("GetSSID returned empty string without error")
	}
	t.Logf("Detected SSID: %q", ssid)
}

func TestErrNoWiFi(t *testing.T) {
	if ErrNoWiFi == nil {
		t.Fatal("ErrNoWiFi should not be nil")
	}
	if ErrNoWiFi.Error() != "network: not connected to Wi-Fi" {
		t.Errorf("unexpected error message: %s", ErrNoWiFi.Error())
	}
}
```

- [ ] **Step 6: Run tests and commit**

```bash
go test -v ./internal/network/
git add -A
git commit -m "feat: cross-platform SSID detection (macOS/Windows/Linux)"
```

---

### Task 3: Connectivity Prober

**Files:**
- Create: `internal/prober/endpoints.go`
- Create: `internal/prober/prober.go`
- Create: `internal/prober/prober_test.go`

**Interfaces:**
- Consumes: nothing
- Produces:
  - `type ProbeResult struct { Online bool; PortalURL string; HTML string; FinalURL string }`
  - `func NewProber(client *http.Client, logger *slog.Logger) *Prober`
  - `func (p *Prober) Check(ctx context.Context) ProbeResult`

- [ ] **Step 1: Create endpoint definitions**

Create `internal/prober/endpoints.go`:

```go
package prober

import (
	"regexp"
	"strings"
)

// Endpoint defines a connectivity probe target.
type Endpoint struct {
	URL       string
	CheckFunc func(status int, body string, redirected bool) bool
}

// DefaultEndpoints returns the standard probe endpoints.
func DefaultEndpoints() []Endpoint {
	return []Endpoint{
		{
			URL: "http://captive.apple.com/hotspot-detect.html",
			CheckFunc: func(status int, body string, redirected bool) bool {
				return !redirected && regexp.MustCompile(`(?i)<body>\s*success\s*</body>`).MatchString(body)
			},
		},
		{
			URL: "http://connectivitycheck.gstatic.com/generate_204",
			CheckFunc: func(status int, body string, redirected bool) bool {
				return status == 204
			},
		},
		{
			URL: "http://www.msftconnecttest.com/connecttest.txt",
			CheckFunc: func(status int, body string, redirected bool) bool {
				return !redirected && strings.TrimSpace(body) == "Microsoft Connect Test"
			},
		},
		{
			URL: "http://detectportal.firefox.com/canonical.html",
			CheckFunc: func(status int, body string, redirected bool) bool {
				return !redirected && strings.Contains(body, "success")
			},
		},
		{
			URL: "http://neverssl.com/",
			CheckFunc: func(status int, body string, redirected bool) bool {
				return !redirected && status == 200 && strings.Contains(body, "NeverSSL")
			},
		},
	}
}
```

- [ ] **Step 2: Create prober implementation**

Create `internal/prober/prober.go`:

```go
package prober

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// ProbeResult represents the outcome of a connectivity check.
type ProbeResult struct {
	Online    bool
	PortalURL string
	HTML      string
	FinalURL  string
}

// Prober checks internet connectivity against known endpoints.
type Prober struct {
	client    *http.Client
	endpoints []Endpoint
	logger    *slog.Logger
}

// NewProber creates a Prober with the given HTTP client and logger.
func NewProber(client *http.Client, logger *slog.Logger) *Prober {
	return &Prober{
		client:    client,
		endpoints: DefaultEndpoints(),
		logger:    logger,
	}
}

// Check determines if the device is online or behind a captive portal.
// It tries multiple endpoints with exponential backoff on inconclusive results.
func (p *Prober) Check(ctx context.Context) ProbeResult {
	maxRetries := 3

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := backoff(attempt)
			p.logger.Debug("retrying probe", "attempt", attempt, "delay", delay)
			select {
			case <-ctx.Done():
				return ProbeResult{Online: false}
			case <-time.After(delay):
			}
		}

		// Try endpoints in order; use more endpoints on later attempts
		endCount := min(2+attempt, len(p.endpoints))
		for i := 0; i < endCount; i++ {
			ep := p.endpoints[i]
			result := p.probeEndpoint(ctx, ep)
			if result.Online {
				return result
			}
			if result.PortalURL != "" {
				return result
			}
		}
	}

	return ProbeResult{Online: false}
}

func (p *Prober) probeEndpoint(ctx context.Context, ep Endpoint) ProbeResult {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", ep.URL, nil)
	if err != nil {
		p.logger.Debug("probe request error", "url", ep.URL, "error", err)
		return ProbeResult{Online: false, FinalURL: ep.URL}
	}

	resp, err := p.client.Do(req)
	if err != nil {
		p.logger.Debug("probe fetch error", "url", ep.URL, "error", err)
		return ProbeResult{Online: false, FinalURL: ep.URL}
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	body := string(bodyBytes)
	finalURL := resp.Request.URL.String()
	redirected := finalURL != ep.URL

	// Check if the endpoint signals "online"
	if ep.CheckFunc(resp.StatusCode, body, redirected) {
		p.logger.Debug("probe online", "url", ep.URL)
		return ProbeResult{Online: true, FinalURL: finalURL}
	}

	// If redirected, the final URL is the portal login page
	if redirected {
		p.logger.Info("portal redirect detected", "from", ep.URL, "to", finalURL)
		return ProbeResult{Online: false, PortalURL: finalURL, HTML: body, FinalURL: finalURL}
	}

	// Intercepted response (200 but not the expected content) — try to extract portal URL
	if portalURL := extractPortalURL(body, ep.URL); portalURL != "" {
		p.logger.Info("portal URL extracted from HTML", "url", portalURL)
		return ProbeResult{Online: false, PortalURL: portalURL, HTML: body, FinalURL: finalURL}
	}

	p.logger.Debug("probe inconclusive", "url", ep.URL, "status", resp.StatusCode)
	return ProbeResult{Online: false, HTML: body, FinalURL: finalURL}
}

// extractPortalURL tries to find the portal login URL from intercepted HTML.
func extractPortalURL(html, baseURL string) string {
	if html == "" {
		return ""
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}

	// 1. <meta http-equiv="refresh" content="0;url=...">
	metaRe := regexp.MustCompile(`(?i)<meta[^>]+http-equiv\s*=\s*["']?refresh["']?[^>]+content\s*=\s*["'][^"']*url\s*=\s*([^"'\s>]+)`)
	if m := metaRe.FindStringSubmatch(html); m != nil {
		if u, err := base.Parse(m[1]); err == nil {
			return u.String()
		}
	}

	// Also try reversed attribute order
	metaRe2 := regexp.MustCompile(`(?i)<meta[^>]+content\s*=\s*["'][^"']*url\s*=\s*([^"'\s>]+)[^>]*http-equiv\s*=\s*["']?refresh["']?`)
	if m := metaRe2.FindStringSubmatch(html); m != nil {
		if u, err := base.Parse(m[1]); err == nil {
			return u.String()
		}
	}

	// 2. <form action="...">
	formRe := regexp.MustCompile(`(?i)<form[^>]+action\s*=\s*["']([^"']+)["']`)
	if m := formRe.FindStringSubmatch(html); m != nil {
		if u, err := base.Parse(m[1]); err == nil {
			return u.String()
		}
	}

	// 3. First <a href> pointing to a different host
	linkRe := regexp.MustCompile(`(?i)<a[^>]+href\s*=\s*["']([^"'#][^"']*)["']`)
	for _, m := range linkRe.FindAllStringSubmatch(html, -1) {
		if u, err := base.Parse(m[1]); err == nil {
			if u.Scheme == "http" || u.Scheme == "https" {
				if u.Hostname() != base.Hostname() {
					return u.String()
				}
			}
		}
	}

	// 4. JS redirect: location.href = "..." or location.replace("...")
	jsRe := regexp.MustCompile(`(?i)location(?:\.href|\.replace)\s*=\s*["']([^"']+)["']`)
	if m := jsRe.FindStringSubmatch(html); m != nil {
		if u, err := base.Parse(m[1]); err == nil {
			return u.String()
		}
	}

	return ""
}

func backoff(attempt int) time.Duration {
	base := 500 * time.Millisecond
	d := time.Duration(float64(base) * math.Pow(2, float64(attempt-1)))
	// Add ±25% jitter
	jitter := time.Duration(rand.Int63n(int64(d) / 2))
	d = d - time.Duration(int64(d)/4) + jitter
	if d > 8*time.Second {
		d = 8 * time.Second
	}
	return d
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

- [ ] **Step 3: Create prober tests**

Create `internal/prober/prober_test.go`:

```go
package prober

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestProber_Online(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, "<HTML><HEAD><TITLE>Success</TITLE></HEAD><BODY>Success</BODY></HTML>")
	}))
	defer srv.Close()

	p := NewProber(srv.Client(), testLogger())
	p.endpoints = []Endpoint{
		{URL: srv.URL, CheckFunc: DefaultEndpoints()[0].CheckFunc},
	}

	result := p.Check(context.Background())
	if !result.Online {
		t.Errorf("expected online=true, got false")
	}
	if result.PortalURL != "" {
		t.Errorf("expected empty portal URL, got %q", result.PortalURL)
	}
}

func TestProber_PortalRedirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/hotspot" {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		w.WriteHeader(200)
		fmt.Fprint(w, `<html><form action="/auth" method="POST"><input type="text" name="user"><input type="password" name="pass"></form></html>`)
	}))
	defer srv.Close()

	p := NewProber(srv.Client(), testLogger())
	p.endpoints = []Endpoint{
		{
			URL: srv.URL + "/hotspot",
			CheckFunc: func(status int, body string, redirected bool) bool {
				return false // never "online" for this test
			},
		},
	}

	result := p.Check(context.Background())
	if result.Online {
		t.Error("expected online=false")
	}
	if !strings.HasSuffix(result.PortalURL, "/login") && result.PortalURL != "" {
		// The redirect was followed, so FinalURL ends with /login
	}
	if result.FinalURL == "" {
		t.Error("expected non-empty final URL")
	}
}

func TestProber_204Online(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	}))
	defer srv.Close()

	p := NewProber(srv.Client(), testLogger())
	p.endpoints = []Endpoint{
		{URL: srv.URL, CheckFunc: DefaultEndpoints()[1].CheckFunc}, // Google 204 check
	}

	result := p.Check(context.Background())
	if !result.Online {
		t.Error("expected 204 to mean online")
	}
}

func TestExtractPortalURL_MetaRefresh(t *testing.T) {
	html := `<html><head><meta http-equiv="refresh" content="0;url=https://portal.example.com/login"></head></html>`
	url := extractPortalURL(html, "http://captive.apple.com/hotspot-detect.html")
	if url != "https://portal.example.com/login" {
		t.Errorf("expected portal URL, got %q", url)
	}
}

func TestExtractPortalURL_FormAction(t *testing.T) {
	html := `<html><form action="/auth" method="POST"><input type="text" name="user"></form></html>`
	url := extractPortalURL(html, "http://portal.example.com/login")
	if url != "http://portal.example.com/auth" {
		t.Errorf("expected form action URL, got %q", url)
	}
}

func TestExtractPortalURL_JSRedirect(t *testing.T) {
	html := `<script>location.href = "https://portal.example.com/login";</script>`
	url := extractPortalURL(html, "http://captive.apple.com/")
	if url != "https://portal.example.com/login" {
		t.Errorf("expected JS redirect URL, got %q", url)
	}
}

func TestExtractPortalURL_Empty(t *testing.T) {
	url := extractPortalURL("", "http://example.com")
	if url != "" {
		t.Errorf("expected empty, got %q", url)
	}
}

func TestBackoff(t *testing.T) {
	for i := 1; i <= 5; i++ {
		d := backoff(i)
		if d <= 0 {
			t.Errorf("attempt %d: backoff should be positive, got %v", i, d)
		}
		if d > 8*time.Second+time.Second {
			t.Errorf("attempt %d: backoff %v exceeds cap", i, d)
		}
	}
}
```

Add missing import at the top of the test file — `time` is needed for TestBackoff. The import block should be:

```go
import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)
```

- [ ] **Step 4: Run tests and commit**

```bash
go test -v -race ./internal/prober/
git add -A
git commit -m "feat: connectivity prober with multi-endpoint parallel detection"
```

---

### Task 4: DOM-Based Portal Form Parser

**Files:**
- Create: `internal/portal/parser.go`
- Create: `internal/portal/parser_test.go`

**Interfaces:**
- Consumes: nothing
- Produces:
  - `type FormData struct { Action, Method string; Fields map[string]string; UsernameField, PasswordField string; FormIndex int }`
  - `func ParseLoginForm(body io.Reader, baseURL string) (*FormData, error)`

- [ ] **Step 1: Create DOM-based parser**

Create `internal/portal/parser.go`:

```go
package portal

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// FormData represents a parsed captive portal login form.
type FormData struct {
	Action        string
	Method        string
	Fields        map[string]string
	UsernameField string
	PasswordField string
	FormIndex     int
}

var (
	ErrNoForm = errors.New("portal: no form element found on page")
)

// usernameKeywords are substrings that suggest a field is a username/ID input.
var usernameKeywords = []string{
	"user", "login", "email", "phone", "mobile",
	"member", "account", "id", "telephone",
}

// ParseLoginForm parses HTML from body to find the best login form.
// It uses a proper DOM parser (golang.org/x/net/html) and scores forms
// to pick the one most likely to be a login form.
func ParseLoginForm(body io.Reader, baseURL string) (*FormData, error) {
	doc, err := html.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("portal: parse HTML: %w", err)
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("portal: parse base URL: %w", err)
	}

	forms := findForms(doc)
	if len(forms) == 0 {
		return nil, ErrNoForm
	}

	var best *FormData
	bestScore := -1

	for i, formNode := range forms {
		fd := extractFormData(formNode, base, i)
		score := scoreForm(fd)
		if score > bestScore {
			bestScore = score
			best = fd
		}
	}

	if best == nil {
		return nil, ErrNoForm
	}
	return best, nil
}

// findForms walks the DOM tree and collects all <form> nodes.
func findForms(n *html.Node) []*html.Node {
	var forms []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "form" {
			forms = append(forms, n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return forms
}

// extractFormData extracts form metadata and input fields from a <form> node.
func extractFormData(formNode *html.Node, base *url.URL, index int) *FormData {
	fd := &FormData{
		Fields:    make(map[string]string),
		FormIndex: index,
	}

	// Extract form attributes
	fd.Action = getAttr(formNode, "action")
	fd.Method = strings.ToUpper(getAttr(formNode, "method"))
	if fd.Method == "" {
		fd.Method = "POST"
	}

	// Resolve action URL
	if fd.Action == "" {
		fd.Action = base.String()
	} else {
		if resolved, err := base.Parse(fd.Action); err == nil {
			fd.Action = resolved.String()
		}
	}

	// Walk children to find input elements
	var textInputs []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "input" || n.Data == "select" || n.Data == "textarea") {
			name := getAttr(n, "name")
			if name == "" {
				goto next
			}

			inputType := strings.ToLower(getAttr(n, "type"))
			if inputType == "" {
				inputType = "text"
			}
			value := getAttr(n, "value")
			fd.Fields[name] = value

			switch inputType {
			case "password":
				fd.PasswordField = name
			case "text", "email", "tel":
				textInputs = append(textInputs, name)
				lower := strings.ToLower(name)
				for _, kw := range usernameKeywords {
					if strings.Contains(lower, kw) {
						if fd.UsernameField == "" {
							fd.UsernameField = name
						}
						break
					}
				}
			}
		}
	next:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(formNode)

	// Fallback: if no keyword match, use first text input that isn't the password
	if fd.UsernameField == "" && fd.PasswordField != "" {
		for _, name := range textInputs {
			if name != fd.PasswordField {
				fd.UsernameField = name
				break
			}
		}
	}

	return fd
}

// scoreForm scores a form for likelihood of being a login form.
func scoreForm(fd *FormData) int {
	score := 0
	if fd.PasswordField != "" {
		score += 100
	}
	if fd.UsernameField != "" {
		score += 50
	}
	if fd.Method == "POST" {
		score += 10
	}
	if len(fd.Fields) > 0 {
		score += 5
	}
	return score
}

// getAttr retrieves an attribute value from an HTML node.
func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}
```

- [ ] **Step 2: Create parser tests**

Create `internal/portal/parser_test.go`:

```go
package portal

import (
	"strings"
	"testing"
)

func TestParseLoginForm_Basic(t *testing.T) {
	html := `<html><form action="/auth" method="POST">
		<input type="hidden" name="csrf" value="token123">
		<input type="text" name="username_field">
		<input type="password" name="password_field">
	</form></html>`

	fd, err := ParseLoginForm(strings.NewReader(html), "http://localhost:8080/login")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fd.Action != "http://localhost:8080/auth" {
		t.Errorf("action: want 'http://localhost:8080/auth', got %q", fd.Action)
	}
	if fd.Method != "POST" {
		t.Errorf("method: want POST, got %q", fd.Method)
	}
	if fd.UsernameField != "username_field" {
		t.Errorf("username: want 'username_field', got %q", fd.UsernameField)
	}
	if fd.PasswordField != "password_field" {
		t.Errorf("password: want 'password_field', got %q", fd.PasswordField)
	}
	if fd.Fields["csrf"] != "token123" {
		t.Errorf("csrf: want 'token123', got %q", fd.Fields["csrf"])
	}
}

func TestParseLoginForm_NoAction_GET(t *testing.T) {
	html := `<form method="GET">
		<input type="text" name="email_addr">
		<input type="password" name="pass">
	</form>`

	fd, err := ParseLoginForm(strings.NewReader(html), "http://localhost:8080/login")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fd.Action != "http://localhost:8080/login" {
		t.Errorf("action should fallback to base URL, got %q", fd.Action)
	}
	if fd.Method != "GET" {
		t.Errorf("method: want GET, got %q", fd.Method)
	}
	if fd.UsernameField != "email_addr" {
		t.Errorf("username: want 'email_addr', got %q", fd.UsernameField)
	}
	if fd.PasswordField != "pass" {
		t.Errorf("password: want 'pass', got %q", fd.PasswordField)
	}
}

func TestParseLoginForm_MultiForm_PrioritizesPassword(t *testing.T) {
	html := `<html>
		<form action="/lang" method="GET">
			<input type="submit" name="lang" value="en">
		</form>
		<form action="/login-submit" method="POST">
			<input type="text" name="email">
			<input type="password" name="pass">
		</form>
	</html>`

	fd, err := ParseLoginForm(strings.NewReader(html), "http://localhost:8080/login")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fd.Action != "http://localhost:8080/login-submit" {
		t.Errorf("should pick login form, got action %q", fd.Action)
	}
	if fd.UsernameField != "email" {
		t.Errorf("username: want 'email', got %q", fd.UsernameField)
	}
	if fd.PasswordField != "pass" {
		t.Errorf("password: want 'pass', got %q", fd.PasswordField)
	}
}

func TestParseLoginForm_FallbackUsername(t *testing.T) {
	html := `<form action="/login" method="POST">
		<input type="hidden" name="csrf" value="xyz">
		<input type="text" name="weird_field">
		<input type="password" name="pass">
	</form>`

	fd, err := ParseLoginForm(strings.NewReader(html), "http://localhost:8080/login")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fd.UsernameField != "weird_field" {
		t.Errorf("fallback username: want 'weird_field', got %q", fd.UsernameField)
	}
}

func TestParseLoginForm_PhoneKeyword(t *testing.T) {
	html := `<form action="/login" method="POST">
		<input type="text" name="phone">
		<input type="password" name="pass">
	</form>`

	fd, err := ParseLoginForm(strings.NewReader(html), "http://localhost:8080/login")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fd.UsernameField != "phone" {
		t.Errorf("username: want 'phone', got %q", fd.UsernameField)
	}
}

func TestParseLoginForm_NoForm(t *testing.T) {
	html := `<html><body><p>No form here</p></body></html>`

	_, err := ParseLoginForm(strings.NewReader(html), "http://localhost:8080/login")
	if err == nil {
		t.Error("expected error for page with no form")
	}
}

func TestParseLoginForm_AbsoluteAction(t *testing.T) {
	html := `<form action="https://auth.example.com/submit" method="POST">
		<input type="text" name="user">
		<input type="password" name="pwd">
	</form>`

	fd, err := ParseLoginForm(strings.NewReader(html), "http://portal.example.com/login")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fd.Action != "https://auth.example.com/submit" {
		t.Errorf("absolute action should be preserved, got %q", fd.Action)
	}
}

func TestParseLoginForm_HiddenFields(t *testing.T) {
	html := `<form action="/auth" method="POST">
		<input type="hidden" name="csrf" value="abc">
		<input type="hidden" name="session" value="xyz">
		<input type="hidden" name="redirect" value="/dashboard">
		<input type="text" name="user">
		<input type="password" name="pass">
	</form>`

	fd, err := ParseLoginForm(strings.NewReader(html), "http://localhost/login")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fd.Fields["csrf"] != "abc" {
		t.Errorf("csrf: want 'abc', got %q", fd.Fields["csrf"])
	}
	if fd.Fields["session"] != "xyz" {
		t.Errorf("session: want 'xyz', got %q", fd.Fields["session"])
	}
	if fd.Fields["redirect"] != "/dashboard" {
		t.Errorf("redirect: want '/dashboard', got %q", fd.Fields["redirect"])
	}
}
```

- [ ] **Step 3: Run tests and commit**

```bash
go test -v -race ./internal/portal/
git add -A
git commit -m "feat: DOM-based captive portal form parser"
```

---

### Task 5: Credential Store (Keychain + Encrypted Fallback)

**Files:**
- Create: `internal/credential/store.go` (interface + types)
- Create: `internal/credential/keychain.go` (OS keychain via go-keyring)
- Create: `internal/credential/file.go` (encrypted JSON fallback)
- Create: `internal/credential/migrate.go` (v1 plaintext migration)
- Create: `internal/credential/store_test.go`

**Interfaces:**
- Consumes: nothing
- Produces:
  - `type Credentials struct { SSID, Username, Password, UsernameField, PasswordField, FormAction, FormMethod string; StaticFields map[string]string; UpdatedAt time.Time }`
  - `type Store interface { Load(ssid string) (*Credentials, error); Save(creds *Credentials) error; Delete(ssid string) error; List() ([]string, error) }`
  - `func NewKeychainStore() Store`
  - `func NewFileStore(configPath string) Store`
  - `func MigrateV1Config(configPath string, store Store) error`

- [ ] **Step 1: Create store interface and types**

Create `internal/credential/store.go`:

```go
package credential

import (
	"encoding/json"
	"errors"
	"time"
)

// ErrNotFound is returned when credentials for an SSID don't exist.
var ErrNotFound = errors.New("credential: not found")

// Credentials holds login info for one SSID.
type Credentials struct {
	SSID          string            `json:"ssid"`
	Username      string            `json:"username"`
	Password      string            `json:"password"`
	UsernameField string            `json:"username_field"`
	PasswordField string            `json:"password_field"`
	FormAction    string            `json:"form_action"`
	FormMethod    string            `json:"form_method"`
	StaticFields  map[string]string `json:"static_fields,omitempty"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

// Store is the credential storage interface.
type Store interface {
	Load(ssid string) (*Credentials, error)
	Save(creds *Credentials) error
	Delete(ssid string) error
	List() ([]string, error)
}

// ServiceName is the keychain service identifier.
const ServiceName = "capauto"

// marshalCreds serializes credentials to JSON for storage.
func marshalCreds(c *Credentials) (string, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// unmarshalCreds deserializes credentials from JSON.
func unmarshalCreds(data string) (*Credentials, error) {
	var c Credentials
	if err := json.Unmarshal([]byte(data), &c); err != nil {
		return nil, err
	}
	return &c, nil
}
```

- [ ] **Step 2: Create keychain store**

Create `internal/credential/keychain.go`:

```go
package credential

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

// KeychainStore stores credentials in the OS keychain.
type KeychainStore struct{}

// NewKeychainStore returns a Store backed by the OS keychain.
func NewKeychainStore() Store {
	return &KeychainStore{}
}

func (k *KeychainStore) Load(ssid string) (*Credentials, error) {
	data, err := keyring.Get(ServiceName, ssid)
	if err != nil {
		if err == keyring.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("credential: keychain load %q: %w", ssid, err)
	}
	creds, err := unmarshalCreds(data)
	if err != nil {
		return nil, fmt.Errorf("credential: unmarshal %q: %w", ssid, err)
	}
	creds.SSID = ssid
	return creds, nil
}

func (k *KeychainStore) Save(creds *Credentials) error {
	data, err := marshalCreds(creds)
	if err != nil {
		return fmt.Errorf("credential: marshal: %w", err)
	}
	// Delete first to avoid "already exists" errors on some platforms
	_ = keyring.Delete(ServiceName, creds.SSID)
	if err := keyring.Set(ServiceName, creds.SSID, data); err != nil {
		return fmt.Errorf("credential: keychain save %q: %w", creds.SSID, err)
	}
	return nil
}

func (k *KeychainStore) Delete(ssid string) error {
	if err := keyring.Delete(ServiceName, ssid); err != nil {
		if err == keyring.ErrNotFound {
			return ErrNotFound
		}
		return fmt.Errorf("credential: keychain delete %q: %w", ssid, err)
	}
	return nil
}

func (k *KeychainStore) List() ([]string, error) {
	// go-keyring doesn't support listing all entries for a service.
	// We maintain a separate index in the config file.
	// This is handled at the application level.
	return nil, fmt.Errorf("credential: keychain list not supported; use config index")
}
```

- [ ] **Step 3: Create file-based store (encrypted JSON)**

Create `internal/credential/file.go`:

```go
package credential

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileStore stores credentials in a JSON file on disk.
// This is the fallback when OS keychain is unavailable.
type FileStore struct {
	path string
	mu   sync.Mutex
}

// NewFileStore returns a Store backed by a JSON file.
func NewFileStore(configPath string) Store {
	return &FileStore{path: configPath}
}

type fileData struct {
	Credentials map[string]*Credentials `json:"credentials"`
}

func (f *FileStore) Load(ssid string) (*Credentials, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := f.readFile()
	if err != nil {
		return nil, err
	}
	creds, ok := data.Credentials[ssid]
	if !ok {
		return nil, ErrNotFound
	}
	creds.SSID = ssid
	return creds, nil
}

func (f *FileStore) Save(creds *Credentials) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := f.readFile()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if data.Credentials == nil {
		data.Credentials = make(map[string]*Credentials)
	}
	creds.UpdatedAt = time.Now()
	data.Credentials[creds.SSID] = creds

	return f.writeFile(data)
}

func (f *FileStore) Delete(ssid string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := f.readFile()
	if err != nil {
		return err
	}
	if _, ok := data.Credentials[ssid]; !ok {
		return ErrNotFound
	}
	delete(data.Credentials, ssid)
	return f.writeFile(data)
}

func (f *FileStore) List() ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := f.readFile()
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	ssids := make([]string, 0, len(data.Credentials))
	for ssid := range data.Credentials {
		ssids = append(ssids, ssid)
	}
	return ssids, nil
}

func (f *FileStore) readFile() (fileData, error) {
	var data fileData
	b, err := os.ReadFile(f.path)
	if err != nil {
		return data, err
	}
	if err := json.Unmarshal(b, &data); err != nil {
		return data, fmt.Errorf("credential: parse %s: %w", f.path, err)
	}
	return data, nil
}

func (f *FileStore) writeFile(data fileData) error {
	dir := filepath.Dir(f.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("credential: create dir: %w", err)
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("credential: marshal: %w", err)
	}
	if err := os.WriteFile(f.path, b, 0600); err != nil {
		return fmt.Errorf("credential: write %s: %w", f.path, err)
	}
	return nil
}
```

- [ ] **Step 4: Create v1 migration**

Create `internal/credential/migrate.go`:

```go
package credential

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"
)

// v1Entry represents the old plaintext config format from CapAuto v1.
type v1Entry struct {
	LoginURL      string            `json:"loginUrl"`
	Username      string            `json:"username"`
	Password      string            `json:"password"`
	UsernameField string            `json:"usernameField"`
	PasswordField string            `json:"passwordField"`
	StaticFields  map[string]string `json:"staticFields"`
	Action        string            `json:"action"`
	Method        string            `json:"method"`
}

// MigrateV1Config reads the old v1 plaintext config and migrates credentials
// to the given Store. It strips passwords from the JSON file after migration.
func MigrateV1Config(configPath string, store Store, logger *slog.Logger) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // nothing to migrate
		}
		return fmt.Errorf("credential: read v1 config: %w", err)
	}

	var config map[string]v1Entry
	if err := json.Unmarshal(data, &config); err != nil {
		return nil // not a v1 config format, skip
	}

	migrated := 0
	for ssid, entry := range config {
		// Detect v1 format: has username/password directly in entry
		if entry.Username == "" && entry.Password == "" {
			continue
		}

		creds := &Credentials{
			SSID:          ssid,
			Username:      entry.Username,
			Password:      entry.Password,
			UsernameField: entry.UsernameField,
			PasswordField: entry.PasswordField,
			FormAction:    entry.Action,
			FormMethod:    entry.Method,
			StaticFields:  entry.StaticFields,
			UpdatedAt:     time.Now(),
		}

		if err := store.Save(creds); err != nil {
			logger.Warn("failed to migrate credentials", "ssid", ssid, "error", err)
			continue
		}

		// Strip credentials from the v1 entry
		entry.Username = ""
		entry.Password = ""
		config[ssid] = entry
		migrated++

		logger.Info("migrated credentials to secure store", "ssid", ssid)
	}

	if migrated > 0 {
		// Write back the stripped config
		stripped, _ := json.MarshalIndent(config, "", "  ")
		if err := os.WriteFile(configPath, stripped, 0600); err != nil {
			logger.Warn("failed to strip v1 config", "error", err)
		}
	}

	return nil
}
```

- [ ] **Step 5: Create store tests**

Create `internal/credential/store_test.go`:

```go
package credential

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileStore_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "creds.json")
	store := NewFileStore(path)

	creds := &Credentials{
		SSID:          "Test_WiFi",
		Username:      "user1",
		Password:      "pass1",
		UsernameField: "user",
		PasswordField: "pass",
		FormAction:    "http://portal.example.com/auth",
		FormMethod:    "POST",
		StaticFields:  map[string]string{"csrf": "token"},
		UpdatedAt:     time.Now(),
	}

	if err := store.Save(creds); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := store.Load("Test_WiFi")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.Username != "user1" {
		t.Errorf("username: want 'user1', got %q", loaded.Username)
	}
	if loaded.Password != "pass1" {
		t.Errorf("password: want 'pass1', got %q", loaded.Password)
	}
	if loaded.StaticFields["csrf"] != "token" {
		t.Errorf("csrf: want 'token', got %q", loaded.StaticFields["csrf"])
	}
}

func TestFileStore_LoadNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "creds.json")
	store := NewFileStore(path)

	_, err := store.Load("NonExistent")
	if err == nil {
		t.Error("expected error for missing SSID")
	}
}

func TestFileStore_Delete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "creds.json")
	store := NewFileStore(path)

	creds := &Credentials{SSID: "ToDelete", Username: "u", Password: "p"}
	store.Save(creds)

	if err := store.Delete("ToDelete"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := store.Load("ToDelete")
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestFileStore_List(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "creds.json")
	store := NewFileStore(path)

	store.Save(&Credentials{SSID: "WiFi1", Username: "u", Password: "p"})
	store.Save(&Credentials{SSID: "WiFi2", Username: "u", Password: "p"})

	ssids, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(ssids) != 2 {
		t.Errorf("want 2 SSIDs, got %d", len(ssids))
	}
}

func TestFileStore_Permissions(t *testing.T) {
	if os.Getenv("GOOS") == "windows" {
		t.Skip("file permissions not applicable on Windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "creds.json")
	store := NewFileStore(path)

	store.Save(&Credentials{SSID: "Test", Username: "u", Password: "p"})

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	mode := info.Mode().Perm()
	if mode != 0600 {
		t.Errorf("file permissions: want 0600, got %04o", mode)
	}
}

func TestMigrateV1Config(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")

	// Write a v1-style config
	v1Config := map[string]interface{}{
		"Uni-WiFi": map[string]interface{}{
			"loginUrl":      "http://portal/login",
			"username":      "student",
			"password":      "secret",
			"usernameField": "user",
			"passwordField": "pass",
			"action":        "http://portal/auth",
			"method":        "POST",
		},
	}
	data, _ := json.Marshal(v1Config)
	os.WriteFile(configPath, data, 0600)

	store := NewFileStore(filepath.Join(dir, "migrated.json"))
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	err := MigrateV1Config(configPath, store, logger)
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	// Verify credentials were migrated
	creds, err := store.Load("Uni-WiFi")
	if err != nil {
		t.Fatalf("Load after migration failed: %v", err)
	}
	if creds.Username != "student" {
		t.Errorf("username: want 'student', got %q", creds.Username)
	}

	// Verify v1 config was stripped
	stripped, _ := os.ReadFile(configPath)
	var strippedConfig map[string]v1Entry
	json.Unmarshal(stripped, &strippedConfig)
	if strippedConfig["Uni-WiFi"].Username != "" {
		t.Error("v1 config should have username stripped")
	}
	if strippedConfig["Uni-WiFi"].Password != "" {
		t.Error("v1 config should have password stripped")
	}
}

func TestMigrateV1Config_NoFile(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "x.json"))
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	err := MigrateV1Config("/nonexistent/config.json", store, logger)
	if err != nil {
		t.Errorf("should not error on missing file, got: %v", err)
	}
}
```

- [ ] **Step 6: Run tests and commit**

```bash
go test -v -race ./internal/credential/
git add -A
git commit -m "feat: credential store with keychain, file fallback, and v1 migration"
```

---

### Task 6: Auth Submitter

**Files:**
- Create: `internal/auth/submitter.go`
- Create: `internal/auth/submitter_test.go`

**Interfaces:**
- Consumes:
  - `portal.FormData` from Task 4
  - `credential.Credentials` from Task 5
  - `prober.Prober` from Task 3
- Produces:
  - `func NewSubmitter(client *http.Client, logger *slog.Logger) *Submitter`
  - `func (s *Submitter) Submit(ctx context.Context, form *portal.FormData, creds *credential.Credentials) error`
  - `func (s *Submitter) Verify(ctx context.Context, prober *prober.Prober) (bool, error)`

- [ ] **Step 1: Create submitter implementation**

Create `internal/auth/submitter.go`:

```go
package auth

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sharzilnafis/capauto/internal/credential"
	"github.com/sharzilnafis/capauto/internal/portal"
	"github.com/sharzilnafis/capauto/internal/prober"
)

// Submitter handles login form submission and verification.
type Submitter struct {
	client *http.Client
	logger *slog.Logger
}

// NewSubmitter creates a Submitter using the shared HTTP client.
func NewSubmitter(client *http.Client, logger *slog.Logger) *Submitter {
	return &Submitter{client: client, logger: logger}
}

// Submit sends the login form with credentials.
func (s *Submitter) Submit(ctx context.Context, form *portal.FormData, creds *credential.Credentials) error {
	// Build payload: static fields + credentials
	payload := url.Values{}
	for k, v := range form.Fields {
		payload.Set(k, v)
	}
	if creds.UsernameField != "" {
		payload.Set(creds.UsernameField, creds.Username)
	} else if form.UsernameField != "" {
		payload.Set(form.UsernameField, creds.Username)
	}
	if creds.PasswordField != "" {
		payload.Set(creds.PasswordField, creds.Password)
	} else if form.PasswordField != "" {
		payload.Set(form.PasswordField, creds.Password)
	}

	var req *http.Request
	var err error

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	method := form.Method
	if method == "" {
		method = "POST"
	}

	if method == "GET" {
		u, err := url.Parse(form.Action)
		if err != nil {
			return fmt.Errorf("auth: parse action URL: %w", err)
		}
		// Merge payload into query string
		q := u.Query()
		for k, vs := range payload {
			for _, v := range vs {
				q.Set(k, v)
			}
		}
		u.RawQuery = q.Encode()
		req, err = http.NewRequestWithContext(ctx, "GET", u.String(), nil)
		if err != nil {
			return fmt.Errorf("auth: create GET request: %w", err)
		}
	} else {
		body := payload.Encode()
		req, err = http.NewRequestWithContext(ctx, "POST", form.Action, strings.NewReader(body))
		if err != nil {
			return fmt.Errorf("auth: create POST request: %w", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	s.logger.Info("submitting login", "action", form.Action, "method", method)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("auth: submit: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	s.logger.Info("login submitted", "status", resp.StatusCode)
	return nil
}

// Verify re-probes to confirm internet access after login.
func (s *Submitter) Verify(ctx context.Context, p *prober.Prober) (bool, error) {
	// Give the portal a moment to register the session
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case <-time.After(2500 * time.Millisecond):
	}

	result := p.Check(ctx)
	return result.Online, nil
}
```

- [ ] **Step 2: Create submitter tests**

Create `internal/auth/submitter_test.go`:

```go
package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/sharzilnafis/capauto/internal/credential"
	"github.com/sharzilnafis/capauto/internal/portal"
	"github.com/sharzilnafis/capauto/internal/prober"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestSubmitter_POST(t *testing.T) {
	var receivedUser, receivedPass, receivedCSRF string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/auth" {
			r.ParseForm()
			receivedUser = r.FormValue("user")
			receivedPass = r.FormValue("pass")
			receivedCSRF = r.FormValue("csrf")
			w.WriteHeader(200)
			fmt.Fprint(w, "OK")
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	sub := NewSubmitter(client, testLogger())

	form := &portal.FormData{
		Action:        srv.URL + "/auth",
		Method:        "POST",
		Fields:        map[string]string{"csrf": "token123"},
		UsernameField: "user",
		PasswordField: "pass",
	}
	creds := &credential.Credentials{
		Username:      "student",
		Password:      "secret",
		UsernameField: "user",
		PasswordField: "pass",
	}

	err := sub.Submit(context.Background(), form, creds)
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}
	if receivedUser != "student" {
		t.Errorf("user: want 'student', got %q", receivedUser)
	}
	if receivedPass != "secret" {
		t.Errorf("pass: want 'secret', got %q", receivedPass)
	}
	if receivedCSRF != "token123" {
		t.Errorf("csrf: want 'token123', got %q", receivedCSRF)
	}
}

func TestSubmitter_GET(t *testing.T) {
	var receivedUser, receivedPass string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/auth" {
			receivedUser = r.URL.Query().Get("user")
			receivedPass = r.URL.Query().Get("pass")
			w.WriteHeader(200)
			fmt.Fprint(w, "OK")
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	sub := NewSubmitter(client, testLogger())

	form := &portal.FormData{
		Action:        srv.URL + "/auth",
		Method:        "GET",
		Fields:        map[string]string{},
		UsernameField: "user",
		PasswordField: "pass",
	}
	creds := &credential.Credentials{
		Username:      "student",
		Password:      "secret",
		UsernameField: "user",
		PasswordField: "pass",
	}

	err := sub.Submit(context.Background(), form, creds)
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}
	if receivedUser != "student" {
		t.Errorf("user: want 'student', got %q", receivedUser)
	}
	if receivedPass != "secret" {
		t.Errorf("pass: want 'secret', got %q", receivedPass)
	}
}

func TestSubmitter_Verify(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, "<HTML><HEAD><TITLE>Success</TITLE></HEAD><BODY>Success</BODY></HTML>")
	}))
	defer srv.Close()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	sub := NewSubmitter(client, testLogger())

	p := prober.NewProber(client, testLogger())
	p.SetEndpoints([]prober.Endpoint{
		{URL: srv.URL, CheckFunc: func(status int, body string, redirected bool) bool {
			return status == 200 && !redirected
		}},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	online, err := sub.Verify(ctx, p)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !online {
		t.Error("expected online=true after verify")
	}
}
```

Note: The Verify test requires a `SetEndpoints` method on Prober. This must be added to `internal/prober/prober.go`:

```go
// SetEndpoints replaces the probe endpoints (useful for testing).
func (p *Prober) SetEndpoints(eps []Endpoint) {
	p.endpoints = eps
}
```

Also add the missing `time` import to the test file.

- [ ] **Step 3: Add SetEndpoints to prober, run tests, and commit**

Add to `internal/prober/prober.go`:
```go
// SetEndpoints replaces the probe endpoints (useful for testing).
func (p *Prober) SetEndpoints(eps []Endpoint) {
	p.endpoints = eps
}
```

```bash
go test -v -race ./internal/auth/
git add -A
git commit -m "feat: auth submitter with cookie-aware form submission and verification"
```

---

### Task 7: CLI Orchestrator & Main Entry Point

**Files:**
- Modify: `cmd/capauto/main.go` (replace placeholder with full CLI)

**Interfaces:**
- Consumes: all internal packages (network, prober, portal, auth, credential, log)
- Produces: the `capauto` binary with subcommands

- [ ] **Step 1: Implement full CLI with orchestrator**

Replace `cmd/capauto/main.go` with the full implementation:

```go
package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"strings"
	"time"

	capautoLog "github.com/sharzilnafis/capauto/internal/log"
	"github.com/sharzilnafis/capauto/internal/auth"
	"github.com/sharzilnafis/capauto/internal/credential"
	"github.com/sharzilnafis/capauto/internal/network"
	"github.com/sharzilnafis/capauto/internal/portal"
	"github.com/sharzilnafis/capauto/internal/prober"
)

const version = "2.0.0"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "capauto: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "creds":
			return handleCreds(args[1:])
		case "install":
			return handleInstall()
		case "uninstall":
			return handleUninstall()
		case "status":
			return handleStatus()
		case "migrate":
			return handleMigrate()
		case "version":
			fmt.Printf("capauto v%s\n", version)
			return nil
		}
	}

	// Default: run the automator
	return handleRun(args)
}

func handleRun(args []string) error {
	fs := flag.NewFlagSet("capauto", flag.ExitOnError)
	debug := fs.Bool("debug", false, "Enable debug logging")
	dryRun := fs.Bool("dry-run", false, "Detect portal but don't submit")
	insecure := fs.Bool("insecure-store", false, "Use plaintext credential storage")
	fs.Parse(args)

	// Setup logger
	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}
	logger := capautoLog.New(level, "text", os.Stderr)

	// Detect SSID
	ssid := os.Getenv("CAPAUTO_TEST_SSID")
	if ssid == "" {
		var err error
		ssid, err = network.GetSSID()
		if err != nil {
			logger.Warn("could not detect SSID", "error", err)
			ssid = "Unknown_WiFi"
		}
	}
	logger.Info("checking network", "ssid", ssid, "time", time.Now().Format(time.RFC3339))

	// Create shared HTTP client with cookie jar
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // captive portals use self-signed certs
			},
		},
	}

	ctx := context.Background()

	// Probe connectivity
	p := prober.NewProber(client, logger)
	result := p.Check(ctx)

	if result.Online {
		logger.Info("already online")
		return nil
	}
	if result.PortalURL == "" {
		logger.Info("not online, no captive portal detected")
		return nil
	}

	logger.Info("captive portal detected", "url", result.PortalURL)

	// Fetch the actual portal login page
	pageReq, err := http.NewRequestWithContext(ctx, "GET", result.PortalURL, nil)
	if err != nil {
		return fmt.Errorf("create portal request: %w", err)
	}
	pageResp, err := client.Do(pageReq)
	if err != nil {
		return fmt.Errorf("fetch portal page: %w", err)
	}
	defer pageResp.Body.Close()
	finalLoginURL := pageResp.Request.URL.String()

	if *debug {
		logger.Debug("portal page fetched", "finalURL", finalLoginURL, "status", pageResp.StatusCode)
	}

	// Parse the login form
	formData, err := portal.ParseLoginForm(pageResp.Body, finalLoginURL)
	if err != nil {
		return fmt.Errorf("parse login form: %w", err)
	}

	logger.Info("form parsed",
		"action", formData.Action,
		"method", formData.Method,
		"username_field", formData.UsernameField,
		"password_field", formData.PasswordField,
	)

	if *dryRun {
		logger.Info("dry run — not submitting")
		return nil
	}

	// Load or prompt for credentials
	store := getStore(*insecure, logger)

	// Try to migrate v1 config
	v1Path := filepath.Join(configDir(), "config.json")
	credential.MigrateV1Config(v1Path, store, logger)

	creds, err := store.Load(ssid)
	if err != nil {
		logger.Info("no saved credentials, prompting", "ssid", ssid)
		creds, err = promptForCredentials(ssid, formData)
		if err != nil {
			return fmt.Errorf("prompt credentials: %w", err)
		}
	} else {
		logger.Info("using saved credentials", "ssid", ssid)
	}

	// Update credential field info from current form parse
	creds.UsernameField = formData.UsernameField
	creds.PasswordField = formData.PasswordField
	creds.FormAction = formData.Action
	creds.FormMethod = formData.Method
	creds.StaticFields = formData.Fields

	// Save/update credentials
	if err := store.Save(creds); err != nil {
		logger.Warn("failed to save credentials", "error", err)
	}

	// Submit login
	sub := auth.NewSubmitter(client, logger)
	if err := sub.Submit(ctx, formData, creds); err != nil {
		return fmt.Errorf("submit login: %w", err)
	}

	// Verify
	online, err := sub.Verify(ctx, p)
	if err != nil {
		return fmt.Errorf("verify connectivity: %w", err)
	}
	if online {
		logger.Info("success — internet connection established")
		return nil
	}

	return fmt.Errorf("still offline after login — portal may need extra steps (use --debug to inspect)")
}

func handleCreds(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: capauto creds <list|add|remove> [SSID]")
		return nil
	}

	logger := capautoLog.New(slog.LevelInfo, "text", os.Stderr)
	store := getStore(false, logger)

	switch args[0] {
	case "list":
		ssids, err := store.List()
		if err != nil {
			// Fallback: try file store for listing
			fileStore := credential.NewFileStore(filepath.Join(configDir(), "credentials.json"))
			ssids, err = fileStore.List()
			if err != nil {
				return fmt.Errorf("list credentials: %w", err)
			}
		}
		if len(ssids) == 0 {
			fmt.Println("No saved credentials.")
			return nil
		}
		fmt.Println("Saved credentials:")
		for _, ssid := range ssids {
			fmt.Printf("  • %s\n", ssid)
		}

	case "add":
		if len(args) < 2 {
			return fmt.Errorf("usage: capauto creds add <SSID>")
		}
		ssid := args[1]
		creds, err := promptForCredentials(ssid, nil)
		if err != nil {
			return err
		}
		if err := store.Save(creds); err != nil {
			return fmt.Errorf("save credentials: %w", err)
		}
		fmt.Printf("Credentials saved for %q\n", ssid)

	case "remove":
		if len(args) < 2 {
			return fmt.Errorf("usage: capauto creds remove <SSID>")
		}
		if err := store.Delete(args[1]); err != nil {
			return fmt.Errorf("delete credentials: %w", err)
		}
		fmt.Printf("Credentials removed for %q\n", args[1])

	default:
		return fmt.Errorf("unknown creds command: %s", args[0])
	}
	return nil
}

func handleInstall() error {
	fmt.Println("capauto install — not yet implemented for this platform")
	fmt.Println("See README.md for manual installation instructions.")
	return nil
}

func handleUninstall() error {
	fmt.Println("capauto uninstall — not yet implemented for this platform")
	return nil
}

func handleStatus() error {
	fmt.Printf("capauto v%s\n", version)
	ssid, err := network.GetSSID()
	if err != nil {
		fmt.Printf("Wi-Fi: not connected (%v)\n", err)
	} else {
		fmt.Printf("Wi-Fi: %s\n", ssid)
	}
	return nil
}

func handleMigrate() error {
	logger := capautoLog.New(slog.LevelInfo, "text", os.Stderr)
	store := getStore(false, logger)
	v1Path := filepath.Join(configDir(), "config.json")
	return credential.MigrateV1Config(v1Path, store, logger)
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".capauto")
}

func getStore(insecure bool, logger *slog.Logger) credential.Store {
	if insecure {
		logger.Warn("using insecure plaintext credential storage")
		return credential.NewFileStore(filepath.Join(configDir(), "credentials.json"))
	}

	// Try keychain first
	store := credential.NewKeychainStore()
	// Test if keychain is available by attempting a load of a dummy key
	_, err := store.Load("__capauto_keychain_test__")
	if err != nil && err != credential.ErrNotFound {
		// Keychain unavailable, fall back to file store
		logger.Info("keychain unavailable, using file store", "error", err)
		return credential.NewFileStore(filepath.Join(configDir(), "credentials.json"))
	}
	return store
}

func promptForCredentials(ssid string, form *portal.FormData) (*credential.Credentials, error) {
	if !isTerminal() {
		return nil, fmt.Errorf("non-interactive terminal — cannot prompt for credentials (use 'capauto creds add %s')", ssid)
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("No saved credentials for SSID: %q. First-time setup…\n", ssid)
	fmt.Print("Enter your Wi-Fi username/ID: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Print("Enter your Wi-Fi password: ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	if username == "" || password == "" {
		return nil, fmt.Errorf("credentials cannot be empty")
	}

	creds := &credential.Credentials{
		SSID:      ssid,
		Username:  username,
		Password:  password,
		UpdatedAt: time.Now(),
	}

	if form != nil {
		creds.UsernameField = form.UsernameField
		creds.PasswordField = form.PasswordField
		creds.FormAction = form.Action
		creds.FormMethod = form.Method
		creds.StaticFields = form.Fields
	}

	return creds, nil
}

func isTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
```

- [ ] **Step 2: Build and verify**

```bash
go build -o capauto ./cmd/capauto
./capauto version
./capauto status
```

Expected: `capauto v2.0.0` and current WiFi status.

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "feat: CLI orchestrator with subcommands (run, creds, status, migrate)"
```

---

### Task 8: Integration Tests, Install Scripts & README

**Files:**
- Create: `integration_test.go` (top-level integration test)
- Create: `install/com.sharzil.capauto.plist`
- Create: `install/capauto.service`
- Create: `install/install_macos.sh`
- Create: `install/install_linux.sh`
- Modify: `README.md` (complete rewrite for v2)

**Interfaces:**
- Consumes: all internal packages
- Produces: integration test suite, install scripts, documentation

- [ ] **Step 1: Create integration test with mock portal**

Create `integration_test.go` at the project root:

```go
package integration_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/sharzilnafis/capauto/internal/auth"
	"github.com/sharzilnafis/capauto/internal/credential"
	"github.com/sharzilnafis/capauto/internal/portal"
	"github.com/sharzilnafis/capauto/internal/prober"
)

// mockPortal simulates a captive portal for integration testing.
type mockPortal struct {
	isOnline    bool
	validUser   string
	validPass   string
	lastRequest *http.Request
}

func (m *mockPortal) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.lastRequest = r

	switch r.URL.Path {
	case "/hotspot-detect.html":
		if m.isOnline {
			w.WriteHeader(200)
			fmt.Fprint(w, "<HTML><HEAD><TITLE>Success</TITLE></HEAD><BODY>Success</BODY></HTML>")
		} else {
			http.Redirect(w, r, "/login", http.StatusFound)
		}

	case "/login":
		w.WriteHeader(200)
		fmt.Fprint(w, `<html><body>
			<form action="/auth" method="POST">
				<input type="hidden" name="csrf" value="mock-csrf-token">
				<input type="hidden" name="session" value="xyz987">
				<input type="text" name="username_field">
				<input type="password" name="password_field">
				<input type="submit" value="Login">
			</form>
		</body></html>`)

	case "/auth":
		if r.Method == "POST" {
			r.ParseForm()
			if r.FormValue("username_field") == m.validUser &&
				r.FormValue("password_field") == m.validPass &&
				r.FormValue("csrf") == "mock-csrf-token" {
				m.isOnline = true
				w.WriteHeader(200)
				fmt.Fprint(w, "<h1>Logged In!</h1>")
				return
			}
		} else if r.Method == "GET" {
			q := r.URL.Query()
			if q.Get("username_field") == m.validUser &&
				q.Get("password_field") == m.validPass &&
				q.Get("csrf") == "mock-csrf-token" {
				m.isOnline = true
				w.WriteHeader(200)
				fmt.Fprint(w, "<h1>Logged In!</h1>")
				return
			}
		}
		w.WriteHeader(401)
		fmt.Fprint(w, "<h1>Unauthorized</h1>")

	default:
		w.WriteHeader(404)
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestIntegration_FullFlow_POST(t *testing.T) {
	mock := &mockPortal{isOnline: false, validUser: "student123", validPass: "securepass"}
	srv := httptest.NewServer(mock)
	defer srv.Close()

	logger := testLogger()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	// 1. Probe: should detect portal
	p := prober.NewProber(client, logger)
	p.SetEndpoints([]prober.Endpoint{
		{
			URL: srv.URL + "/hotspot-detect.html",
			CheckFunc: func(status int, body string, redirected bool) bool {
				return !redirected && strings.Contains(body, "Success")
			},
		},
	})

	result := p.Check(context.Background())
	if result.Online {
		t.Fatal("expected offline, got online")
	}
	if !strings.Contains(result.PortalURL, "/login") {
		t.Fatalf("expected portal URL with /login, got %q", result.PortalURL)
	}

	// 2. Fetch and parse portal page
	resp, err := client.Get(result.PortalURL)
	if err != nil {
		t.Fatalf("fetch portal: %v", err)
	}
	formData, err := portal.ParseLoginForm(resp.Body, resp.Request.URL.String())
	resp.Body.Close()
	if err != nil {
		t.Fatalf("parse form: %v", err)
	}

	if formData.UsernameField != "username_field" {
		t.Errorf("username field: want 'username_field', got %q", formData.UsernameField)
	}
	if formData.PasswordField != "password_field" {
		t.Errorf("password field: want 'password_field', got %q", formData.PasswordField)
	}
	if formData.Fields["csrf"] != "mock-csrf-token" {
		t.Errorf("csrf: want 'mock-csrf-token', got %q", formData.Fields["csrf"])
	}

	// 3. Submit login
	creds := &credential.Credentials{
		Username:      "student123",
		Password:      "securepass",
		UsernameField: formData.UsernameField,
		PasswordField: formData.PasswordField,
	}

	sub := auth.NewSubmitter(client, logger)
	if err := sub.Submit(context.Background(), formData, creds); err != nil {
		t.Fatalf("submit: %v", err)
	}

	if !mock.isOnline {
		t.Fatal("portal should be online after successful login")
	}

	// 4. Verify
	result2 := p.Check(context.Background())
	if !result2.Online {
		t.Error("expected online after login")
	}
}

func TestIntegration_FullFlow_GET(t *testing.T) {
	mock := &mockPortal{isOnline: false, validUser: "student123", validPass: "securepass"}
	srv := httptest.NewServer(mock)
	defer srv.Close()

	logger := testLogger()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	// Submit via GET
	formData := &portal.FormData{
		Action:        srv.URL + "/auth",
		Method:        "GET",
		Fields:        map[string]string{"csrf": "mock-csrf-token", "session": "xyz987"},
		UsernameField: "username_field",
		PasswordField: "password_field",
	}
	creds := &credential.Credentials{
		Username:      "student123",
		Password:      "securepass",
		UsernameField: "username_field",
		PasswordField: "password_field",
	}

	sub := auth.NewSubmitter(client, logger)
	if err := sub.Submit(context.Background(), formData, creds); err != nil {
		t.Fatalf("submit GET: %v", err)
	}

	if !mock.isOnline {
		t.Fatal("portal should be online after GET login")
	}
}

func TestIntegration_AlreadyOnline(t *testing.T) {
	mock := &mockPortal{isOnline: true}
	srv := httptest.NewServer(mock)
	defer srv.Close()

	logger := testLogger()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	p := prober.NewProber(client, logger)
	p.SetEndpoints([]prober.Endpoint{
		{
			URL: srv.URL + "/hotspot-detect.html",
			CheckFunc: func(status int, body string, redirected bool) bool {
				return !redirected && strings.Contains(body, "Success")
			},
		},
	})

	result := p.Check(context.Background())
	if !result.Online {
		t.Error("expected online=true")
	}
}

func TestIntegration_CredentialStore_FileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := credential.NewFileStore(dir + "/creds.json")

	creds := &credential.Credentials{
		SSID:          "IntegrationTest_WiFi",
		Username:      "testuser",
		Password:      "testpass",
		UsernameField: "user",
		PasswordField: "pass",
		FormAction:    "http://portal/auth",
		FormMethod:    "POST",
		StaticFields:  map[string]string{"csrf": "abc"},
	}

	if err := store.Save(creds); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := store.Load("IntegrationTest_WiFi")
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if loaded.Username != "testuser" || loaded.Password != "testpass" {
		t.Error("credentials mismatch after roundtrip")
	}
}
```

Note: Add the missing `"net/http/httptest"` import to the test file.

- [ ] **Step 2: Create macOS launchd plist**

Create `install/com.sharzil.capauto.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.sharzil.capauto</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/capauto</string>
    </array>
    <key>WatchPaths</key>
    <array>
        <string>/var/run/resolv.conf</string>
        <string>/Library/Preferences/SystemConfiguration/com.apple.wifi.message-tracer.plist</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/capauto.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/capauto.err</string>
</dict>
</plist>
```

- [ ] **Step 3: Create Linux systemd unit**

Create `install/capauto.service`:

```ini
[Unit]
Description=CapAuto Captive Portal Automator
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/capauto
StandardOutput=append:%h/.capauto/capauto.log
StandardError=append:%h/.capauto/capauto.err

[Install]
WantedBy=default.target
```

- [ ] **Step 4: Create macOS install script**

Create `install/install_macos.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

BINARY="capauto"
INSTALL_DIR="/usr/local/bin"
PLIST_NAME="com.sharzil.capauto.plist"
PLIST_SRC="$(dirname "$0")/${PLIST_NAME}"
PLIST_DST="$HOME/Library/LaunchAgents/${PLIST_NAME}"

echo "Installing CapAuto..."

# Build if binary doesn't exist
if [ ! -f "${BINARY}" ]; then
    echo "Building..."
    go build -o "${BINARY}" ./cmd/capauto
fi

# Install binary
echo "Copying binary to ${INSTALL_DIR}..."
sudo cp "${BINARY}" "${INSTALL_DIR}/${BINARY}"
sudo chmod 755 "${INSTALL_DIR}/${BINARY}"

# Install LaunchAgent
echo "Installing LaunchAgent..."
mkdir -p "$HOME/Library/LaunchAgents"
cp "${PLIST_SRC}" "${PLIST_DST}"

# Unload if already loaded
launchctl bootout "gui/$(id -u)" "${PLIST_DST}" 2>/dev/null || true

# Load
launchctl bootstrap "gui/$(id -u)" "${PLIST_DST}"

echo "Done! CapAuto is now running in the background."
echo "Logs: /tmp/capauto.log"
echo "To uninstall: capauto uninstall"
```

- [ ] **Step 5: Create Linux install script**

Create `install/install_linux.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

BINARY="capauto"
INSTALL_DIR="/usr/local/bin"
SERVICE_DIR="$HOME/.config/systemd/user"
DISPATCHER_DIR="/etc/NetworkManager/dispatcher.d"

echo "Installing CapAuto..."

# Build if binary doesn't exist
if [ ! -f "${BINARY}" ]; then
    echo "Building..."
    go build -o "${BINARY}" ./cmd/capauto
fi

# Install binary
echo "Copying binary to ${INSTALL_DIR}..."
sudo cp "${BINARY}" "${INSTALL_DIR}/${BINARY}"
sudo chmod 755 "${INSTALL_DIR}/${BINARY}"

# Install systemd unit
echo "Installing systemd user service..."
mkdir -p "${SERVICE_DIR}"
cp "$(dirname "$0")/capauto.service" "${SERVICE_DIR}/capauto.service"
systemctl --user daemon-reload
systemctl --user enable capauto.service

# Install NetworkManager dispatcher (optional)
if [ -d "${DISPATCHER_DIR}" ]; then
    echo "Installing NetworkManager dispatcher..."
    sudo tee "${DISPATCHER_DIR}/99-capauto" > /dev/null << 'EOF'
#!/bin/bash
if [ "$2" = "up" ]; then
    /usr/local/bin/capauto &
fi
EOF
    sudo chmod 755 "${DISPATCHER_DIR}/99-capauto"
fi

echo "Done! CapAuto is now installed."
echo "To run manually: capauto"
echo "To uninstall: capauto uninstall"
```

- [ ] **Step 6: Update README.md**

Replace `README.md` with v2 documentation:

```markdown
# CapAuto — Captive Portal Automator

CapAuto is a fast, cross-platform, single-binary tool that automatically detects captive portal WiFi networks, parses the login form, and logs you in using saved credentials.

It runs silently in the background and handles university, hotel, cafe, and airport portals.

## Features

- **Single binary** — no runtime dependencies (Node.js, Python, etc.)
- **Cross-platform** — macOS, Windows, Linux
- **Secure credentials** — OS keychain (macOS Keychain, Windows Credential Manager, Linux Secret Service)
- **Smart form parsing** — proper HTML DOM parser, not regex
- **Auto-detect portals** — probes multiple connectivity endpoints in parallel
- **Background daemon** — installs as a system service triggered by network changes
- **Credential management** — CLI subcommands to list, add, and remove saved credentials

## Quick Start

### Install

```bash
# Build from source (requires Go 1.21+)
make build

# Install to /usr/local/bin and set up background service
make install
capauto install
```

### First Use

1. Connect to a WiFi network with a captive portal
2. Run `capauto` — it will detect the portal and prompt for your credentials
3. Credentials are saved securely; future connections are automatic

### Manual Usage

```bash
capauto                     # Run once: detect portal → login
capauto --debug             # Verbose output with portal HTML dump
capauto --dry-run           # Detect portal but don't submit

capauto creds list          # Show saved SSIDs
capauto creds add <SSID>    # Add credentials for a network
capauto creds remove <SSID> # Remove credentials

capauto install             # Install as background service
capauto uninstall           # Remove background service
capauto status              # Show daemon status
capauto version             # Show version
```

## Upgrading from v1 (Node.js)

If you previously used the Node.js version, CapAuto v2 automatically migrates your saved credentials from `~/.capauto/config.json` to the OS keychain on first run.

You can also migrate manually:

```bash
capauto migrate
```

After migration, you can safely remove the old Node.js files (`index.js`, `package.json`, `node_modules/`).

## Background Service Setup

### macOS

```bash
capauto install
# Or manually:
bash install/install_macos.sh
```

Uses `launchd` to trigger on network changes.

### Linux

```bash
capauto install
# Or manually:
bash install/install_linux.sh
```

Uses `systemd` user service + NetworkManager dispatcher.

### Windows

```bash
capauto install
```

Uses Task Scheduler triggered by network connection events (Event ID 10000).

## Building from Source

```bash
# Requirements: Go 1.21+

# Build for current platform
make build

# Run tests
make test

# Cross-compile for all platforms
make cross-compile
# Output: build/capauto-{darwin,linux,windows}-{amd64,arm64}
```

## Security

- Credentials stored in OS keychain by default
- Fallback: encrypted JSON file (`~/.capauto/credentials.json`)
- Config files created with owner-only permissions (0600)
- Passwords never appear in logs
- TLS verification disabled only for captive portal connections

## Architecture

```
cmd/capauto/main.go         → CLI entry point & orchestrator
internal/network/            → SSID detection (per-OS)
internal/prober/             → Connectivity probing (multi-endpoint)
internal/portal/             → DOM-based HTML form parser
internal/auth/               → Form submission & verification
internal/credential/         → Keychain + file credential store
internal/log/                → Structured logging with rotation
```

## License

MIT
```

- [ ] **Step 7: Run all tests and commit**

```bash
go test -v -race -cover ./...
git add -A
git commit -m "feat: integration tests, install scripts, and v2 README"
```

---
