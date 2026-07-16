package strict_test

import (
	"testing"

	"github.com/kud360/goxsd8/builtin"
	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// TestDateTimeStampMappingAliasesDateTime pins that xs:dateTimeStamp (§3.4.28) is
// backend-mapped by reusing dateTime's mapping verbatim: Mapping reports ok with a
// non-nil Parse and Canonical (dateTimeStamp defines no separate canonical mapping,
// §3.4.28.1), and a tz-bearing literal round-trips exactly as it does for dateTime.
func TestDateTimeStampMappingAliasesDateTime(t *testing.T) {
	m, ok := strict.New().Mapping(xsd.QName{Space: xsd.XMLSchemaNS, Local: "dateTimeStamp"})
	if !ok {
		t.Fatal("strict backend does not map xs:dateTimeStamp")
	}
	if m.Parse == nil {
		t.Fatal("xs:dateTimeStamp mapping has nil Parse")
	}
	if m.Canonical == nil {
		t.Fatal("xs:dateTimeStamp mapping has nil Canonical")
	}
	const lex = "2002-10-10T12:00:00Z"
	v, err := m.Parse(lex, nil)
	if err != nil {
		t.Fatalf("Parse(%q): unexpected error %v", lex, err)
	}
	got, err := m.Canonical(v)
	if err != nil {
		t.Fatalf("Canonical(%q): unexpected error %v", lex, err)
	}
	if got != lex {
		t.Errorf("Canonical(Parse(%q)) = %q, want %q", lex, got, lex)
	}
}

// TestDateTimeStampSeededExplicitTimezone is the load-bearing acceptance test for
// #122: the REAL seeded builtin xs:dateTimeStamp type — carrying the generated
// typespec's fixed explicitTimezone=required facet (§3.4.28) — enforces the
// mandatory timezone through value.ValidateLexical. A tz-bearing literal is
// accepted; a tz-absent one is rejected as cvc-explicitTimezone-valid (§4.3.14.3).
// This proves the fixed facet reaches enforcement for the actual seeded type, not
// only for the synthetic derive()-built type TestExplicitTimezoneFacet exercises.
func TestDateTimeStampSeededExplicitTimezone(t *testing.T) {
	components, err := builtin.Seed(strict.New())
	if err != nil {
		t.Fatalf("Seed(strict.New()): %v", err)
	}
	want := xsd.QName{Space: xsd.XMLSchemaNS, Local: "dateTimeStamp"}
	var dts *xsd.SimpleType
	for _, c := range components {
		if c.Name() == want {
			dts = c
			break
		}
	}
	if dts == nil {
		t.Fatal("Seed did not return the xs:dateTimeStamp component")
	}

	if _, err := value.ValidateLexical(strict.New(), dts, "2002-10-10T12:00:00Z", nil); err != nil {
		t.Fatalf("tz-bearing dateTimeStamp should validate: %v", err)
	}

	_, err = value.ValidateLexical(strict.New(), dts, "2002-10-10T12:00:00", nil)
	if err == nil {
		t.Fatal("tz-absent dateTimeStamp must be rejected, got nil")
	}
	if rule, ok := xsderr.RuleOf(err); !ok || rule != "cvc-explicitTimezone-valid" {
		t.Errorf("tz-absent dateTimeStamp: rule = %q (ok=%v), want cvc-explicitTimezone-valid", rule, ok)
	}
}
