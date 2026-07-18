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

// scaledStub is a test-only value.Scaled: a numeric value carrying an explicit
// ·scale· (present), or a special (present=false) whose ·scale· is absent —
// modeling precisionDecimal's numeric vs NaN/±INF arms without pulling in the
// strict backend.
type scaledStub struct {
	scale   int
	present bool
}

func (s scaledStub) Scale() (int, bool) { return s.scale, s.present }

// TestScaleFacetCheckValue exercises both polarities of both scale facets
// (cvc-maxScale-valid xsd-precisionDecimal.md §4.2.3, cvc-minScale-valid §4.3.3)
// and the vacuous-pass clause: a special value whose ·scale· is absent is
// facet-valid w.r.t. both facets regardless of {value}. Facet {value}s cover a
// negative bound too — proving the integer (not nonNegativeInteger) domain.
func TestScaleFacetCheckValue(t *testing.T) {
	cases := []struct {
		name     string
		kind     xsd.FacetKind
		limit    string
		v        scaledStub
		wantRule xsderr.Rule // "" means accept
	}{
		{"maxScale within bound", xsd.FacetMaxScale, "2", scaledStub{scale: 2, present: true}, ""},
		{"maxScale below bound", xsd.FacetMaxScale, "2", scaledStub{scale: 1, present: true}, ""},
		{"maxScale exceeds bound", xsd.FacetMaxScale, "2", scaledStub{scale: 3, present: true}, "cvc-maxScale-valid"},
		{"maxScale negative bound rejects", xsd.FacetMaxScale, "-1", scaledStub{scale: 0, present: true}, "cvc-maxScale-valid"},
		{"maxScale negative bound accepts", xsd.FacetMaxScale, "-1", scaledStub{scale: -2, present: true}, ""},
		{"minScale within bound", xsd.FacetMinScale, "2", scaledStub{scale: 2, present: true}, ""},
		{"minScale above bound", xsd.FacetMinScale, "2", scaledStub{scale: 3, present: true}, ""},
		{"minScale below bound", xsd.FacetMinScale, "2", scaledStub{scale: 1, present: true}, "cvc-minScale-valid"},
		{"minScale negative bound accepts", xsd.FacetMinScale, "-1", scaledStub{scale: -1, present: true}, ""},
		{"minScale negative bound rejects", xsd.FacetMinScale, "-1", scaledStub{scale: -2, present: true}, "cvc-minScale-valid"},
		// Vacuous pass (clause 2): a special (absent ·scale·) passes both facets
		// regardless of {value}, even a bound that would reject any numeric scale.
		{"maxScale special vacuous", xsd.FacetMaxScale, "-5", scaledStub{present: false}, ""},
		{"minScale special vacuous", xsd.FacetMinScale, "5", scaledStub{present: false}, ""},
	}
	for _, c := range cases {
		sf, err := newScaleFacet(xsd.NewFacet(c.kind, []string{c.limit}, false))
		if err != nil {
			t.Fatalf("%s: newScaleFacet: %v", c.name, err)
		}
		got := sf.CheckValue(c.v)
		if c.wantRule == "" {
			if got != nil {
				t.Errorf("%s: CheckValue = %v, want accept", c.name, got)
			}
			continue
		}
		if got == nil {
			t.Errorf("%s: CheckValue = nil, want reject with %s", c.name, c.wantRule)
			continue
		}
		if r, _ := xsderr.RuleOf(got); r != c.wantRule {
			t.Errorf("%s: CheckValue charged %s, want %s", c.name, r, c.wantRule)
		}
	}
}

// TestScaleFacetNonScaledPanics confirms a candidate lacking the Scaled
// capability under a scale facet is a caught schema-construction error
// (cos-applicable-facets §4.1.5), not a validity verdict — the boundFacet
// panic convention.
func TestScaleFacetNonScaledPanics(t *testing.T) {
	sf, err := newScaleFacet(xsd.NewFacet(xsd.FacetMaxScale, []string{"2"}, false))
	if err != nil {
		t.Fatalf("newScaleFacet: %v", err)
	}
	defer func() {
		if recover() == nil {
			t.Error("CheckValue(non-Scaled): want panic, got none")
		}
	}()
	_ = sf.CheckValue("not scaled")
}

// TestNewScaleFacetRejectsBadValue confirms facetInt charges the per-facet rule
// on a non-integer {value} and on a wrong value count.
func TestNewScaleFacetRejectsBadValue(t *testing.T) {
	if _, err := newScaleFacet(xsd.NewFacet(xsd.FacetMinScale, []string{"x"}, false)); err == nil {
		t.Error("newScaleFacet(non-integer): want error, got nil")
	} else if r, _ := xsderr.RuleOf(err); r != "cvc-minScale-valid" {
		t.Errorf("newScaleFacet(non-integer) charged %s, want cvc-minScale-valid", r)
	}
	if _, err := newScaleFacet(xsd.NewFacet(xsd.FacetMaxScale, []string{"1", "2"}, false)); err == nil {
		t.Error("newScaleFacet(two values): want error, got nil")
	}
}
