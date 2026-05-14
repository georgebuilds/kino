package db

import (
	"fmt"
	"time"
)

// NetWorthPoint is one month's snapshot of overall financial position.
type NetWorthPoint struct {
	Month       string `json:"month"`       // "YYYY-MM"
	NetWorth    int64  `json:"netWorth"`    // assets + liabilities (liabilities are negative)
	Assets      int64  `json:"assets"`      // sum of asset-type accounts
	Liabilities int64  `json:"liabilities"` // sum of liability-type accounts (≤ 0)
}

// assetAccountTypes are accounts that contribute positively to net worth.
var assetAccountTypes = map[string]bool{
	"checking":   true,
	"savings":    true,
	"investment": true,
	"cash":       true,
	"crypto":     true,
	"other":      true,
}

// liabilityAccountTypes are accounts where a balance means money owed.
var liabilityAccountTypes = map[string]bool{
	"credit_card": true,
	"loan":        true,
}

// GetNetWorthHistory returns monthly net-worth snapshots for the past `months`
// calendar months (inclusive of the current month).
//
// Algorithm: we know each account's current balance (= sum of all its
// transactions, maintained by RecalcBalance).  Working backwards, each prior
// month's balance = next-month's balance − transactions that occurred in the
// next month.
func (db *DB) GetNetWorthHistory(months int) ([]NetWorthPoint, error) {
	if months <= 0 {
		months = 12
	}

	// ── 1. Current balance per account ───────────────────────────────────────
	type accRec struct {
		typ     string
		balance int64
	}
	rows, err := db.Query(`
		SELECT id, type, balance_cents FROM accounts WHERE is_hidden = 0
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	accMap := map[int64]accRec{}
	for rows.Next() {
		var id int64
		var r accRec
		if err := rows.Scan(&id, &r.typ, &r.balance); err != nil {
			return nil, err
		}
		accMap[id] = r
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var currentAssets, currentLiab int64
	for _, r := range accMap {
		if assetAccountTypes[r.typ] {
			currentAssets += r.balance
		} else if liabilityAccountTypes[r.typ] {
			currentLiab += r.balance
		}
	}

	// ── 2. Generate month labels (ascending: oldest … current) ───────────────
	type mk struct{ y, m int }
	monthKeys := make([]mk, months)
	now := time.Now().UTC()
	for i := 0; i < months; i++ {
		t := now.AddDate(0, -(months-1-i), 0)
		monthKeys[i] = mk{t.Year(), int(t.Month())}
	}

	// ── 3. Monthly transaction deltas per account type for the full window ────
	startDate := fmt.Sprintf("%04d-%02d-01", monthKeys[0].y, monthKeys[0].m)

	txRows, err := db.Query(`
		SELECT t.account_id,
		       strftime('%Y-%m', t.date) AS month,
		       SUM(t.amount_cents)       AS delta
		FROM transactions t
		JOIN accounts a ON a.id = t.account_id
		WHERE a.is_hidden = 0
		  AND t.date >= ?
		GROUP BY t.account_id, month
	`, startDate)
	if err != nil {
		return nil, err
	}
	defer txRows.Close()

	type monthDelta struct{ assets, liab int64 }
	deltas := map[string]monthDelta{}

	for txRows.Next() {
		var accID int64
		var month string
		var delta int64
		if err := txRows.Scan(&accID, &month, &delta); err != nil {
			return nil, err
		}
		r, ok := accMap[accID]
		if !ok {
			continue
		}
		md := deltas[month]
		if assetAccountTypes[r.typ] {
			md.assets += delta
		} else if liabilityAccountTypes[r.typ] {
			md.liab += delta
		}
		deltas[month] = md
	}
	if err := txRows.Err(); err != nil {
		return nil, err
	}

	// ── 4. Walk backwards from current to reconstruct each month's balance ────
	points := make([]NetWorthPoint, months)
	runAssets := currentAssets
	runLiab := currentLiab

	for i := months - 1; i >= 0; i-- {
		key := monthKeys[i]
		ms := fmt.Sprintf("%04d-%02d", key.y, key.m)

		points[i] = NetWorthPoint{
			Month:       ms,
			NetWorth:    runAssets + runLiab,
			Assets:      runAssets,
			Liabilities: runLiab,
		}

		if i > 0 {
			// Subtract this month's activity to reach previous month-end
			md := deltas[ms]
			runAssets -= md.assets
			runLiab -= md.liab
		}
	}

	return points, nil
}
