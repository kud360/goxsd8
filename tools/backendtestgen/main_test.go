package main

import (
	"os"
	"testing"
)

const (
	testDatatypes = "../../docs/specs/md/xmlschema11-2.md"
	testPrecision = "../../docs/specs/md/xsd-precisionDecimal.md"
	testCommitted = "../../value/backendtest/gen_vectors.go"
)

func generate(t *testing.T) []byte {
	t.Helper()
	src, err := build(testDatatypes, testPrecision)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	return src
}

func readSpec(t *testing.T) string {
	t.Helper()
	content, err := os.ReadFile(testDatatypes)
	if err != nil {
		t.Fatalf("reading spec: %v", err)
	}
	return string(content)
}

// TestBooleanVectors pins the spec-derived boolean corpus: the valid
// round-trips and the near-miss invalids the generator must produce
// (§3.3.2.2, f-booleanLexmap/f-booleanCanmap, nt-booleanRep). It fails loudly
// if a spec edit or a parser change drifts the vectors.
func TestBooleanVectors(t *testing.T) {
	b, err := parseBoolean(readSpec(t))
	if err != nil {
		t.Fatalf("parseBoolean: %v", err)
	}

	wantValid := []roundtrip{
		{"true", "true"}, {"false", "false"}, {"1", "true"}, {"0", "false"},
	}
	if len(b.Valid) != len(wantValid) {
		t.Fatalf("valid: got %v, want %v", b.Valid, wantValid)
	}
	for i, w := range wantValid {
		if b.Valid[i] != w {
			t.Errorf("valid[%d]: got %v, want %v", i, b.Valid[i], w)
		}
	}

	wantInvalid := []string{"True", "TRUE", "False", "FALSE", "", "2"}
	if len(b.Invalid) != len(wantInvalid) {
		t.Fatalf("invalid: got %q, want %q", b.Invalid, wantInvalid)
	}
	for i, w := range wantInvalid {
		if b.Invalid[i] != w {
			t.Errorf("invalid[%d]: got %q, want %q", i, b.Invalid[i], w)
		}
	}
}

// TestDecimalVectors pins the spec-derived decimal corpus: the worked example
// lexicals of the production (decimal-lexical-representation) with the canonical
// forms f-decimalCanmap assigns (§3.3.3.2), and the regex-verified invalid
// near-misses. It fails loudly if a spec edit or a parser change drifts them.
func TestDecimalVectors(t *testing.T) {
	d, err := parseDecimal(readSpec(t))
	if err != nil {
		t.Fatalf("parseDecimal: %v", err)
	}

	wantValid := []roundtrip{
		{"-1.23", "-1.23"},
		{"12678967.543233", "12678967.543233"},
		{"+100000.00", "100000"}, // '+' dropped, trailing fractional zeros dropped
		{"210", "210"},
	}
	if len(d.Valid) != len(wantValid) {
		t.Fatalf("valid: got %v, want %v", d.Valid, wantValid)
	}
	for i, w := range wantValid {
		if d.Valid[i] != w {
			t.Errorf("valid[%d]: got %v, want %v", i, d.Valid[i], w)
		}
	}

	wantInvalid := []string{"-1.23E2", "+", ".", ""} // exponent, bare sign, bare point, empty
	if len(d.Invalid) != len(wantInvalid) {
		t.Fatalf("invalid: got %q, want %q", d.Invalid, wantInvalid)
	}
	for i, w := range wantInvalid {
		if d.Invalid[i] != w {
			t.Errorf("invalid[%d]: got %q, want %q", i, d.Invalid[i], w)
		}
	}
}

// TestStringVectors pins the string corpus: representative round-trips under the
// identity mapping (§3.3.1.2) and NO invalid lexicals (every Char* sequence is
// in the lexical space, nt-stringRep).
func TestStringVectors(t *testing.T) {
	s, err := parseString(readSpec(t))
	if err != nil {
		t.Fatalf("parseString: %v", err)
	}
	wantValid := []roundtrip{
		{"", ""}, {"abc", "abc"}, {"café", "café"}, {"𝔘nicode", "𝔘nicode"},
	}
	if len(s.Valid) != len(wantValid) {
		t.Fatalf("valid: got %v, want %v", s.Valid, wantValid)
	}
	for i, w := range wantValid {
		if s.Valid[i] != w {
			t.Errorf("valid[%d]: got %v, want %v", i, s.Valid[i], w)
		}
	}
	if len(s.Invalid) != 0 {
		t.Errorf("invalid: got %q, want none (every string is in the lexical space)", s.Invalid)
	}
}

// floatingWant is the spec canonical corpus float and double must both produce
// (§3.3.4.2/§3.3.5.2): identical because the sample values are exactly
// representable at both precisions, so the shortest round-trip is the same.
var floatingWant = struct {
	valid   []roundtrip
	invalid []string
}{
	valid: []roundtrip{
		{"INF", "INF"}, {"-INF", "-INF"}, {"NaN", "NaN"}, {"+INF", "INF"},
		{"0", "0.0E0"}, {"-0", "-0.0E0"}, {"1", "1.0E0"}, {"-1", "-1.0E0"},
		{"1.5E1", "1.5E1"}, {"100", "1.0E2"}, {".5", "5.0E-1"},
		{"-0.001", "-1.0E-3"}, {"3.14", "3.14E0"},
	},
	invalid: []string{"+NaN", "-NaN", "Infinity", "INF ", "1.5e", "++1", "1.0.0", ""},
}

// TestFloatVectors and TestDoubleVectors pin the spec-derived float/double corpora:
// the special literals (nt-numSpecReps) and representative numerals with the
// canonical forms scientificCanonicalMap assigns (§3.3.4.2/§3.3.5.2), plus the
// regex-verified near-miss invalids (notably the +NaN/-NaN the stricter special
// grammar excludes). They fail loudly if a spec edit or a parser change drifts.
func TestFloatVectors(t *testing.T)  { checkFloatingVectors(t, "float", 32) }
func TestDoubleVectors(t *testing.T) { checkFloatingVectors(t, "double", 64) }

func checkFloatingVectors(t *testing.T, local string, bitSize int) {
	t.Helper()
	tv, err := parseFloating(readSpec(t), local, bitSize)
	if err != nil {
		t.Fatalf("parseFloating(%s): %v", local, err)
	}
	if len(tv.Valid) != len(floatingWant.valid) {
		t.Fatalf("%s valid: got %v, want %v", local, tv.Valid, floatingWant.valid)
	}
	for i, w := range floatingWant.valid {
		if tv.Valid[i] != w {
			t.Errorf("%s valid[%d]: got %v, want %v", local, i, tv.Valid[i], w)
		}
	}
	if len(tv.Invalid) != len(floatingWant.invalid) {
		t.Fatalf("%s invalid: got %q, want %q", local, tv.Invalid, floatingWant.invalid)
	}
	for i, w := range floatingWant.invalid {
		if tv.Invalid[i] != w {
			t.Errorf("%s invalid[%d]: got %q, want %q", local, i, tv.Invalid[i], w)
		}
	}
}

// TestHexBinaryVectors pins the spec-derived hexBinary corpus (§3.3.15.2,
// nt-hexBinary, f-hexBinaryCanonical): the representative round-trips — empty,
// lowercase and uppercase input, a multi-octet value — canonicalising to
// uppercase A–F, and the regex-verified invalid near-misses (odd length, a
// non-hex digit).
func TestHexBinaryVectors(t *testing.T) {
	h, err := parseHexBinary(readSpec(t))
	if err != nil {
		t.Fatalf("parseHexBinary: %v", err)
	}
	wantValid := []roundtrip{
		{"", ""}, {"0FB7", "0FB7"}, {"0fb7", "0FB7"}, {"deadBEEF", "DEADBEEF"}, {"ff", "FF"},
	}
	if len(h.Valid) != len(wantValid) {
		t.Fatalf("valid: got %v, want %v", h.Valid, wantValid)
	}
	for i, w := range wantValid {
		if h.Valid[i] != w {
			t.Errorf("valid[%d]: got %v, want %v", i, h.Valid[i], w)
		}
	}
	wantInvalid := []string{"F", "0FB", "0G", "gg"} // odd length, odd length, non-hex, non-hex
	if len(h.Invalid) != len(wantInvalid) {
		t.Fatalf("invalid: got %q, want %q", h.Invalid, wantInvalid)
	}
	for i, w := range wantInvalid {
		if h.Invalid[i] != w {
			t.Errorf("invalid[%d]: got %q, want %q", i, h.Invalid[i], w)
		}
	}
}

// TestBase64BinaryVectors pins the spec-derived base64Binary corpus (§3.3.16.2,
// nt-Base64Binary): representative round-trips exercising the empty sequence, an
// unpadded quad and both padding widths, and the regex-verified invalids — a
// non-multiple-of-four count and a bad restricted final character under single
// ("AQJ=") and double ("AB==") padding.
func TestBase64BinaryVectors(t *testing.T) {
	b, err := parseBase64Binary(readSpec(t))
	if err != nil {
		t.Fatalf("parseBase64Binary: %v", err)
	}
	wantValid := []roundtrip{
		{"", ""}, {"AQID", "AQID"}, {"AQI=", "AQI="}, {"AQ==", "AQ=="}, {"TWFu", "TWFu"},
	}
	if len(b.Valid) != len(wantValid) {
		t.Fatalf("valid: got %v, want %v", b.Valid, wantValid)
	}
	for i, w := range wantValid {
		if b.Valid[i] != w {
			t.Errorf("valid[%d]: got %v, want %v", i, b.Valid[i], w)
		}
	}
	wantInvalid := []string{"AQI", "AQJ=", "AB==", "A==="}
	if len(b.Invalid) != len(wantInvalid) {
		t.Fatalf("invalid: got %q, want %q", b.Invalid, wantInvalid)
	}
	for i, w := range wantInvalid {
		if b.Invalid[i] != w {
			t.Errorf("invalid[%d]: got %q, want %q", i, b.Invalid[i], w)
		}
	}
}

// TestDurationVectors pins the spec-derived duration corpus (§3.3.6.2,
// nt-durationRep; f-durationMap/f-durationCanMap, §E.2): the representative
// round-trips — a year, year+month, month, the zero duration ("PT0S"), a day, a
// minute, an hour value that normalizes into days ("PT36H" → "P1DT12H"), a
// seconds value that normalizes into minutes ("PT60S" → "PT1M"), an all-fields
// value, a fractional-seconds value and a negative — and the regex-verified
// invalids (a missing 'P', bare "P"/"PT", an out-of-place 'S', a sign inside a
// field).
func TestDurationVectors(t *testing.T) {
	d, err := parseDuration(readSpec(t))
	if err != nil {
		t.Fatalf("parseDuration: %v", err)
	}
	wantValid := []roundtrip{
		{"P1Y", "P1Y"}, {"P1Y2M", "P1Y2M"}, {"P1M", "P1M"}, {"P0M", "PT0S"},
		{"P1D", "P1D"}, {"PT1M", "PT1M"}, {"PT36H", "P1DT12H"}, {"PT60S", "PT1M"},
		{"P1DT2H3M4S", "P1DT2H3M4S"}, {"PT1.5S", "PT1.5S"},
		{"-P1Y2M3DT4H5M6S", "-P1Y2M3DT4H5M6S"},
	}
	if len(d.Valid) != len(wantValid) {
		t.Fatalf("valid: got %v, want %v", d.Valid, wantValid)
	}
	for i, w := range wantValid {
		if d.Valid[i] != w {
			t.Errorf("valid[%d]: got %v, want %v", i, d.Valid[i], w)
		}
	}
	wantInvalid := []string{"P", "PT", "P1S", "1Y", "PT1D", "PY"}
	if len(d.Invalid) != len(wantInvalid) {
		t.Fatalf("invalid: got %q, want %q", d.Invalid, wantInvalid)
	}
	for i, w := range wantInvalid {
		if d.Invalid[i] != w {
			t.Errorf("invalid[%d]: got %q, want %q", i, d.Invalid[i], w)
		}
	}
}

// TestDateTimeVectors pins the spec-derived dateTime corpus (§3.3.7.2,
// nt-dateTimeRep; vp-dateTimeLexRep/vp-dateTimeCanRep, §E.3.5/§E.3.6): the
// representative round-trips — a plain instant, a fractional-second, the three
// timezone spellings, a leap-day, an endOfDayFrag that carries into the next day
// ("…T24:00:00" → next-day "…T00:00:00"), a negative year and a four-digit-padded
// year — and the oracle-verified invalids, including the day-of-month violations
// (2023-02-30, non-leap 2023-02-29) the grammar regex alone would accept.
func TestDateTimeVectors(t *testing.T) {
	d, err := parseDateTime(readSpec(t))
	if err != nil {
		t.Fatalf("parseDateTime: %v", err)
	}
	wantValid := []roundtrip{
		{"2001-10-26T21:32:52", "2001-10-26T21:32:52"},
		{"2001-10-26T21:32:52.125", "2001-10-26T21:32:52.125"},
		{"2001-10-26T21:32:52+02:00", "2001-10-26T21:32:52+02:00"},
		{"2001-10-26T19:32:52Z", "2001-10-26T19:32:52Z"},
		{"2001-10-26T21:32:52-05:00", "2001-10-26T21:32:52-05:00"},
		{"2024-02-29T00:00:00", "2024-02-29T00:00:00"},
		{"2023-01-01T24:00:00", "2023-01-02T00:00:00"},
		{"-0045-03-15T00:00:00Z", "-0045-03-15T00:00:00Z"},
		{"0001-01-01T00:00:00", "0001-01-01T00:00:00"},
	}
	if len(d.Valid) != len(wantValid) {
		t.Fatalf("valid: got %v, want %v", d.Valid, wantValid)
	}
	for i, w := range wantValid {
		if d.Valid[i] != w {
			t.Errorf("valid[%d]: got %v, want %v", i, d.Valid[i], w)
		}
	}
	wantInvalid := []string{
		"2023-13-01T00:00:00", "2023-02-30T00:00:00", "2023-02-29T00:00:00",
		"2023-01-01T25:00:00", "2023-01-01T00:60:00", "2023-01-0100:00:00",
		"2023-01-01T00:00:00+15:00", "2023-1-01T00:00:00",
	}
	if len(d.Invalid) != len(wantInvalid) {
		t.Fatalf("invalid: got %q, want %q", d.Invalid, wantInvalid)
	}
	for i, w := range wantInvalid {
		if d.Invalid[i] != w {
			t.Errorf("invalid[%d]: got %q, want %q", i, d.Invalid[i], w)
		}
	}
}

// checkVectors is the shared body of the per-type vector pin tests: it asserts
// the parser produced exactly wantValid/wantInvalid in order.
func checkVectors(t *testing.T, local string, got typeVectors, wantValid []roundtrip, wantInvalid []string) {
	t.Helper()
	if len(got.Valid) != len(wantValid) {
		t.Fatalf("%s valid: got %v, want %v", local, got.Valid, wantValid)
	}
	for i, w := range wantValid {
		if got.Valid[i] != w {
			t.Errorf("%s valid[%d]: got %v, want %v", local, i, got.Valid[i], w)
		}
	}
	if len(got.Invalid) != len(wantInvalid) {
		t.Fatalf("%s invalid: got %q, want %q", local, got.Invalid, wantInvalid)
	}
	for i, w := range wantInvalid {
		if got.Invalid[i] != w {
			t.Errorf("%s invalid[%d]: got %q, want %q", local, i, got.Invalid[i], w)
		}
	}
}

// TestTimeVectors pins the spec-derived time corpus (§3.3.8.2, nt-timeRep;
// vp-timeLexRep/vp-timeCanRep): the timezone spellings, fractional seconds and
// the endOfDayFrag "24:00:00" that maps to midnight (no day carry).
func TestTimeVectors(t *testing.T) {
	tv, err := parseTime(readSpec(t))
	if err != nil {
		t.Fatalf("parseTime: %v", err)
	}
	checkVectors(t, "time", tv,
		[]roundtrip{
			{"13:20:00", "13:20:00"}, {"13:20:00.125", "13:20:00.125"},
			{"13:20:00Z", "13:20:00Z"}, {"13:20:00+02:00", "13:20:00+02:00"},
			{"13:20:00-05:00", "13:20:00-05:00"}, {"24:00:00", "00:00:00"},
		},
		[]string{"13:20", "25:00:00", "13:60:00", "1:20:00", "13:20:00+15:00", ""})
}

// TestDateVectors pins the date corpus (§3.3.9.2, nt-dateRep) including the
// oracle-verified year-dependent day violation (non-leap 2023-02-29).
func TestDateVectors(t *testing.T) {
	tv, err := parseDate(readSpec(t))
	if err != nil {
		t.Fatalf("parseDate: %v", err)
	}
	checkVectors(t, "date", tv,
		[]roundtrip{
			{"2002-10-10", "2002-10-10"}, {"2002-10-10Z", "2002-10-10Z"},
			{"2002-10-10+02:00", "2002-10-10+02:00"}, {"2002-10-10-05:00", "2002-10-10-05:00"},
			{"2024-02-29", "2024-02-29"}, {"-0045-03-15", "-0045-03-15"}, {"0001-01-01", "0001-01-01"},
		},
		[]string{"2002-13-01", "2002-10-32", "2023-02-29", "2023-02-30", "2002-10-10T00:00:00", "2002-1-01"})
}

// TestGYearMonthVectors pins the gYearMonth corpus (§3.3.10.2, nt-gYearMonthRep).
func TestGYearMonthVectors(t *testing.T) {
	tv, err := parseGYearMonth(readSpec(t))
	if err != nil {
		t.Fatalf("parseGYearMonth: %v", err)
	}
	checkVectors(t, "gYearMonth", tv,
		[]roundtrip{
			{"2002-10", "2002-10"}, {"2002-10Z", "2002-10Z"}, {"2002-10+02:00", "2002-10+02:00"},
			{"2002-10-05:00", "2002-10-05:00"}, {"-0045-03", "-0045-03"}, {"0001-01", "0001-01"},
		},
		[]string{"2002", "2002-13", "2002-10-10", "2002-1", ""})
}

// TestGYearVectors pins the gYear corpus (§3.3.11.2, nt-gYearRep): gYear permits
// a timezone (§3.3.11.1).
func TestGYearVectors(t *testing.T) {
	tv, err := parseGYear(readSpec(t))
	if err != nil {
		t.Fatalf("parseGYear: %v", err)
	}
	checkVectors(t, "gYear", tv,
		[]roundtrip{
			{"2002", "2002"}, {"2002Z", "2002Z"}, {"2002+02:00", "2002+02:00"},
			{"2002-05:00", "2002-05:00"}, {"-0045", "-0045"}, {"12345", "12345"},
		},
		[]string{"999", "2002-10", "02002", ""})
}

// TestGMonthDayVectors pins the gMonthDay corpus (§3.3.12.2, nt-gMonthDayRep):
// the unconditional leap-day --02-29 and the oracle-verified year-free day
// violations (--02-30, --04-31).
func TestGMonthDayVectors(t *testing.T) {
	tv, err := parseGMonthDay(readSpec(t))
	if err != nil {
		t.Fatalf("parseGMonthDay: %v", err)
	}
	checkVectors(t, "gMonthDay", tv,
		[]roundtrip{
			{"--10-10", "--10-10"}, {"--02-29", "--02-29"}, {"--12-31", "--12-31"},
			{"--01-01Z", "--01-01Z"}, {"--12-12+13:00", "--12-12+13:00"}, {"--12-12-05:00", "--12-12-05:00"},
		},
		[]string{"--13-01", "--02-30", "--04-31", "--10-10-", "10-10", "--1-01"})
}

// TestGDayVectors pins the gDay corpus (§3.3.13.2, nt-gDayRep): day 31 valid, no
// day-of-month representation constraint.
func TestGDayVectors(t *testing.T) {
	tv, err := parseGDay(readSpec(t))
	if err != nil {
		t.Fatalf("parseGDay: %v", err)
	}
	checkVectors(t, "gDay", tv,
		[]roundtrip{
			{"---15", "---15"}, {"---31", "---31"}, {"---15Z", "---15Z"},
			{"---15-13:00", "---15-13:00"}, {"---16+13:00", "---16+13:00"}, {"---01", "---01"},
		},
		[]string{"---00", "---32", "--15", "15", "---1", ""})
}

// TestGMonthVectors pins the gMonth corpus (§3.3.14.2, nt-gMonthRep).
func TestGMonthVectors(t *testing.T) {
	tv, err := parseGMonth(readSpec(t))
	if err != nil {
		t.Fatalf("parseGMonth: %v", err)
	}
	checkVectors(t, "gMonth", tv,
		[]roundtrip{
			{"--10", "--10"}, {"--01", "--01"}, {"--12", "--12"},
			{"--05Z", "--05Z"}, {"--11+13:00", "--11+13:00"}, {"--10-05:00", "--10-05:00"},
		},
		[]string{"--13", "--00", "10", "--10-10", "--1", ""})
}

// TestApplicableFacets pins that each cohort type carries its cos-applicable-facets
// list in spec order (§4.1.5), sourced from the shared builtin spec parser.
func TestApplicableFacets(t *testing.T) {
	facets, err := applicableFacets(testDatatypes, testPrecision)
	if err != nil {
		t.Fatalf("applicableFacets: %v", err)
	}
	want := map[string][]string{
		"boolean":      {"whiteSpace", "pattern", "assertions"},
		"string":       {"whiteSpace", "length", "minLength", "maxLength", "pattern", "enumeration", "assertions"},
		"decimal":      {"whiteSpace", "totalDigits", "fractionDigits", "pattern", "enumeration", "maxInclusive", "maxExclusive", "minInclusive", "minExclusive", "assertions"},
		"float":        {"whiteSpace", "pattern", "enumeration", "maxInclusive", "maxExclusive", "minInclusive", "minExclusive", "assertions"},
		"double":       {"whiteSpace", "pattern", "enumeration", "maxInclusive", "maxExclusive", "minInclusive", "minExclusive", "assertions"},
		"hexBinary":    {"whiteSpace", "length", "minLength", "maxLength", "pattern", "enumeration", "assertions"},
		"base64Binary": {"whiteSpace", "length", "minLength", "maxLength", "pattern", "enumeration", "assertions"},
		"duration":     {"whiteSpace", "pattern", "enumeration", "maxInclusive", "maxExclusive", "minInclusive", "minExclusive", "assertions"},
		"dateTime":     {"whiteSpace", "explicitTimezone", "pattern", "enumeration", "maxInclusive", "maxExclusive", "minInclusive", "minExclusive", "assertions"},
		"time":         {"whiteSpace", "explicitTimezone", "pattern", "enumeration", "maxInclusive", "maxExclusive", "minInclusive", "minExclusive", "assertions"},
		"date":         {"whiteSpace", "explicitTimezone", "pattern", "enumeration", "maxInclusive", "maxExclusive", "minInclusive", "minExclusive", "assertions"},
		"gYearMonth":   {"whiteSpace", "explicitTimezone", "pattern", "enumeration", "maxInclusive", "maxExclusive", "minInclusive", "minExclusive", "assertions"},
		"gYear":        {"whiteSpace", "explicitTimezone", "pattern", "enumeration", "maxInclusive", "maxExclusive", "minInclusive", "minExclusive", "assertions"},
		"gMonthDay":    {"whiteSpace", "explicitTimezone", "pattern", "enumeration", "maxInclusive", "maxExclusive", "minInclusive", "minExclusive", "assertions"},
		"gDay":         {"whiteSpace", "explicitTimezone", "pattern", "enumeration", "maxInclusive", "maxExclusive", "minInclusive", "minExclusive", "assertions"},
		"gMonth":       {"whiteSpace", "explicitTimezone", "pattern", "enumeration", "maxInclusive", "maxExclusive", "minInclusive", "minExclusive", "assertions"},
	}
	for name, w := range want {
		got := facets[name]
		if len(got) != len(w) {
			t.Fatalf("%s: got %q, want %q", name, got, w)
		}
		for i := range w {
			if got[i] != w[i] {
				t.Errorf("%s facet[%d]: got %q, want %q", name, i, got[i], w[i])
			}
		}
	}
}

// TestDeterministic pins STYLE D1: two generation runs are byte-identical.
func TestDeterministic(t *testing.T) {
	first := generate(t)
	second := generate(t)
	if string(first) != string(second) {
		t.Fatal("emit is not byte-identical across runs")
	}
}

// TestCommittedUpToDate fails if value/backendtest/gen_vectors.go has drifted
// from what the generator produces — a hand edit or a missed `go generate`.
func TestCommittedUpToDate(t *testing.T) {
	want, err := os.ReadFile(testCommitted)
	if err != nil {
		t.Fatalf("reading committed file: %v", err)
	}
	if string(generate(t)) != string(want) {
		t.Fatalf("%s is stale; run `go generate ./...`", testCommitted)
	}
}
