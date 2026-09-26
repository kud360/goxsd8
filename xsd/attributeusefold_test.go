package xsd

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// These tests are package-internal: foldAttributeUses runs inside
// SchemaBuilder.Finalize and is unexported (STYLE T5), so the fold is observed
// through the {attribute uses} the finalized *Schema hands back. The component
// builders come from complexderivation_test.go and complexextension_test.go —
// one set of helpers, not three (STYLE T4).

// fUses is the expanded names of a finalized type's {attribute uses}, in the
// order the property holds them. Order is asserted, not just membership: the fold
// is required to be deterministic and document-ordered (STYLE D2), so a test that
// compared sets would not notice a fold that ordered by map iteration.
func fUses(t *testing.T, s *Schema, name QName) []string {
	t.Helper()
	def, ok := s.Type(name)
	if !ok {
		t.Fatalf("type %s is not in the finalized schema", name)
	}
	c, ok := def.(ComplexType)
	if !ok {
		t.Fatalf("type %s is not a complex type definition", name)
	}
	var names []string
	for _, u := range c.AttributeUses() {
		names = append(names, u.DeclarationName().Local)
	}
	return names
}

// fEqual compares two name lists elementwise.
func fEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestAttributeUsesFoldRestriction pins §3.4.2.4 clause 3.2 over a three-level
// chain: a restriction inherits its base's {attribute uses} EXCEPT a name it
// declares itself (clause 3.2.1), and the fold is transitive, so a use declared
// two levels up reaches the bottom of the chain.
//
// The order is part of the assertion: own uses first, in document order, then the
// base's already-folded set in its own order.
func TestAttributeUsesFoldRestriction(t *testing.T) {
	s := xSchema(t, func(b *SchemaBuilder) {
		b.AddType(dPrimitive(t, uq("str")))
		b.AddType(dType(t, uq("A"), anyTypeName, EmptyContent{},
			[]AttributeUse{dAttr(t, uq("x"), uq("str")), dAttr(t, uq("y"), uq("str"))}, nil))
		b.AddType(dType(t, uq("B"), uq("A"), EmptyContent{}, nil, nil))
		b.AddType(dType(t, uq("C"), uq("B"), EmptyContent{},
			[]AttributeUse{dAttr(t, uq("x"), uq("str"))}, nil))
	})
	for _, tc := range []struct {
		name QName
		want []string
	}{
		{uq("A"), []string{"x", "y"}},
		{uq("B"), []string{"x", "y"}}, // clause 3.2, nothing of its own
		{uq("C"), []string{"x", "y"}}, // clause 3.2.1 drops the inherited x
	} {
		t.Run(tc.name.Local, func(t *testing.T) {
			if got := fUses(t, s, tc.name); !fEqual(got, tc.want) {
				t.Fatalf("{attribute uses} of %s = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

// TestAttributeUsesFoldRestrictionKeepsOwnUse pins the half of clause 3.2.1 a
// name list cannot see: when the restriction re-declares an inherited name, the
// member that survives is the RESTRICTION's use, not the base's. The base's use
// is optional and the restriction's required, so the two are distinguishable.
func TestAttributeUsesFoldRestrictionKeepsOwnUse(t *testing.T) {
	s := xSchema(t, func(b *SchemaBuilder) {
		b.AddType(dPrimitive(t, uq("str")))
		b.AddType(dType(t, uq("A"), anyTypeName, EmptyContent{},
			[]AttributeUse{dAttr(t, uq("x"), uq("str"))}, nil))
		b.AddType(dType(t, uq("B"), uq("A"), EmptyContent{},
			[]AttributeUse{dAttrUse(t, uq("x"), uq("str"), true, nil)}, nil))
	})
	def, _ := s.Type(uq("B"))
	uses := def.(ComplexType).AttributeUses()
	if len(uses) != 1 {
		t.Fatalf("{attribute uses} of B = %d members, want 1 (clause 3.2.1 excludes the inherited x)", len(uses))
	}
	if !uses[0].Required() {
		t.Fatal("clause 3.2.1 kept the base's optional use of x instead of the restriction's required one")
	}
}

// TestAttributeUsesFoldExtension pins clause 3.1: an extension inherits its
// base's uses UNCONDITIONALLY, appended after its own.
func TestAttributeUsesFoldExtension(t *testing.T) {
	s := xSchema(t, func(b *SchemaBuilder) {
		b.AddType(dPrimitive(t, uq("str")))
		b.AddType(dType(t, uq("A"), anyTypeName, EmptyContent{},
			[]AttributeUse{dAttr(t, uq("x"), uq("str"))}, nil))
		b.AddType(xType(t, uq("E"), uq("A"), EmptyContent{},
			[]AttributeUse{dAttr(t, uq("e"), uq("str"))}, nil))
	})
	want := []string{"e", "x"}
	if got := fUses(t, s, uq("E")); !fEqual(got, want) {
		t.Fatalf("{attribute uses} of E = %v, want %v", got, want)
	}
}

// TestAttributeUsesFoldSimpleBase pins clause 3.3: a base that is not a complex
// type definition contributes nothing, so a simple-content extension keeps its
// own uses alone. It also pins the ·xs:anyType· termination — A's base is
// xs:anyType, whose §3.4.7 {attribute uses} is empty, and the fold neither loops
// on that self-derivation nor invents a member for it.
func TestAttributeUsesFoldSimpleBase(t *testing.T) {
	s := xSchema(t, func(b *SchemaBuilder) {
		str := dPrimitive(t, uq("str"))
		b.AddType(str)
		b.AddType(xType(t, uq("S"), uq("str"), SimpleContent{SimpleType: str},
			[]AttributeUse{dAttr(t, uq("u"), uq("str"))}, nil))
		b.AddType(dType(t, uq("A"), anyTypeName, EmptyContent{}, nil, nil))
	})
	if got := fUses(t, s, uq("S")); !fEqual(got, []string{"u"}) {
		t.Fatalf("{attribute uses} of S = %v, want [u] (clause 3.3 inherits nothing from a simple base)", got)
	}
	if got := fUses(t, s, uq("A")); got != nil {
		t.Fatalf("{attribute uses} of A = %v, want empty (xs:anyType has none)", got)
	}
}

// TestDerivationOKRestrictionRequiredInheritedTwoLevels is the charge the fold
// makes reachable, and the reason the fold is not merely a tidying-up: A requires
// @x, B restricts A and re-declares nothing, C restricts B and relaxes @x to
// optional. c-ran's cvc-complex-type clause 3 half compares C against B, and @x
// is B's only by §3.4.2.4 clause 3 — with {attribute uses} left as the producer
// mapped it, B carries no use at all and C's relaxation goes uncharged.
//
// The control row keeps @x required in C, which must stay valid: the same fold
// that finds the required use must not turn an ordinary re-declaration into a
// rejection.
func TestDerivationOKRestrictionRequiredInheritedTwoLevels(t *testing.T) {
	for _, tc := range []struct {
		name     string
		use      AttributeUse
		wantRule bool
	}{
		{"relaxing an inherited required attribute to optional is charged", dAttr(t, uq("x"), uq("str")), true},
		{"keeping it required is valid", dAttrUse(t, uq("x"), uq("str"), true, nil), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := dFinalize(t, func(b *SchemaBuilder) {
				b.AddType(dType(t, uq("A"), anyTypeName, EmptyContent{},
					[]AttributeUse{dAttrUse(t, uq("x"), uq("str"), true, nil)}, nil))
				b.AddType(dType(t, uq("B"), uq("A"), EmptyContent{}, nil, nil))
				b.AddType(dType(t, uq("C"), uq("B"), EmptyContent{}, []AttributeUse{tc.use}, nil))
			})
			if !tc.wantRule {
				if err != nil {
					t.Fatalf("a restriction that keeps the inherited attribute required was rejected: %v", err)
				}
				return
			}
			expectRule(t, err, ruleDerivationOKRestriction)
		})
	}
}

// dProhibiting builds a complex type restricting base which declares uses of its
// own AND gives the expanded names in prohibited use="prohibited" (§3.4.2.4
// clause 3.2.2). It is the one shape dType cannot express, so it lives here
// rather than beside dType: only the fold reads that slot.
func dProhibiting(t *testing.T, name, base QName, uses []AttributeUse, prohibited []QName) ComplexType {
	t.Helper()
	ct, err := NewComplexType(xsderr.Loc{}, name, base, nil, DerivationRestriction, false,
		attributeUseMembers(uses), prohibited, nil, EmptyContent{}, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType(%s): %v", name, err)
	}
	return ct
}

// TestAttributeUsesFoldProhibited pins §3.4.2.4 clause 3.2.2: a restriction that
// gives an <attribute> use="prohibited" does NOT inherit the same-named use from
// its base, and the exclusion stops there — it is a fact about B's own source, so
// a type restricting B further inherits whatever B ended up with and nothing is
// walked up the chain.
func TestAttributeUsesFoldProhibited(t *testing.T) {
	s := xSchema(t, func(b *SchemaBuilder) {
		b.AddType(dPrimitive(t, uq("str")))
		b.AddType(dType(t, uq("A"), anyTypeName, EmptyContent{},
			[]AttributeUse{dAttr(t, uq("x"), uq("str")), dAttr(t, uq("y"), uq("str"))}, nil))
		b.AddType(dProhibiting(t, uq("B"), uq("A"), nil, []QName{uq("x")}))
		b.AddType(dType(t, uq("C"), uq("B"), EmptyContent{}, nil, nil))
	})
	for _, tc := range []struct {
		name QName
		want []string
	}{
		{uq("A"), []string{"x", "y"}},
		{uq("B"), []string{"y"}}, // clause 3.2.2 blocks the inherited x
		{uq("C"), []string{"y"}}, // and C inherits B's set, which no longer has x
	} {
		t.Run(tc.name.Local, func(t *testing.T) {
			if got := fUses(t, s, tc.name); !fEqual(got, tc.want) {
				t.Fatalf("{attribute uses} of %s = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

// TestAttributeUsesFoldProhibitedIgnoredOnExtension pins the other half of the
// §3.4.2.4 Note: use="prohibited" is "pointless, though not an error" outside a
// restriction, and the <attribute> "is simply ignored". An extension therefore
// inherits the name regardless (clause 3.1 is unconditional).
func TestAttributeUsesFoldProhibitedIgnoredOnExtension(t *testing.T) {
	s := xSchema(t, func(b *SchemaBuilder) {
		b.AddType(dPrimitive(t, uq("str")))
		b.AddType(dType(t, uq("A"), anyTypeName, EmptyContent{},
			[]AttributeUse{dAttr(t, uq("x"), uq("str"))}, nil))
		ext, err := NewComplexType(xsderr.Loc{}, uq("E"), uq("A"), nil, DerivationExtension, false,
			nil, []QName{uq("x")}, nil, EmptyContent{}, nil, nil)
		if err != nil {
			t.Fatalf("NewComplexType(E): %v", err)
		}
		b.AddType(ext)
	})
	if got := fUses(t, s, uq("E")); !fEqual(got, []string{"x"}) {
		t.Fatalf("{attribute uses} of E = %v, want [x] (prohibited is ignored on an extension)", got)
	}
}

// TestCTPropsCorrectExtensionOverProhibitingRestriction is the false reject
// clause 3.2.2 exists to prevent, and the regression #401's first attempt shipped:
// A declares @x, B restricts A prohibiting @x, E extends B declaring its OWN @x.
// B.{attribute uses} is empty, so E holds exactly one use named x and the schema
// is valid. With clause 3.2.2 unapplied, B still carries A's x, E's fold appends
// its own on top, and ct-props-correct clause 4 rejects a duplicate the source
// never wrote.
//
// The second row is the control that keeps the first honest: with no prohibition
// in the chain, an extension that re-declares a name its base really does carry,
// with a use that is NOT identical to the inherited one, is still rejected.
// Without it, "accepts E" could be bought by disabling the clause-4 check
// altogether. Its @x is {required} where A's is optional, which is the whole of
// the difference — clause 3.1 collapses a re-declaration only when every property
// matches recursively (#1082, TestExtensionFoldsOneMemberForAnIdenticalUse).
func TestCTPropsCorrectExtensionOverProhibitingRestriction(t *testing.T) {
	for _, tc := range []struct {
		name       string
		b          ComplexType
		eUse       AttributeUse
		wantReject bool
	}{
		{"B prohibits @x, so E may declare its own", dProhibiting(t, uq("B"), uq("A"), nil, []QName{uq("x")}),
			dAttr(t, uq("x"), uq("str")), false},
		{"B inherits @x silently, so E may not re-declare it differently", dType(t, uq("B"), uq("A"), EmptyContent{}, nil, nil),
			dAttrUse(t, uq("x"), uq("str"), true, nil), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := dFinalize(t, func(b *SchemaBuilder) {
				b.AddType(dType(t, uq("A"), anyTypeName, EmptyContent{},
					[]AttributeUse{dAttr(t, uq("x"), uq("str"))}, nil))
				b.AddType(tc.b)
				b.AddType(xType(t, uq("E"), uq("B"), EmptyContent{},
					[]AttributeUse{tc.eUse}, nil))
			})
			if !tc.wantReject {
				if err != nil {
					t.Fatalf("an extension re-declaring a name its base prohibited was rejected: %v", err)
				}
				return
			}
			expectRule(t, err, ruleCTPropsCorrect)
		})
	}
}

// TestExtensionFoldsOneMemberForAnIdenticalUse pins §3.4.2.4's "union of SETS":
// a member reached by clause 1 or 2 AND by clause 3.1 from a base that already
// carries it is ONE member of the union, so ct-props-correct clause 4 — "no two
// DISTINCT members" — has nothing to charge (#1082).
//
// The two E types differ in ONE property of their own @x and in nothing else.
// Identical to A's, it collapses and E holds a single use for x; {required} where
// A's is optional, it is a second member and the clause charges. A fold that
// collapsed by expanded name instead would have to accept the second one too.
//
// That rejection is what makes the acceptance worth something: the dedup is keyed
// on cos-ct-extends clause 1.2's recursive property identity
// (attributeUsesIdentical), never on the name, so it cannot swallow the
// re-declaration clause 4 exists to catch.
func TestExtensionFoldsOneMemberForAnIdenticalUse(t *testing.T) {
	types := func(eX AttributeUse) func(*SchemaBuilder) {
		return func(b *SchemaBuilder) {
			b.AddType(dType(t, uq("A"), anyTypeName, EmptyContent{},
				[]AttributeUse{dAttr(t, uq("x"), uq("str"))}, nil))
			b.AddType(xType(t, uq("E"), uq("A"), EmptyContent{},
				[]AttributeUse{eX, dAttr(t, uq("y"), uq("str"))}, nil))
		}
	}
	identical := types(dAttr(t, uq("x"), uq("str")))
	if err := dFinalize(t, identical); err != nil {
		t.Fatalf("an extension whose own @x is identical to the inherited one was rejected: %v", err)
	}
	s := xSchema(t, func(b *SchemaBuilder) {
		b.AddType(dPrimitive(t, uq("str")))
		identical(b)
	})
	if got := fUses(t, s, uq("E")); !fEqual(got, []string{"x", "y"}) {
		t.Fatalf("{attribute uses} of E = %v, want [x y] — the union holds the twice-reached member once", got)
	}
	expectRule(t, dFinalize(t, types(dAttrUse(t, uq("x"), uq("str"), true, nil))), ruleCTPropsCorrect)
}

// TestDerivationOKRestrictionProhibitedRequired pins the charge clause 3.2.2
// makes reachable: A requires @x and B restricts A prohibiting it, so an element
// valid against B omits @x and is invalid against A. c-ran's cvc-complex-type
// clause 3 half now finds no member for x in B and charges it — the "base-required
// name with NO member in T" branch that could not fire while the clause was a gap.
func TestDerivationOKRestrictionProhibitedRequired(t *testing.T) {
	err := dFinalize(t, func(b *SchemaBuilder) {
		b.AddType(dType(t, uq("A"), anyTypeName, EmptyContent{},
			[]AttributeUse{dAttrUse(t, uq("x"), uq("str"), true, nil)}, nil))
		b.AddType(dProhibiting(t, uq("B"), uq("A"), nil, []QName{uq("x")}))
	})
	expectRule(t, err, ruleDerivationOKRestriction)
}

// oOverReportChain is the ONE chain that makes extensionStepAttributeUses' exact
// per-step recovery observable end to end (#1102), five derivation steps of
// alternating method under an ancestor whose ##any {attribute wildcard} is what
// admits the new name a restriction step declares:
//
//	qA(anyAttribute) ←ext qE1 ←restr qR(@r) ←ext qE2 ←restr qR2(prohibits @r) ←ext qT
//
// Every step's OWN §3.4.2.4 clause 1-and-2 contribution is fixed by
// construction: qE1 and qE2 declare nothing at all, so @r enters the chain only
// through qR, a RESTRICTION step, and leaves it again at qR2 through clause
// 3.2.2. The intermediate cos-ct-extends clause 1.5's Note prescribes — the
// extension steps re-ordered first and collapsed — therefore carries @r nowhere,
// whatever qT declares. The NAME records what the chain was built to expose:
// while the step answered its whole folded set, qE2's answer carried @r and M
// bound the name to qR's use.
//
// declared is qR's use for @r, the one no extension step's own contribution
// holds. own is what qT declares for itself, which may be empty.
func oOverReportChain(t *testing.T, declared AttributeUse, own []AttributeUse) func(*SchemaBuilder) {
	t.Helper()
	return func(b *SchemaBuilder) {
		w := dWildcard(t)
		b.AddType(dType(t, uq("qA"), anyTypeName, EmptyContent{}, nil, &w))
		b.AddType(xType(t, uq("qE1"), uq("qA"), EmptyContent{}, nil, nil))
		b.AddType(dType(t, uq("qR"), uq("qE1"), EmptyContent{}, []AttributeUse{declared}, nil))
		b.AddType(xType(t, uq("qE2"), uq("qR"), EmptyContent{}, nil, nil))
		b.AddType(dProhibiting(t, uq("qR2"), uq("qE2"), nil, []QName{uq("r")}))
		b.AddType(xType(t, uq("qT"), uq("qR2"), EmptyContent{}, own, nil))
	}
}

// oInheritableAttr builds an optional attribute use of the given {inheritable},
// which dAttrUse fixes at false. It is the ONE property the fixtures below vary:
// key-ldtype reads {type definition}s alone, so an {inheritable} mismatch is
// charged by loc-testSubP clause 5.3 and by nothing else in the chain.
func oInheritableAttr(t *testing.T, inheritable bool) AttributeUse {
	t.Helper()
	decl, err := NewAttributeDeclaration(xsderr.Loc{}, uq("r"), TypeDefinitionRef{Name: uq("str")}, aLocalScope(t), nil, false)
	if err != nil {
		t.Fatalf("NewAttributeDeclaration(r): %v", err)
	}
	u, err := NewAttributeUse(xsderr.Loc{}, false, LocalAttributeDeclaration{Declaration: decl}, nil, &inheritable)
	if err != nil {
		t.Fatalf("NewAttributeUse(r): %v", err)
	}
	return u
}

// TestCollapsedIntermediateOverReportFalseReject pins that the collapsed
// intermediate carries each extension step's OWN uses and no member of any
// step's base (#1102), as the valid schema this tree once rejected and now
// accepts.
//
// qE2's own contribution is empty and extensionStepAttributeUses answers it
// exactly — the clause-1-and-2 value retained past the fold (ownAttributeUses) —
// so M carries @r nowhere, and derivation-ok-restriction clause 3's
// ·subsumption· half (checkAttributeRestriction, attributerestriction.go) binds
// the name through qA's ##any wildcard, which is what the TRUE intermediate
// binds it to. A lax wildcard binding ·subsumes· any use binding (loc-testSubP
// clause 2), so both rows are valid schemas and both are accepted.
//
// The rows differ in {inheritable} alone. That is the discriminator because it
// isolates this arm: a {type definition} mismatch would be charged by
// cos-ct-extends clause 1.6 as well, whose key-ldtype case 3 reads @r's type
// straight off qE2 through the prohibiting qR2 — so a re-typing fixture is
// rejected whatever the collapse answers and pins nothing. loc-testSubP clause
// 5.3 demands equal {inheritable} and key-ldtype cannot see the property at all,
// which is what makes the second row the one that MOVED: while the step answered
// its whole folded set, M bound @r to qR's use and clause 5.3 charged the
// mismatch.
func TestCollapsedIntermediateOverReportFalseReject(t *testing.T) {
	for _, tc := range []struct {
		name        string
		inheritable bool
	}{
		{"qT matches qR's use for @r", true},
		{"qT differs from it in {inheritable} alone", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := dFinalize(t, oOverReportChain(t, oInheritableAttr(t, true),
				[]AttributeUse{oInheritableAttr(t, tc.inheritable)}))
			if err != nil {
				t.Fatalf("qT declaring @r over an intermediate that carries no use for it was rejected: %v", err)
			}
		})
	}
}

// TestCollapsedIntermediateOverReportRequiredIsOpen pins the two verdicts the
// oOverReportChain fixture produces now that the collapse carries each extension
// step's own uses exactly (#1102). The NAME records what the fixture was built to
// expose: while the step answered its whole folded set, M carried a use for @r
// and this test pinned that reader's direction as FAIL-OPEN.
//
//   - a required @r prohibited downstream is charged at the PROHIBITION, against
//     qE2, by derivation-ok-restriction clause 3 — the schema is invalid for a
//     reason of its own and never reaches clause 1.5's opinion of it. That charge
//     reads the two FOLDED sets, which the recovery does not touch;
//   - an optional @r is the one the prohibition may legally remove, and M carries
//     NO use for the name: @r entered the chain at qR, a RESTRICTION step the
//     re-ordering drops, so no extension step's own clause-1-and-2 contribution
//     ever held it. checkAttributeRestrictionRequired (attributerestriction.go)
//     has nothing to charge because M has no member, not because its charge is
//     unreachable.
func TestCollapsedIntermediateOverReportRequiredIsOpen(t *testing.T) {
	err := dFinalize(t, oOverReportChain(t, dAttrUse(t, uq("r"), uq("str"), true, nil), nil))
	expectRule(t, err, ruleDerivationOKRestriction)
	const charged = "complex type {urn:upa}qR2 restricts {urn:upa}qE2 but its {attribute uses} carry no use for attribute {urn:upa}r"
	if !strings.HasPrefix(err.Error(), "?: [derivation-ok-restriction] "+charged) {
		t.Fatalf("the required @r is charged as %q, want it charged against qE2 at the prohibiting step qR2", err)
	}
	s := xSchema(t, func(b *SchemaBuilder) {
		b.AddType(dPrimitive(t, uq("str")))
		oOverReportChain(t, dAttr(t, uq("r"), uq("str")), nil)(b)
	})
	def, _ := s.Type(uq("qT"))
	if got := len(def.(ComplexType).AttributeUses()); got != 0 {
		t.Fatalf("qT carries %d {attribute uses}, want none — clause 3.2.2 removed @r at qR2", got)
	}
	m, ok, err := s.collapsedExtension(def.(ComplexType))
	if err != nil || !ok {
		t.Fatalf("collapsedExtension(qT) = (ok=%t, err=%v), want a synthesized intermediate", ok, err)
	}
	if uses := m.AttributeUses(); len(uses) != 0 {
		t.Fatalf("M's {attribute uses} = %v, want none — @r entered the chain at a restriction step, which the re-ordering drops", uses)
	}
}
