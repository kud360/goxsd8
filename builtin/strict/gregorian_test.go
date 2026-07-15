package strict_test

import (
	"testing"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsderr"
)

// roundtripCases parses each lexical and asserts it canonicalizes to the paired
// form (·XLexicalMap·/·XCanonicalMap·, §E.3.5/§E.3.6).
func roundtripCases(t *testing.T, local string, cases map[string]string) {
	t.Helper()
	m := mappingFor(t, local)
	for lex, want := range cases {
		v, err := m.Parse(lex, nil)
		if err != nil {
			t.Errorf("%s: Parse(%q): unexpected error %v", local, lex, err)
			continue
		}
		got, err := m.Canonical(v)
		if err != nil {
			t.Errorf("%s: Canonical(%q): unexpected error %v", local, lex, err)
			continue
		}
		if got != want {
			t.Errorf("%s: Canonical(Parse(%q)) = %q, want %q", local, lex, got, want)
		}
	}
}

// rejectCases asserts each lexical is outside the type's lexical space, rejected
// with rule cvc-datatype-valid (§4.1.4).
func rejectCases(t *testing.T, local string, lexicals []string) {
	t.Helper()
	m := mappingFor(t, local)
	for _, lex := range lexicals {
		_, err := m.Parse(lex, nil)
		if err == nil {
			t.Errorf("%s: Parse(%q): want lexical-space error, got nil", local, lex)
			continue
		}
		if rule, ok := xsderr.RuleOf(err); !ok || rule != "cvc-datatype-valid" {
			t.Errorf("%s: Parse(%q): rule = %q (ok=%v), want cvc-datatype-valid", local, lex, rule, ok)
		}
	}
}

// TestTimeParseAndCanonical pins ·timeLexicalMap·/·timeCanonicalMap·
// (vp-timeLexRep/vp-timeCanRep, §E.3.5/§E.3.6): the timezone spellings, fractional
// seconds, and the endOfDayFrag "24:00:00" that maps to midnight (NOT a day
// carry — time has no day; §3.3.8.2 Note).
func TestTimeParseAndCanonical(t *testing.T) {
	roundtripCases(t, "time", map[string]string{
		"13:20:00":       "13:20:00",
		"13:20:00.5":     "13:20:00.5",
		"13:20:00.50":    "13:20:00.5", // trailing fractional zeros drop
		"13:20:00Z":      "13:20:00Z",
		"13:20:00+00:00": "13:20:00Z", // +00:00 canonicalizes to Z
		"13:20:00+14:00": "13:20:00+14:00",
		"13:20:00-05:00": "13:20:00-05:00",
		"00:00:00":       "00:00:00",
		"24:00:00":       "00:00:00",       // endOfDay → midnight, no day
		"24:00:00.0":     "00:00:00",       // endOfDay with fractional zeros
		"24:00:00-05:00": "00:00:00-05:00", // endOfDay keeps timezone
	})
}

func TestTimeReject(t *testing.T) {
	rejectCases(t, "time", []string{
		"", "13:20", "13:20:00:00", "25:00:00", "13:60:00", "13:20:60",
		"1:20:00", "13:20:00+15:00", "13:20:00+02", " 13:20:00", "13:20:00 ",
		"24:00:01", "24:01:00", "2001-10-26T13:20:00",
	})
}

// TestDateParseAndCanonical pins ·dateLexicalMap·/·dateCanonicalMap· and the
// year-dependent con-date-dayValue (§3.3.9.1): a leap-year Feb 29, negative and
// padded years, timezone spellings. There is NO endOfDayFrag for date.
func TestDateParseAndCanonical(t *testing.T) {
	roundtripCases(t, "date", map[string]string{
		"2002-10-10":       "2002-10-10",
		"2002-10-10Z":      "2002-10-10Z",
		"2002-10-10+13:00": "2002-10-10+13:00",
		"2002-10-10-05:00": "2002-10-10-05:00",
		"2024-02-29":       "2024-02-29", // leap-year Feb 29 valid
		"-0045-03-15":      "-0045-03-15",
		"0001-01-01":       "0001-01-01",
		"0000-01-01":       "0000-01-01", // XSD 1.1 permits year 0
		"12345-06-07":      "12345-06-07",
	})
}

func TestDateReject(t *testing.T) {
	rejectCases(t, "date", []string{
		"", "2002-10", "2002-13-01", "2002-00-01", "2002-10-32",
		"2023-02-29", // Feb 29 in a non-leap year (con-date-dayValue)
		"2023-02-30", "2023-04-31",
		"2002-10-10T00:00:00", "2002-10-10+15:00", "999-01-01",
		"2002-1-01", "2002-10-1",
	})
}

// TestGYearMonthParseAndCanonical pins ·gYearMonthLexicalMap·/·gYearMonthCanonicalMap·.
func TestGYearMonthParseAndCanonical(t *testing.T) {
	roundtripCases(t, "gYearMonth", map[string]string{
		"2002-10":       "2002-10",
		"2002-10Z":      "2002-10Z",
		"2002-10+13:00": "2002-10+13:00",
		"2002-10-05:00": "2002-10-05:00",
		"-0045-03":      "-0045-03",
		"0000-01":       "0000-01",
		"12345-06":      "12345-06",
	})
}

func TestGYearMonthReject(t *testing.T) {
	rejectCases(t, "gYearMonth", []string{
		"", "2002", "2002-13", "2002-00", "2002-10-10", "999-01",
		"2002-1", "2002-10+15:00",
	})
}

// TestGYearParseAndCanonical pins ·gYearLexicalMap·/·gYearCanonicalMap·. gYear
// permits a timezone (§3.3.11.1: ·timezoneOffset· stays optional).
func TestGYearParseAndCanonical(t *testing.T) {
	roundtripCases(t, "gYear", map[string]string{
		"2002":       "2002",
		"2002Z":      "2002Z",
		"2002+13:00": "2002+13:00",
		"2002-05:00": "2002-05:00",
		"-0045":      "-0045",
		"0000":       "0000",
		"12345":      "12345",
	})
}

func TestGYearReject(t *testing.T) {
	rejectCases(t, "gYear", []string{
		"", "999", "02002", "2002-10", "2002+15:00", " 2002", "2002 ",
	})
}

// TestGMonthDayParseAndCanonical pins ·gMonthDayLexicalMap·/·gMonthDayCanonicalMap·
// and the year-FREE con-gMonthDay-dayValue (§3.3.12.1): --02-29 is
// unconditionally valid (no year to check against).
func TestGMonthDayParseAndCanonical(t *testing.T) {
	roundtripCases(t, "gMonthDay", map[string]string{
		"--10-10":       "--10-10",
		"--02-29":       "--02-29", // unconditionally valid, no year
		"--12-31":       "--12-31",
		"--01-01Z":      "--01-01Z",
		"--12-12+13:00": "--12-12+13:00",
		"--12-12+11:00": "--12-12+11:00",
	})
}

func TestGMonthDayReject(t *testing.T) {
	rejectCases(t, "gMonthDay", []string{
		"", "--13-01", "--00-01", "--02-30", // Feb has no 30 (con-gMonthDay-dayValue)
		"--04-31", "--06-31", "--09-31", "--11-31", // 30-day months
		"10-10", "-10-10", "--10-10-", "--1-01", "--10-1", "--10-10+15:00",
	})
}

// TestGDayParseAndCanonical pins ·gDayLexicalMap·/·gDayCanonicalMap·. gDay has NO
// day-of-month representation constraint: the flat 1-31 range is the whole rule.
func TestGDayParseAndCanonical(t *testing.T) {
	roundtripCases(t, "gDay", map[string]string{
		"---15":       "---15",
		"---01":       "---01",
		"---31":       "---31", // day 31 valid for gDay (no month to bound it)
		"---15Z":      "---15Z",
		"---15-13:00": "---15-13:00",
		"---16+13:00": "---16+13:00",
	})
}

func TestGDayReject(t *testing.T) {
	rejectCases(t, "gDay", []string{
		"", "---00", "---32", "--15", "----15", "15", "---1", "---15+15:00",
	})
}

// TestGMonthParseAndCanonical pins ·gMonthLexicalMap·/·gMonthCanonicalMap·. The
// canonical map uses gMonth's ·month· (the spec's "gM's ·day·" wording is a
// transcription artifact; gMonth has no day — PRINCIPLES 25).
func TestGMonthParseAndCanonical(t *testing.T) {
	roundtripCases(t, "gMonth", map[string]string{
		"--10":       "--10",
		"--01":       "--01",
		"--12":       "--12",
		"--05Z":      "--05Z",
		"--11+13:00": "--11+13:00",
		"--10-05:00": "--10-05:00",
	})
}

func TestGMonthReject(t *testing.T) {
	rejectCases(t, "gMonth", []string{
		"", "--13", "--00", "10", "-10", "--10-10", "--1", "--10+15:00",
	})
}

// TestGDayOrderAnomaly pins the spec's worked gDay order examples (§3.3.13.1):
// ---15 < ---16, but ---15−13:00 > ---16+13:00 (timezone offsets do not wrap the
// month boundary), and ---15−11:00 = ---16+13:00 (same first moment). ---15−13:00
// and ---16 are Incomparable under the ±14h imputation.
func TestGDayOrderAnomaly(t *testing.T) {
	m := mappingFor(t, "gDay")
	parse := func(lex string) value.Value {
		v, err := m.Parse(lex, nil)
		if err != nil {
			t.Fatalf("Parse(%q): %v", lex, err)
		}
		return v
	}
	ord := func(lex string) value.Ordered {
		o, ok := parse(lex).(value.Ordered)
		if !ok {
			t.Fatalf("gDay %q is not value.Ordered", lex)
		}
		return o
	}
	type wc struct {
		a, b string
		want value.Ordering
	}
	for _, c := range []wc{
		{"---15", "---16", value.Less},
		{"---15-13:00", "---16+13:00", value.Greater},
		{"---15-11:00", "---16+13:00", value.Equal},
		{"---15-13:00", "---16", value.Incomparable},
		{"---01+13:00", "---31-13:00", value.Less}, // no month wrap-around (§3.3.13.1 Note)
	} {
		if got := ord(c.a).Cmp(parse(c.b)); got != c.want {
			t.Errorf("Cmp(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

// TestGMonthDayTimezoneAnomaly pins §3.3.12.1's worked example:
// --12-12+13:00 < --12-12+11:00 (a more-easterly offset starts earlier).
func TestGMonthDayTimezoneAnomaly(t *testing.T) {
	m := mappingFor(t, "gMonthDay")
	a, _ := m.Parse("--12-12+13:00", nil)
	b, _ := m.Parse("--12-12+11:00", nil)
	if got := a.(value.Ordered).Cmp(b); got != value.Less {
		t.Errorf("Cmp(--12-12+13:00, --12-12+11:00) = %v, want Less", got)
	}
}

// TestSevenPropertyEqIdentical pins the Eq/Identical divergence shared by the
// family: a timezone-shifted pair denoting the same first moment is Eq but NOT
// Identical (different stored ·timezoneOffset·, §2.2.2). date example from
// §3.3.9.1: 2000-01-01+13:00 and 1999-12-31−11:00 begin at the same moment.
func TestSevenPropertyEqIdentical(t *testing.T) {
	m := mappingFor(t, "date")
	a, _ := m.Parse("2000-01-01+13:00", nil)
	b, _ := m.Parse("1999-12-31-11:00", nil)
	if !a.(value.Eq).Eq(b) {
		t.Error("Eq(2000-01-01+13:00, 1999-12-31-11:00) = false, want true (same first moment)")
	}
	if a.(value.Identical).Identical(b) {
		t.Error("Identical of same-instant, different-offset dates = true, want false")
	}
	if !a.(value.Identical).Identical(mustParse(t, m, "2000-01-01+13:00")) {
		t.Error("Identical must hold for structurally equal dates")
	}
}

func mustParse(t *testing.T, m value.Mapping, lex string) value.Value {
	t.Helper()
	v, err := m.Parse(lex, nil)
	if err != nil {
		t.Fatalf("Parse(%q): %v", lex, err)
	}
	return v
}

// TestSevenPropertyCrossTypeIncomparable pins that comparing across the family
// (e.g. gYear vs gMonth, or any sibling vs a foreign value) is Incomparable /
// unequal / not identical — a type assertion failure, never a panic.
func TestSevenPropertyCrossTypeIncomparable(t *testing.T) {
	gy := mustParse(t, mappingFor(t, "gYear"), "2002Z")
	gm := mustParse(t, mappingFor(t, "gMonth"), "--10Z")
	if got := gy.(value.Ordered).Cmp(gm); got != value.Incomparable {
		t.Errorf("Cmp(gYear, gMonth) = %v, want Incomparable", got)
	}
	if gy.(value.Eq).Eq(gm) {
		t.Error("Eq(gYear, gMonth) = true, want false")
	}
	if gy.(value.Identical).Identical(gm) {
		t.Error("Identical(gYear, gMonth) = true, want false")
	}
	if got := gy.(value.Ordered).Cmp("nope"); got != value.Incomparable {
		t.Errorf("Cmp(gYear, string) = %v, want Incomparable", got)
	}
}

// TestSevenPropertyHasTimezone pins value.TimezoneAware across the family (the
// explicitTimezone facet, §4.3.14, reads it).
func TestSevenPropertyHasTimezone(t *testing.T) {
	cases := map[string]map[string]bool{
		"time":       {"13:20:00": false, "13:20:00Z": true, "13:20:00-05:00": true},
		"date":       {"2002-10-10": false, "2002-10-10Z": true},
		"gYearMonth": {"2002-10": false, "2002-10+02:00": true},
		"gYear":      {"2002": false, "2002Z": true},
		"gMonthDay":  {"--10-10": false, "--10-10Z": true},
		"gDay":       {"---15": false, "---15-05:00": true},
		"gMonth":     {"--10": false, "--10Z": true},
	}
	for local, lexes := range cases {
		m := mappingFor(t, local)
		for lex, want := range lexes {
			v := mustParse(t, m, lex)
			tz, ok := v.(value.TimezoneAware)
			if !ok {
				t.Fatalf("%s value %q is not value.TimezoneAware", local, lex)
			}
			if got := tz.HasTimezone(); got != want {
				t.Errorf("%s: HasTimezone(%q) = %v, want %v", local, lex, got, want)
			}
		}
	}
}

// TestSevenPropertyNotLengthed pins the deliberate capability boundary
// (cos-applicable-facets §4.1.5): none of the family is Lengthed/DigitCounted/
// Scaled — no applicable facet needs them.
func TestSevenPropertyNotLengthed(t *testing.T) {
	samples := map[string]string{
		"time": "13:20:00Z", "date": "2002-10-10Z", "gYearMonth": "2002-10Z",
		"gYear": "2002Z", "gMonthDay": "--10-10Z", "gDay": "---15Z", "gMonth": "--10Z",
	}
	for local, lex := range samples {
		v := mustParse(t, mappingFor(t, local), lex)
		if _, ok := v.(value.Lengthed); ok {
			t.Errorf("%s implements value.Lengthed; it must not", local)
		}
		if _, ok := v.(value.DigitCounted); ok {
			t.Errorf("%s implements value.DigitCounted; it must not", local)
		}
		if _, ok := v.(value.Scaled); ok {
			t.Errorf("%s implements value.Scaled; it must not", local)
		}
	}
}

// TestSevenPropertyCanonicalForeign pins the warden guardrail: Canonical on a
// foreign value is an *xsderr.Error, never a panic.
func TestSevenPropertyCanonicalForeign(t *testing.T) {
	for _, local := range []string{"time", "date", "gYearMonth", "gYear", "gMonthDay", "gDay", "gMonth"} {
		m := mappingFor(t, local)
		_, err := m.Canonical("not a value")
		if err == nil {
			t.Errorf("%s: Canonical(foreign): want error, got nil", local)
			continue
		}
		if rule, ok := xsderr.RuleOf(err); !ok || rule != "cvc-datatype-valid" {
			t.Errorf("%s: Canonical(foreign): rule = %q (ok=%v), want cvc-datatype-valid", local, rule, ok)
		}
	}
}
