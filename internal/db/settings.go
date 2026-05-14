package db

import (
	"database/sql"
	"fmt"
)

func (db *DB) GetSetting(key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("setting key must not be empty")
	}
	var val string
	err := db.QueryRow(`SELECT value FROM app_settings WHERE key = ?`, key).Scan(&val)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return val, err
}

func (db *DB) SetSetting(key, value string) error {
	if key == "" {
		return fmt.Errorf("setting key must not be empty")
	}
	_, err := db.Exec(`
		INSERT INTO app_settings(key, value) VALUES(?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, key, value)
	return err
}
