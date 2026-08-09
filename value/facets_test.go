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
		xsd.RestrictionDerivation{}, qnamePrim, nil, nil)
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

// plainBackend maps every name it holds to a Mapping whose Parse yields the literal
// itself — a bare Go string, a Value implementing NONE of the facet capabilities
// (no Ordered, Lengthed, DigitCounted, Scaled or TimezoneAware). It is how a test
// reaches the facet-precondition cohort through the real pipeline: pairing any value
// facet with this value space is exactly the cos-applicable-facets (§4.1.5) violation
// builtin.CheckSimpleTypeRestriction exists to reject.
type plainBackend map[xsd.QName]bool

func (b plainBackend) Mapping(typ xsd.QName) (Mapping, bool) {
	if !b[typ] {
		return Mapping{}, false
	}
	return Mapping{Parse: func(lexical string, _ Context) (Value, error) {
		return lexical, nil
	}}, true
}

// preconditionType builds a primitive carrying a length facet (§4.3.1) beside its
// whiteSpace entry, so a plainBackend-governed literal reaches lengthFacet.CheckValue
// without the Lengthed capability and ValidateLexical reports a facet-precondition
// fault for EVERY literal.
//
// length rather than a bound facet, deliberately: a length facet's {value} is a plain
// count, so compile() succeeds and the fault arises in the VALUE-FACET stage — which
// is the stage the three discriminating callers (valueSpace.ValidDefault's gate 4,
// dispatchUnion, checkEnumerationRestriction) actually have to handle. A bound facet
// would fault inside compile() instead, where ValidDefault's gate 3 would mask gate 4.
func preconditionType(t *testing.T, local string) *xsd.SimpleType {
	t.Helper()
	st, err := xsd.NewPrimitiveType(xsderr.Loc{}, xsd.QName{Space: xsd.XMLSchemaNS, Local: local},
		[]xsd.Facet{
			xsd.NewFacet(xsd.FacetWhiteSpace, []string{"collapse"}, true),
			xsd.NewFacet(xsd.FacetLength, []string{"2"}, false),
		}, nil)
	if err != nil {
		t.Fatalf("NewPrimitiveType(%s): %v", local, err)
	}
	return st
}

// TestValueFacetCapabilityFaultsAreErrors pins five of the six cos-applicable-facets
// (§4.1.5) capability sites: a candidate lacking the capability its facet needs is a
// caught schema-construction fault, REPORTED as an *xsderr.Error charged to
// cos-applicable-facets and discriminable through IsFacetPrecondition — never a
// panic, and never a validity verdict about the candidate.
//
// The distinction is the whole point: an ordinary facet rejection carries the facet's
// own cvc-* rule and means "this value is invalid", while these mean "this type could
// not have been valid for any value", which three callers must answer differently
// (ValidateLexical's contract). A test that only asserted "some error" would not
// notice the two being conflated.
func TestValueFacetCapabilityFaultsAreErrors(t *testing.T) {
	stringPrim := primType(t, "string", "preserve")
	lf, err := newLengthFacet(stringPrim, xsd.NewFacet(xsd.FacetLength, []string{"2"}, false))
	if err != nil {
		t.Fatalf("newLengthFacet: %v", err)
	}
	df, err := newDigitsFacet(xsd.NewFacet(xsd.FacetTotalDigits, []string{"2"}, false))
	if err != nil {
		t.Fatalf("newDigitsFacet: %v", err)
	}
	tf, err := newExplicitTimezoneFacet(xsd.NewFacet(xsd.FacetExplicitTimezone, []string{"required"}, false))
	if err != nil {
		t.Fatalf("newExplicitTimezoneFacet: %v", err)
	}
	sf, err := newScaleFacet(xsd.NewFacet(xsd.FacetMaxScale, []string{"2"}, false))
	if err != nil {
		t.Fatalf("newScaleFacet: %v", err)
	}

	cases := []struct {
		name    string
		facet   ValueFacet
		missing string
	}{
		{"bound facet on a candidate that is not Ordered", boundFacet{limit: intValue(1), kind: xsd.FacetMaxInclusive}, "Ordered"},
		{"length facet on a candidate that is not Lengthed", lf, "Lengthed"},
		{"digit facet on a candidate that is not DigitCounted", df, "DigitCounted"},
		{"explicitTimezone facet on a candidate that is not TimezoneAware", tf, "TimezoneAware"},
		{"scale facet on a candidate that is not Scaled", sf, "Scaled"},
	}
	// A bare Go string carries no capability at all, so every facet above faults on it.
	const bare = "no capabilities"
	for _, c := range cases {
		got := c.facet.CheckValue(bare)
		if got == nil {
			t.Errorf("%s: CheckValue = nil, want a %s precondition fault", c.name, c.missing)
			continue
		}
		if !IsFacetPrecondition(got) {
			t.Errorf("%s: IsFacetPrecondition = false, want true — a caller cannot tell this fault from a rejection: %v", c.name, got)
		}
		if r, _ := xsderr.RuleOf(got); r != "cos-applicable-facets" {
			t.Errorf("%s: CheckValue charged %s, want cos-applicable-facets (§4.1.5)", c.name, r)
		}
	}
}

// TestNewBoundFacetUnorderedLimitIsError pins the SIXTH capability site, the only one
// on the construction side: a bound facet whose own {value} parses to a value that is
// not Ordered faults at compile() rather than at check time. It is driven end to end
// through the exported ValidateLexical to prove the fault also PROPAGATES out of
// compile unchanged and stays discriminable there.
func TestNewBoundFacetUnorderedLimitIsError(t *testing.T) {
	qn := xsd.QName{Space: xsd.XMLSchemaNS, Local: "unorderedSpace"}
	st, err := xsd.NewPrimitiveType(xsderr.Loc{}, qn, []xsd.Facet{
		xsd.NewFacet(xsd.FacetWhiteSpace, []string{"collapse"}, true),
		xsd.NewFacet(xsd.FacetMaxInclusive, []string{"5"}, false),
	}, nil)
	if err != nil {
		t.Fatalf("NewPrimitiveType: %v", err)
	}
	_, verr := ValidateLexical(plainBackend{qn: true}, st, "1", nil)
	if verr == nil {
		t.Fatal("ValidateLexical(maxInclusive facet over an unordered value space) = nil error, want a precondition fault")
	}
	if !IsFacetPrecondition(verr) {
		t.Errorf("IsFacetPrecondition = false, want true: %v", verr)
	}
	if r, _ := xsderr.RuleOf(verr); r != "cos-applicable-facets" {
		t.Errorf("charged %s, want cos-applicable-facets (§4.1.5)", r)
	}
}

// TestFacetRejectionIsNotAPrecondition is the mutation guard for every
// discrimination site: an ORDINARY facet rejection — a real cvc-* validity verdict —
// must NOT satisfy IsFacetPrecondition. Were the predicate ever widened to "any
// facet-ish error", ValidDefault would report genuine invalid defaults undecided,
// dispatchUnion would abort on a member that merely rejected, and
// checkEnumerationRestriction would skip real §4.3.5.5 violations — three
// false-ACCEPT regressions that no other test in this package would catch.
func TestFacetRejectionIsNotAPrecondition(t *testing.T) {
	sf, err := newScaleFacet(xsd.NewFacet(xsd.FacetMaxScale, []string{"2"}, false))
	if err != nil {
		t.Fatalf("newScaleFacet: %v", err)
	}
	rejection := sf.CheckValue(scaledStub{scale: 3, present: true})
	if rejection == nil {
		t.Fatal("CheckValue(scale 3 under maxScale 2) = nil, want a cvc-maxScale-valid rejection")
	}
	if IsFacetPrecondition(rejection) {
		t.Errorf("IsFacetPrecondition(cvc-maxScale-valid rejection) = true, want false: %v", rejection)
	}
}

// TestUnsupportedFacetKindRejected confirms an out-of-range numeric FacetKind —
// one outside the closed 16-member enum, hence unsupported by the processor — is
// now rejected at schema construction by st-props-correct clause 5 (#46), so it
// can never reach compile()'s facet-kind switch. This defends the #158/#133
// bug class (a facet the processor cannot check must not be silently dropped)
// one layer earlier than compile()'s fail-loud default, which remains in place
// as defense-in-depth for a future in-range-but-unwired enum extension.
func TestUnsupportedFacetKindRejected(t *testing.T) {
	const unsupported = xsd.FacetKind(99) // outside the FacetKind enum
	_, err := xsd.NewPrimitiveType(xsderr.Loc{}, xsd.QName{Space: xsd.XMLSchemaNS, Local: "forged"},
		[]xsd.Facet{xsd.NewFacet(unsupported, []string{"x"}, false)}, nil)
	if err == nil {
		t.Fatal("NewPrimitiveType(unsupported facet): want rejection, got nil")
	}
	if r, _ := xsderr.RuleOf(err); r != "st-props-correct" {
		t.Errorf("NewPrimitiveType(unsupported facet) charged %s, want st-props-correct (clause 5)", r)
	}
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
