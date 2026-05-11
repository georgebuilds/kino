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

func TestListBudgets_RoundTrip(t *testing.T) {
	d := newTestDB(t)

	catA := &models.Category{Name: "BudgCatA", Color: "#aaa", Icon: "tag"}
	catB := &models.Category{Name: "BudgCatB", Color: "#bbb", Icon: "tag"}
	if err := d.CreateCategory(catA); err != nil {
		t.Fatalf("create catA: %v", err)
	}
	if err := d.CreateCategory(catB); err != nil {
		t.Fatalf("create catB: %v", err)
	}

	endDate := mustDate(2025, 12, 31)
	b1 := &models.Budget{
		CategoryID:  catA.ID,
		AmountCents: 10000,
		Period:      models.BudgetMonthly,
		RollsOver:   true,
		StartDate:   mustDate(2025, 1, 1),
		EndDate:     &endDate,
	}
	b2 := &models.Budget{
		CategoryID:  catB.ID,
		AmountCents: 5000,
		Period:      models.BudgetWeekly,
		RollsOver:   false,
		StartDate:   mustDate(2025, 6, 1),
		// No EndDate
	}

	if err := d.CreateBudget(b1); err != nil {
		t.Fatalf("CreateBudget b1: %v", err)
	}
	if err := d.CreateBudget(b2); err != nil {
		t.Fatalf("CreateBudget b2: %v", err)
	}

	list, err := d.ListBudgets()
	if err != nil {
		t.Fatalf("ListBudgets: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("ListBudgets len = %d, want 2", len(list))
	}

	// b1 — with EndDate
	if list[0].CategoryID != catA.ID || list[0].AmountCents != 10000 || !list[0].RollsOver {
		t.Fatalf("b1 fields mismatch: %+v", list[0])
	}
	if list[0].EndDate == nil {
		t.Fatal("b1.EndDate is nil, want non-nil")
	}
	if list[0].EndDate.Format("2006-01-02") != "2025-12-31" {
		t.Fatalf("b1.EndDate = %v, want 2025-12-31", list[0].EndDate)
	}

	// b2 — without EndDate
	if list[1].CategoryID != catB.ID || list[1].AmountCents != 5000 || list[1].RollsOver {
		t.Fatalf("b2 fields mismatch: %+v", list[1])
	}
	if list[1].EndDate != nil {
		t.Fatalf("b2.EndDate = %v, want nil", list[1].EndDate)
	}
}

func TestUpdateBudget_Persists(t *testing.T) {
	d := newTestDB(t)

	cat := &models.Category{Name: "UpdBudCat", Color: "#ccc", Icon: "tag"}
	if err := d.CreateCategory(cat); err != nil {
		t.Fatalf("CreateCategory: %v", err)
	}

	b := &models.Budget{
		CategoryID:  cat.ID,
		AmountCents: 2000,
		Period:      models.BudgetMonthly,
		RollsOver:   false,
		StartDate:   mustDate(2025, 1, 1),
	}
	if err := d.CreateBudget(b); err != nil {
		t.Fatalf("CreateBudget: %v", err)
	}

	b.AmountCents = 9999
	b.Period = models.BudgetAnnual
	b.RollsOver = true
	b.StartDate = mustDate(2025, 3, 1)
	if err := d.UpdateBudget(b); err != nil {
		t.Fatalf("UpdateBudget: %v", err)
	}

	list, err := d.ListBudgets()
	if err != nil {
		t.Fatalf("ListBudgets: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListBudgets len = %d, want 1", len(list))
	}
	got := list[0]
	if got.AmountCents != 9999 || got.Period != models.BudgetAnnual || !got.RollsOver {
		t.Fatalf("UpdateBudget did not persist: %+v", got)
	}
	if got.StartDate.Format("2006-01-02") != "2025-03-01" {
		t.Fatalf("StartDate = %v, want 2025-03-01", got.StartDate)
	}
}

func TestDeleteBudget_Removes(t *testing.T) {
	d := newTestDB(t)

	cat := &models.Category{Name: "DelBudCat", Color: "#ddd", Icon: "tag"}
	if err := d.CreateCategory(cat); err != nil {
		t.Fatalf("CreateCategory: %v", err)
	}

	b := &models.Budget{
		CategoryID:  cat.ID,
		AmountCents: 1000,
		Period:      models.BudgetMonthly,
		StartDate:   mustDate(2025, 1, 1),
	}
	if err := d.CreateBudget(b); err != nil {
		t.Fatalf("CreateBudget: %v", err)
	}

	if err := d.DeleteBudget(b.ID); err != nil {
		t.Fatalf("DeleteBudget: %v", err)
	}

	list, err := d.ListBudgets()
	if err != nil {
		t.Fatalf("ListBudgets: %v", err)
	}
	for _, got := range list {
		if got.ID == b.ID {
			t.Fatalf("deleted budget id=%d still present in ListBudgets", b.ID)
		}
	}
}

func TestGetBudgetProgress_IgnoresIncomeRows(t *testing.T) {
	d := newTestDB(t)
	accID := insertTestAccount(t, d, "WalletIncome")

	catA := &models.Category{Name: "BudgIncomeCat", Color: "#eee", Icon: "tag", IsIncome: false}
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

	// Income transaction (amount_cents > 0) in the budget window — should be ignored.
	income := &models.Transaction{
		AccountID:   accID,
		Date:        mustDate(2025, 9, 10),
		AmountCents: 5000,
		Payee:       "Salary",
		CategoryID:  i64ptr(catA.ID),
	}
	if err := d.CreateTransaction(income); err != nil {
		t.Fatalf("CreateTransaction income: %v", err)
	}

	budgeted, _, err := d.GetBudgetProgress("2025-09-01", "2025-10-01")
	if err != nil {
		t.Fatalf("GetBudgetProgress: %v", err)
	}
	if len(budgeted) != 1 {
		t.Fatalf("budgeted len = %d, want 1", len(budgeted))
	}
	if budgeted[0].SpentCents != 0 {
		t.Fatalf("SpentCents = %d, want 0 (income rows excluded by amount_cents < 0)", budgeted[0].SpentCents)
	}
}

func TestCreateBudget_WithEndDate_Roundtrips(t *testing.T) {
	d := newTestDB(t)

	cat := &models.Category{Name: "EndDateCat", Color: "#fff", Icon: "tag"}
	if err := d.CreateCategory(cat); err != nil {
		t.Fatalf("CreateCategory: %v", err)
	}

	end := mustDate(2026, 6, 30)
	b := &models.Budget{
		CategoryID:  cat.ID,
		AmountCents: 3000,
		Period:      models.BudgetMonthly,
		StartDate:   mustDate(2025, 7, 1),
		EndDate:     &end,
	}
	if err := d.CreateBudget(b); err != nil {
		t.Fatalf("CreateBudget: %v", err)
	}

	list, err := d.ListBudgets()
	if err != nil {
		t.Fatalf("ListBudgets: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListBudgets len = %d, want 1", len(list))
	}
	if list[0].EndDate == nil {
		t.Fatal("EndDate is nil after roundtrip, want non-nil")
	}
	if list[0].EndDate.Format("2006-01-02") != "2026-06-30" {
		t.Fatalf("EndDate = %v, want 2026-06-30", list[0].EndDate)
	}
}
