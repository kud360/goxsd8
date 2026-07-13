package strict

import (
	"testing"

	"github.com/kud360/goxsd8/value"
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

// newPrim builds a cohort primitive by its spec local name (decimal/string/
// float), carrying the primitive's own whiteSpace facet (§3.16.7.4: string is
// preserve, every other atomic is collapse) exactly as builtin.Seed materializes
// it, so value.ValidateLexical's whiteSpace stage (effectiveWhiteSpace, reading
// EffectiveFacets) resolves it.
func newPrim(t *testing.T, local string) *xsd.SimpleType {
	t.Helper()
	ws := "collapse"
	if local == "string" {
		ws = "preserve"
	}
	p, err := xsd.NewPrimitiveType(xsderr.Loc{}, xsd.QName{Space: xsd.XMLSchemaNS, Local: local},
		[]xsd.Facet{xsd.NewFacet(xsd.FacetWhiteSpace, []string{ws}, ws != "preserve")}, nil)
	if err != nil {
		t.Fatalf("NewPrimitiveType(%q): %v", local, err)
	}
	return p
}

// derive builds an atomic restriction of base named {urn:test, name} carrying
// ownFacets, for the hand-built graphs these tests validate against. Its
// {primitive type definition} is base's primitive ancestor, found by walking
// Base() to IsPrimitive (§2.4.2).
func derive(t *testing.T, name string, base *xsd.SimpleType, ownFacets ...xsd.Facet) *xsd.SimpleType {
	t.Helper()
	var prim *xsd.SimpleType
	for s := base; s != nil; s = s.Base() {
		if s.IsPrimitive() {
			prim = s
			break
		}
	}
	st, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: name},
		xsd.Atomic{Primitive: prim}, base, ownFacets, nil)
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
	// which is not in xsd.XMLSchemaNS and so is not directly mapped by strict.New().
	// leaf's effective facets are the inherited whiteSpace (from the decimal
	// primitive, §3.16.7.4) and this maxInclusive; isolate the latter.
	var maxIncl xsd.EffectiveFacet
	var found bool
	for _, ef := range leaf.EffectiveFacets() {
		if ef.Facet().Kind() == xsd.FacetMaxInclusive {
			maxIncl = ef
			found = true
		}
	}
	if !found {
		t.Fatalf("leaf has no effective maxInclusive facet")
	}
	declaring := maxIncl.Declaring()
	if declaring != (xsd.QName{Space: "urn:test", Local: "level1"}) {
		t.Fatalf("declaring type = %s, want {urn:test}level1", declaring)
	}
	if _, mapped := New().Mapping(declaring); mapped {
		t.Fatalf("precondition broken: strict.New maps declaring type %s directly", declaring)
	}

	// "150" > 100: rejected via the declaring type's decimal space.
	_, err := value.ValidateLexical(New(), leaf, "150", nil)
	wantRule(t, err, "cvc-maxInclusive-valid")

	// "50" ≤ 100: accepted.
	_, err = value.ValidateLexical(New(), leaf, "50", nil)
	wantAccept(t, err)
}

// TestBoundFacetNaNExcluded is the load-bearing partial-order test (#60): a
// bounding facet applied to a value that is INCOMPARABLE with the bound must
// EXCLUDE that value from the restricted value space (§3.3.4.3/§3.3.5.3 Note),
// routing through a real cvc-*-valid rejection — NOT a panic. float is the first
// cohort primitive whose order is partial, so this path is exercisable at last.
//
// Two directions: (1) a NaN instance against a numeric maxInclusive bound is
// incomparable, so excluded; (2) a NaN bound value makes the restricted space
// empty, so even an ordinary instance is excluded.
func TestBoundFacetNaNExcluded(t *testing.T) {
	floatPrim := newPrim(t, "float")

	// (1) NaN instance, numeric bound: NaN is incomparable with 10, so excluded.
	maxTen := derive(t, "maxTen", floatPrim, xsd.NewFacet(xsd.FacetMaxInclusive, []string{"10"}, false))
	_, err := value.ValidateLexical(New(), maxTen, "NaN", nil)
	wantRule(t, err, "cvc-maxInclusive-valid")
	// A comparable in-range instance still passes the same facet.
	_, err = value.ValidateLexical(New(), maxTen, "5", nil)
	wantAccept(t, err)

	// (2) NaN bound value: no float is comparable with NaN, so the restricted
	// value space is empty and every instance — even 5 — is excluded.
	maxNaN := derive(t, "maxNaN", floatPrim, xsd.NewFacet(xsd.FacetMinInclusive, []string{"NaN"}, false))
	_, err = value.ValidateLexical(New(), maxNaN, "5", nil)
	wantRule(t, err, "cvc-minInclusive-valid")
}

// TestPatternFacet exercises the pattern (lexical) stage on a string restricted
// to [a-z]+ (cvc-pattern-valid). Note the XSD flavor anchors the whole literal
// and treats ^/$ as literal characters, so the pattern is "[a-z]+", not
// "^[a-z]+$".
func TestPatternFacet(t *testing.T) {
	stringPrim := newPrim(t, "string")
	lower := derive(t, "lower", stringPrim, xsd.NewFacet(xsd.FacetPattern, []string{"[a-z]+"}, false))

	_, err := value.ValidateLexical(New(), lower, "abc", nil)
	wantAccept(t, err)

	_, err = value.ValidateLexical(New(), lower, "ab3", nil)
	wantRule(t, err, "cvc-pattern-valid")
}

// TestPatternFacetTwoStepAND is the end-to-end guard for #94: a pattern facet
// declared at TWO derivation steps must be ANDed, not superseded (§4.3.4.2
// xr-pattern; §4.3.4.4 cvc-pattern-valid). base restricts string to [a-z]+;
// derived restricts base to ".{3}" (exactly three chars). A literal matching
// the derived pattern but VIOLATING the base's ("a1z" — three chars, but the 1
// is not [a-z]) must be REJECTED. Before the overlayFacet keep-both fix the base
// pattern was silently dropped and this literal was a false-accept.
func TestPatternFacetTwoStepAND(t *testing.T) {
	stringPrim := newPrim(t, "string")
	base := derive(t, "lowerBase", stringPrim, xsd.NewFacet(xsd.FacetPattern, []string{"[a-z]+"}, false))
	derived := derive(t, "threeChars", base, xsd.NewFacet(xsd.FacetPattern, []string{".{3}"}, false))

	// Matches both patterns: three chars, all lowercase.
	_, err := value.ValidateLexical(New(), derived, "abc", nil)
	wantAccept(t, err)

	// Matches derived (three chars) but violates base ([a-z]+): rejected.
	_, err = value.ValidateLexical(New(), derived, "a1z", nil)
	wantRule(t, err, "cvc-pattern-valid")

	// Matches base ([a-z]+) but violates derived (not three chars): rejected.
	_, err = value.ValidateLexical(New(), derived, "abcd", nil)
	wantRule(t, err, "cvc-pattern-valid")
}

// TestEnumerationFacet exercises the enumeration value-facet stage on a string
// restricted to a small set (cvc-enumeration-valid).
func TestEnumerationFacet(t *testing.T) {
	stringPrim := newPrim(t, "string")
	colors := derive(t, "color", stringPrim, xsd.NewFacet(xsd.FacetEnumeration, []string{"red", "green", "blue"}, false))

	_, err := value.ValidateLexical(New(), colors, "green", nil)
	wantAccept(t, err)

	_, err = value.ValidateLexical(New(), colors, "purple", nil)
	wantRule(t, err, "cvc-enumeration-valid")
}

// TestDigitsFacets exercises totalDigits and fractionDigits on decimal as
// upper-bound constraints (cvc-totalDigits-valid, cvc-fractionDigits-valid).
func TestDigitsFacets(t *testing.T) {
	decimalPrim := newPrim(t, "decimal")

	total3 := derive(t, "total3", decimalPrim, xsd.NewFacet(xsd.FacetTotalDigits, []string{"3"}, false))
	if _, err := value.ValidateLexical(New(), total3, "123", nil); err != nil {
		t.Fatalf("totalDigits=3 should accept 123: %v", err)
	}
	_, err := value.ValidateLexical(New(), total3, "1234", nil)
	wantRule(t, err, "cvc-totalDigits-valid")

	frac2 := derive(t, "frac2", decimalPrim, xsd.NewFacet(xsd.FacetFractionDigits, []string{"2"}, false))
	if _, err := value.ValidateLexical(New(), frac2, "1.23", nil); err != nil {
		t.Fatalf("fractionDigits=2 should accept 1.23: %v", err)
	}
	_, err = value.ValidateLexical(New(), frac2, "1.234", nil)
	wantRule(t, err, "cvc-fractionDigits-valid")
}

// TestLengthFacets exercises length/minLength/maxLength on string in codepoint
// units (cvc-length-valid, cvc-minLength-valid, cvc-maxLength-valid).
func TestLengthFacets(t *testing.T) {
	stringPrim := newPrim(t, "string")

	len3 := derive(t, "len3", stringPrim, xsd.NewFacet(xsd.FacetLength, []string{"3"}, false))
	if _, err := value.ValidateLexical(New(), len3, "abc", nil); err != nil {
		t.Fatalf("length=3 should accept abc: %v", err)
	}
	_, err := value.ValidateLexical(New(), len3, "abcd", nil)
	wantRule(t, err, "cvc-length-valid")

	min3 := derive(t, "min3", stringPrim, xsd.NewFacet(xsd.FacetMinLength, []string{"3"}, false))
	_, err = value.ValidateLexical(New(), min3, "ab", nil)
	wantRule(t, err, "cvc-minLength-valid")

	max3 := derive(t, "max3", stringPrim, xsd.NewFacet(xsd.FacetMaxLength, []string{"3"}, false))
	_, err = value.ValidateLexical(New(), max3, "abcd", nil)
	wantRule(t, err, "cvc-maxLength-valid")
}
