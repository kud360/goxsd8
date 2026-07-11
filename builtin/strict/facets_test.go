package strict

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// wantRule asserts err is an *xsderr.Error carrying exactly rule.
func wantRule(t *testing.T, err error, rule xsderr.Rule) {
	t.Helper()
	if err == nil {
		t.Fatalf("want rejection with rule %s, got nil", rule)
	}
	got, ok := xsderr.RuleOf(err)
	if !ok {
		t.Fatalf("want *xsderr.Error with rule %s, got %T: %v", rule, err, err)
	}
	if got != rule {
		t.Fatalf("want rule %s, got %s (%v)", rule, got, err)
	}
}

// wantAccept asserts err is nil (the literal was accepted).
func wantAccept(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("want accept, got rejection %v", err)
	}
}

// newPrim builds a cohort primitive by its spec local name (decimal/string),
// mirroring builtin.Seed's NewPrimitiveType path so whiteSpaceOf resolves it.
func newPrim(t *testing.T, local string) *xsd.SimpleType {
	t.Helper()
	p, err := xsd.NewPrimitiveType(xsderr.Loc{}, xsd.QName{Space: xsdNS, Local: local}, nil, nil)
	if err != nil {
		t.Fatalf("NewPrimitiveType(%q): %v", local, err)
	}
	return p
}

// derive builds an atomic restriction of base named {urn:test, name} carrying
// ownFacets, for the hand-built graphs these tests validate against.
func derive(t *testing.T, name string, base *xsd.SimpleType, ownFacets ...xsd.Facet) *xsd.SimpleType {
	t.Helper()
	st, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: name},
		xsd.Atomic{Primitive: primitiveOf(base)}, base, ownFacets, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(%q): %v", name, err)
	}
	return st
}

// TestWidestSpaceInheritedBound is the load-bearing test: an inherited
// maxInclusive facet must be compared in the value space of the type that
// DECLARES it (st-restrict-facets §3.16.6.4), resolved by walking from the
// declaring type to its nearest mapped ancestor — NOT the leaf's mapping and
// NOT merely "declaring == primitive".
//
// Chain: decimalPrim (mapped) ← lvl1 {urn:test} declares maxInclusive=100
// (NOT mapped by strict.New) ← leaf {urn:test} (inherits, NOT mapped). So
// EffectiveFacet.Declaring() is lvl1's QName, which strict.New does not map,
// forcing declaringMapping to walk lvl1 → decimalPrim to find the mapping.
func TestWidestSpaceInheritedBound(t *testing.T) {
	decimalPrim := newPrim(t, "decimal")
	lvl1 := derive(t, "level1", decimalPrim, xsd.NewFacet(xsd.FacetMaxInclusive, []string{"100"}, false))
	leaf := derive(t, "leaf", lvl1)

	// Premise: the effective maxInclusive on leaf is DECLARED by lvl1 (urn:test),
	// which is not in xsdNS and so is not directly mapped by strict.New().
	facets := leaf.EffectiveFacets()
	if len(facets) != 1 {
		t.Fatalf("leaf effective facets = %d, want 1 (the inherited maxInclusive)", len(facets))
	}
	declaring := facets[0].Declaring()
	if declaring != (xsd.QName{Space: "urn:test", Local: "level1"}) {
		t.Fatalf("declaring type = %s, want {urn:test}level1", declaring)
	}
	if _, mapped := New().Mapping(declaring); mapped {
		t.Fatalf("precondition broken: strict.New maps declaring type %s directly", declaring)
	}

	// "150" > 100: rejected via the declaring type's decimal space.
	_, err := checkAgainstType(New(), leaf, "150", nil)
	wantRule(t, err, "cvc-maxInclusive-valid")

	// "50" ≤ 100: accepted.
	_, err = checkAgainstType(New(), leaf, "50", nil)
	wantAccept(t, err)
}

// TestPatternFacet exercises the pattern (lexical) stage on a string restricted
// to [a-z]+ (cvc-pattern-valid). Note the XSD flavor anchors the whole literal
// and treats ^/$ as literal characters, so the pattern is "[a-z]+", not
// "^[a-z]+$".
func TestPatternFacet(t *testing.T) {
	stringPrim := newPrim(t, "string")
	lower := derive(t, "lower", stringPrim, xsd.NewFacet(xsd.FacetPattern, []string{"[a-z]+"}, false))

	_, err := checkAgainstType(New(), lower, "abc", nil)
	wantAccept(t, err)

	_, err = checkAgainstType(New(), lower, "ab3", nil)
	wantRule(t, err, "cvc-pattern-valid")
}

// TestEnumerationFacet exercises the enumeration value-facet stage on a string
// restricted to a small set (cvc-enumeration-valid).
func TestEnumerationFacet(t *testing.T) {
	stringPrim := newPrim(t, "string")
	colors := derive(t, "color", stringPrim, xsd.NewFacet(xsd.FacetEnumeration, []string{"red", "green", "blue"}, false))

	_, err := checkAgainstType(New(), colors, "green", nil)
	wantAccept(t, err)

	_, err = checkAgainstType(New(), colors, "purple", nil)
	wantRule(t, err, "cvc-enumeration-valid")
}

// TestDigitsFacets exercises totalDigits and fractionDigits on decimal as
// upper-bound constraints (cvc-totalDigits-valid, cvc-fractionDigits-valid).
func TestDigitsFacets(t *testing.T) {
	decimalPrim := newPrim(t, "decimal")

	total3 := derive(t, "total3", decimalPrim, xsd.NewFacet(xsd.FacetTotalDigits, []string{"3"}, false))
	if _, err := checkAgainstType(New(), total3, "123", nil); err != nil {
		t.Fatalf("totalDigits=3 should accept 123: %v", err)
	}
	_, err := checkAgainstType(New(), total3, "1234", nil)
	wantRule(t, err, "cvc-totalDigits-valid")

	frac2 := derive(t, "frac2", decimalPrim, xsd.NewFacet(xsd.FacetFractionDigits, []string{"2"}, false))
	if _, err := checkAgainstType(New(), frac2, "1.23", nil); err != nil {
		t.Fatalf("fractionDigits=2 should accept 1.23: %v", err)
	}
	_, err = checkAgainstType(New(), frac2, "1.234", nil)
	wantRule(t, err, "cvc-fractionDigits-valid")
}

// TestLengthFacets exercises length/minLength/maxLength on string in codepoint
// units (cvc-length-valid, cvc-minLength-valid, cvc-maxLength-valid).
func TestLengthFacets(t *testing.T) {
	stringPrim := newPrim(t, "string")

	len3 := derive(t, "len3", stringPrim, xsd.NewFacet(xsd.FacetLength, []string{"3"}, false))
	if _, err := checkAgainstType(New(), len3, "abc", nil); err != nil {
		t.Fatalf("length=3 should accept abc: %v", err)
	}
	_, err := checkAgainstType(New(), len3, "abcd", nil)
	wantRule(t, err, "cvc-length-valid")

	min3 := derive(t, "min3", stringPrim, xsd.NewFacet(xsd.FacetMinLength, []string{"3"}, false))
	_, err = checkAgainstType(New(), min3, "ab", nil)
	wantRule(t, err, "cvc-minLength-valid")

	max3 := derive(t, "max3", stringPrim, xsd.NewFacet(xsd.FacetMaxLength, []string{"3"}, false))
	_, err = checkAgainstType(New(), max3, "abcd", nil)
	wantRule(t, err, "cvc-maxLength-valid")
}
