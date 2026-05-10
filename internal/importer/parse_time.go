package importer

import "time"

// parseWithTime wraps time.Parse and returns YYYY-MM-DD on success.
func parseWithTime(layout, value string) (string, error) {
	t, err := time.Parse(layout, value)
	if err != nil {
		return "", err
	}
	return t.Format("2006-01-02"), nil
}
