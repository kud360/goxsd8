package xsd

import (
	"reflect"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// TestNewFacetFixedNormalization checks that NewFacet honors {fixed} only for
// kinds that have the property (FacetKind.HasFixed) and normalizes it away
// otherwise, so a "fixed set on a kind with no {fixed}" state is unstorable.
func TestNewFacetFixedNormalization(t *testing.T) {
	cases := []struct {
		kind      FacetKind
		fixedIn   bool
		wantFixed bool
		wantOK    bool
	}{
		{FacetLength, true, true, true},
		{FacetWhiteSpace, false, false, true},
		{FacetExplicitTimezone, true, true, true},
		{FacetPattern, true, false, false},     // normalized: pattern has no {fixed}
		{FacetEnumeration, true, false, false}, // normalized: enumeration has no {fixed}
		// FacetAssertions (also fixed-less) is excluded: NewFacet panics for it;
		// see TestNewFacetAssertionsPanics and NewAssertionsFacet.
	}
	for _, c := range cases {
		f := NewFacet(c.kind, []string{"x"}, c.fixedIn)
		gotFixed, gotOK := f.Fixed()
		if gotOK != c.wantOK {
			t.Errorf("%s: Fixed() ok = %v, want %v", c.kind, gotOK, c.wantOK)
		}
		if gotFixed != c.wantFixed {
			t.Errorf("%s: Fixed() fixed = %v, want %v", c.kind, gotFixed, c.wantFixed)
		}
	}
}

// TestNewFacetValuesCopied verifies NewFacet copies the input values and
// Values returns a copy, so no caller aliases the facet's backing array.
func TestNewFacetValuesCopied(t *testing.T) {
	in := []string{"a", "b"}
	f := NewFacet(FacetEnumeration, in, false)
	in[0] = "mutated"
	got := f.Values()
	if got[0] != "a" {
		t.Fatalf("NewFacet aliased caller slice: Values()[0] = %q, want %q", got[0], "a")
	}
	got[1] = "clobber"
	if again := f.Values(); again[1] != "b" {
		t.Fatalf("Values() aliased internal slice: got %q, want %q", again[1], "b")
	}
}

// TestNewFacetAssertionsPanics confirms NewFacet rejects FacetAssertions: the
// assertions facet models {value} as Assertion components (§4.3.13), so it must
// go through NewAssertionsFacet, and using the wrong constructor is a caught
// programmer error, not a silent mis-build.
func TestNewFacetAssertionsPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("NewFacet(FacetAssertions, ...): want panic, got none")
		}
	}()
	_ = NewFacet(FacetAssertions, []string{"true()"}, false)
}

// TestNewAssertionsFacetRoundTrip verifies NewAssertionsFacet builds an
// assertions-kind Facet whose Assertions round-trips in document order, with no
// {fixed} property, and with defensive-copy semantics on both the input slice
// and the returned slice.
func TestNewAssertionsFacetRoundTrip(t *testing.T) {
	a0 := NewAssertion(NewXPathExpression("@a > 0", nil, nil, nil), nil)
	a1 := NewAssertion(NewXPathExpression("@b < 10", nil, nil, nil), nil)
	in := []Assertion{a0, a1}
	f := NewAssertionsFacet(in)

	if f.Kind() != FacetAssertions {
		t.Fatalf("Kind() = %s, want assertions", f.Kind())
	}
	if _, ok := f.Fixed(); ok {
		t.Error("Fixed() ok = true, want false (assertions has no {fixed})")
	}

	got, ok := f.Assertions()
	if !ok {
		t.Fatal("Assertions() ok = false, want true for an assertions facet")
	}
	if len(got) != 2 {
		t.Fatalf("Assertions() len = %d, want 2", len(got))
	}
	if got[0].Test().Expression() != "@a > 0" || got[1].Test().Expression() != "@b < 10" {
		t.Errorf("Assertions() document order wrong: got %q, %q",
			got[0].Test().Expression(), got[1].Test().Expression())
	}

	// Mutating the caller's input slice must not affect the facet.
	in[0] = a1
	if again, _ := f.Assertions(); again[0].Test().Expression() != "@a > 0" {
		t.Errorf("NewAssertionsFacet aliased caller slice: got %q, want %q",
			again[0].Test().Expression(), "@a > 0")
	}

	// Mutating the returned slice must not affect the facet.
	got[0] = a1
	if again, _ := f.Assertions(); again[0].Test().Expression() != "@a > 0" {
		t.Errorf("Assertions() aliased internal slice: got %q, want %q",
			again[0].Test().Expression(), "@a > 0")
	}
}

// TestAssertionsOnNonAssertionsFacet verifies Assertions reports ok == false and
// nil for a facet whose kind is not FacetAssertions.
func TestAssertionsOnNonAssertionsFacet(t *testing.T) {
	f := NewFacet(FacetLength, []string{"3"}, false)
	got, ok := f.Assertions()
	if ok {
		t.Error("Assertions() ok = true for a length facet, want false")
	}
	if got != nil {
		t.Errorf("Assertions() = %v for a length facet, want nil", got)
	}
}

// TestFacetKindString spot-checks the verbatim §4.3 tokens and the diagnostic
// fallback for an invalid value.
func TestFacetKindString(t *testing.T) {
	cases := map[FacetKind]string{
		FacetLength:           "length",
		FacetMinLength:        "minLength",
		FacetMaxLength:        "maxLength",
		FacetPattern:          "pattern",
		FacetEnumeration:      "enumeration",
		FacetWhiteSpace:       "whiteSpace",
		FacetMaxInclusive:     "maxInclusive",
		FacetMaxExclusive:     "maxExclusive",
		FacetMinExclusive:     "minExclusive",
		FacetMinInclusive:     "minInclusive",
		FacetTotalDigits:      "totalDigits",
		FacetFractionDigits:   "fractionDigits",
		FacetAssertions:       "assertions",
		FacetExplicitTimezone: "explicitTimezone",
	}
	for k, want := range cases {
		if got := k.String(); got != want {
			t.Errorf("FacetKind(%d).String() = %q, want %q", k, got, want)
		}
	}
	if got := FacetKind(0).String(); got != "FacetKind(0)" {
		t.Errorf("zero String() = %q, want %q", got, "FacetKind(0)")
	}
	if got := FacetKind(99).String(); got != "FacetKind(99)" {
		t.Errorf("invalid String() = %q, want %q", got, "FacetKind(99)")
	}
}

// TestFacetKindHasFixed pins the three fixed-less kinds against the rest.
func TestFacetKindHasFixed(t *testing.T) {
	noFixed := map[FacetKind]bool{FacetPattern: true, FacetEnumeration: true, FacetAssertions: true}
	all := []FacetKind{
		FacetLength, FacetMinLength, FacetMaxLength, FacetPattern, FacetEnumeration,
		FacetWhiteSpace, FacetMaxInclusive, FacetMaxExclusive, FacetMinExclusive,
		FacetMinInclusive, FacetTotalDigits, FacetFractionDigits, FacetAssertions,
		FacetExplicitTimezone,
	}
	for _, k := range all {
		want := !noFixed[k]
		if got := k.HasFixed(); got != want {
			t.Errorf("%s.HasFixed() = %v, want %v", k, got, want)
		}
	}
}

// TestNewSimpleTypeRejectsSubstitutionFinal checks that a {final} entry outside
// the legal simple-type subset (here DerivationSubstitution) is rejected with
// st-props-correct.
func TestNewSimpleTypeRejectsSubstitutionFinal(t *testing.T) {
	_, err := NewSimpleType(xsderr.Loc{}, QName{}, Atomic{Primitive: anyAtomicType}, anyAtomicType, nil,
		[]DerivationMethod{DerivationRestriction, DerivationSubstitution})
	if err == nil {
		t.Fatal("NewSimpleType accepted DerivationSubstitution in {final}, want rejection")
	}
	if r, _ := xsderr.RuleOf(err); r != ruleSTPropsCorrect {
		t.Fatalf("rule = %q, want %q", r, ruleSTPropsCorrect)
	}
}

// TestNewSimpleTypeAcceptsLegalFinal confirms the four legal simple-type
// {final} tokens are all accepted and returned in document order as a copy.
func TestNewSimpleTypeAcceptsLegalFinal(t *testing.T) {
	final := []DerivationMethod{DerivationRestriction, DerivationExtension, DerivationList, DerivationUnion}
	st, err := NewSimpleType(xsderr.Loc{}, QName{}, Atomic{Primitive: anyAtomicType}, anyAtomicType, nil, final)
	if err != nil {
		t.Fatalf("NewSimpleType rejected legal {final}: %v", err)
	}
	got := st.Final()
	if len(got) != len(final) {
		t.Fatalf("Final() len = %d, want %d", len(got), len(final))
	}
	for i := range final {
		if got[i] != final[i] {
			t.Errorf("Final()[%d] = %s, want %s", i, got[i], final[i])
		}
	}
	got[0] = DerivationUnion // mutating the copy must not affect st
	if st.Final()[0] != DerivationRestriction {
		t.Error("Final() returned an aliased slice")
	}
}

// TestNewSimpleTypeRejectsDuplicateFacetKind checks clause 4 of
// st-props-correct: no two own facets of the same kind.
func TestNewSimpleTypeRejectsDuplicateFacetKind(t *testing.T) {
	facets := []Facet{
		NewFacet(FacetMinLength, []string{"1"}, false),
		NewFacet(FacetMinLength, []string{"2"}, false),
	}
	_, err := NewSimpleType(xsderr.Loc{}, QName{}, Atomic{Primitive: anyAtomicType}, anyAtomicType, facets, nil)
	if err == nil {
		t.Fatal("NewSimpleType accepted duplicate facet kind, want rejection")
	}
	if r, _ := xsderr.RuleOf(err); r != ruleSTPropsCorrect {
		t.Fatalf("rule = %q, want %q", r, ruleSTPropsCorrect)
	}
}

// TestAnchorsNilContract exercises the anySimpleType/anyAtomicType nil
// contracts that this issue must actually construct, not just document.
func TestAnchorsNilContract(t *testing.T) {
	// anySimpleType: variety absent, base absent, IsAnySimpleType true.
	if !anySimpleType.IsAnySimpleType() {
		t.Error("anySimpleType.IsAnySimpleType() = false, want true")
	}
	if anySimpleType.Base() != nil {
		t.Error("anySimpleType.Base() != nil")
	}
	if anySimpleType.Variety() != nil {
		t.Error("anySimpleType.Variety() != nil, want absent")
	}

	// anyAtomicType: base is anySimpleType, variety Atomic with absent primitive.
	if anyAtomicType.IsAnySimpleType() {
		t.Error("anyAtomicType.IsAnySimpleType() = true, want false")
	}
	if anyAtomicType.Base() != anySimpleType {
		t.Error("anyAtomicType.Base() is not anySimpleType")
	}
	at, ok := anyAtomicType.Variety().(Atomic)
	if !ok {
		t.Fatalf("anyAtomicType.Variety() type = %T, want Atomic", anyAtomicType.Variety())
	}
	if at.Primitive != nil {
		t.Error("anyAtomicType {primitive type definition} is not absent")
	}
}

// TestIsPrimitive checks the derived primitive predicate across the anchors, a
// hand-built primitive-like type (base = anyAtomicType), and a type derived
// from that primitive.
func TestIsPrimitive(t *testing.T) {
	// A primitive-like type: its base IS anyAtomicType.
	prim, err := NewSimpleType(xsderr.Loc{}, QName{Space: XMLSchemaNS, Local: "decimal"},
		Atomic{Primitive: nil}, anyAtomicType, nil, nil)
	if err != nil {
		t.Fatalf("building primitive: %v", err)
	}
	// A derived type restricting the primitive.
	derived, err := NewSimpleType(xsderr.Loc{}, QName{Space: XMLSchemaNS, Local: "integer"},
		Atomic{Primitive: prim}, prim, nil, nil)
	if err != nil {
		t.Fatalf("building derived: %v", err)
	}

	if !prim.IsPrimitive() {
		t.Error("hand-built primitive IsPrimitive() = false, want true")
	}
	if derived.IsPrimitive() {
		t.Error("derived-from-primitive IsPrimitive() = true, want false")
	}
	if anyAtomicType.IsPrimitive() {
		t.Error("anyAtomicType.IsPrimitive() = true, want false (special, not primitive)")
	}
	if anySimpleType.IsPrimitive() {
		t.Error("anySimpleType.IsPrimitive() = true, want false (special, not primitive)")
	}
}

// mkAssertion builds a bare Assertion carrying only the given XPath test, the
// shape the assertions-accumulation tests below exercise.
func mkAssertion(expr string) Assertion {
	return NewAssertion(NewXPathExpression(expr, nil, nil, nil), nil)
}

// assertionsFacet returns the single FacetAssertions EffectiveFacet in eff (and
// fails if there is not exactly one), plus its assertion {test} expressions in
// document order — the accumulated {value} the §4.3.13.2 tests assert over.
func assertionsFacet(t *testing.T, eff []EffectiveFacet) (EffectiveFacet, []string) {
	t.Helper()
	var found EffectiveFacet
	count := 0
	for _, f := range eff {
		if f.Facet().Kind() != FacetAssertions {
			continue
		}
		count++
		found = f
	}
	if count != 1 {
		t.Fatalf("EffectiveFacets has %d assertions facets, want exactly 1", count)
	}
	as, ok := found.Facet().Assertions()
	if !ok {
		t.Fatal("Assertions() ok = false on an assertions facet")
	}
	exprs := make([]string, len(as))
	for i, a := range as {
		exprs[i] = a.Test().Expression()
	}
	return found, exprs
}

func wantExprs(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("accumulated assertions = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("accumulated assertions = %v, want %v", got, want)
		}
	}
}

// TestEffectiveFacetsAssertionsAccumulateTwoLevel exercises §4.3.13.2: a
// derived type's assertions {value} is the base type's Assertions followed by
// the derived type's own new Assertions, in that order (append, not replace).
// It also pins Declaring to the most-derived contributor and checks the
// cos-assertions-restriction (§4.3.13.4) prefix invariant holds.
func TestEffectiveFacetsAssertionsAccumulateTwoLevel(t *testing.T) {
	base, err := NewSimpleType(xsderr.Loc{}, QName{Local: "base"}, Atomic{}, anyAtomicType,
		[]Facet{NewAssertionsFacet([]Assertion{mkAssertion("@a > 0"), mkAssertion("@b < 10")})}, nil)
	if err != nil {
		t.Fatal(err)
	}
	derived, err := NewSimpleType(xsderr.Loc{}, QName{Local: "derived"}, Atomic{}, base,
		[]Facet{NewAssertionsFacet([]Assertion{mkAssertion("@c = 1")})}, nil)
	if err != nil {
		t.Fatal(err)
	}

	ef, got := assertionsFacet(t, derived.EffectiveFacets())
	wantExprs(t, got, []string{"@a > 0", "@b < 10", "@c = 1"})

	// Declaring reflects the most-derived contributor's position (Q2).
	if d := ef.Declaring(); d != (QName{Local: "derived"}) {
		t.Errorf("merged assertions Declaring() = %v, want {Local: derived}", d)
	}

	// cos-assertions-restriction (§4.3.13.4): base's {value} is a literal
	// prefix of the derived's accumulated {value}.
	_, baseExprs := assertionsFacet(t, base.EffectiveFacets())
	if len(baseExprs) > len(got) {
		t.Fatalf("base {value} longer than derived: base=%v derived=%v", baseExprs, got)
	}
	for i := range baseExprs {
		if got[i] != baseExprs[i] {
			t.Fatalf("base {value} %v is not a prefix of derived's %v", baseExprs, got)
		}
	}
}

// TestEffectiveFacetsAssertionsAccumulateThreeLevel exercises recursive
// accumulation across A <- B <- C (§4.3.13.2 point 4): C's effective assertions
// {value} is A's ++ B's-own ++ C's-own, oldest first.
func TestEffectiveFacetsAssertionsAccumulateThreeLevel(t *testing.T) {
	a, err := NewSimpleType(xsderr.Loc{}, QName{Local: "A"}, Atomic{}, anyAtomicType,
		[]Facet{NewAssertionsFacet([]Assertion{mkAssertion("a1")})}, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewSimpleType(xsderr.Loc{}, QName{Local: "B"}, Atomic{}, a,
		[]Facet{NewAssertionsFacet([]Assertion{mkAssertion("b1"), mkAssertion("b2")})}, nil)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewSimpleType(xsderr.Loc{}, QName{Local: "C"}, Atomic{}, b,
		[]Facet{NewAssertionsFacet([]Assertion{mkAssertion("c1")})}, nil)
	if err != nil {
		t.Fatal(err)
	}

	ef, got := assertionsFacet(t, c.EffectiveFacets())
	wantExprs(t, got, []string{"a1", "b1", "b2", "c1"})
	if d := ef.Declaring(); d != (QName{Local: "C"}) {
		t.Errorf("merged assertions Declaring() = %v, want {Local: C}", d)
	}
}

// TestEffectiveFacetsReplaceKindStillReplaces is the regression guard that the
// FacetAssertions accumulation is kind-selective: a single-valued replace-kind
// facet (FacetMaxInclusive) across a base/derived chain still REPLACES — the
// derived's value wins and the base's is dropped, not accumulated.
func TestEffectiveFacetsReplaceKindStillReplaces(t *testing.T) {
	base, err := NewSimpleType(xsderr.Loc{}, QName{Local: "base"}, Atomic{}, anyAtomicType,
		[]Facet{NewFacet(FacetMaxInclusive, []string{"100"}, false)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	derived, err := NewSimpleType(xsderr.Loc{}, QName{Local: "derived"}, Atomic{}, base,
		[]Facet{NewFacet(FacetMaxInclusive, []string{"50"}, false)}, nil)
	if err != nil {
		t.Fatal(err)
	}

	eff := derived.EffectiveFacets()
	if len(eff) != 1 {
		t.Fatalf("EffectiveFacets len = %d, want 1 (maxInclusive replaced, not accumulated)", len(eff))
	}
	if got := eff[0].Facet().Values(); len(got) != 1 || got[0] != "50" {
		t.Errorf("maxInclusive {value} = %v, want [50] (derived replaces base)", got)
	}
}

// TestEffectiveFacetsAssertionsMixedWithReplaceKind proves both behaviors
// coexist in ONE EffectiveFacets call: on the same two types, the assertions
// facet accumulates while a replace-kind facet (FacetLength) still replaces.
func TestEffectiveFacetsAssertionsMixedWithReplaceKind(t *testing.T) {
	base, err := NewSimpleType(xsderr.Loc{}, QName{Local: "base"}, Atomic{}, anyAtomicType,
		[]Facet{
			NewFacet(FacetLength, []string{"8"}, false),
			NewAssertionsFacet([]Assertion{mkAssertion("base1")}),
		}, nil)
	if err != nil {
		t.Fatal(err)
	}
	derived, err := NewSimpleType(xsderr.Loc{}, QName{Local: "derived"}, Atomic{}, base,
		[]Facet{
			NewFacet(FacetLength, []string{"4"}, false),
			NewAssertionsFacet([]Assertion{mkAssertion("derived1")}),
		}, nil)
	if err != nil {
		t.Fatal(err)
	}

	eff := derived.EffectiveFacets()
	_, got := assertionsFacet(t, eff)
	wantExprs(t, got, []string{"base1", "derived1"})

	lengthCount := 0
	for _, f := range eff {
		if f.Facet().Kind() != FacetLength {
			continue
		}
		lengthCount++
		if v := f.Facet().Values(); len(v) != 1 || v[0] != "4" {
			t.Errorf("length {value} = %v, want [4] (derived replaces base)", v)
		}
	}
	if lengthCount != 1 {
		t.Fatalf("EffectiveFacets has %d length facets, want exactly 1", lengthCount)
	}
}

// patternValues returns the Values() of every FacetPattern EffectiveFacet in
// eff, in order — one inner slice per surviving pattern facet.
func patternValues(eff []EffectiveFacet) [][]string {
	var out [][]string
	for _, f := range eff {
		if f.Facet().Kind() == FacetPattern {
			out = append(out, f.Facet().Values())
		}
	}
	return out
}

// TestEffectiveFacetsPatternKeepsBothTwoLevel exercises §4.3.4.2 (xr-pattern):
// a derived type that re-declares pattern does NOT supersede the base's pattern
// (unlike the 12 replace-kind facets) — both survive as separate EffectiveFacet
// entries so they can be ANDed at validation, base before derived.
func TestEffectiveFacetsPatternKeepsBothTwoLevel(t *testing.T) {
	base, err := NewSimpleType(xsderr.Loc{}, QName{Local: "base"}, Atomic{}, anyAtomicType,
		[]Facet{NewFacet(FacetPattern, []string{"[a-z]+"}, false)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	derived, err := NewSimpleType(xsderr.Loc{}, QName{Local: "derived"}, Atomic{}, base,
		[]Facet{NewFacet(FacetPattern, []string{"a.*"}, false)}, nil)
	if err != nil {
		t.Fatal(err)
	}

	got := patternValues(derived.EffectiveFacets())
	want := [][]string{{"[a-z]+"}, {"a.*"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pattern EffectiveFacets = %v, want %v (both survive, base then derived)", got, want)
	}
}

// TestEffectiveFacetsPatternKeepsBothThreeLevel mirrors the assertions
// three-level accumulation: A <- B <- C each with its own pattern facet yields
// three separate surviving FacetPattern EffectiveFacets, in base-to-derived
// order (§4.3.4.2 cross-step AND).
func TestEffectiveFacetsPatternKeepsBothThreeLevel(t *testing.T) {
	a, err := NewSimpleType(xsderr.Loc{}, QName{Local: "A"}, Atomic{}, anyAtomicType,
		[]Facet{NewFacet(FacetPattern, []string{"a1"}, false)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewSimpleType(xsderr.Loc{}, QName{Local: "B"}, Atomic{}, a,
		[]Facet{NewFacet(FacetPattern, []string{"b1"}, false)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewSimpleType(xsderr.Loc{}, QName{Local: "C"}, Atomic{}, b,
		[]Facet{NewFacet(FacetPattern, []string{"c1"}, false)}, nil)
	if err != nil {
		t.Fatal(err)
	}

	got := patternValues(c.EffectiveFacets())
	want := [][]string{{"a1"}, {"b1"}, {"c1"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pattern EffectiveFacets = %v, want %v (all three survive, oldest first)", got, want)
	}
}

// TestEffectiveFacetsAssertionsBaseHasNoneDerivedAdds exercises the
// plain-append branch of overlayFacet: when acc has no prior FacetAssertions
// entry, the derived type's assertions facet is appended unchanged.
func TestEffectiveFacetsAssertionsBaseHasNoneDerivedAdds(t *testing.T) {
	base, err := NewSimpleType(xsderr.Loc{}, QName{Local: "base"}, Atomic{}, anyAtomicType, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	derived, err := NewSimpleType(xsderr.Loc{}, QName{Local: "derived"}, Atomic{}, base,
		[]Facet{NewAssertionsFacet([]Assertion{mkAssertion("d1")})}, nil)
	if err != nil {
		t.Fatal(err)
	}

	ef, got := assertionsFacet(t, derived.EffectiveFacets())
	wantExprs(t, got, []string{"d1"})
	if d := ef.Declaring(); d != (QName{Local: "derived"}) {
		t.Errorf("Declaring() = %v, want {Local: derived}", d)
	}
}

// TestEffectiveFacetsAssertionsBaseHasDerivedAddsNone covers both ways a
// derived type can contribute no new assertions: (1) it declares no assertions
// facet at all — the base's facet is inherited unchanged, keeping the base's
// Declaring; (2) it declares an empty assertions facet — the base's Assertions
// survive but the merged facet takes the derived's most-derived position and
// Declaring.
func TestEffectiveFacetsAssertionsBaseHasDerivedAddsNone(t *testing.T) {
	base, err := NewSimpleType(xsderr.Loc{}, QName{Local: "base"}, Atomic{}, anyAtomicType,
		[]Facet{NewAssertionsFacet([]Assertion{mkAssertion("b1"), mkAssertion("b2")})}, nil)
	if err != nil {
		t.Fatal(err)
	}

	// (1) derived declares NO assertions facet: base's facet is inherited as-is.
	noFacet, err := NewSimpleType(xsderr.Loc{}, QName{Local: "noFacet"}, Atomic{}, base,
		[]Facet{NewFacet(FacetLength, []string{"3"}, false)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	ef1, got1 := assertionsFacet(t, noFacet.EffectiveFacets())
	wantExprs(t, got1, []string{"b1", "b2"})
	if d := ef1.Declaring(); d != (QName{Local: "base"}) {
		t.Errorf("inherited assertions Declaring() = %v, want {Local: base}", d)
	}

	// (2) derived declares an EMPTY assertions facet: base's Assertions survive,
	// but the merged facet takes the derived's position and Declaring.
	emptyFacet, err := NewSimpleType(xsderr.Loc{}, QName{Local: "emptyFacet"}, Atomic{}, base,
		[]Facet{NewAssertionsFacet(nil)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	ef2, got2 := assertionsFacet(t, emptyFacet.EffectiveFacets())
	wantExprs(t, got2, []string{"b1", "b2"})
	if d := ef2.Declaring(); d != (QName{Local: "emptyFacet"}) {
		t.Errorf("merged empty-derived assertions Declaring() = %v, want {Local: emptyFacet}", d)
	}
}

// TestEffectiveFacetsAssertionsNoDedup guards against a "helpful" set-union:
// §4.3.13.2 accumulation is a plain append, so identical assertion {test}s
// declared at two levels BOTH survive — length is base-count + derived-count.
func TestEffectiveFacetsAssertionsNoDedup(t *testing.T) {
	base, err := NewSimpleType(xsderr.Loc{}, QName{Local: "base"}, Atomic{}, anyAtomicType,
		[]Facet{NewAssertionsFacet([]Assertion{mkAssertion("@x = 1")})}, nil)
	if err != nil {
		t.Fatal(err)
	}
	derived, err := NewSimpleType(xsderr.Loc{}, QName{Local: "derived"}, Atomic{}, base,
		[]Facet{NewAssertionsFacet([]Assertion{mkAssertion("@x = 1")})}, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, got := assertionsFacet(t, derived.EffectiveFacets())
	wantExprs(t, got, []string{"@x = 1", "@x = 1"})
}

// TestEffectiveFacetsAssertionsMergeCopyIndependence checks the new merge path
// returns non-aliased data: mutating a slice returned from the merged facet's
// Assertions() does not affect the stored facet on a later call.
func TestEffectiveFacetsAssertionsMergeCopyIndependence(t *testing.T) {
	base, err := NewSimpleType(xsderr.Loc{}, QName{Local: "base"}, Atomic{}, anyAtomicType,
		[]Facet{NewAssertionsFacet([]Assertion{mkAssertion("b1")})}, nil)
	if err != nil {
		t.Fatal(err)
	}
	derived, err := NewSimpleType(xsderr.Loc{}, QName{Local: "derived"}, Atomic{}, base,
		[]Facet{NewAssertionsFacet([]Assertion{mkAssertion("d1")})}, nil)
	if err != nil {
		t.Fatal(err)
	}

	ef, _ := assertionsFacet(t, derived.EffectiveFacets())
	first, _ := ef.Facet().Assertions()
	first[0] = mkAssertion("MUTATED")

	_, again := assertionsFacet(t, derived.EffectiveFacets())
	wantExprs(t, again, []string{"b1", "d1"})
}

// TestOwnVsEffectiveFacets exercises the §3.16.6.4 overlay across a 3-level
// restriction chain: anyAtomicType -> primitive -> mid -> leaf. A more-derived
// same-kind facet masks the base's facet, and non-superseded facets survive.
func TestOwnVsEffectiveFacets(t *testing.T) {
	prim, err := NewSimpleType(xsderr.Loc{}, QName{Local: "prim"}, Atomic{}, anyAtomicType,
		[]Facet{NewFacet(FacetWhiteSpace, []string{"collapse"}, true)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	mid, err := NewSimpleType(xsderr.Loc{}, QName{Local: "mid"}, Atomic{Primitive: prim}, prim,
		[]Facet{
			NewFacet(FacetMinLength, []string{"1"}, false),
			NewFacet(FacetMaxLength, []string{"10"}, false),
		}, nil)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := NewSimpleType(xsderr.Loc{}, QName{Local: "leaf"}, Atomic{Primitive: prim}, mid,
		[]Facet{NewFacet(FacetMaxLength, []string{"5"}, false)}, nil)
	if err != nil {
		t.Fatal(err)
	}

	// OwnFacets is only the leaf's own contribution.
	own := leaf.OwnFacets()
	if len(own) != 1 || own[0].Kind() != FacetMaxLength || own[0].Values()[0] != "5" {
		t.Fatalf("leaf.OwnFacets() = %+v, want single maxLength=5", own)
	}

	// EffectiveFacets accumulates the whole chain, base-to-derived, with the
	// leaf's maxLength=5 masking mid's maxLength=10.
	eff := leaf.EffectiveFacets()
	byKind := map[FacetKind]EffectiveFacet{}
	var order []FacetKind
	for _, f := range eff {
		if _, dup := byKind[f.Facet().Kind()]; dup {
			t.Fatalf("EffectiveFacets has duplicate kind %s", f.Facet().Kind())
		}
		byKind[f.Facet().Kind()] = f
		order = append(order, f.Facet().Kind())
	}
	if len(eff) != 3 {
		t.Fatalf("EffectiveFacets len = %d (%v), want 3", len(eff), order)
	}
	if byKind[FacetWhiteSpace].Facet().Values()[0] != "collapse" {
		t.Error("whiteSpace from primitive did not survive")
	}
	if byKind[FacetMinLength].Facet().Values()[0] != "1" {
		t.Error("minLength from mid did not survive")
	}
	if byKind[FacetMaxLength].Facet().Values()[0] != "5" {
		t.Errorf("maxLength = %q, want leaf's 5 (masking mid's 10)", byKind[FacetMaxLength].Facet().Values()[0])
	}

	// Provenance: each effective facet reports the {name} of the type on the
	// chain that DECLARED it, not the leaf that inherits it. whiteSpace came
	// from prim, minLength from mid, and the overriding maxLength from leaf.
	if got := byKind[FacetWhiteSpace].Declaring(); got != (QName{Local: "prim"}) {
		t.Errorf("whiteSpace Declaring() = %v, want {Local: prim}", got)
	}
	if got := byKind[FacetMinLength].Declaring(); got != (QName{Local: "mid"}) {
		t.Errorf("minLength Declaring() = %v, want {Local: mid}", got)
	}
	if got := byKind[FacetMaxLength].Declaring(); got != (QName{Local: "leaf"}) {
		t.Errorf("maxLength Declaring() = %v, want leaf (overriding type's own name)", got)
	}

	// Deterministic base-to-derived order: whiteSpace (prim) before minLength
	// (mid) before the overriding maxLength (leaf position).
	want := []FacetKind{FacetWhiteSpace, FacetMinLength, FacetMaxLength}
	for i := range want {
		if order[i] != want[i] {
			t.Errorf("EffectiveFacets order = %v, want %v", order, want)
		}
	}

	// An anonymous restriction (zero {name}) that contributes its own facet
	// reports the zero QName as provenance — the zero-value-means-anonymous
	// convention, not a missing value.
	anon, err := NewSimpleType(xsderr.Loc{}, QName{}, Atomic{Primitive: prim}, leaf,
		[]Facet{NewFacet(FacetLength, []string{"3"}, false)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range anon.EffectiveFacets() {
		if f.Facet().Kind() != FacetLength {
			continue
		}
		if got := f.Declaring(); got != (QName{}) {
			t.Errorf("anonymous-declared length Declaring() = %v, want zero QName", got)
		}
	}

	// Anchors carry no facets.
	if anySimpleType.EffectiveFacets() != nil {
		t.Error("anySimpleType.EffectiveFacets() != nil")
	}
	if anyAtomicType.EffectiveFacets() != nil {
		t.Error("anyAtomicType.EffectiveFacets() != nil")
	}
}
