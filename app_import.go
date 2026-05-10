package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"kino/internal/db"
	"kino/internal/importer"
	"kino/internal/models"
)

// ── View models ───────────────────────────────────────────────────────────────

// ImportResult is returned to the frontend after every import operation.
type ImportResult struct {
	Inserted      int              `json:"inserted"`
	Skipped       int              `json:"skipped"`       // exact-hash duplicates
	PossibleDupes []db.PossibleDupe `json:"possibleDupes"` // fuzzy matches for review
	FileName      string           `json:"fileName"`
	Source        string           `json:"source"` // "csv" or "ofx"
}

// DupeAction is the user's decision about a possible duplicate pair.
type DupeAction string

const (
	DupeKeepBoth  DupeAction = "keep_both"   // leave both rows as-is
	DupeDeleteNew DupeAction = "delete_new"   // discard the newly imported row
	DupeMerge     DupeAction = "merge"        // keep existing, adopt new hash + fields
)

// ── File-picker + import ──────────────────────────────────────────────────────

// PickAndImportCSV opens the native file picker filtered to CSV files,
// parses the chosen file, and bulk-inserts into the given account.
func (a *App) PickAndImportCSV(accountID int64) (ImportResult, error) {
	if err := a.requireDB(); err != nil {
		return ImportResult{}, err
	}

	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Import CSV",
		Filters: []runtime.FileFilter{
			{DisplayName: "CSV files (*.csv)", Pattern: "*.csv"},
		},
	})
	if err != nil || path == "" {
		return ImportResult{}, err // user cancelled → no error, empty result
	}

	return a.importCSVFromPath(accountID, path)
}

// PickAndImportOFX opens the native file picker filtered to OFX/QFX files.
func (a *App) PickAndImportOFX(accountID int64) (ImportResult, error) {
	if err := a.requireDB(); err != nil {
		return ImportResult{}, err
	}

	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Import OFX / QFX",
		Filters: []runtime.FileFilter{
			{DisplayName: "OFX / QFX files (*.ofx, *.qfx)", Pattern: "*.ofx;*.qfx"},
		},
	})
	if err != nil || path == "" {
		return ImportResult{}, err
	}

	return a.importOFXFromPath(accountID, path)
}

// ── Internal import helpers ───────────────────────────────────────────────────

func (a *App) importCSVFromPath(accountID int64, path string) (ImportResult, error) {
	f, err := openOSFile(path)
	if err != nil {
		return ImportResult{}, err
	}
	defer f.Close()

	rows, err := importer.ParseCSV(f, accountID)
	if err != nil {
		return ImportResult{}, fmt.Errorf("parse CSV: %w", err)
	}

	return a.bulkImport(accountID, rows, "csv", filepath.Base(path))
}

func (a *App) importOFXFromPath(accountID int64, path string) (ImportResult, error) {
	f, err := openOSFile(path)
	if err != nil {
		return ImportResult{}, err
	}
	defer f.Close()

	rows, _, err := importer.ParseOFX(f, accountID)
	if err != nil {
		return ImportResult{}, fmt.Errorf("parse OFX: %w", err)
	}

	return a.bulkImport(accountID, rows, "ofx", filepath.Base(path))
}

func (a *App) bulkImport(accountID int64, rows []importer.Row, source, fileName string) (ImportResult, error) {
	if len(rows) == 0 {
		return ImportResult{FileName: fileName, Source: source}, nil
	}

	// Convert importer.Row → models.Transaction
	today := time.Now().UTC()
	txs := make([]models.Transaction, 0, len(rows))
	for _, r := range rows {
		t, err := time.Parse("2006-01-02", r.Date)
		if err != nil {
			continue
		}
		txs = append(txs, models.Transaction{
			AccountID:       accountID,
			Date:            t,
			AmountCents:     r.AmountCents,
			Payee:           r.Payee,
			PayeeNormalized: r.PayeeNormalized,
			Notes:           r.Notes,
			ImportHash:      r.ImportHash,
			ImportSource:    r.ImportSource,
			CreatedAt:       today,
			UpdatedAt:       today,
		})
	}

	inserted, skipped, newIDs, err := a.db.BulkInsert(txs)
	if err != nil {
		return ImportResult{}, fmt.Errorf("import: %w", err)
	}

	// Recalculate account balance after insert
	if inserted > 0 {
		_ = a.db.RecalcBalance(accountID)
	}

	// Fuzzy duplicate check
	dupes, err := a.db.FindFuzzyDuplicates(newIDs)
	if err != nil {
		// Non-fatal — import already succeeded
		dupes = nil
	}

	return ImportResult{
		Inserted:      inserted,
		Skipped:       skipped,
		PossibleDupes: dupes,
		FileName:      fileName,
		Source:        source,
	}, nil
}

// ── Duplicate resolution ──────────────────────────────────────────────────────

// ResolveDuplicate applies the user's decision about a possible duplicate pair.
//
//   - keep_both: do nothing (both rows stay)
//   - delete_new: delete the newly imported transaction
//   - merge: keep the existing row, adopt the new row's hash (so future imports
//     skip it), preserve the best category/notes, delete the new row
func (a *App) ResolveDuplicate(action string, keepID, deleteID int64) error {
	if err := a.requireDB(); err != nil {
		return err
	}

	switch DupeAction(strings.ToLower(action)) {
	case DupeKeepBoth:
		return nil

	case DupeDeleteNew:
		if err := a.db.DeleteTransaction(deleteID); err != nil {
			return err
		}
		// Recalc for the account the deleted row belonged to
		if tx, err := a.db.GetTransaction(keepID); err == nil && tx != nil {
			_ = a.db.RecalcBalance(tx.AccountID)
		}
		return nil

	case DupeMerge:
		return a.db.MergeTransaction(keepID, deleteID)

	default:
		return fmt.Errorf("unknown action %q (want: keep_both, delete_new, merge)", action)
	}
}

func openOSFile(path string) (*os.File, error) {
	return os.Open(path)
}
