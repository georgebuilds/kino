package main

import (
	"strings"
	"testing"

	"kino/internal/models"
)

// seedCashFlowCategories creates two income and two expense categories, then
// inserts transactions in the given year/month via the given account.
// Returns (inc1ID, inc2ID, exp1ID, exp2ID).
func seedCashFlowCategories(t *testing.T, a *App, accID int64, year, month int) (int64, int64, int64, int64) {
	t.Helper()

	mkCat := func(name string, isIncome bool) int64 {
		c := models.Category{Name: name, Color: "#aaaaaa", Icon: "tag", IsIncome: isIncome}
		if err := a.db.CreateCategory(&c); err != nil {
			t.Fatalf("create cat %q: %v", name, err)
		}
		return c.ID
	}

	inc1ID := mkCat("Salary", true)
	inc2ID := mkCat("Freelance", true)
	exp1ID := mkCat("Rent", false)
	exp2ID := mkCat("Food", false)

	day := mustDate(year, month, 15)
	insertTestTx(t, a, accID, day, 10000, &inc1ID) // +$100
	insertTestTx(t, a, accID, day, 5000, &inc2ID)  // +$50
	insertTestTx(t, a, accID, day, -3000, &exp1ID) // -$30
	insertTestTx(t, a, accID, day, -2000, &exp2ID) // -$20

	return inc1ID, inc2ID, exp1ID, exp2ID
}

func TestGetCashFlow_ProportionalLinks(t *testing.T) {
	a := newTestApp(t)
	accID := insertTestAccount(t, a, "Checking")
	seedCashFlowCategories(t, a, accID, 2025, 6)

	cf, err := a.GetCashFlow(2025, 6)
	if err != nil {
		t.Fatalf("GetCashFlow: %v", err)
	}

	if cf.IncomeCents != 15000 {
		t.Errorf("IncomeCents = %d, want 15000", cf.IncomeCents)
	}
	if cf.ExpenseCents != 5000 {
		t.Errorf("ExpenseCents = %d, want 5000", cf.ExpenseCents)
	}
	if cf.SavedCents != 10000 {
		t.Errorf("SavedCents = %d, want 10000", cf.SavedCents)
	}

	// "Saved" node should appear in RightNodes
	hasSaved := false
	for _, rn := range cf.RightNodes {
		if rn.ID == "saved" {
			hasSaved = true
		}
	}
	if !hasSaved {
		t.Error("RightNodes should contain a 'saved' node when income > expenses")
	}

	// 2 income × 3 right nodes (2 expense + 1 saved) = 6 possible links;
	// all should have v = left.value * right.value / 15000 > 0.
	if len(cf.Links) == 0 {
		t.Fatal("Links should not be empty")
	}
	if len(cf.LeftNodes) != 2 {
		t.Errorf("LeftNodes len = %d, want 2", len(cf.LeftNodes))
	}
	// 2 expense categories + saved
	if len(cf.RightNodes) != 3 {
		t.Errorf("RightNodes len = %d, want 3", len(cf.RightNodes))
	}

	// Verify proportional math for one link: inc1 (10000) → saved (10000) / denom(15000) = 6666
	for _, lk := range cf.Links {
		if strings.HasSuffix(lk.TargetID, "saved") && strings.Contains(lk.SourceID, "inc-") {
			// find which left node this came from
			for _, ln := range cf.LeftNodes {
				if ln.ID == lk.SourceID {
					expectedV := ln.ValueCents * 10000 / 15000
					if lk.ValueCents != expectedV {
						t.Errorf("link %s→saved value = %d, want %d", lk.SourceID, lk.ValueCents, expectedV)
					}
				}
			}
		}
	}
}

func TestGetCashFlow_NoSavedNode_WhenSpendingExceedsIncome(t *testing.T) {
	a := newTestApp(t)
	accID := insertTestAccount(t, a, "Checking")

	cat1 := models.Category{Name: "Salary", Color: "#111", Icon: "tag", IsIncome: true}
	if err := a.db.CreateCategory(&cat1); err != nil {
		t.Fatalf("create cat1: %v", err)
	}
	cat2 := models.Category{Name: "Rent", Color: "#222", Icon: "tag"}
	if err := a.db.CreateCategory(&cat2); err != nil {
		t.Fatalf("create cat2: %v", err)
	}

	day := mustDate(2025, 7, 10)
	insertTestTx(t, a, accID, day, 5000, &cat1.ID)   // income $50
	insertTestTx(t, a, accID, day, -10000, &cat2.ID) // expense $100

	cf, err := a.GetCashFlow(2025, 7)
	if err != nil {
		t.Fatalf("GetCashFlow: %v", err)
	}

	if cf.SavedCents > 0 {
		t.Errorf("SavedCents = %d, want <= 0 when spending exceeds income", cf.SavedCents)
	}

	for _, rn := range cf.RightNodes {
		if rn.ID == "saved" {
			t.Error("RightNodes should NOT contain 'saved' when income <= expenses")
		}
	}
}

func TestGetCashFlow_EmptyMonth(t *testing.T) {
	a := newTestApp(t)
	_, err := a.GetCashFlow(2025, 5)
	if err != nil {
		t.Fatalf("GetCashFlow on empty DB: %v", err)
	}
	// Should return zero totals, no panic.
	// (Function returns early when totalIncome==0 && totalExpenses==0)
}

func TestGetCashFlow_DecemberRollover(t *testing.T) {
	a := newTestApp(t)
	accID := insertTestAccount(t, a, "Checking")

	incCat := models.Category{Name: "Work", Color: "#111", Icon: "tag", IsIncome: true}
	if err := a.db.CreateCategory(&incCat); err != nil {
		t.Fatalf("create incCat: %v", err)
	}

	insertTestTx(t, a, accID, mustDate(2025, 12, 31), 1000, &incCat.ID) // inside
	insertTestTx(t, a, accID, mustDate(2026, 1, 1), 9999, &incCat.ID)   // outside — exclusive boundary

	cf, err := a.GetCashFlow(2025, 12)
	if err != nil {
		t.Fatalf("GetCashFlow: %v", err)
	}
	if cf.IncomeCents != 1000 {
		t.Errorf("IncomeCents = %d, want 1000 (Jan should be excluded)", cf.IncomeCents)
	}
}

func TestGetCashFlow_RequireDB_Error(t *testing.T) {
	_, err := (&App{}).GetCashFlow(2025, 1)
	if err == nil {
		t.Fatal("expected error from no-DB App")
	}
	if !strings.Contains(err.Error(), "no file open") {
		t.Errorf("error %q should mention 'no file open'", err)
	}
}

// TestGetCashFlow_ZeroIncome_ExpensesOnly exercises the denom fallback at
// app_cashflow.go ~line 158: when totalIncome == 0 but totalExpenses > 0 the
// denominator falls back to totalExpenses so the division in the links loop
// does not panic.  With no left nodes the links slice will be empty — that is
// fine; the important things are: no panic, IncomeCents == 0, and
// ExpenseCents reflects the inserted transactions.
func TestGetCashFlow_ZeroIncome_ExpensesOnly(t *testing.T) {
	a := newTestApp(t)
	accID := insertTestAccount(t, a, "Checking")

	expCat := models.Category{Name: "Groceries", Color: "#C4603A", Icon: "utensils"}
	if err := a.db.CreateCategory(&expCat); err != nil {
		t.Fatalf("create expCat: %v", err)
	}

	day := mustDate(2025, 9, 5)
	insertTestTx(t, a, accID, day, -8000, &expCat.ID) // -$80 expense, no income

	cf, err := a.GetCashFlow(2025, 9)
	if err != nil {
		t.Fatalf("GetCashFlow: %v", err)
	}

	if cf.IncomeCents != 0 {
		t.Errorf("IncomeCents = %d, want 0", cf.IncomeCents)
	}
	if cf.ExpenseCents != 8000 {
		t.Errorf("ExpenseCents = %d, want 8000", cf.ExpenseCents)
	}
	// savedCents = 0 - 8000 = -8000; no "saved" node
	if cf.SavedCents > 0 {
		t.Errorf("SavedCents = %d, want <= 0 (pure-expense month)", cf.SavedCents)
	}
	for _, rn := range cf.RightNodes {
		if rn.ID == "saved" {
			t.Error("RightNodes should NOT contain 'saved' node when income is zero")
		}
	}
	// No left nodes → links loop produces nothing regardless of denom value.
	if len(cf.LeftNodes) != 0 {
		t.Errorf("LeftNodes len = %d, want 0 (no income transactions)", len(cf.LeftNodes))
	}
	if len(cf.Links) != 0 {
		t.Errorf("Links len = %d, want 0 (no left nodes to link from)", len(cf.Links))
	}
}

// TestGetCashFlow_NullCategory_BucketedAsUncategorized verifies that
// transactions with nil CategoryID are bucketed into the seeded
// "Uncategorized" category (id 12) via COALESCE in both SQL queries.
func TestGetCashFlow_NullCategory_BucketedAsUncategorized(t *testing.T) {
	a := newTestApp(t)
	accID := insertTestAccount(t, a, "Checking")

	day := mustDate(2025, 10, 1)
	insertTestTx(t, a, accID, day, 12000, nil)  // income, no category
	insertTestTx(t, a, accID, day, -4000, nil)  // expense, no category

	cf, err := a.GetCashFlow(2025, 10)
	if err != nil {
		t.Fatalf("GetCashFlow: %v", err)
	}

	if cf.IncomeCents != 12000 {
		t.Errorf("IncomeCents = %d, want 12000", cf.IncomeCents)
	}
	if cf.ExpenseCents != 4000 {
		t.Errorf("ExpenseCents = %d, want 4000", cf.ExpenseCents)
	}

	if len(cf.LeftNodes) != 1 {
		t.Fatalf("LeftNodes len = %d, want 1 (all income bucketed into Uncategorized)", len(cf.LeftNodes))
	}
	if !strings.Contains(cf.LeftNodes[0].Name, "Uncategorized") {
		t.Errorf("LeftNodes[0].Name = %q, want it to contain 'Uncategorized'", cf.LeftNodes[0].Name)
	}

	// RightNodes: one expense bucket (Uncategorized) — no "Saved" because income > expenses,
	// so there will also be a "saved" node.
	hasUncategorizedExpense := false
	for _, rn := range cf.RightNodes {
		if strings.Contains(rn.Name, "Uncategorized") {
			hasUncategorizedExpense = true
		}
	}
	if !hasUncategorizedExpense {
		t.Error("RightNodes should contain an 'Uncategorized' expense node")
	}
}
