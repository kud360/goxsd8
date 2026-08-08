package xsd

import (
	"strings"
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
		uses, nil, wildcard, content, nil, nil, nil)
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
		nil, nil, nil, EmptyContent{}, nil, nil, nil)
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
//
// grand's {base type definition} is ABSENT for the same isolating reason, and
// #392 is what made it necessary. That shape — an ancestor declaring e1, a
// restriction step removing it, an extension step adding it back with an
// extension-derived type — is EXACTLY what clause 1.5 rejects once the mixed
// chain is decided by construction, because the collapsed intermediate carries
// grand's e1 and the ·locally declared type· comparison against it runs under
// {extension, list, union}, where clause 1.6's runs under the empty set. An
// absent base makes clause 1.5's chain walk decline before it reaches xs:anyType,
// so the row here is decided by clause 1.6 and nothing else; the clause-1.5
// verdict on that same shape is pinned by
// TestCosCTExtendsClause15MixedChainElementType, which is where it belongs.
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
				b.AddType(dType(t, uq("grand"), QName{},
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

// TestCosCTExtendsFoldBackedClauses is the control for the two clauses whose
// satisfaction rides on a §3.4.2 base fold rather than on anything the source
// document says: 1.3 (attribute-wildcard cos-ns-subset, over §3.4.2.5 clause
// 2.2's fold, #265) and 1.7 ({assertions} prefix, over §3.4.2.1 clause 1's,
// #346). The fixture is the shape a fold-less assembly would get wrong — an
// extension whose OWN {attribute wildcard} is narrower than the base's, carrying
// the base's assertion ahead of its own — and it must pass, because both folds
// have run by the time either clause reads the components.
func TestCosCTExtendsFoldBackedClauses(t *testing.T) {
	anyW := uWildcard(t, NamespaceConstraintAny, nil, ProcessLax)
	narrowW := uWildcard(t, NamespaceConstraintEnumeration, []Namespace{NamespaceName(uns)}, ProcessLax)
	baseAssert := dAssert("true()")
	err := dFinalize(t, func(b *SchemaBuilder) {
		base, err := NewComplexType(xsderr.Loc{}, uq("base"), anyTypeName, nil, DerivationRestriction, false,
			nil, nil, &anyW, EmptyContent{}, nil, []Assertion{baseAssert}, nil)
		if err != nil {
			t.Fatalf("NewComplexType(base): %v", err)
		}
		b.AddType(base)
		derived, err := NewComplexType(xsderr.Loc{}, uq("derived"), uq("base"), nil, DerivationExtension, false,
			nil, nil, &narrowW, EmptyContent{}, nil, []Assertion{baseAssert, dAssert("@a > 0")}, nil)
		if err != nil {
			t.Fatalf("NewComplexType(derived): %v", err)
		}
		b.AddType(derived)
	})
	if err != nil {
		t.Fatalf("a folded extension was rejected by a fold-backed clause: %v", err)
	}
}

// TestCosCTExtendsClause17 pins clause 1.7 — B.{assertions} is a prefix of
// T.{assertions} — in both directions. §3.4.2.1 clause 1's fold makes the
// relation hold by construction for a type the producer mapped, so every
// rejecting case here is assembled through the exported constructor instead
// (dAssertType), which folds nothing.
func TestCosCTExtendsClause17(t *testing.T) {
	baseAssert, ownAssert := dAssert("true()"), dAssert("@a > 0")
	for _, tc := range []struct {
		name    string
		derived []Assertion
		wantOK  bool
	}{
		{"the fold's own output: the base's assertion, then the type's own", []Assertion{baseAssert, ownAssert}, true},
		{"the base's assertions alone are a prefix of themselves", []Assertion{baseAssert}, true},
		{"an extension that keeps only its own assertion", []Assertion{ownAssert}, false},
		{"an extension carrying no assertions at all", nil, false},
		{"an extension that puts its own assertion FIRST", []Assertion{ownAssert, baseAssert}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := dFinalize(t, func(b *SchemaBuilder) {
				b.AddType(dAssertType(t, uq("base"), anyTypeName, DerivationRestriction, []Assertion{baseAssert}))
				b.AddType(dAssertType(t, uq("derived"), uq("base"), DerivationExtension, tc.derived))
			})
			if tc.wantOK {
				if err != nil {
					t.Fatalf("an extension whose {assertions} extend the base's was rejected: %v", err)
				}
				return
			}
			expectRule(t, err, ruleCosCTExtends)
		})
	}
}

// TestCosCTExtendsClause15MixedChainAttribute pins clause 1.5 for a chain that
// MIXES the two derivation methods, over the shape the §3.4.6.2 Note names first:
// A declares @x, a restriction step PROHIBITS it, and an extension step declares
// @x again. Both real steps are legal and T ends up holding exactly one use for x
// (§3.4.2.4 clause 3.2.2), so nothing but clause 1.5 can see the chain as a
// whole.
//
// The re-ordering does NOT replay the prohibition — the Note re-orders the
// extension steps, not the restrictions — so A's use is still in the collapsed
// intermediate when T's own arrives. Whether that is legal turns entirely on
// COMPATIBILITY, which is what the Note says ("added back … in an incompatible
// way (for example, with a conflicting type assignment or value constraint)"):
//
//   - re-declared with the SAME type, the second step is the vacuous restriction
//     clause 1.5 explicitly permits and the schema is valid;
//   - re-declared with an UNRELATED type, T's ·locally declared type· for x is not
//     ·validly substitutable· for the intermediate's under {extension, list,
//     union}, so no two-step derivation exists and clause 1.5 rejects.
//
// Mutation check: deleting the collapse and returning nil, or dropping the
// second row's type change, makes the reject row pass and the test fail.
func TestCosCTExtendsClause15MixedChainAttribute(t *testing.T) {
	for _, tc := range []struct {
		name     string
		declared QName
		wantOK   bool
	}{
		{"the extension adds @x back with the type the restriction removed", uq("str"), true},
		{"the extension adds @x back with a conflicting type assignment", uq("other"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := dFinalize(t, func(b *SchemaBuilder) {
				b.AddType(dType(t, uq("A"), anyTypeName, EmptyContent{},
					[]AttributeUse{dAttr(t, uq("x"), uq("str"))}, nil))
				b.AddType(dProhibiting(t, uq("B"), uq("A"), nil, []QName{uq("x")}))
				b.AddType(xType(t, uq("E"), uq("B"), EmptyContent{},
					[]AttributeUse{dAttr(t, uq("x"), tc.declared)}, nil))
			})
			if tc.wantOK {
				if err != nil {
					t.Fatalf("a mixed chain that satisfies clause 1.5 was rejected: %v", err)
				}
				return
			}
			expectClause15(t, err)
		})
	}
}

// TestCosCTExtendsClause15MixedChainElementType pins the same verdict over the
// ELEMENT half, which is the shape the Note's "conflicting type assignment"
// describes most directly: grand declares e1 with ldtBase, mid restricts the
// content away entirely, and derived extends mid re-declaring e1.
//
// The collapsed intermediate keeps grand's e1 — the restriction step that removed
// it is not replayed — so the residual restriction has to narrow e1's type from
// ldtBase to whatever derived declares, under derivation-ok-restriction clause 4's
// blocking keywords {extension, list, union}:
//
//   - re-declared with ldtBase itself, that narrowing is the identity and the
//     schema is valid;
//   - re-declared with ldtExt, an EXTENSION of ldtBase, it is exactly what
//     restriction may not do, and clause 1.5 rejects.
//
// The second row is also the counterpart of TestCosCTExtendsClause16's passing
// row, and the pair is the point: ·without limitation· (clause 1.6, the empty
// blocking set) admits an extension-derived ·locally declared type· against the
// IMMEDIATE base, while clause 1.5 measures the same declaration against the
// collapsed intermediate under restriction's blocking set and rejects it. Both
// verdicts are correct and they are about different bases; see that test's own
// doc for why its fixture stays out of clause 1.5's reach.
func TestCosCTExtendsClause15MixedChainElementType(t *testing.T) {
	for _, tc := range []struct {
		name        string
		elementType QName
		wantOK      bool
	}{
		{"the extension adds e1 back with the type the restriction removed", uq("ldtBase"), true},
		{"the extension adds e1 back with an extension-derived type", uq("ldtExt"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := dFinalize(t, func(b *SchemaBuilder) {
				b.AddType(dType(t, uq("ldtBase"), anyTypeName, EmptyContent{}, nil, nil))
				b.AddType(xType(t, uq("ldtExt"), uq("ldtBase"), EmptyContent{}, nil, nil))
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
			if tc.wantOK {
				if err != nil {
					t.Fatalf("a mixed chain that satisfies clause 1.5 was rejected: %v", err)
				}
				return
			}
			expectClause15(t, err)
		})
	}
}

// TestCosCTExtendsClause15CollapsedIntermediate exercises the synthesis itself,
// so a failure points at the collapse rather than at whichever clause happened to
// charge. The chain is four steps of alternating method — A ←extension← E1
// ←restriction← R ←extension← T — and M must come out carrying every extension
// step's own contribution over A's, with the restriction step's contribution
// nowhere in it.
func TestCosCTExtendsClause15CollapsedIntermediate(t *testing.T) {
	s := xSchema(t, func(b *SchemaBuilder) {
		b.AddType(dSimple(t, uq("str"), AnyAtomicType()))
		b.AddType(dType(t, uq("cA"), anyTypeName, EmptyContent{}, []AttributeUse{dAttr(t, uq("a"), uq("str"))}, nil))
		b.AddType(xType(t, uq("cE1"), uq("cA"), EmptyContent{}, []AttributeUse{dAttr(t, uq("b"), uq("str"))}, nil))
		b.AddType(dProhibiting(t, uq("cR"), uq("cE1"), nil, []QName{uq("a")}))
		b.AddType(xType(t, uq("cT"), uq("cR"), EmptyContent{}, []AttributeUse{dAttr(t, uq("c"), uq("str"))}, nil))
	})
	def, _ := s.Type(uq("cT"))
	m, ok, err := s.collapsedExtension(def.(ComplexType))
	if err != nil || !ok {
		t.Fatalf("collapsedExtension(cT) = (ok=%t, err=%v), want a synthesized intermediate", ok, err)
	}
	if m.Name() != uq("cA") {
		t.Fatalf("the collapsed intermediate is named %s, want the chain's xs:anyType-based ancestor cA", m.Name())
	}
	if m.DerivationMethod() != DerivationExtension || m.Base() != TypeDefinitionOrRef(TypeDefinitionRef{Name: uq("cA")}) {
		t.Fatalf("the collapsed intermediate is a %s of %#v, want an extension referring to cA by name", m.DerivationMethod(), m.Base())
	}
	if len(m.Final()) != 0 {
		t.Fatalf("the collapsed intermediate carries a {final}, which would charge derivation-ok-restriction clause 1 against a type that does not exist")
	}
	var names []string
	for _, u := range m.AttributeUses() {
		names = append(names, attributeUseName(u).Local)
	}
	// cT's own @c, then E1's own @b, then A's @a: the restriction step's
	// prohibition of @a is NOT replayed, which is the whole of the re-ordering.
	if !fEqual(names, []string{"c", "b", "a"}) {
		t.Fatalf("the collapsed intermediate's {attribute uses} = %v, want [c b a]", names)
	}
	if _, found := s.Type(uq("cA")); !found {
		t.Fatalf("the real ancestor cA left the schema")
	}
	real, _ := s.Type(uq("cA"))
	if got := len(real.(ComplexType).AttributeUses()); got != 1 {
		t.Fatalf("the real cA now has %d {attribute uses}: the synthesized intermediate leaked into the schema", got)
	}
}

// TestOwnAttributeUsesMixedChain pins the positional recovery the collapse rests
// on, over the chain shape that would break a set-difference reading: four steps,
// ext-restr-ext, with a restriction step that PROHIBITS a name an earlier
// extension inherited. §3.4.2.4 clause 3's fold makes folded(c) = own(c) ++
// folded(b) for an extension step, per step and against its own immediate base
// only, so no history perturbs the arithmetic.
//
// The tail verification is pinned too: handed a base that is not the type's own,
// the recovery DECLINES rather than returning a prefix that means nothing.
func TestOwnAttributeUsesMixedChain(t *testing.T) {
	s := xSchema(t, func(b *SchemaBuilder) {
		b.AddType(dSimple(t, uq("str"), AnyAtomicType()))
		b.AddType(dType(t, uq("oA"), anyTypeName, EmptyContent{}, []AttributeUse{dAttr(t, uq("a"), uq("str"))}, nil))
		b.AddType(xType(t, uq("oE1"), uq("oA"), EmptyContent{}, []AttributeUse{dAttr(t, uq("b"), uq("str"))}, nil))
		b.AddType(dProhibiting(t, uq("oR"), uq("oE1"), nil, []QName{uq("a")}))
		b.AddType(xType(t, uq("oE2"), uq("oR"), EmptyContent{}, []AttributeUse{dAttr(t, uq("c"), uq("str"))}, nil))
	})
	oOwn := func(derived, base QName) []string {
		t.Helper()
		d, _ := s.Type(derived)
		b, _ := s.Type(base)
		uses, ok := ownAttributeUses(d.(ComplexType), b.(ComplexType))
		if !ok {
			t.Fatalf("ownAttributeUses(%s, %s) declined a step whose fold it inverts", derived, base)
		}
		var names []string
		for _, u := range uses {
			names = append(names, attributeUseName(u).Local)
		}
		return names
	}
	if got := oOwn(uq("oE1"), uq("oA")); !fEqual(got, []string{"b"}) {
		t.Fatalf("own {attribute uses} of oE1 = %v, want [b]", got)
	}
	// The load-bearing row: oR dropped @a, so folded(oE2) is [c b] and the prefix
	// is [c] — the arithmetic is against oE2's OWN base, not against the chain.
	if got := oOwn(uq("oE2"), uq("oR")); !fEqual(got, []string{"c"}) {
		t.Fatalf("own {attribute uses} of oE2 = %v, want [c]", got)
	}
	d, _ := s.Type(uq("oE2"))
	wrong, _ := s.Type(uq("oE1"))
	if _, ok := ownAttributeUses(d.(ComplexType), wrong.(ComplexType)); ok {
		t.Fatalf("ownAttributeUses accepted a base that is not the type's own, so the tail verification is not guarding the coupling")
	}
}

// expectClause15 asserts a cos-ct-extends rejection that clause 1.5 in
// particular made. The rule ID alone is not enough — six other clauses charge it
// — and the clause number lives in the prose by this tree's convention (#262), so
// the two-step wording is what identifies it.
func expectClause15(t *testing.T, err error) {
	t.Helper()
	expectRule(t, err, ruleCosCTExtends)
	if !strings.Contains(err.Error(), "in two steps") {
		t.Fatalf("expected a cos-ct-extends clause 1.5 rejection, got another clause: %v", err)
	}
}

// oRedefiningRestriction builds the ONE fixture shape #392 and #505 could not
// have covered between them, because neither existed when the other was written:
// a redefining <complexType> (§4.2.4 src-expredef clause 1.1) that PROHIBITS an
// attribute use its own {name}-·absent· original declares.
//
// oPair (ownedbase_test.go) builds every other owned-base fixture, but it hard-
// codes an empty prohibited-name list and puts the original's base under the
// caller's control; this shape needs the opposite of both — a prohibition, and an
// original whose {base type definition} IS ·xs:anyType·, which is what makes the
// ANONYMOUS original the ancestor A of cos-ct-extends clause 1.5.
func oRedefiningRestriction(t *testing.T, name QName, originalUses []AttributeUse, prohibited []QName) ComplexType {
	t.Helper()
	id := NewComponentID()
	original, err := NewAnonymousComplexType(xsderr.Loc{}, ComplexTypeDefinitionContext{Component: id},
		anyTypeName, nil, DerivationRestriction, false, originalUses, nil, nil, EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewAnonymousComplexType (the clause-1.1 original of %s): %v", name, err)
	}
	ct, err := NewComplexTypeOwningBase(xsderr.Loc{}, id, name, original, nil, DerivationRestriction, false,
		nil, prohibited, nil, EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexTypeOwningBase(%s): %v", name, err)
	}
	return ct
}

// TestCosCTExtendsClause15MixedChainThroughOwnedBase is the interaction #392 and
// #505 create together and neither one's own suite could reach: the §3.4.6.2
// Note's mixed chain, with its RESTRICTION step supplied by a redefining
// <complexType> — so the ancestor A that clause 1.5 collapses onto is the
// src-expredef clause 1.1 ANONYMOUS original, reachable only through the
// InlineTypeDefinition arm of {base type definition} and by no name at all.
//
// The chain is xs:anyType ←restriction← O(@x) ←restriction, prohibiting @x← mid
// ←extension re-declaring @x← derived, where O is mid's owned original. It is
// TestCosCTExtendsClause15MixedChainAttribute's chain with the middle step folded
// into a redefinition, and it decides the same way for the same reason: the
// re-ordering does not replay the prohibition, so O's @x is still in the
// collapsed intermediate when derived's own arrives, and whether that is legal
// turns on compatibility alone.
//
// Both rows exercise the reconciliation newCollapsedExtension needed to survive an
// anonymous A (complextype.go): the accept row FAILS outright if M's {base type
// definition} is built as a TypeDefinitionRef carrying A's ·absent· name, because
// checkTypeDefinitionOrRef rejects that encoding and clause 1.5 turns the
// construction error into a charge against a valid schema.
func TestCosCTExtendsClause15MixedChainThroughOwnedBase(t *testing.T) {
	for _, tc := range []struct {
		name     string
		declared QName
		wantOK   bool
	}{
		{"the extension adds @x back with the type the redefinition prohibited", uq("str"), true},
		{"the extension adds @x back with a conflicting type assignment", uq("other"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := dFinalize(t, func(b *SchemaBuilder) {
				b.AddType(oRedefiningRestriction(t, uq("mid"),
					[]AttributeUse{dAttr(t, uq("x"), uq("str"))}, []QName{uq("x")}))
				b.AddType(xType(t, uq("derived"), uq("mid"), EmptyContent{},
					[]AttributeUse{dAttr(t, uq("x"), tc.declared)}, nil))
			})
			if tc.wantOK {
				if err != nil {
					t.Fatalf("a mixed chain through a redefining <complexType> that satisfies clause 1.5 was rejected: %v", err)
				}
				return
			}
			expectClause15(t, err)
		})
	}
}

// TestCosCTExtendsClause15CollapsedOverAnonymousAncestor pins the SYNTHESIS for
// the same chain, so a failure points at how M is built rather than at whichever
// clause happened to charge. Two properties are asserted that no end-to-end
// verdict can see, because both are fail-open when wrong:
//
//   - M's {base type definition} is the InlineTypeDefinition arm holding A
//     itself. A ref cannot name an anonymous component, and a nil slot would
//     silently stop key-ldtype case 3's recursion off M at M — the one thing M
//     carries A's identity in order to avoid.
//   - M satisfies §3.4.1's {name}/{context} XOR, on the {context} side. M borrows
//     the pair off A whole, so an anonymous A yields an anonymous M with A's
//     {context}; hard-coding a nil {context} beside A's ·absent· {name} builds
//     the one component shape the tableau forbids, and newComplexType's
//     precondition makes that its CALLER's obligation, which is this test.
func TestCosCTExtendsClause15CollapsedOverAnonymousAncestor(t *testing.T) {
	s := xSchema(t, func(b *SchemaBuilder) {
		b.AddType(dSimple(t, uq("str"), AnyAtomicType()))
		b.AddType(oRedefiningRestriction(t, uq("mid"),
			[]AttributeUse{dAttr(t, uq("x"), uq("str"))}, []QName{uq("x")}))
		b.AddType(xType(t, uq("derived"), uq("mid"), EmptyContent{},
			[]AttributeUse{dAttr(t, uq("x"), uq("str"))}, nil))
	})
	def, _ := s.Type(uq("derived"))
	steps, a, ok := s.baseChainToAnyType(def.(ComplexType))
	if !ok || len(steps) != 2 {
		t.Fatalf("baseChainToAnyType(derived) = (%d steps, ok=%t), want the two steps derived and mid", len(steps), ok)
	}
	if a.Name() != (QName{}) {
		t.Fatalf("the ancestor whose {base type definition} is xs:anyType is %s, want the redefinition's ANONYMOUS clause-1.1 original", a.Name())
	}
	m, ok, err := s.collapsedExtension(def.(ComplexType))
	if err != nil || !ok {
		t.Fatalf("collapsedExtension(derived) = (ok=%t, err=%v), want a synthesized intermediate over the anonymous ancestor", ok, err)
	}
	inline, owned := m.Base().(InlineTypeDefinition)
	if !owned {
		t.Fatalf("the collapsed intermediate's {base type definition} is %#v, want the InlineTypeDefinition arm carrying the anonymous ancestor", m.Base())
	}
	base, isComplex := inline.Definition.(ComplexType)
	if !isComplex || base.Name() != (QName{}) {
		t.Fatalf("the collapsed intermediate's owned base is %#v, want the anonymous ancestor itself", inline.Definition)
	}
	if _, present := m.Context(); m.Name() != (QName{}) || !present {
		t.Fatalf("the collapsed intermediate is named %s with {context} present=%t, want §3.4.1's XOR satisfied on the {context} side as the anonymous ancestor's is", m.Name(), present)
	}
	var names []string
	for _, u := range m.AttributeUses() {
		names = append(names, attributeUseName(u).Local)
	}
	// The ancestor's @x alone: the redefinition's prohibition is not replayed, and
	// derived's own re-declaration is dropped rather than duplicated, because no
	// legal intermediate can hold two uses for one name (collapsedAttributeUses).
	if !fEqual(names, []string{"x"}) {
		t.Fatalf("the collapsed intermediate's {attribute uses} = %v, want [x]", names)
	}
}
