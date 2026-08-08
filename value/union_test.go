package value

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// unionMemberVal is a test-only atomic value TAGGED with the member type that
// produced it, so a dispatch test can assert WHICH member won — the whole point of
// dv_union's ·active member type· (§4.1.4 cl.2.3) — and not merely that the union
// accepted. It carries the §2.2.1/§2.2.2 relations enumeration matching needs;
// identity coincides with equality here (no NaN/signed-zero carve-out), the
// itemStub convention.
type unionMemberVal struct {
	member  string
	lexical string
}

func (v unionMemberVal) Eq(other Value) bool {
	o, ok := other.(unionMemberVal)
	return ok && o == v
}

func (v unionMemberVal) Identical(other Value) bool { return v.Eq(other) }

// memberBackend maps each member type's QName to a predicate over the ALREADY
// whiteSpace-normalized lexical (ValidateLexical normalizes per member before
// Parse), producing a unionMemberVal tagged with that type's local name. A type
// absent from the map is ungoverned, which is how the all-members-governed rule
// (unionGoverned) is exercised. It matches this package's mock idiom
// (emptyBackend, stubItemBackend) and keeps value's tests free of the
// builtin/strict import cycle.
type memberBackend map[xsd.QName]func(string) bool

func (mb memberBackend) Mapping(typ xsd.QName) (Mapping, bool) {
	accepts, mapped := mb[typ]
	if !mapped {
		return Mapping{}, false
	}
	return Mapping{Parse: func(lexical string, _ Context) (Value, error) {
		if !accepts(lexical) {
			return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
				"%s: %q is not in the lexical space", typ, lexical)
		}
		return unionMemberVal{member: typ.Local, lexical: lexical}, nil
	}}, true
}

// allDigits reports whether s is a non-empty run of ASCII digits — the "numeric"
// member's lexical space in these tests.
func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// unionType2 builds a union-variety *xsd.SimpleType over members, constructed
// directly from xs:anySimpleType and therefore facet-free (cos-st-restricts clause
// 3.2.1.2 forbids own facets on a constructed union — a union that carries facets
// must restrict a union base, which unionRestriction builds).
func unionType2(t *testing.T, local string, members ...*xsd.SimpleType) *xsd.SimpleType {
	t.Helper()
	u, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: local},
		xsd.NewUnion(members...), xsd.AnySimpleType(), nil, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(union %q): %v", local, err)
	}
	return u
}

// unionRestriction builds a union-variety restriction of base carrying own facets:
// the same members positionally (cos-st-restricts clause 3.2.2.3 pairs a
// restriction's members with the base's by position), so pattern and enumeration —
// the only constraining facets applicable to a union besides assertions
// (cos-applicable-facets §4.1.5) — can be exercised on a real component.
func unionRestriction(t *testing.T, local string, base *xsd.SimpleType, own []xsd.Facet) *xsd.SimpleType {
	t.Helper()
	members := base.Variety().(xsd.Union).Members()
	u, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: local},
		xsd.NewUnion(members...), base, own, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(union restriction %q): %v", local, err)
	}
	return u
}

// TestValidateLexicalUnionFirstMemberWins pins dv_union's member selection
// (§4.1.4 cl.2.3 with the ·active member type· definition): the FIRST member in
// {member type definitions} order that accepts the literal wins, and the value the
// union yields is that member's own value verbatim (V is a pass-through, not a
// union-shaped wrapper). The two unions differ ONLY in member order over a literal
// both members accept, so the tags they produce differ — a "any member accepts"
// implementation would pass either way.
func TestValidateLexicalUnionFirstMemberWins(t *testing.T) {
	num := primType(t, "numeric", "collapse")
	text := primType(t, "text", "preserve")
	b := memberBackend{
		num.Name():  allDigits,
		text.Name(): func(string) bool { return true },
	}
	numFirst := unionType2(t, "numFirst", num, text)
	textFirst := unionType2(t, "textFirst", text, num)

	for _, tc := range []struct {
		name    string
		st      *xsd.SimpleType
		lexical string
		want    unionMemberVal
	}{
		{"digits, numeric member first", numFirst, "7", unionMemberVal{member: "numeric", lexical: "7"}},
		{"digits, text member first", textFirst, "7", unionMemberVal{member: "text", lexical: "7"}},
		{"non-digits skips the numeric member", numFirst, "abc", unionMemberVal{member: "text", lexical: "abc"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, err := ValidateLexical(b, tc.st, tc.lexical, nil)
			if err != nil {
				t.Fatalf("ValidateLexical(%s, %q) = %v, want accept", tc.st.Name(), tc.lexical, err)
			}
			if v != Value(tc.want) {
				t.Errorf("ValidateLexical(%s, %q) = %#v, want %#v (active member type = first accepting member, §4.1.4 cl.2.3)",
					tc.st.Name(), tc.lexical, v, tc.want)
			}
		})
	}
}

// TestValidateLexicalUnionNestedRecursion pins the ·active basic member· descent:
// a {member type definitions} entry may itself be a union (§3.16.1
// std-member_type_definitions), and dispatch must recurse into ITS members rather
// than assume one flat level (PRINCIPLES 11: direct members, never a flattened
// membership). The value returned is produced by the BASIC (non-union) member at
// the bottom of that chain.
func TestValidateLexicalUnionNestedRecursion(t *testing.T) {
	num := primType(t, "numeric", "collapse")
	text := primType(t, "text", "preserve")
	b := memberBackend{
		num.Name():  allDigits,
		text.Name(): func(string) bool { return true },
	}
	inner := unionType2(t, "inner", text)
	outer := unionType2(t, "outer", num, inner)

	// "abc" is rejected by the numeric member, so the nested union is tried and its
	// own first member — the basic member "text" — decides.
	v, err := ValidateLexical(b, outer, "abc", nil)
	if err != nil {
		t.Fatalf("ValidateLexical(nested union, %q) = %v, want accept via the nested member", "abc", err)
	}
	if want := (unionMemberVal{member: "text", lexical: "abc"}); v != Value(want) {
		t.Errorf("ValidateLexical(nested union, %q) = %#v, want %#v (active basic member)", "abc", v, want)
	}

	// "7" is accepted by the DIRECT numeric member first, so the nested union is
	// never reached — order is preserved across the nesting.
	v, err = ValidateLexical(b, outer, "7", nil)
	if err != nil {
		t.Fatalf("ValidateLexical(nested union, %q) = %v, want accept", "7", err)
	}
	if want := (unionMemberVal{member: "numeric", lexical: "7"}); v != Value(want) {
		t.Errorf("ValidateLexical(nested union, %q) = %#v, want %#v (direct member wins over the nested union)", "7", v, want)
	}
}

// TestValidateLexicalUnionPatternUsesActiveMemberWhiteSpace pins the dv_vfacets
// note: the literal a union's OWN pattern facet is matched against (clause 1,
// cvc-pattern-valid §4.3.4.4) is the one normalized by the ·pre-lexical· facets
// "associated with B in clause 2.3" — the active basic member's whiteSpace — not
// the raw literal and not a union-level normalization, which does not exist
// (§4.1.5, §4.3.6). PRINCIPLES 11 states the same invariant.
//
// The raw literal "  7   7  " collapses to "7 7" through the collapse-normalized
// member, so the pattern spelling the COLLAPSED form accepts and the pattern
// spelling the RAW form rejects. A pattern check hoisted ahead of the dispatch
// over the raw literal would invert both verdicts.
func TestValidateLexicalUnionPatternUsesActiveMemberWhiteSpace(t *testing.T) {
	collapsed := primType(t, "collapsed", "collapse")
	b := memberBackend{collapsed.Name(): func(string) bool { return true }}
	base := unionType2(t, "wsBase", collapsed)

	const raw = "  7   7  "
	onCollapsed := unionRestriction(t, "onCollapsed", base,
		[]xsd.Facet{xsd.NewFacet(xsd.FacetPattern, []string{"7 7"}, false)})
	v, err := ValidateLexical(b, onCollapsed, raw, nil)
	if err != nil {
		t.Fatalf("ValidateLexical(union pattern %q, raw %q) = %v, want accept (pattern sees the member-collapsed literal)", "7 7", raw, err)
	}
	if want := (unionMemberVal{member: "collapsed", lexical: "7 7"}); v != Value(want) {
		t.Errorf("ValidateLexical(union pattern) = %#v, want %#v", v, want)
	}

	onRaw := unionRestriction(t, "onRaw", base,
		[]xsd.Facet{xsd.NewFacet(xsd.FacetPattern, []string{raw}, false)})
	_, err = ValidateLexical(b, onRaw, raw, nil)
	if err == nil {
		t.Fatalf("ValidateLexical(union pattern %q, raw %q) = nil, want the pattern to reject the RAW spelling", raw, raw)
	}
	if r, _ := xsderr.RuleOf(err); r != "cvc-pattern-valid" {
		t.Errorf("ValidateLexical(union raw-spelled pattern) charged %s, want cvc-pattern-valid (§4.3.4.4)", r)
	}
}

// TestValidateLexicalUnionEnumerationPostDispatch pins clause 3 (dv_vfacets): a
// union's own enumeration facet is checked against V — the value the active member
// produced — using V's own identity/equality relations (§2.2.1/§2.2.2,
// cvc-enumeration-valid §4.3.5.4). The facet's own {value} members are themselves
// dispatched through the union (src-enumeration-value §4.3.5.3, via unionMapping),
// so "7" as an enumeration member and "7" as an instance both resolve to the
// numeric member and compare equal, while a literal that dispatches to a different
// member never matches it.
func TestValidateLexicalUnionEnumerationPostDispatch(t *testing.T) {
	num := primType(t, "numeric", "collapse")
	text := primType(t, "text", "preserve")
	b := memberBackend{
		num.Name():  allDigits,
		text.Name(): func(string) bool { return true },
	}
	base := unionType2(t, "enumBase", num, text)
	enumerated := unionRestriction(t, "enumerated", base, []xsd.Facet{
		xsd.NewEnumerationFacet([]xsd.EnumerationMember{xsd.NewEnumerationMember("7", nil, nil)}),
	})

	v, err := ValidateLexical(b, enumerated, "7", nil)
	if err != nil {
		t.Fatalf("ValidateLexical(union enumeration, %q) = %v, want accept", "7", err)
	}
	if want := (unionMemberVal{member: "numeric", lexical: "7"}); v != Value(want) {
		t.Errorf("ValidateLexical(union enumeration, %q) = %#v, want %#v", "7", v, want)
	}

	// Both of these DISPATCH successfully (the text member accepts anything), so a
	// rejection here can only come from the union's own enumeration facet running
	// after the dispatch.
	for _, lexical := range []string{"8", "abc"} {
		_, err := ValidateLexical(b, enumerated, lexical, nil)
		if err == nil {
			t.Fatalf("ValidateLexical(union enumeration, %q) = nil, want an enumeration rejection", lexical)
		}
		if r, _ := xsderr.RuleOf(err); r != "cvc-enumeration-valid" {
			t.Errorf("ValidateLexical(union enumeration, %q) charged %s, want cvc-enumeration-valid (§4.3.5.4)", lexical, r)
		}
	}
}

// TestValidateLexicalUnionEmptyMembershipRejectsEverything pins the xs:error shape
// (Structures §3.16.7.3): the union with an empty {member type definitions} has an
// empty value space AND an empty lexical space, so every literal — the empty string
// included — is rejected. The dispatch loop reaches this with zero candidates and
// no special case, which is the property under test.
func TestValidateLexicalUnionEmptyMembershipRejectsEverything(t *testing.T) {
	empty := unionType2(t, "errorLike")
	for _, lexical := range []string{"", " ", "0", "anything"} {
		v, err := ValidateLexical(memberBackend{}, empty, lexical, nil)
		if err == nil {
			t.Fatalf("ValidateLexical(empty union, %q) = (%#v, nil), want a rejection (§3.16.7.3)", lexical, v)
		}
		if r, _ := xsderr.RuleOf(err); r != "cvc-datatype-valid" {
			t.Errorf("ValidateLexical(empty union, %q) charged %s, want cvc-datatype-valid (§4.1.4 cl.2.3)", lexical, r)
		}
	}
}

// TestGoverningMappingUnionRequiresEveryMember pins unionGoverned: a union is
// governed only when the backend governs EVERY member, because dv_union takes the
// FIRST member that accepts and an unmapped member is indistinguishable from one
// that rejects — a partially mapped union would silently hand back a later
// member's value. A fully mapped union resolves to a unionMapping whose Parse is
// the dispatch itself (the facet-{value} parsing seam).
func TestGoverningMappingUnionRequiresEveryMember(t *testing.T) {
	num := primType(t, "numeric", "collapse")
	text := primType(t, "text", "preserve")
	u := unionType2(t, "partly", num, text)

	partial := memberBackend{num.Name(): allDigits}
	if _, ok := governingMapping(partial, u); ok {
		t.Error("governingMapping(union with one unmapped member) ok = true, want false (an unmapped member is a BACKEND gap, not a verdict)")
	}

	full := memberBackend{
		num.Name():  allDigits,
		text.Name(): func(string) bool { return true },
	}
	m, ok := governingMapping(full, u)
	if !ok {
		t.Fatal("governingMapping(fully mapped union) ok = false, want a resolved unionMapping")
	}
	v, err := m.Parse("7", nil)
	if err != nil {
		t.Fatalf("unionMapping.Parse(%q): %v", "7", err)
	}
	if want := (unionMemberVal{member: "numeric", lexical: "7"}); v != Value(want) {
		t.Errorf("unionMapping.Parse(%q) = %#v, want %#v (the active member's own value)", "7", v, want)
	}
}

// TestDispatchUnionAbortsOnFacetPreconditionFault pins the one error class the member
// scan must NOT collect: a facet-precondition fault on member 0 (cos-applicable-facets
// §4.1.5, IsFacetPrecondition) aborts the dispatch instead of being folded into the
// rejection list.
//
// The union here is built so that folding produces a FALSE ACCEPT, not merely a
// misleading message: member 1 accepts every literal, so a scan that recorded member 0
// as "rejected" would report the union VALID with member 1's value as V — off a member
// dv_union (§4.1.4 cl.2.3) never entitled it to reach, since member 0 was never
// decided at all. That is the worst outcome available here, which is why the fault
// propagates and the union is decided by nobody.
func TestDispatchUnionAbortsOnFacetPreconditionFault(t *testing.T) {
	faulting := preconditionType(t, "unmeasurable")
	accepting := primType(t, "text", "preserve")
	b := plainBackend{faulting.Name(): true, accepting.Name(): true}
	u := unionType2(t, "faultingFirst", faulting, accepting)

	v, err := ValidateLexical(b, u, "ab", nil)
	if err == nil {
		t.Fatalf("ValidateLexical(union whose member 0 faults) = (%#v, nil): member 1 wrongly decided a union member 0 never decided", v)
	}
	if !IsFacetPrecondition(err) {
		t.Errorf("ValidateLexical(union whose member 0 faults): IsFacetPrecondition = false, want true — the fault was folded into a cvc-datatype-valid rejection: %v", err)
	}

	// The mutation guard: an ordinary member REJECTION is still the dispatch mechanism,
	// so a later member must still win. Were the abort widened to any member error,
	// this would fail.
	num := primType(t, "numeric", "collapse")
	text := primType(t, "text2", "preserve")
	mb := memberBackend{num.Name(): allDigits, text.Name(): func(string) bool { return true }}
	if _, err := ValidateLexical(mb, unionType2(t, "numFirst2", num, text), "abc", nil); err != nil {
		t.Errorf("ValidateLexical(union, literal only member 1 accepts) = %v, want accept via member 1", err)
	}
}

// TestValidateLexicalUnionMemberWithNoApplicableFacets pins the zero-mode guard in
// validateUnion's clause-1 stage. When the ·active basic member· is an ATOMIC type
// whose {primitive type definition} is absent, cos-applicable-facets (§4.1.5) makes NO
// facet applicable to it, so it resolves no whiteSpace mode and the union's own pattern
// stage must test the RAW literal rather than normalize with the zero mode.
//
// The state is reachable, not hypothetical: cos-st-restricts clause 3.1 rejects the two
// ·special· ANCHOR nodes as members by identity, so an absent-primitive atomic a caller
// builds itself passes that check and can be a member. Without the guard,
// normalizeWhiteSpace panics on the zero mode — the very panic class this cohort exists
// to remove.
func TestValidateLexicalUnionMemberWithNoApplicableFacets(t *testing.T) {
	member, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: "noPrimitive"},
		xsd.NewAtomic(nil), xsd.AnyAtomicType(), nil, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(atomic with absent {primitive type definition}): %v", err)
	}
	u := unionType2(t, "facetlessMember", member)

	const raw = "  x  "
	v, verr := ValidateLexical(plainBackend{member.Name(): true}, u, raw, nil)
	if verr != nil {
		t.Fatalf("ValidateLexical(union over a facet-less member) = %v, want accept", verr)
	}
	if v != Value(raw) {
		t.Errorf("ValidateLexical = %#v, want the RAW literal %q — no whiteSpace mode is in force, so nothing normalizes", v, raw)
	}
}
