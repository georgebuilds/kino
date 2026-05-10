package main

import "fmt"

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
	// Last day: first day of next month minus one day via SQLite date math.
	to := fmt.Sprintf("%04d-%02d-01", year, month+1) // SQLite handles month=13 → next year

	var s MonthSummary

	// Net worth = sum of all account balances.
	_ = a.db.QueryRow(
		`SELECT COALESCE(SUM(balance_cents),0) FROM accounts WHERE is_hidden = 0`,
	).Scan(&s.NetWorthCents)

	// Net worth last month (approximation: NW now minus this month's net flow).
	var monthNet int64
	_ = a.db.QueryRow(
		`SELECT COALESCE(SUM(amount_cents),0) FROM transactions WHERE date >= ? AND date < ?`,
		from, to,
	).Scan(&monthNet)
	s.NetWorthDeltaCents = monthNet

	// Income: positive transactions this month (excluding transfers).
	_ = a.db.QueryRow(`
		SELECT COALESCE(SUM(amount_cents),0) FROM transactions
		WHERE date >= ? AND date < ? AND amount_cents > 0 AND is_transfer = 0
	`, from, to).Scan(&s.IncomeCents)

	// Expenses: negative transactions this month (excluding transfers), as positive number.
	_ = a.db.QueryRow(`
		SELECT COALESCE(SUM(amount_cents),0) FROM transactions
		WHERE date >= ? AND date < ? AND amount_cents < 0 AND is_transfer = 0
	`, from, to).Scan(&s.ExpenseCents)
	s.ExpenseCents = -s.ExpenseCents // make positive for display
	s.SavedCents = s.IncomeCents - s.ExpenseCents

	// Category breakdown for expense transactions.
	rows, err := a.db.Query(`
		SELECT c.id, c.name, c.color, ABS(SUM(t.amount_cents)) as total
		FROM transactions t
		JOIN categories c ON c.id = t.category_id
		WHERE t.date >= ? AND t.date < ?
		  AND t.amount_cents < 0
		  AND t.is_transfer = 0
		GROUP BY c.id
		ORDER BY total DESC
		LIMIT 12
	`, from, to)
	if err != nil {
		return s, err
	}
	defer rows.Close()

	for rows.Next() {
		var ct CatTotal
		if err := rows.Scan(&ct.CategoryID, &ct.CategoryName, &ct.Color, &ct.AmountCents); err != nil {
			continue
		}
		s.CategoryTotals = append(s.CategoryTotals, ct)
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
