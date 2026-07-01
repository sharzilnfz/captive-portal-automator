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
const ServiceName = "autocap"

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
