package validate

import (
	"slices"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// The fixtures below drive the RECURSIVE half of cvc-assess-elt (§3.3.4.6)
// clause 3: a child of an already-assessed element is assessed against the
// ·governing element declaration· the parent's own content model ·attributed·
// it to (§3.4.4.4), one arm per [xsd.Attribution] variant. Every schema is
// built through the exported constructors and FINALIZED, so each content model
// is one cos-nonambig admitted and each substitution group edge is one
// e-props-correct clause 4 admitted.

// dType builds a NAMED complex type over content, carrying uses, derived from
// base by method. An empty base name leaves {base type definition} ·absent·,
// which is what every fixture but the substitution pair wants.
func dType(t *testing.T, name, base string, method xsd.DerivationMethod, uses []xsd.AttributeUse, content xsd.ContentType) xsd.ComplexType {
	t.Helper()
	var baseName xsd.QName
	if base != "" {
		baseName = xsd.QName{Local: base}
	}
	ct, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Local: name}, baseName, nil,
		method, false, uses, nil, nil, content, nil, nil, nil)
	if err != nil {
		t.Fatalf("building the %s complex type: %v", name, err)
	}
	return ct
}

// dTyped wraps a LOCAL element declaration named local, whose {type definition}
// names typ, in a 1..1 particle scoped to the complex type named parent.
func dTyped(t *testing.T, parent, local, typ string) xsd.Particle {
	t.Helper()
	scope, err := xsd.NewLocalScope(xsderr.Loc{}, xsd.ComplexTypeScopeParent{Name: xsd.QName{Local: parent}})
	if err != nil {
		t.Fatalf("NewLocalScope: %v", err)
	}
	d, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: local},
		xsd.TypeDefinitionRef{Name: xsd.QName{Local: typ}}, nil, scope,
		nil, false, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("building the %s element declaration: %v", local, err)
	}
	return dParticleOver(t, xsd.ResolvedTerm{Term: d})
}

// dWildcard wraps a ·complete wildcard· with the given {process contents} in a
// 1..1 particle: {variety} any, so every name reaches cvc-wildcard's other
// clauses and none is turned away by the namespace constraint.
func dWildcard(t *testing.T, pc xsd.ProcessContents) xsd.Particle {
	t.Helper()
	c, err := xsd.NewNamespaceConstraint(xsderr.Loc{}, xsd.NamespaceConstraintAny, nil, nil, nil)
	if err != nil {
		t.Fatalf("building the namespace constraint: %v", err)
	}
	w, err := xsd.NewWildcard(xsderr.Loc{}, c, pc, nil)
	if err != nil {
		t.Fatalf("building the %s wildcard: %v", pc, err)
	}
	return dParticleOver(t, xsd.ResolvedTerm{Term: w})
}

// dParticleOver wraps term in a 1..1 particle.
func dParticleOver(t *testing.T, term xsd.TermOrRef) xsd.Particle {
	t.Helper()
	o, err := xsd.NewOccurs(xsderr.Loc{}, 1, 1)
	if err != nil {
		t.Fatalf("NewOccurs: %v", err)
	}
	p, err := xsd.NewParticle(xsderr.Loc{}, o, term, nil)
	if err != nil {
		t.Fatalf("NewParticle: %v", err)
	}
	return p
}

// dTopLevel is a top-level element declaration named local whose {type
// definition} names typ, with the given {substitution group affiliations}.
func dTopLevel(t *testing.T, local, typ string, affiliations ...xsd.QName) xsd.ElementDeclaration {
	t.Helper()
	d, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: local},
		xsd.TypeDefinitionRef{Name: xsd.QName{Local: typ}}, nil, xsd.NewGlobalScope(),
		nil, false, nil, affiliations, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("building the top-level %s element declaration: %v", local, err)
	}
	return d
}

// dSchema declares "root" over a RootType whose {content type} is an
// element-only sequence of particles, seeding whatever else the fixture needs
// through extra.
func dSchema(t *testing.T, extra func(*xsd.SchemaBuilder), particles ...xsd.Particle) *xsd.Schema {
	t.Helper()
	return cSchemaFrom(t, dType(t, "RootType", "", xsd.DerivationRestriction, nil,
		cSequence(t, false, particles...)), extra)
}

// dElem is an element information item named local at line, holding kids.
func dElem(local string, line int, kids ...Child) *testElement {
	return &testElement{name: xsd.QName{Local: local}, kids: kids, loc: loc(line, 1)}
}

// A child ·attributed to· an element particle is assessed against that
// particle's {term}, its own [[children]] against that declaration's {content
// type} — recursively, so a defect TWO levels below the validation root is
// charged at its own position (cvc-assess-elt clause 3.1 over key-governing-ed
// clause 2).
func TestDescendantContentIsAssessedAtDepth(t *testing.T) {
	schema := dSchema(t, func(b *xsd.SchemaBuilder) {
		b.AddType(dType(t, "KidType", "", xsd.DerivationRestriction, nil,
			cSequence(t, false, dTyped(t, "KidType", "grand", "GrandType"))))
		b.AddType(dType(t, "GrandType", "", xsd.DerivationRestriction, nil,
			cSequence(t, false, cParticle(t, "leaf", 1, 1))))
	}, dTyped(t, "RootType", "kid", "KidType"))

	// <root><kid><grand><leaf/></grand></kid></root>: every level's sequence is
	// exactly what its particle admits.
	wantSilence(t, cAssess(t, schema, dElem("root", 1,
		ElementChild(dElem("kid", 2, ElementChild(dElem("grand", 3,
			ElementChild(dElem("leaf", 4)))))))),
		"each descendant's [[children]] satisfy its ·governing type definition·")

	// The same tree with <grand> empty: GrandType's particle wants a <leaf>,
	// and the charge carries the position it ran out at — two levels down.
	got := cAssess(t, schema, dElem("root", 1,
		ElementChild(dElem("kid", 2, ElementChild(dElem("grand", 3))))))

	wantContentCharge(t, got, "cvc-complex-content", "1", loc(3, 1))
}

// The attribute half descends with it: a descendant's {required} attribute uses
// are cvc-complex-type clause 3's exactly as the root's are.
func TestDescendantAttributesAreAssessedAtDepth(t *testing.T) {
	schema := dSchema(t, func(b *xsd.SchemaBuilder) {
		b.AddType(dType(t, "KidType", "", xsd.DerivationRestriction,
			[]xsd.AttributeUse{aUse(t, "req", true, nil)}, xsd.EmptyContent{}))
	}, dTyped(t, "RootType", "kid", "KidType"))

	kid := dElem("kid", 2)
	kid.attrs = []Attribute{&testAttribute{name: local("req"), value: "v", loc: loc(2, 10)}}
	wantSilence(t, cAssess(t, schema, dElem("root", 1, ElementChild(kid))),
		"the descendant carries the attribute its {required} use names")

	got := cAssess(t, schema, dElem("root", 1, ElementChild(dElem("kid", 2))))

	wantContentCharge(t, got, "cvc-complex-type", "3", loc(2, 1))
	if !strings.Contains(got[0].Msg, "req") {
		t.Errorf("Msg = %q, want it to name the missing attribute", got[0].Msg)
	}
}

// A child admitted through cvc-accept clause 2.3.2 is assessed against the
// ·substituting declaration· and never against the particle's own D: the
// ·context-determined declaration· of a substituting item is the member's
// (§3.4.4.4 over key-governing-ed clause 2), and the head's {content type}
// would reject content the member's admits.
func TestDescendantSubstitutedForItsHeadIsAssessedAgainstTheMember(t *testing.T) {
	head := dType(t, "HeadType", "", xsd.DerivationRestriction, nil, xsd.EmptyContent{})
	schema := dSchema(t, func(b *xsd.SchemaBuilder) {
		b.AddType(head)
		b.AddType(dType(t, "MemberType", "HeadType", xsd.DerivationExtension, nil,
			cSequence(t, false, cParticle(t, "grand", 1, 1))))
		b.AddElement(dTopLevel(t, "kid", "HeadType"))
		b.AddElement(dTopLevel(t, "member", "MemberType", local("kid")))
	}, dParticleOver(t, xsd.ElementDeclarationRef{Name: local("kid")}))

	// HeadType's {content type} is empty and MemberType's admits one <grand>,
	// so this tree is silent only if the MEMBER's type governed.
	wantSilence(t, cAssess(t, schema, dElem("root", 1,
		ElementChild(dElem("member", 2, ElementChild(dElem("grand", 3)))))),
		"the ·substituting declaration· governs, not the particle's head")

	got := cAssess(t, schema, dElem("root", 1, ElementChild(dElem("member", 2))))

	wantContentCharge(t, got, "cvc-complex-content", "1", loc(2, 1))
}

// A child ·attributed to· a strict or a lax Wildcard has its ·expanded name·
// ·resolved· against the schema's top-level element declarations
// (key-governing-ed clause 3, cvc-resolve-instance §3.17.6.3) and is assessed
// against what that resolution found. §3.10.4.1 draws no distinction between
// the two in the resolution step, so the two arms decide alike.
func TestDescendantAttributedToAResolvingWildcardIsAssessed(t *testing.T) {
	for _, pc := range []xsd.ProcessContents{xsd.ProcessStrict, xsd.ProcessLax} {
		t.Run(pc.String(), func(t *testing.T) {
			schema := dSchema(t, func(b *xsd.SchemaBuilder) {
				b.AddType(dType(t, "KidType", "", xsd.DerivationRestriction, nil,
					cSequence(t, false, cParticle(t, "grand", 1, 1))))
				b.AddElement(dTopLevel(t, "kid", "KidType"))
			}, dWildcard(t, pc))

			wantSilence(t, cAssess(t, schema, dElem("root", 1,
				ElementChild(dElem("kid", 2, ElementChild(dElem("grand", 3)))))),
				"the resolved declaration's {content type} admits the sequence")

			got := cAssess(t, schema, dElem("root", 1, ElementChild(dElem("kid", 2))))
			wantContentCharge(t, got, "cvc-complex-content", "1", loc(2, 1))
		})
	}
}

// A name that ·resolves· to no top-level declaration leaves the child with no
// ·governing element declaration·, which is cvc-assess-elt clause 3.3 — ·laxly
// assessed· against xs:anyType — and no charge of its own, under a STRICT
// wildcard as much as under a lax one: what an unresolved name under a strict
// wildcard costs is the parent's [validity] (§3.3.5.1 clause 1.1.3), a property
// this package computes for no item.
func TestDescendantAttributedToAWildcardDeclinesAnUnresolvedName(t *testing.T) {
	for _, pc := range []xsd.ProcessContents{xsd.ProcessStrict, xsd.ProcessLax} {
		t.Run(pc.String(), func(t *testing.T) {
			schema := dSchema(t, nil, dWildcard(t, pc))

			wantSilence(t, cAssess(t, schema, dElem("root", 1,
				ElementChild(dElem("stranger", 2, ElementChild(dElem("anything", 3)))))),
				"an unresolved name is not a charge against the child")
		})
	}
}

// A child ·attributed to· a skip Wildcard is not ·assessed· at all (clause
// 3.2), and neither is any element below it: ·skipped· (§3.10.4.1 key-skipped)
// holds for every descendant of a skipped item, which the walk gets by not
// descending at all. The log is what distinguishes this from clause 3.3 — both
// charge nothing, and only one of them leaves the subtree unvisited.
func TestDescendantAttributedToASkipWildcardIsNotAssessed(t *testing.T) {
	schema := dSchema(t, func(b *xsd.SchemaBuilder) {
		b.AddType(dType(t, "KidType", "", xsd.DerivationRestriction, nil,
			cSequence(t, false, cParticle(t, "grand", 1, 1))))
		b.AddElement(dTopLevel(t, "kid", "KidType"))
	}, dWildcard(t, xsd.ProcessSkip))

	log, visits := recordingLogger()
	v, err := New(schema, testBackend(), WithLogger(log))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// <kid> is missing the <grand> KidType requires, and carries a child of its
	// own that no declaration admits: a skipped subtree is assessed for none of it.
	res := v.Assess(dElem("root", 1, ElementChild(dElem("kid", 2, ElementChild(dElem("junk", 3))))))

	wantSilence(t, res.Violations(), "a ·skipped· item's schema-validity is not assessed")
	want := []string{
		"assessing element validate.name=root validate.loc=instance.xml:1:1",
		"assessing content validate.name=kid validate.loc=instance.xml:2:1 " +
			"validate.rule=cvc-complex-content validate.clause=1 validate.outcome=attributed to wildcard any",
		"assessing element validate.name=kid validate.loc=instance.xml:2:1 " +
			"validate.rule=cvc-assess-elt validate.clause=3.2 validate.outcome=skipped",
		"assessing content validate.name=root validate.loc=instance.xml:1:1 " +
			"validate.rule=cvc-complex-content validate.clause=1 validate.outcome=accepted",
	}
	if !slices.Equal(*visits, want) {
		t.Errorf("walk visited\n\t%s\nwant\n\t%s",
			strings.Join(*visits, "\n\t"), strings.Join(want, "\n\t"))
	}
}

// An element whose own ·governing type definition· was not determined attributes
// its [[children]] to nothing, so the descent stops being typed there rather
// than resuming further down (clause 3.3 all the way down).
func TestDescendantOfAnUngovernedElementIsAssessedAgainstNothing(t *testing.T) {
	schema := dSchema(t, func(b *xsd.SchemaBuilder) {
		b.AddType(dType(t, "KidType", "", xsd.DerivationRestriction, nil,
			cSequence(t, false, cParticle(t, "grand", 1, 1))))
		b.AddElement(dTopLevel(t, "kid", "KidType"))
	}, cParticle(t, "opaque", 1, 1))

	// <opaque> is a local declaration with an ·absent· {type definition}, so
	// nothing below it is governed — including a <kid> a top-level declaration
	// would otherwise have governed.
	wantSilence(t, cAssess(t, schema, dElem("root", 1,
		ElementChild(dElem("opaque", 2, ElementChild(dElem("kid", 3)))))),
		"an ungoverned element attributes its [[children]] to nothing")
}
