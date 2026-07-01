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
		fmt.Println("Usage: autocap creds <list|add|remove> [SSID]")
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
			fmt.Printf("  • %s\n", ssid)
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
