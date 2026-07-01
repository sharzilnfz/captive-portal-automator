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
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
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
