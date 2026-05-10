package db

import "database/sql"

func (db *DB) GetSetting(key string) (string, error) {
	var val string
	err := db.QueryRow(`SELECT value FROM app_settings WHERE key = ?`, key).Scan(&val)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return val, err
}

func (db *DB) SetSetting(key, value string) error {
	_, err := db.Exec(`
		INSERT INTO app_settings(key, value) VALUES(?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, key, value)
	return err
}
