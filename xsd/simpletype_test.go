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
		{FacetPattern, true, false, false}, // normalized: pattern has no {fixed}
		// FacetEnumeration and FacetAssertions (also fixed-less) are excluded:
		// NewFacet panics for both; see TestNewFacetEnumerationPanics /
		// TestNewFacetAssertionsPanics and NewEnumerationFacet / NewAssertionsFacet.
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
	f := NewFacet(FacetPattern, in, false)
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

// TestNewFacetEnumerationPanics confirms NewFacet rejects FacetEnumeration: the
// enumeration facet models {value} as EnumerationMembers carrying namespace
// context (§4.3.5/§3.3.18), so it must go through NewEnumerationFacet, and using
// the wrong constructor is a caught programmer error, not a silent mis-build.
func TestNewFacetEnumerationPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("NewFacet(FacetEnumeration, ...): want panic, got none")
		}
	}()
	_ = NewFacet(FacetEnumeration, []string{"red"}, false)
}

// TestNewEnumerationFacetRoundTrip verifies NewEnumerationFacet builds an
// enumeration-kind Facet whose EnumerationMembers round-trips in document order
// with no {fixed} property, whose Values derives the plain lexical strings from
// the members (STYLE D3, not stored twice), and with defensive-copy semantics on
// both the input slice and the returned slice.
func TestNewEnumerationFacetRoundTrip(t *testing.T) {
	ns := "myNamespace"
	m0 := NewEnumerationMember("foo:fo", []NamespaceBinding{NewNamespaceBinding("foo", ns)}, nil)
	m1 := NewEnumerationMember("bar", nil, &ns)
	in := []EnumerationMember{m0, m1}
	f := NewEnumerationFacet(in)

	if f.Kind() != FacetEnumeration {
		t.Fatalf("Kind() = %s, want enumeration", f.Kind())
	}
	if _, ok := f.Fixed(); ok {
		t.Error("Fixed() ok = true, want false (enumeration has no {fixed})")
	}

	// Values is derived from the members' Lexical() in document order.
	if got := f.Values(); !reflect.DeepEqual(got, []string{"foo:fo", "bar"}) {
		t.Errorf("Values() = %v, want [foo:fo bar]", got)
	}

	got, ok := f.EnumerationMembers()
	if !ok {
		t.Fatal("EnumerationMembers() ok = false, want true for an enumeration facet")
	}
	if len(got) != 2 {
		t.Fatalf("EnumerationMembers() len = %d, want 2", len(got))
	}
	if got[0].Lexical() != "foo:fo" || got[1].Lexical() != "bar" {
		t.Errorf("EnumerationMembers() document order wrong: got %q, %q",
			got[0].Lexical(), got[1].Lexical())
	}
	// Member 0 carries a "foo" binding and no default; member 1 carries a default.
	if binds := got[0].NamespaceBindings(); len(binds) != 1 || binds[0].Prefix() != "foo" || binds[0].Namespace() != ns {
		t.Errorf("member 0 NamespaceBindings() = %v, want one foo=%s binding", binds, ns)
	}
	if _, ok := got[0].DefaultNamespace(); ok {
		t.Error("member 0 DefaultNamespace() ok = true, want false (absent)")
	}
	if d, ok := got[1].DefaultNamespace(); !ok || d != ns {
		t.Errorf("member 1 DefaultNamespace() = %q,%v, want %q,true", d, ok, ns)
	}

	// Mutating the caller's input slice must not affect the facet.
	in[0] = m1
	if again, _ := f.EnumerationMembers(); again[0].Lexical() != "foo:fo" {
		t.Errorf("NewEnumerationFacet aliased caller slice: got %q, want %q",
			again[0].Lexical(), "foo:fo")
	}
	// Mutating the returned slice must not affect the facet.
	got[0] = m1
	if again, _ := f.EnumerationMembers(); again[0].Lexical() != "foo:fo" {
		t.Errorf("EnumerationMembers() aliased internal slice: got %q, want %q",
			again[0].Lexical(), "foo:fo")
	}
}

// TestEnumerationMembersOnNonEnumerationFacet verifies EnumerationMembers reports
// ok == false and nil for a facet whose kind is not FacetEnumeration.
func TestEnumerationMembersOnNonEnumerationFacet(t *testing.T) {
	f := NewFacet(FacetLength, []string{"3"}, false)
	got, ok := f.EnumerationMembers()
	if ok {
		t.Error("EnumerationMembers() ok = true for a length facet, want false")
	}
	if got != nil {
		t.Errorf("EnumerationMembers() = %v for a length facet, want nil", got)
	}
}

// TestEnumerationMemberBindingCopied verifies NewEnumerationMember copies the
// input bindings slice and NamespaceBindings returns a copy, so no caller aliases
// the member's backing array (mirroring XPathExpression's discipline).
func TestEnumerationMemberBindingCopied(t *testing.T) {
	in := []NamespaceBinding{NewNamespaceBinding("a", "urn:a"), NewNamespaceBinding("b", "urn:b")}
	m := NewEnumerationMember("a:x", in, nil)
	in[0] = NewNamespaceBinding("a", "mutated")
	got := m.NamespaceBindings()
	if got[0].Namespace() != "urn:a" {
		t.Fatalf("NewEnumerationMember aliased caller slice: got %q, want %q", got[0].Namespace(), "urn:a")
	}
	got[1] = NewNamespaceBinding("b", "clobber")
	if again := m.NamespaceBindings(); again[1].Namespace() != "urn:b" {
		t.Fatalf("NamespaceBindings() aliased internal slice: got %q, want %q", again[1].Namespace(), "urn:b")
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
	dec := mustPrim(t, "decimal")
	_, err := newCheckedSimpleType(xsderr.Loc{}, QName{}, RestrictionDerivation{}, dec, nil,
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
	dec := mustPrim(t, "decimal")
	st, err := newCheckedSimpleType(xsderr.Loc{}, QName{}, RestrictionDerivation{}, dec, nil, final)
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
	dec := mustPrim(t, "decimal")
	_, err := newCheckedSimpleType(xsderr.Loc{}, QName{}, RestrictionDerivation{}, dec, facets, nil)
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
	if mustBase(anySimpleType) != nil {
		t.Error("anySimpleType.Base() != nil")
	}
	if mustVariety(anySimpleType) != nil {
		t.Error("anySimpleType.Variety() != nil, want absent")
	}

	// anyAtomicType: base is anySimpleType, variety Atomic with absent primitive.
	if anyAtomicType.IsAnySimpleType() {
		t.Error("anyAtomicType.IsAnySimpleType() = true, want false")
	}
	if mustBase(anyAtomicType) != anySimpleType {
		t.Error("anyAtomicType.Base() is not anySimpleType")
	}
	if _, ok := mustVariety(anyAtomicType).(Atomic); !ok {
		t.Fatalf("anyAtomicType.Variety() type = %T, want Atomic", mustVariety(anyAtomicType))
	}
	if mustPrimitive(anyAtomicType) != nil {
		t.Error("anyAtomicType {primitive type definition} is not absent")
	}
	if anyAtomicType.IsPrimitive() {
		t.Error("anyAtomicType.IsPrimitive() = true, want false")
	}
}

// TestAnyAtomicTypeResolvedTriple pins xs:anyAtomicType's own encoding: it is
// the ONE component that is atomic by fiat (Datatypes §4.1.6) while its
// {primitive type definition} is ·absent· and it is not itself a primitive
// datatype. No other arm of SimpleTypeDerivation can express that triple —
// a restriction would inherit xs:anySimpleType's absent {variety}, and the
// primitive arm would report the node as its own {primitive type definition} —
// so the anchor carries a package-private arm of its own rather than being keyed
// on nil-ness of the derivation, which would give "absent derivation" a second
// meaning. checkAtomicGraph reads mustVariety(t.base) for EVERY primitive, so the
// first of the three is load-bearing for the whole primitive cohort.
func TestAnyAtomicTypeResolvedTriple(t *testing.T) {
	anchor := AnyAtomicType()
	if _, ok := mustVariety(anchor).(Atomic); !ok {
		t.Errorf("AnyAtomicType().Variety() = %T, want Atomic", mustVariety(anchor))
	}
	if got := mustPrimitive(anchor); got != nil {
		t.Errorf("AnyAtomicType().Primitive() = %v, want nil (·absent·)", got)
	}
	if anchor.IsPrimitive() {
		t.Error("AnyAtomicType().IsPrimitive() = true, want false")
	}
}

// TestIsPrimitive checks the derived primitive predicate across the anchors, a
// primitive datatype (base = anyAtomicType), and a type derived from that
// primitive.
func TestIsPrimitive(t *testing.T) {
	// A primitive datatype: NewPrimitiveType fixes its base to anyAtomicType.
	prim, err := newCheckedPrimitiveType(xsderr.Loc{}, QName{Space: XMLSchemaNS, Local: "decimal"},
		nil, nil)
	if err != nil {
		t.Fatalf("building primitive: %v", err)
	}
	// A derived type restricting the primitive.
	derived, err := newCheckedSimpleType(xsderr.Loc{}, QName{Space: XMLSchemaNS, Local: "integer"},
		RestrictionDerivation{}, prim, nil, nil)
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

// TestDerivedPropertiesFollowTheBaseChain checks the §3.16.2.1 mapping the four
// derived readers implement: each arm mints its own property, and a
// RestrictionDerivation mints none — it reports whatever the nearest ancestor
// that DOES mint one reports, with no producer copying a pointer down. The
// two-step chains are the point: a restriction of a restriction of a primitive
// still reports that primitive, and a restriction of a list still reports the
// list's item.
func TestDerivedPropertiesFollowTheBaseChain(t *testing.T) {
	prim, err := newCheckedPrimitiveType(xsderr.Loc{}, QName{Local: "prim"}, nil, nil)
	if err != nil {
		t.Fatalf("building prim: %v", err)
	}
	derived := mustSimple(t, "derived", RestrictionDerivation{}, prim, nil)
	derived2 := mustSimple(t, "derived2", RestrictionDerivation{}, derived, nil)

	list := mustSimple(t, "list", ListDerivation{Item: prim}, anySimpleType, constructedListFacets())
	listRestricted := mustSimple(t, "listRestricted", RestrictionDerivation{}, list, nil)

	other, err := newCheckedPrimitiveType(xsderr.Loc{}, QName{Local: "other"}, nil, nil)
	if err != nil {
		t.Fatalf("building other: %v", err)
	}
	union := mustSimple(t, "union", UnionDerivation{Members: []*SimpleType{prim, other}}, anySimpleType, nil)
	unionRestricted := mustSimple(t, "unionRestricted",
		UnionDerivation{Members: []*SimpleType{prim, other}}, union, nil)

	// {variety}: minted by the list/union/primitive arms, inherited by restriction.
	for _, c := range []struct {
		name string
		st   *SimpleType
		want Variety
	}{
		{"prim", prim, Atomic{}},
		{"derived2", derived2, Atomic{}},
		{"list", list, List{}},
		{"listRestricted", listRestricted, List{}},
		{"union", union, Union{}},
		{"unionRestricted", unionRestricted, Union{}},
		{"anySimpleType", anySimpleType, nil},
	} {
		if got := mustVariety(c.st); got != c.want {
			t.Errorf("%s.Variety() = %#v, want %#v", c.name, got, c.want)
		}
	}

	// {primitive type definition}: a primitive is its own; a restriction takes
	// its base's; list and union have none.
	for _, c := range []struct {
		name string
		st   *SimpleType
		want *SimpleType
	}{
		{"prim", prim, prim},
		{"derived", derived, prim},
		{"derived2", derived2, prim},
		{"list", list, nil},
		{"union", union, nil},
	} {
		if got := mustPrimitive(c.st); got != c.want {
			t.Errorf("%s.Primitive() = %v, want %v", c.name, got, c.want)
		}
	}

	// {item type definition}: minted by the list arm, inherited by restriction.
	if got := mustItem(list); got != prim {
		t.Errorf("list.Item() = %v, want prim", got)
	}
	if got := mustItem(listRestricted); got != prim {
		t.Errorf("listRestricted.Item() = %v, want prim (derived from the base list, no copy)", got)
	}
	if got := mustItem(derived); got != nil {
		t.Errorf("derived.Item() = %v, want nil (an atomic type has no {item type definition})", got)
	}

	// {member type definitions}: minted by the union arm, inherited by
	// restriction, and returned in document order.
	for _, c := range []struct {
		name string
		st   *SimpleType
	}{{"union", union}, {"unionRestricted", unionRestricted}} {
		got := mustMembers(c.st)
		if len(got) != 2 || got[0] != prim || got[1] != other {
			t.Errorf("%s.Members() = %v, want [prim other]", c.name, got)
		}
	}
	if got := mustMembers(derived); got != nil {
		t.Errorf("derived.Members() = %v, want nil", got)
	}
}

// TestDerivedReadersAreTotal pins that the four derived readers never panic on
// the partially-built shapes the constructors themselves produce and
// CheckDerivation (derivation.go) later reads them off: a type with a nil
// derivation AND a nil base (the anonymous placeholder several callers build),
// and a restriction whose base's own base is nil. Each returns the ·absent·
// value instead.
func TestDerivedReadersAreTotal(t *testing.T) {
	anon, err := newCheckedSimpleType(xsderr.Loc{}, QName{}, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("newCheckedSimpleType(nil derivation, nil base): %v", err)
	}
	// A restriction whose base chain runs out: anon's own base is nil.
	overAnon := &SimpleType{name: QName{Local: "overAnon"}, derivation: RestrictionDerivation{}, base: OwnedSimpleType{Definition: anon}}

	for _, c := range []struct {
		name string
		st   *SimpleType
	}{{"nil derivation and nil base", anon}, {"restriction over a nil-based base", overAnon}} {
		if got := mustVariety(c.st); got != nil {
			t.Errorf("%s: Variety() = %#v, want nil", c.name, got)
		}
		if got := mustPrimitive(c.st); got != nil {
			t.Errorf("%s: Primitive() = %v, want nil", c.name, got)
		}
		if got := mustItem(c.st); got != nil {
			t.Errorf("%s: Item() = %v, want nil", c.name, got)
		}
		if got := mustMembers(c.st); got != nil {
			t.Errorf("%s: Members() = %v, want nil", c.name, got)
		}
		if c.st.IsPrimitive() {
			t.Errorf("%s: IsPrimitive() = true, want false", c.name)
		}
	}
}

// TestUnionMembersPreserveSequenceVerbatim proves a UnionDerivation's membership
// is neither sorted, deduplicated, nor stripped of nil members: position is
// load-bearing, because cos-st-restricts clause 3.2.2.3 pairs a restriction's
// members with the base's positionally, and checkUnionGraph — not the
// constructor — is what rejects a nil member.
func TestUnionMembersPreserveSequenceVerbatim(t *testing.T) {
	a, err := newCheckedPrimitiveType(xsderr.Loc{}, QName{Local: "a"}, nil, nil)
	if err != nil {
		t.Fatalf("building a: %v", err)
	}
	b, err := newCheckedPrimitiveType(xsderr.Loc{}, QName{Local: "b"}, nil, nil)
	if err != nil {
		t.Fatalf("building b: %v", err)
	}

	want := []*SimpleType{b, a, b, nil}
	// checkUnionGraph rejects the nil member, so the component is built as a
	// struct literal: what is under test is copyDerivation and Members, not the
	// graph check.
	u := &SimpleType{derivation: copyDerivation(UnionDerivation{Members: want})}
	got := mustMembers(u)
	if len(got) != len(want) {
		t.Fatalf("Members() length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Members()[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

// TestUnionMembershipIsCopiedBothWays proves the immutability that makes a
// mutation-induced membership cycle structurally unrepresentable (see
// CheckDerivation): NewSimpleType copies a UnionDerivation's Members IN
// (copyDerivation), and SimpleType.Members copies it OUT, so neither the
// caller's backing array nor a returned slice aliases the component's own —
// which is what the exported Members field would otherwise give away.
func TestUnionMembershipIsCopiedBothWays(t *testing.T) {
	a, err := newCheckedPrimitiveType(xsderr.Loc{}, QName{Local: "a"}, nil, nil)
	if err != nil {
		t.Fatalf("building a: %v", err)
	}
	b, err := newCheckedPrimitiveType(xsderr.Loc{}, QName{Local: "b"}, nil, nil)
	if err != nil {
		t.Fatalf("building b: %v", err)
	}

	src := []*SimpleType{a}
	u := mustSimple(t, "u", UnionDerivation{Members: src}, anySimpleType, nil)

	// Mutating the caller's backing array must not reach u.
	src[0] = b
	if got := mustMembers(u); len(got) != 1 || got[0] != a {
		t.Errorf("after mutating the caller's slice, Members() = %v, want [a] — NewSimpleType must copy in", got)
	}

	// Mutating a returned slice must not reach u either.
	out := mustMembers(u)
	out[0] = b
	if got := mustMembers(u); len(got) != 1 || got[0] != a {
		t.Errorf("after mutating a Members() result, Members() = %v, want [a] — Members must copy out", got)
	}
}

// mustSimple builds a named simple type through NewSimpleType or fails the test.
func mustSimple(t *testing.T, name string, derivation SimpleTypeDerivation, base *SimpleType, ownFacets []Facet) *SimpleType {
	t.Helper()
	st, err := newCheckedSimpleType(xsderr.Loc{}, QName{Local: name}, derivation, base, ownFacets, nil)
	if err != nil {
		t.Fatalf("newCheckedSimpleType(%s): %v", name, err)
	}
	return st
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
	base, err := newCheckedPrimitiveType(xsderr.Loc{}, QName{Local: "base"},
		[]Facet{NewAssertionsFacet([]Assertion{mkAssertion("@a > 0"), mkAssertion("@b < 10")})}, nil)
	if err != nil {
		t.Fatal(err)
	}
	derived, err := newCheckedSimpleType(xsderr.Loc{}, QName{Local: "derived"}, RestrictionDerivation{}, base,
		[]Facet{NewAssertionsFacet([]Assertion{mkAssertion("@c = 1")})}, nil)
	if err != nil {
		t.Fatal(err)
	}

	ef, got := assertionsFacet(t, mustEffectiveFacets(derived))
	wantExprs(t, got, []string{"@a > 0", "@b < 10", "@c = 1"})

	// Declaring reflects the most-derived contributor's position (Q2).
	if d := ef.Declaring(); d != (QName{Local: "derived"}) {
		t.Errorf("merged assertions Declaring() = %v, want {Local: derived}", d)
	}

	// cos-assertions-restriction (§4.3.13.4): base's {value} is a literal
	// prefix of the derived's accumulated {value}.
	_, baseExprs := assertionsFacet(t, mustEffectiveFacets(base))
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
	a, err := newCheckedPrimitiveType(xsderr.Loc{}, QName{Local: "A"},
		[]Facet{NewAssertionsFacet([]Assertion{mkAssertion("a1")})}, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := newCheckedSimpleType(xsderr.Loc{}, QName{Local: "B"}, RestrictionDerivation{}, a,
		[]Facet{NewAssertionsFacet([]Assertion{mkAssertion("b1"), mkAssertion("b2")})}, nil)
	if err != nil {
		t.Fatal(err)
	}
	c, err := newCheckedSimpleType(xsderr.Loc{}, QName{Local: "C"}, RestrictionDerivation{}, b,
		[]Facet{NewAssertionsFacet([]Assertion{mkAssertion("c1")})}, nil)
	if err != nil {
		t.Fatal(err)
	}

	ef, got := assertionsFacet(t, mustEffectiveFacets(c))
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
	base, err := newCheckedPrimitiveType(xsderr.Loc{}, QName{Local: "base"},
		[]Facet{NewFacet(FacetMaxInclusive, []string{"100"}, false)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	derived, err := newCheckedSimpleType(xsderr.Loc{}, QName{Local: "derived"}, RestrictionDerivation{}, base,
		[]Facet{NewFacet(FacetMaxInclusive, []string{"50"}, false)}, nil)
	if err != nil {
		t.Fatal(err)
	}

	eff := mustEffectiveFacets(derived)
	if len(eff) != 1 {
		t.Fatalf("EffectiveFacets len = %d, want 1 (maxInclusive replaced, not accumulated)", len(eff))
	}
	if got := eff[0].Facet().Values(); len(got) != 1 || got[0] != "50" {
		t.Errorf("maxInclusive {value} = %v, want [50] (derived replaces base)", got)
	}
}

// TestEffectiveFacetsAssertionsMixedWithReplaceKind proves both behaviors
// coexist in ONE EffectiveFacets call: on the same two types, the assertions
// facet accumulates while a replace-kind facet (FacetMaxLength) still replaces.
// maxLength, not length, carries the replace-kind role here: length may not move
// at all across a restriction (length-valid-restriction, §4.3.1.4), so a
// narrowing length pair no longer constructs.
func TestEffectiveFacetsAssertionsMixedWithReplaceKind(t *testing.T) {
	base, err := newCheckedPrimitiveType(xsderr.Loc{}, QName{Local: "base"},
		[]Facet{
			NewFacet(FacetMaxLength, []string{"8"}, false),
			NewAssertionsFacet([]Assertion{mkAssertion("base1")}),
		}, nil)
	if err != nil {
		t.Fatal(err)
	}
	derived, err := newCheckedSimpleType(xsderr.Loc{}, QName{Local: "derived"}, RestrictionDerivation{}, base,
		[]Facet{
			NewFacet(FacetMaxLength, []string{"4"}, false),
			NewAssertionsFacet([]Assertion{mkAssertion("derived1")}),
		}, nil)
	if err != nil {
		t.Fatal(err)
	}

	eff := mustEffectiveFacets(derived)
	_, got := assertionsFacet(t, eff)
	wantExprs(t, got, []string{"base1", "derived1"})

	maxLengthCount := 0
	for _, f := range eff {
		if f.Facet().Kind() != FacetMaxLength {
			continue
		}
		maxLengthCount++
		if v := f.Facet().Values(); len(v) != 1 || v[0] != "4" {
			t.Errorf("maxLength {value} = %v, want [4] (derived replaces base)", v)
		}
	}
	if maxLengthCount != 1 {
		t.Fatalf("EffectiveFacets has %d maxLength facets, want exactly 1", maxLengthCount)
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
	base, err := newCheckedPrimitiveType(xsderr.Loc{}, QName{Local: "base"},
		[]Facet{NewFacet(FacetPattern, []string{"[a-z]+"}, false)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	derived, err := newCheckedSimpleType(xsderr.Loc{}, QName{Local: "derived"}, RestrictionDerivation{}, base,
		[]Facet{NewFacet(FacetPattern, []string{"a.*"}, false)}, nil)
	if err != nil {
		t.Fatal(err)
	}

	got := patternValues(mustEffectiveFacets(derived))
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
	a, err := newCheckedPrimitiveType(xsderr.Loc{}, QName{Local: "A"},
		[]Facet{NewFacet(FacetPattern, []string{"a1"}, false)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := newCheckedSimpleType(xsderr.Loc{}, QName{Local: "B"}, RestrictionDerivation{}, a,
		[]Facet{NewFacet(FacetPattern, []string{"b1"}, false)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	c, err := newCheckedSimpleType(xsderr.Loc{}, QName{Local: "C"}, RestrictionDerivation{}, b,
		[]Facet{NewFacet(FacetPattern, []string{"c1"}, false)}, nil)
	if err != nil {
		t.Fatal(err)
	}

	got := patternValues(mustEffectiveFacets(c))
	want := [][]string{{"a1"}, {"b1"}, {"c1"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pattern EffectiveFacets = %v, want %v (all three survive, oldest first)", got, want)
	}
}

// TestEffectiveFacetsAssertionsBaseHasNoneDerivedAdds exercises the
// plain-append branch of overlayFacet: when acc has no prior FacetAssertions
// entry, the derived type's assertions facet is appended unchanged.
func TestEffectiveFacetsAssertionsBaseHasNoneDerivedAdds(t *testing.T) {
	base, err := newCheckedPrimitiveType(xsderr.Loc{}, QName{Local: "base"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	derived, err := newCheckedSimpleType(xsderr.Loc{}, QName{Local: "derived"}, RestrictionDerivation{}, base,
		[]Facet{NewAssertionsFacet([]Assertion{mkAssertion("d1")})}, nil)
	if err != nil {
		t.Fatal(err)
	}

	ef, got := assertionsFacet(t, mustEffectiveFacets(derived))
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
	base, err := newCheckedPrimitiveType(xsderr.Loc{}, QName{Local: "base"},
		[]Facet{NewAssertionsFacet([]Assertion{mkAssertion("b1"), mkAssertion("b2")})}, nil)
	if err != nil {
		t.Fatal(err)
	}

	// (1) derived declares NO assertions facet: base's facet is inherited as-is.
	noFacet, err := newCheckedSimpleType(xsderr.Loc{}, QName{Local: "noFacet"}, RestrictionDerivation{}, base,
		[]Facet{NewFacet(FacetLength, []string{"3"}, false)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	ef1, got1 := assertionsFacet(t, mustEffectiveFacets(noFacet))
	wantExprs(t, got1, []string{"b1", "b2"})
	if d := ef1.Declaring(); d != (QName{Local: "base"}) {
		t.Errorf("inherited assertions Declaring() = %v, want {Local: base}", d)
	}

	// (2) derived declares an EMPTY assertions facet: base's Assertions survive,
	// but the merged facet takes the derived's position and Declaring.
	emptyFacet, err := newCheckedSimpleType(xsderr.Loc{}, QName{Local: "emptyFacet"}, RestrictionDerivation{}, base,
		[]Facet{NewAssertionsFacet(nil)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	ef2, got2 := assertionsFacet(t, mustEffectiveFacets(emptyFacet))
	wantExprs(t, got2, []string{"b1", "b2"})
	if d := ef2.Declaring(); d != (QName{Local: "emptyFacet"}) {
		t.Errorf("merged empty-derived assertions Declaring() = %v, want {Local: emptyFacet}", d)
	}
}

// TestEffectiveFacetsAssertionsNoDedup guards against a "helpful" set-union:
// §4.3.13.2 accumulation is a plain append, so identical assertion {test}s
// declared at two levels BOTH survive — length is base-count + derived-count.
func TestEffectiveFacetsAssertionsNoDedup(t *testing.T) {
	base, err := newCheckedPrimitiveType(xsderr.Loc{}, QName{Local: "base"},
		[]Facet{NewAssertionsFacet([]Assertion{mkAssertion("@x = 1")})}, nil)
	if err != nil {
		t.Fatal(err)
	}
	derived, err := newCheckedSimpleType(xsderr.Loc{}, QName{Local: "derived"}, RestrictionDerivation{}, base,
		[]Facet{NewAssertionsFacet([]Assertion{mkAssertion("@x = 1")})}, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, got := assertionsFacet(t, mustEffectiveFacets(derived))
	wantExprs(t, got, []string{"@x = 1", "@x = 1"})
}

// TestEffectiveFacetsAssertionsMergeCopyIndependence checks the new merge path
// returns non-aliased data: mutating a slice returned from the merged facet's
// Assertions() does not affect the stored facet on a later call.
func TestEffectiveFacetsAssertionsMergeCopyIndependence(t *testing.T) {
	base, err := newCheckedPrimitiveType(xsderr.Loc{}, QName{Local: "base"},
		[]Facet{NewAssertionsFacet([]Assertion{mkAssertion("b1")})}, nil)
	if err != nil {
		t.Fatal(err)
	}
	derived, err := newCheckedSimpleType(xsderr.Loc{}, QName{Local: "derived"}, RestrictionDerivation{}, base,
		[]Facet{NewAssertionsFacet([]Assertion{mkAssertion("d1")})}, nil)
	if err != nil {
		t.Fatal(err)
	}

	ef, _ := assertionsFacet(t, mustEffectiveFacets(derived))
	first, _ := ef.Facet().Assertions()
	first[0] = mkAssertion("MUTATED")

	_, again := assertionsFacet(t, mustEffectiveFacets(derived))
	wantExprs(t, again, []string{"b1", "d1"})
}

// TestOwnVsEffectiveFacets exercises the §3.16.6.4 overlay across a 3-level
// restriction chain: anyAtomicType -> primitive -> mid -> leaf. A more-derived
// same-kind facet masks the base's facet, and non-superseded facets survive.
func TestOwnVsEffectiveFacets(t *testing.T) {
	prim, err := newCheckedPrimitiveType(xsderr.Loc{}, QName{Local: "prim"},
		[]Facet{NewFacet(FacetWhiteSpace, []string{"collapse"}, true)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	mid, err := newCheckedSimpleType(xsderr.Loc{}, QName{Local: "mid"}, RestrictionDerivation{}, prim,
		[]Facet{
			NewFacet(FacetMinLength, []string{"1"}, false),
			NewFacet(FacetMaxLength, []string{"10"}, false),
		}, nil)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := newCheckedSimpleType(xsderr.Loc{}, QName{Local: "leaf"}, RestrictionDerivation{}, mid,
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
	eff := mustEffectiveFacets(leaf)
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
	anon, err := newCheckedSimpleType(xsderr.Loc{}, QName{}, RestrictionDerivation{}, leaf,
		[]Facet{NewFacet(FacetLength, []string{"3"}, false)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range mustEffectiveFacets(anon) {
		if f.Facet().Kind() != FacetLength {
			continue
		}
		if got := f.Declaring(); got != (QName{}) {
			t.Errorf("anonymous-declared length Declaring() = %v, want zero QName", got)
		}
	}

	// Anchors carry no facets.
	if mustEffectiveFacets(anySimpleType) != nil {
		t.Error("anySimpleType.EffectiveFacets() != nil")
	}
	if mustEffectiveFacets(anyAtomicType) != nil {
		t.Error("anyAtomicType.EffectiveFacets() != nil")
	}
}
