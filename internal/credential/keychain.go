package credential

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

// KeychainStore stores credentials in the OS keychain.
// An optional indexPath tracks stored SSIDs because go-keyring
// does not expose a "list all keys" API.
type KeychainStore struct {
	indexPath string
}

// NewKeychainStore returns a Store backed by the OS keychain with no SSID index.
// List() will return an error; use NewKeychainStoreWithIndex for listing support.
func NewKeychainStore() Store {
	return &KeychainStore{}
}

// NewKeychainStoreWithIndex returns a Store backed by the OS keychain.
// indexPath is a JSON file that tracks which SSIDs have been saved so
// that List() can enumerate them (go-keyring has no native listing API).
func NewKeychainStoreWithIndex(indexPath string) Store {
	return &KeychainStore{indexPath: indexPath}
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
	_ = keyring.Delete(ServiceName, creds.SSID)
	if err := keyring.Set(ServiceName, creds.SSID, data); err != nil {
		return fmt.Errorf("credential: keychain save %q: %w", creds.SSID, err)
	}
	if k.indexPath != "" {
		_ = k.addToIndex(creds.SSID)
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
	if k.indexPath != "" {
		_ = k.removeFromIndex(ssid)
	}
	return nil
}

// List returns all SSIDs tracked in the index file.
// Returns an error if no index path was configured.
func (k *KeychainStore) List() ([]string, error) {
	if k.indexPath == "" {
		return nil, fmt.Errorf("credential: keychain list not supported; use NewKeychainStoreWithIndex")
	}
	return k.readIndex()
}

// ── index helpers ──────────────────────────────────────────────────────────────

func (k *KeychainStore) readIndex() ([]string, error) {
	data, err := os.ReadFile(k.indexPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("credential: read keychain index: %w", err)
	}
	var ssids []string
	if err := json.Unmarshal(data, &ssids); err != nil {
		return nil, fmt.Errorf("credential: parse keychain index: %w", err)
	}
	return ssids, nil
}

func (k *KeychainStore) writeIndex(ssids []string) error {
	if err := os.MkdirAll(filepath.Dir(k.indexPath), 0700); err != nil {
		return fmt.Errorf("credential: create index dir: %w", err)
	}
	data, err := json.MarshalIndent(ssids, "", "  ")
	if err != nil {
		return fmt.Errorf("credential: marshal index: %w", err)
	}
	return os.WriteFile(k.indexPath, data, 0600)
}

func (k *KeychainStore) addToIndex(ssid string) error {
	ssids, _ := k.readIndex()
	for _, s := range ssids {
		if s == ssid {
			return nil // already present — no-op
		}
	}
	return k.writeIndex(append(ssids, ssid))
}

func (k *KeychainStore) removeFromIndex(ssid string) error {
	ssids, err := k.readIndex()
	if err != nil {
		return err
	}
	filtered := ssids[:0]
	for _, s := range ssids {
		if s != ssid {
			filtered = append(filtered, s)
		}
	}
	return k.writeIndex(filtered)
}
