package db

import (
	"database/sql"
	"fmt"
	"time"

	"kino/internal/models"
)

func (db *DB) ListAccounts() ([]models.Account, error) {
	return db.listAccounts(false)
}

func (db *DB) listAccounts(includeHidden bool) ([]models.Account, error) {
	q := `SELECT id, name, type, institution, balance_cents, currency,
	             is_hidden, sort_order, last_synced_at, created_at, updated_at
	      FROM accounts`
	if !includeHidden {
		q += ` WHERE is_hidden = 0`
	}
	q += ` ORDER BY sort_order, id`

	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Account
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (db *DB) GetAccount(id int64) (*models.Account, error) {
	row := db.QueryRow(`
		SELECT id, name, type, institution, balance_cents, currency,
		       is_hidden, sort_order, last_synced_at, created_at, updated_at
		FROM accounts WHERE id = ?
	`, id)

	a, err := scanAccount(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (db *DB) CreateAccount(a *models.Account) error {
	if a.Currency == "" {
		a.Currency = "USD"
	}
	res, err := db.Exec(`
		INSERT INTO accounts(name, type, institution, balance_cents, currency, is_hidden, sort_order)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, a.Name, a.Type, a.Institution, a.BalanceCents, a.Currency, a.IsHidden, a.SortOrder)
	if err != nil {
		return fmt.Errorf("create account: %w", err)
	}
	a.ID, _ = res.LastInsertId()
	a.CreatedAt = time.Now().UTC()
	a.UpdatedAt = a.CreatedAt
	return nil
}

func (db *DB) UpdateAccount(a *models.Account) error {
	_, err := db.Exec(`
		UPDATE accounts SET
			name = ?, type = ?, institution = ?, balance_cents = ?,
			currency = ?, is_hidden = ?, sort_order = ?,
			updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE id = ?
	`, a.Name, a.Type, a.Institution, a.BalanceCents,
		a.Currency, a.IsHidden, a.SortOrder, a.ID)
	return err
}

func (db *DB) DeleteAccount(id int64) error {
	_, err := db.Exec(`DELETE FROM accounts WHERE id = ?`, id)
	return err
}

// RecalcBalance recomputes and stores an account's balance from its transactions.
func (db *DB) RecalcBalance(accountID int64) error {
	_, err := db.Exec(`
		UPDATE accounts SET
			balance_cents = (
				SELECT COALESCE(SUM(amount_cents), 0) FROM transactions WHERE account_id = ?
			),
			updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE id = ?
	`, accountID, accountID)
	return err
}

func scanAccount(s scanner) (models.Account, error) {
	var a models.Account
	var lastSynced sql.NullString
	var createdAt, updatedAt string

	err := s.Scan(
		&a.ID, &a.Name, &a.Type, &a.Institution, &a.BalanceCents,
		&a.Currency, &a.IsHidden, &a.SortOrder, &lastSynced,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return a, err
	}

	if lastSynced.Valid {
		t := parseTime(lastSynced.String)
		a.LastSyncedAt = &t
	}
	a.CreatedAt = parseTime(createdAt)
	a.UpdatedAt = parseTime(updatedAt)
	return a, nil
}
