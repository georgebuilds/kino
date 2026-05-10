package main

import (
	"fmt"
	"time"

	"kino/internal/db"
	"kino/internal/models"
)

// ── View models ───────────────────────────────────────────────────────────────

// BudgetLine is the display model for one budget row: budget target + actual spend.
type BudgetLine struct {
	ID            int64  `json:"id"`
	CategoryID    int64  `json:"categoryId"`
	CategoryName  string `json:"categoryName"`
	CategoryColor string `json:"categoryColor"`
	CategoryIcon  string `json:"categoryIcon"`
	BudgetCents   int64  `json:"budgetCents"`
	SpentCents    int64  `json:"spentCents"`
	Period        string `json:"period"`
	RollsOver     bool   `json:"rollsOver"`
}

// UnbudgetedLine is a category with expense spending but no budget this period.
type UnbudgetedLine struct {
	CategoryID    int64  `json:"categoryId"`
	CategoryName  string `json:"categoryName"`
	CategoryColor string `json:"categoryColor"`
	CategoryIcon  string `json:"categoryIcon"`
	SpentCents    int64  `json:"spentCents"`
}

// BudgetPage is the full response for the budgets view.
type BudgetPage struct {
	Lines            []BudgetLine     `json:"lines"`
	Unbudgeted       []UnbudgetedLine `json:"unbudgeted"`
	TotalBudgetCents int64            `json:"totalBudgetCents"`
	TotalSpentCents  int64            `json:"totalSpentCents"`
}

// ── Wails-exposed methods ─────────────────────────────────────────────────────

// GetBudgetPage returns budget lines with actual spend for the given month,
// plus any categories that have spending but no budget this period.
func (a *App) GetBudgetPage(year, month int) (BudgetPage, error) {
	if err := a.requireDB(); err != nil {
		return BudgetPage{}, err
	}

	// Build date range strings for the month
	dateFrom := fmt.Sprintf("%04d-%02d-01", year, month)
	nextYear, nextMonth := year, month+1
	if nextMonth > 12 {
		nextMonth = 1
		nextYear++
	}
	dateTo := fmt.Sprintf("%04d-%02d-01", nextYear, nextMonth)

	budgeted, unbudgeted, err := a.db.GetBudgetProgress(dateFrom, dateTo)
	if err != nil {
		return BudgetPage{}, err
	}

	var lines []BudgetLine
	var totalBudget, totalSpent int64

	for _, b := range budgeted {
		lines = append(lines, BudgetLine{
			ID:            b.Budget.ID,
			CategoryID:    b.Budget.CategoryID,
			CategoryName:  b.CategoryName,
			CategoryColor: b.CategoryColor,
			CategoryIcon:  b.CategoryIcon,
			BudgetCents:   b.Budget.AmountCents,
			SpentCents:    b.SpentCents,
			Period:        string(b.Budget.Period),
			RollsOver:     b.Budget.RollsOver,
		})
		totalBudget += b.Budget.AmountCents
		totalSpent += b.SpentCents
	}

	var ulines []UnbudgetedLine
	for _, u := range unbudgeted {
		ulines = append(ulines, UnbudgetedLine{
			CategoryID:    u.CategoryID,
			CategoryName:  u.CategoryName,
			CategoryColor: u.CategoryColor,
			CategoryIcon:  u.CategoryIcon,
			SpentCents:    u.SpentCents,
		})
		totalSpent += u.SpentCents
	}

	return BudgetPage{
		Lines:            lines,
		Unbudgeted:       ulines,
		TotalBudgetCents: totalBudget,
		TotalSpentCents:  totalSpent,
	}, nil
}

// CreateBudget creates a new budget. StartDate defaults to today if zero.
func (a *App) CreateBudget(b models.Budget) (models.Budget, error) {
	if err := a.requireDB(); err != nil {
		return b, err
	}
	if b.StartDate.IsZero() {
		b.StartDate = time.Now()
	}
	if err := a.db.CreateBudget(&b); err != nil {
		return b, err
	}
	return b, nil
}

// UpdateBudget updates an existing budget row.
func (a *App) UpdateBudget(b models.Budget) error {
	if err := a.requireDB(); err != nil {
		return err
	}
	return a.db.UpdateBudget(&b)
}

// DeleteBudget removes a budget by ID.
func (a *App) DeleteBudget(id int64) error {
	if err := a.requireDB(); err != nil {
		return err
	}
	return a.db.DeleteBudget(id)
}

// ListBudgets returns the raw budget rows (used by the modal to detect duplicates).
func (a *App) ListBudgets() ([]models.Budget, error) {
	if err := a.requireDB(); err != nil {
		return nil, err
	}
	return a.db.ListBudgets()
}

// dbBudget is just to avoid an unused import — db is used above.
var _ = db.BudgetWithSpend{}
