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

// ParseCSV reads r as a bank CSV export and returns rows ready for BulkInsert.
// accountID is used when computing the content hash.
func ParseCSV(r io.Reader, accountID int64) ([]Row, error) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true
	cr.LazyQuotes = true

	// Read until we find a header row (skip blank lines and comment lines)
	var headers []string
	var headerIdx int
	var allRecords [][]string

	records, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("csv read: %w", err)
	}

	for i, rec := range records {
		if looksLikeHeader(rec) {
			headers = rec
			headerIdx = i
			break
		}
	}
	if headers == nil {
		return nil, fmt.Errorf("csv: could not find a header row (expected columns like Date, Amount, Description)")
	}
	allRecords = records[headerIdx+1:]

	cm, err := detectColumns(headers)
	if err != nil {
		return nil, err
	}

	var rows []Row
	for lineNo, rec := range allRecords {
		if len(rec) == 0 || isBlankRow(rec) {
			continue
		}

		row, err := parseCSVRow(rec, cm, accountID)
		if err != nil {
			// Skip unparseable rows but don't abort the whole import
			_ = lineNo
			continue
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// looksLikeHeader returns true when a record contains at least one recognised
// column-kind keyword.
func looksLikeHeader(rec []string) bool {
	for _, cell := range rec {
		k := strings.ToLower(strings.TrimSpace(cell))
		if _, ok := knownHeaders[k]; ok {
			return true
		}
	}
	return false
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
		debitStr  := safeGet(rec, cm.debit)
		creditStr := safeGet(rec, cm.credit)

		if debitStr != "" && debitStr != "0" && debitStr != "0.00" {
			v, e := ParseAmount(debitStr)
			if e != nil {
				return Row{}, e
			}
			cents = -abs(v) // debit = money out = negative
		}
		if creditStr != "" && creditStr != "0" && creditStr != "0.00" {
			v, e := ParseAmount(creditStr)
			if e != nil {
				return Row{}, e
			}
			cents = abs(v) // credit = money in = positive
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
