package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const credentialsFile = "credentials.json"

// ErrNoCredentials indicates nothing has been stored yet.
var ErrNoCredentials = errors.New("windgo: no stored credentials")

// Credentials is what we persist between sessions.
type Credentials struct {
	Token    string `json:"token"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Provider string `json:"provider"`
}

func configDir() (string, error) {
	if dir := os.Getenv("WINDGO_CONFIG_DIR"); dir != "" {
		return dir, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "windgo"), nil
}

// Save writes credentials to disk with 0600 permissions.
func Save(creds Credentials) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, credentialsFile+".tmp")
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, credentialsFile))
}

// Load retrieves credentials from disk.
func Load() (*Credentials, error) {
	dir, err := configDir()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, credentialsFile))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNoCredentials
		}
		return nil, err
	}
	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}
	if creds.Token == "" {
		return nil, errors.New("stored token missing")
	}
	return &creds, nil
}

// Clear removes stored credentials.
func Clear() error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, credentialsFile)
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
