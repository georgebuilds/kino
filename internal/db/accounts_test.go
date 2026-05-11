package db

import (
	"path/filepath"
	"testing"

	"kino/internal/models"
)

// newTestDB creates a fresh on-disk .kino DB scoped to t.TempDir().
func newTestDB(t *testing.T) *DB {
	t.Helper()
	p := filepath.Join(t.TempDir(), "test.kino")
	d, err := Open(p)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d
}

func TestCreateAndListAccounts(t *testing.T) {
	d := newTestDB(t)

	visible := &models.Account{
		Name: "Everyday", Type: models.AccountChecking, BalanceCents: 12345,
		Currency: "USD", IsHidden: false,
	}
	hidden := &models.Account{
		Name: "Old Savings", Type: models.AccountSavings, BalanceCents: 9999,
		Currency: "USD", IsHidden: true,
	}
	if err := d.CreateAccount(visible); err != nil {
		t.Fatalf("create visible: %v", err)
	}
	if err := d.CreateAccount(hidden); err != nil {
		t.Fatalf("create hidden: %v", err)
	}

	listed, err := d.ListAccounts()
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("ListAccounts returned %d accounts, want 1 (hidden excluded)", len(listed))
	}
	if listed[0].ID != visible.ID || listed[0].Name != "Everyday" {
		t.Fatalf("listed account = %+v, want visible account %+v", listed[0], visible)
	}

	// The hidden account should be returned when we ask for it directly.
	all, err := d.listAccounts(true)
	if err != nil {
		t.Fatalf("listAccounts(true): %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("listAccounts(true) returned %d accounts, want 2", len(all))
	}
}

func TestDeleteAccount_Succeeds(t *testing.T) {
	d := newTestDB(t)

	a := &models.Account{Name: "Doomed", Type: models.AccountChecking, Currency: "USD"}
	if err := d.CreateAccount(a); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Regression: DeleteAccount used to reference a non-existent is_system column.
	if err := d.DeleteAccount(a.ID); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	got, err := d.GetAccount(a.ID)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if got != nil {
		t.Fatalf("GetAccount after delete = %+v, want nil", got)
	}
}

func TestRecalcBalance(t *testing.T) {
	d := newTestDB(t)

	a := &models.Account{Name: "Calc", Type: models.AccountChecking, BalanceCents: 0, Currency: "USD"}
	if err := d.CreateAccount(a); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Three transactions: +500, -200, +50  → balance should be 350.
	amounts := []int64{500, -200, 50}
	for i, amt := range amounts {
		tx := &models.Transaction{
			AccountID:   a.ID,
			Date:        mustDate(2025, 1, 10+i),
			AmountCents: amt,
			Payee:       "n/a",
		}
		if err := d.CreateTransaction(tx); err != nil {
			t.Fatalf("create tx %d: %v", i, err)
		}
	}

	if err := d.RecalcBalance(a.ID); err != nil {
		t.Fatalf("RecalcBalance: %v", err)
	}
	got, err := d.GetAccount(a.ID)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if got == nil {
		t.Fatal("GetAccount returned nil")
	}
	if got.BalanceCents != 350 {
		t.Fatalf("BalanceCents = %d, want 350", got.BalanceCents)
	}
}

func TestUpdateAccount(t *testing.T) {
	d := newTestDB(t)

	a := &models.Account{Name: "Before", Type: models.AccountChecking, BalanceCents: 100, Currency: "USD"}
	if err := d.CreateAccount(a); err != nil {
		t.Fatalf("create: %v", err)
	}

	a.Name = "After"
	a.BalanceCents = 9999
	a.Currency = "EUR"
	if err := d.UpdateAccount(a); err != nil {
		t.Fatalf("UpdateAccount: %v", err)
	}

	got, err := d.GetAccount(a.ID)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if got == nil {
		t.Fatal("GetAccount returned nil after update")
	}
	if got.Name != "After" || got.BalanceCents != 9999 || got.Currency != "EUR" {
		t.Fatalf("UpdateAccount did not persist: got %+v", got)
	}
}
