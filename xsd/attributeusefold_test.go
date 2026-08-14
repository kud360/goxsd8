package xsd

import (
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
		uses, prohibited, nil, EmptyContent{}, nil, nil, nil)
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
			nil, []QName{uq("x")}, nil, EmptyContent{}, nil, nil, nil)
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
// in the chain, an extension that re-declares a name its base really does carry
// is still rejected. Without it, "accepts E" could be bought by disabling the
// clause-4 check altogether.
func TestCTPropsCorrectExtensionOverProhibitingRestriction(t *testing.T) {
	for _, tc := range []struct {
		name       string
		b          ComplexType
		wantReject bool
	}{
		{"B prohibits @x, so E may declare its own", dProhibiting(t, uq("B"), uq("A"), nil, []QName{uq("x")}), false},
		{"B inherits @x silently, so E may not re-declare it", dType(t, uq("B"), uq("A"), EmptyContent{}, nil, nil), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := dFinalize(t, func(b *SchemaBuilder) {
				b.AddType(dType(t, uq("A"), anyTypeName, EmptyContent{},
					[]AttributeUse{dAttr(t, uq("x"), uq("str"))}, nil))
				b.AddType(tc.b)
				b.AddType(xType(t, uq("E"), uq("B"), EmptyContent{},
					[]AttributeUse{dAttr(t, uq("x"), uq("str"))}, nil))
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
