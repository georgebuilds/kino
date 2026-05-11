package importer

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"
)

// ParseOFX reads an OFX or QFX file (both v1 SGML and v2 XML) and returns
// rows ready for BulkInsert.
//
// accountID is the Kino account the user chose.
// accountExternalID is the <ACCTID> from the file (used in the hash so that
// the same FITID from two different bank accounts never collide).
//
// Per-transaction parse failures (missing FITID, bad date, etc.) are collected
// into the warnings slice; structural failures (missing <OFX> element, XML
// parse error) return an error.
func ParseOFX(r io.Reader, accountID int64) ([]Row, string, []string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, "", nil, err
	}
	s := string(data)

	if isOFXv2(s) {
		return parseOFXv2([]byte(s), accountID)
	}
	return parseOFXv1(s, accountID)
}

func isOFXv2(s string) bool {
	trimmed := strings.TrimSpace(s)
	return strings.HasPrefix(trimmed, "<?xml") ||
		strings.HasPrefix(trimmed, "<OFX>") ||
		strings.Contains(s[:min(200, len(s))], "<?OFX")
}

// ── OFX v1 SGML parser ────────────────────────────────────────────────────────
// OFX 1.x is SGML: tags don't need closing tags, values follow on the same line
// as the opening tag:  <TRNAMT>-45.23

func parseOFXv1(s string, accountID int64) ([]Row, string, []string, error) {
	bodyStart := strings.Index(s, "<OFX>")
	if bodyStart == -1 {
		bodyStart = strings.Index(s, "<ofx>")
	}
	if bodyStart == -1 {
		return nil, "", nil, fmt.Errorf("ofx: cannot find <OFX> element")
	}
	body := s[bodyStart:]

	tags := tokeniseOFX1(body)

	acctID := findTag(tags, "ACCTID")
	if acctID == "" {
		acctID = fmt.Sprintf("account-%d", accountID)
	}

	var rows []Row
	var warnings []string
	inTx := false
	txIdx := 0
	var cur ofxRow

	for _, tok := range tags {
		switch tok.tag {
		case "STMTTRN":
			inTx = true
			cur = ofxRow{}
		case "/STMTTRN":
			if inTx {
				txIdx++
				row, err := cur.toRow(accountID, acctID)
				if err != nil {
					if len(warnings) < maxWarnings {
						label := cur.fitid
						if label == "" {
							label = fmt.Sprintf("#%d", txIdx)
						}
						warnings = append(warnings, fmt.Sprintf("transaction %s: %v", label, err))
					}
				} else {
					rows = append(rows, row)
				}
				inTx = false
			}
		default:
			if inTx {
				cur.apply(tok.tag, tok.val)
			}
		}
	}
	return rows, acctID, warnings, nil
}

type ofxToken struct{ tag, val string }

func tokeniseOFX1(s string) []ofxToken {
	var tokens []ofxToken
	i := 0
	for i < len(s) {
		lt := strings.IndexByte(s[i:], '<')
		if lt == -1 {
			break
		}
		start := i + lt
		gt := strings.IndexByte(s[start:], '>')
		if gt == -1 {
			break
		}
		tagEnd := start + gt
		tag := strings.ToUpper(strings.TrimSpace(s[start+1 : tagEnd]))
		// Find the value end: next '<' or newline
		valStart := tagEnd + 1
		valEnd := len(s)
		if nl := strings.IndexAny(s[valStart:], "<\r\n"); nl != -1 {
			valEnd = valStart + nl
		}
		val := strings.TrimSpace(s[valStart:valEnd])
		if tag != "" {
			tokens = append(tokens, ofxToken{tag: tag, val: val})
		}
		i = valEnd
	}
	return tokens
}

func findTag(tokens []ofxToken, tag string) string {
	for _, t := range tokens {
		if t.tag == tag {
			return t.val
		}
	}
	return ""
}

type ofxRow struct {
	fitid    string
	dtPosted string
	trnAmt   string
	name     string
	memo     string
}

func (o *ofxRow) apply(tag, val string) {
	switch tag {
	case "FITID":
		o.fitid = val
	case "DTPOSTED", "DTUSER":
		if o.dtPosted == "" {
			o.dtPosted = val
		}
	case "TRNAMT":
		o.trnAmt = val
	case "NAME":
		o.name = val
	case "MEMO":
		o.memo = val
	}
}

func (o *ofxRow) toRow(accountID int64, acctExtID string) (Row, error) {
	if o.fitid == "" {
		return Row{}, fmt.Errorf("missing FITID")
	}
	date, err := parseOFXDate(o.dtPosted)
	if err != nil {
		return Row{}, fmt.Errorf("bad DTPOSTED %q: %w", o.dtPosted, err)
	}
	cents, err := ParseAmount(o.trnAmt)
	if err != nil {
		return Row{}, err
	}
	payee := strings.TrimSpace(o.name)
	if payee == "" {
		payee = strings.TrimSpace(o.memo)
	}
	notes := ""
	if o.memo != "" && o.memo != payee {
		notes = strings.TrimSpace(o.memo)
	}
	norm := NormalizePayee(payee)
	hash := HashOFX(acctExtID, o.fitid)

	return Row{
		Date:            date,
		AmountCents:     cents,
		Payee:           payee,
		PayeeNormalized: norm,
		Notes:           notes,
		ImportHash:      hash,
		ImportSource:    string(SourceOFX),
	}, nil
}

// parseOFXDate handles OFX date formats:
// YYYYMMDDHHMMSS, YYYYMMDDHHMMSS.XXX, YYYYMMDD[TZ], YYYYMMDD
func parseOFXDate(s string) (string, error) {
	s = strings.TrimSpace(s)
	// Strip timezone bracket: "20250115120000[-5:EST]" → "20250115120000"
	if i := strings.Index(s, "["); i != -1 {
		s = s[:i]
	}
	s = strings.TrimSpace(s)
	// Take first 8 characters = YYYYMMDD
	if len(s) < 8 {
		return "", fmt.Errorf("date too short: %q", s)
	}
	t, err := time.Parse("20060102", s[:8])
	if err != nil {
		return "", err
	}
	return t.Format("2006-01-02"), nil
}

// ── OFX v2 XML parser ─────────────────────────────────────────────────────────

type ofxXML struct {
	XMLName xml.Name    `xml:"OFX"`
	BankMsg bankMsgXML  `xml:"BANKMSGSRSV1"`
	CCMsg   ccMsgXML    `xml:"CREDITCARDMSGSRSV1"`
}

type bankMsgXML struct {
	Stmts []stmtXML `xml:"STMTTRNRS>STMTRS"`
}

type ccMsgXML struct {
	Stmts []stmtXML `xml:"CCSTMTTRNRS>CCSTMTRS"`
}

type stmtXML struct {
	AcctID string    `xml:"BANKACCTFROM>ACCTID"`
	Txs    []txXML   `xml:"BANKTRANLIST>STMTTRN"`
}

type txXML struct {
	FITID    string `xml:"FITID"`
	DtPosted string `xml:"DTPOSTED"`
	TrnAmt   string `xml:"TRNAMT"`
	Name     string `xml:"NAME"`
	Memo     string `xml:"MEMO"`
}

func parseOFXv2(data []byte, accountID int64) ([]Row, string, []string, error) {
	s := string(data)
	if i := strings.Index(s, "<OFX>"); i > 0 {
		data = []byte(s[i:])
	}

	var doc ofxXML
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, "", nil, fmt.Errorf("ofx xml: %w", err)
	}

	stmts := append(doc.BankMsg.Stmts, doc.CCMsg.Stmts...)
	if len(stmts) == 0 {
		return nil, "", nil, fmt.Errorf("ofx: no statement found")
	}

	var rows []Row
	var warnings []string
	var firstAcctExtID string
	for _, stmt := range stmts {
		acctExtID := stmt.AcctID
		if acctExtID == "" {
			acctExtID = fmt.Sprintf("account-%d", accountID)
		}
		if firstAcctExtID == "" {
			firstAcctExtID = acctExtID
		}
		for _, tx := range stmt.Txs {
			o := ofxRow{
				fitid:    tx.FITID,
				dtPosted: tx.DtPosted,
				trnAmt:   tx.TrnAmt,
				name:     tx.Name,
				memo:     tx.Memo,
			}
			row, err := o.toRow(accountID, acctExtID)
			if err != nil {
				if len(warnings) < maxWarnings {
					label := tx.FITID
					if label == "" {
						label = "(no FITID)"
					}
					warnings = append(warnings, fmt.Sprintf("transaction %s: %v", label, err))
				}
				continue
			}
			rows = append(rows, row)
		}
	}
	return rows, firstAcctExtID, warnings, nil
}
