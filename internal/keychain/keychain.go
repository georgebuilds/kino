// Package keychain wraps the OS keychain for storing sensitive credentials.
// Transaction data lives in the .kino file unencrypted; only sync tokens and
// bank credentials are stored here.
package keychain

import (
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

const service = "kino"

// Set stores a secret under the given key.
func Set(key, secret string) error {
	if key == "" {
		return fmt.Errorf("keychain: key must not be empty")
	}
	if err := keyring.Set(service, key, secret); err != nil {
		return fmt.Errorf("keychain set %q: %w", key, err)
	}
	return nil
}

// Get retrieves a secret. Returns ("", ErrNotFound) if the key doesn't exist.
func Get(key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("keychain: key must not be empty")
	}
	val, err := keyring.Get(service, key)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("keychain get %q: %w", key, err)
	}
	return val, nil
}

// Delete removes a secret. No-ops if the key doesn't exist.
func Delete(key string) error {
	if key == "" {
		return fmt.Errorf("keychain: key must not be empty")
	}
	err := keyring.Delete(service, key)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("keychain delete %q: %w", key, err)
	}
	return nil
}

var ErrNotFound = errors.New("keychain: secret not found")
