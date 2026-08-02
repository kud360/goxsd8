package xsd

import (
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// These tests are package-internal: checkComplexTypeExtension runs inside
// SchemaBuilder.Finalize and is unexported (STYLE T5), so the assertions are
// made on the error Finalize returns. The component builders come from
// complexderivation_test.go, particleattribution_test.go and
// effectivetotalrange_test.go — one set of helpers, not four (STYLE T4).

// xType is dType's extension twin: dType hard-codes DerivationRestriction, and
// every type under test here derives by extension.
func xType(t *testing.T, name, base QName, content ContentType, uses []AttributeUse, wildcard *Wildcard) ComplexType {
	t.Helper()
	ct, err := NewComplexType(xsderr.Loc{}, name, base, nil, DerivationExtension, false,
		uses, wildcard, content, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType(%s): %v", name, err)
	}
	return ct
}

// xFinal builds an empty-content complex type whose {final} carries the given
// derivation methods, to serve as a base clause 1.1 must reject an extension of.
func xFinal(t *testing.T, name, base QName, final []DerivationMethod) ComplexType {
	t.Helper()
	ct, err := NewComplexType(xsderr.Loc{}, name, base, final, DerivationRestriction, false,
		nil, nil, EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType(%s): %v", name, err)
	}
	return ct
}

// xAnyTypeParticle is the {content type}.{particle} the seeded xs:anyType
// carries, so an extension of it can satisfy cos-particle-extend clause 1 ("they
// are the same particle") without adding a second wildcard the content model
// would then be ambiguous over.
func xAnyTypeParticle(t *testing.T) Particle {
	t.Helper()
	ec, ok := dAnyType(t).ContentType().(ElementContent)
	if !ok {
		t.Fatalf("xs:anyType {content type} is not element content")
	}
	return ec.Particle
}

// xOpenContent builds an {open content} record with the given mode over a
// wildcard of the given namespace constraint.
func xOpenContent(t *testing.T, mode OpenContentMode, w Wildcard) *OpenContent {
	t.Helper()
	oc, err := NewOpenContent(xsderr.Loc{}, mode, w)
	if err != nil {
		t.Fatalf("NewOpenContent: %v", err)
	}
	return &oc
}

// xSchema is dFinalize's twin for the one test that needs the finalized
// *Schema rather than the error, so a predicate can be exercised directly.
func xSchema(t *testing.T, build func(*SchemaBuilder)) *Schema {
	t.Helper()
	b := NewSchemaBuilder()
	b.AddType(dAnyType(t))
	b.AddType(uNamedType(t, uq("T")))
	build(b)
	s, err := b.Finalize()
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	return s
}

// TestCosCTExtendsClause11 pins clause 1.1: a base whose {final} contains
// extension may not be extended.
func TestCosCTExtendsClause11(t *testing.T) {
	err := dFinalize(t, func(b *SchemaBuilder) {
		b.AddType(xFinal(t, uq("base"), anyTypeName, []DerivationMethod{DerivationExtension}))
		b.AddType(xType(t, uq("derived"), uq("base"), EmptyContent{}, nil, nil))
	})
	expectRule(t, err, ruleCosCTExtends)
}

// TestCosCTExtendsClause11Passes is the control: a {final} of restriction alone
// does not block extension.
func TestCosCTExtendsClause11Passes(t *testing.T) {
	err := dFinalize(t, func(b *SchemaBuilder) {
		b.AddType(xFinal(t, uq("base"), anyTypeName, []DerivationMethod{DerivationRestriction}))
		b.AddType(xType(t, uq("derived"), uq("base"), EmptyContent{}, nil, nil))
	})
	if err != nil {
		t.Fatalf("extending a base final only for restriction was rejected: %v", err)
	}
}

// TestCosCTExtendsClause12 pins what an extension may do with its base's
// attributes. The PASSING row is the load-bearing one: the extension declares
// only its own new attribute and inherits the base's, which is the ordinary
// shape — §3.4.2.4 clause 3.1 puts the base's use into the extension's
// {attribute uses}, and a property-identity predicate that reported it as
// different from itself would false-reject every extension of a base with
// attributes.
//
// The two rejecting rows are charged ct-props-correct clause 4, NOT c-cte, and
// that is the spec's own routing rather than an accident of ordering: clause 3.1
// inherits the base's uses UNCONDITIONALLY, so re-declaring the name leaves the
// extension holding two uses for it, which clause 4 forbids outright — an
// extension may add attributes, never restate the base's, whether or not the
// restatement is identical. Before §3.4.2.4 clause 3 was materialised (#401) the
// re-declaration hid the inherited use instead, and c-cte was what caught it.
func TestCosCTExtendsClause12(t *testing.T) {
	for _, tc := range []struct {
		name     string
		uses     []AttributeUse
		wantRule xsderr.Rule // empty: the extension must be accepted
	}{
		{"an extension that adds its own attribute inherits the base's unchanged",
			[]AttributeUse{dAttr(t, uq("b"), uq("str"))}, ""},
		{"an extension that re-declares the base's attribute as required duplicates it",
			[]AttributeUse{dAttrUse(t, uq("a"), uq("str"), true, nil)}, ruleCTPropsCorrect},
		{"an extension that re-declares the base's attribute with another type duplicates it",
			[]AttributeUse{dAttr(t, uq("a"), uq("other"))}, ruleCTPropsCorrect},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := dFinalize(t, func(b *SchemaBuilder) {
				b.AddType(dType(t, uq("base"), anyTypeName, EmptyContent{},
					[]AttributeUse{dAttr(t, uq("a"), uq("str"))}, nil))
				b.AddType(xType(t, uq("derived"), uq("base"), EmptyContent{}, tc.uses, nil))
			})
			if tc.wantRule == "" {
				if err != nil {
					t.Fatalf("a valid extension was rejected: %v", err)
				}
				return
			}
			expectRule(t, err, tc.wantRule)
		})
	}
}

// TestCosCTExtendsClause141 pins clause 1.4.1: simple content on both sides with
// THE SAME {simple type definition}. The passing row uses the same *SimpleType
// the base holds, which is what the producer yields (it reuses the base's
// pointer, never rebuilding one).
func TestCosCTExtendsClause141(t *testing.T) {
	for _, tc := range []struct {
		name    string
		derived string
		wantOK  bool
	}{
		{"the same simple type definition satisfies the clause", "str", true},
		{"a derived simple type is not the same one", "narrow", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := dFinalize(t, func(b *SchemaBuilder) {
				str := dSimple(t, uq("str2"), AnyAtomicType())
				narrow := dSimple(t, uq("narrow2"), str)
				b.AddType(str)
				b.AddType(narrow)
				derived := str
				if tc.derived == "narrow" {
					derived = narrow
				}
				b.AddType(dType(t, uq("base"), anyTypeName, SimpleContent{SimpleType: str}, nil, nil))
				b.AddType(xType(t, uq("derived"), uq("base"), SimpleContent{SimpleType: derived}, nil, nil))
			})
			if tc.wantOK && err != nil {
				t.Fatalf("a valid simple-content extension was rejected: %v", err)
			}
			if !tc.wantOK {
				expectRule(t, err, ruleCosCTExtends)
			}
		})
	}
}

// TestCosCTExtendsClause14BranchSelection pins that T's own {content type}
// variety selects THE branch of clause 1.4 that must carry the derivation: a
// simple T is judged by 1.4.1 alone, an empty T by 1.4.2 alone, and an
// element-only/mixed T by 1.4.3 alone. Each row pairs a T variety with a base
// variety the selected branch rejects.
func TestCosCTExtendsClause14BranchSelection(t *testing.T) {
	str := dSimple(t, uq("str3"), AnyAtomicType())
	model := uGroup(t, CompositorSequence, rElem(t, 1, 1))
	for _, tc := range []struct {
		name        string
		baseContent ContentType
		derived     ContentType
	}{
		{"clause 1.4.1: simple T over an element-only base", dElementContent(t, false, model), SimpleContent{SimpleType: str}},
		{"clause 1.4.2: empty T over a simple base", SimpleContent{SimpleType: str}, EmptyContent{}},
		{"clause 1.4.3.2: element-only T over a simple base", SimpleContent{SimpleType: str}, dElementContent(t, false, model)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := dFinalize(t, func(b *SchemaBuilder) {
				b.AddType(str)
				b.AddType(dType(t, uq("base"), anyTypeName, tc.baseContent, nil, nil))
				b.AddType(xType(t, uq("derived"), uq("base"), tc.derived, nil, nil))
			})
			expectRule(t, err, ruleCosCTExtends)
		})
	}
}

// TestCosCTExtendsClause1432 pins the two branches of clause 1.4.3.2 that do not
// reach the particle: an EMPTY base discharges the clause outright (1.4.3.2.1),
// and a mixed/element-only disagreement fails it (1.4.3.2.2.1).
func TestCosCTExtendsClause1432(t *testing.T) {
	model := uGroup(t, CompositorSequence, rElem(t, 1, 1))
	for _, tc := range []struct {
		name        string
		baseContent ContentType
		mixed       bool
		wantOK      bool
	}{
		{"clause 1.4.3.2.1: any element content extends an empty base", EmptyContent{}, false, true},
		{"clause 1.4.3.2.2.1: an element-only T over a mixed base", dElementContent(t, true, model), false, false},
		{"clause 1.4.3.2.2.1: a mixed T over an element-only base", dElementContent(t, false, model), true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := dFinalize(t, func(b *SchemaBuilder) {
				b.AddType(dType(t, uq("base"), anyTypeName, tc.baseContent, nil, nil))
				b.AddType(xType(t, uq("derived"), uq("base"), dElementContent(t, tc.mixed, model), nil, nil))
			})
			if tc.wantOK && err != nil {
				t.Fatalf("a valid element-content extension was rejected: %v", err)
			}
			if !tc.wantOK {
				expectRule(t, err, ruleCosCTExtends)
			}
		})
	}
}

// TestCosParticleExtend pins clause 1.4.3.2.2.2 and all three clauses of its
// delegate cos-particle-extend (§3.9.6.2), and pins the DELEGATE's own rule ID on
// the failing rows — a coarse cos-ct-extends there would mean the 1.4.3.2.2.1
// branch selection was not being relied on.
func TestCosParticleExtend(t *testing.T) {
	baseSeq := uGroup(t, CompositorSequence, rNamedElem(t, "e1", 1, 1))
	baseParticle := uOne(t, ResolvedTerm{Term: baseSeq})
	baseAll := uGroup(t, CompositorAll, rNamedElem(t, "e1", 1, 1))
	for _, tc := range []struct {
		name        string
		baseContent ElementContent
		derived     ElementContent
		wantOK      bool
	}{
		{"clause 1: the same particle",
			ElementContent{Particle: baseParticle},
			ElementContent{Particle: uOne(t, ResolvedTerm{Term: baseSeq})}, true},
		{"clause 2: a 1..1 sequence whose first member is the base's particle",
			ElementContent{Particle: baseParticle},
			ElementContent{Particle: uOne(t, ResolvedTerm{Term: uGroup(t, CompositorSequence,
				baseParticle, rNamedElem(t, "e2", 1, 1))})}, true},
		{"clause 2 rejects a sequence whose FIRST member is not the base's particle",
			ElementContent{Particle: baseParticle},
			ElementContent{Particle: uOne(t, ResolvedTerm{Term: uGroup(t, CompositorSequence,
				rNamedElem(t, "e2", 1, 1), baseParticle)})}, false},
		{"clause 3: the base's all group is a prefix of the extension's",
			ElementContent{Particle: uOne(t, ResolvedTerm{Term: baseAll})},
			ElementContent{Particle: uOne(t, ResolvedTerm{Term: uGroup(t, CompositorAll,
				rNamedElem(t, "e1", 1, 1), rNamedElem(t, "e2", 1, 1))})}, true},
		{"clause 3 rejects an all group the base's is not a prefix of",
			ElementContent{Particle: uOne(t, ResolvedTerm{Term: baseAll})},
			ElementContent{Particle: uOne(t, ResolvedTerm{Term: uGroup(t, CompositorAll,
				rNamedElem(t, "e2", 1, 1), rNamedElem(t, "e1", 1, 1))})}, false},
		{"an unrelated particle satisfies no clause",
			ElementContent{Particle: baseParticle},
			ElementContent{Particle: uOne(t, ResolvedTerm{Term: uGroup(t, CompositorSequence,
				rNamedElem(t, "e2", 1, 1))})}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := dFinalize(t, func(b *SchemaBuilder) {
				b.AddType(dType(t, uq("base"), anyTypeName, tc.baseContent, nil, nil))
				b.AddType(xType(t, uq("derived"), uq("base"), tc.derived, nil, nil))
			})
			if tc.wantOK && err != nil {
				t.Fatalf("a valid particle extension was rejected: %v", err)
			}
			if !tc.wantOK {
				expectRule(t, err, ruleCosParticleExtend)
			}
		})
	}
}

// TestCosCTExtendsOpenContent pins clauses 1.4.3.2.2.3 (mode compatibility) and
// 1.4.3.2.2.4 (the base's open-content wildcard is a cos-ns-subset of the
// extension's). The particle is the same on both sides throughout, so
// cos-particle-extend clause 1 discharges 1.4.3.2.2.2 and only the open-content
// conditions can decide the row.
func TestCosCTExtendsOpenContent(t *testing.T) {
	seq := uGroup(t, CompositorSequence, rNamedElem(t, "e1", 1, 1))
	particle := uOne(t, ResolvedTerm{Term: seq})
	anyW := uWildcard(t, NamespaceConstraintAny, nil, ProcessLax)
	oneNS := uWildcard(t, NamespaceConstraintEnumeration, []Namespace{NamespaceName(uns)}, ProcessLax)
	for _, tc := range []struct {
		name   string
		base   *OpenContent
		self   *OpenContent
		wantOK bool
	}{
		{"clause 1.4.3.2.2.3.1: an absent base open content admits any", nil,
			xOpenContent(t, OpenContentSuffix, anyW), true},
		{"clause 1.4.3.2.2.3.2: an interleave extension admits any base mode",
			xOpenContent(t, OpenContentSuffix, anyW), xOpenContent(t, OpenContentInterleave, anyW), true},
		{"clause 1.4.3.2.2.3.3: both suffix",
			xOpenContent(t, OpenContentSuffix, anyW), xOpenContent(t, OpenContentSuffix, anyW), true},
		{"clause 1.4.3.2.2.3: an interleave base under a suffix extension satisfies no branch",
			xOpenContent(t, OpenContentInterleave, anyW), xOpenContent(t, OpenContentSuffix, anyW), false},
		{"clause 1.4.3.2.2.3: a present base open content with none on the extension",
			xOpenContent(t, OpenContentSuffix, anyW), nil, false},
		{"clause 1.4.3.2.2.4: the base's ##any is no subset of an enumerated extension",
			xOpenContent(t, OpenContentSuffix, anyW), xOpenContent(t, OpenContentSuffix, oneNS), false},
		{"clause 1.4.3.2.2.4: an enumerated base under an ##any extension",
			xOpenContent(t, OpenContentSuffix, oneNS), xOpenContent(t, OpenContentSuffix, anyW), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := dFinalize(t, func(b *SchemaBuilder) {
				b.AddType(dType(t, uq("base"), anyTypeName,
					ElementContent{Particle: particle, OpenContent: tc.base}, nil, nil))
				b.AddType(xType(t, uq("derived"), uq("base"),
					ElementContent{Particle: particle, OpenContent: tc.self}, nil, nil))
			})
			if tc.wantOK && err != nil {
				t.Fatalf("a valid open-content extension was rejected: %v", err)
			}
			if !tc.wantOK {
				expectRule(t, err, ruleCosCTExtends)
			}
		})
	}
}

// TestCosCTExtendsClause16 pins clause 1.6 (c-vs-ctd-e) and is the regression
// test for its ONE load-bearing difference from derivation-ok-restriction clause
// 4: ·without limitation· is the EMPTY blocking-keyword set, not {extension,
// list, union}.
//
// The passing row types the element with a type EXTENSION-derived from the
// base's ·locally declared type·, which is validly substitutable absolutely and
// is NOT validly substitutable subject to {extension, …}. A copy-paste of
// restrictionBlockingKeywords into the extension call site fails exactly this
// row, and nothing else in the suite.
//
// The fixture puts the base's ·locally declared type· one level up on purpose:
// mid (the extension's base) has EMPTY content, so clause 1.4 is discharged by
// 1.4.3.2.1 and the element declarations never have to satisfy
// cos-particle-extend, leaving clause 1.6 as the only clause that can decide the
// row. key-ldt-elem's case 3 then reads the type from mid's own base.
func TestCosCTExtendsClause16(t *testing.T) {
	for _, tc := range []struct {
		name        string
		elementType QName
		wantOK      bool
	}{
		{"a type derived from the base's by EXTENSION is substitutable ·without limitation·", uq("ldtExt"), true},
		{"an unrelated type is substitutable under no blocking set at all", uq("ldtOther"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := dFinalize(t, func(b *SchemaBuilder) {
				b.AddType(dType(t, uq("ldtBase"), anyTypeName, EmptyContent{}, nil, nil))
				b.AddType(xType(t, uq("ldtExt"), uq("ldtBase"), EmptyContent{}, nil, nil))
				b.AddType(dType(t, uq("ldtOther"), anyTypeName, EmptyContent{}, nil, nil))
				b.AddType(dType(t, uq("grand"), anyTypeName,
					dElementContent(t, false, uGroup(t, CompositorSequence,
						uParticle(t, uOccurs(t, 0, 1), ResolvedTerm{Term: uLocal(t, uq("e1"), uq("ldtBase"))}))),
					nil, nil))
				b.AddType(dType(t, uq("mid"), uq("grand"), EmptyContent{}, nil, nil))
				b.AddType(xType(t, uq("derived"), uq("mid"),
					dElementContent(t, false, uGroup(t, CompositorSequence,
						uOne(t, ResolvedTerm{Term: uLocal(t, uq("e1"), tc.elementType)}))),
					nil, nil))
			})
			if tc.wantOK && err != nil {
				t.Fatalf("a ·locally declared type· valid ·without limitation· was rejected: %v", err)
			}
			if !tc.wantOK {
				expectRule(t, err, ruleCosCTExtends)
			}
		})
	}
}

// TestCosCTExtendsClause2 pins case 2 — B is a simple type definition, the
// <simpleContent><extension> path — which the restriction twin has no analogue
// for and which a reflex copy of checkComplexTypeRestriction's simple-base skip
// would silently delete.
func TestCosCTExtendsClause2(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content func(base *SimpleType) ContentType
		final   []DerivationMethod
		wantOK  bool
	}{
		{"clause 2.1/2.2 hold for simple content over the base itself",
			func(base *SimpleType) ContentType { return SimpleContent{SimpleType: base} }, nil, true},
		{"clause 2.1 rejects a non-simple {content type}",
			func(*SimpleType) ContentType { return EmptyContent{} }, nil, false},
		{"clause 2.1 rejects a {simple type definition} that is not the base",
			func(*SimpleType) ContentType { return SimpleContent{SimpleType: AnyAtomicType()} }, nil, false},
		{"clause 2.2 rejects a base final for extension",
			func(base *SimpleType) ContentType { return SimpleContent{SimpleType: base} },
			[]DerivationMethod{DerivationExtension}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := dFinalize(t, func(b *SchemaBuilder) {
				base, err := NewSimpleType(xsderr.Loc{}, uq("sbase"), Atomic{}, AnyAtomicType(), nil, tc.final)
				if err != nil {
					t.Fatalf("NewSimpleType(sbase): %v", err)
				}
				b.AddType(base)
				b.AddType(xType(t, uq("derived"), uq("sbase"), tc.content(base), nil, nil))
			})
			if tc.wantOK && err != nil {
				t.Fatalf("a valid extension of a simple type was rejected: %v", err)
			}
			if !tc.wantOK {
				expectRule(t, err, ruleCosCTExtends)
			}
		})
	}
}

// TestCosCTExtendsClause15PureChain pins clause 1.5's decided half: a chain in
// which EVERY step from the ancestor whose {base type definition} is xs:anyType
// down to T is an extension satisfies the clause by the identity argument — the
// Note's re-ordering moves nothing, the collapsed intermediate IS T, and the
// residual restriction is vacuous — so nothing is synthesized and the schema is
// accepted.
//
// Both types below extend with a particle IDENTICAL to their base's
// (cos-particle-extend clause 1) and stay mixed, which is what xs:anyType's own
// {content type} forces on any extension of it: cos-ct-extends has no xs:anyType
// exemption, so clause 1.4.3.2.2.1 is decided literally against the ur-type's
// mixed variety.
func TestCosCTExtendsClause15PureChain(t *testing.T) {
	particle := xAnyTypeParticle(t)
	err := dFinalize(t, func(b *SchemaBuilder) {
		b.AddType(xType(t, uq("chain1"), anyTypeName, ElementContent{Mixed: true, Particle: particle},
			[]AttributeUse{dAttr(t, uq("a"), uq("str"))}, nil))
		b.AddType(xType(t, uq("chain2"), uq("chain1"), ElementContent{Mixed: true, Particle: particle},
			[]AttributeUse{dAttr(t, uq("b"), uq("str"))}, nil))
	})
	if err != nil {
		t.Fatalf("a pure-extension chain from xs:anyType was rejected: %v", err)
	}
	s := xSchema(t, func(b *SchemaBuilder) {
		b.AddType(xType(t, uq("chain1"), anyTypeName, ElementContent{Mixed: true, Particle: particle}, nil, nil))
		b.AddType(xType(t, uq("chain2"), uq("chain1"), ElementContent{Mixed: true, Particle: particle}, nil, nil))
		b.AddType(dType(t, uq("mixedChain1"), anyTypeName, EmptyContent{}, nil, nil))
		b.AddType(xType(t, uq("mixedChain2"), uq("mixedChain1"), EmptyContent{}, nil, nil))
	})
	chain2, _ := s.Type(uq("chain2"))
	if !s.pureExtensionChain(chain2.(ComplexType)) {
		t.Fatalf("a chain of two extensions rooted at xs:anyType did not read as pure")
	}
	mixed, _ := s.Type(uq("mixedChain2"))
	if s.pureExtensionChain(mixed.(ComplexType)) {
		t.Fatalf("a chain with a restriction step read as pure, so the clause-1.5 GAP would never apply")
	}
}

// TestCosCTExtendsDeferredClauses locks in the two clauses this file
// deliberately does not charge — 1.3 (attribute-wildcard cos-ns-subset) and 1.7
// ({assertions} prefix). Both are satisfied BY CONSTRUCTION for a faithfully
// mapped type through folds no producer performs yet (#265), so over the
// components this repo actually builds, charging either is a FALSE REJECT. The
// fixture is exactly that shape — an extension whose own {attribute wildcard} is
// NARROWER than the base's and which repeats none of the base's {assertions} —
// and it must pass. If a later change turns either into a live check, this test
// fails and the fold has to land with it.
func TestCosCTExtendsDeferredClauses(t *testing.T) {
	anyW := uWildcard(t, NamespaceConstraintAny, nil, ProcessLax)
	narrowW := uWildcard(t, NamespaceConstraintEnumeration, []Namespace{NamespaceName(uns)}, ProcessLax)
	err := dFinalize(t, func(b *SchemaBuilder) {
		base, err := NewComplexType(xsderr.Loc{}, uq("base"), anyTypeName, nil, DerivationRestriction, false,
			nil, &anyW, EmptyContent{}, nil,
			[]Assertion{NewAssertion(NewXPathExpression("true()", nil, nil, nil), nil)}, nil)
		if err != nil {
			t.Fatalf("NewComplexType(base): %v", err)
		}
		b.AddType(base)
		b.AddType(xType(t, uq("derived"), uq("base"), EmptyContent{}, nil, &narrowW))
	})
	if err != nil {
		t.Fatalf("the clause 1.3/1.7 GAP shape was rejected, so a fail-open skip became a false reject: %v", err)
	}
}
