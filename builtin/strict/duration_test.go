package strict_test

import (
	"testing"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsderr"
)

// TestDurationParseAndCanonical exercises ·durationMap·/·durationCanonicalMap·
// (f-durationMap/f-durationCanMap, §3.3.6.2/§E.2): every valid lexical parses and
// its value renders the spec canonical form, including the zero-suppression edge
// cases (a zero duration → "PT0S", 'T' dropped when no time field, days without
// 'T') and the seconds/hours normalizations (60S → 1M, 36H → 1D12H).
func TestDurationParseAndCanonical(t *testing.T) {
	m := mappingFor(t, "duration")
	cases := map[string]string{
		"P1Y":             "P1Y",
		"P1Y2M":           "P1Y2M",
		"P12M":            "P1Y",  // 12 months normalize to 1 year
		"P1M":             "P1M",  // y=0 branch emits only the month fragment
		"P0M":             "PT0S", // zero duration canonicalizes to PT0S
		"P0Y":             "PT0S", // another spelling of the zero duration
		"PT0S":            "PT0S", // canonical zero, round-trips
		"-P0M":            "PT0S", // signed zero is the signless zero
		"P1D":             "P1D",  // a day stands alone with no 'T'
		"PT1H":            "PT1H",
		"PT1M":            "PT1M",    // minute after 'T'
		"PT36H":           "P1DT12H", // hours normalize into days
		"PT60S":           "PT1M",    // seconds normalize into minutes
		"PT61S":           "PT1M1S",
		"P1DT2H3M4S":      "P1DT2H3M4S",
		"PT1.5S":          "PT1.5S", // fractional seconds keep the point
		"PT0.5S":          "PT0.5S", // integer part is 0 but present
		"P1347Y":          "P1347Y", // large year value (msData duration003)
		"-P1Y2M3DT4H5M6S": "-P1Y2M3DT4H5M6S",
		"-PT1M":           "-PT1M", // sign on a pure day-time duration
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

// TestDurationReject pins the lexical-space boundary (cvc-datatype-valid,
// §4.1.4): bare "P" (no field) and "PT" (T final) are the two grammar traps a
// naive all-optional transcription would wrongly accept; the rest cover a
// missing 'P', a per-field sign, out-of-order fields, and stray whitespace
// (whiteSpace is a separate pipeline stage).
func TestDurationReject(t *testing.T) {
	m := mappingFor(t, "duration")
	for _, lex := range []string{
		"P", "PT", "", "P-1Y", "-P", "1Y", "P1S", "PT1D", "P1Y2MT",
		"P1M2Y", "PT1S2M", "p1y", " P1Y", "P1Y ", "P1.5Y", "P1YT",
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

// TestDurationEqIdentical pins that Eq and Identical are structural over the
// (·months·, ·seconds·) tuple (§3.3.6.1): distinct lexicals denoting the same
// value are equal AND identical (P1Y == P12M, PT60S == PT1M, the signed and
// unsigned zero), different values are not, and a foreign value never matches.
func TestDurationEqIdentical(t *testing.T) {
	m := mappingFor(t, "duration")
	parse := func(lex string) value.Value {
		v, err := m.Parse(lex, nil)
		if err != nil {
			t.Fatalf("Parse(%q): %v", lex, err)
		}
		return v
	}

	same := [][2]string{{"P1Y", "P12M"}, {"PT60S", "PT1M"}, {"P0M", "PT0S"}, {"-P0D", "P0Y"}}
	for _, pair := range same {
		a, b := parse(pair[0]), parse(pair[1])
		if eq, ok := a.(value.Eq); !ok || !eq.Eq(b) {
			t.Errorf("Eq(%q, %q) = false, want true", pair[0], pair[1])
		}
		if id, ok := a.(value.Identical); !ok || !id.Identical(b) {
			t.Errorf("Identical(%q, %q) = false, want true", pair[0], pair[1])
		}
	}

	diff := [][2]string{{"P1M", "P30D"}, {"P1Y", "P1Y1D"}, {"PT1S", "-PT1S"}, {"P1D", "PT23H"}}
	for _, pair := range diff {
		a, b := parse(pair[0]), parse(pair[1])
		if a.(value.Eq).Eq(b) {
			t.Errorf("Eq(%q, %q) = true, want false", pair[0], pair[1])
		}
		if a.(value.Identical).Identical(b) {
			t.Errorf("Identical(%q, %q) = true, want false", pair[0], pair[1])
		}
	}

	// A foreign value (a string, another value space) never matches.
	if parse("P1Y").(value.Eq).Eq("P1Y") {
		t.Error("Eq(duration, string) = true, want false")
	}
	if parse("P1Y").(value.Identical).Identical(42) {
		t.Error("Identical(duration, int) = true, want false")
	}
}

// TestDurationOrder pins the four-reference-dateTime partial order (§3.3.6.1):
// comparable pairs order correctly, the two spec-given incomparable pairs (P1M
// vs P30D and P1M vs P31D) are Incomparable, equal values compare Equal, and a
// foreign argument is Incomparable (rf-ordered).
func TestDurationOrder(t *testing.T) {
	m := mappingFor(t, "duration")
	parse := func(lex string) value.Ordered {
		v, err := m.Parse(lex, nil)
		if err != nil {
			t.Fatalf("Parse(%q): %v", lex, err)
		}
		o, ok := v.(value.Ordered)
		if !ok {
			t.Fatalf("duration value %q does not implement value.Ordered", lex)
		}
		return o
	}

	type wc struct {
		a, b string
		want value.Ordering
	}
	for _, c := range []wc{
		{"P1Y", "P2Y", value.Less},
		{"P2Y", "P1Y", value.Greater},
		{"P1Y", "P12M", value.Equal},
		{"PT1M", "PT60S", value.Equal},
		{"P1D", "PT25H", value.Less},
		{"-P1D", "P1D", value.Less},
		{"P1M", "P27D", value.Greater},      // a month exceeds 27 days at every reference
		{"P1M", "P30D", value.Incomparable}, // §3.3.6.1's own incomparable example
		{"P1M", "P31D", value.Incomparable}, // §2.2.3's second incomparable example
	} {
		if got := parse(c.a).Cmp(parse(c.b)); got != c.want {
			t.Errorf("Cmp(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}

	// A non-duration argument is Incomparable, not a spurious order.
	if got := parse("P1Y").Cmp("P1Y"); got != value.Incomparable {
		t.Errorf("Cmp(duration, string) = %v, want Incomparable", got)
	}
}

// TestDurationCanonicalForeign pins the warden guardrail: Canonical on a foreign
// value is an *xsderr.Error, never a panic.
func TestDurationCanonicalForeign(t *testing.T) {
	m := mappingFor(t, "duration")
	_, err := m.Canonical("P1Y")
	if err == nil {
		t.Fatal("Canonical(foreign): want error, got nil")
	}
	if rule, ok := xsderr.RuleOf(err); !ok || rule != "cvc-datatype-valid" {
		t.Errorf("Canonical(foreign): rule = %q (ok=%v), want cvc-datatype-valid", rule, ok)
	}
}

// TestDurationNotLengthed pins the deliberate capability boundary
// (cos-applicable-facets §4.1.5): duration is NOT Lengthed/DigitCounted/Scaled/
// TimezoneAware — no duration-applicable facet needs them.
func TestDurationNotLengthed(t *testing.T) {
	m := mappingFor(t, "duration")
	v, err := m.Parse("P1Y", nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, ok := v.(value.Lengthed); ok {
		t.Error("duration implements value.Lengthed; it must not")
	}
	if _, ok := v.(value.DigitCounted); ok {
		t.Error("duration implements value.DigitCounted; it must not")
	}
	if _, ok := v.(value.Scaled); ok {
		t.Error("duration implements value.Scaled; it must not")
	}
	if _, ok := v.(value.TimezoneAware); ok {
		t.Error("duration implements value.TimezoneAware; it must not")
	}
}
