package xsd

import (
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
		{FacetAssertions, true, false, false},  // normalized: assertions has no {fixed}
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
	prim, err := NewSimpleType(xsderr.Loc{}, QName{Space: xsdNamespace, Local: "decimal"},
		Atomic{Primitive: nil}, anyAtomicType, nil, nil)
	if err != nil {
		t.Fatalf("building primitive: %v", err)
	}
	// A derived type restricting the primitive.
	derived, err := NewSimpleType(xsderr.Loc{}, QName{Space: xsdNamespace, Local: "integer"},
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
	byKind := map[FacetKind]Facet{}
	var order []FacetKind
	for _, f := range eff {
		if _, dup := byKind[f.Kind()]; dup {
			t.Fatalf("EffectiveFacets has duplicate kind %s", f.Kind())
		}
		byKind[f.Kind()] = f
		order = append(order, f.Kind())
	}
	if len(eff) != 3 {
		t.Fatalf("EffectiveFacets len = %d (%v), want 3", len(eff), order)
	}
	if byKind[FacetWhiteSpace].Values()[0] != "collapse" {
		t.Error("whiteSpace from primitive did not survive")
	}
	if byKind[FacetMinLength].Values()[0] != "1" {
		t.Error("minLength from mid did not survive")
	}
	if byKind[FacetMaxLength].Values()[0] != "5" {
		t.Errorf("maxLength = %q, want leaf's 5 (masking mid's 10)", byKind[FacetMaxLength].Values()[0])
	}

	// Deterministic base-to-derived order: whiteSpace (prim) before minLength
	// (mid) before the overriding maxLength (leaf position).
	want := []FacetKind{FacetWhiteSpace, FacetMinLength, FacetMaxLength}
	for i := range want {
		if order[i] != want[i] {
			t.Errorf("EffectiveFacets order = %v, want %v", order, want)
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
