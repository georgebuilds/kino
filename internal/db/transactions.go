package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"kino/internal/models"
)

// TxFilter controls which transactions are returned and how many.
type TxFilter struct {
	AccountID  *int64 `json:"accountId"`
	CategoryID *int64 `json:"categoryId"`
	DateFrom   string `json:"dateFrom"` // YYYY-MM-DD, empty = no limit
	DateTo     string `json:"dateTo"`   // YYYY-MM-DD, empty = no limit
	Search     string `json:"search"`
	Limit      int    `json:"limit"`  // 0 → default 100
	Offset     int    `json:"offset"`
}

// TxPage is the paginated result returned to the frontend.
type TxPage struct {
	Transactions []models.Transaction `json:"transactions"`
	Total        int                  `json:"total"`
}

func (db *DB) ListTransactions(f TxFilter) (TxPage, error) {
	conds, args := buildTxWhere(f)
	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM transactions `+where, args...,
	).Scan(&total); err != nil {
		return TxPage{}, err
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 100
	}

	rows, err := db.Query(
		`SELECT id, account_id, date, amount_cents, payee, payee_normalized, notes,
		        category_id, is_transfer, transfer_pair_id, is_reconciled,
		        import_hash, import_source, created_at, updated_at
		 FROM transactions `+where+
			` ORDER BY date DESC, id DESC LIMIT ? OFFSET ?`,
		append(args, limit, f.Offset)...,
	)
	if err != nil {
		return TxPage{}, err
	}
	defer rows.Close()

	var txs []models.Transaction
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
			return TxPage{}, err
		}
		txs = append(txs, t)
	}
	if err := rows.Err(); err != nil {
		return TxPage{}, err
	}
	if txs == nil {
		txs = []models.Transaction{}
	}
	return TxPage{Transactions: txs, Total: total}, nil
}

func (db *DB) GetTransaction(id int64) (*models.Transaction, error) {
	row := db.QueryRow(`
		SELECT id, account_id, date, amount_cents, payee, payee_normalized, notes,
		       category_id, is_transfer, transfer_pair_id, is_reconciled,
		       import_hash, import_source, created_at, updated_at
		FROM transactions WHERE id = ?
	`, id)
	t, err := scanTransaction(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (db *DB) CreateTransaction(t *models.Transaction) error {
	if t.PayeeNormalized == "" {
		t.PayeeNormalized = t.Payee
	}
	res, err := db.Exec(`
		INSERT INTO transactions(
			account_id, date, amount_cents, payee, payee_normalized, notes,
			category_id, is_transfer, transfer_pair_id, is_reconciled,
			import_hash, import_source
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
	`,
		t.AccountID,
		t.Date.Format("2006-01-02"),
		t.AmountCents,
		t.Payee,
		t.PayeeNormalized,
		t.Notes,
		nullInt64(t.CategoryID),
		t.IsTransfer,
		nullInt64(t.TransferPairID),
		t.IsReconciled,
		t.ImportHash,
		t.ImportSource,
	)
	if err != nil {
		return fmt.Errorf("create transaction: %w", err)
	}
	t.ID, _ = res.LastInsertId()
	t.CreatedAt = time.Now().UTC()
	t.UpdatedAt = t.CreatedAt
	return nil
}

func (db *DB) UpdateTransaction(t *models.Transaction) error {
	_, err := db.Exec(`
		UPDATE transactions SET
			account_id = ?, date = ?, amount_cents = ?, payee = ?,
			payee_normalized = ?, notes = ?, category_id = ?,
			is_transfer = ?, is_reconciled = ?,
			updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE id = ?
	`,
		t.AccountID,
		t.Date.Format("2006-01-02"),
		t.AmountCents,
		t.Payee,
		t.PayeeNormalized,
		t.Notes,
		nullInt64(t.CategoryID),
		t.IsTransfer,
		t.IsReconciled,
		t.ID,
	)
	return err
}

func (db *DB) DeleteTransaction(id int64) error {
	_, err := db.Exec(`DELETE FROM transactions WHERE id = ?`, id)
	return err
}

// BulkInsert inserts a slice of transactions, skipping exact hash duplicates.
// Returns counts of newly inserted rows, exact-hash skips, and the IDs of
// every newly inserted row (used by FindFuzzyDuplicates).
func (db *DB) BulkInsert(txs []models.Transaction) (inserted, dupes int, newIDs []int64, err error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, 0, nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO transactions(
			account_id, date, amount_cents, payee, payee_normalized, notes,
			category_id, is_transfer, import_hash, import_source
		) VALUES (?,?,?,?,?,?,?,?,?,?)
	`)
	if err != nil {
		return 0, 0, nil, err
	}
	defer stmt.Close()

	for i := range txs {
		t := &txs[i]
		if t.PayeeNormalized == "" {
			t.PayeeNormalized = t.Payee
		}
		res, execErr := stmt.Exec(
			t.AccountID,
			t.Date.Format("2006-01-02"),
			t.AmountCents,
			t.Payee,
			t.PayeeNormalized,
			t.Notes,
			nullInt64(t.CategoryID),
			t.IsTransfer,
			t.ImportHash,
			t.ImportSource,
		)
		if execErr != nil {
			err = fmt.Errorf("bulk insert row %d: %w", i, execErr)
			return 0, 0, nil, err
		}
		n, _ := res.RowsAffected()
		if n > 0 {
			inserted++
			id, _ := res.LastInsertId()
			newIDs = append(newIDs, id)
		} else {
			dupes++
		}
	}

	err = tx.Commit()
	return inserted, dupes, newIDs, err
}

// PossibleDupe pairs a newly inserted transaction with an existing one that
// has the same account / date / amount but a different import source.
type PossibleDupe struct {
	NewTx      models.Transaction `json:"newTx"`
	ExistingTx models.Transaction `json:"existingTx"`
}

// FindFuzzyDuplicates scans the newly inserted IDs for transactions that
// share (account_id, date, amount_cents) with an older row from a different
// source.  It only considers rows inserted before the current batch (id not
// in newIDs) to avoid pairing two rows from the same import against each other.
func (db *DB) FindFuzzyDuplicates(newIDs []int64) ([]PossibleDupe, error) {
	if len(newIDs) == 0 {
		return nil, nil
	}

	// Build a placeholder list  "?,?,?…"
	ph := make([]string, len(newIDs))
	args := make([]any, len(newIDs))
	for i, id := range newIDs {
		ph[i] = "?"
		args[i] = id
	}
	placeholders := strings.Join(ph, ",")

	query := fmt.Sprintf(`
		SELECT
			n.id, n.account_id, n.date, n.amount_cents, n.payee, n.payee_normalized,
			n.notes, n.category_id, n.is_transfer, n.transfer_pair_id, n.is_reconciled,
			n.import_hash, n.import_source, n.created_at, n.updated_at,
			o.id, o.account_id, o.date, o.amount_cents, o.payee, o.payee_normalized,
			o.notes, o.category_id, o.is_transfer, o.transfer_pair_id, o.is_reconciled,
			o.import_hash, o.import_source, o.created_at, o.updated_at
		FROM transactions n
		JOIN transactions o
			ON  o.account_id   = n.account_id
			AND o.date         = n.date
			AND o.amount_cents = n.amount_cents
			AND o.id           != n.id
			AND o.id           NOT IN (%s)
		WHERE n.id IN (%s)
		GROUP BY n.id, o.id
		ORDER BY n.id, o.id
	`, placeholders, placeholders)

	// args appears twice (once for NOT IN, once for WHERE IN)
	fullArgs := append(args, args...)
	rows, err := db.Query(query, fullArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seen := map[[2]int64]bool{}
	var out []PossibleDupe

	for rows.Next() {
		var newTx, existingTx models.Transaction
		var nDate, nCreated, nUpdated string
		var nCatID, nTransferPairID sql.NullInt64
		var oDate, oCreated, oUpdated string
		var oCatID, oTransferPairID sql.NullInt64

		err := rows.Scan(
			&newTx.ID, &newTx.AccountID, &nDate, &newTx.AmountCents,
			&newTx.Payee, &newTx.PayeeNormalized, &newTx.Notes,
			&nCatID, &newTx.IsTransfer, &nTransferPairID, &newTx.IsReconciled,
			&newTx.ImportHash, &newTx.ImportSource, &nCreated, &nUpdated,
			&existingTx.ID, &existingTx.AccountID, &oDate, &existingTx.AmountCents,
			&existingTx.Payee, &existingTx.PayeeNormalized, &existingTx.Notes,
			&oCatID, &existingTx.IsTransfer, &oTransferPairID, &existingTx.IsReconciled,
			&existingTx.ImportHash, &existingTx.ImportSource, &oCreated, &oUpdated,
		)
		if err != nil {
			return nil, err
		}

		key := [2]int64{newTx.ID, existingTx.ID}
		if seen[key] {
			continue
		}
		seen[key] = true

		newTx.Date = parseTime(nDate)
		newTx.CreatedAt = parseTime(nCreated)
		newTx.UpdatedAt = parseTime(nUpdated)
		if nCatID.Valid {
			newTx.CategoryID = &nCatID.Int64
		}
		if nTransferPairID.Valid {
			newTx.TransferPairID = &nTransferPairID.Int64
		}

		existingTx.Date = parseTime(oDate)
		existingTx.CreatedAt = parseTime(oCreated)
		existingTx.UpdatedAt = parseTime(oUpdated)
		if oCatID.Valid {
			existingTx.CategoryID = &oCatID.Int64
		}
		if oTransferPairID.Valid {
			existingTx.TransferPairID = &oTransferPairID.Int64
		}

		out = append(out, PossibleDupe{NewTx: newTx, ExistingTx: existingTx})
	}
	return out, rows.Err()
}

// MergeTransaction keeps keepID, copies deleteID's import_hash onto it (so
// future imports with that hash are also silently skipped), then deletes
// deleteID.  Category and notes from deleteID are preserved if keepID lacks them.
func (db *DB) MergeTransaction(keepID, deleteID int64) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Fetch both rows
	var keepHash, deleteHash string
	var keepCatID sql.NullInt64
	var keepNotes string
	if err = tx.QueryRow(
		`SELECT import_hash, category_id, notes FROM transactions WHERE id = ?`, keepID,
	).Scan(&keepHash, &keepCatID, &keepNotes); err != nil {
		return fmt.Errorf("keep row: %w", err)
	}

	var deleteCatID sql.NullInt64
	var deleteNotes string
	if err = tx.QueryRow(
		`SELECT import_hash, category_id, notes FROM transactions WHERE id = ?`, deleteID,
	).Scan(&deleteHash, &deleteCatID, &deleteNotes); err != nil {
		return fmt.Errorf("delete row: %w", err)
	}

	// Merge: adopt the delete row's hash so future imports skip it.
	// Also adopt category/notes from delete row if keep row is missing them.
	newCatID := keepCatID
	if !newCatID.Valid && deleteCatID.Valid {
		newCatID = deleteCatID
	}
	newNotes := keepNotes
	if newNotes == "" && deleteNotes != "" {
		newNotes = deleteNotes
	}

	// The OFX hash should win: it's bank-stable and will be used in future syncs.
	// If neither is OFX (e.g. two CSV imports), keep the existing hash.
	var keepSource, deleteSource string
	_ = tx.QueryRow(`SELECT import_source FROM transactions WHERE id = ?`, keepID).Scan(&keepSource)
	_ = tx.QueryRow(`SELECT import_source FROM transactions WHERE id = ?`, deleteID).Scan(&deleteSource)

	winningHash := keepHash
	if deleteSource == "ofx" && keepSource != "ofx" {
		winningHash = deleteHash
	} else if keepHash == "" && deleteHash != "" {
		winningHash = deleteHash
	}

	if _, err = tx.Exec(`
		UPDATE transactions SET
			import_hash = ?,
			category_id = ?,
			notes       = ?,
			updated_at  = strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE id = ?
	`, winningHash, newCatID, newNotes, keepID); err != nil {
		return err
	}

	if _, err = tx.Exec(`DELETE FROM transactions WHERE id = ?`, deleteID); err != nil {
		return err
	}

	return tx.Commit()
}


// ─── helpers ─────────────────────────────────────────────────────────────────

func buildTxWhere(f TxFilter) ([]string, []any) {
	conds := []string{"1=1"}
	var args []any

	if f.AccountID != nil {
		conds = append(conds, "account_id = ?")
		args = append(args, *f.AccountID)
	}
	if f.CategoryID != nil {
		conds = append(conds, "category_id = ?")
		args = append(args, *f.CategoryID)
	}
	if f.DateFrom != "" {
		conds = append(conds, "date >= ?")
		args = append(args, f.DateFrom)
	}
	if f.DateTo != "" {
		conds = append(conds, "date <= ?")
		args = append(args, f.DateTo)
	}
	if f.Search != "" {
		conds = append(conds, "(payee LIKE ? OR payee_normalized LIKE ? OR notes LIKE ?)")
		q := "%" + f.Search + "%"
		args = append(args, q, q, q)
	}
	return conds, args
}

func scanTransaction(s scanner) (models.Transaction, error) {
	var t models.Transaction
	var dateStr, createdAt, updatedAt string
	var categoryID, transferPairID sql.NullInt64

	err := s.Scan(
		&t.ID, &t.AccountID, &dateStr, &t.AmountCents,
		&t.Payee, &t.PayeeNormalized, &t.Notes,
		&categoryID, &t.IsTransfer, &transferPairID, &t.IsReconciled,
		&t.ImportHash, &t.ImportSource,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return t, err
	}

	t.Date = parseTime(dateStr)
	t.CreatedAt = parseTime(createdAt)
	t.UpdatedAt = parseTime(updatedAt)

	if categoryID.Valid {
		t.CategoryID = &categoryID.Int64
	}
	if transferPairID.Valid {
		t.TransferPairID = &transferPairID.Int64
	}
	return t, nil
}
