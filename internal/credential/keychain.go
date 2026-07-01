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
	return nil, fmt.Errorf("credential: keychain list not supported; use config index")
}
