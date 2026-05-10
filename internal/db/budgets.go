package db

import (
	"database/sql"
	"fmt"
	"time"

	"kino/internal/models"
)

// ── CRUD ──────────────────────────────────────────────────────────────────────

func (db *DB) ListBudgets() ([]models.Budget, error) {
	rows, err := db.Query(`
		SELECT id, category_id, amount_cents, period, rolls_over, start_date, end_date
		FROM budgets
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Budget
	for rows.Next() {
		b, err := scanBudget(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (db *DB) CreateBudget(b *models.Budget) error {
	res, err := db.Exec(`
		INSERT INTO budgets(category_id, amount_cents, period, rolls_over, start_date, end_date)
		VALUES (?, ?, ?, ?, ?, ?)
	`, b.CategoryID, b.AmountCents, b.Period, b.RollsOver,
		b.StartDate.Format("2006-01-02"), nullTimeToStr(b.EndDate))
	if err != nil {
		return fmt.Errorf("create budget: %w", err)
	}
	b.ID, _ = res.LastInsertId()
	return nil
}

func (db *DB) UpdateBudget(b *models.Budget) error {
	_, err := db.Exec(`
		UPDATE budgets SET
			category_id  = ?,
			amount_cents = ?,
			period       = ?,
			rolls_over   = ?,
			start_date   = ?,
			end_date     = ?
		WHERE id = ?
	`, b.CategoryID, b.AmountCents, b.Period, b.RollsOver,
		b.StartDate.Format("2006-01-02"), nullTimeToStr(b.EndDate), b.ID)
	return err
}

func (db *DB) DeleteBudget(id int64) error {
	_, err := db.Exec(`DELETE FROM budgets WHERE id = ?`, id)
	return err
}

// ── Progress query ────────────────────────────────────────────────────────────

// BudgetWithSpend is the raw result of the progress query — both budget and
// actual spending for a given period. The App layer converts it to BudgetLine.
type BudgetWithSpend struct {
	Budget      models.Budget
	CategoryName  string
	CategoryColor string
	CategoryIcon  string
	SpentCents    int64
}

// UnbudgetedSpend is a category that has expense transactions this period but
// no matching budget row.
type UnbudgetedSpend struct {
	CategoryID    int64
	CategoryName  string
	CategoryColor string
	CategoryIcon  string
	SpentCents    int64
}

// GetBudgetProgress returns all budgets joined with their month's spending,
// plus categories that have unbudgeted spending.
func (db *DB) GetBudgetProgress(dateFrom, dateTo string) ([]BudgetWithSpend, []UnbudgetedSpend, error) {
	// Budgeted rows with actual spend
	rows, err := db.Query(`
		SELECT
			b.id, b.category_id, b.amount_cents, b.period, b.rolls_over,
			b.start_date, b.end_date,
			c.name, c.color, c.icon,
			COALESCE(SUM(CASE WHEN t.amount_cents < 0 THEN -t.amount_cents ELSE 0 END), 0) AS spent
		FROM budgets b
		JOIN categories c ON c.id = b.category_id
		LEFT JOIN transactions t
			ON  t.category_id = b.category_id
			AND t.date >= ?
			AND t.date <  ?
			AND t.amount_cents < 0
			AND t.is_transfer = 0
		GROUP BY b.id
		ORDER BY c.sort_order, c.name
	`, dateFrom, dateTo)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var budgeted []BudgetWithSpend
	for rows.Next() {
		var bws BudgetWithSpend
		var startDate, endDate sql.NullString

		err := rows.Scan(
			&bws.Budget.ID, &bws.Budget.CategoryID, &bws.Budget.AmountCents,
			&bws.Budget.Period, &bws.Budget.RollsOver,
			&startDate, &endDate,
			&bws.CategoryName, &bws.CategoryColor, &bws.CategoryIcon,
			&bws.SpentCents,
		)
		if err != nil {
			return nil, nil, err
		}
		if startDate.Valid {
			bws.Budget.StartDate = parseTime(startDate.String)
		}
		if endDate.Valid {
			t := parseTime(endDate.String)
			bws.Budget.EndDate = &t
		}
		budgeted = append(budgeted, bws)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	// Unbudgeted spending
	urows, err := db.Query(`
		SELECT
			c.id, c.name, c.color, c.icon,
			COALESCE(SUM(-t.amount_cents), 0) AS spent
		FROM transactions t
		JOIN categories c ON c.id = t.category_id
		WHERE t.date >= ?
		  AND t.date <  ?
		  AND t.amount_cents < 0
		  AND t.is_transfer = 0
		  AND c.is_income   = 0
		  AND c.id NOT IN (SELECT category_id FROM budgets)
		GROUP BY c.id
		HAVING spent > 0
		ORDER BY spent DESC
	`, dateFrom, dateTo)
	if err != nil {
		return nil, nil, err
	}
	defer urows.Close()

	var unbudgeted []UnbudgetedSpend
	for urows.Next() {
		var u UnbudgetedSpend
		if err := urows.Scan(&u.CategoryID, &u.CategoryName, &u.CategoryColor, &u.CategoryIcon, &u.SpentCents); err != nil {
			return nil, nil, err
		}
		unbudgeted = append(unbudgeted, u)
	}
	return budgeted, unbudgeted, urows.Err()
}

// ── Scan helpers ──────────────────────────────────────────────────────────────

func scanBudget(s scanner) (models.Budget, error) {
	var b models.Budget
	var startDate, endDate sql.NullString

	err := s.Scan(
		&b.ID, &b.CategoryID, &b.AmountCents, &b.Period, &b.RollsOver,
		&startDate, &endDate,
	)
	if err != nil {
		return b, err
	}
	if startDate.Valid {
		b.StartDate = parseTime(startDate.String)
	}
	if endDate.Valid {
		t := parseTime(endDate.String)
		b.EndDate = &t
	}
	return b, nil
}

// nullTimeToStr converts a nullable *time.Time to an interface{} suitable for
// SQLite TEXT or NULL insertion.
func nullTimeToStr(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format("2006-01-02")
}
