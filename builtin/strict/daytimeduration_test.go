package strict_test

import (
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/value"
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

// TestDayTimeDurationReject pins the narrower lexical space (cvc-datatype-valid,
// §3.4.27.1): duration lexicals carrying the year-month half — including the
// pre-'T' month branch a verbatim parseDuration alias would wrongly accept — are
// rejected, alongside the shared duration grammar traps.
func TestDayTimeDurationReject(t *testing.T) {
	m := mappingFor(t, "dayTimeDuration")
	for _, lex := range []string{
		"P1Y", "P1M", "P1Y2M", "P1YT2H", "P1MT5M", "P1Y1D",
		"P", "PT", "", "-P", "1D", "PT1.5H", "p1d", " PT1H", "PT1H ",
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

// TestDayTimeDurationSeededPattern is the #122-shaped corroboration and the mirror
// of TestYearMonthDurationSeededPattern: the REAL seeded builtin xs:dayTimeDuration
// — carrying the generated fixed pattern facet [^YM]*(T.*)? (§3.4.27.2) — enforces
// that pattern through value.ValidateLexical. A day-time literal is accepted; a
// year-month literal (a 'Y'/pre-'T' 'M' character) is rejected as
// cvc-pattern-valid (§4.3.4.4) — accepted where it is REJECTED for the mirror
// yearMonthDuration.
func TestDayTimeDurationSeededPattern(t *testing.T) {
	st := seededType(t, "dayTimeDuration")

	if _, err := value.ValidateLexical(strict.New(), st, "P1DT2H3M4S", nil); err != nil {
		t.Fatalf("day-time dayTimeDuration should validate: %v", err)
	}

	_, err := value.ValidateLexical(strict.New(), st, "P1Y", nil)
	if err == nil {
		t.Fatal("year-month literal must be rejected for dayTimeDuration, got nil")
	}
	if rule, ok := xsderr.RuleOf(err); !ok || rule != "cvc-pattern-valid" {
		t.Errorf("year-month dayTimeDuration: rule = %q (ok=%v), want cvc-pattern-valid", rule, ok)
	}
}
