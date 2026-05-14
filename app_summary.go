package main

import (
	"fmt"

	"kino/internal/db"
)

// MonthSummary is everything the Overview page needs in one query round-trip.
type MonthSummary struct {
	NetWorthCents      int64      `json:"netWorthCents"`
	NetWorthDeltaCents int64      `json:"netWorthDeltaCents"`
	IncomeCents        int64      `json:"incomeCents"`
	ExpenseCents       int64      `json:"expenseCents"`
	SavedCents         int64      `json:"savedCents"`
	TopCategory        string     `json:"topCategory"`
	TopCategoryCents   int64      `json:"topCategoryCents"`
	CategoryTotals     []CatTotal `json:"categoryTotals"`
}

type CatTotal struct {
	CategoryID   int64  `json:"categoryId"`
	CategoryName string `json:"categoryName"`
	Color        string `json:"color"`
	AmountCents  int64  `json:"amountCents"`
}

// GetMonthSummary returns aggregated data for a given month (1-indexed).
func (a *App) GetMonthSummary(year, month int) (MonthSummary, error) {
	if err := a.requireDB(); err != nil {
		return MonthSummary{}, err
	}

	// Date range for the requested month.
	from := fmt.Sprintf("%04d-%02d-01", year, month)
	ny, nm := year, month+1
	if nm > 12 {
		nm = 1
		ny++
	}
	to := fmt.Sprintf("%04d-%02d-01", ny, nm)

	var s MonthSummary

	// Net worth = sum of all account balances.
	if err := a.db.QueryRow(
		`SELECT COALESCE(SUM(balance_cents),0) FROM accounts WHERE is_hidden = 0`,
	).Scan(&s.NetWorthCents); err != nil {
		return MonthSummary{}, err
	}

	var expenseSigned int64
	if err := a.db.QueryRow(`
		SELECT
			COALESCE(SUM(CASE WHEN is_transfer = 0 THEN amount_cents END), 0)                     AS month_net,
			COALESCE(SUM(CASE WHEN amount_cents > 0 AND is_transfer = 0 THEN amount_cents END), 0) AS income,
			COALESCE(SUM(CASE WHEN amount_cents < 0 AND is_transfer = 0 THEN amount_cents END), 0) AS expense_signed
		FROM transactions
		WHERE date >= ? AND date < ?
	`, from, to).Scan(&s.NetWorthDeltaCents, &s.IncomeCents, &expenseSigned); err != nil {
		return MonthSummary{}, err
	}
	s.ExpenseCents = -expenseSigned
	s.SavedCents = s.IncomeCents - s.ExpenseCents

	// Category breakdown for expense transactions.
	// NULL-category rows are bucketed into the seeded "Uncategorized" category
	// so the breakdown reconciles with ExpenseCents.
	rows, err := a.db.Query(`
		SELECT c.id, c.name, c.color, ABS(SUM(t.amount_cents)) as total
		FROM transactions t
		JOIN categories c ON c.id = COALESCE(t.category_id, ?)
		WHERE t.date >= ? AND t.date < ?
		  AND t.amount_cents < 0
		  AND t.is_transfer = 0
		GROUP BY c.id
		ORDER BY total DESC
		LIMIT 12
	`, db.UncategorizedCategoryID, from, to)
	if err != nil {
		return s, err
	}
	defer rows.Close()

	for rows.Next() {
		var ct CatTotal
		if err := rows.Scan(&ct.CategoryID, &ct.CategoryName, &ct.Color, &ct.AmountCents); err != nil {
			return MonthSummary{}, err
		}
		s.CategoryTotals = append(s.CategoryTotals, ct)
	}
	if err := rows.Err(); err != nil {
		return MonthSummary{}, err
	}
	if len(s.CategoryTotals) > 0 {
		s.TopCategory = s.CategoryTotals[0].CategoryName
		s.TopCategoryCents = s.CategoryTotals[0].AmountCents
	}
	if s.CategoryTotals == nil {
		s.CategoryTotals = []CatTotal{}
	}

	return s, nil
}
