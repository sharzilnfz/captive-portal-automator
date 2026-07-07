package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	autocapLog "github.com/sharzilnafis/autocap/internal/log"
	"github.com/sharzilnafis/autocap/internal/auth"
	"github.com/sharzilnafis/autocap/internal/credential"
	"github.com/sharzilnafis/autocap/internal/network"
	"github.com/sharzilnafis/autocap/internal/portal"
	"github.com/sharzilnafis/autocap/internal/prober"
)

const version = "2.0.0"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "autocap: %v\n", err)
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
			fmt.Printf("autocap v%s\n", version)
			return nil
		case "help", "--help", "-h":
			printUsage()
			return nil
		}
	}

	return handleAutomate(args)
}

func handleAutomate(args []string) error {
	fs := flag.NewFlagSet("autocap", flag.ExitOnError)
	debug := fs.Bool("debug", false, "Enable debug logging")
	dryRun := fs.Bool("dry-run", false, "Detect portal but don't submit")
	insecure := fs.Bool("insecure-store", false, "Use plaintext credential storage")
	fs.Parse(args)

	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}
	logger := autocapLog.New(level, "text", os.Stderr)

	// Detect SSID
	ssid := os.Getenv("AUTOCAP_TEST_SSID")
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

	// Buffer body so it can be both logged and parsed.
	bodyBytes, err := io.ReadAll(pageResp.Body)
	if err != nil {
		return fmt.Errorf("read portal body: %w", err)
	}

	if *debug {
		logger.Debug("portal page fetched", "finalURL", finalLoginURL, "status", pageResp.StatusCode)
		snip := string(bodyBytes)
		if len(snip) > 2000 {
			snip = snip[:2000]
		}
		logger.Debug("portal HTML", "body", snip)
	}

	// Parse the login form, following one level of redirect if needed.
	var formData *portal.FormData
	formData, _, err = parseFormWithFollowRedirect(ctx, client, bodyBytes, finalLoginURL, logger, *debug)
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

	// Load or prompt for credentials (skip for click-through portals)
	var creds *credential.Credentials
	isClickThrough := formData.UsernameField == "" && formData.PasswordField == ""

	if isClickThrough {
		logger.Info("click-through portal detected (no credentials required)")
		creds = &credential.Credentials{
			SSID: ssid,
		}
	} else {
		store := getStore(*insecure, logger)

		// Try to migrate v1 config
		v1Path := filepath.Join(configDir(), "config.json")
		credential.MigrateV1Config(v1Path, store, logger)

		var err error
		creds, err = store.Load(ssid)
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

		if err := store.Save(creds); err != nil {
			logger.Warn("failed to save credentials", "error", err)
		}
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

// parseFormWithFollowRedirect parses a login form from bodyBytes with baseURL.
// Resolution order:
//  1. Standard ParseLoginForm (static <form> element).
//  2. If ErrRedirect, fetch that URL and retry once.
//  3. If the form was synthesised from id-based inputs (JS-driven portal),
//     try to resolve the real action URL from external JS scripts;
//     for Ruijie/H3C ePortal portals fall back to the canonical endpoint.
func parseFormWithFollowRedirect(
	ctx context.Context,
	client *http.Client,
	bodyBytes []byte,
	baseURL string,
	logger *slog.Logger,
	debug bool,
) (*portal.FormData, string, error) {
	fd, finalURL, err := tryParseForm(ctx, client, bodyBytes, baseURL, logger, debug)
	if err != nil {
		// If no form was found, check if this is a Ruijie ePortal with
		// JS-rendered login — synthesize the form from known Ruijie fields.
		if errors.Is(err, portal.ErrNoForm) {
			if ruijieFD := tryRuijieSynthesis(bodyBytes, baseURL, logger); ruijieFD != nil {
				return ruijieFD, baseURL, nil
			}
		}
		return nil, baseURL, err
	}

	// If the action is still the bare portal page URL the form was synthesised
	// from id-based inputs — we need to find the real AJAX submission endpoint.
	if fd.Action == baseURL || fd.Action == finalURL {
		if resolved := resolveFormAction(ctx, client, bodyBytes, baseURL, logger, debug); resolved != "" {
			logger.Info("resolved JS login action", "url", resolved)
			fd.Action = resolved
		}
	}

	// Ruijie ePortal (classic) requires a queryString parameter whose value
	// is the URL-encoded query string from the original portal page URL.
	if strings.Contains(fd.Action, "InterFace.do") {
		if qs := portalQueryString(baseURL); qs != "" {
			fd.Fields["queryString"] = qs
			fd.Fields["operatorPwd"] = ""
			fd.Fields["operatorUserId"] = ""
			fd.Fields["validcode"] = ""
			fd.Fields["passwordEncrypt"] = "false"
			fd.Fields["service"] = ""
			logger.Info("Ruijie ePortal detected — added queryString", "queryString", qs)
		}
	}

	// Ruijie ePortal (newer / 2nd-gen) uses /portalauth/syncPortalResult
	// and expects: userName (not username), password, plus each query
	// parameter from the portal page URL as an individual POST field.
	if strings.Contains(fd.Action, "portalauth") {
		// Remap 'username' → 'userName' (Ruijie convention).
		if fd.UsernameField == "username" {
			fd.UsernameField = "userName"
			delete(fd.Fields, "username")
			fd.Fields["userName"] = ""
		}

		// Carry portal page URL query params into POST fields.
		if parsed, parseErr := url.Parse(finalURL); parseErr == nil {
			for k, vs := range parsed.Query() {
				if _, exists := fd.Fields[k]; !exists && len(vs) > 0 {
					fd.Fields[k] = vs[0]
				}
			}
		}
		logger.Info("Ruijie portalauth detected — added page params",
			"action", fd.Action,
			"usernameField", fd.UsernameField,
		)
	}

	return fd, finalURL, nil
}

// tryParseForm performs the core form-parse + one-level redirect-follow.
func tryParseForm(
	ctx context.Context,
	client *http.Client,
	bodyBytes []byte,
	baseURL string,
	logger *slog.Logger,
	debug bool,
) (*portal.FormData, string, error) {
	fd, err := portal.ParseLoginForm(bytes.NewReader(bodyBytes), baseURL)
	if err == nil {
		return fd, baseURL, nil
	}

	var redir *portal.ErrRedirect
	if !errors.As(err, &redir) {
		return nil, baseURL, err
	}

	logger.Info("portal redirects to iframe/frame/JS target, following", "url", redir.URL)

	req, fetchErr := http.NewRequestWithContext(ctx, "GET", redir.URL, nil)
	if fetchErr != nil {
		return nil, redir.URL, fmt.Errorf("create redirect request: %w", fetchErr)
	}
	resp, fetchErr := client.Do(req)
	if fetchErr != nil {
		return nil, redir.URL, fmt.Errorf("fetch redirect page: %w", fetchErr)
	}
	defer resp.Body.Close()

	redirectedURL := resp.Request.URL.String()
	redirectBody, fetchErr := io.ReadAll(resp.Body)
	if fetchErr != nil {
		return nil, redirectedURL, fmt.Errorf("read redirect body: %w", fetchErr)
	}

	if debug {
		snip := string(redirectBody)
		if len(snip) > 2000 {
			snip = snip[:2000]
		}
		logger.Debug("redirect page HTML", "url", redirectedURL, "body", snip)
	}

	fd, err = portal.ParseLoginForm(bytes.NewReader(redirectBody), redirectedURL)
	if err != nil {
		return nil, redirectedURL, err
	}
	return fd, redirectedURL, nil
}

// resolveFormAction fetches the portal page's external JS scripts and attempts
// to extract the login submission URL.  Falls back to the Ruijie canonical
// endpoint when the portal is identified as a Ruijie/H3C ePortal.
func resolveFormAction(
	ctx context.Context,
	client *http.Client,
	pageBody []byte,
	baseURL string,
	logger *slog.Logger,
	debug bool,
) string {
	base, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}

	// Parse the document again to extract script URLs and detect Ruijie.
	doc, parseErr := portal.ParseDoc(bytes.NewReader(pageBody))
	if parseErr != nil {
		return ""
	}

	scriptURLs := portal.ExtractScriptURLs(doc, base)

	// Scan each linked JS file for a login action URL.
	for _, scriptURL := range scriptURLs {
		sreq, reqErr := http.NewRequestWithContext(ctx, "GET", scriptURL, nil)
		if reqErr != nil {
			continue
		}
		sresp, respErr := client.Do(sreq)
		if respErr != nil {
			continue
		}
		scriptBytes, readErr := io.ReadAll(io.LimitReader(sresp.Body, 512*1024))
		sresp.Body.Close()
		if readErr != nil {
			continue
		}
		script := string(scriptBytes)
		if debug {
			logger.Debug("inspecting JS for login action", "url", scriptURL, "size", len(script))
		}
		if action := portal.FindLoginActionInScript(script, baseURL); action != "" {
			return action
		}
	}

	// Ruijie fingerprint fallback — use canonical endpoint without needing JS.
	if portal.IsRuijiePortal(doc) {
		action := portal.RuijieLoginURL(baseURL)
		logger.Info("Ruijie ePortal fingerprinted — using canonical login URL", "action", action)
		return action
	}

	return ""
}

// tryRuijieSynthesis parses the HTML body, checks for Ruijie ePortal
// fingerprints, and returns a fully populated FormData if detected.
// Returns nil when the page is not a Ruijie portal.
func tryRuijieSynthesis(bodyBytes []byte, baseURL string, logger *slog.Logger) *portal.FormData {
	doc, err := portal.ParseDoc(bytes.NewReader(bodyBytes))
	if err != nil {
		return nil
	}

	fd := portal.SynthesizeRuijieForm(doc, baseURL)
	if fd == nil {
		return nil
	}

	// Populate queryString from the portal page URL.
	if qs := portalQueryString(baseURL); qs != "" {
		fd.Fields["queryString"] = qs
	}

	logger.Info("Ruijie ePortal detected — synthesized login form",
		"action", fd.Action,
		"queryString", fd.Fields["queryString"],
	)

	return fd
}

// portalQueryString returns the raw query string from portalURL, URL-encoded
// for use as the Ruijie ePortal queryString POST parameter.
func portalQueryString(portalURL string) string {
	u, err := url.Parse(portalURL)
	if err != nil || u.RawQuery == "" {
		return ""
	}
	return u.RawQuery
}


func handleCreds(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: autocap creds <list|add|remove|reset> [SSID]")
		return nil
	}

	logger := autocapLog.New(slog.LevelInfo, "text", os.Stderr)
	store := getStore(false, logger)

	switch args[0] {
	case "list":
		ssids, err := store.List()
		if err != nil {
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
			fmt.Printf("  • %q\n", ssid)
		}

	case "add":
		if len(args) < 2 {
			return fmt.Errorf("usage: autocap creds add <SSID>")
		}
		creds, err := promptForCredentials(args[1], nil)
		if err != nil {
			return err
		}
		if err := store.Save(creds); err != nil {
			return fmt.Errorf("save credentials: %w", err)
		}
		fmt.Printf("Credentials saved for %q\n", args[1])

	case "remove":
		if len(args) < 2 {
			return fmt.Errorf("usage: autocap creds remove <SSID>")
		}
		if err := store.Delete(args[1]); err != nil {
			return fmt.Errorf("delete credentials: %w", err)
		}
		fmt.Printf("Credentials removed for %q\n", args[1])

	case "reset":
		ssids, err := store.List()
		if err != nil {
			// Try file store as well
			fileStore := credential.NewFileStore(filepath.Join(configDir(), "credentials.json"))
			ssids, _ = fileStore.List()
		}
		if len(ssids) == 0 {
			fmt.Println("No saved credentials to reset.")
			return nil
		}
		for _, ssid := range ssids {
			if delErr := store.Delete(ssid); delErr != nil && !errors.Is(delErr, credential.ErrNotFound) {
				logger.Warn("failed to delete credential", "ssid", ssid, "error", delErr)
			}
		}
		// Also nuke the index file directly.
		indexPath := filepath.Join(configDir(), "keyring_index.json")
		os.Remove(indexPath)
		// Also remove file-store credentials if present.
		os.Remove(filepath.Join(configDir(), "credentials.json"))
		fmt.Printf("All credentials removed (%d entries).\n", len(ssids))

	default:
		return fmt.Errorf("unknown creds command: %s", args[0])
	}
	return nil
}

func handleInstall() error {
	fmt.Println("autocap install — see README.md for platform-specific instructions")
	fmt.Println("  macOS:  bash install/install_macos.sh")
	fmt.Println("  Linux:  bash install/install_linux.sh")
	return nil
}

func handleUninstall() error {
	fmt.Println("autocap uninstall — see README.md for platform-specific instructions")
	return nil
}

func handleStatus() error {
	fmt.Printf("autocap v%s\n", version)
	ssid, err := network.GetSSID()
	if err != nil {
		fmt.Printf("Wi-Fi: not connected (%v)\n", err)
	} else {
		fmt.Printf("Wi-Fi: %s\n", ssid)
	}
	return nil
}

func handleMigrate() error {
	logger := autocapLog.New(slog.LevelInfo, "text", os.Stderr)
	store := getStore(false, logger)
	v1Path := filepath.Join(configDir(), "config.json")
	return credential.MigrateV1Config(v1Path, store, logger)
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".autocap")
}

func getStore(insecure bool, logger *slog.Logger) credential.Store {
	if insecure {
		logger.Warn("using insecure plaintext credential storage")
		return credential.NewFileStore(filepath.Join(configDir(), "credentials.json"))
	}

	indexPath := filepath.Join(configDir(), "keyring_index.json")
	store := credential.NewKeychainStoreWithIndex(indexPath)
	_, err := store.Load("__autocap_keychain_test__")
	if err != nil && err != credential.ErrNotFound {
		logger.Info("keychain unavailable, using file store", "error", err)
		return credential.NewFileStore(filepath.Join(configDir(), "credentials.json"))
	}
	return store
}

func promptForCredentials(ssid string, form *portal.FormData) (*credential.Credentials, error) {
	if !isTerminal() {
		return nil, fmt.Errorf("non-interactive terminal — cannot prompt (use 'autocap creds add %s')", ssid)
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("No saved credentials for SSID: %q. First-time setup…\n", ssid)

	var username, password string
	if form == nil || form.UsernameField != "" {
		promptText := "Enter your Wi-Fi username/ID: "
		if form != nil && form.UsernameField != "" {
			lower := strings.ToLower(form.UsernameField)
			if strings.Contains(lower, "email") {
				promptText = "Enter your Email: "
			} else if strings.Contains(lower, "phone") || strings.Contains(lower, "mobile") {
				promptText = "Enter your Phone number: "
			} else if strings.Contains(lower, "code") || strings.Contains(lower, "voucher") {
				promptText = "Enter your Voucher/Access Code: "
			}
		}
		fmt.Print(promptText)
		username, _ = reader.ReadString('\n')
		username = strings.TrimSpace(username)
	}

	if form == nil || form.PasswordField != "" {
		fmt.Print("Enter your Wi-Fi password: ")
		password, _ = reader.ReadString('\n')
		password = strings.TrimSpace(password)
	}

	// Validation: require at least one field if the form asks for it
	if (form == nil || form.UsernameField != "") && username == "" {
		return nil, fmt.Errorf("username/ID cannot be empty")
	}
	if (form == nil || form.PasswordField != "") && password == "" {
		return nil, fmt.Errorf("password cannot be empty")
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

func printUsage() {
	fmt.Println(`AutoCap v` + version + ` — Captive Portal Automator

Usage:
  autocap                     Run once: detect portal → login
  autocap --debug             Verbose output with portal HTML dump
  autocap --dry-run           Detect portal but don't submit
  autocap --insecure-store    Use plaintext credential storage

  autocap creds list          Show saved SSIDs
  autocap creds add <SSID>    Add credentials for a network
  autocap creds remove <SSID> Remove credentials

  autocap install             Install as background service
  autocap uninstall           Remove background service
  autocap status              Show Wi-Fi status
  autocap migrate             Migrate v1 plaintext config
  autocap version             Show version`)
}
