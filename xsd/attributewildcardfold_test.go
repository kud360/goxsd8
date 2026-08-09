package xsd

import (
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// These tests are package-internal: foldAttributeWildcards runs inside
// SchemaBuilder.Finalize and is unexported (STYLE T5), so the fold is observed
// through the {attribute wildcard} the finalized *Schema hands back. The
// component builders come from complexderivation_test.go,
// complexextension_test.go and contentrestricts_test.go — one set of helpers, not
// four (STYLE T4).

// wNSA and wNSB are two namespace names the wildcards under test admit, distinct
// from uns (the namespace the type names live in) so that "admits urn:a" is never
// satisfied by accident.
var (
	wNSA = NamespaceName("urn:a")
	wNSB = NamespaceName("urn:b")
)

// wWild builds an {attribute wildcard} over the given constraint, as the pointer
// NewComplexType's Optional slot takes.
func wWild(t *testing.T, nc NamespaceConstraint, pc ProcessContents) *Wildcard {
	t.Helper()
	w, err := NewWildcard(xsderr.Loc{}, nc, pc, nil)
	if err != nil {
		t.Fatalf("NewWildcard: %v", err)
	}
	return &w
}

// wOf reads a finalized type's {attribute wildcard}.
func wOf(t *testing.T, s *Schema, name QName) (Wildcard, bool) {
	t.Helper()
	def, ok := s.Type(name)
	if !ok {
		t.Fatalf("type %s is not in the finalized schema", name)
	}
	c, ok := def.(ComplexType)
	if !ok {
		t.Fatalf("type %s is not a complex type definition", name)
	}
	return c.AttributeWildcard()
}

// TestAttributeWildcardFoldExtensionInheritsBase pins §3.4.2.5 clause 2.2.2.2: an
// extension that declares no <anyAttribute> of its own takes the ·base wildcard·
// whole — {process contents} included, since there is no ·complete wildcard· to
// take it from.
func TestAttributeWildcardFoldExtensionInheritsBase(t *testing.T) {
	s := xSchema(t, func(b *SchemaBuilder) {
		b.AddType(dType(t, uq("A"), anyTypeName, EmptyContent{}, nil,
			wWild(t, cNC(t, NamespaceConstraintEnumeration, []Namespace{wNSA}, nil, nil), ProcessSkip)))
		b.AddType(xType(t, uq("E"), uq("A"), EmptyContent{}, nil, nil))
	})
	w, ok := wOf(t, s, uq("E"))
	if !ok {
		t.Fatal("{attribute wildcard} of E is absent, want the base's (clause 2.2.2.2)")
	}
	if !w.NamespaceConstraint().AllowsNamespace(wNSA) || w.NamespaceConstraint().AllowsNamespace(wNSB) {
		t.Fatalf("{attribute wildcard} of E admits the wrong namespaces: %s", w.NamespaceConstraint().Variety())
	}
	if w.ProcessContents() != ProcessSkip {
		t.Fatalf("{process contents} of E's inherited wildcard = %s, want skip (the ·base wildcard· whole)", w.ProcessContents())
	}
}

// TestAttributeWildcardFoldExtensionUnionsWithBase pins clause 2.2.2.3: when both
// wildcards are present the result admits the union of the two namespace sets,
// and its {process contents} comes from the ·complete wildcard· — the extension's
// OWN declaration — never from the base's.
//
// The finalize run itself is half the assertion. E's own wildcard admits neither
// more nor less than A's, so before the fold cos-ct-extends clause 1.3 would have
// read E as failing to cover its base and rejected a valid schema; xSchema fails
// the test on any Finalize error.
func TestAttributeWildcardFoldExtensionUnionsWithBase(t *testing.T) {
	s := xSchema(t, func(b *SchemaBuilder) {
		b.AddType(dType(t, uq("A"), anyTypeName, EmptyContent{}, nil,
			wWild(t, cNC(t, NamespaceConstraintEnumeration, []Namespace{wNSA}, nil, nil), ProcessSkip)))
		b.AddType(xType(t, uq("E"), uq("A"), EmptyContent{}, nil,
			wWild(t, cNC(t, NamespaceConstraintEnumeration, []Namespace{wNSB}, nil, nil), ProcessStrict)))
	})
	w, ok := wOf(t, s, uq("E"))
	if !ok {
		t.Fatal("{attribute wildcard} of E is absent, want the union (clause 2.2.2.3)")
	}
	for _, ns := range []Namespace{wNSA, wNSB} {
		if !w.NamespaceConstraint().AllowsNamespace(ns) {
			t.Fatalf("the union does not admit %v, so cos-aw-union was not applied", ns)
		}
	}
	if w.NamespaceConstraint().AllowsNamespace(NamespaceName("urn:c")) {
		t.Fatal("the union admits a namespace neither operand did")
	}
	if w.ProcessContents() != ProcessStrict {
		t.Fatalf("{process contents} of the union = %s, want strict (the ·complete wildcard·'s, clause 2.2.2.3)", w.ProcessContents())
	}
}

// TestAttributeWildcardFoldRestrictionInheritsNothing pins clause 2.1: a
// restriction's {attribute wildcard} is its ·complete wildcard· alone, so a
// restriction of a base that HAS one and declares none itself stays ·absent·.
//
// This is the case that keeps the fold from collapsing every type onto
// ·xs:anyType·'s lax ##any wildcard: a restriction anywhere in the chain stops the
// inheritance, which is exactly why the fix could not be a chain walk at the read
// site.
func TestAttributeWildcardFoldRestrictionInheritsNothing(t *testing.T) {
	s := xSchema(t, func(b *SchemaBuilder) {
		b.AddType(dType(t, uq("A"), anyTypeName, EmptyContent{}, nil,
			wWild(t, cNC(t, NamespaceConstraintAny, nil, nil, nil), ProcessLax)))
		b.AddType(dType(t, uq("R"), uq("A"), EmptyContent{}, nil, nil))
	})
	if _, ok := wOf(t, s, uq("R")); ok {
		t.Fatal("{attribute wildcard} of R is present, but clause 2.1 inherits nothing into a restriction")
	}
}

// TestAttributeWildcardFoldExtensionOfExtension pins that clause 2.2.1.1 names the
// base's {attribute wildcard} PROPERTY — itself this rule's output — and not the
// base's <anyAttribute>, so the inheritance is transitive across a chain of
// extensions that restate nothing.
func TestAttributeWildcardFoldExtensionOfExtension(t *testing.T) {
	s := xSchema(t, func(b *SchemaBuilder) {
		b.AddType(dType(t, uq("A"), anyTypeName, EmptyContent{}, nil,
			wWild(t, cNC(t, NamespaceConstraintEnumeration, []Namespace{wNSA}, nil, nil), ProcessLax)))
		b.AddType(xType(t, uq("E1"), uq("A"), EmptyContent{}, nil, nil))
		b.AddType(xType(t, uq("E2"), uq("E1"), EmptyContent{}, nil, nil))
	})
	for _, name := range []QName{uq("E1"), uq("E2")} {
		w, ok := wOf(t, s, name)
		if !ok {
			t.Fatalf("{attribute wildcard} of %s is absent, want A's carried down the extension chain", name)
		}
		if !w.NamespaceConstraint().AllowsNamespace(wNSA) {
			t.Fatalf("{attribute wildcard} of %s does not admit A's namespace", name)
		}
	}
}

// TestDerivationOKRestrictionWildcardThroughExtension is this issue's acceptance
// shape and the reason the fold is not merely a tidying-up: A carries an
// <anyAttribute> that admits everything, B extends A and restates nothing, and C
// restricts B with a genuinely narrower wildcard. C's wildcard is a cos-ns-subset
// subset of what B really admits, so the schema is valid — but with {attribute
// wildcard} left as the producer maps it, B reads as admitting NOTHING and c-ran's
// cvc-complex-type clause 2.2 half rejects C for a name the base really does
// allow. That is a FALSE REJECT, the direction opposite to every fail-open gap
// around it.
//
// The second row is the control that keeps the first honest: with no wildcard
// anywhere up the chain, C's wildcard really is unadmitted and must still be
// charged. Without it, "accepts C" could be bought by not charging the clause at
// all.
func TestDerivationOKRestrictionWildcardThroughExtension(t *testing.T) {
	for _, tc := range []struct {
		name       string
		baseWild   *Wildcard
		wantReject bool
	}{
		{"a wildcard inherited through an extension covers the restriction's",
			wWild(t, cNC(t, NamespaceConstraintAny, nil, nil, nil), ProcessLax), false},
		{"with no wildcard up the chain the restriction's is uncovered",
			nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := dFinalize(t, func(b *SchemaBuilder) {
				b.AddType(dType(t, uq("A"), anyTypeName, EmptyContent{}, nil, tc.baseWild))
				b.AddType(xType(t, uq("B"), uq("A"), EmptyContent{}, nil, nil))
				b.AddType(dType(t, uq("C"), uq("B"), EmptyContent{}, nil,
					wWild(t, cNC(t, NamespaceConstraintEnumeration, []Namespace{wNSA}, nil, nil), ProcessLax)))
			})
			if !tc.wantReject {
				if err != nil {
					t.Fatalf("a restriction of an extension that inherited the admitting wildcard was rejected: %v", err)
				}
				return
			}
			expectRule(t, err, ruleDerivationOKRestriction)
		})
	}
}

// TestDerivationOKRestrictionAttributeWildcardSubset pins the cvc-complex-type
// clause 2.2 (c-avaw) half of derivation-ok-restriction clause 3 over the four
// shapes the comparison can take. The two accepting rows are as load-bearing as
// the two rejecting ones: a restriction may narrow the base's wildcard or drop it
// entirely, and charging either would false-reject an ordinary schema.
func TestDerivationOKRestrictionAttributeWildcardSubset(t *testing.T) {
	anyNC := cNC(t, NamespaceConstraintAny, nil, nil, nil)
	oneNC := cNC(t, NamespaceConstraintEnumeration, []Namespace{wNSA}, nil, nil)
	for _, tc := range []struct {
		name              string
		baseWild, ownWild *Wildcard
		wantReject        bool
	}{
		{"narrowing the base's wildcard is a valid restriction",
			wWild(t, anyNC, ProcessLax), wWild(t, oneNC, ProcessLax), false},
		{"dropping the base's wildcard is a valid restriction",
			wWild(t, anyNC, ProcessLax), nil, false},
		{"widening the base's wildcard is not",
			wWild(t, oneNC, ProcessLax), wWild(t, anyNC, ProcessLax), true},
		{"declaring a wildcard the base does not have at all is not",
			nil, wWild(t, oneNC, ProcessLax), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := dFinalize(t, func(b *SchemaBuilder) {
				b.AddType(dType(t, uq("base"), anyTypeName, EmptyContent{}, nil, tc.baseWild))
				b.AddType(dType(t, uq("derived"), uq("base"), EmptyContent{}, nil, tc.ownWild))
			})
			if !tc.wantReject {
				if err != nil {
					t.Fatalf("a valid attribute-wildcard restriction was rejected: %v", err)
				}
				return
			}
			expectRule(t, err, ruleDerivationOKRestriction)
		})
	}
}

// TestDerivationOKRestrictionAttributeWildcardSharedUse pins the clause-2.1
// pre-emption inside the c-avaw half: cvc-complex-type clause 2.2 is reached only
// "otherwise", so a name BOTH types govern with an {attribute use} is assessed by
// clause 2.1 on both sides and B's notQName for it may not be charged against T's
// wildcard (#430). The base is the counter-example verbatim — <xs:attribute
// name="foo"/> beside <anyAttribute namespace="##any" notQName="foo"/> — and the
// restriction opens its own wildcard to ##any.
//
// The first row is the shape that inherits the foo use through §3.4.2.4 clause
// 3.2 without naming it, the second re-declares it (clause 3.2.1), and the two
// must agree: whether the shared use is inherited or restated changes nothing
// about which clause assesses an item named foo.
//
// The third row is the control that keeps the exemption from degenerating into
// "condition 1 of cos-ns-subset is off". A notQName naming something NEITHER type
// holds a use for is still a name only clause 2.2 can admit, so widening past it
// is still a real c-ran violation.
func TestDerivationOKRestrictionAttributeWildcardSharedUse(t *testing.T) {
	for _, tc := range []struct {
		name        string
		disallowed  QName
		derivedUses []AttributeUse
		wantReject  bool
	}{
		{"a name both types govern by an inherited {attribute use} is not charged",
			uq("foo"), nil, false},
		{"nor is one the restriction re-declares for itself",
			uq("foo"), []AttributeUse{dAttr(t, uq("foo"), uq("str"))}, false},
		{"a name no {attribute use} governs is still charged",
			uq("bar"), nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := dFinalize(t, func(b *SchemaBuilder) {
				b.AddType(dType(t, uq("base"), anyTypeName, EmptyContent{},
					[]AttributeUse{dAttr(t, uq("foo"), uq("str"))},
					wWild(t, cNC(t, NamespaceConstraintAny, nil, []QName{tc.disallowed}, nil), ProcessLax)))
				b.AddType(dType(t, uq("derived"), uq("base"), EmptyContent{}, tc.derivedUses,
					wWild(t, cNC(t, NamespaceConstraintAny, nil, nil, nil), ProcessLax)))
			})
			if !tc.wantReject {
				if err != nil {
					t.Fatalf("a restriction whose base disallows by name an attribute both types declare was rejected: %v", err)
				}
				return
			}
			expectRule(t, err, ruleDerivationOKRestriction)
		})
	}
}

// TestDerivationOKRestrictionAttributeWildcardUnsharedUse pins the asymmetry that
// makes the exempt set an INTERSECTION: a use T holds and B does NOT exempts
// nothing, because c-ran clause 3 still asks an item carrying that name to
// satisfy cvc-complex-type clause 2 with respect to B, which — holding no use for
// it — can only do so through clause 2.2.
//
// It calls the predicate directly rather than through Finalize because Finalize
// charges this shape earlier: checkRestrictionAttributes walks T's uses first and
// attributeDefaultBinding already refuses a name B neither declares nor admits.
// Routing through Finalize would therefore pass on the strength of a different
// check and say nothing about this one.
func TestDerivationOKRestrictionAttributeWildcardUnsharedUse(t *testing.T) {
	for _, tc := range []struct {
		name       string
		baseUses   []AttributeUse
		wantReject bool
	}{
		{"a use only the restriction holds exempts nothing",
			nil, true},
		{"the same use held by both sides does",
			[]AttributeUse{dAttr(t, uq("bar"), uq("str"))}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := dType(t, uq("base"), anyTypeName, EmptyContent{}, tc.baseUses,
				wWild(t, cNC(t, NamespaceConstraintAny, nil, []QName{uq("bar")}, nil), ProcessLax))
			derived := dType(t, uq("derived"), uq("base"), EmptyContent{},
				[]AttributeUse{dAttr(t, uq("bar"), uq("str"))},
				wWild(t, cNC(t, NamespaceConstraintAny, nil, nil, nil), ProcessLax))
			err := checkRestrictionAttributeWildcard(derived, base)
			if !tc.wantReject {
				if err != nil {
					t.Fatalf("c-avaw charged a name both types govern with an {attribute use}: %v", err)
				}
				return
			}
			expectRule(t, err, ruleDerivationOKRestriction)
		})
	}
}

// TestCosCTExtendsClause13 pins cos-ct-extends clause 1.3 — B's {attribute
// wildcard}.{namespace constraint} is a subset of T's — by calling the predicate
// directly rather than through Finalize, because Finalize CANNOT reach its two
// rejections: §3.4.2.5 clause 2.2 makes an extension's {attribute wildcard} the
// union of its own with the base's (attributewildcardfold.go), and a union is a
// superset of both operands, so a folded extension satisfies 1.3 by construction.
// The rejections remain live for a component assembled programmatically past the
// fold, which is the population the spec wrote the clause for. This file's
// neighbours state the same fact about their own unreachable branches rather than
// leaving a reader to rediscover it (see checkExtensionAttributeUses).
//
// TestAttributeWildcardFoldExtensionUnionsWithBase is the other half of the pair:
// it runs the shape that WOULD violate 1.3 unfolded through Finalize and requires
// it to be accepted.
func TestCosCTExtendsClause13(t *testing.T) {
	anyW := wWild(t, cNC(t, NamespaceConstraintAny, nil, nil, nil), ProcessLax)
	oneW := wWild(t, cNC(t, NamespaceConstraintEnumeration, []Namespace{wNSA}, nil, nil), ProcessLax)
	for _, tc := range []struct {
		name              string
		baseWild, ownWild *Wildcard
		wantReject        bool
	}{
		{"a base with no {attribute wildcard} discharges the clause", nil, nil, false},
		{"an extension covering the base's wildcard is valid", oneW, anyW, false},
		{"an extension with no wildcard where the base has one is not", oneW, nil, true},
		{"an extension whose wildcard admits fewer names than the base's is not", anyW, oneW, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := dType(t, uq("base"), anyTypeName, EmptyContent{}, nil, tc.baseWild)
			ext := xType(t, uq("derived"), uq("base"), EmptyContent{}, nil, tc.ownWild)
			err := checkExtensionAttributeWildcard(ext, base)
			if !tc.wantReject {
				if err != nil {
					t.Fatalf("clause 1.3 charged a valid extension: %v", err)
				}
				return
			}
			expectRule(t, err, ruleCosCTExtends)
		})
	}
}
