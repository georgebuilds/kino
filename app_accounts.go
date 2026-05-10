package main

import "kino/internal/models"

func (a *App) ListAccounts() ([]models.Account, error) {
	if err := a.requireDB(); err != nil {
		return nil, err
	}
	accs, err := a.db.ListAccounts()
	if accs == nil {
		accs = []models.Account{}
	}
	return accs, err
}

func (a *App) CreateAccount(acc models.Account) (models.Account, error) {
	if err := a.requireDB(); err != nil {
		return acc, err
	}
	if err := a.db.CreateAccount(&acc); err != nil {
		return acc, err
	}
	return acc, nil
}

func (a *App) UpdateAccount(acc models.Account) error {
	if err := a.requireDB(); err != nil {
		return err
	}
	return a.db.UpdateAccount(&acc)
}

func (a *App) DeleteAccount(id int64) error {
	if err := a.requireDB(); err != nil {
		return err
	}
	return a.db.DeleteAccount(id)
}
