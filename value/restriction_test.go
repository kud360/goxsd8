package value

import (
	"strconv"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// intValue is a test-only totally ordered value: enough to exercise the bound
// cross-checks without pulling in a real backend (package value's own tests
// cannot import builtin/strict, which imports this package).
type intValue int

func (i intValue) Eq(other Value) bool {
	o, ok := other.(intValue)
	return ok && o == i
}

func (i intValue) Cmp(other Value) Ordering {
	o, ok := other.(intValue)
	if !ok {
		return Incomparable
	}
	switch {
	case i < o:
		return Less
	case i > o:
		return Greater
	default:
		return Equal
	}
}

// intBackend maps one named type to a decimal-integer lexical mapping. A
// lexical that is not an integer is rejected as an *xsderr.Error, exactly as a
// real backend's Parse would report a value outside the type's space.
type intBackend struct{ mapped xsd.QName }

func (b intBackend) Mapping(typ xsd.QName) (Mapping, bool) {
	if typ != b.mapped {
		return Mapping{}, false
	}
	return Mapping{Parse: func(lexical string, _ Context) (Value, error) {
		n, err := strconv.Atoi(lexical)
		if err != nil {
			return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
				"test backend: %q is not an integer", lexical)
		}
		return intValue(n), nil
	}}, true
}

// restrictionBase builds a primitive named "int" carrying a collapse whiteSpace
// facet plus ownFacets, and returns it with a backend that maps it.
func restrictionBase(t *testing.T, ownFacets ...xsd.Facet) (*xsd.SimpleType, Backend) {
	t.Helper()
	facets := append([]xsd.Facet{xsd.NewFacet(xsd.FacetWhiteSpace, []string{"collapse"}, true)}, ownFacets...)
	qn := xsd.QName{Space: xsd.XMLSchemaNS, Local: "int"}
	base, err := xsd.NewPrimitiveType(xsderr.Loc{}, qn, facets, nil)
	if err != nil {
		t.Fatalf("build base: %v", err)
	}
	return base, intBackend{mapped: qn}
}

// restrict builds a restriction of base carrying ownFacets and runs
// CheckFacetRestriction over it.
func restrict(t *testing.T, b Backend, base *xsd.SimpleType, ownFacets ...xsd.Facet) error {
	t.Helper()
	st, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: "derived"},
		base.Variety(), base, ownFacets, nil)
	if err != nil {
		t.Fatalf("NewSimpleType: %v", err)
	}
	return CheckFacetRestriction(b, st)
}

func bound(kind xsd.FacetKind, v string) xsd.Facet {
	return xsd.NewFacet(kind, []string{v}, false)
}

// TestBoundRestrictionFourWayCrossCheck is the heart of §4.3.7.4–§4.3.10.4: a
// derived bound is checked against EVERY bound facet the base carries, not just
// the homonymous one. Each row names the base facet kind that must catch the
// derived one, so a "same-kind only" implementation fails every cross-kind row.
func TestBoundRestrictionFourWayCrossCheck(t *testing.T) {
	cases := []struct {
		name       string
		baseKind   xsd.FacetKind
		baseVal    string
		derivedKnd xsd.FacetKind
		derivedVal string
		rejected   bool
	}{
		// minInclusive-valid-restriction (§4.3.10.4), all four clauses.
		{"minInc under base minInc", xsd.FacetMinInclusive, "10", xsd.FacetMinInclusive, "5", true},
		{"minInc at base minInc", xsd.FacetMinInclusive, "10", xsd.FacetMinInclusive, "10", false},
		{"minInc over base maxInc", xsd.FacetMaxInclusive, "10", xsd.FacetMinInclusive, "11", true},
		{"minInc at base maxInc", xsd.FacetMaxInclusive, "10", xsd.FacetMinInclusive, "10", false},
		{"minInc at base minExc", xsd.FacetMinExclusive, "10", xsd.FacetMinInclusive, "10", true},
		{"minInc over base minExc", xsd.FacetMinExclusive, "10", xsd.FacetMinInclusive, "11", false},
		{"minInc at base maxExc", xsd.FacetMaxExclusive, "10", xsd.FacetMinInclusive, "10", true},

		// maxInclusive-valid-restriction (§4.3.7.4), all four clauses.
		{"maxInc over base maxInc", xsd.FacetMaxInclusive, "10", xsd.FacetMaxInclusive, "11", true},
		{"maxInc at base maxExc", xsd.FacetMaxExclusive, "10", xsd.FacetMaxInclusive, "10", true},
		{"maxInc under base maxExc", xsd.FacetMaxExclusive, "10", xsd.FacetMaxInclusive, "9", false},
		{"maxInc under base minInc", xsd.FacetMinInclusive, "10", xsd.FacetMaxInclusive, "9", true},
		{"maxInc at base minExc", xsd.FacetMinExclusive, "10", xsd.FacetMaxInclusive, "10", true},

		// maxExclusive-valid-restriction (§4.3.8.4), all four clauses.
		{"maxExc over base maxExc", xsd.FacetMaxExclusive, "10", xsd.FacetMaxExclusive, "11", true},
		{"maxExc over base maxInc", xsd.FacetMaxInclusive, "10", xsd.FacetMaxExclusive, "11", true},
		{"maxExc at base maxInc", xsd.FacetMaxInclusive, "10", xsd.FacetMaxExclusive, "10", false},
		{"maxExc at base minInc", xsd.FacetMinInclusive, "10", xsd.FacetMaxExclusive, "10", true},
		{"maxExc at base minExc", xsd.FacetMinExclusive, "10", xsd.FacetMaxExclusive, "10", true},

		// minExclusive-valid-restriction (§4.3.9.4), all four clauses.
		{"minExc under base minExc", xsd.FacetMinExclusive, "10", xsd.FacetMinExclusive, "9", true},
		{"minExc under base minInc", xsd.FacetMinInclusive, "10", xsd.FacetMinExclusive, "9", true},
		{"minExc at base minInc", xsd.FacetMinInclusive, "10", xsd.FacetMinExclusive, "10", false},
		{"minExc at base maxInc", xsd.FacetMaxInclusive, "10", xsd.FacetMinExclusive, "10", true},
		{"minExc at base maxExc", xsd.FacetMaxExclusive, "10", xsd.FacetMinExclusive, "10", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			base, b := restrictionBase(t, bound(c.baseKind, c.baseVal))
			err := restrict(t, b, base, bound(c.derivedKnd, c.derivedVal))
			if !c.rejected {
				if err != nil {
					t.Fatalf("accepted restriction rejected: %v", err)
				}
				return
			}
			rule, ok := xsderr.RuleOf(err)
			if !ok {
				t.Fatalf("want a rejection, got %v", err)
			}
			want := boundRestrictionRule(c.derivedKnd)
			if rule != want {
				t.Fatalf("rejection charges %q, want %q (the DERIVED facet's rule); err=%v", rule, want, err)
			}
		})
	}
}

// TestBoundRestrictionFacetValueNormalized proves a facet's lexical {value} is
// whiteSpace-normalized through the BASE type's mode before it is parsed: the
// XML mapping interprets the value attribute through the base's lexical mapping,
// whose first stage is that normalization.
func TestBoundRestrictionFacetValueNormalized(t *testing.T) {
	base, b := restrictionBase(t, bound(xsd.FacetMaxInclusive, "10"))
	if err := restrict(t, b, base, bound(xsd.FacetMaxInclusive, " 9 ")); err != nil {
		t.Fatalf("padded facet value rejected: %v", err)
	}
}

// TestBoundRestrictionUnparsableValue rejects a bound {value} outside the base
// type's value space under that facet's own valid-restriction rule: every
// numbered condition presupposes the {value} is a member of that space.
func TestBoundRestrictionUnparsableValue(t *testing.T) {
	base, b := restrictionBase(t, bound(xsd.FacetMaxInclusive, "10"))
	err := restrict(t, b, base, bound(xsd.FacetMaxInclusive, "not-a-number"))
	rule, ok := xsderr.RuleOf(err)
	if !ok || rule != "maxInclusive-valid-restriction" {
		t.Fatalf("rule = %q (ok=%v), want maxInclusive-valid-restriction; err=%v", rule, ok, err)
	}
}

// TestEnumerationValidRestriction covers §4.3.5.5: each derived enumeration
// member must be in the value space of the {base type definition}.
func TestEnumerationValidRestriction(t *testing.T) {
	base, b := restrictionBase(t)
	good := xsd.NewEnumerationFacet([]xsd.EnumerationMember{
		xsd.NewEnumerationMember("1", nil, nil),
		xsd.NewEnumerationMember("2", nil, nil),
	})
	if err := restrict(t, b, base, good); err != nil {
		t.Fatalf("in-space enumeration rejected: %v", err)
	}

	bad := xsd.NewEnumerationFacet([]xsd.EnumerationMember{
		xsd.NewEnumerationMember("1", nil, nil),
		xsd.NewEnumerationMember("banana", nil, nil),
	})
	err := restrict(t, b, base, bad)
	rule, ok := xsderr.RuleOf(err)
	if !ok || rule != "enumeration-valid-restriction" {
		t.Fatalf("rule = %q (ok=%v), want enumeration-valid-restriction; err=%v", rule, ok, err)
	}
	if !strings.Contains(err.Error(), "banana") {
		t.Errorf("message %q does not name the offending member", err.Error())
	}
}

// TestCheckFacetRestrictionNoBackendMappingFailsOpen pins the fail-open
// decision: a backend that does not map the base type is a BACKEND gap, not a
// schema invalidity, so the value-space comparisons are skipped rather than
// turned into a rejection — even for a restriction that would otherwise be
// caught.
func TestCheckFacetRestrictionNoBackendMappingFailsOpen(t *testing.T) {
	base, _ := restrictionBase(t, bound(xsd.FacetMaxInclusive, "10"))
	unmapped := intBackend{mapped: xsd.QName{Space: xsd.XMLSchemaNS, Local: "somethingElse"}}
	if err := restrict(t, unmapped, base, bound(xsd.FacetMaxInclusive, "999")); err != nil {
		t.Fatalf("unmapped base must fail open, got: %v", err)
	}
}

// TestCheckFacetRestrictionAnySimpleType proves the root of the hierarchy — the
// one simple type with an absent {base type definition} — is a no-op rather than
// a nil dereference.
func TestCheckFacetRestrictionAnySimpleType(t *testing.T) {
	b := intBackend{mapped: xsd.QName{Space: xsd.XMLSchemaNS, Local: "int"}}
	if err := CheckFacetRestriction(b, xsd.AnySimpleType()); err != nil {
		t.Fatalf("CheckFacetRestriction(anySimpleType) = %v, want nil", err)
	}
}

// TestBoundRestrictionViolatesIncomparable pins the Incomparable reading: every
// numbered condition is an order test, so a pair no order relates satisfies
// none of them and is not a violation.
func TestBoundRestrictionViolatesIncomparable(t *testing.T) {
	kinds := []xsd.FacetKind{
		xsd.FacetMaxInclusive, xsd.FacetMaxExclusive, xsd.FacetMinInclusive, xsd.FacetMinExclusive,
	}
	for _, derived := range kinds {
		for _, base := range kinds {
			if boundRestrictionViolates(derived, base, Incomparable) {
				t.Errorf("boundRestrictionViolates(%s, %s, Incomparable) = true, want false", derived, base)
			}
		}
	}
}

// TestWhiteSpaceInForceNoUsableMode covers the non-panicking mode resolution
// facet-{value} parsing needs, including the three states effectiveWhiteSpace
// turns into a panic: a nil type, no whiteSpace facet in force at all
// (xs:anyAtomicType), and a {value} outside the §4.3.6.1 domain. Each answers the
// zero mode, which facetValue reads as "parse the lexical unchanged".
func TestWhiteSpaceInForceNoUsableMode(t *testing.T) {
	if got := whiteSpaceInForce(nil); got != 0 {
		t.Errorf("whiteSpaceInForce(nil) = %d, want zero mode 0", got)
	}
	if got := whiteSpaceInForce(xsd.AnyAtomicType()); got != 0 {
		t.Errorf("whiteSpaceInForce(anyAtomicType) = %d, want zero mode 0", got)
	}

	bogus, err := xsd.NewPrimitiveType(xsderr.Loc{}, xsd.QName{Space: xsd.XMLSchemaNS, Local: "bogus"},
		[]xsd.Facet{xsd.NewFacet(xsd.FacetWhiteSpace, []string{"squash"}, false)}, nil)
	if err != nil {
		t.Fatalf("build bogus primitive: %v", err)
	}
	if got := whiteSpaceInForce(bogus); got != 0 {
		t.Errorf("whiteSpaceInForce(out-of-domain {value}) = %d, want zero mode 0", got)
	}

	ok, err := xsd.NewPrimitiveType(xsderr.Loc{}, xsd.QName{Space: xsd.XMLSchemaNS, Local: "fine"},
		[]xsd.Facet{xsd.NewFacet(xsd.FacetWhiteSpace, []string{"collapse"}, true)}, nil)
	if err != nil {
		t.Fatalf("build fine primitive: %v", err)
	}
	if got := whiteSpaceInForce(ok); got != collapseWS {
		t.Errorf("whiteSpaceInForce = %d, want collapse %d", got, collapseWS)
	}
}

// enumOf builds an enumeration facet from bare lexical members (no namespace
// context — the context-free cohort).
func enumOf(lexicals ...string) xsd.Facet {
	members := make([]xsd.EnumerationMember, 0, len(lexicals))
	for _, l := range lexicals {
		members = append(members, xsd.NewEnumerationMember(l, nil, nil))
	}
	return xsd.NewEnumerationFacet(members)
}

// wsPrimitive builds a primitive named "int" whose whiteSpace facet carries mode,
// paired with a backend that maps it to the integer lexical mapping.
func wsPrimitive(t *testing.T, mode string) (*xsd.SimpleType, Backend) {
	t.Helper()
	qn := xsd.QName{Space: xsd.XMLSchemaNS, Local: "int"}
	base, err := xsd.NewPrimitiveType(xsderr.Loc{}, qn,
		[]xsd.Facet{xsd.NewFacet(xsd.FacetWhiteSpace, []string{mode}, true)}, nil)
	if err != nil {
		t.Fatalf("build %s primitive: %v", mode, err)
	}
	return base, intBackend{mapped: qn}
}

// TestFacetValueNormalizedAtConstruction pins the seam Findings 1+2 moved: a
// facet's raw {value} attribute is whiteSpace-normalized through the BASE type's
// mode before Mapping.Parse sees it, at CONSTRUCTION time — inside
// newBoundFacet/newEnumFacet, which drive instance validation — not only inside
// the restriction check. A facet's {value} property is "a value from the value
// space of the {base type definition}" (§4.3.7.1 f-mai-value, §4.3.5.1), and
// reaching that value space runs the base's lexical mapping, whose first stage is
// its whiteSpace normalization (key-vv §3.1.3, key-nv §3.1.4, cvc-simple-type
// §3.16.4).
//
// Without the normalization the test backend's Parse rejects " 9 " / "\t7\n"
// outright, so compile() fails and ValidateLexical returns an error instead of a
// value — the shape this test would report before the fix.
func TestFacetValueNormalizedAtConstruction(t *testing.T) {
	base, b := wsPrimitive(t, "collapse")
	st, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: "derived"},
		base.Variety(), base,
		[]xsd.Facet{bound(xsd.FacetMaxInclusive, " 9 "), enumOf("\t7\n", "  8  ")}, nil)
	if err != nil {
		t.Fatalf("NewSimpleType: %v", err)
	}

	v, err := ValidateLexical(b, st, " 7 ", nil)
	if err != nil {
		t.Fatalf("ValidateLexical(\" 7 \") = %v, want the enumeration member 7 to match", err)
	}
	if v != intValue(7) {
		t.Errorf("ValidateLexical(\" 7 \") = %v, want intValue(7)", v)
	}

	// The bound really is 9, not an unparsed string: 8 is enumerated and under
	// the bound, so only the enumeration can reject it, and it does not.
	if _, err := ValidateLexical(b, st, "8", nil); err != nil {
		t.Errorf("ValidateLexical(\"8\") = %v, want valid under maxInclusive \" 9 \"", err)
	}

	// And CheckFacetRestriction agrees with the constructed facets rather than
	// forking from them: " 9 " is a legal restriction of the base's maxInclusive 9.
	restricted, b2 := restrictionBase(t, bound(xsd.FacetMaxInclusive, "9"))
	if err := restrict(t, b2, restricted, bound(xsd.FacetMaxInclusive, " 9 ")); err != nil {
		t.Errorf("CheckFacetRestriction(maxInclusive \" 9 \" under base 9) = %v, want nil", err)
	}
}

// TestFacetValueNotNormalizedUnderPreserve is the negative half: the mode comes
// from the base's whiteSpace facet, never a blanket collapse. Under preserve
// (§4.3.6) the padding survives into Mapping.Parse, which rejects it — so a
// padded facet {value} on a preserve-normalized base is a construction error, and
// the normalization is genuinely mode-driven.
func TestFacetValueNotNormalizedUnderPreserve(t *testing.T) {
	base, b := wsPrimitive(t, "preserve")
	st, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: "derived"},
		base.Variety(), base, []xsd.Facet{bound(xsd.FacetMaxInclusive, " 9 ")}, nil)
	if err != nil {
		t.Fatalf("NewSimpleType: %v", err)
	}
	if _, err := ValidateLexical(b, st, "7", nil); err == nil {
		t.Error("ValidateLexical under a preserve base with maxInclusive \" 9 \" = nil error, want the padded facet {value} to be rejected")
	}
}

// TestBoundRestrictionBaseSideRuleAttribution pins Finding 5: when it is the
// BASE's bound facet whose {value} is outside the value space, the rejection is
// charged under THAT facet's own valid-restriction rule, not under the derived
// facet's. The base carries a malformed minExclusive while the derived facet is a
// maxInclusive, so a "reuse the derived rule" implementation reports
// maxInclusive-valid-restriction and fails here (STYLE E2).
func TestBoundRestrictionBaseSideRuleAttribution(t *testing.T) {
	base, b := restrictionBase(t, bound(xsd.FacetMinExclusive, "not-a-number"))
	err := restrict(t, b, base, bound(xsd.FacetMaxInclusive, "5"))
	rule, ok := xsderr.RuleOf(err)
	if !ok || rule != "minExclusive-valid-restriction" {
		t.Fatalf("rule = %q (ok=%v), want minExclusive-valid-restriction (the BASE operand's own rule); err=%v", rule, ok, err)
	}
}
