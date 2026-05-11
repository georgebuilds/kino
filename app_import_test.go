package main

import (
	"strings"
	"testing"

	"kino/internal/importer"
	"kino/internal/models"
)

// makeRow returns a valid importer.Row with a proper YYYY-MM-DD date.
func makeRow(date string, amountCents int64, accountID int64, suffix string) importer.Row {
	normalized := importer.NormalizePayee("TestPayee " + suffix)
	return importer.Row{
		Date:            date,
		AmountCents:     amountCents,
		Payee:           "TestPayee " + suffix,
		PayeeNormalized: normalized,
		ImportHash:      importer.HashCSV(accountID, date, amountCents, normalized),
		ImportSource:    "csv",
	}
}

func TestBulkImport_EmptyRows_NoOp(t *testing.T) {
	a := newTestApp(t)
	accID := insertTestAccount(t, a, "Checking")

	res, err := a.bulkImport(accID, []importer.Row{}, []string{"warn1"}, "csv", "f.csv")
	if err != nil {
		t.Fatalf("bulkImport empty: %v", err)
	}
	if res.Inserted != 0 {
		t.Errorf("Inserted = %d, want 0", res.Inserted)
	}
	if res.FileName != "f.csv" {
		t.Errorf("FileName = %q, want f.csv", res.FileName)
	}
	if res.Source != "csv" {
		t.Errorf("Source = %q, want csv", res.Source)
	}
	if len(res.Warnings) != 1 || res.Warnings[0] != "warn1" {
		t.Errorf("Warnings = %v, want [warn1]", res.Warnings)
	}
}

func TestBulkImport_SkipsRowsWithBadDate(t *testing.T) {
	a := newTestApp(t)
	accID := insertTestAccount(t, a, "Checking")

	rows := []importer.Row{
		makeRow("2025-03-10", -500, accID, "a"),
		{Date: "not-a-date", AmountCents: -100, Payee: "Bad", ImportHash: "badhash", ImportSource: "csv"},
		makeRow("2025-03-11", -200, accID, "b"),
	}

	res, err := a.bulkImport(accID, rows, nil, "csv", "test.csv")
	if err != nil {
		t.Fatalf("bulkImport: %v", err)
	}
	// The bad-date row is silently skipped (continue at app_import.go:122)
	if res.Inserted != 2 {
		t.Errorf("Inserted = %d, want 2 (bad-date row skipped)", res.Inserted)
	}
}

func TestBulkImport_RecalcsBalance_OnInsert(t *testing.T) {
	a := newTestApp(t)
	accID := insertTestAccount(t, a, "Checking")

	rows := []importer.Row{makeRow("2025-05-01", 1234, accID, "x")}
	if _, err := a.bulkImport(accID, rows, nil, "csv", "test.csv"); err != nil {
		t.Fatalf("bulkImport: %v", err)
	}

	acc, err := a.db.GetAccount(accID)
	if err != nil || acc == nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if acc.BalanceCents != 1234 {
		t.Errorf("BalanceCents = %d, want 1234 (RecalcBalance should have fired)", acc.BalanceCents)
	}
}

func TestBulkImport_FindsFuzzyDuplicates(t *testing.T) {
	a := newTestApp(t)
	accID := insertTestAccount(t, a, "Checking")

	// Pre-seed an existing transaction via CreateTransaction (not BulkInsert)
	existing := models.Transaction{
		AccountID:    accID,
		Date:         mustDate(2025, 5, 10),
		AmountCents:  -500,
		Payee:        "ACME",
		ImportHash:   "existinghash",
		ImportSource: "ofx",
	}
	if err := a.db.CreateTransaction(&existing); err != nil {
		t.Fatalf("seed existing: %v", err)
	}

	// Import a row with same date/amount but different hash/source
	normalized := importer.NormalizePayee("ACME")
	row := importer.Row{
		Date:            "2025-05-10",
		AmountCents:     -500,
		Payee:           "ACME",
		PayeeNormalized: normalized,
		ImportHash:      importer.HashCSV(accID, "2025-05-10", -500, normalized),
		ImportSource:    "csv",
	}

	res, err := a.bulkImport(accID, []importer.Row{row}, nil, "csv", "test.csv")
	if err != nil {
		t.Fatalf("bulkImport: %v", err)
	}
	if len(res.PossibleDupes) != 1 {
		t.Errorf("PossibleDupes len = %d, want 1", len(res.PossibleDupes))
	}
}

// --- ResolveDuplicate tests ---

func seedTwoCsvTx(t *testing.T, a *App) (keepID, deleteID int64) {
	t.Helper()
	accID := insertTestAccount(t, a, "Checking")

	keep := models.Transaction{
		AccountID:    accID,
		Date:         mustDate(2025, 6, 1),
		AmountCents:  -1000,
		Payee:        "Keep",
		ImportHash:   "keephash",
		ImportSource: "csv",
	}
	if err := a.db.CreateTransaction(&keep); err != nil {
		t.Fatalf("seed keep: %v", err)
	}
	del := models.Transaction{
		AccountID:    accID,
		Date:         mustDate(2025, 6, 1),
		AmountCents:  -1000,
		Payee:        "Delete",
		ImportHash:   "delhash",
		ImportSource: "csv",
	}
	if err := a.db.CreateTransaction(&del); err != nil {
		t.Fatalf("seed del: %v", err)
	}
	return keep.ID, del.ID
}

func TestResolveDuplicate_KeepBoth_NoOp(t *testing.T) {
	a := newTestApp(t)
	keepID, deleteID := seedTwoCsvTx(t, a)

	if err := a.ResolveDuplicate("keep_both", keepID, deleteID); err != nil {
		t.Fatalf("ResolveDuplicate keep_both: %v", err)
	}
	// Both should still exist
	if tx, _ := a.db.GetTransaction(keepID); tx == nil {
		t.Error("keepID should still exist after keep_both")
	}
	if tx, _ := a.db.GetTransaction(deleteID); tx == nil {
		t.Error("deleteID should still exist after keep_both")
	}
}

func TestResolveDuplicate_DeleteNew_RemovesAndRecalcs(t *testing.T) {
	a := newTestApp(t)
	keepID, deleteID := seedTwoCsvTx(t, a)

	if err := a.ResolveDuplicate("delete_new", keepID, deleteID); err != nil {
		t.Fatalf("ResolveDuplicate delete_new: %v", err)
	}
	if tx, _ := a.db.GetTransaction(deleteID); tx != nil {
		t.Error("deleteID should be gone after delete_new")
	}
	if tx, _ := a.db.GetTransaction(keepID); tx == nil {
		t.Error("keepID should still exist")
	}

	// Balance should reflect only the keep transaction (-1000)
	keep, _ := a.db.GetTransaction(keepID)
	acc, _ := a.db.GetAccount(keep.AccountID)
	if acc.BalanceCents != -1000 {
		t.Errorf("BalanceCents = %d, want -1000 after delete_new", acc.BalanceCents)
	}
}

func TestResolveDuplicate_Merge(t *testing.T) {
	a := newTestApp(t)
	accID := insertTestAccount(t, a, "Checking")

	// CSV keep + OFX delete
	keep := models.Transaction{
		AccountID:    accID,
		Date:         mustDate(2025, 7, 1),
		AmountCents:  -2000,
		Payee:        "Keep",
		ImportHash:   "csvhash",
		ImportSource: "csv",
	}
	if err := a.db.CreateTransaction(&keep); err != nil {
		t.Fatalf("seed keep: %v", err)
	}
	del := models.Transaction{
		AccountID:    accID,
		Date:         mustDate(2025, 7, 1),
		AmountCents:  -2000,
		Payee:        "Delete",
		ImportHash:   "ofxhash",
		ImportSource: "ofx",
	}
	if err := a.db.CreateTransaction(&del); err != nil {
		t.Fatalf("seed del: %v", err)
	}

	if err := a.ResolveDuplicate("merge", keep.ID, del.ID); err != nil {
		t.Fatalf("ResolveDuplicate merge: %v", err)
	}

	// del should be gone
	if tx, _ := a.db.GetTransaction(del.ID); tx != nil {
		t.Error("deleteID should be gone after merge")
	}

	// keep should have the OFX hash (OFX wins according to MergeTransaction logic)
	merged, _ := a.db.GetTransaction(keep.ID)
	if merged == nil {
		t.Fatal("keepID should still exist after merge")
	}
	if merged.ImportHash != "ofxhash" {
		t.Errorf("after merge ImportHash = %q, want ofxhash", merged.ImportHash)
	}
}

func TestResolveDuplicate_CaseInsensitive(t *testing.T) {
	a := newTestApp(t)
	keepID, deleteID := seedTwoCsvTx(t, a)

	// "Keep_Both" mixed case should work
	if err := a.ResolveDuplicate("Keep_Both", keepID, deleteID); err != nil {
		t.Fatalf("ResolveDuplicate Keep_Both: %v", err)
	}
	if tx, _ := a.db.GetTransaction(keepID); tx == nil {
		t.Error("keepID should still exist")
	}
	if tx, _ := a.db.GetTransaction(deleteID); tx == nil {
		t.Error("deleteID should still exist")
	}
}

func TestResolveDuplicate_UnknownAction_Errors(t *testing.T) {
	a := newTestApp(t)
	keepID, deleteID := seedTwoCsvTx(t, a)

	err := a.ResolveDuplicate("bogus", keepID, deleteID)
	if err == nil {
		t.Fatal("expected error for unknown action")
	}
	if !strings.Contains(err.Error(), "unknown action") {
		t.Errorf("error %q should contain 'unknown action'", err)
	}
}

func TestResolveDuplicate_RequireDB_Error(t *testing.T) {
	err := (&App{}).ResolveDuplicate("keep_both", 1, 2)
	if err == nil {
		t.Fatal("expected error from no-DB App")
	}
	if !strings.Contains(err.Error(), "no file open") {
		t.Errorf("error %q should mention 'no file open'", err)
	}
}

