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

// TestEnumerationRejectionNamesItsType pins what a cvc-enumeration-valid
// rejection (§4.3.5.4) says: the enumeration facet and the type that declares
// it, with no inheritance when the facet is the checked type's own (Color) and
// the checked type's inheritance when it is not (Shade, which declares only a
// maxLength) (#3335), and the declaring type's members as written, in document
// order, escaped as a cvc-pattern-valid message escapes its members (#3582).
func TestEnumerationRejectionNamesItsType(t *testing.T) {
	b := strict.New()
	seeded, err := builtin.Seed(b)
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}
	var str, xsInt *xsd.SimpleType
	for _, st := range seeded {
		switch st.Name().Local {
		case "string":
			str = st
		case "int":
			xsInt = st
		}
	}
	derive := func(local string, base *xsd.SimpleType, facet xsd.Facet) *xsd.SimpleType {
		t.Helper()
		st, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:t", Local: local}, xsd.RestrictionDerivation{},
			xsd.OwnedSimpleType{Definition: base}, []xsd.Facet{facet}, nil)
		if err != nil {
			t.Fatalf("NewSimpleType(%s): %v", local, err)
		}
		if err := st.CheckDerivation(ownedChain{}); err != nil {
			t.Fatalf("CheckDerivation(%s): %v", local, err)
		}
		return st
	}
	color := derive("Color", str, xsd.NewEnumerationFacet([]xsd.EnumerationMember{
		xsd.NewEnumerationMember("red", nil, nil), xsd.NewEnumerationMember("green", nil, nil),
	}))
	shade := derive("Shade", color, xsd.NewFacet(xsd.FacetMaxLength, []string{"5"}, false))
	small := derive("Small", xsInt, xsd.NewEnumerationFacet([]xsd.EnumerationMember{
		xsd.NewEnumerationMember("+007", nil, nil), xsd.NewEnumerationMember("5", nil, nil),
	}))
	punct := derive("Punct", str, xsd.NewEnumerationFacet([]xsd.EnumerationMember{
		xsd.NewEnumerationMember(`a"b`, nil, nil), xsd.NewEnumerationMember("x, y", nil, nil),
	}))
	for _, tc := range []struct {
		name string
		st   *xsd.SimpleType
		lit  string
		want string
	}{
		{"own", color, "blue",
			`value is not equal or identical to any member of the enumeration facet of the simple type {urn:t}Color, whose {value} holds the values written "red", "green", but cvc-enumeration-valid requires it to be equal or identical to one`},
		{"inherited", shade, "blue",
			`value is not equal or identical to any member of the enumeration facet that the simple type {urn:t}Shade inherits from the simple type {urn:t}Color, whose {value} holds the values written "red", "green", but cvc-enumeration-valid requires it to be equal or identical to one`},
		// The member is quoted as written, not as its parsed value renders.
		{"spelling kept", small, "8",
			`value is not equal or identical to any member of the enumeration facet of the simple type {urn:t}Small, whose {value} holds the values written "+007", "5", but cvc-enumeration-valid requires it to be equal or identical to one`},
		// A quote inside a member is escaped and a `", "` inside one stays
		// inside its quotes, as in a cvc-pattern-valid message.
		{"escaped", punct, "z",
			`value is not equal or identical to any member of the enumeration facet of the simple type {urn:t}Punct, whose {value} holds the values written "a&quot;b", "x, y", but cvc-enumeration-valid requires it to be equal or identical to one`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := value.ValidateLexical(b, ownedChain{}, tc.st, tc.lit, nil, declineEvery{})
			var xe *xsderr.Error
			if !errors.As(err, &xe) || xe.Rule != "cvc-enumeration-valid" {
				t.Fatalf("ValidateLexical(%s, %q) = %v, want a cvc-enumeration-valid rejection", tc.st.Name(), tc.lit, err)
			}
			if xe.Msg != tc.want {
				t.Errorf("ValidateLexical(%s, %q) Msg =\n%s\nwant\n%s", tc.st.Name(), tc.lit, xe.Msg, tc.want)
			}
		})
	}
	// Matching compares values, not the spellings a rejection quotes: "7" is
	// the member written "+007".
	t.Run("value match", func(t *testing.T) {
		if _, err := value.ValidateLexical(b, ownedChain{}, small, "7", nil, declineEvery{}); err != nil {
			t.Errorf("ValidateLexical(%s, %q) = %v, want no error", small.Name(), "7", err)
		}
	})
}
