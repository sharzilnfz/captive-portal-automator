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

	store.Save(&Credentials{SSID: "ToDelete", Username: "u", Password: "p"})

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

	creds, err := store.Load("Uni-WiFi")
	if err != nil {
		t.Fatalf("Load after migration failed: %v", err)
	}
	if creds.Username != "student" {
		t.Errorf("username: want 'student', got %q", creds.Username)
	}

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
