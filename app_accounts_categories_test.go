package main

import (
	"strings"
	"testing"

	"kino/internal/db"
	"kino/internal/models"
)

// ─── Accounts ─────────────────────────────────────────────────────────────────

func TestApp_ListAccounts(t *testing.T) {
	a := newTestApp(t)

	acc, err := a.CreateAccount(models.Account{Name: "Savings", Type: models.AccountSavings, Currency: "USD"})
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}

	list, err := a.ListAccounts()
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListAccounts len = %d, want 1", len(list))
	}
	if list[0].ID != acc.ID || list[0].Name != "Savings" {
		t.Errorf("unexpected account: %+v", list[0])
	}
}

func TestApp_CreateUpdateDeleteAccount(t *testing.T) {
	a := newTestApp(t)

	acc, err := a.CreateAccount(models.Account{Name: "Before", Type: models.AccountChecking, Currency: "USD"})
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if acc.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	acc.Name = "After"
	acc.Currency = "EUR"
	if err := a.UpdateAccount(acc); err != nil {
		t.Fatalf("UpdateAccount: %v", err)
	}

	list, _ := a.ListAccounts()
	if list[0].Name != "After" {
		t.Errorf("Name after update = %q, want After", list[0].Name)
	}

	if err := a.DeleteAccount(acc.ID); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	list2, _ := a.ListAccounts()
	if len(list2) != 0 {
		t.Errorf("after delete ListAccounts len = %d, want 0", len(list2))
	}
}

func TestApp_ListAccounts_RequireDB(t *testing.T) {
	_, err := (&App{}).ListAccounts()
	if err == nil || !strings.Contains(err.Error(), "no file open") {
		t.Errorf("expected requireDB error, got %v", err)
	}
}

func TestApp_CreateAccount_RequireDB(t *testing.T) {
	_, err := (&App{}).CreateAccount(models.Account{})
	if err == nil || !strings.Contains(err.Error(), "no file open") {
		t.Errorf("expected requireDB error, got %v", err)
	}
}

func TestApp_UpdateAccount_RequireDB(t *testing.T) {
	err := (&App{}).UpdateAccount(models.Account{})
	if err == nil || !strings.Contains(err.Error(), "no file open") {
		t.Errorf("expected requireDB error, got %v", err)
	}
}

func TestApp_DeleteAccount_RequireDB(t *testing.T) {
	err := (&App{}).DeleteAccount(1)
	if err == nil || !strings.Contains(err.Error(), "no file open") {
		t.Errorf("expected requireDB error, got %v", err)
	}
}

// ─── Categories ───────────────────────────────────────────────────────────────

func TestApp_ListCategories_ReturnsSeedData(t *testing.T) {
	a := newTestApp(t)

	cats, err := a.ListCategories()
	if err != nil {
		t.Fatalf("ListCategories: %v", err)
	}
	// Schema seeds 12 system categories
	if len(cats) < 12 {
		t.Errorf("ListCategories returned %d, want at least 12 (system categories)", len(cats))
	}
}

func TestApp_CreateUpdateDeleteCategory(t *testing.T) {
	a := newTestApp(t)

	cat, err := a.CreateCategory(models.Category{Name: "MyExpense", Color: "#112233", Icon: "tag"})
	if err != nil {
		t.Fatalf("CreateCategory: %v", err)
	}
	if cat.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	cat.Name = "Renamed"
	if err := a.UpdateCategory(cat); err != nil {
		t.Fatalf("UpdateCategory: %v", err)
	}

	cats, _ := a.ListCategories()
	var found bool
	for _, c := range cats {
		if c.ID == cat.ID {
			if c.Name != "Renamed" {
				t.Errorf("Name after update = %q, want Renamed", c.Name)
			}
			found = true
		}
	}
	if !found {
		t.Error("updated category not found in ListCategories")
	}

	if err := a.DeleteCategory(cat.ID); err != nil {
		t.Fatalf("DeleteCategory: %v", err)
	}
	cats2, _ := a.ListCategories()
	for _, c := range cats2 {
		if c.ID == cat.ID {
			t.Error("deleted category still appears in ListCategories")
		}
	}
}

func TestApp_GetCategoryTransactionCount(t *testing.T) {
	a := newTestApp(t)
	accID := insertTestAccount(t, a, "Checking")

	cat, err := a.CreateCategory(models.Category{Name: "Groceries", Color: "#aabbcc", Icon: "tag"})
	if err != nil {
		t.Fatalf("CreateCategory: %v", err)
	}

	insertTestTx(t, a, accID, mustDate(2025, 1, 1), -500, &cat.ID)
	insertTestTx(t, a, accID, mustDate(2025, 1, 2), -300, &cat.ID)

	count, err := a.GetCategoryTransactionCount(cat.ID)
	if err != nil {
		t.Fatalf("GetCategoryTransactionCount: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestApp_ListCategories_RequireDB(t *testing.T) {
	_, err := (&App{}).ListCategories()
	if err == nil || !strings.Contains(err.Error(), "no file open") {
		t.Errorf("expected requireDB error, got %v", err)
	}
}

func TestApp_CreateCategory_RequireDB(t *testing.T) {
	_, err := (&App{}).CreateCategory(models.Category{})
	if err == nil || !strings.Contains(err.Error(), "no file open") {
		t.Errorf("expected requireDB error, got %v", err)
	}
}

func TestApp_GetCategoryTransactionCount_RequireDB(t *testing.T) {
	_, err := (&App{}).GetCategoryTransactionCount(1)
	if err == nil || !strings.Contains(err.Error(), "no file open") {
		t.Errorf("expected requireDB error, got %v", err)
	}
}

// ─── Transactions ─────────────────────────────────────────────────────────────

func TestApp_GetTransaction_AndListAndCreateUpdateDelete(t *testing.T) {
	a := newTestApp(t)
	accID := insertTestAccount(t, a, "Checking")

	tx, err := a.CreateTransaction(models.Transaction{
		AccountID:   accID,
		Date:        mustDate(2025, 3, 1),
		AmountCents: -999,
		Payee:       "Coffee Shop",
	})
	if err != nil {
		t.Fatalf("CreateTransaction: %v", err)
	}
	if tx.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	got, err := a.GetTransaction(tx.ID)
	if err != nil {
		t.Fatalf("GetTransaction: %v", err)
	}
	if got == nil || got.AmountCents != -999 {
		t.Errorf("GetTransaction returned %+v, want AmountCents=-999", got)
	}

	page, err := a.ListTransactions(db.TxFilter{AccountID: &accID})
	if err != nil {
		t.Fatalf("ListTransactions: %v", err)
	}
	if page.Total != 1 {
		t.Errorf("Total = %d, want 1", page.Total)
	}

	tx.Payee = "Updated Coffee"
	if err := a.UpdateTransaction(tx); err != nil {
		t.Fatalf("UpdateTransaction: %v", err)
	}
	got2, _ := a.GetTransaction(tx.ID)
	if got2.Payee != "Updated Coffee" {
		t.Errorf("Payee after update = %q, want 'Updated Coffee'", got2.Payee)
	}

	if err := a.DeleteTransaction(tx.ID); err != nil {
		t.Fatalf("DeleteTransaction: %v", err)
	}
	got3, _ := a.GetTransaction(tx.ID)
	if got3 != nil {
		t.Error("deleted transaction should return nil")
	}
}

func TestApp_ListTransactions_RequireDB(t *testing.T) {
	_, err := (&App{}).ListTransactions(db.TxFilter{})
	if err == nil || !strings.Contains(err.Error(), "no file open") {
		t.Errorf("expected requireDB error, got %v", err)
	}
}

func TestApp_GetTransaction_RequireDB(t *testing.T) {
	_, err := (&App{}).GetTransaction(1)
	if err == nil || !strings.Contains(err.Error(), "no file open") {
		t.Errorf("expected requireDB error, got %v", err)
	}
}

func TestApp_CreateTransaction_RequireDB(t *testing.T) {
	_, err := (&App{}).CreateTransaction(models.Transaction{})
	if err == nil || !strings.Contains(err.Error(), "no file open") {
		t.Errorf("expected requireDB error, got %v", err)
	}
}

func TestApp_UpdateTransaction_RequireDB(t *testing.T) {
	err := (&App{}).UpdateTransaction(models.Transaction{})
	if err == nil || !strings.Contains(err.Error(), "no file open") {
		t.Errorf("expected requireDB error, got %v", err)
	}
}

func TestApp_DeleteTransaction_RequireDB(t *testing.T) {
	err := (&App{}).DeleteTransaction(1)
	if err == nil || !strings.Contains(err.Error(), "no file open") {
		t.Errorf("expected requireDB error, got %v", err)
	}
}

func TestApp_UpdateTransaction_CrossAccount_RecalcsBothAccounts(t *testing.T) {
	a := newTestApp(t)
	accAID := insertTestAccount(t, a, "Account A")
	accBID := insertTestAccount(t, a, "Account B")

	// Create a transaction on account A.
	tx, err := a.CreateTransaction(models.Transaction{
		AccountID:   accAID,
		Date:        mustDate(2025, 1, 1),
		AmountCents: -5000,
		Payee:       "test payee",
	})
	if err != nil {
		t.Fatalf("CreateTransaction: %v", err)
	}

	// Verify account A's balance is -5000.
	accA, err := a.db.GetAccount(accAID)
	if err != nil {
		t.Fatalf("GetAccount A after create: %v", err)
	}
	if accA.BalanceCents != -5000 {
		t.Errorf("account A balance after create = %d, want -5000", accA.BalanceCents)
	}

	// Move the transaction to account B.
	tx.AccountID = accBID
	if err := a.UpdateTransaction(tx); err != nil {
		t.Fatalf("UpdateTransaction: %v", err)
	}

	// Verify account B's balance is now -5000 (has the transaction).
	accB, err := a.db.GetAccount(accBID)
	if err != nil {
		t.Fatalf("GetAccount B after update: %v", err)
	}
	if accB.BalanceCents != -5000 {
		t.Errorf("account B balance after move = %d, want -5000", accB.BalanceCents)
	}

	// Verify account A's balance is now 0 (no longer has the transaction).
	accA2, err := a.db.GetAccount(accAID)
	if err != nil {
		t.Fatalf("GetAccount A after update: %v", err)
	}
	if accA2.BalanceCents != 0 {
		t.Errorf("account A balance after move = %d, want 0", accA2.BalanceCents)
	}
}

func TestApp_UpdateCategory_RequireDB(t *testing.T) {
	err := (&App{}).UpdateCategory(models.Category{})
	if err == nil || !strings.Contains(err.Error(), "no file open") {
		t.Errorf("expected requireDB error, got %v", err)
	}
}

func TestApp_DeleteCategory_RequireDB(t *testing.T) {
	err := (&App{}).DeleteCategory(1)
	if err == nil || !strings.Contains(err.Error(), "no file open") {
		t.Errorf("expected requireDB error, got %v", err)
	}
}

// ─── Net worth history ────────────────────────────────────────────────────────

func TestApp_GetNetWorthHistory_ReturnsNPoints(t *testing.T) {
	a := newTestApp(t)

	pts, err := a.GetNetWorthHistory(6)
	if err != nil {
		t.Fatalf("GetNetWorthHistory: %v", err)
	}
	if len(pts) != 6 {
		t.Errorf("len = %d, want 6", len(pts))
	}
}

func TestApp_GetNetWorthHistory_RequireDB(t *testing.T) {
	_, err := (&App{}).GetNetWorthHistory(12)
	if err == nil || !strings.Contains(err.Error(), "no file open") {
		t.Errorf("expected requireDB error, got %v", err)
	}
}
