package importer

import (
	"strings"
	"testing"
)

// ── Minimal OFX 1.x SGML body ────────────────────────────────────────────────

const ofxV1Minimal = `OFXHEADER:100
DATA:OFXSGML
VERSION:102
SECURITY:NONE
ENCODING:USASCII
CHARSET:1252
COMPRESSION:NONE
OLDFILEUID:NONE
NEWFILEUID:NONE

<OFX>
<BANKMSGSRSV1>
<STMTTRNRS>
<STMTRS>
<BANKACCTFROM>
<ACCTID>123456789
</BANKACCTFROM>
<BANKTRANLIST>
<STMTTRN>
<TRNTYPE>DEBIT
<DTPOSTED>20250115120000
<TRNAMT>-45.23
<FITID>FIT-1
<NAME>STARBUCKS #1234
<MEMO>Coffee
</STMTTRN>
</BANKTRANLIST>
</STMTRS>
</STMTTRNRS>
</BANKMSGSRSV1>
</OFX>
`

func TestParseOFX_V1_Minimal(t *testing.T) {
	rows, acctExtID, warnings, err := ParseOFX(strings.NewReader(ofxV1Minimal), 1)
	if err != nil {
		t.Fatalf("ParseOFX unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if acctExtID != "123456789" {
		t.Errorf("acctExtID = %q, want %q", acctExtID, "123456789")
	}
	r := rows[0]
	if r.Date != "2025-01-15" {
		t.Errorf("Date = %q, want %q", r.Date, "2025-01-15")
	}
	if r.AmountCents != -4523 {
		t.Errorf("AmountCents = %d, want %d", r.AmountCents, -4523)
	}
	if r.Payee != "STARBUCKS #1234" {
		t.Errorf("Payee = %q, want %q", r.Payee, "STARBUCKS #1234")
	}
	if r.ImportSource != string(SourceOFX) {
		t.Errorf("ImportSource = %q, want %q", r.ImportSource, SourceOFX)
	}
	if r.ImportHash == "" {
		t.Error("ImportHash is empty")
	}
}

func TestParseOFX_V1_LongSingleLine(t *testing.T) {
	// Build the OFX 1.x payload entirely on a single line. This regression-
	// tests that tokeniseOFX1 walks '<'/'>' directly instead of using
	// bufio.Scanner (which has a default 64KiB line limit).
	var b strings.Builder
	b.WriteString("OFXHEADER:100 DATA:OFXSGML VERSION:102 ")
	b.WriteString("<OFX>")
	b.WriteString("<BANKMSGSRSV1><STMTTRNRS><STMTRS>")
	b.WriteString("<BANKACCTFROM><ACCTID>987654321</BANKACCTFROM>")
	b.WriteString("<BANKTRANLIST>")

	const txCount = 50
	for i := 0; i < txCount; i++ {
		// Each STMTTRN is on the same line (no newlines).
		b.WriteString("<STMTTRN>")
		b.WriteString("<TRNTYPE>DEBIT")
		b.WriteString("<DTPOSTED>20250101")
		b.WriteString("<TRNAMT>-1.00")
		b.WriteString("<FITID>FIT-")
		// Distinct FITIDs so all hashes differ.
		b.WriteString(itoa(i))
		b.WriteString("<NAME>Vendor ")
		b.WriteString(itoa(i))
		b.WriteString("</STMTTRN>")
	}
	// Add some padding so the line is comfortably larger than the
	// default 64KiB scanner buffer.
	pad := strings.Repeat("X", 70_000)
	b.WriteString("<PAD>")
	b.WriteString(pad)
	b.WriteString("</PAD>")
	b.WriteString("</BANKTRANLIST></STMTRS></STMTTRNRS></BANKMSGSRSV1></OFX>")

	rows, acctExtID, warnings, err := ParseOFX(strings.NewReader(b.String()), 1)
	if err != nil {
		t.Fatalf("ParseOFX unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if acctExtID != "987654321" {
		t.Errorf("acctExtID = %q, want %q", acctExtID, "987654321")
	}
	if len(rows) != txCount {
		t.Fatalf("expected %d rows from long single-line OFX, got %d", txCount, len(rows))
	}
	// Verify each hash is unique (i.e. each FITID was actually distinct).
	seen := make(map[string]struct{}, txCount)
	for _, r := range rows {
		seen[r.ImportHash] = struct{}{}
	}
	if len(seen) != txCount {
		t.Errorf("expected %d distinct hashes, got %d (collision in long-line parse)", txCount, len(seen))
	}
}

// itoa avoids pulling in strconv just for two callers and keeps tests dep-free
// (we only need small non-negative ints).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// ── Minimal OFX 2.x XML body ─────────────────────────────────────────────────

const ofxV2Minimal = `<?xml version="1.0" encoding="UTF-8"?>
<?OFX OFXHEADER="200" VERSION="200"?>
<OFX>
  <BANKMSGSRSV1>
    <STMTTRNRS>
      <STMTRS>
        <BANKACCTFROM><ACCTID>111222333</ACCTID></BANKACCTFROM>
        <BANKTRANLIST>
          <STMTTRN>
            <TRNTYPE>DEBIT</TRNTYPE>
            <DTPOSTED>20250115120000</DTPOSTED>
            <TRNAMT>-12.34</TRNAMT>
            <FITID>FIT-V2-1</FITID>
            <NAME>Amazon</NAME>
            <MEMO>order #1</MEMO>
          </STMTTRN>
        </BANKTRANLIST>
      </STMTRS>
    </STMTTRNRS>
  </BANKMSGSRSV1>
</OFX>
`

func TestParseOFX_V2_Minimal(t *testing.T) {
	rows, acctExtID, warnings, err := ParseOFX(strings.NewReader(ofxV2Minimal), 1)
	if err != nil {
		t.Fatalf("ParseOFX unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if acctExtID != "111222333" {
		t.Errorf("acctExtID = %q, want %q", acctExtID, "111222333")
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	r := rows[0]
	if r.Date != "2025-01-15" {
		t.Errorf("Date = %q, want %q", r.Date, "2025-01-15")
	}
	if r.AmountCents != -1234 {
		t.Errorf("AmountCents = %d, want %d", r.AmountCents, -1234)
	}
	if r.Payee != "Amazon" {
		t.Errorf("Payee = %q, want %q", r.Payee, "Amazon")
	}
	if r.ImportSource != string(SourceOFX) {
		t.Errorf("ImportSource = %q, want %q", r.ImportSource, SourceOFX)
	}
	if r.ImportHash == "" {
		t.Error("ImportHash is empty")
	}
}

func TestParseOFX_V2_MultipleStatementsDistinctACCTIDs(t *testing.T) {
	// Two statements: one bank, one credit card. Both have a transaction
	// with FITID="100". Before the per-statement ACCTID fix, both rows
	// would hash to the same value (using stmts[0].AcctID for everything).
	const input = `<?xml version="1.0" encoding="UTF-8"?>
<OFX>
  <BANKMSGSRSV1>
    <STMTTRNRS>
      <STMTRS>
        <BANKACCTFROM><ACCTID>BANK-AAA</ACCTID></BANKACCTFROM>
        <BANKTRANLIST>
          <STMTTRN>
            <DTPOSTED>20250115</DTPOSTED>
            <TRNAMT>-10.00</TRNAMT>
            <FITID>100</FITID>
            <NAME>Bank Tx</NAME>
          </STMTTRN>
        </BANKTRANLIST>
      </STMTRS>
    </STMTTRNRS>
  </BANKMSGSRSV1>
  <CREDITCARDMSGSRSV1>
    <CCSTMTTRNRS>
      <CCSTMTRS>
        <BANKACCTFROM><ACCTID>CARD-BBB</ACCTID></BANKACCTFROM>
        <BANKTRANLIST>
          <STMTTRN>
            <DTPOSTED>20250116</DTPOSTED>
            <TRNAMT>-20.00</TRNAMT>
            <FITID>100</FITID>
            <NAME>Card Tx</NAME>
          </STMTTRN>
        </BANKTRANLIST>
      </CCSTMTRS>
    </CCSTMTTRNRS>
  </CREDITCARDMSGSRSV1>
</OFX>
`
	rows, firstAcctExtID, warnings, err := ParseOFX(strings.NewReader(input), 1)
	if err != nil {
		t.Fatalf("ParseOFX unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows (one per statement), got %d", len(rows))
	}
	if rows[0].ImportHash == rows[1].ImportHash {
		t.Errorf("ImportHash collision across statements (same FITID, different ACCTID): %q", rows[0].ImportHash)
	}
	// The first non-empty acctExtID should be the bank statement's ACCTID.
	if firstAcctExtID != "BANK-AAA" {
		t.Errorf("firstAcctExtID = %q, want %q (first non-empty ACCTID encountered)", firstAcctExtID, "BANK-AAA")
	}
	// Verify the hashes are exactly what HashOFX would produce for the
	// respective per-statement ACCTIDs.
	wantBank := HashOFX("BANK-AAA", "100")
	wantCard := HashOFX("CARD-BBB", "100")
	if rows[0].ImportHash != wantBank {
		t.Errorf("bank-row hash = %q, want %q", rows[0].ImportHash, wantBank)
	}
	if rows[1].ImportHash != wantCard {
		t.Errorf("card-row hash = %q, want %q", rows[1].ImportHash, wantCard)
	}
}

func TestParseOFX_V2_BadDate_SkipsTxWithWarning(t *testing.T) {
	const input = `<?xml version="1.0" encoding="UTF-8"?>
<OFX>
  <BANKMSGSRSV1>
    <STMTTRNRS>
      <STMTRS>
        <BANKACCTFROM><ACCTID>ACCT-X</ACCTID></BANKACCTFROM>
        <BANKTRANLIST>
          <STMTTRN>
            <DTPOSTED>20250115</DTPOSTED>
            <TRNAMT>-1.00</TRNAMT>
            <FITID>GOOD-1</FITID>
            <NAME>Good Tx</NAME>
          </STMTTRN>
          <STMTTRN>
            <DTPOSTED>not-a-date</DTPOSTED>
            <TRNAMT>-1.00</TRNAMT>
            <FITID>BAD-DATE-1</FITID>
            <NAME>Whatever</NAME>
          </STMTTRN>
        </BANKTRANLIST>
      </STMTRS>
    </STMTTRNRS>
  </BANKMSGSRSV1>
</OFX>
`
	rows, _, warnings, err := ParseOFX(strings.NewReader(input), 1)
	if err != nil {
		t.Fatalf("ParseOFX unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row (the good one), got %d", len(rows))
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0], "BAD-DATE-1") {
		t.Errorf("expected FITID %q in warning, got: %v", "BAD-DATE-1", warnings[0])
	}
}

func TestIsOFXv2(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Starts with <?xml after trimming → true.
		{"xml declaration", `<?xml version="1.0"?><OFX></OFX>`, true},
		// Leading whitespace trimmed before check.
		{"xml with leading whitespace", "  \n<?xml version=\"1.0\"?><OFX></OFX>", true},
		// Bare <OFX> start → true.
		{"bare OFX element", "<OFX></OFX>", true},
		// OFX v1 SGML header → false.
		{"v1 OFXHEADER", "OFXHEADER:100\nDATA:OFXSGML\n<OFX>", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isOFXv2(tc.input)
			if got != tc.want {
				t.Errorf("isOFXv2(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestTokeniseOFX1_BasicTags(t *testing.T) {
	// tokeniseOFX1 emits one token per tag. For leaf tags the val is the
	// text between the '>' and the next '<' or newline. For container
	// closing tags (e.g. </B>) the val is "".
	input := "<A>x<B>y</B></A>"
	tokens := tokeniseOFX1(input)

	// Expect 4 tokens: A, B, /B, /A
	if len(tokens) != 4 {
		t.Fatalf("expected 4 tokens, got %d: %+v", len(tokens), tokens)
	}
	check := func(i int, wantTag, wantVal string) {
		t.Helper()
		if tokens[i].tag != wantTag {
			t.Errorf("tokens[%d].tag = %q, want %q", i, tokens[i].tag, wantTag)
		}
		if tokens[i].val != wantVal {
			t.Errorf("tokens[%d].val = %q, want %q", i, tokens[i].val, wantVal)
		}
	}
	check(0, "A", "x")
	check(1, "B", "y")
	check(2, "/B", "")
	check(3, "/A", "")
}

func TestFindTag_NotFound_ReturnsEmpty(t *testing.T) {
	tokens := []ofxToken{
		{tag: "DTPOSTED", val: "20250101"},
		{tag: "TRNAMT", val: "-1.00"},
	}
	got := findTag(tokens, "NAME")
	if got != "" {
		t.Errorf("findTag returned %q for missing tag, want %q", got, "")
	}
}

func TestFindTag_Found(t *testing.T) {
	tokens := []ofxToken{
		{tag: "DTPOSTED", val: "20250101"},
		{tag: "NAME", val: "STARBUCKS"},
		{tag: "TRNAMT", val: "-4.50"},
	}
	got := findTag(tokens, "NAME")
	if got != "STARBUCKS" {
		t.Errorf("findTag(NAME) = %q, want %q", got, "STARBUCKS")
	}
}

// TestParseOFX_V1_CreditCardMsgsrsv1 verifies that the OFX v1 parser handles
// a <CREDITCARDMSGSRSV1> block. The v1 parser tokenises the entire body after
// <OFX> without restricting to BANKMSGSRSV1, so STMTTRN entries within a
// credit-card block are parsed identically to bank transactions.
func TestParseOFX_V1_CreditCardMsgsrsv1(t *testing.T) {
	const input = `OFXHEADER:100
DATA:OFXSGML
VERSION:102

<OFX>
<CREDITCARDMSGSRSV1>
<CCSTMTTRNRS>
<CCSTMTRS>
<BANKACCTFROM>
<ACCTID>CC-9999
</BANKACCTFROM>
<BANKTRANLIST>
<STMTTRN>
<TRNTYPE>DEBIT
<DTPOSTED>20250301
<TRNAMT>-22.50
<FITID>CC-FIT-1
<NAME>COFFEE SHOP
</STMTTRN>
</BANKTRANLIST>
</CCSTMTRS>
</CCSTMTTRNRS>
</CREDITCARDMSGSRSV1>
</OFX>
`
	rows, acctExtID, warnings, err := ParseOFX(strings.NewReader(input), 1)
	if err != nil {
		t.Fatalf("ParseOFX unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if acctExtID != "CC-9999" {
		t.Errorf("acctExtID = %q, want %q", acctExtID, "CC-9999")
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].AmountCents != -2250 {
		t.Errorf("AmountCents = %d, want -2250", rows[0].AmountCents)
	}
}

func TestParseOFX_V1_PositiveAmount(t *testing.T) {
	const input = `OFXHEADER:100
DATA:OFXSGML

<OFX>
<BANKMSGSRSV1>
<STMTTRNRS>
<STMTRS>
<BANKACCTFROM>
<ACCTID>ACCT-POS
</BANKACCTFROM>
<BANKTRANLIST>
<STMTTRN>
<TRNTYPE>CREDIT
<DTPOSTED>20250301
<TRNAMT>100.00
<FITID>POS-FIT-1
<NAME>Salary Deposit
</STMTTRN>
</BANKTRANLIST>
</STMTRS>
</STMTTRNRS>
</BANKMSGSRSV1>
</OFX>
`
	rows, _, warnings, err := ParseOFX(strings.NewReader(input), 1)
	if err != nil {
		t.Fatalf("ParseOFX unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].AmountCents != 10000 {
		t.Errorf("AmountCents = %d, want 10000", rows[0].AmountCents)
	}
}

// TestParseOFX_V1_MissingFITID verifies that a STMTTRN with no FITID is
// skipped (toRow returns "missing FITID" error) and a warning is recorded.
func TestParseOFX_V1_MissingFITID(t *testing.T) {
	const input = `OFXHEADER:100
DATA:OFXSGML

<OFX>
<BANKMSGSRSV1>
<STMTTRNRS>
<STMTRS>
<BANKACCTFROM>
<ACCTID>ACCT-NOFIT
</BANKACCTFROM>
<BANKTRANLIST>
<STMTTRN>
<TRNTYPE>DEBIT
<DTPOSTED>20250401
<TRNAMT>-5.00
<NAME>No FITID Vendor
</STMTTRN>
</BANKTRANLIST>
</STMTRS>
</STMTTRNRS>
</BANKMSGSRSV1>
</OFX>
`
	rows, _, warnings, err := ParseOFX(strings.NewReader(input), 1)
	if err != nil {
		t.Fatalf("ParseOFX unexpected error: %v", err)
	}
	// The row must be skipped because FITID is required.
	if len(rows) != 0 {
		t.Errorf("expected 0 rows (missing FITID skipped), got %d", len(rows))
	}
	// A warning must be recorded mentioning the missing FITID.
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0], "FITID") {
		t.Errorf("expected warning to mention FITID, got: %v", warnings[0])
	}
}

func TestParseOFXDate_TimezoneOffset(t *testing.T) {
	// Timezone bracket is stripped; only the date part (first 8 chars) is used.
	got, err := parseOFXDate("20250115120000.000[-5:EST]")
	if err != nil {
		t.Fatalf("parseOFXDate unexpected error: %v", err)
	}
	if got != "2025-01-15" {
		t.Errorf("parseOFXDate = %q, want %q", got, "2025-01-15")
	}
}

func TestParseOFXDate_Variants(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"YYYYMMDD only", "20250115"},
		{"YYYYMMDDHHMMSS", "20250115120000"},
		{"YYYYMMDDHHMMSS.XXX with tz bracket", "20250115120000.000[-5:EST]"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseOFXDate(tc.input)
			if err != nil {
				t.Fatalf("parseOFXDate(%q) unexpected error: %v", tc.input, err)
			}
			if got != "2025-01-15" {
				t.Errorf("parseOFXDate(%q) = %q, want %q", tc.input, got, "2025-01-15")
			}
		})
	}
}
