package value

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// TestLengthExemptPrimitive checks the clause-1.3 predicate (cvc-length-valid
// §4.3.1.3, cvc-minLength-valid §4.3.2.3, cvc-maxLength-valid §4.3.3.3): only a
// QName or NOTATION {primitive type definition} is exempt, keyed off the atomic
// {variety}'s Primitive — a derivation of QName/NOTATION is still exempt, while
// string and a non-atomic type are not, and the predicate never panics.
func TestLengthExemptPrimitive(t *testing.T) {
	qnamePrim := primType(t, "QName", "collapse")
	notationPrim := primType(t, "NOTATION", "collapse")
	stringPrim := primType(t, "string", "preserve")

	derivedQName, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: "myqname"},
		xsd.Atomic{Primitive: qnamePrim}, qnamePrim, nil, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(myqname): %v", err)
	}

	cases := []struct {
		name string
		st   *xsd.SimpleType
		want bool
	}{
		{"QName primitive", qnamePrim, true},
		{"NOTATION primitive", notationPrim, true},
		{"QName restriction", derivedQName, true},
		{"string primitive", stringPrim, false},
	}
	for _, c := range cases {
		if got := lengthExemptPrimitive(c.st); got != c.want {
			t.Errorf("lengthExemptPrimitive(%s) = %v, want %v", c.name, got, c.want)
		}
	}

	// A nil {variety} (xs:anySimpleType) is non-atomic: not exempt, no panic.
	if lengthExemptPrimitive(xsd.AnySimpleType()) {
		t.Error("lengthExemptPrimitive(anySimpleType) = true, want false (non-atomic variety)")
	}
}
