package strict_test

import (
	"testing"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsderr"
)

// TestDateTimeParseAndCanonical exercises ·dateTimeLexicalMap·/·dateTimeCanonicalMap·
// (vp-dateTimeLexRep/vp-dateTimeCanRep, §E.3.5/§E.3.6): every valid lexical parses
// and its value renders the spec canonical form, including the timezone spellings
// (Z / +hh:mm / −hh:mm), the leap-day, the endOfDayFrag carry ("…T24:00:00" rolls
// into the next calendar day, even across a month/year boundary), fractional
// seconds, negative and four-digit-padded years, and a large (five-digit) year.
func TestDateTimeParseAndCanonical(t *testing.T) {
	m := mappingFor(t, "dateTime")
	cases := map[string]string{
		"2001-10-26T21:32:52":       "2001-10-26T21:32:52",
		"2001-10-26T21:32:52.125":   "2001-10-26T21:32:52.125",
		"2001-10-26T21:32:52.1250":  "2001-10-26T21:32:52.125", // trailing fractional zeros drop
		"2001-10-26T21:32:52+02:00": "2001-10-26T21:32:52+02:00",
		"2001-10-26T19:32:52Z":      "2001-10-26T19:32:52Z",
		"2001-10-26T19:32:52+00:00": "2001-10-26T19:32:52Z", // +00:00 canonicalizes to Z
		"2001-10-26T21:32:52-05:00": "2001-10-26T21:32:52-05:00",
		"2001-10-26T21:32:52+14:00": "2001-10-26T21:32:52+14:00", // max offset
		"2024-02-29T00:00:00":       "2024-02-29T00:00:00",       // leap-year Feb 29 valid
		"2023-01-01T24:00:00":       "2023-01-02T00:00:00",       // endOfDay carries one day
		"2023-01-31T24:00:00":       "2023-02-01T00:00:00",       // carry across a month boundary
		"2023-12-31T24:00:00":       "2024-01-01T00:00:00",       // carry across a year boundary
		"2023-01-01T24:00:00.0":     "2023-01-02T00:00:00",       // endOfDay with fractional zeros
		"2023-01-01T24:00:00+02:00": "2023-01-02T00:00:00+02:00", // endOfDay keeps the timezone
		"-0045-03-15T00:00:00Z":     "-0045-03-15T00:00:00Z",     // negative year, four-digit pad
		"0001-01-01T00:00:00":       "0001-01-01T00:00:00",       // year 1 pads to four digits
		"0000-01-01T00:00:00":       "0000-01-01T00:00:00",       // XSD 1.1 permits year 0
		"12345-06-07T08:09:10":      "12345-06-07T08:09:10",      // five-digit year is unpadded
	}
	for lex, wantCanon := range cases {
		v, err := m.Parse(lex, nil)
		if err != nil {
			t.Errorf("Parse(%q): unexpected error %v", lex, err)
			continue
		}
		got, err := m.Canonical(v)
		if err != nil {
			t.Errorf("Canonical(%q): unexpected error %v", lex, err)
			continue
		}
		if got != wantCanon {
			t.Errorf("Canonical(Parse(%q)) = %q, want %q", lex, got, wantCanon)
		}
	}
}

// TestDateTimeReject pins the lexical-space boundary (cvc-datatype-valid, §4.1.4):
// the grammar traps (missing 'T', out-of-range month/hour/minute/second, single-
// digit fields, a bad or out-of-range timezone) AND the day-of-month value
// constraint that the grammar regex cannot express (con-dateTime-day/
// con-dateTime-dayValue, §3.3.7.1): Feb 29 in a non-leap year, Feb 30, and April
// 31 must all be rejected, both as the same rule ID as a grammar miss.
func TestDateTimeReject(t *testing.T) {
	m := mappingFor(t, "dateTime")
	for _, lex := range []string{
		"", "2023-01-01", "2023-01-01T00:00", "2023-01-0100:00:00",
		"2023-00-01T00:00:00", "2023-13-01T00:00:00", "2023-1-01T00:00:00",
		"2023-01-00T00:00:00", "2023-01-32T00:00:00",
		"2023-01-01T25:00:00", "2023-01-01T00:60:00", "2023-01-01T00:00:60",
		"2023-02-29T00:00:00", "2024-02-30T00:00:00", "2023-04-31T00:00:00",
		"2023-01-01T00:00:00+15:00", "2023-01-01T00:00:00+02",
		"2023-01-01t00:00:00", " 2023-01-01T00:00:00", "2023-01-01T00:00:00 ",
		"999-01-01T00:00:00", "2023-01-01T24:00:01",
	} {
		_, err := m.Parse(lex, nil)
		if err == nil {
			t.Errorf("Parse(%q): want lexical-space error, got nil", lex)
			continue
		}
		if rule, ok := xsderr.RuleOf(err); !ok || rule != "cvc-datatype-valid" {
			t.Errorf("Parse(%q): rule = %q (ok=%v), want cvc-datatype-valid", lex, rule, ok)
		}
	}
}

// TestDateTimeOrderAndEqIdentical pins the partial order (§D.2.1, over
// ·timeOnTimeline·) together with the Eq/Identical divergence. A timezone-shifted
// pair denoting the same instant is Eq but NOT Identical (their stored
// ·timezoneOffset· differs); definite Less/Greater pairs order correctly; and a
// timezone-less value that a timezone-aware value straddles under the ±14h
// imputation is Incomparable, while one it does not straddle is definitely
// ordered.
func TestDateTimeOrderAndEqIdentical(t *testing.T) {
	m := mappingFor(t, "dateTime")
	parse := func(lex string) value.Value {
		v, err := m.Parse(lex, nil)
		if err != nil {
			t.Fatalf("Parse(%q): %v", lex, err)
		}
		return v
	}
	ordered := func(lex string) value.Ordered {
		o, ok := parse(lex).(value.Ordered)
		if !ok {
			t.Fatalf("dateTime value %q does not implement value.Ordered", lex)
		}
		return o
	}

	// Same instant, different stored offset: Eq (same ·timeOnTimeline·) but not
	// Identical (§2.2.2), the genuine dateTime divergence.
	a := parse("2002-10-10T12:00:00-05:00")
	b := parse("2002-10-10T17:00:00Z")
	if !a.(value.Eq).Eq(b) {
		t.Error("Eq(12:00:00-05:00, 17:00:00Z) = false, want true (same instant)")
	}
	if a.(value.Identical).Identical(b) {
		t.Error("Identical(12:00:00-05:00, 17:00:00Z) = true, want false (offsets differ)")
	}
	// Structurally-equal values are both Eq and Identical.
	if !a.(value.Identical).Identical(parse("2002-10-10T12:00:00-05:00")) {
		t.Error("Identical must hold for structurally equal dateTimes")
	}

	type wc struct {
		a, b string
		want value.Ordering
	}
	for _, c := range []wc{
		{"2002-10-10T12:00:00-05:00", "2002-10-10T17:00:00Z", value.Equal}, // same instant
		{"2001-01-01T00:00:00Z", "2001-01-01T00:00:01Z", value.Less},
		{"2001-01-01T00:00:01Z", "2001-01-01T00:00:00Z", value.Greater},
		{"2001-01-01T00:00:00", "2001-01-01T00:00:00", value.Equal}, // both timezone-less
		// Timezone-less vs timezone-aware, far apart: the ±14h window does not
		// straddle, so the order is definite.
		{"2001-01-01T00:00:00", "2001-06-01T00:00:00Z", value.Less},
		{"2001-06-01T00:00:00Z", "2001-01-01T00:00:00", value.Greater},
		// Timezone-less vs timezone-aware within the same day: the ±14h imputation
		// straddles, so they are incomparable (§D.2.1).
		{"2001-01-01T12:00:00", "2001-01-01T12:00:00Z", value.Incomparable},
		{"2001-01-01T12:00:00Z", "2001-01-01T12:00:00", value.Incomparable},
	} {
		if got := ordered(c.a).Cmp(parse(c.b)); got != c.want {
			t.Errorf("Cmp(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}

	// A non-dateTime argument is Incomparable / unequal / not identical.
	if got := ordered("2001-01-01T00:00:00Z").Cmp("nope"); got != value.Incomparable {
		t.Errorf("Cmp(dateTime, string) = %v, want Incomparable", got)
	}
	if parse("2001-01-01T00:00:00Z").(value.Eq).Eq(42) {
		t.Error("Eq(dateTime, int) = true, want false")
	}
	if parse("2001-01-01T00:00:00Z").(value.Identical).Identical("x") {
		t.Error("Identical(dateTime, string) = true, want false")
	}
}

// TestDateTimeHasTimezone pins value.TimezoneAware (the explicitTimezone facet,
// §4.3.15, reads it): a value with a timezoneFrag reports HasTimezone true; one
// without reports false. Both the Z spelling and an offset carry a timezone.
func TestDateTimeHasTimezone(t *testing.T) {
	m := mappingFor(t, "dateTime")
	for lex, want := range map[string]bool{
		"2001-01-01T00:00:00":       false,
		"2001-01-01T00:00:00Z":      true,
		"2001-01-01T00:00:00+02:00": true,
		"2001-01-01T00:00:00-05:00": true,
	} {
		v, err := m.Parse(lex, nil)
		if err != nil {
			t.Fatalf("Parse(%q): %v", lex, err)
		}
		tz, ok := v.(value.TimezoneAware)
		if !ok {
			t.Fatalf("dateTime value %q does not implement value.TimezoneAware", lex)
		}
		if got := tz.HasTimezone(); got != want {
			t.Errorf("HasTimezone(%q) = %v, want %v", lex, got, want)
		}
	}
}

// TestDateTimeCanonicalForeign pins the warden guardrail: Canonical on a foreign
// value is an *xsderr.Error, never a panic.
func TestDateTimeCanonicalForeign(t *testing.T) {
	m := mappingFor(t, "dateTime")
	_, err := m.Canonical("2001-01-01T00:00:00Z")
	if err == nil {
		t.Fatal("Canonical(foreign): want error, got nil")
	}
	if rule, ok := xsderr.RuleOf(err); !ok || rule != "cvc-datatype-valid" {
		t.Errorf("Canonical(foreign): rule = %q (ok=%v), want cvc-datatype-valid", rule, ok)
	}
}

// TestDateTimeNotLengthed pins the deliberate capability boundary
// (cos-applicable-facets §4.1.5): dateTime is NOT Lengthed/DigitCounted/Scaled —
// no dateTime-applicable facet needs them.
func TestDateTimeNotLengthed(t *testing.T) {
	m := mappingFor(t, "dateTime")
	v, err := m.Parse("2001-01-01T00:00:00Z", nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, ok := v.(value.Lengthed); ok {
		t.Error("dateTime implements value.Lengthed; it must not")
	}
	if _, ok := v.(value.DigitCounted); ok {
		t.Error("dateTime implements value.DigitCounted; it must not")
	}
	if _, ok := v.(value.Scaled); ok {
		t.Error("dateTime implements value.Scaled; it must not")
	}
}
