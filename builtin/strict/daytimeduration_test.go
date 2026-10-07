package strict_test

import (
	"fmt"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xpath"
	"github.com/kud360/goxsd8/xsderr"
)

// TestDayTimeDurationParseAndCanonical exercises ·dayTimeDurationMap· /
// ·dayTimeDurationCanonicalMap· (§3.4.27.1/§E.2): the day-time half parses and
// renders duration's canonical day-time fragment, including the zero value ("T0S",
// which — unlike yearMonthDuration's zero — IS in this type's lexical space), the
// hours→days / seconds→minutes normalizations, and the "PT5M" minutes-after-'T'
// case the design flags as easy to drop.
func TestDayTimeDurationParseAndCanonical(t *testing.T) {
	m := mappingFor(t, "dayTimeDuration")
	cases := map[string]string{
		"P1D":        "P1D",
		"PT1H":       "PT1H",
		"PT5M":       "PT5M", // minutes after 'T' (not the pre-'T' month branch)
		"PT36H":      "P1DT12H",
		"PT60S":      "PT1M",
		"P1DT2H3M4S": "P1DT2H3M4S",
		"PT1.5S":     "PT1.5S",
		"PT0S":       "PT0S", // the zero value canonicalizes within [^YM]*(T.*)?
		"P0D":        "PT0S",
		"-P0D":       "PT0S", // signed zero is the signless zero
		"-PT1M":      "-PT1M",
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

// TestDayTimeDurationReject pins the narrower lexical space (§3.4.27.1) and the
// cvc-datatype-valid clause each rejection cites: a literal outside duration's
// lexical space fails clause 2.1, and a duration literal carrying the year-month
// half — including the pre-'T' month branch a verbatim parseDuration alias would
// wrongly accept — fails the [^YM]*(T.*)? pattern facet, clause 1.
func TestDayTimeDurationReject(t *testing.T) {
	m := mappingFor(t, "dayTimeDuration")
	pattern := `%q has a year or month field, so it does not match dayTimeDuration's pattern facet [^YM]*(T.*)?, which cvc-datatype-valid clause 1 requires it to match`
	primitive := `%q is not in the lexical space of duration, the primitive of dayTimeDuration, which cvc-datatype-valid clause 2.1 requires it to be in`
	for _, c := range []struct{ lex, msg string }{
		{"P1Y", pattern}, {"P1M", pattern}, {"P1Y2M", pattern}, {"P1YT2H", pattern},
		{"P1MT5M", pattern}, {"P1Y1D", pattern},
		{"P", primitive}, {"PT", primitive}, {"", primitive}, {"-P", primitive},
		{"1D", primitive}, {"PT1.5H", primitive}, {"p1d", primitive},
		{" PT1H", primitive}, {"PT1H ", primitive},
	} {
		_, err := m.Parse(c.lex, nil)
		if err == nil {
			t.Errorf("Parse(%q): want lexical-space error, got nil", c.lex)
			continue
		}
		want := "?: [cvc-datatype-valid] " + fmt.Sprintf(c.msg, c.lex)
		if err.Error() != want {
			t.Errorf("Parse(%q) = %q, want %q", c.lex, err.Error(), want)
		}
	}
}

// TestDayTimeDurationSeededPattern is the #122-shaped corroboration and the mirror
// of TestYearMonthDurationSeededPattern: the REAL seeded builtin xs:dayTimeDuration
// — carrying the generated fixed pattern facet [^YM]*(T.*)? (§3.4.27.2) — enforces
// that pattern through value.ValidateLexical. A day-time literal is accepted; a
// year-month literal (a 'Y'/pre-'T' 'M' character) is rejected as
// cvc-pattern-valid (§4.3.4.4) — accepted where it is REJECTED for the mirror
// yearMonthDuration.
func TestDayTimeDurationSeededPattern(t *testing.T) {
	st := seededType(t, "dayTimeDuration")

	if _, err := value.ValidateLexical(strict.New(), noSchema{}, st, "P1DT2H3M4S", nil, xpath.FacetAssertions()); err != nil {
		t.Fatalf("day-time dayTimeDuration should validate: %v", err)
	}

	_, err := value.ValidateLexical(strict.New(), noSchema{}, st, "P1Y", nil, xpath.FacetAssertions())
	if err == nil {
		t.Fatal("year-month literal must be rejected for dayTimeDuration, got nil")
	}
	if rule, ok := xsderr.RuleOf(err); !ok || rule != "cvc-pattern-valid" {
		t.Errorf("year-month dayTimeDuration: rule = %q (ok=%v), want cvc-pattern-valid", rule, ok)
	}
}
