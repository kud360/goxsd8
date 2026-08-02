package xsd

import (
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// These tests are package-internal: checkComplexDerivations runs inside
// SchemaBuilder.Finalize and is unexported (STYLE T5), so the assertions are made
// on the error Finalize returns. The component builders come from
// particleattribution_test.go and effectivetotalrange_test.go — one set of
// helpers, not three (STYLE T4).

// dAnyType is the ur-type xs:anyType (§3.4.7) as the producer seeds it: a mixed
// complex type over a lax ##any element wildcard, with a lax ##any attribute
// wildcard, and itself as its {base type definition}. Phase D needs it present
// whenever a type under test names it as a base, and its presence is also what
// makes derivation-ok-restriction clause 2.1 reachable in these tests.
func dAnyType(t *testing.T) ComplexType {
	t.Helper()
	w := uWildcard(t, NamespaceConstraintAny, nil, ProcessLax)
	inner := uParticle(t, uUnbounded(t, 0), ResolvedTerm{Term: w})
	seq := uGroup(t, CompositorSequence, inner)
	ct, err := NewComplexType(xsderr.Loc{}, anyTypeName, anyTypeName, nil, DerivationRestriction, false,
		nil, nil, &w, ElementContent{Mixed: true, Particle: uOne(t, ResolvedTerm{Term: seq})}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType(xs:anyType): %v", err)
	}
	return ct
}

// dType builds a complex type restricting base, with the given {content type},
// {attribute uses} and {attribute wildcard}.
func dType(t *testing.T, name, base QName, content ContentType, uses []AttributeUse, wildcard *Wildcard) ComplexType {
	t.Helper()
	ct, err := NewComplexType(xsderr.Loc{}, name, base, nil, DerivationRestriction, false,
		uses, nil, wildcard, content, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType(%s): %v", name, err)
	}
	return ct
}

// dFinal builds a complex type whose {final} carries the given derivation
// methods.
func dFinal(t *testing.T, name, base QName, final []DerivationMethod) ComplexType {
	t.Helper()
	ct, err := NewComplexType(xsderr.Loc{}, name, base, final, DerivationRestriction, false,
		nil, nil, nil, EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType(%s): %v", name, err)
	}
	return ct
}

// dSimple builds a named atomic simple type restricting base.
func dSimple(t *testing.T, name QName, base *SimpleType) *SimpleType {
	t.Helper()
	st, err := NewSimpleType(xsderr.Loc{}, name, Atomic{}, base, nil, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(%s): %v", name, err)
	}
	return st
}

// dAttr builds an optional attribute use over a sibling local declaration of the
// given name and type.
func dAttr(t *testing.T, name, typeName QName) AttributeUse {
	t.Helper()
	return dAttrUse(t, name, typeName, false, nil)
}

// dAttrUse builds an attribute use over a sibling local declaration, with the
// given {required} and the declaration's {value constraint}.
func dAttrUse(t *testing.T, name, typeName QName, required bool, vc *ValueConstraint) AttributeUse {
	t.Helper()
	decl, err := NewAttributeDeclaration(xsderr.Loc{}, name, TypeDefinitionRef{Name: typeName}, ScopeLocal, vc, false, nil)
	if err != nil {
		t.Fatalf("NewAttributeDeclaration(%s): %v", name, err)
	}
	u, err := NewAttributeUse(xsderr.Loc{}, required, LocalAttributeDeclaration{Declaration: decl}, nil, false, nil)
	if err != nil {
		t.Fatalf("NewAttributeUse(%s): %v", name, err)
	}
	return u
}

// dFinalize assembles a schema carrying xs:anyType, the two named simple types
// the attribute tests use, and whatever build adds, then finalizes it.
func dFinalize(t *testing.T, build func(*SchemaBuilder)) error {
	t.Helper()
	b := NewSchemaBuilder()
	b.AddType(dAnyType(t))
	b.AddType(uNamedType(t, uq("T")))
	str := dSimple(t, uq("str"), AnyAtomicType())
	b.AddType(str)
	b.AddType(dSimple(t, uq("narrow"), str))
	b.AddType(dSimple(t, uq("other"), AnyAtomicType()))
	build(b)
	_, err := b.Finalize()
	return err
}

// dElementContent wraps a model group as an element-only or mixed {content type}.
func dElementContent(t *testing.T, mixed bool, g ModelGroup) ElementContent {
	t.Helper()
	return ElementContent{Mixed: mixed, Particle: uOne(t, ResolvedTerm{Term: g})}
}

// TestDerivationOKRestrictionClause1 pins clause 1: a base whose {final}
// contains restriction may not be restricted.
func TestDerivationOKRestrictionClause1(t *testing.T) {
	err := dFinalize(t, func(b *SchemaBuilder) {
		b.AddType(dFinal(t, uq("base"), anyTypeName, []DerivationMethod{DerivationRestriction}))
		b.AddType(dType(t, uq("derived"), uq("base"), EmptyContent{}, nil, nil))
	})
	expectRule(t, err, ruleDerivationOKRestriction)
}

// TestDerivationOKRestrictionClause1Passes is the control: an {final} of
// extension alone does not block restriction.
func TestDerivationOKRestrictionClause1Passes(t *testing.T) {
	err := dFinalize(t, func(b *SchemaBuilder) {
		b.AddType(dFinal(t, uq("base"), anyTypeName, []DerivationMethod{DerivationExtension}))
		b.AddType(dType(t, uq("derived"), uq("base"), EmptyContent{}, nil, nil))
	})
	if err != nil {
		t.Fatalf("restricting a base final only for extension was rejected: %v", err)
	}
}

// TestDerivationOKRestrictionClause21 pins clause 2.1: a base that IS xs:anyType
// discharges clause 2 outright, whatever the derived {content type} is.
func TestDerivationOKRestrictionClause21(t *testing.T) {
	err := dFinalize(t, func(b *SchemaBuilder) {
		b.AddType(dType(t, uq("derived"), anyTypeName,
			dElementContent(t, false, uGroup(t, CompositorSequence, rElem(t, 1, 1))), nil, nil))
	})
	if err != nil {
		t.Fatalf("restricting xs:anyType was rejected: %v", err)
	}
}

// TestDerivationOKRestrictionClause23 pins clause 2.3 — an EMPTY restriction of
// an element-only base is valid exactly when the base's {particle} is
// ·emptiable·. This is the case that exercises cos-group-emptiable /
// cos-seq-range / cos-choice-range end to end.
func TestDerivationOKRestrictionClause23(t *testing.T) {
	for _, tc := range []struct {
		name      string
		baseModel ModelGroup
		wantOK    bool
	}{
		{"base with a mandatory element is not emptiable",
			uGroup(t, CompositorSequence, rElem(t, 1, 1)), false},
		{"base whose only member is optional is emptiable",
			uGroup(t, CompositorSequence, rElem(t, 0, 1)), true},
		{"base sequence summing to a mandatory member is not emptiable",
			uGroup(t, CompositorSequence, rNamedElem(t, "e1", 0, 1), rNamedElem(t, "e2", 1, 1)), false},
		{"base choice with one skippable branch is emptiable",
			uGroup(t, CompositorChoice, rNamedElem(t, "e1", 1, 1), rNamedElem(t, "e2", 0, 1)), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := dFinalize(t, func(b *SchemaBuilder) {
				b.AddType(dType(t, uq("base"), anyTypeName, dElementContent(t, false, tc.baseModel), nil, nil))
				b.AddType(dType(t, uq("derived"), uq("base"), EmptyContent{}, nil, nil))
			})
			if tc.wantOK && err != nil {
				t.Fatalf("emptying an ·emptiable· base was rejected: %v", err)
			}
			if !tc.wantOK {
				expectRule(t, err, ruleDerivationOKRestriction)
			}
		})
	}
}

// TestDerivationOKRestrictionClause22 pins clause 2.2: a simple-content
// restriction's {simple type definition} must be validly derived from the base's
// (2.2.2.1).
func TestDerivationOKRestrictionClause22(t *testing.T) {
	for _, tc := range []struct {
		name       string
		derivedSTs string
		wantOK     bool
	}{
		{"narrowing to a derived simple type is valid", "narrow", true},
		{"widening to an unrelated simple type is not", "other", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := dFinalize(t, func(b *SchemaBuilder) {
				str := dSimple(t, uq("str2"), AnyAtomicType())
				narrow := dSimple(t, uq("narrow2"), str)
				other := dSimple(t, uq("other2"), AnyAtomicType())
				b.AddType(str)
				b.AddType(narrow)
				b.AddType(other)
				derived := narrow
				if tc.derivedSTs == "other" {
					derived = other
				}
				b.AddType(dType(t, uq("base"), anyTypeName, SimpleContent{SimpleType: str}, nil, nil))
				b.AddType(dType(t, uq("derived"), uq("base"), SimpleContent{SimpleType: derived}, nil, nil))
			})
			if tc.wantOK && err != nil {
				t.Fatalf("a valid simple-content restriction was rejected: %v", err)
			}
			if !tc.wantOK {
				expectRule(t, err, ruleDerivationOKRestriction)
			}
		})
	}
}

// TestDerivationOKRestrictionClause241 pins clause 2.4.1: a MIXED restriction of
// an element-only base fails the variety match, and no other branch of clause 2
// rescues it. This is the case that must survive the 2.4.2 seam returning true —
// if the seam ever leaked into 2.4.1 this test would go green wrongly.
func TestDerivationOKRestrictionClause241(t *testing.T) {
	err := dFinalize(t, func(b *SchemaBuilder) {
		b.AddType(dType(t, uq("base"), anyTypeName,
			dElementContent(t, false, uGroup(t, CompositorSequence, rElem(t, 1, 1))), nil, nil))
		b.AddType(dType(t, uq("derived"), uq("base"),
			dElementContent(t, true, uGroup(t, CompositorSequence, rElem(t, 1, 1))), nil, nil))
	})
	expectRule(t, err, ruleDerivationOKRestriction)
}

// TestDerivationOKRestrictionClause242 pins that clause 2.4.2 is now DECIDED
// (cos-content-act-restrict, contentrestricts.go) rather than provisionally
// accepted: an element-only restriction of an element-only base passes 2.4.1 and
// is then rejected because its content model admits sequences the base does not.
// The shape is the one the pre-#263 seam test asserted must be accepted, so the
// landing shows up as a deliberate change here.
func TestDerivationOKRestrictionClause242(t *testing.T) {
	err := dFinalize(t, func(b *SchemaBuilder) {
		b.AddType(dType(t, uq("base"), anyTypeName,
			dElementContent(t, false, uGroup(t, CompositorSequence, rElem(t, 1, 1))), nil, nil))
		b.AddType(dType(t, uq("derived"), uq("base"),
			dElementContent(t, false, uGroup(t, CompositorSequence, rElem(t, 5, 9))), nil, nil))
	})
	expectRule(t, err, ruleDerivationOKRestriction)
}

// TestDerivationOKRestrictionClause3UnknownAttribute pins clause 3 (c-ran): a
// restriction may not introduce an attribute the base neither declares nor
// admits through an {attribute wildcard}.
func TestDerivationOKRestrictionClause3UnknownAttribute(t *testing.T) {
	err := dFinalize(t, func(b *SchemaBuilder) {
		b.AddType(dType(t, uq("base"), anyTypeName, EmptyContent{}, nil, nil))
		b.AddType(dType(t, uq("derived"), uq("base"), EmptyContent{},
			[]AttributeUse{dAttr(t, uq("a"), uq("str"))}, nil))
	})
	expectRule(t, err, ruleDerivationOKRestriction)
}

// TestDerivationOKRestrictionClause3WildcardAdmits is the control for the
// previous test: a base with an {attribute wildcard} admitting the name gives a
// ·default binding·, so the restriction is accepted.
func TestDerivationOKRestrictionClause3WildcardAdmits(t *testing.T) {
	err := dFinalize(t, func(b *SchemaBuilder) {
		w := uWildcard(t, NamespaceConstraintAny, nil, ProcessLax)
		b.AddType(dType(t, uq("base"), anyTypeName, EmptyContent{}, nil, &w))
		b.AddType(dType(t, uq("derived"), uq("base"), EmptyContent{},
			[]AttributeUse{dAttr(t, uq("a"), uq("str"))}, nil))
	})
	if err != nil {
		t.Fatalf("an attribute admitted by the base's {attribute wildcard} was rejected: %v", err)
	}
}

// TestDerivationOKRestrictionClause3InheritedAttribute pins the §3.4.2.4 clause
// 3 fold clause 3 (c-ran) reads B.{attribute uses} through: in the chain
// A(@x) ← B(inheriting @x, re-declaring nothing) ← C, x IS a member of
// B.{attribute uses}, even though the producer maps only B's own <attribute>
// children onto B. Charging C for it would be a false reject.
//
// The table's rejecting rows are what keep the passing rows honest: a widened
// type is still charged (so the inherited use is really found and compared under
// loc-testSubP clause 5, not waved through), and a name absent from the WHOLE
// chain is still charged (so the walk does not simply admit everything once it
// reaches xs:anyType).
func TestDerivationOKRestrictionClause3InheritedAttribute(t *testing.T) {
	for _, tc := range []struct {
		name   string
		use    AttributeUse
		wantOK bool
	}{
		{"re-declaring an attribute the base only inherits is valid",
			dAttr(t, uq("x"), uq("str")), true},
		{"re-declaring the inherited attribute as required is valid",
			dAttrUse(t, uq("x"), uq("str"), true, nil), true},
		{"narrowing the inherited attribute's type is valid",
			dAttr(t, uq("x"), uq("narrow")), true},
		{"widening the inherited attribute's type is still charged",
			dAttr(t, uq("x"), uq("other")), false},
		{"a name absent from the whole base chain is still charged",
			dAttr(t, uq("z"), uq("str")), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := dFinalize(t, func(b *SchemaBuilder) {
				b.AddType(dType(t, uq("A"), anyTypeName, EmptyContent{},
					[]AttributeUse{dAttr(t, uq("x"), uq("str"))}, nil))
				b.AddType(dType(t, uq("B"), uq("A"), EmptyContent{}, nil, nil))
				b.AddType(dType(t, uq("C"), uq("B"), EmptyContent{}, []AttributeUse{tc.use}, nil))
			})
			if tc.wantOK && err != nil {
				t.Fatalf("an attribute the base inherits was rejected: %v", err)
			}
			if !tc.wantOK {
				expectRule(t, err, ruleDerivationOKRestriction)
			}
		})
	}
}

// TestDerivationOKRestrictionClause3WidenedType pins clause 3 via loc-testSubP
// clause 5.1: a restriction may narrow an attribute's type but not widen it.
func TestDerivationOKRestrictionClause3WidenedType(t *testing.T) {
	for _, tc := range []struct {
		name        string
		derivedType QName
		wantOK      bool
	}{
		{"narrowing the attribute type is valid", uq("narrow"), true},
		{"widening it to an unrelated type is not", uq("other"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := dFinalize(t, func(b *SchemaBuilder) {
				b.AddType(dType(t, uq("base"), anyTypeName, EmptyContent{},
					[]AttributeUse{dAttr(t, uq("a"), uq("str"))}, nil))
				b.AddType(dType(t, uq("derived"), uq("base"), EmptyContent{},
					[]AttributeUse{dAttr(t, uq("a"), tc.derivedType)}, nil))
			})
			if tc.wantOK && err != nil {
				t.Fatalf("narrowing an attribute's type was rejected: %v", err)
			}
			if !tc.wantOK {
				expectRule(t, err, ruleDerivationOKRestriction)
			}
		})
	}
}

// TestDerivationOKRestrictionClause3RelaxedRequired pins the cvc-complex-type
// clause 3 half of c-ran: an attribute the base REQUIRES may not be relaxed to
// optional by a restriction.
func TestDerivationOKRestrictionClause3RelaxedRequired(t *testing.T) {
	err := dFinalize(t, func(b *SchemaBuilder) {
		b.AddType(dType(t, uq("base"), anyTypeName, EmptyContent{},
			[]AttributeUse{dAttrUse(t, uq("a"), uq("str"), true, nil)}, nil))
		b.AddType(dType(t, uq("derived"), uq("base"), EmptyContent{},
			[]AttributeUse{dAttrUse(t, uq("a"), uq("str"), false, nil)}, nil))
	})
	expectRule(t, err, ruleDerivationOKRestriction)
}

// TestDerivationOKRestrictionClause3DroppedFixed pins loc-testSubP clause 5.2: a
// restriction may not drop the base's ·effective value constraint· of {variety}
// fixed.
func TestDerivationOKRestrictionClause3DroppedFixed(t *testing.T) {
	fixed := NewValueConstraint(ValueFixed, "7")
	for _, tc := range []struct {
		name      string
		derivedVC *ValueConstraint
		wantOK    bool
	}{
		{"keeping the same fixed value is valid", &fixed, true},
		{"dropping the fixed value constraint is not", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := dFinalize(t, func(b *SchemaBuilder) {
				b.AddType(dType(t, uq("base"), anyTypeName, EmptyContent{},
					[]AttributeUse{dAttrUse(t, uq("a"), uq("str"), false, &fixed)}, nil))
				b.AddType(dType(t, uq("derived"), uq("base"), EmptyContent{},
					[]AttributeUse{dAttrUse(t, uq("a"), uq("str"), false, tc.derivedVC)}, nil))
			})
			if tc.wantOK && err != nil {
				t.Fatalf("preserving the base's fixed value was rejected: %v", err)
			}
			if !tc.wantOK {
				expectRule(t, err, ruleDerivationOKRestriction)
			}
		})
	}
}

// TestDerivationOKRestrictionClause4Element pins clause 4 (c-vs-ctd-r) over
// elements: the ·locally declared type· of an element name within the
// restriction must be ·validly substitutable· for its ·locally declared type·
// within the base.
func TestDerivationOKRestrictionClause4Element(t *testing.T) {
	for _, tc := range []struct {
		name        string
		derivedType QName
		wantOK      bool
	}{
		{"narrowing an element's type is valid", uq("narrow"), true},
		{"retyping it to an unrelated type is not", uq("other"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := dFinalize(t, func(b *SchemaBuilder) {
				baseModel := uGroup(t, CompositorSequence,
					uOne(t, ResolvedTerm{Term: uLocal(t, uq("e"), uq("str"))}))
				derivedModel := uGroup(t, CompositorSequence,
					uOne(t, ResolvedTerm{Term: uLocal(t, uq("e"), tc.derivedType)}))
				b.AddType(dType(t, uq("base"), anyTypeName, dElementContent(t, false, baseModel), nil, nil))
				b.AddType(dType(t, uq("derived"), uq("base"), dElementContent(t, false, derivedModel), nil, nil))
			})
			if tc.wantOK && err != nil {
				t.Fatalf("narrowing an element's declared type was rejected: %v", err)
			}
			if !tc.wantOK {
				expectRule(t, err, ruleDerivationOKRestriction)
			}
		})
	}
}

// TestDerivationOKRestrictionClause4AnyTypeBase pins the ·xs:anyType· escape
// clause 4 needs: a base that leaves an element untyped declares it xs:anyType
// (§3.3.2.1 case 4), and EVERY type — simple types included — is validly
// substitutable for xs:anyType. Rejecting here would false-reject the commonest
// restriction shape there is.
func TestDerivationOKRestrictionClause4AnyTypeBase(t *testing.T) {
	err := dFinalize(t, func(b *SchemaBuilder) {
		baseModel := uGroup(t, CompositorSequence,
			uOne(t, ResolvedTerm{Term: uLocal(t, uq("e"), anyTypeName)}))
		derivedModel := uGroup(t, CompositorSequence,
			uOne(t, ResolvedTerm{Term: uLocal(t, uq("e"), uq("str"))}))
		b.AddType(dType(t, uq("base"), anyTypeName, dElementContent(t, false, baseModel), nil, nil))
		b.AddType(dType(t, uq("derived"), uq("base"), dElementContent(t, false, derivedModel), nil, nil))
	})
	if err != nil {
		t.Fatalf("typing an element the base left as xs:anyType was rejected: %v", err)
	}
}

// TestCTPropsCorrectSimpleBase pins ct-props-correct clause 2: a simple {base
// type definition} forces {derivation method} = extension, so a restriction of
// one is rejected — and charged ct-props-correct, not the coarser
// derivation-ok-restriction clause 1.
func TestCTPropsCorrectSimpleBase(t *testing.T) {
	err := dFinalize(t, func(b *SchemaBuilder) {
		b.AddType(dType(t, uq("derived"), uq("str"), EmptyContent{}, nil, nil))
	})
	expectRule(t, err, ruleCTPropsCorrect)
}

// TestCTPropsCorrectDuplicateAttributeUses pins ct-props-correct clause 4: two
// {attribute uses} may not share an {attribute declaration} expanded name.
func TestCTPropsCorrectDuplicateAttributeUses(t *testing.T) {
	err := dFinalize(t, func(b *SchemaBuilder) {
		b.AddType(dType(t, uq("ct"), anyTypeName, EmptyContent{},
			[]AttributeUse{dAttr(t, uq("a"), uq("str")), dAttr(t, uq("a"), uq("str"))}, nil))
	})
	expectRule(t, err, ruleCTPropsCorrect)
}

// TestCTPropsCorrectDistinctAttributeUsesPass is the control: two uses of
// different names are fine.
func TestCTPropsCorrectDistinctAttributeUsesPass(t *testing.T) {
	err := dFinalize(t, func(b *SchemaBuilder) {
		b.AddType(dType(t, uq("ct"), anyTypeName, EmptyContent{},
			[]AttributeUse{dAttr(t, uq("a"), uq("str")), dAttr(t, uq("b"), uq("str"))}, nil))
	})
	if err != nil {
		t.Fatalf("two distinctly named attribute uses were rejected: %v", err)
	}
}

// TestDerivedOKComplexHonoursBlockingSet pins that cos-ct-derived-ok's blocking
// subset is load-bearing on the complex side (warden's caveat to confirmation
// 1): the same pair is derived-OK under the empty set and not under {restriction}.
func TestDerivedOKComplexHonoursBlockingSet(t *testing.T) {
	b := NewSchemaBuilder()
	b.AddType(dAnyType(t))
	b.AddType(dType(t, uq("base"), anyTypeName, EmptyContent{}, nil, nil))
	b.AddType(dType(t, uq("derived"), uq("base"), EmptyContent{}, nil, nil))
	s, err := b.Finalize()
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	base, _ := s.Type(uq("base"))
	derived, _ := s.Type(uq("derived"))
	d, ok := derived.(ComplexType)
	if !ok {
		t.Fatalf("derived is not a ComplexType")
	}
	if !s.derivedOKComplex(d, base, nil) {
		t.Fatalf("derived is not OK from base under the empty blocking set")
	}
	if s.derivedOKComplex(d, base, []DerivationMethod{DerivationRestriction}) {
		t.Fatalf("derived was accepted from base with restriction in the blocking set")
	}
	if !s.derivedOKComplex(d, base, restrictionBlockingKeywords) {
		t.Fatalf("clause 4's {extension, list, union} set must not block a restriction")
	}
}
