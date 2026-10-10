package value_test

import (
	"errors"
	"testing"

	"github.com/kud360/goxsd8/builtin"
	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// TestBoundRejectionNamesTheLimit pins what a bound-facet rejection says: the
// facet, its {value} as the schema wrote it — "+007", not the parsed 7 — and the
// order relation its Validation Rule requires, cited inline (#2369). The
// float row is the incomparable branch: NaN is excluded from the restricted
// value space (Datatypes §3.3.4.3 Note), not ordered against the limit.
func TestBoundRejectionNamesTheLimit(t *testing.T) {
	b := strict.New()
	seeded, err := builtin.Seed(b)
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}
	builtins := map[string]*xsd.SimpleType{}
	for _, st := range seeded {
		builtins[st.Name().Local] = st
	}
	restrict := func(base string, kind xsd.FacetKind, limit string) *xsd.SimpleType {
		t.Helper()
		st, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:t", Local: "Bounded"}, xsd.RestrictionDerivation{},
			xsd.OwnedSimpleType{Definition: builtins[base]}, []xsd.Facet{xsd.NewFacet(kind, []string{limit}, false)}, nil)
		if err != nil {
			t.Fatalf("NewSimpleType(%s %s %q): %v", base, kind, limit, err)
		}
		if err := st.CheckDerivation(ownedChain{}); err != nil {
			t.Fatalf("CheckDerivation(%s %s %q): %v", base, kind, limit, err)
		}
		return st
	}
	for _, tc := range []struct {
		name    string
		st      *xsd.SimpleType
		lexical string
		rule    xsderr.Rule
		want    string
	}{
		{"maxInclusive", restrict("int", xsd.FacetMaxInclusive, "+007"), "8", "cvc-maxInclusive-valid",
			`value violates the maxInclusive facet, whose {value} is "+007", but cvc-maxInclusive-valid requires a value less than or equal to it`},
		{"maxExclusive", restrict("int", xsd.FacetMaxExclusive, "+007"), "7", "cvc-maxExclusive-valid",
			`value violates the maxExclusive facet, whose {value} is "+007", but cvc-maxExclusive-valid requires a value less than it`},
		{"minInclusive", restrict("int", xsd.FacetMinInclusive, "+007"), "6", "cvc-minInclusive-valid",
			`value violates the minInclusive facet, whose {value} is "+007", but cvc-minInclusive-valid requires a value greater than or equal to it`},
		{"minExclusive", restrict("int", xsd.FacetMinExclusive, "+007"), "7", "cvc-minExclusive-valid",
			`value violates the minExclusive facet, whose {value} is "+007", but cvc-minExclusive-valid requires a value greater than it`},
		{"incomparable", restrict("float", xsd.FacetMaxInclusive, "+7.0"), "NaN", "cvc-maxInclusive-valid",
			`value is incomparable with the maxInclusive facet's {value} "+7.0", so it is excluded from the restricted value space, but cvc-maxInclusive-valid requires a value less than or equal to it`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := value.ValidateLexical(b, ownedChain{}, tc.st, tc.lexical, nil, declineEvery{})
			var xe *xsderr.Error
			if !errors.As(err, &xe) || xe.Rule != tc.rule {
				t.Fatalf("ValidateLexical(%q) = %v, want a %s rejection", tc.lexical, err, tc.rule)
			}
			if xe.Msg != tc.want {
				t.Errorf("ValidateLexical(%q) Msg =\n%s\nwant\n%s", tc.lexical, xe.Msg, tc.want)
			}
		})
	}
}
