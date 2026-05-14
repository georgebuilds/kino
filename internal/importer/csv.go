package importer

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

// csvColKind classifies what a CSV column represents.
type csvColKind int

const (
	colUnknown csvColKind = iota
	colDate
	colAmount
	colDebit  // positive = money out
	colCredit // positive = money in
	colPayee
	colNotes
)

// knownHeaders maps lower-cased header variants to a column kind.
var knownHeaders = map[string]csvColKind{
	// date
	"date":                 colDate,
	"transaction date":     colDate,
	"trans date":           colDate,
	"trans. date":          colDate,
	"post date":            colDate,
	"posted date":          colDate,
	"posting date":         colDate,
	"settlement date":      colDate,
	"value date":           colDate,
	// amount (signed)
	"amount":               colAmount,
	"transaction amount":   colAmount,
	"net amount":           colAmount,
	"amt":                  colAmount,
	// debit / credit split columns
	"debit":                colDebit,
	"debit amount":         colDebit,
	"withdrawals":          colDebit,
	"withdrawal":           colDebit,
	"money out":            colDebit,
	"credit":               colCredit,
	"credit amount":        colCredit,
	"deposits":             colCredit,
	"deposit":              colCredit,
	"money in":             colCredit,
	// payee
	"description":          colPayee,
	"transaction description": colPayee,
	"payee":                colPayee,
	"merchant":             colPayee,
	"name":                 colPayee,
	"memo":                 colPayee,
	"details":              colPayee,
	"particulars":          colPayee,
	// notes
	"notes":                colNotes,
	"reference":            colNotes,
	"check number":         colNotes,
	"cheque number":        colNotes,
}

// colMap holds resolved column indices for a CSV file.
type colMap struct {
	date   int
	amount int
	debit  int
	credit int
	payee  int
	notes  int
}

const missing = -1

// maxWarnings caps the warning slice so a totally bad file doesn't blow up memory.
const maxWarnings = 20

// ParseCSV reads r as a bank CSV export and returns rows ready for BulkInsert.
// accountID is used when computing the content hash.
//
// Hard structural problems (no header row, missing required column, I/O error)
// return an error. Per-row parse failures are collected into the warnings slice
// and the bad row is skipped; the caller surfaces these to the user.
func ParseCSV(r io.Reader, accountID int64) ([]Row, []string, error) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true
	cr.LazyQuotes = true
	cr.FieldsPerRecord = -1

	records, err := cr.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("csv read: %w", err)
	}

	var headers []string
	var headerIdx int
	for i, rec := range records {
		if looksLikeHeader(rec) {
			headers = rec
			headerIdx = i
			break
		}
	}
	if headers == nil {
		return nil, nil, fmt.Errorf("csv: could not find a header row (expected columns like Date, Amount, Description)")
	}

	cm, err := detectColumns(headers)
	if err != nil {
		return nil, nil, err
	}

	var rows []Row
	var warnings []string
	for lineNo, rec := range records[headerIdx+1:] {
		if len(rec) == 0 || isBlankRow(rec) {
			continue
		}

		row, err := parseCSVRow(rec, cm, accountID)
		if err != nil {
			if len(warnings) < maxWarnings {
				warnings = append(warnings, fmt.Sprintf("row %d: %v", headerIdx+lineNo+2, err))
			}
			continue
		}
		rows = append(rows, row)
	}
	return rows, warnings, nil
}

// looksLikeHeader returns true when a record contains at least 2 recognised
// column-kind keywords and no cell looks like a numeric data value. Requiring
// multiple keyword matches prevents data rows that happen to contain a word
// like "credit" from being mistaken for a header.
func looksLikeHeader(rec []string) bool {
	matches := 0
	for _, cell := range rec {
		k := strings.ToLower(strings.TrimSpace(cell))
		if _, ok := knownHeaders[k]; ok {
			matches++
		}
		// A cell that looks like a number (optional sign/currency, then digits)
		// is a strong signal that this is a data row, not a header.
		if looksNumeric(strings.TrimSpace(cell)) {
			return false
		}
	}
	return matches >= 2
}

// looksNumeric returns true for strings that are clearly numeric data values
// (digits, possibly preceded by an optional sign or currency symbol, and
// optionally containing decimal separators).
func looksNumeric(s string) bool {
	if s == "" {
		return false
	}
	// Strip common leading currency/sign characters.
	s = strings.TrimLeft(s, "$€£-+(")
	if s == "" {
		return false
	}
	// Must start with a digit after stripping.
	return s[0] >= '0' && s[0] <= '9'
}

func isBlankRow(rec []string) bool {
	for _, c := range rec {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

func detectColumns(headers []string) (colMap, error) {
	cm := colMap{
		date: missing, amount: missing, debit: missing,
		credit: missing, payee: missing, notes: missing,
	}
	for i, h := range headers {
		k := strings.ToLower(strings.TrimSpace(h))
		switch knownHeaders[k] {
		case colDate:
			if cm.date == missing {
				cm.date = i
			}
		case colAmount:
			if cm.amount == missing {
				cm.amount = i
			}
		case colDebit:
			if cm.debit == missing {
				cm.debit = i
			}
		case colCredit:
			if cm.credit == missing {
				cm.credit = i
			}
		case colPayee:
			if cm.payee == missing {
				cm.payee = i
			}
		case colNotes:
			if cm.notes == missing {
				cm.notes = i
			}
		}
	}

	if cm.date == missing {
		return cm, fmt.Errorf("csv: no date column found (expected: Date, Transaction Date, Post Date, …)")
	}
	if cm.amount == missing && (cm.debit == missing && cm.credit == missing) {
		return cm, fmt.Errorf("csv: no amount column found (expected: Amount, Debit, Credit, …)")
	}
	if cm.payee == missing {
		return cm, fmt.Errorf("csv: no payee/description column found (expected: Description, Payee, Memo, …)")
	}
	return cm, nil
}

func safeGet(rec []string, idx int) string {
	if idx == missing || idx >= len(rec) {
		return ""
	}
	return strings.TrimSpace(rec[idx])
}

func parseCSVRow(rec []string, cm colMap, accountID int64) (Row, error) {
	dateStr, err := ParseDate(safeGet(rec, cm.date))
	if err != nil {
		return Row{}, err
	}

	var cents int64
	if cm.amount != missing {
		cents, err = ParseAmount(safeGet(rec, cm.amount))
		if err != nil {
			return Row{}, err
		}
	} else {
		// Separate debit/credit columns
		debitStr := safeGet(rec, cm.debit)
		creditStr := safeGet(rec, cm.credit)

		var debitCents, creditCents int64
		if debitStr != "" {
			v, e := ParseAmount(debitStr)
			if e != nil {
				return Row{}, e
			}
			debitCents = v
		}
		if creditStr != "" {
			v, e := ParseAmount(creditStr)
			if e != nil {
				return Row{}, e
			}
			creditCents = v
		}

		if debitCents != 0 && creditCents != 0 {
			return Row{}, fmt.Errorf("row has both debit %q and credit %q", debitStr, creditStr)
		}
		switch {
		case debitCents != 0:
			// Debit column convention: positive values = money out (negate),
			// but preserve negative debit values as-is (credit reversals).
			if debitCents > 0 {
				cents = -debitCents
			} else {
				cents = debitCents
			}
		case creditCents != 0:
			cents = abs(creditCents) // credit = money in = positive
		}
	}

	payee := safeGet(rec, cm.payee)
	notes := safeGet(rec, cm.notes)
	norm  := NormalizePayee(payee)

	hash := HashCSV(accountID, dateStr, cents, norm)

	return Row{
		Date:            dateStr,
		AmountCents:     cents,
		Payee:           payee,
		PayeeNormalized: norm,
		Notes:           notes,
		ImportHash:      hash,
		ImportSource:    string(SourceCSV),
	}, nil
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
