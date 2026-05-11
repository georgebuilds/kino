package importer

import (
	"strings"
	"testing"
)

func TestParseCSV_HappyPath(t *testing.T) {
	const accountID = int64(7)
	in := strings.NewReader(`Date,Description,Amount
2025-01-15,STARBUCKS #1234,-4.95
2025-01-16,Paycheck Acme Inc.,2500.00
2025-01-17,Walmart,-32.10
`)
	rows, warnings, err := ParseCSV(in, accountID)
	if err != nil {
		t.Fatalf("ParseCSV unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}

	want := []struct {
		date         string
		amount       int64
		payee        string
		normContains string
	}{
		{"2025-01-15", -495, "STARBUCKS #1234", "starbucks"},
		{"2025-01-16", 250000, "Paycheck Acme Inc.", "paycheck acme"},
		{"2025-01-17", -3210, "Walmart", "walmart"},
	}
	for i, w := range want {
		got := rows[i]
		if got.Date != w.date {
			t.Errorf("rows[%d].Date = %q, want %q", i, got.Date, w.date)
		}
		if got.AmountCents != w.amount {
			t.Errorf("rows[%d].AmountCents = %d, want %d", i, got.AmountCents, w.amount)
		}
		if got.Payee != w.payee {
			t.Errorf("rows[%d].Payee = %q, want %q", i, got.Payee, w.payee)
		}
		if !strings.Contains(got.PayeeNormalized, w.normContains) {
			t.Errorf("rows[%d].PayeeNormalized = %q, want it to contain %q", i, got.PayeeNormalized, w.normContains)
		}
		if got.ImportHash == "" {
			t.Errorf("rows[%d].ImportHash is empty", i)
		}
		if got.ImportSource != string(SourceCSV) {
			t.Errorf("rows[%d].ImportSource = %q, want %q", i, got.ImportSource, SourceCSV)
		}
	}
}

func TestParseCSV_DebitCreditSplit(t *testing.T) {
	in := strings.NewReader(`Date,Description,Debit,Credit
2025-01-15,ATM Withdrawal,40.00,
2025-01-16,Direct Deposit,,1500.00
`)
	rows, warnings, err := ParseCSV(in, 1)
	if err != nil {
		t.Fatalf("ParseCSV unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	if rows[0].AmountCents != -4000 {
		t.Errorf("debit row: AmountCents = %d, want -4000 (money out → negative)", rows[0].AmountCents)
	}
	if rows[1].AmountCents != 150000 {
		t.Errorf("credit row: AmountCents = %d, want 150000 (money in → positive)", rows[1].AmountCents)
	}
}

func TestParseCSV_DebitAndCreditOnSameRow_Warns(t *testing.T) {
	in := strings.NewReader(`Date,Description,Debit,Credit
2025-01-15,Good Row,40.00,
2025-01-16,Weird Row,10.00,20.00
2025-01-17,Another Good Row,,99.00
`)
	rows, warnings, err := ParseCSV(in, 1)
	if err != nil {
		t.Fatalf("ParseCSV unexpected error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected the two valid rows to import, got %d rows: %+v", len(rows), rows)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected one warning, got %d: %v", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0], "row 3") {
		t.Errorf("expected warning to mention 'row 3', got: %v", warnings[0])
	}
	if !strings.Contains(warnings[0], "debit") || !strings.Contains(warnings[0], "credit") {
		t.Errorf("expected warning to mention both debit and credit, got: %v", warnings[0])
	}
}

func TestParseCSV_NoHeader_Errors(t *testing.T) {
	// No recognised header keywords here at all.
	in := strings.NewReader(`foo,bar,baz
1,2,3
`)
	_, _, err := ParseCSV(in, 1)
	if err == nil {
		t.Fatal("expected error for CSV with no header row, got nil")
	}
	if !strings.Contains(err.Error(), "could not find a header row") {
		t.Errorf("expected 'could not find a header row' in error, got: %v", err)
	}
}

func TestParseCSV_BadDate_SkipsRowWithWarning(t *testing.T) {
	// Header on line 1, good data row on line 2, bad data row on line 3.
	// Warning line number = headerIdx + lineNo + 2 = 0 + 1 + 2 = 3.
	in := strings.NewReader(`Date,Description,Amount
2025-01-15,Good Row,-4.95
not-a-date,Bad Row,-10.00
2025-01-17,Another Good Row,-2.50
`)
	rows, warnings, err := ParseCSV(in, 1)
	if err != nil {
		t.Fatalf("ParseCSV unexpected error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 valid rows, got %d", len(rows))
	}
	if len(warnings) != 1 {
		t.Fatalf("expected one warning, got %d: %v", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0], "row 3") {
		t.Errorf("expected warning to mention 'row 3', got: %v", warnings[0])
	}
}

func TestDetectColumns_AmbiguousNames(t *testing.T) {
	// Both "Transaction Date" and "Post Date" are date kinds. The first match wins.
	headers := []string{"Transaction Date", "Post Date", "Description", "Amount"}
	cm, err := detectColumns(headers)
	if err != nil {
		t.Fatalf("detectColumns unexpected error: %v", err)
	}
	if cm.date != 0 {
		t.Errorf("expected date column index = 0 (Transaction Date), got %d", cm.date)
	}
	if cm.payee != 2 {
		t.Errorf("expected payee column index = 2 (Description), got %d", cm.payee)
	}
	if cm.amount != 3 {
		t.Errorf("expected amount column index = 3 (Amount), got %d", cm.amount)
	}
}

func TestLooksLikeHeader(t *testing.T) {
	tests := []struct {
		name string
		rec  []string
		want bool
	}{
		{"contains Date", []string{"Date", "Description", "Amount"}, true},
		{"case-insensitive 'AMOUNT'", []string{"foo", "AMOUNT"}, true},
		{"trailing whitespace tolerated", []string{"  date  ", "x"}, true},
		{"data row with values", []string{"2025-01-15", "Starbucks", "-4.95"}, false},
		{"only unknown columns", []string{"foo", "bar", "baz"}, false},
		{"empty record", []string{}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := looksLikeHeader(tc.rec)
			if got != tc.want {
				t.Errorf("looksLikeHeader(%v) = %v, want %v", tc.rec, got, tc.want)
			}
		})
	}
}
