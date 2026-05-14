package main

import (
	"fmt"

	"kino/internal/db"
)

// ── View models ───────────────────────────────────────────────────────────────

// FlowNode is one bar in the Sankey diagram (income source or expense bucket).
type FlowNode struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Color      string `json:"color"`
	ValueCents int64  `json:"valueCents"`
	IsIncome   bool   `json:"isIncome"`
}

// FlowLink is one ribbon connecting a left node to a right node.
type FlowLink struct {
	SourceID   string `json:"sourceId"`
	TargetID   string `json:"targetId"`
	ValueCents int64  `json:"valueCents"`
	Color      string `json:"color"` // inherits from source node
}

// CashFlow is the full payload for the cash flow / Sankey view.
type CashFlow struct {
	LeftNodes    []FlowNode `json:"leftNodes"`    // income sources
	RightNodes   []FlowNode `json:"rightNodes"`   // expense categories + Saved
	Links        []FlowLink `json:"links"`
	IncomeCents  int64      `json:"incomeCents"`
	ExpenseCents int64      `json:"expenseCents"`
	SavedCents   int64      `json:"savedCents"`
}

// ── Wails method ──────────────────────────────────────────────────────────────

// GetCashFlow returns Sankey data for the given month.
//
// Left nodes  = income transactions grouped by category.
// Right nodes = expense transactions grouped by category, plus a "Saved" node
// if income > expenses.
// Links       = every (left, right) pair where the ribbon value is
// left.value × right.value / totalIncome (proportional allocation).
func (a *App) GetCashFlow(year, month int) (CashFlow, error) {
	if err := a.requireDB(); err != nil {
		return CashFlow{}, err
	}

	dateFrom := fmt.Sprintf("%04d-%02d-01", year, month)
	ny, nm := year, month+1
	if nm > 12 {
		nm = 1
		ny++
	}
	dateTo := fmt.Sprintf("%04d-%02d-01", ny, nm)

	// ── Income by category ────────────────────────────────────────────────
	// NULL-category rows are bucketed into the seeded "Uncategorized" category.
	iRows, err := a.db.Query(`
		SELECT
			CAST(c.id AS TEXT) AS cat_id,
			c.name             AS cat_name,
			c.color            AS cat_color,
			SUM(t.amount_cents) AS total
		FROM transactions t
		JOIN categories c ON c.id = COALESCE(t.category_id, ?)
		WHERE t.date >= ? AND t.date < ?
		  AND t.amount_cents > 0
		  AND t.is_transfer = 0
		GROUP BY c.id
		HAVING total > 0
		ORDER BY total DESC
	`, db.UncategorizedCategoryID, dateFrom, dateTo)
	if err != nil {
		return CashFlow{}, err
	}
	defer iRows.Close()

	var leftNodes []FlowNode
	var totalIncome int64
	for iRows.Next() {
		var n FlowNode
		n.IsIncome = true
		if err := iRows.Scan(&n.ID, &n.Name, &n.Color, &n.ValueCents); err != nil {
			return CashFlow{}, err
		}
		n.ID = "inc-" + n.ID
		leftNodes = append(leftNodes, n)
		totalIncome += n.ValueCents
	}
	if err := iRows.Err(); err != nil {
		return CashFlow{}, err
	}

	// ── Expenses by category ──────────────────────────────────────────────
	// NULL-category rows are bucketed into the seeded "Uncategorized" category.
	eRows, err := a.db.Query(`
		SELECT
			CAST(c.id AS TEXT) AS cat_id,
			c.name             AS cat_name,
			c.color            AS cat_color,
			SUM(-t.amount_cents) AS total
		FROM transactions t
		JOIN categories c ON c.id = COALESCE(t.category_id, ?)
		WHERE t.date >= ? AND t.date < ?
		  AND t.amount_cents < 0
		  AND t.is_transfer = 0
		GROUP BY c.id
		HAVING total > 0
		ORDER BY total DESC
	`, db.UncategorizedCategoryID, dateFrom, dateTo)
	if err != nil {
		return CashFlow{}, err
	}
	defer eRows.Close()

	var rightNodes []FlowNode
	var totalExpenses int64
	for eRows.Next() {
		var n FlowNode
		if err := eRows.Scan(&n.ID, &n.Name, &n.Color, &n.ValueCents); err != nil {
			return CashFlow{}, err
		}
		n.ID = "exp-" + n.ID
		rightNodes = append(rightNodes, n)
		totalExpenses += n.ValueCents
	}
	if err := eRows.Err(); err != nil {
		return CashFlow{}, err
	}

	// ── Saved node ────────────────────────────────────────────────────────
	savedCents := totalIncome - totalExpenses
	if savedCents > 0 {
		rightNodes = append(rightNodes, FlowNode{
			ID:         "saved",
			Name:       "Saved",
			Color:      "#1A8A61",
			ValueCents: savedCents,
		})
	}

	// If no data at all, return empty
	if totalIncome == 0 && totalExpenses == 0 {
		return CashFlow{
			LeftNodes: leftNodes, RightNodes: rightNodes,
			IncomeCents: 0, ExpenseCents: 0, SavedCents: 0,
		}, nil
	}

	// ── Links: proportional allocation ───────────────────────────────────
	// Each income source flows into every right node proportionally.
	// ribbon value = left.value × right.value / totalIncome
	//
	// Edge case: when totalIncome == 0 but expenses > 0, we still need links.
	// We use totalExpenses as the denominator in that case.
	denom := totalIncome
	if denom == 0 {
		denom = totalExpenses
	}

	var links []FlowLink
	for _, ln := range leftNodes {
		for _, rn := range rightNodes {
			v := int64(float64(ln.ValueCents) * float64(rn.ValueCents) / float64(denom))
			if v <= 0 {
				continue
			}
			links = append(links, FlowLink{
				SourceID:   ln.ID,
				TargetID:   rn.ID,
				ValueCents: v,
				Color:      ln.Color,
			})
		}
	}

	return CashFlow{
		LeftNodes:    leftNodes,
		RightNodes:   rightNodes,
		Links:        links,
		IncomeCents:  totalIncome,
		ExpenseCents: totalExpenses,
		SavedCents:   savedCents,
	}, nil
}
