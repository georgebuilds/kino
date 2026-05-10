package main

import "kino/internal/db"

// GetNetWorthHistory returns monthly net-worth snapshots.
// months: number of calendar months to return (12 or 24; defaults to 12).
func (a *App) GetNetWorthHistory(months int) ([]db.NetWorthPoint, error) {
	if err := a.requireDB(); err != nil {
		return nil, err
	}
	return a.db.GetNetWorthHistory(months)
}
