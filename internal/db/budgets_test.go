package db

import (
	"testing"

	"kino/internal/models"
)

func TestGetBudgetProgress_BudgetedAndUnbudgeted(t *testing.T) {
	d := newTestDB(t)
	accID := insertTestAccount(t, d, "Wallet")

	catA := &models.Category{Name: "Eats", Color: "#abc", Icon: "u", IsIncome: false}
	catB := &models.Category{Name: "Toys", Color: "#bcd", Icon: "u", IsIncome: false}
	if err := d.CreateCategory(catA); err != nil {
		t.Fatalf("create catA: %v", err)
	}
	if err := d.CreateCategory(catB); err != nil {
		t.Fatalf("create catB: %v", err)
	}

	// $200 budget on catA.
	budget := &models.Budget{
		CategoryID:  catA.ID,
		AmountCents: 20000,
		Period:      models.BudgetMonthly,
		StartDate:   mustDate(2025, 9, 1),
	}
	if err := d.CreateBudget(budget); err != nil {
		t.Fatalf("CreateBudget: %v", err)
	}

	// $50 expense in catA, $30 in catB. Same month.
	rows := []models.Transaction{
		{AccountID: accID, Date: mustDate(2025, 9, 5), AmountCents: -5000, Payee: "x", CategoryID: i64ptr(catA.ID)},
		{AccountID: accID, Date: mustDate(2025, 9, 7), AmountCents: -3000, Payee: "y", CategoryID: i64ptr(catB.ID)},
	}
	for i := range rows {
		if err := d.CreateTransaction(&rows[i]); err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}

	// Note the query uses `date < dateTo`, so pass first-of-next-month.
	budgeted, unbudgeted, err := d.GetBudgetProgress("2025-09-01", "2025-10-01")
	if err != nil {
		t.Fatalf("GetBudgetProgress: %v", err)
	}

	if len(budgeted) != 1 {
		t.Fatalf("budgeted: got %d, want 1", len(budgeted))
	}
	if budgeted[0].Budget.CategoryID != catA.ID || budgeted[0].SpentCents != 5000 {
		t.Fatalf("budgeted[0] = %+v, want catID=%d Spent=5000", budgeted[0], catA.ID)
	}

	if len(unbudgeted) != 1 {
		t.Fatalf("unbudgeted: got %d, want 1: %+v", len(unbudgeted), unbudgeted)
	}
	if unbudgeted[0].CategoryID != catB.ID || unbudgeted[0].SpentCents != 3000 {
		t.Fatalf("unbudgeted[0] = %+v, want catID=%d Spent=3000", unbudgeted[0], catB.ID)
	}
}

func TestGetBudgetProgress_ExcludesTransfers(t *testing.T) {
	d := newTestDB(t)
	accID := insertTestAccount(t, d, "Wallet")

	catA := &models.Category{Name: "EatsXfer", Color: "#abc", Icon: "u"}
	if err := d.CreateCategory(catA); err != nil {
		t.Fatalf("create catA: %v", err)
	}

	budget := &models.Budget{
		CategoryID:  catA.ID,
		AmountCents: 20000,
		Period:      models.BudgetMonthly,
		StartDate:   mustDate(2025, 9, 1),
	}
	if err := d.CreateBudget(budget); err != nil {
		t.Fatalf("CreateBudget: %v", err)
	}

	// One real expense and one transfer in the same category.
	rows := []models.Transaction{
		{AccountID: accID, Date: mustDate(2025, 9, 5), AmountCents: -5000, Payee: "x", CategoryID: i64ptr(catA.ID)},
		{AccountID: accID, Date: mustDate(2025, 9, 6), AmountCents: -9999, Payee: "xfer", CategoryID: i64ptr(catA.ID), IsTransfer: true},
	}
	for i := range rows {
		if err := d.CreateTransaction(&rows[i]); err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}

	budgeted, _, err := d.GetBudgetProgress("2025-09-01", "2025-10-01")
	if err != nil {
		t.Fatalf("GetBudgetProgress: %v", err)
	}
	if len(budgeted) != 1 {
		t.Fatalf("budgeted len = %d, want 1", len(budgeted))
	}
	if budgeted[0].SpentCents != 5000 {
		t.Fatalf("SpentCents = %d, want 5000 (transfer excluded)", budgeted[0].SpentCents)
	}
}
