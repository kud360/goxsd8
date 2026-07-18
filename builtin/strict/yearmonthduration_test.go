package strict_test

import (
	"testing"

	"github.com/kud360/goxsd8/builtin"
	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// TestYearMonthDurationParseAndCanonical exercises ·yearMonthDurationMap· /
// ·yearMonthDurationCanonicalMap· (§3.4.26.1/§E.2): the year-month half parses and
// renders duration's canonical year-month fragment, with the 12M→1Y normalization
// and the y=0 branch that emits only the month fragment.
func TestYearMonthDurationParseAndCanonical(t *testing.T) {
	m := mappingFor(t, "yearMonthDuration")
	cases := map[string]string{
		"P1Y":    "P1Y",
		"P1Y2M":  "P1Y2M",
		"P12M":   "P1Y", // 12 months normalize to 1 year
		"P1M":    "P1M", // y=0 branch emits only the month fragment
		"P18M":   "P1Y6M",
		"-P1Y2M": "-P1Y2M",
		"-P13M":  "-P1Y1M",
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

// TestYearMonthDurationReject pins the narrower lexical space (cvc-datatype-valid,
// §3.4.26.1): duration lexicals carrying the day-time half — the exact literals a
// verbatim parseDuration alias would wrongly accept as a yearMonthDuration — are
// rejected here, alongside the shared duration grammar traps.
func TestYearMonthDurationReject(t *testing.T) {
	m := mappingFor(t, "yearMonthDuration")
	for _, lex := range []string{
		"P1D", "PT5H", "PT5M", "P1DT2H", "P1Y1D", "PT0S", "P0DT0S",
		"P", "PT", "", "-P", "1Y", "P1.5Y", "p1y", " P1Y", "P1Y ",
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

// TestYearMonthDurationZeroNoCanonical pins §3.4.26.1's Note: the zero value
// (·months·=0, spelled "P0Y"/"P0M"/…) is a VALID lexical that Parse accepts, yet
// it has NO canonical representation (duration's "PT0S" is outside [^DT]*). Parse
// must succeed; Canonical must return a NON-nil error that is NOT a validity
// verdict (no xsderr rule), since the value is legal, not invalid.
func TestYearMonthDurationZeroNoCanonical(t *testing.T) {
	m := mappingFor(t, "yearMonthDuration")
	for _, lex := range []string{"P0Y", "P0M", "-P0M", "P0Y0M"} {
		v, err := m.Parse(lex, nil)
		if err != nil {
			t.Fatalf("Parse(%q): the zero value is a valid lexical, got error %v", lex, err)
		}
		got, err := m.Canonical(v)
		if err == nil {
			t.Errorf("Canonical(%q) = %q, want the no-canonical-representation error", lex, got)
			continue
		}
		if rule, ok := xsderr.RuleOf(err); ok {
			t.Errorf("Canonical(%q): got validity rule %q, want a plain (non-verdict) error", lex, rule)
		}
	}
}

// TestYearMonthDurationSeededPattern is the #122-shaped corroboration: the REAL
// seeded builtin xs:yearMonthDuration — carrying the generated fixed pattern facet
// [^DT]* (§3.4.26.2) — enforces that pattern through value.ValidateLexical. A
// year-month literal is accepted; a day-time literal (a 'D'/'T' character) is
// rejected as cvc-pattern-valid (§4.3.4.4), proving the fixed facet reaches
// enforcement for the seeded type, and REJECTED where it is ACCEPTED for the
// mirror dayTimeDuration.
func TestYearMonthDurationSeededPattern(t *testing.T) {
	st := seededType(t, "yearMonthDuration")

	if _, err := value.ValidateLexical(strict.New(), st, "P1Y2M", nil); err != nil {
		t.Fatalf("year-month yearMonthDuration should validate: %v", err)
	}

	_, err := value.ValidateLexical(strict.New(), st, "P1DT2H", nil)
	if err == nil {
		t.Fatal("day-time literal must be rejected for yearMonthDuration, got nil")
	}
	if rule, ok := xsderr.RuleOf(err); !ok || rule != "cvc-pattern-valid" {
		t.Errorf("day-time yearMonthDuration: rule = %q (ok=%v), want cvc-pattern-valid", rule, ok)
	}
}

// seededType returns the seeded builtin xs:<local> SimpleType from a strict Seed,
// so a test can drive value.ValidateLexical over the real generated typespec
// (facets included), not a synthetic derive()-built type.
func seededType(t *testing.T, local string) *xsd.SimpleType {
	t.Helper()
	components, err := builtin.Seed(strict.New())
	if err != nil {
		t.Fatalf("Seed(strict.New()): %v", err)
	}
	want := xsd.QName{Space: xsd.XMLSchemaNS, Local: local}
	for _, c := range components {
		if c.Name() == want {
			return c
		}
	}
	t.Fatalf("Seed did not return the xs:%s component", local)
	return nil
}
