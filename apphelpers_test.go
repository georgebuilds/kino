package main

import (
	"path/filepath"
	"testing"
	"time"

	"kino/internal/db"
	"kino/internal/models"
)

// newTestApp opens a fresh test DB at t.TempDir()/test.kino and returns
// an *App wired to it. The ctx field stays nil — none of the tested methods
// use it.
func newTestApp(t *testing.T) *App {
	t.Helper()
	p := filepath.Join(t.TempDir(), "test.kino")
	d, err := db.Open(p)
	if err != nil {
		t.Fatalf("newTestApp: open db: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return &App{db: d}
}

// mustDate returns a UTC time at midnight for the given Y/M/D.
func mustDate(y, m, d int) time.Time {
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
}

// i64ptr returns a pointer to the given int64 value.
func i64ptr(v int64) *int64 { return &v }

// insertTestAccount creates an account via app.db.CreateAccount and returns its ID.
func insertTestAccount(t *testing.T, a *App, name string) int64 {
	t.Helper()
	acc := models.Account{Name: name, Type: models.AccountChecking, Currency: "USD"}
	if err := a.db.CreateAccount(&acc); err != nil {
		t.Fatalf("insertTestAccount %q: %v", name, err)
	}
	return acc.ID
}

// insertTestTx creates a transaction directly via app.db.CreateTransaction.
func insertTestTx(t *testing.T, a *App, accountID int64, date time.Time, amountCents int64, catID *int64) int64 {
	t.Helper()
	tx := models.Transaction{
		AccountID:   accountID,
		Date:        date,
		AmountCents: amountCents,
		Payee:       "test",
		CategoryID:  catID,
	}
	if err := a.db.CreateTransaction(&tx); err != nil {
		t.Fatalf("insertTestTx: %v", err)
	}
	return tx.ID
}
