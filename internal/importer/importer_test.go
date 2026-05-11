package importer

import (
	"strings"
	"testing"
)

func TestParseAmount(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{"empty string returns zero, no error", "", 0, false},
		{"plain integer", "100", 10000, false},
		{"signed positive with dollar sign and comma", "$1,234.56", 123456, false},
		{"parens means negative", "(45.00)", -4500, false},
		{"leading minus", "-7.00", -700, false},
		{"euro symbol stripped", "€10", 1000, false},
		{"pound symbol stripped", "£10", 1000, false},
		{"single decimal digit padded", "5.5", 550, false},
		{"three+ decimals truncated to 2", "5.555", 555, false},
		{"EU comma-decimal with dot thousands", "1.234,56", 123456, false},
		// Note: production code only triggers EU mode when both '.' and ','
		// are present. A lone comma is treated as a thousands separator.
		{"lone comma treated as thousands separator", "1234,56", 12345600, false},
		{"pure junk errors", "abc", 0, true},
		{"pure punctuation errors", ".,", 0, true},
		{"negative with currency", "-$45.00", -4500, false},
		// Parentheses + currency symbol: both negative notation and $ stripping apply.
		{"parens with currency", "($45.00)", -4500, false},
		// Explicit zero string.
		{"explicit zero", "0", 0, false},
		// "1,56" has a comma but no dot, so the EU-detection branch (requires
		// both '.' and ',') is not entered. The else branch strips the comma,
		// leaving "156", which parses as 156 dollars = 15600 cents. This is
		// acknowledged as a known quirk (see TODO in importer memory).
		{"single-comma EU pure decimal treated as thousands-stripped", "1,56", 15600, false},
		// strings.Map strips all Unicode letters, so "USD" prefix is removed.
		{"multi-letter currency code stripped", "USD 12.00", 1200, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseAmount(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseAmount(%q) expected error, got nil (value %d)", tc.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseAmount(%q) unexpected error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("ParseAmount(%q) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"ISO format", "2025-01-15", "2025-01-15", false},
		{"US slash zero-padded", "01/15/2025", "2025-01-15", false},
		{"US slash non-padded", "1/2/2025", "2025-01-02", false},
		{"US dash zero-padded", "01-15-2025", "2025-01-15", false},
		{"slash YMD", "2025/01/15", "2025-01-15", false},
		{"DD/MM/YYYY", "31/12/2025", "2025-12-31", false},
		{"long month name", "January 2, 2025", "2025-01-02", false},
		{"short month name no zero", "Jan 2, 2025", "2025-01-02", false},
		{"short month name zero-padded", "Jan 02, 2025", "2025-01-02", false},
		{"day-first short month", "02 Jan 2025", "2025-01-02", false},
		{"not a date errors", "not-a-date", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseDate(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseDate(%q) expected error, got %q", tc.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseDate(%q) unexpected error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("ParseDate(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestNormalizePayee(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"store number stripped", "STARBUCKS #1234", "starbucks"},
		{"store number with trailing letter stripped", "STARBUCKS #1234A", "starbucks"},
		{"legal suffix Inc.", "Acme Inc.", "acme"},
		{"legal suffix LLC", "Acme LLC", "acme"},
		{"legal suffix Corp", "Acme Corp", "acme"},
		{"non-ASCII characters collapsed", "Café Roma", "caf  roma"}, // é replaced with space, collapsed via reSpaces => "caf roma"? Let's verify carefully.
		{"multi-space collapsed", "Hello    World", "hello world"},
		{"leading star stripped", "*WALMART", "walmart"},
		{"lower-casing", "MIXED Case Inc", "mixed case"},
		{"trim outer whitespace", "  Acme  ", "acme"},
		// reLegalSufx requires \s+ before the suffix, so "Acme Co." → "Acme" (space before Co. present).
		{"legal suffix Co.", "Acme Co.", "acme"},
		// Same pattern for Ltd.
		{"legal suffix Ltd.", "Acme Ltd.", "acme"},
		// strings.Trim removes leading/trailing chars in the set `*-_.,;"'`.
		// After other normalisation steps "Acme..." becomes "acme..." → Trim removes trailing dots.
		{"trailing punctuation trimmed", "Acme...", "acme"},
		// strings.Trim removes the leading '-' because '-' is in the trim cutset.
		{"leading dash trimmed", "-WALMART", "walmart"},
	}
	// Special case correction: reNonPrint replaces each non-ASCII byte
	// (é is 2 UTF-8 bytes) with one space each, then reSpaces collapses
	// runs of whitespace to a single space. So "Café Roma" → "Caf  Roma"
	// (2 spaces from 2 bytes) → "Caf Roma" → "caf roma" after lower.
	tests[5].want = "caf roma"

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizePayee(tc.input)
			if got != tc.want {
				t.Errorf("NormalizePayee(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestHashOFX_Stable(t *testing.T) {
	h1 := HashOFX("ACCT-A", "FITID-100")
	h2 := HashOFX("ACCT-A", "FITID-100")
	if h1 != h2 {
		t.Errorf("HashOFX not stable for identical input: %q vs %q", h1, h2)
	}

	hDifferentAccount := HashOFX("ACCT-B", "FITID-100")
	if hDifferentAccount == h1 {
		t.Errorf("HashOFX collided across accounts: %q == %q", hDifferentAccount, h1)
	}

	// Ensure OFX hash and CSV hash for "matching" semantic inputs never collide
	// (the source prefix in the hash input guarantees this).
	csvHash := HashCSV(1, "2025-01-15", -4500, "starbucks")
	ofxHash := HashOFX("ACCT-A", "FITID-100")
	if csvHash == ofxHash {
		t.Errorf("CSV hash and OFX hash collided: %q", csvHash)
	}

	// Spot-check: an OFX hash should not match a CSV hash built from
	// the OFX's own external account/FITID either.
	if HashCSV(0, "", 0, "FITID-100") == HashOFX("ACCT-A", "FITID-100") {
		t.Error("OFX and CSV hash spaces overlapped")
	}

	if !strings.HasPrefix(hashStr("ofx|ACCT-A|FITID-100"), "") {
		t.Skip("hashStr unavailable; not a real failure")
	}
}

func TestHashCSV_Stable(t *testing.T) {
	const acctID = int64(42)
	const date = "2025-01-15"
	const cents = int64(-1234)

	a := HashCSV(acctID, date, cents, "starbucks")
	b := HashCSV(acctID, date, cents, "starbucks")
	if a != b {
		t.Errorf("HashCSV not stable for identical inputs: %q vs %q", a, b)
	}

	// Normalisation collapses STARBUCKS #1 and STARBUCKS #99 to the same
	// normalised payee; if account+date+amount match, hashes must match too.
	norm1 := NormalizePayee("STARBUCKS #1")
	norm99 := NormalizePayee("STARBUCKS #99")
	if norm1 != norm99 {
		t.Fatalf("normalisation did not collapse store numbers: %q vs %q", norm1, norm99)
	}
	h1 := HashCSV(acctID, date, cents, norm1)
	h99 := HashCSV(acctID, date, cents, norm99)
	if h1 != h99 {
		t.Errorf("HashCSV differs for normalised store numbers: %q vs %q", h1, h99)
	}

	// Sanity: different amount → different hash
	hDiff := HashCSV(acctID, date, cents+1, norm1)
	if hDiff == h1 {
		t.Error("HashCSV did not change for different amount")
	}
}
