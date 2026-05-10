package models

import "time"

// All monetary values are stored in cents (int64) to avoid floating-point drift.
// Negative = money out, positive = money in, from the user's perspective.

type AccountType string

const (
	AccountChecking   AccountType = "checking"
	AccountSavings    AccountType = "savings"
	AccountCreditCard AccountType = "credit_card"
	AccountInvestment AccountType = "investment"
	AccountLoan       AccountType = "loan"
	AccountCrypto     AccountType = "crypto"
	AccountCash       AccountType = "cash"
	AccountOther      AccountType = "other"
)

type Account struct {
	ID           int64       `json:"id"`
	Name         string      `json:"name"`
	Type         AccountType `json:"type"`
	Institution  string      `json:"institution"`
	BalanceCents int64       `json:"balanceCents"`
	Currency     string      `json:"currency"`
	IsHidden     bool        `json:"isHidden"`
	SortOrder    int         `json:"sortOrder"`
	LastSyncedAt *time.Time  `json:"lastSyncedAt"`
	CreatedAt    time.Time   `json:"createdAt"`
	UpdatedAt    time.Time   `json:"updatedAt"`
}

// ─────────────────────────────────────────────────────────────────────────────

type Transaction struct {
	ID              int64      `json:"id"`
	AccountID       int64      `json:"accountId"`
	Date            time.Time  `json:"date"`
	AmountCents     int64      `json:"amountCents"`
	Payee           string     `json:"payee"`
	PayeeNormalized string     `json:"payeeNormalized"`
	Notes           string     `json:"notes"`
	CategoryID      *int64     `json:"categoryId"`
	IsTransfer      bool       `json:"isTransfer"`
	TransferPairID  *int64     `json:"transferPairId"`
	IsReconciled    bool       `json:"isReconciled"`
	ImportHash      string     `json:"importHash"`
	ImportSource    string     `json:"importSource"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

// ─────────────────────────────────────────────────────────────────────────────

type Category struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	ParentID  *int64 `json:"parentId"`
	Color     string `json:"color"`
	Icon      string `json:"icon"`
	IsIncome  bool   `json:"isIncome"`
	IsSystem  bool   `json:"isSystem"`
	SortOrder int    `json:"sortOrder"`
}

// ─────────────────────────────────────────────────────────────────────────────

type BudgetPeriod string

const (
	BudgetMonthly BudgetPeriod = "monthly"
	BudgetWeekly  BudgetPeriod = "weekly"
	BudgetAnnual  BudgetPeriod = "annual"
)

type Budget struct {
	ID          int64        `json:"id"`
	CategoryID  int64        `json:"categoryId"`
	AmountCents int64        `json:"amountCents"`
	Period      BudgetPeriod `json:"period"`
	RollsOver   bool         `json:"rollsOver"`
	StartDate   time.Time    `json:"startDate"`
	EndDate     *time.Time   `json:"endDate"`
}

// ─────────────────────────────────────────────────────────────────────────────

type GoalType string

const (
	GoalSavings    GoalType = "savings"
	GoalDebtPayoff GoalType = "debt_payoff"
)

type Goal struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	Type            GoalType   `json:"type"`
	TargetCents     int64      `json:"targetCents"`
	CurrentCents    int64      `json:"currentCents"`
	LinkedAccountID *int64     `json:"linkedAccountId"`
	TargetDate      *time.Time `json:"targetDate"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

// ─────────────────────────────────────────────────────────────────────────────

type PayeeRule struct {
	ID          int64  `json:"id"`
	Pattern     string `json:"pattern"`
	IsRegex     bool   `json:"isRegex"`
	ReplaceName string `json:"replaceName"`
	CategoryID  *int64 `json:"categoryId"`
	Priority    int    `json:"priority"`
}

// ─────────────────────────────────────────────────────────────────────────────

type AppSetting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
