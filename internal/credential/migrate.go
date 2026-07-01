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
			return nil
		}
		return fmt.Errorf("credential: read v1 config: %w", err)
	}

	var config map[string]v1Entry
	if err := json.Unmarshal(data, &config); err != nil {
		return nil // not a v1 config format, skip
	}

	migrated := 0
	for ssid, entry := range config {
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

		entry.Username = ""
		entry.Password = ""
		config[ssid] = entry
		migrated++

		logger.Info("migrated credentials to secure store", "ssid", ssid)
	}

	if migrated > 0 {
		stripped, _ := json.MarshalIndent(config, "", "  ")
		if err := os.WriteFile(configPath, stripped, 0600); err != nil {
			logger.Warn("failed to strip v1 config", "error", err)
		}
	}

	return nil
}
