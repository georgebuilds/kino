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
