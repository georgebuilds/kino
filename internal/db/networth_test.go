package db

import (
	"testing"
	"time"

	"kino/internal/models"
)

// monthOffset returns a stable mid-month date `k` whole calendar months ago
// from time.Now(). Using day=15 sidesteps end-of-month normalisation traps.
func monthOffset(now time.Time, k int) time.Time {
	y, m, _ := now.Date()
	return time.Date(y, m-time.Month(k), 15, 0, 0, 0, 0, time.UTC)
}

// monthLabel formats t as "YYYY-MM" — matches strftime('%Y-%m', date) in SQLite.
func monthLabel(t time.Time) string {
	return t.Format("2006-01")
}

func TestGetNetWorthHistory_ReconstructsBackwards(t *testing.T) {
	d := newTestDB(t)
	now := time.Now()

	// One asset (checking) account.
	a := &models.Account{Name: "Cash", Type: models.AccountChecking, Currency: "USD"}
	if err := d.CreateAccount(a); err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}

	// One tx in each of the past 3 months.
	txs := []models.Transaction{
		{AccountID: a.ID, Date: monthOffset(now, 2), AmountCents: 100, Payee: "p2"},   // m-2: +100
		{AccountID: a.ID, Date: monthOffset(now, 1), AmountCents: -30, Payee: "p1"},   // m-1: -30
		{AccountID: a.ID, Date: monthOffset(now, 0), AmountCents: 50, Payee: "p0"},    // cur: +50
	}
	for i := range txs {
		if err := d.CreateTransaction(&txs[i]); err != nil {
			t.Fatalf("create tx %d: %v", i, err)
		}
	}
	if err := d.RecalcBalance(a.ID); err != nil {
		t.Fatalf("RecalcBalance: %v", err)
	}

	points, err := d.GetNetWorthHistory(3)
	if err != nil {
		t.Fatalf("GetNetWorthHistory: %v", err)
	}
	if len(points) != 3 {
		t.Fatalf("len(points) = %d, want 3", len(points))
	}

	// Months must be oldest → newest.
	wantLabels := []string{
		monthLabel(monthOffset(now, 2)),
		monthLabel(monthOffset(now, 1)),
		monthLabel(monthOffset(now, 0)),
	}
	for i, want := range wantLabels {
		if points[i].Month != want {
			t.Fatalf("points[%d].Month = %q, want %q", i, points[i].Month, want)
		}
	}

	// Balance after RecalcBalance = 100 - 30 + 50 = 120.
	// Walking backwards from current:
	//   current Assets  = 120
	//   m-1     Assets  = 120 - 50 = 70
	//   m-2     Assets  = 70  - (-30) = 100
	wantAssets := []int64{100, 70, 120}
	for i, want := range wantAssets {
		if points[i].Assets != want {
			t.Fatalf("points[%d].Assets = %d, want %d", i, points[i].Assets, want)
		}
		// Pure-asset account: NetWorth == Assets, Liabilities == 0.
		if points[i].Liabilities != 0 {
			t.Fatalf("points[%d].Liabilities = %d, want 0", i, points[i].Liabilities)
		}
		if points[i].NetWorth != want {
			t.Fatalf("points[%d].NetWorth = %d, want %d", i, points[i].NetWorth, want)
		}
	}
}

func TestGetNetWorthHistory_AssetAndLiability(t *testing.T) {
	d := newTestDB(t)
	now := time.Now()

	checking := &models.Account{Name: "Chk", Type: models.AccountChecking, Currency: "USD"}
	credit := &models.Account{Name: "Card", Type: models.AccountCreditCard, Currency: "USD"}
	if err := d.CreateAccount(checking); err != nil {
		t.Fatalf("create chk: %v", err)
	}
	if err := d.CreateAccount(credit); err != nil {
		t.Fatalf("create card: %v", err)
	}

	// Transactions in current month so balances are non-zero and recent.
	txs := []models.Transaction{
		{AccountID: checking.ID, Date: monthOffset(now, 0), AmountCents: 5000, Payee: "deposit"},
		{AccountID: credit.ID, Date: monthOffset(now, 0), AmountCents: -1000, Payee: "purchase"},
	}
	for i := range txs {
		if err := d.CreateTransaction(&txs[i]); err != nil {
			t.Fatalf("create tx %d: %v", i, err)
		}
	}
	if err := d.RecalcBalance(checking.ID); err != nil {
		t.Fatalf("recalc chk: %v", err)
	}
	if err := d.RecalcBalance(credit.ID); err != nil {
		t.Fatalf("recalc card: %v", err)
	}

	points, err := d.GetNetWorthHistory(1)
	if err != nil {
		t.Fatalf("GetNetWorthHistory: %v", err)
	}
	if len(points) != 1 {
		t.Fatalf("len(points) = %d, want 1", len(points))
	}

	if points[0].Assets != 5000 {
		t.Fatalf("Assets = %d, want 5000", points[0].Assets)
	}
	if points[0].Liabilities != -1000 {
		t.Fatalf("Liabilities = %d, want -1000", points[0].Liabilities)
	}
	if points[0].NetWorth != 4000 {
		t.Fatalf("NetWorth = %d, want 4000", points[0].NetWorth)
	}
}

func TestGetNetWorthHistory_DefaultsTo12_WhenZero(t *testing.T) {
	d := newTestDB(t)

	points, err := d.GetNetWorthHistory(0)
	if err != nil {
		t.Fatalf("GetNetWorthHistory(0): %v", err)
	}
	if len(points) != 12 {
		t.Fatalf("len(points) = %d, want 12 (default when months=0)", len(points))
	}
}

func TestGetNetWorthHistory_NoAccounts_ReturnsEmptyish(t *testing.T) {
	d := newTestDB(t)

	points, err := d.GetNetWorthHistory(3)
	if err != nil {
		t.Fatalf("GetNetWorthHistory(3) with no accounts: %v", err)
	}
	if len(points) != 3 {
		t.Fatalf("len(points) = %d, want 3", len(points))
	}
	for i, p := range points {
		if p.Assets != 0 || p.Liabilities != 0 || p.NetWorth != 0 {
			t.Fatalf("points[%d] = %+v, want all zeros (no accounts)", i, p)
		}
	}
}

func TestGetNetWorthHistory_IgnoresHiddenAccounts(t *testing.T) {
	d := newTestDB(t)
	now := time.Now()

	hidden := &models.Account{Name: "Hidden", Type: models.AccountChecking, Currency: "USD", IsHidden: true}
	if err := d.CreateAccount(hidden); err != nil {
		t.Fatalf("CreateAccount hidden: %v", err)
	}

	// Transaction on the hidden account in the current month.
	tx := &models.Transaction{
		AccountID:   hidden.ID,
		Date:        monthOffset(now, 0),
		AmountCents: 99999,
		Payee:       "invisible",
	}
	if err := d.CreateTransaction(tx); err != nil {
		t.Fatalf("CreateTransaction: %v", err)
	}
	if err := d.RecalcBalance(hidden.ID); err != nil {
		t.Fatalf("RecalcBalance: %v", err)
	}

	points, err := d.GetNetWorthHistory(1)
	if err != nil {
		t.Fatalf("GetNetWorthHistory: %v", err)
	}
	if len(points) != 1 {
		t.Fatalf("len(points) = %d, want 1", len(points))
	}
	if points[0].Assets != 0 {
		t.Fatalf("Assets = %d, want 0 (hidden account excluded)", points[0].Assets)
	}
	if points[0].NetWorth != 0 {
		t.Fatalf("NetWorth = %d, want 0 (hidden account excluded)", points[0].NetWorth)
	}
}

func TestGetNetWorthHistory_OldTransactionsExcluded(t *testing.T) {
	d := newTestDB(t)
	now := time.Now()

	a := &models.Account{Name: "OldTxAcct", Type: models.AccountChecking, Currency: "USD"}
	if err := d.CreateAccount(a); err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}

	// A transaction 24 months ago — outside the 3-month window.
	oldTx := &models.Transaction{
		AccountID:   a.ID,
		Date:        monthOffset(now, 24),
		AmountCents: 1000,
		Payee:       "old",
	}
	if err := d.CreateTransaction(oldTx); err != nil {
		t.Fatalf("CreateTransaction old: %v", err)
	}
	if err := d.RecalcBalance(a.ID); err != nil {
		t.Fatalf("RecalcBalance: %v", err)
	}

	// Current balance = 1000 (the old tx is the only one).
	// GetNetWorthHistory(3) walks back 3 months from current.
	// Since there are no transactions within the 3-month window,
	// all three months should show the same balance of 1000.
	points, err := d.GetNetWorthHistory(3)
	if err != nil {
		t.Fatalf("GetNetWorthHistory: %v", err)
	}
	if len(points) != 3 {
		t.Fatalf("len(points) = %d, want 3", len(points))
	}
	// All months should reflect the same current balance (old tx not in the window).
	for i, p := range points {
		if p.Assets != 1000 {
			t.Fatalf("points[%d].Assets = %d, want 1000 (old tx affects balance but not in-window delta)", i, p.Assets)
		}
	}
}
