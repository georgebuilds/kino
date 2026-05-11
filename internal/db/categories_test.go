package db

import (
	"strings"
	"testing"

	"kino/internal/models"
)

// helper: insert an account so transactions can FK into it.
func insertTestAccount(t *testing.T, d *DB, name string) int64 {
	t.Helper()
	a := &models.Account{Name: name, Type: models.AccountChecking, Currency: "USD"}
	if err := d.CreateAccount(a); err != nil {
		t.Fatalf("create account %q: %v", name, err)
	}
	return a.ID
}

// helper: shortcut to point to a literal int64.
func i64ptr(v int64) *int64 { return &v }

func TestDeleteCategory_NonSystem_ReassignsTransactionsAndDeletes(t *testing.T) {
	d := newTestDB(t)
	accID := insertTestAccount(t, d, "Acct")

	cat := &models.Category{Name: "Groceries", Color: "#abcdef", Icon: "tag"}
	if err := d.CreateCategory(cat); err != nil {
		t.Fatalf("CreateCategory: %v", err)
	}

	tx := &models.Transaction{
		AccountID:   accID,
		Date:        mustDate(2025, 3, 15),
		AmountCents: -1234,
		Payee:       "Whole Foods",
		CategoryID:  i64ptr(cat.ID),
	}
	if err := d.CreateTransaction(tx); err != nil {
		t.Fatalf("CreateTransaction: %v", err)
	}

	if err := d.DeleteCategory(cat.ID); err != nil {
		t.Fatalf("DeleteCategory: %v", err)
	}

	gone, err := d.GetCategory(cat.ID)
	if err != nil {
		t.Fatalf("GetCategory: %v", err)
	}
	if gone != nil {
		t.Fatalf("category still exists after delete: %+v", gone)
	}

	got, err := d.GetTransaction(tx.ID)
	if err != nil {
		t.Fatalf("GetTransaction: %v", err)
	}
	if got == nil {
		t.Fatal("transaction missing after category delete")
	}
	if got.CategoryID == nil || *got.CategoryID != 12 {
		t.Fatalf("transaction.CategoryID = %v, want 12 (Uncategorized)", got.CategoryID)
	}
}

func TestDeleteCategory_System_ReturnsError_LeavesTransactionsAlone(t *testing.T) {
	d := newTestDB(t)
	accID := insertTestAccount(t, d, "Acct")

	// Confirm seeded system category exists.
	sys, err := d.GetCategory(1)
	if err != nil {
		t.Fatalf("GetCategory(1): %v", err)
	}
	if sys == nil || !sys.IsSystem {
		t.Fatalf("seed category 1 missing or not system: %+v", sys)
	}

	// A transaction categorized as system category 1 (Income).
	tx := &models.Transaction{
		AccountID:   accID,
		Date:        mustDate(2025, 4, 1),
		AmountCents: 50000,
		Payee:       "Payroll",
		CategoryID:  i64ptr(1),
	}
	if err := d.CreateTransaction(tx); err != nil {
		t.Fatalf("CreateTransaction: %v", err)
	}

	// Attempt the delete — must fail.
	delErr := d.DeleteCategory(1)
	if delErr == nil {
		t.Fatal("DeleteCategory(system) returned nil error, want error")
	}

	// Category must still exist.
	still, err := d.GetCategory(1)
	if err != nil {
		t.Fatalf("GetCategory: %v", err)
	}
	if still == nil {
		t.Fatal("system category was deleted despite error")
	}

	// Atomicity regression: the UPDATE must not have leaked.
	got, err := d.GetTransaction(tx.ID)
	if err != nil {
		t.Fatalf("GetTransaction: %v", err)
	}
	if got == nil {
		t.Fatal("transaction disappeared")
	}
	if got.CategoryID == nil || *got.CategoryID != 1 {
		t.Fatalf("transaction.CategoryID = %v, want 1 (untouched after failed delete)", got.CategoryID)
	}
}

func TestDeleteCategory_Missing_ReturnsError(t *testing.T) {
	d := newTestDB(t)
	err := d.DeleteCategory(99999)
	if err == nil {
		t.Fatal("DeleteCategory(99999) returned nil, want error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("error %q does not mention 'not found'", err.Error())
	}
}

func TestCountTransactionsByCategory(t *testing.T) {
	d := newTestDB(t)
	accID := insertTestAccount(t, d, "Acct")

	cat := &models.Category{Name: "Coffee", Color: "#6b3", Icon: "cup"}
	if err := d.CreateCategory(cat); err != nil {
		t.Fatalf("CreateCategory: %v", err)
	}

	const want = 4
	for i := 0; i < want; i++ {
		tx := &models.Transaction{
			AccountID:   accID,
			Date:        mustDate(2025, 5, 10+i),
			AmountCents: -500,
			Payee:       "Cafe",
			CategoryID:  i64ptr(cat.ID),
		}
		if err := d.CreateTransaction(tx); err != nil {
			t.Fatalf("CreateTransaction %d: %v", i, err)
		}
	}

	n, err := d.CountTransactionsByCategory(cat.ID)
	if err != nil {
		t.Fatalf("CountTransactionsByCategory: %v", err)
	}
	if n != want {
		t.Fatalf("count = %d, want %d", n, want)
	}
}
