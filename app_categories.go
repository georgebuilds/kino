package main

import "kino/internal/models"

func (a *App) ListCategories() ([]models.Category, error) {
	if err := a.requireDB(); err != nil {
		return nil, err
	}
	cats, err := a.db.ListCategories()
	if cats == nil {
		cats = []models.Category{}
	}
	return cats, err
}

func (a *App) CreateCategory(cat models.Category) (models.Category, error) {
	if err := a.requireDB(); err != nil {
		return cat, err
	}
	if err := a.db.CreateCategory(&cat); err != nil {
		return cat, err
	}
	return cat, nil
}

func (a *App) UpdateCategory(cat models.Category) error {
	if err := a.requireDB(); err != nil {
		return err
	}
	return a.db.UpdateCategory(&cat)
}

func (a *App) DeleteCategory(id int64) error {
	if err := a.requireDB(); err != nil {
		return err
	}
	return a.db.DeleteCategory(id)
}

func (a *App) GetCategoryTransactionCount(id int64) (int64, error) {
	if err := a.requireDB(); err != nil {
		return 0, err
	}
	return a.db.CountTransactionsByCategory(id)
}
