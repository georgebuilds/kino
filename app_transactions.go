package main

import (
	"errors"

	"kino/internal/db"
	"kino/internal/models"
)

// TxFilter is re-exported at the main package level so Wails generates
// the TypeScript type for it automatically.
type TxFilter = db.TxFilter

// TxPage is similarly re-exported.
type TxPage = db.TxPage

func (a *App) ListTransactions(filter TxFilter) (TxPage, error) {
	if err := a.requireDB(); err != nil {
		return TxPage{}, err
	}
	return a.db.ListTransactions(filter)
}

func (a *App) GetTransaction(id int64) (*models.Transaction, error) {
	if err := a.requireDB(); err != nil {
		return nil, err
	}
	return a.db.GetTransaction(id)
}

func (a *App) CreateTransaction(t models.Transaction) (models.Transaction, error) {
	if err := a.requireDB(); err != nil {
		return t, err
	}
	if err := a.db.CreateTransaction(&t); err != nil {
		return t, err
	}
	// Keep account balance in sync.
	if err := a.db.RecalcBalance(t.AccountID); err != nil {
		return t, err
	}
	return t, nil
}

func (a *App) UpdateTransaction(t models.Transaction) error {
	if err := a.requireDB(); err != nil {
		return err
	}
	// Fetch old to know which account might need rebalancing.
	old, err := a.db.GetTransaction(t.ID)
	if err != nil {
		// Can't determine old account; proceed and recalc the new account only.
		old = nil
	}
	if err := a.db.UpdateTransaction(&t); err != nil {
		return err
	}
	var errs []error
	if err := a.db.RecalcBalance(t.AccountID); err != nil {
		errs = append(errs, err)
	}
	if old != nil && old.AccountID != t.AccountID {
		if err := a.db.RecalcBalance(old.AccountID); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (a *App) DeleteTransaction(id int64) error {
	if err := a.requireDB(); err != nil {
		return err
	}
	t, err := a.db.GetTransaction(id)
	if err != nil {
		// Can't determine account; proceed with deletion, skip recalc.
		t = nil
	}
	if err := a.db.DeleteTransaction(id); err != nil {
		return err
	}
	if t != nil {
		if err := a.db.RecalcBalance(t.AccountID); err != nil {
			return err
		}
	}
	return nil
}
