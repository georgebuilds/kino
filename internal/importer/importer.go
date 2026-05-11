// Package importer parses bank export files (CSV, OFX/QFX) into a common
// row format ready for BulkInsert. It also owns the hash strategy that
// prevents duplicates both within a source and across sources.
package importer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
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
	reNonPrint  = regexp.MustCompile(`[^\x20-\x7E]`)
)

// NormalizePayee strips store numbers, legal suffixes, and extra whitespace so
// that "STARBUCKS #1234" and "STARBUCKS #0099" hash identically.
func NormalizePayee(s string) string {
	s = reNonPrint.ReplaceAllString(s, " ")
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

var reNonNumeric = regexp.MustCompile(`[^0-9.\-]`)

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

	// Strip currency symbols and spaces
	s = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return -1
		}
		return r
	}, s)
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, "€", "")
	s = strings.ReplaceAll(s, "£", "")
	s = strings.TrimSpace(s)

	if strings.HasPrefix(s, "-") {
		negative = true
		s = s[1:]
	}

	// Detect EU format: 1.234,56 → 1234.56
	if strings.Count(s, ",") == 1 && strings.Count(s, ".") >= 1 {
		// comma is decimal separator when it comes last
		lastComma := strings.LastIndex(s, ",")
		lastDot := strings.LastIndex(s, ".")
		if lastComma > lastDot {
			s = strings.ReplaceAll(s, ".", "")
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

	// Parse as float → cents
	var whole, frac int64
	parts := strings.SplitN(s, ".", 2)
	if parts[0] != "" {
		if _, err := fmt.Sscanf(parts[0], "%d", &whole); err != nil {
			return 0, fmt.Errorf("parse amount %q: %w", original, err)
		}
	}
	if len(parts) == 2 {
		fracStr := parts[1]
		// Pad or truncate to 2 decimal places
		switch len(fracStr) {
		case 0:
			frac = 0
		case 1:
			if _, err := fmt.Sscanf(fracStr, "%d", &frac); err != nil {
				return 0, fmt.Errorf("parse amount %q: %w", original, err)
			}
			frac *= 10
		default:
			if _, err := fmt.Sscanf(fracStr[:2], "%d", &frac); err != nil {
				return 0, fmt.Errorf("parse amount %q: %w", original, err)
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
