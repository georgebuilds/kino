package db

import (
	"database/sql"
	"fmt"

	"kino/internal/models"
)

const UncategorizedCategoryID int64 = 12

func (db *DB) ListCategories() ([]models.Category, error) {
	rows, err := db.Query(`
		SELECT id, name, parent_id, color, icon, is_income, is_system, sort_order
		FROM categories
		ORDER BY sort_order, id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Category
	for rows.Next() {
		c, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (db *DB) GetCategory(id int64) (*models.Category, error) {
	row := db.QueryRow(`
		SELECT id, name, parent_id, color, icon, is_income, is_system, sort_order
		FROM categories WHERE id = ?
	`, id)
	c, err := scanCategory(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (db *DB) CreateCategory(c *models.Category) error {
	res, err := db.Exec(`
		INSERT INTO categories(name, parent_id, color, icon, is_income, is_system, sort_order)
		VALUES (?, ?, ?, ?, ?, 0, ?)
	`, c.Name, nullInt64(c.ParentID), c.Color, c.Icon, c.IsIncome, c.SortOrder)
	if err != nil {
		return fmt.Errorf("create category: %w", err)
	}
	c.ID, _ = res.LastInsertId()
	return nil
}

func (db *DB) UpdateCategory(c *models.Category) error {
	if c.IsSystem {
		return fmt.Errorf("system categories cannot be modified")
	}
	res, err := db.Exec(`
		UPDATE categories SET
			name = ?, parent_id = ?, color = ?, icon = ?,
			is_income = ?, sort_order = ?
		WHERE id = ? AND is_system = 0
	`, c.Name, nullInt64(c.ParentID), c.Color, c.Icon,
		c.IsIncome, c.SortOrder, c.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("category not found or is a system category")
	}
	return nil
}

func (db *DB) DeleteCategory(id int64) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Re-assign transactions to Uncategorized before deleting.
	if _, err = tx.Exec(`UPDATE transactions SET category_id = ? WHERE category_id = ?`, UncategorizedCategoryID, id); err != nil {
		return err
	}

	var res sql.Result
	if res, err = tx.Exec(`DELETE FROM categories WHERE id = ? AND is_system = 0`, id); err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		// Either the category didn't exist or it was a system category.
		var exists bool
		if scanErr := tx.QueryRow(`SELECT 1 FROM categories WHERE id = ?`, id).Scan(&exists); scanErr == sql.ErrNoRows {
			err = fmt.Errorf("category %d not found", id)
		} else {
			err = fmt.Errorf("system categories cannot be deleted")
		}
		return err
	}

	err = tx.Commit()
	return err
}

func (db *DB) CountTransactionsByCategory(id int64) (int64, error) {
	var n int64
	err := db.QueryRow(`SELECT COUNT(*) FROM transactions WHERE category_id = ?`, id).Scan(&n)
	return n, err
}

func scanCategory(s scanner) (models.Category, error) {
	var c models.Category
	var parentID sql.NullInt64

	err := s.Scan(&c.ID, &c.Name, &parentID, &c.Color, &c.Icon,
		&c.IsIncome, &c.IsSystem, &c.SortOrder)
	if err != nil {
		return c, err
	}
	if parentID.Valid {
		c.ParentID = &parentID.Int64
	}
	return c, nil
}

func nullInt64(v *int64) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}
