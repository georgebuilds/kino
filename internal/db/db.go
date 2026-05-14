// Package db manages the .kino SQLite file — open, create, migrate, close.
package db

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const kinoExtension = ".kino"

// scanner is satisfied by both *sql.Row and *sql.Rows, letting scan helpers
// work for both single-row and multi-row queries.
type scanner interface{ Scan(dest ...any) error }

// parseTime handles both RFC3339Nano (timestamps) and YYYY-MM-DD (dates).
func parseTime(s string) time.Time {
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t
	}
	return time.Time{}
}

// DB wraps *sql.DB and carries the file path for display in the UI.
type DB struct {
	*sql.DB
	Path string
}

// Open opens (or creates) a .kino database file at path.
func Open(path string) (*DB, error) {
	if filepath.Ext(path) != kinoExtension {
		return nil, fmt.Errorf("kino: file must have %s extension", kinoExtension)
	}

	sqlDB, err := sql.Open("sqlite", "file:"+path+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("kino: open %s: %w", path, err)
	}

	// Keep a single writer connection — WAL handles concurrent reads fine.
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	db := &DB{DB: sqlDB, Path: path}
	if err := db.migrate(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("kino: migrate: %w", err)
	}
	return db, nil
}

// migrate applies any outstanding schema migrations in order.
// Each migration is idempotent; we track applied versions in kino_meta.
func (db *DB) migrate() error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS kino_meta (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)
	`); err != nil {
		return err
	}

	var version int
	row := db.QueryRow(`SELECT value FROM kino_meta WHERE key = 'schema_version'`)
	_ = row.Scan(&version) // no row = version stays 0

	migrations := []struct {
		ver int
		sql string
	}{
		{1, schemav1},
	}

	for _, m := range migrations {
		if version >= m.ver {
			continue
		}
		tx, err := db.DB.Begin()
		if err != nil {
			return fmt.Errorf("migration v%d begin: %w", m.ver, err)
		}
		if _, err = tx.Exec(m.sql); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration v%d: %w", m.ver, err)
		}
		if _, err = tx.Exec(
			`INSERT INTO kino_meta(key,value) VALUES('schema_version',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
			m.ver,
		); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err = tx.Commit(); err != nil {
			return fmt.Errorf("migration v%d commit: %w", m.ver, err)
		}
		version = m.ver
	}
	return nil
}

const schemav1 = `
CREATE TABLE IF NOT EXISTS accounts (
	id           INTEGER PRIMARY KEY,
	name         TEXT    NOT NULL,
	type         TEXT    NOT NULL DEFAULT 'checking',
	institution  TEXT    NOT NULL DEFAULT '',
	balance_cents INTEGER NOT NULL DEFAULT 0,
	currency     TEXT    NOT NULL DEFAULT 'USD',
	is_hidden    INTEGER NOT NULL DEFAULT 0,
	sort_order   INTEGER NOT NULL DEFAULT 0,
	last_synced_at TEXT,
	created_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
	updated_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE TABLE IF NOT EXISTS categories (
	id         INTEGER PRIMARY KEY,
	name       TEXT    NOT NULL,
	parent_id  INTEGER REFERENCES categories(id) ON DELETE SET NULL,
	color      TEXT    NOT NULL DEFAULT '#5A6B60',
	icon       TEXT    NOT NULL DEFAULT 'tag',
	is_income  INTEGER NOT NULL DEFAULT 0,
	is_system  INTEGER NOT NULL DEFAULT 0,
	sort_order INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS transactions (
	id                INTEGER PRIMARY KEY,
	account_id        INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
	date              TEXT    NOT NULL,
	amount_cents      INTEGER NOT NULL,
	payee             TEXT    NOT NULL DEFAULT '',
	payee_normalized  TEXT    NOT NULL DEFAULT '',
	notes             TEXT    NOT NULL DEFAULT '',
	category_id       INTEGER REFERENCES categories(id) ON DELETE SET NULL,
	is_transfer       INTEGER NOT NULL DEFAULT 0,
	transfer_pair_id  INTEGER REFERENCES transactions(id) ON DELETE SET NULL,
	is_reconciled     INTEGER NOT NULL DEFAULT 0,
	import_hash       TEXT    NOT NULL DEFAULT '',
	import_source     TEXT    NOT NULL DEFAULT '',
	created_at        TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
	updated_at        TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE INDEX IF NOT EXISTS idx_transactions_account_date
	ON transactions(account_id, date DESC);

CREATE INDEX IF NOT EXISTS idx_transactions_category
	ON transactions(category_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_transactions_import_hash
	ON transactions(import_hash) WHERE import_hash != '';

CREATE TABLE IF NOT EXISTS budgets (
	id            INTEGER PRIMARY KEY,
	category_id   INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
	amount_cents  INTEGER NOT NULL,
	period        TEXT    NOT NULL DEFAULT 'monthly',
	rolls_over    INTEGER NOT NULL DEFAULT 0,
	start_date    TEXT    NOT NULL,
	end_date      TEXT
);

CREATE TABLE IF NOT EXISTS goals (
	id                INTEGER PRIMARY KEY,
	name              TEXT    NOT NULL,
	type              TEXT    NOT NULL DEFAULT 'savings',
	target_cents      INTEGER NOT NULL DEFAULT 0,
	current_cents     INTEGER NOT NULL DEFAULT 0,
	linked_account_id INTEGER REFERENCES accounts(id) ON DELETE SET NULL,
	target_date       TEXT,
	created_at        TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
	updated_at        TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE TABLE IF NOT EXISTS payee_rules (
	id           INTEGER PRIMARY KEY,
	pattern      TEXT    NOT NULL,
	is_regex     INTEGER NOT NULL DEFAULT 0,
	replace_name TEXT    NOT NULL DEFAULT '',
	category_id  INTEGER REFERENCES categories(id) ON DELETE SET NULL,
	priority     INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS app_settings (
	key   TEXT PRIMARY KEY,
	value TEXT NOT NULL
);

` + seedCategories

// seedCategories inserts the default category set on first run.
// INSERT OR IGNORE means re-running migrations never overwrites user edits.
const seedCategories = `
INSERT OR IGNORE INTO categories(id,name,color,icon,is_income,is_system,sort_order) VALUES
  (1,  'Income',        '#1A8A61', 'banknote',      1, 1,  0),
  (2,  'Housing',       '#6D4C9E', 'home',          0, 1,  1),
  (3,  'Food & Dining', '#C4603A', 'utensils',      0, 1,  2),
  (4,  'Transport',     '#2A7FA8', 'car',           0, 1,  3),
  (5,  'Subscriptions', '#B84A72', 'tv',            0, 1,  4),
  (6,  'Health',        '#4A9E8A', 'heart-pulse',   0, 1,  5),
  (7,  'Entertainment', '#8A5A2A', 'ticket',        0, 1,  6),
  (8,  'Shopping',      '#A87A28', 'shopping-bag',  0, 1,  7),
  (9,  'Savings',       '#C4943A', 'piggy-bank',    0, 1,  8),
  (10, 'Investments',   '#147050', 'trending-up',   0, 1,  9),
  (11, 'Transfers',     '#5A6B60', 'arrow-left-right',0,1,10),
  (12, 'Uncategorized', '#5A6B60', 'tag',           0, 1, 99);
`
