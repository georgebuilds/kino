package db

import "time"

// mustDate returns a UTC time at midnight for the given Y/M/D. Production code
// formats Transaction.Date to "YYYY-MM-DD" on write, so the hour-of-day is
// irrelevant; using UTC midnight keeps tests reproducible.
func mustDate(y, m, d int) time.Time {
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
}
