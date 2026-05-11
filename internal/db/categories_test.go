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

func TestGetCategory_Found_And_Missing(t *testing.T) {
	d := newTestDB(t)

	cat := &models.Category{Name: "Found", Color: "#123456", Icon: "tag"}
	if err := d.CreateCategory(cat); err != nil {
		t.Fatalf("CreateCategory: %v", err)
	}

	got, err := d.GetCategory(cat.ID)
	if err != nil {
		t.Fatalf("GetCategory: %v", err)
	}
	if got == nil {
		t.Fatal("GetCategory returned nil for existing category")
	}
	if got.Name != "Found" {
		t.Fatalf("Name = %q, want %q", got.Name, "Found")
	}

	missing, err := d.GetCategory(99999)
	if err != nil {
		t.Fatalf("GetCategory(missing) error: %v", err)
	}
	if missing != nil {
		t.Fatalf("GetCategory(99999) = %+v, want nil", missing)
	}
}

func TestCreateCategory_PopulatesID_AndRoundTrips(t *testing.T) {
	d := newTestDB(t)

	cat := &models.Category{Name: "RoundTrip", Color: "#aabbcc", Icon: "star", IsIncome: true, SortOrder: 5}
	if err := d.CreateCategory(cat); err != nil {
		t.Fatalf("CreateCategory: %v", err)
	}
	if cat.ID == 0 {
		t.Fatal("CreateCategory did not populate ID")
	}

	list, err := d.ListCategories()
	if err != nil {
		t.Fatalf("ListCategories: %v", err)
	}
	var found bool
	for _, c := range list {
		if c.ID == cat.ID {
			found = true
			if c.Name != "RoundTrip" || c.Color != "#aabbcc" || c.Icon != "star" {
				t.Fatalf("round-trip mismatch: %+v", c)
			}
			break
		}
	}
	if !found {
		t.Fatalf("created category id=%d not found in ListCategories", cat.ID)
	}
}

func TestUpdateCategory_NonSystem_Persists(t *testing.T) {
	d := newTestDB(t)

	cat := &models.Category{Name: "Before", Color: "#000000", Icon: "tag", IsIncome: false, SortOrder: 1}
	if err := d.CreateCategory(cat); err != nil {
		t.Fatalf("CreateCategory: %v", err)
	}

	cat.Name = "After"
	cat.Color = "#ffffff"
	cat.Icon = "star"
	cat.SortOrder = 42
	cat.IsIncome = true
	if err := d.UpdateCategory(cat); err != nil {
		t.Fatalf("UpdateCategory: %v", err)
	}

	got, err := d.GetCategory(cat.ID)
	if err != nil {
		t.Fatalf("GetCategory: %v", err)
	}
	if got == nil {
		t.Fatal("GetCategory returned nil after update")
	}
	if got.Name != "After" || got.Color != "#ffffff" || got.Icon != "star" ||
		got.SortOrder != 42 || !got.IsIncome {
		t.Fatalf("UpdateCategory did not persist: got %+v", got)
	}
}

func TestUpdateCategory_System_ReturnsError_NoChange(t *testing.T) {
	d := newTestDB(t)

	// id=12 "Uncategorized" is a seeded system category.
	sys, err := d.GetCategory(12)
	if err != nil {
		t.Fatalf("GetCategory(12): %v", err)
	}
	if sys == nil || !sys.IsSystem {
		t.Fatalf("seed category 12 missing or not system: %+v", sys)
	}

	// Attempt to mutate it — UpdateCategory checks c.IsSystem on the struct.
	sys.Name = "Hacked"
	updateErr := d.UpdateCategory(sys)
	if updateErr == nil {
		t.Fatal("UpdateCategory(system) returned nil, want error")
	}

	// Verify the name is unchanged.
	still, err := d.GetCategory(12)
	if err != nil {
		t.Fatalf("GetCategory after failed update: %v", err)
	}
	if still == nil {
		t.Fatal("system category disappeared after failed update")
	}
	if still.Name == "Hacked" {
		t.Fatal("system category name was changed despite error")
	}
}

func TestListCategories_OrdersBySortOrder(t *testing.T) {
	d := newTestDB(t)

	// Create three user categories with different sort_order values.
	cats := []*models.Category{
		{Name: "Middle", Color: "#111", Icon: "tag", SortOrder: 50},
		{Name: "Last", Color: "#222", Icon: "tag", SortOrder: 100},
		{Name: "First", Color: "#333", Icon: "tag", SortOrder: 10},
	}
	for _, c := range cats {
		if err := d.CreateCategory(c); err != nil {
			t.Fatalf("CreateCategory %q: %v", c.Name, err)
		}
	}

	list, err := d.ListCategories()
	if err != nil {
		t.Fatalf("ListCategories: %v", err)
	}

	// Extract just the user-created names (seeded system categories come first
	// or are interleaved by sort_order; find the three we inserted in order).
	var got []string
	for _, c := range list {
		if c.Name == "First" || c.Name == "Middle" || c.Name == "Last" {
			got = append(got, c.Name)
		}
	}
	want := []string{"First", "Middle", "Last"}
	if len(got) != len(want) {
		t.Fatalf("user category count = %d, want %d; all: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order[%d] = %q, want %q", i, got[i], want[i])
		}
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
