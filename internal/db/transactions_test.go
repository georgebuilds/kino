package db

import (
	"testing"

	"kino/internal/models"
)

func TestListTransactions_BasicFilter(t *testing.T) {
	d := newTestDB(t)
	acc1 := insertTestAccount(t, d, "A1")
	acc2 := insertTestAccount(t, d, "A2")

	cat := &models.Category{Name: "Misc", Color: "#000", Icon: "tag"}
	if err := d.CreateCategory(cat); err != nil {
		t.Fatalf("CreateCategory: %v", err)
	}

	// 3 in acc1: two in cat (Jan), one no-cat (Feb).
	// 2 in acc2: both no-cat (March).
	rows := []models.Transaction{
		{AccountID: acc1, Date: mustDate(2025, 1, 5), AmountCents: -100, Payee: "x", CategoryID: i64ptr(cat.ID)},
		{AccountID: acc1, Date: mustDate(2025, 1, 6), AmountCents: -200, Payee: "x", CategoryID: i64ptr(cat.ID)},
		{AccountID: acc1, Date: mustDate(2025, 2, 1), AmountCents: -300, Payee: "x"},
		{AccountID: acc2, Date: mustDate(2025, 3, 1), AmountCents: -400, Payee: "x"},
		{AccountID: acc2, Date: mustDate(2025, 3, 2), AmountCents: -500, Payee: "x"},
	}
	for i := range rows {
		if err := d.CreateTransaction(&rows[i]); err != nil {
			t.Fatalf("create row %d: %v", i, err)
		}
	}

	// AccountID filter → 3 in acc1.
	id := acc1
	page, err := d.ListTransactions(TxFilter{AccountID: &id})
	if err != nil {
		t.Fatalf("ListTransactions(AccountID): %v", err)
	}
	if page.Total != 3 || len(page.Transactions) != 3 {
		t.Fatalf("AccountID filter: Total=%d len=%d, want 3/3", page.Total, len(page.Transactions))
	}

	// CategoryID filter → 2.
	cid := cat.ID
	page, err = d.ListTransactions(TxFilter{CategoryID: &cid})
	if err != nil {
		t.Fatalf("ListTransactions(CategoryID): %v", err)
	}
	if page.Total != 2 {
		t.Fatalf("CategoryID filter: Total=%d, want 2", page.Total)
	}

	// Date range covering January only → 2.
	page, err = d.ListTransactions(TxFilter{DateFrom: "2025-01-01", DateTo: "2025-01-31"})
	if err != nil {
		t.Fatalf("ListTransactions(date range): %v", err)
	}
	if page.Total != 2 {
		t.Fatalf("Date range filter: Total=%d, want 2", page.Total)
	}
}

func TestListTransactions_SearchEscapesWildcards(t *testing.T) {
	d := newTestDB(t)
	acc := insertTestAccount(t, d, "A")

	rows := []models.Transaction{
		{AccountID: acc, Date: mustDate(2025, 6, 1), AmountCents: -100, Payee: "100% milk"},
		{AccountID: acc, Date: mustDate(2025, 6, 2), AmountCents: -100, Payee: "1005 milk"},
		{AccountID: acc, Date: mustDate(2025, 6, 3), AmountCents: -100, Payee: "ten_percent shop"},
		{AccountID: acc, Date: mustDate(2025, 6, 4), AmountCents: -100, Payee: "tenXpercent shop"},
	}
	for i := range rows {
		if err := d.CreateTransaction(&rows[i]); err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}

	// "100%" must match only the literal-percent row.
	page, err := d.ListTransactions(TxFilter{Search: "100%"})
	if err != nil {
		t.Fatalf("Search 100%%: %v", err)
	}
	if page.Total != 1 || page.Transactions[0].Payee != "100% milk" {
		t.Fatalf(`Search "100%%" got Total=%d payee=%q, want 1 / "100%% milk"`,
			page.Total,
			func() string {
				if len(page.Transactions) == 0 {
					return ""
				}
				return page.Transactions[0].Payee
			}())
	}

	// "ten_percent" must match only the literal-underscore row.
	page, err = d.ListTransactions(TxFilter{Search: "ten_percent"})
	if err != nil {
		t.Fatalf("Search ten_percent: %v", err)
	}
	if page.Total != 1 || page.Transactions[0].Payee != "ten_percent shop" {
		t.Fatalf(`Search "ten_percent" got Total=%d payee=%q, want 1 / "ten_percent shop"`,
			page.Total,
			func() string {
				if len(page.Transactions) == 0 {
					return ""
				}
				return page.Transactions[0].Payee
			}())
	}
}

func TestListTransactions_Pagination(t *testing.T) {
	d := newTestDB(t)
	acc := insertTestAccount(t, d, "A")

	for i := 0; i < 5; i++ {
		tx := &models.Transaction{
			AccountID:   acc,
			Date:        mustDate(2025, 7, 1+i),
			AmountCents: int64(-(i + 1) * 100),
			Payee:       "x",
		}
		if err := d.CreateTransaction(tx); err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}

	page, err := d.ListTransactions(TxFilter{Limit: 2})
	if err != nil {
		t.Fatalf("ListTransactions: %v", err)
	}
	if page.Total != 5 {
		t.Fatalf("Total = %d, want 5", page.Total)
	}
	if len(page.Transactions) != 2 {
		t.Fatalf("len(Transactions) = %d, want 2", len(page.Transactions))
	}
}

func TestCreate_And_GetTransaction(t *testing.T) {
	d := newTestDB(t)
	acc := insertTestAccount(t, d, "A")

	tx := &models.Transaction{
		AccountID:    acc,
		Date:         mustDate(2025, 8, 9),
		AmountCents:  -4242,
		Payee:        "Coffee Shop",
		Notes:        "double espresso",
		IsTransfer:   false,
		ImportHash:   "hash-xyz",
		ImportSource: "csv",
	}
	if err := d.CreateTransaction(tx); err != nil {
		t.Fatalf("CreateTransaction: %v", err)
	}
	if tx.ID == 0 {
		t.Fatal("CreateTransaction did not set ID")
	}

	got, err := d.GetTransaction(tx.ID)
	if err != nil {
		t.Fatalf("GetTransaction: %v", err)
	}
	if got == nil {
		t.Fatal("GetTransaction returned nil")
	}
	if got.AccountID != acc || got.AmountCents != -4242 || got.Payee != "Coffee Shop" ||
		got.Notes != "double espresso" || got.ImportHash != "hash-xyz" || got.ImportSource != "csv" {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
	if got.PayeeNormalized != "Coffee Shop" {
		t.Fatalf("PayeeNormalized = %q, want auto-populated to payee", got.PayeeNormalized)
	}
	if got.Date.Format("2006-01-02") != "2025-08-09" {
		t.Fatalf("Date = %s, want 2025-08-09", got.Date.Format("2006-01-02"))
	}
}

func TestBulkInsert_Idempotent(t *testing.T) {
	d := newTestDB(t)
	acc := insertTestAccount(t, d, "A")

	mk := func() []models.Transaction {
		return []models.Transaction{
			{AccountID: acc, Date: mustDate(2025, 1, 1), AmountCents: -100, Payee: "p1", ImportHash: "hash-1", ImportSource: "csv"},
			{AccountID: acc, Date: mustDate(2025, 1, 2), AmountCents: -200, Payee: "p2", ImportHash: "hash-2", ImportSource: "csv"},
			{AccountID: acc, Date: mustDate(2025, 1, 3), AmountCents: -300, Payee: "p3", ImportHash: "hash-3", ImportSource: "csv"},
		}
	}

	ins, dupes, ids, err := d.BulkInsert(mk())
	if err != nil {
		t.Fatalf("first BulkInsert: %v", err)
	}
	if ins != 3 || dupes != 0 || len(ids) != 3 {
		t.Fatalf("first run: inserted=%d dupes=%d ids=%d, want 3/0/3", ins, dupes, len(ids))
	}

	ins2, dupes2, ids2, err := d.BulkInsert(mk())
	if err != nil {
		t.Fatalf("second BulkInsert: %v", err)
	}
	if ins2 != 0 || dupes2 != 3 || len(ids2) != 0 {
		t.Fatalf("second run: inserted=%d dupes=%d ids=%d, want 0/3/0", ins2, dupes2, len(ids2))
	}
}

func TestFindFuzzyDuplicates_CrossSource(t *testing.T) {
	d := newTestDB(t)
	acc := insertTestAccount(t, d, "A")

	// Existing CSV row.
	existing := models.Transaction{
		AccountID: acc, Date: mustDate(2025, 1, 15), AmountCents: -2500,
		Payee: "Vendor", ImportHash: "hashA", ImportSource: "csv",
	}
	insA, _, _, err := d.BulkInsert([]models.Transaction{existing})
	if err != nil {
		t.Fatalf("seed BulkInsert: %v", err)
	}
	if insA != 1 {
		t.Fatalf("seed insert count = %d, want 1", insA)
	}

	// New OFX row, same account/date/amount, different hash.
	newOFX := models.Transaction{
		AccountID: acc, Date: mustDate(2025, 1, 15), AmountCents: -2500,
		Payee: "VENDOR INC", ImportHash: "hashB", ImportSource: "ofx",
	}
	ins, _, newIDs, err := d.BulkInsert([]models.Transaction{newOFX})
	if err != nil {
		t.Fatalf("ofx BulkInsert: %v", err)
	}
	if ins != 1 || len(newIDs) != 1 {
		t.Fatalf("ofx insert: ins=%d newIDs=%v, want 1/[id]", ins, newIDs)
	}

	dupes, err := d.FindFuzzyDuplicates(newIDs)
	if err != nil {
		t.Fatalf("FindFuzzyDuplicates: %v", err)
	}
	if len(dupes) != 1 {
		t.Fatalf("got %d dupes, want 1: %+v", len(dupes), dupes)
	}
	if dupes[0].NewTx.ID != newIDs[0] {
		t.Fatalf("NewTx.ID = %d, want %d", dupes[0].NewTx.ID, newIDs[0])
	}
	if dupes[0].NewTx.ImportSource != "ofx" || dupes[0].ExistingTx.ImportSource != "csv" {
		t.Fatalf("sources: new=%q existing=%q, want ofx/csv", dupes[0].NewTx.ImportSource, dupes[0].ExistingTx.ImportSource)
	}
}

func TestFindFuzzyDuplicates_SameImportNotMatched(t *testing.T) {
	d := newTestDB(t)
	acc := insertTestAccount(t, d, "A")

	// Two rows in a SINGLE BulkInsert: same account/date/amount, different payees → different hashes.
	rows := []models.Transaction{
		{AccountID: acc, Date: mustDate(2025, 2, 10), AmountCents: -500, Payee: "Alpha", ImportHash: "h-alpha", ImportSource: "csv"},
		{AccountID: acc, Date: mustDate(2025, 2, 10), AmountCents: -500, Payee: "Beta", ImportHash: "h-beta", ImportSource: "csv"},
	}
	ins, _, newIDs, err := d.BulkInsert(rows)
	if err != nil {
		t.Fatalf("BulkInsert: %v", err)
	}
	if ins != 2 || len(newIDs) != 2 {
		t.Fatalf("ins=%d newIDs=%d, want 2/2", ins, len(newIDs))
	}

	dupes, err := d.FindFuzzyDuplicates(newIDs)
	if err != nil {
		t.Fatalf("FindFuzzyDuplicates: %v", err)
	}
	if len(dupes) != 0 {
		t.Fatalf("got %d dupes, want 0 (both in newIDs, excluded by NOT IN): %+v", len(dupes), dupes)
	}
}

func TestMergeTransaction_OFXHashWins(t *testing.T) {
	d := newTestDB(t)
	acc := insertTestAccount(t, d, "A")

	keep := &models.Transaction{
		AccountID: acc, Date: mustDate(2025, 3, 1), AmountCents: -1000,
		Payee: "K", ImportHash: "csv-hash", ImportSource: "csv",
	}
	del := &models.Transaction{
		AccountID: acc, Date: mustDate(2025, 3, 1), AmountCents: -1000,
		Payee: "D", ImportHash: "ofx-hash", ImportSource: "ofx",
	}
	if err := d.CreateTransaction(keep); err != nil {
		t.Fatalf("create keep: %v", err)
	}
	if err := d.CreateTransaction(del); err != nil {
		t.Fatalf("create del: %v", err)
	}

	if err := d.MergeTransaction(keep.ID, del.ID); err != nil {
		t.Fatalf("MergeTransaction: %v", err)
	}

	got, err := d.GetTransaction(keep.ID)
	if err != nil {
		t.Fatalf("GetTransaction: %v", err)
	}
	if got == nil {
		t.Fatal("keep row missing after merge")
	}
	if got.ImportHash != "ofx-hash" {
		t.Fatalf("merged ImportHash = %q, want ofx-hash", got.ImportHash)
	}

	gone, err := d.GetTransaction(del.ID)
	if err != nil {
		t.Fatalf("GetTransaction del: %v", err)
	}
	if gone != nil {
		t.Fatalf("delete row still present: %+v", gone)
	}
}

func TestMergeTransaction_AdoptsDeleteCategoryAndNotesWhenKeepIsEmpty(t *testing.T) {
	d := newTestDB(t)
	acc := insertTestAccount(t, d, "A")

	cat := &models.Category{Name: "Adopted", Color: "#aaa", Icon: "tag"}
	if err := d.CreateCategory(cat); err != nil {
		t.Fatalf("CreateCategory: %v", err)
	}

	keep := &models.Transaction{
		AccountID: acc, Date: mustDate(2025, 3, 1), AmountCents: -1000,
		Payee: "K", Notes: "", ImportHash: "k-hash", ImportSource: "csv",
		// CategoryID nil, Notes ""
	}
	del := &models.Transaction{
		AccountID: acc, Date: mustDate(2025, 3, 1), AmountCents: -1000,
		Payee: "D", Notes: "memo here", CategoryID: i64ptr(cat.ID),
		ImportHash: "d-hash", ImportSource: "csv",
	}
	if err := d.CreateTransaction(keep); err != nil {
		t.Fatalf("create keep: %v", err)
	}
	if err := d.CreateTransaction(del); err != nil {
		t.Fatalf("create del: %v", err)
	}

	if err := d.MergeTransaction(keep.ID, del.ID); err != nil {
		t.Fatalf("MergeTransaction: %v", err)
	}

	got, err := d.GetTransaction(keep.ID)
	if err != nil {
		t.Fatalf("GetTransaction: %v", err)
	}
	if got == nil {
		t.Fatal("keep row missing after merge")
	}
	if got.CategoryID == nil || *got.CategoryID != cat.ID {
		t.Fatalf("merged CategoryID = %v, want %d", got.CategoryID, cat.ID)
	}
	if got.Notes != "memo here" {
		t.Fatalf("merged Notes = %q, want \"memo here\"", got.Notes)
	}
}
