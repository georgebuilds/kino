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
		{"three+ decimals rounded", "5.555", 556, false},
		{"three+ decimals rounded down", "5.554", 555, false},
		{"EU comma-decimal with dot thousands", "1.234,56", 123456, false},
		// Lone comma with 2 digits after → EU decimal separator.
		{"lone comma EU decimal 2 digits", "1234,56", 123456, false},
		// Lone comma with 1 digit after → EU decimal separator.
		{"lone comma EU decimal 1 digit", "1234,5", 123450, false},
		// Lone comma with 3+ digits after → thousands separator.
		{"lone comma thousands separator", "1,234", 123400, false},
		{"pure junk errors", "abc", 0, true},
		{"trailing garbage errors", "123abc", 0, true},
		{"pure punctuation errors", ".,", 0, true},
		{"negative with currency", "-$45.00", -4500, false},
		// Parentheses + currency symbol: both negative notation and $ stripping apply.
		{"parens with currency", "($45.00)", -4500, false},
		// Explicit zero string.
		{"explicit zero", "0", 0, false},
		// Lone comma with 2 digits, small number.
		{"single-comma EU small number", "1,56", 156, false},
		// strings.Map strips all Unicode letters, so "USD" prefix is removed.
		{"multi-letter currency code stripped", "USD 12.00", 1200, false},
		// Overflow guard.
		{"overflow value errors", "99999999999999999999", 0, true},
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

func TestParseDate_AmbiguousDate_PrefersUSFormat(t *testing.T) {
	// "06/07/2025" is ambiguous: US MM/DD/YYYY reads it as June 7, while
	// DD/MM/YYYY reads it as July 6. The formats slice in ParseDate tries
	// "01/02/2006" (US) before "02/01/2006" (DD/MM), so the US interpretation
	// must win and the returned date must be 2025-06-07.
	got, err := ParseDate("06/07/2025")
	if err != nil {
		t.Fatalf("ParseDate(%q) unexpected error: %v", "06/07/2025", err)
	}
	if got != "2025-06-07" {
		t.Errorf("ParseDate(%q) = %q, want %q (US MM/DD wins over DD/MM)", "06/07/2025", got, "2025-06-07")
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
		{"non-ASCII printable characters preserved", "Café Roma", "café roma"},
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
