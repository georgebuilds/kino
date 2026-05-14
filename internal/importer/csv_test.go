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

func TestIsBlankRow(t *testing.T) {
	tests := []struct {
		name string
		rec  []string
		want bool
	}{
		{"empty slice", []string{}, true},
		{"single empty string", []string{""}, true},
		{"whitespace-only cells", []string{"", " ", "\t"}, true},
		{"single non-blank cell", []string{"a"}, false},
		{"mixed: empty and non-blank", []string{"", "x"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isBlankRow(tc.rec)
			if got != tc.want {
				t.Errorf("isBlankRow(%v) = %v, want %v", tc.rec, got, tc.want)
			}
		})
	}
}

func TestSafeGet(t *testing.T) {
	rec := []string{"alpha", "  beta  ", "gamma"}

	// missing sentinel (-1) returns empty string.
	if got := safeGet(rec, missing); got != "" {
		t.Errorf("safeGet(rec, missing) = %q, want %q", got, "")
	}
	// index beyond length returns empty string.
	if got := safeGet(rec, 10); got != "" {
		t.Errorf("safeGet(rec, 10) = %q, want %q", got, "")
	}
	// valid index returns trimmed value.
	if got := safeGet(rec, 1); got != "beta" {
		t.Errorf("safeGet(rec, 1) = %q, want %q", got, "beta")
	}
	// first element (no trim needed).
	if got := safeGet(rec, 0); got != "alpha" {
		t.Errorf("safeGet(rec, 0) = %q, want %q", got, "alpha")
	}
}

func TestDetectColumns_NoPayee_Errors(t *testing.T) {
	headers := []string{"Date", "Amount"}
	_, err := detectColumns(headers)
	if err == nil {
		t.Fatal("expected error for headers with no payee/description column, got nil")
	}
	if !strings.Contains(err.Error(), "payee") && !strings.Contains(err.Error(), "description") {
		t.Errorf("expected error to mention payee/description, got: %v", err)
	}
}

func TestDetectColumns_NoDate_Errors(t *testing.T) {
	headers := []string{"Amount", "Description"}
	_, err := detectColumns(headers)
	if err == nil {
		t.Fatal("expected error for headers with no date column, got nil")
	}
	if !strings.Contains(err.Error(), "date") && !strings.Contains(err.Error(), "Date") {
		t.Errorf("expected error to mention date, got: %v", err)
	}
}

func TestDetectColumns_DebitOnly_OK(t *testing.T) {
	// No Amount column, no Credit column — only Debit. Should succeed.
	headers := []string{"Date", "Debit", "Description"}
	cm, err := detectColumns(headers)
	if err != nil {
		t.Fatalf("detectColumns unexpected error: %v", err)
	}
	if cm.amount != missing {
		t.Errorf("cm.amount = %d, want missing (%d)", cm.amount, missing)
	}
	if cm.debit != 1 {
		t.Errorf("cm.debit = %d, want 1", cm.debit)
	}
	if cm.credit != missing {
		t.Errorf("cm.credit = %d, want missing (%d)", cm.credit, missing)
	}
}

func TestDetectColumns_CreditOnly_OK(t *testing.T) {
	// No Amount column, no Debit column — only Credit. Should succeed.
	headers := []string{"Date", "Description", "Credit"}
	cm, err := detectColumns(headers)
	if err != nil {
		t.Fatalf("detectColumns unexpected error: %v", err)
	}
	if cm.amount != missing {
		t.Errorf("cm.amount = %d, want missing (%d)", cm.amount, missing)
	}
	if cm.debit != missing {
		t.Errorf("cm.debit = %d, want missing (%d)", cm.debit, missing)
	}
	if cm.credit != 2 {
		t.Errorf("cm.credit = %d, want 2", cm.credit)
	}
}

func TestParseCSV_NotesColumnPopulates(t *testing.T) {
	in := strings.NewReader(`Date,Description,Amount,Notes
2025-03-01,Grocery Store,-52.10,weekend shopping
`)
	rows, warnings, err := ParseCSV(in, 1)
	if err != nil {
		t.Fatalf("ParseCSV unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].Notes != "weekend shopping" {
		t.Errorf("Notes = %q, want %q", rows[0].Notes, "weekend shopping")
	}
}

func TestParseCSV_ReferenceColumnPopulatesNotes(t *testing.T) {
	in := strings.NewReader(`Date,Description,Amount,Reference
2025-03-01,Grocery Store,-52.10,REF-9876
`)
	rows, warnings, err := ParseCSV(in, 1)
	if err != nil {
		t.Fatalf("ParseCSV unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].Notes != "REF-9876" {
		t.Errorf("Notes = %q, want %q", rows[0].Notes, "REF-9876")
	}
}

func TestParseCSV_ZeroAmount_DebitCreditBothEmpty(t *testing.T) {
	// When both debit and credit columns are blank for a row, neither
	// debitCents nor creditCents is set, so AmountCents should be 0.
	// This is not an error — the row is produced with a zero amount.
	in := strings.NewReader(`Date,Description,Debit,Credit
2025-04-01,No-Movement Row,,
`)
	rows, warnings, err := ParseCSV(in, 1)
	if err != nil {
		t.Fatalf("ParseCSV unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings for zero-amount row, got %v", warnings)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].AmountCents != 0 {
		t.Errorf("AmountCents = %d, want 0", rows[0].AmountCents)
	}
}

func TestParseCSV_MaxWarningsCap(t *testing.T) {
	// Build a CSV with more than 25 rows that all have bad dates.
	// The warning slice must be capped at maxWarnings (20).
	var b strings.Builder
	b.WriteString("Date,Description,Amount\n")
	for i := 0; i < 30; i++ {
		b.WriteString("not-a-date,Bad Row,-1.00\n")
	}
	rows, warnings, err := ParseCSV(strings.NewReader(b.String()), 1)
	if err != nil {
		t.Fatalf("ParseCSV unexpected error: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("expected 0 valid rows, got %d", len(rows))
	}
	if len(warnings) != maxWarnings {
		t.Errorf("len(warnings) = %d, want %d (maxWarnings cap)", len(warnings), maxWarnings)
	}
}

func TestParseCSV_HeaderNotInFirstRow(t *testing.T) {
	// Some bank exports include preamble lines before the real header.
	// ParseCSV should scan for the first line that looks like a header.
	in := strings.NewReader(`Bank export,"Statement period: Jan 2025"
"Account: 1234",
,
Date,Description,Amount
2025-01-10,Coffee,-3.50
`)
	rows, warnings, err := ParseCSV(in, 1)
	if err != nil {
		t.Fatalf("ParseCSV unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].Date != "2025-01-10" {
		t.Errorf("Date = %q, want %q", rows[0].Date, "2025-01-10")
	}
	if rows[0].AmountCents != -350 {
		t.Errorf("AmountCents = %d, want -350", rows[0].AmountCents)
	}
}

func TestAbs(t *testing.T) {
	tests := []struct {
		name  string
		input int64
		want  int64
	}{
		{"zero", 0, 0},
		{"positive", 5, 5},
		{"negative", -5, 5},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := abs(tc.input)
			if got != tc.want {
				t.Errorf("abs(%d) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}

func TestLooksLikeHeader(t *testing.T) {
	tests := []struct {
		name string
		rec  []string
		want bool
	}{
		{"contains Date+Description+Amount", []string{"Date", "Description", "Amount"}, true},
		// Requires at least 2 keyword matches; a single match is not enough.
		{"single keyword match not enough", []string{"foo", "AMOUNT"}, false},
		// Two keyword matches, no numeric cells.
		{"two keyword matches", []string{"Date", "Amount"}, true},
		{"trailing whitespace tolerated", []string{"  date  ", "description"}, true},
		// A date-like cell (digits) causes rejection regardless of keyword count.
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
