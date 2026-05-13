package main

import (
	"strings"
	"testing"
	"time"

	"kino/internal/models"
)

func TestGetBudgetPage_AggregatesTotals(t *testing.T) {
	a := newTestApp(t)
	accID := insertTestAccount(t, a, "Checking")

	// Create an expense category and a budget for it
	cat := models.Category{Name: "Groceries", Color: "#aabbcc", Icon: "tag"}
	if err := a.db.CreateCategory(&cat); err != nil {
		t.Fatalf("create category: %v", err)
	}

	budget := models.Budget{
		CategoryID:  cat.ID,
		AmountCents: 20000,
		Period:      models.BudgetMonthly,
		StartDate:   mustDate(2025, 1, 1),
	}
	if err := a.db.CreateBudget(&budget); err != nil {
		t.Fatalf("create budget: %v", err)
	}

	// Budgeted category expense
	insertTestTx(t, a, accID, mustDate(2025, 3, 10), -5000, &cat.ID)

	// Unbudgeted category
	cat2 := models.Category{Name: "Coffee", Color: "#112233", Icon: "tag"}
	if err := a.db.CreateCategory(&cat2); err != nil {
		t.Fatalf("create cat2: %v", err)
	}
	insertTestTx(t, a, accID, mustDate(2025, 3, 15), -1000, &cat2.ID)

	page, err := a.GetBudgetPage(2025, 3)
	if err != nil {
		t.Fatalf("GetBudgetPage: %v", err)
	}

	if len(page.Lines) != 1 {
		t.Fatalf("Lines len = %d, want 1", len(page.Lines))
	}
	line := page.Lines[0]
	if line.BudgetCents != 20000 {
		t.Errorf("BudgetCents = %d, want 20000", line.BudgetCents)
	}
	if line.SpentCents != 5000 {
		t.Errorf("SpentCents = %d, want 5000", line.SpentCents)
	}

	if len(page.Unbudgeted) != 1 {
		t.Fatalf("Unbudgeted len = %d, want 1", len(page.Unbudgeted))
	}
	if page.Unbudgeted[0].SpentCents != 1000 {
		t.Errorf("Unbudgeted SpentCents = %d, want 1000", page.Unbudgeted[0].SpentCents)
	}

	if page.TotalBudgetCents != 20000 {
		t.Errorf("TotalBudgetCents = %d, want 20000", page.TotalBudgetCents)
	}
	if page.TotalSpentCents != 6000 {
		t.Errorf("TotalSpentCents = %d, want 6000 (5000+1000)", page.TotalSpentCents)
	}
}

func TestGetBudgetPage_DecemberRollover(t *testing.T) {
	a := newTestApp(t)
	accID := insertTestAccount(t, a, "Checking")

	cat := models.Category{Name: "Food", Color: "#aabbcc", Icon: "tag"}
	if err := a.db.CreateCategory(&cat); err != nil {
		t.Fatalf("create category: %v", err)
	}
	budget := models.Budget{
		CategoryID:  cat.ID,
		AmountCents: 50000,
		Period:      models.BudgetMonthly,
		StartDate:   mustDate(2025, 12, 1),
	}
	if err := a.db.CreateBudget(&budget); err != nil {
		t.Fatalf("create budget: %v", err)
	}

	// December transaction — should be included
	insertTestTx(t, a, accID, mustDate(2025, 12, 31), -3000, &cat.ID)
	// January 1 transaction — should NOT be included (dateTo is exclusive 2026-01-01)
	insertTestTx(t, a, accID, mustDate(2026, 1, 1), -9999, &cat.ID)

	page, err := a.GetBudgetPage(2025, 12)
	if err != nil {
		t.Fatalf("GetBudgetPage: %v", err)
	}
	if len(page.Lines) != 1 {
		t.Fatalf("Lines len = %d, want 1", len(page.Lines))
	}
	if page.Lines[0].SpentCents != 3000 {
		t.Errorf("SpentCents = %d, want 3000 (Jan should be excluded)", page.Lines[0].SpentCents)
	}
}

func TestGetBudgetPage_RequireDB_Error(t *testing.T) {
	_, err := (&App{}).GetBudgetPage(2025, 1)
	if err == nil {
		t.Fatal("expected error from no-DB App, got nil")
	}
	if !strings.Contains(err.Error(), "no file open") {
		t.Errorf("error %q should mention 'no file open'", err)
	}
}

// --- CRUD smoke tests ---

func TestListBudgets_CreateUpdateDelete(t *testing.T) {
	a := newTestApp(t)

	cat := models.Category{Name: "Subscriptions", Color: "#111111", Icon: "tv"}
	if err := a.db.CreateCategory(&cat); err != nil {
		t.Fatalf("create category: %v", err)
	}

	// Create
	b, err := a.CreateBudget(models.Budget{
		CategoryID:  cat.ID,
		AmountCents: 999,
		Period:      models.BudgetMonthly,
	})
	if err != nil {
		t.Fatalf("CreateBudget: %v", err)
	}
	if b.ID == 0 {
		t.Fatal("Budget ID should be non-zero after create")
	}
	// StartDate should have defaulted to today
	if b.StartDate.IsZero() {
		t.Error("StartDate should be set by CreateBudget")
	}

	// List
	list, err := a.ListBudgets()
	if err != nil {
		t.Fatalf("ListBudgets: %v", err)
	}
	if len(list) != 1 || list[0].ID != b.ID {
		t.Fatalf("ListBudgets returned %d items, want 1 with id %d", len(list), b.ID)
	}

	// Update
	b.AmountCents = 1500
	b.StartDate = time.Now() // ensure non-zero
	if err := a.UpdateBudget(b); err != nil {
		t.Fatalf("UpdateBudget: %v", err)
	}
	list2, _ := a.ListBudgets()
	if list2[0].AmountCents != 1500 {
		t.Errorf("after update AmountCents = %d, want 1500", list2[0].AmountCents)
	}

	// Delete
	if err := a.DeleteBudget(b.ID); err != nil {
		t.Fatalf("DeleteBudget: %v", err)
	}
	list3, _ := a.ListBudgets()
	if len(list3) != 0 {
		t.Errorf("after delete ListBudgets len = %d, want 0", len(list3))
	}
}

func TestListBudgets_RequireDB_Error(t *testing.T) {
	_, err := (&App{}).ListBudgets()
	if err == nil {
		t.Fatal("expected error from no-DB App")
	}
}

func TestApp_CreateBudget_RequireDB(t *testing.T) {
	_, err := (&App{}).CreateBudget(models.Budget{})
	if err == nil {
		t.Fatal("expected error from no-DB App, got nil")
	}
	if !strings.Contains(err.Error(), "no file open") {
		t.Errorf("error %q should mention 'no file open'", err)
	}
}

func TestApp_UpdateBudget_RequireDB(t *testing.T) {
	err := (&App{}).UpdateBudget(models.Budget{})
	if err == nil {
		t.Fatal("expected error from no-DB App, got nil")
	}
	if !strings.Contains(err.Error(), "no file open") {
		t.Errorf("error %q should mention 'no file open'", err)
	}
}

func TestApp_DeleteBudget_RequireDB(t *testing.T) {
	err := (&App{}).DeleteBudget(1)
	if err == nil {
		t.Fatal("expected error from no-DB App, got nil")
	}
	if !strings.Contains(err.Error(), "no file open") {
		t.Errorf("error %q should mention 'no file open'", err)
	}
}
