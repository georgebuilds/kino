package main

import (
	"fmt"
	"strings"
	"testing"

	"kino/internal/models"
)

func TestGetMonthSummary_PopulatesAllFields(t *testing.T) {
	a := newTestApp(t)

	// Visible account
	visAcc := models.Account{Name: "Checking", Type: models.AccountChecking, Currency: "USD", BalanceCents: 0}
	if err := a.db.CreateAccount(&visAcc); err != nil {
		t.Fatalf("create visible account: %v", err)
	}

	// Hidden account — should NOT be counted in NetWorthCents
	hidAcc := models.Account{Name: "Hidden", Type: models.AccountChecking, Currency: "USD", BalanceCents: 0, IsHidden: true}
	if err := a.db.CreateAccount(&hidAcc); err != nil {
		t.Fatalf("create hidden account: %v", err)
	}

	incCat := models.Category{Name: "Salary", Color: "#1A8A61", Icon: "banknote", IsIncome: true}
	if err := a.db.CreateCategory(&incCat); err != nil {
		t.Fatalf("create incCat: %v", err)
	}
	expCat := models.Category{Name: "Rent", Color: "#6D4C9E", Icon: "home"}
	if err := a.db.CreateCategory(&expCat); err != nil {
		t.Fatalf("create expCat: %v", err)
	}

	day := mustDate(2025, 4, 10)
	insertTestTx(t, a, visAcc.ID, day, 50000, &incCat.ID)  // +$500 income
	insertTestTx(t, a, visAcc.ID, day, -20000, &expCat.ID) // -$200 expense (top category)
	// Transfer transaction — included in NetWorthDelta, excluded from income/expense
	tx := models.Transaction{AccountID: visAcc.ID, Date: day, AmountCents: -5000, Payee: "xfer", IsTransfer: true}
	if err := a.db.CreateTransaction(&tx); err != nil {
		t.Fatalf("create transfer: %v", err)
	}
	// Hidden account transaction — should NOT appear in net worth
	insertTestTx(t, a, hidAcc.ID, day, 99999, nil)

	// Recalc balances
	if err := a.db.RecalcBalance(visAcc.ID); err != nil {
		t.Fatalf("RecalcBalance vis: %v", err)
	}
	if err := a.db.RecalcBalance(hidAcc.ID); err != nil {
		t.Fatalf("RecalcBalance hid: %v", err)
	}

	s, err := a.GetMonthSummary(2025, 4)
	if err != nil {
		t.Fatalf("GetMonthSummary: %v", err)
	}

	// Net worth = visible account balance only = 50000 - 20000 - 5000 = 25000
	if s.NetWorthCents != 25000 {
		t.Errorf("NetWorthCents = %d, want 25000", s.NetWorthCents)
	}

	// The summary query does NOT filter by account visibility — it reads all
	// transactions regardless. So income includes the hidden account's +99999.
	// Income = 50000 (vis) + 99999 (hidden) = 149999
	if s.IncomeCents != 149999 {
		t.Errorf("IncomeCents = %d, want 149999 (includes hidden account tx)", s.IncomeCents)
	}

	// Expense = absolute of non-transfer negative amounts (only visible transfer -5000 is filtered,
	// expense -20000 remains)
	if s.ExpenseCents != 20000 {
		t.Errorf("ExpenseCents = %d, want 20000", s.ExpenseCents)
	}

	// Saved = Income - Expense = 149999 - 20000 = 129999
	if s.SavedCents != 129999 {
		t.Errorf("SavedCents = %d, want 129999", s.SavedCents)
	}

	// NetWorthDelta = SUM(non-transfer amount_cents) = 50000 - 20000 + 99999 = 129999
	if s.NetWorthDeltaCents != 129999 {
		t.Errorf("NetWorthDeltaCents = %d, want 129999", s.NetWorthDeltaCents)
	}

	if s.TopCategory != "Rent" {
		t.Errorf("TopCategory = %q, want %q", s.TopCategory, "Rent")
	}
	if s.TopCategoryCents != 20000 {
		t.Errorf("TopCategoryCents = %d, want 20000", s.TopCategoryCents)
	}

	if len(s.CategoryTotals) == 0 {
		t.Error("CategoryTotals should not be empty")
	}
}

func TestGetMonthSummary_CategoryTotals_CapAt12(t *testing.T) {
	a := newTestApp(t)
	accID := insertTestAccount(t, a, "Checking")

	// Create 13 distinct expense categories, each with one transaction
	for i := 0; i < 13; i++ {
		cat := models.Category{
			Name:  fmt.Sprintf("Exp%02d", i),
			Color: "#aabbcc",
			Icon:  "tag",
		}
		if err := a.db.CreateCategory(&cat); err != nil {
			t.Fatalf("create cat %d: %v", i, err)
		}
		// Vary amounts so ordering is stable
		insertTestTx(t, a, accID, mustDate(2025, 8, 1), -(int64(i+1) * 100), &cat.ID)
	}

	s, err := a.GetMonthSummary(2025, 8)
	if err != nil {
		t.Fatalf("GetMonthSummary: %v", err)
	}
	if len(s.CategoryTotals) != 12 {
		t.Errorf("CategoryTotals len = %d, want 12 (LIMIT 12)", len(s.CategoryTotals))
	}
}

func TestGetMonthSummary_EmptyMonth_NoTopCategory(t *testing.T) {
	a := newTestApp(t)

	s, err := a.GetMonthSummary(2025, 2)
	if err != nil {
		t.Fatalf("GetMonthSummary: %v", err)
	}

	if s.TopCategory != "" {
		t.Errorf("TopCategory = %q, want empty", s.TopCategory)
	}
	if s.TopCategoryCents != 0 {
		t.Errorf("TopCategoryCents = %d, want 0", s.TopCategoryCents)
	}
	// app_summary.go:92 ensures CategoryTotals is never nil
	if s.CategoryTotals == nil {
		t.Error("CategoryTotals should be [] not nil for empty month")
	}
	if len(s.CategoryTotals) != 0 {
		t.Errorf("CategoryTotals len = %d, want 0", len(s.CategoryTotals))
	}
}

func TestGetMonthSummary_DecemberRollover(t *testing.T) {
	a := newTestApp(t)
	accID := insertTestAccount(t, a, "Checking")

	expCat := models.Category{Name: "Food", Color: "#aabbcc", Icon: "tag"}
	if err := a.db.CreateCategory(&expCat); err != nil {
		t.Fatalf("create cat: %v", err)
	}

	insertTestTx(t, a, accID, mustDate(2025, 12, 31), -4000, &expCat.ID) // in December
	insertTestTx(t, a, accID, mustDate(2026, 1, 1), -9999, &expCat.ID)   // in January — excluded

	s, err := a.GetMonthSummary(2025, 12)
	if err != nil {
		t.Fatalf("GetMonthSummary: %v", err)
	}
	if s.ExpenseCents != 4000 {
		t.Errorf("ExpenseCents = %d, want 4000 (Jan excluded)", s.ExpenseCents)
	}
}

func TestGetMonthSummary_RequireDB_Error(t *testing.T) {
	_, err := (&App{}).GetMonthSummary(2025, 1)
	if err == nil {
		t.Fatal("expected error from no-DB App")
	}
	if !strings.Contains(err.Error(), "no file open") {
		t.Errorf("error %q should mention 'no file open'", err)
	}
}

// TestGetMonthSummary_NullCategory_BucketedAsUncategorized verifies that
// expense transactions with nil CategoryID appear in CategoryTotals under the
// seeded "Uncategorized" category (id 12) via the COALESCE in the category
// breakdown query.
func TestGetMonthSummary_NullCategory_BucketedAsUncategorized(t *testing.T) {
	a := newTestApp(t)
	accID := insertTestAccount(t, a, "Checking")

	day := mustDate(2025, 11, 20)
	insertTestTx(t, a, accID, day, -6000, nil) // -$60 expense, no category

	s, err := a.GetMonthSummary(2025, 11)
	if err != nil {
		t.Fatalf("GetMonthSummary: %v", err)
	}

	if s.ExpenseCents != 6000 {
		t.Errorf("ExpenseCents = %d, want 6000", s.ExpenseCents)
	}
	if len(s.CategoryTotals) == 0 {
		t.Fatal("CategoryTotals should not be empty for an uncategorized expense")
	}
	if s.CategoryTotals[0].CategoryName != "Uncategorized" {
		t.Errorf("CategoryTotals[0].CategoryName = %q, want %q", s.CategoryTotals[0].CategoryName, "Uncategorized")
	}
	if s.CategoryTotals[0].AmountCents != 6000 {
		t.Errorf("CategoryTotals[0].AmountCents = %d, want 6000", s.CategoryTotals[0].AmountCents)
	}
	if s.TopCategory != "Uncategorized" {
		t.Errorf("TopCategory = %q, want %q", s.TopCategory, "Uncategorized")
	}
}
