// Package importer parses bank export files (CSV, OFX/QFX) into a common
// row format ready for BulkInsert. It also owns the hash strategy that
// prevents duplicates both within a source and across sources.
package importer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Source identifies where a transaction came from.
type Source string

const (
	SourceCSV    Source = "csv"
	SourceOFX    Source = "ofx"
	SourceManual Source = "manual"
)

// Row is a normalised transaction ready to be handed to db.BulkInsert.
// All monetary values are in cents (negative = expense, positive = income).
type Row struct {
	Date            string // YYYY-MM-DD
	AmountCents     int64
	Payee           string // raw payee from the file
	PayeeNormalized string // cleaned payee (store #s stripped, lowercased)
	Notes           string
	ImportHash      string
	ImportSource    string
}

// ── Payee normalisation ───────────────────────────────────────────────────────

var (
	reStoreNum  = regexp.MustCompile(`\s*#\s*\d+[a-zA-Z]?\s*$`)
	reLegalSufx = regexp.MustCompile(`(?i)\s+(llc|inc\.?|corp\.?|co\.?|ltd\.?)\s*$`)
	reSpaces    = regexp.MustCompile(`\s+`)
)

// NormalizePayee strips store numbers, legal suffixes, and extra whitespace so
// that "STARBUCKS #1234" and "STARBUCKS #0099" hash identically.
func NormalizePayee(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return ' '
	}, s)
	s = reStoreNum.ReplaceAllString(s, "")
	s = reLegalSufx.ReplaceAllString(s, "")
	s = reSpaces.ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)
	// Remove leading/trailing punctuation that some banks wrap around payees
	s = strings.Trim(s, `*-_.,;"'`)
	return strings.ToLower(s)
}

// ── Hash helpers ──────────────────────────────────────────────────────────────

// HashOFX produces a stable hash for an OFX transaction using the bank's own
// FITID.  The "ofx|" prefix ensures it never collides with a CSV hash.
func HashOFX(accountExternalID, fitid string) string {
	return hashStr(fmt.Sprintf("ofx|%s|%s", accountExternalID, fitid))
}

// HashCSV produces a content-based hash for a CSV row.
// Normalising the payee means the same transaction survives minor bank
// reformatting across exports.
func HashCSV(accountID int64, date string, amountCents int64, normalizedPayee string) string {
	return hashStr(fmt.Sprintf("csv|%d|%s|%d|%s", accountID, date, amountCents, normalizedPayee))
}

func hashStr(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// ── Amount parsing ────────────────────────────────────────────────────────────

func isAllLetters(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

// ParseAmount converts a bank amount string to cents.
// Handles: "$1,234.56", "(45.00)", "-45.00", "1234,56" (EU comma-decimal).
func ParseAmount(s string) (int64, error) {
	original := s
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}

	negative := false
	// Parentheses = negative (accounting notation)
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		negative = true
		s = s[1 : len(s)-1]
	}

	// Strip known currency symbols and letter-only currency codes (e.g. "USD").
	// Letters are stripped only when they form a word boundary group (all-letter
	// token separated by spaces from the digits), so "123abc" is not silently
	// accepted — the letters remain and will cause a parse error below.
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, "€", "")
	s = strings.ReplaceAll(s, "£", "")
	s = strings.TrimSpace(s)
	// Strip a leading or trailing all-letter token (currency code like "USD", "EUR").
	if fields := strings.Fields(s); len(fields) > 1 {
		if isAllLetters(fields[0]) {
			s = strings.TrimSpace(s[len(fields[0]):])
		} else if isAllLetters(fields[len(fields)-1]) {
			s = strings.TrimSpace(s[:len(s)-len(fields[len(fields)-1])])
		}
	}
	s = strings.TrimSpace(s)

	if strings.HasPrefix(s, "-") {
		negative = true
		s = s[1:]
	}

	commas := strings.Count(s, ",")
	dots := strings.Count(s, ".")

	if commas == 1 && dots >= 1 {
		// Both separators present: comma is decimal when it comes after the last dot.
		lastComma := strings.LastIndex(s, ",")
		lastDot := strings.LastIndex(s, ".")
		if lastComma > lastDot {
			s = strings.ReplaceAll(s, ".", "")
			s = strings.ReplaceAll(s, ",", ".")
		} else {
			s = strings.ReplaceAll(s, ",", "")
		}
	} else if commas == 1 && dots == 0 {
		// Lone comma: treat as decimal separator only when the part after the
		// comma has 1-2 digits (EU format "1234,56" = $1234.56).
		// Three or more digits after the comma means it is a thousands separator
		// ("1,234" → 1234).
		afterComma := s[strings.Index(s, ",")+1:]
		if len(afterComma) <= 2 {
			s = strings.ReplaceAll(s, ",", ".")
		} else {
			s = strings.ReplaceAll(s, ",", "")
		}
	} else {
		s = strings.ReplaceAll(s, ",", "")
	}

	if strings.Trim(s, "-.,") == "" {
		return 0, fmt.Errorf("parse amount %q: empty after stripping", original)
	}

	// Parse as fixed-point → cents
	var whole, frac int64
	parts := strings.SplitN(s, ".", 2)
	if parts[0] != "" {
		v, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parse amount %q: %w", original, err)
		}
		whole = v
	}
	if whole > math.MaxInt64/100 {
		return 0, fmt.Errorf("parse amount %q: value overflows int64 cents", original)
	}
	if len(parts) == 2 {
		fracStr := parts[1]
		switch {
		case len(fracStr) == 0:
			frac = 0
		case len(fracStr) == 1:
			v, err := strconv.ParseInt(fracStr, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("parse amount %q: %w", original, err)
			}
			frac = v * 10
		default:
			// Round: look at the third digit (if present).
			v, err := strconv.ParseInt(fracStr[:2], 10, 64)
			if err != nil {
				return 0, fmt.Errorf("parse amount %q: %w", original, err)
			}
			frac = v
			if len(fracStr) > 2 {
				third, err2 := strconv.ParseInt(string(fracStr[2]), 10, 64)
				if err2 == nil && third >= 5 {
					frac++
				}
			}
		}
	}

	cents := whole*100 + frac
	if negative {
		cents = -cents
	}
	return cents, nil
}

// ParseDate tries a set of common date formats and returns YYYY-MM-DD.
//
// Slash-separated dates are inherently ambiguous when both the first and second
// components are ≤12 (e.g. "06/07/2025" could be June 7 or July 6). US format
// (MM/DD/YYYY) is tried first and wins in that case. When the first component
// is >12 it cannot be a month, so the DD/MM/YYYY format will match instead.
func ParseDate(s string) (string, error) {
	s = strings.TrimSpace(s)
	formats := []string{
		"2006-01-02",
		"01/02/2006",
		"1/2/2006",
		"01-02-2006",
		"2006/01/02",
		"02/01/2006", // DD/MM/YYYY — tried last to favour US format
		"January 2, 2006",
		"Jan 2, 2006",
		"Jan 02, 2006",
		"02 Jan 2006",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t.Format("2006-01-02"), nil
		}
	}
	return "", fmt.Errorf("unrecognised date %q", s)
}
