package validate

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// The fixtures below drive the CONTENT half of cvc-complex-type (§3.4.4.2) —
// clause 1's four sub-clauses and, through clause 1.4, cvc-complex-content
// (§3.4.4.3) — over the validation root, the only element whose ·governing
// type definition· this package determines. Schemas are built through the
// exported constructors and FINALIZED, so every content model here is one
// cos-nonambig admitted.

// cLocal builds a local element declaration named local with an absent {type
// definition}: the content model reads names and occurrence ranges, never the
// declarations' types.
func cLocal(t *testing.T, local string) xsd.ElementDeclaration {
	t.Helper()
	scope, err := xsd.NewLocalScope(xsderr.Loc{}, xsd.ComplexTypeScopeParent{Name: xsd.QName{Local: "RootType"}})
	if err != nil {
		t.Fatalf("NewLocalScope: %v", err)
	}
	d, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: local}, nil, nil, scope,
		nil, false, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("building the %s element declaration: %v", local, err)
	}
	return d
}

// cParticle wraps a local element declaration in a particle with the given
// occurrence range.
func cParticle(t *testing.T, local string, minOccurs, maxOccurs int) xsd.Particle {
	t.Helper()
	o, err := xsd.NewOccurs(xsderr.Loc{}, minOccurs, maxOccurs)
	if err != nil {
		t.Fatalf("NewOccurs(%d,%d): %v", minOccurs, maxOccurs, err)
	}
	p, err := xsd.NewParticle(xsderr.Loc{}, o, xsd.ResolvedTerm{Term: cLocal(t, local)}, nil)
	if err != nil {
		t.Fatalf("NewParticle: %v", err)
	}
	return p
}

// cSequence is an element-only or mixed {content type} over a <sequence> of
// the given particles.
func cSequence(t *testing.T, mixed bool, particles ...xsd.Particle) xsd.ContentType {
	t.Helper()
	g, err := xsd.NewModelGroup(xsderr.Loc{}, xsd.CompositorSequence, particles, nil)
	if err != nil {
		t.Fatalf("NewModelGroup: %v", err)
	}
	o, err := xsd.NewOccurs(xsderr.Loc{}, 1, 1)
	if err != nil {
		t.Fatalf("NewOccurs: %v", err)
	}
	p, err := xsd.NewParticle(xsderr.Loc{}, o, xsd.ResolvedTerm{Term: g}, nil)
	if err != nil {
		t.Fatalf("NewParticle: %v", err)
	}
	return xsd.ElementContent{Mixed: mixed, Particle: p}
}

// cSchema declares "root" governed by a NAMED complex type carrying content,
// and nothing else.
func cSchema(t *testing.T, content xsd.ContentType) *xsd.Schema {
	t.Helper()
	name := xsd.QName{Local: "RootType"}
	ct, err := xsd.NewComplexType(xsderr.Loc{}, name, xsd.QName{}, nil,
		xsd.DerivationRestriction, false, nil, nil, nil, content, nil, nil, nil)
	if err != nil {
		t.Fatalf("building the governing type: %v", err)
	}
	return cSchemaFrom(t, ct, nil)
}

// cSchemaFrom declares "root" over a NAMED governing type, seeding whatever
// else the fixture needs through extra.
func cSchemaFrom(t *testing.T, ct xsd.ComplexType, extra func(*xsd.SchemaBuilder)) *xsd.Schema {
	t.Helper()
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "root"},
		xsd.TypeDefinitionRef{Name: ct.Name()}, nil,
		xsd.NewGlobalScope(), nil, false, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("building the root element declaration: %v", err)
	}
	b := xsd.NewSchemaBuilder()
	if extra != nil {
		extra(b)
	}
	b.AddType(ct)
	b.AddElement(e)
	schema, err := b.Finalize()
	if err != nil {
		t.Fatalf("finalizing the content schema: %v", err)
	}
	return schema
}

// cRoot is a <root> whose [[children]] are the given items, each at its own
// Loc: a name spells an element child, a name prefixed with "#" the character
// data after the "#".
func cRoot(items ...string) *testElement {
	e := &testElement{name: xsd.QName{Local: "root"}, loc: loc(1, 1)}
	for i, item := range items {
		if text, isText := strings.CutPrefix(item, "#"); isText {
			e.kids = append(e.kids, TextChild(&testText{data: text, loc: loc(2+i, 1)}))
			continue
		}
		e.kids = append(e.kids, ElementChild(&testElement{
			name: xsd.QName{Local: item}, loc: loc(2+i, 1)}))
	}
	return e
}

// cAssess assesses root against schema and returns the violations charged.
func cAssess(t *testing.T, schema *xsd.Schema, root Element) []*xsderr.Error {
	t.Helper()
	v, err := New(schema, testBackend())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	res := v.Assess(root)
	if res.Err() != nil {
		t.Fatalf("Err() = %v, want nil", res.Err())
	}
	return res.Violations()
}

// wantContentCharge fails unless the violations are exactly one, carrying rule at loc
// and naming clause in its message.
func wantContentCharge(t *testing.T, got []*xsderr.Error, rule xsderr.Rule, clause string, at xsderr.Loc) {
	t.Helper()
	if len(got) != 1 {
		t.Fatalf("Violations() = %v, want exactly one", got)
	}
	if got[0].Rule != rule {
		t.Errorf("Rule = %q, want %q", got[0].Rule, rule)
	}
	if got[0].Loc != at {
		t.Errorf("Loc = %s, want the offending position %s", got[0].Loc, at)
	}
	if !strings.HasPrefix(got[0].Msg, "clause "+clause+":") {
		t.Errorf("Msg = %q, want it to charge clause %s", got[0].Msg, clause)
	}
}

// An element-only {content type} whose particle takes the whole sequence
// charges nothing (clause 1.4 over cvc-complex-content clause 1).
func TestContentMatchesAnElementOnlySequence(t *testing.T) {
	schema := cSchema(t, cSequence(t, false, cParticle(t, "a", 1, 1), cParticle(t, "b", 1, 2)))

	wantSilence(t, cAssess(t, schema, cRoot("a", "b", "b")), "the sequence is exactly what the particle admits")
}

// A child no particle admits at its position is charged against its OWN
// location, and the charge is cvc-complex-content, not cvc-complex-type.
func TestContentChargesAnUnadmittedChild(t *testing.T) {
	schema := cSchema(t, cSequence(t, false, cParticle(t, "a", 1, 1), cParticle(t, "b", 1, 1)))

	got := cAssess(t, schema, cRoot("a", "c"))

	wantContentCharge(t, got, "cvc-complex-content", "1", loc(3, 1))
	if !strings.Contains(got[0].Msg, "c") {
		t.Errorf("Msg = %q, want it to name the offending child", got[0].Msg)
	}
}

// A sequence that ends before a required particle is satisfied is charged
// against the CONTAINING element: there is no child at the offending position.
func TestContentChargesASequenceThatEndsShort(t *testing.T) {
	schema := cSchema(t, cSequence(t, false, cParticle(t, "a", 1, 1), cParticle(t, "b", 2, 2)))

	got := cAssess(t, schema, cRoot("a", "b"))

	wantContentCharge(t, got, "cvc-complex-content", "1", loc(1, 1))
}

// Clause 1.3 lets an element-only {content type} carry white space between its
// children, and clause 1.1 lets an empty one carry none — the whole of the
// difference between the two varieties (PRINCIPLES 13). This is the
// element-only direction.
func TestElementOnlyContentAdmitsWhiteSpace(t *testing.T) {
	schema := cSchema(t, cSequence(t, false, cParticle(t, "a", 1, 1)))

	wantSilence(t, cAssess(t, schema, cRoot("#\n  ", "a", "#\n")), "clause 1.3 excepts XML 1.1 white space")
}

// The empty direction of the same pair: an empty {content type} admits no
// character information item at all, white space included.
func TestEmptyContentAdmitsNoWhiteSpace(t *testing.T) {
	schema := cSchema(t, xsd.EmptyContent{})

	got := cAssess(t, schema, cRoot("#   "))

	wantContentCharge(t, got, "cvc-complex-type", "1.1", loc(2, 1))
}

// Character data that is not white space is charged against an element-only
// {content type} (clause 1.3), at the run's own location.
func TestElementOnlyContentChargesCharacterData(t *testing.T) {
	schema := cSchema(t, cSequence(t, false, cParticle(t, "a", 1, 1)))

	got := cAssess(t, schema, cRoot("a", "#text"))

	wantContentCharge(t, got, "cvc-complex-type", "1.3", loc(3, 1))
}

// A mixed {content type} restricts character content in no clause at all, and
// its element children are matched exactly as an element-only one's are.
func TestMixedContentAdmitsCharacterDataAndStillMatches(t *testing.T) {
	schema := cSchema(t, cSequence(t, true, cParticle(t, "a", 1, 1), cParticle(t, "b", 1, 1)))

	wantSilence(t, cAssess(t, schema, cRoot("#free", "a", "#text", "b")), "mixed restricts no character content")

	got := cAssess(t, schema, cRoot("#free", "b"))
	wantContentCharge(t, got, "cvc-complex-content", "1", loc(3, 1))
}

// An empty {content type} admits no element information item either (clause
// 1.1), charged at the child's own location.
func TestEmptyContentChargesAnElementChild(t *testing.T) {
	schema := cSchema(t, xsd.EmptyContent{})

	got := cAssess(t, schema, cRoot("a"))

	wantContentCharge(t, got, "cvc-complex-type", "1.1", loc(2, 1))
}

// A simple {content type} admits character data but no element information
// item [[children]] (clause 1.2).
func TestSimpleContentChargesAnElementChild(t *testing.T) {
	seeded, err := builtin.Seed(testBackend())
	if err != nil {
		t.Fatalf("seeding the builtin types: %v", err)
	}
	name := xsd.QName{Local: "RootType"}
	ct, err := xsd.NewComplexType(xsderr.Loc{}, name, xsd.QName{}, nil,
		xsd.DerivationRestriction, false, nil, nil, nil,
		xsd.SimpleContent{SimpleType: seeded[0]}, nil, nil, nil)
	if err != nil {
		t.Fatalf("building the governing type: %v", err)
	}
	schema := cSchemaFrom(t, ct, func(b *xsd.SchemaBuilder) {
		for _, st := range seeded {
			b.AddType(st)
		}
	})

	wantSilence(t, cAssess(t, schema, cRoot("#42")), "clause 1.2 restricts element children, not character data")

	got := cAssess(t, schema, cRoot("a"))
	wantContentCharge(t, got, "cvc-complex-type", "1.2", loc(2, 1))
}

// One element carries at most one content charge: the first, at the offending
// position. A second unadmitted child is not matched against a position the
// document never reached.
func TestContentChargesOneElementOnce(t *testing.T) {
	schema := cSchema(t, cSequence(t, false, cParticle(t, "a", 1, 1)))

	got := cAssess(t, schema, cRoot("c", "d", "#text"))

	wantContentCharge(t, got, "cvc-complex-content", "1", loc(2, 1))
}

// cvc-complex-type clause 1 applies only to an element that is not ·nilled·,
// which needs the declaration's {nillable} and the attribute's ·actual value·.
// The presence of xsi:nil is the decline.
func TestContentDeclinesANilledElement(t *testing.T) {
	schema := cSchema(t, cSequence(t, false, cParticle(t, "a", 1, 1)))
	root := cRoot("c")
	root.attrs = []Attribute{&testAttribute{
		name:  xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: "nil"},
		value: "true", loc: loc(1, 10)}}

	wantSilence(t, cAssess(t, schema, root), "an element carrying xsi:nil may be ·nilled·")
}

// An ANONYMOUS governing type decides its element's [[children]]: no finalize
// pass folds a {content type} the way §3.4.2.4 clause 3 folds {attribute
// uses}, so attributePropertiesFolded's decline is the attribute half's alone.
func TestAnonymousGoverningTypeStillDecidesContent(t *testing.T) {
	anonymous := func() *xsd.Schema {
		id := xsd.NewComponentID()
		ct, err := xsd.NewAnonymousComplexType(xsderr.Loc{},
			xsd.ElementDeclarationContext{Component: id}, xsd.QName{}, nil,
			xsd.DerivationRestriction, false, nil, nil, nil,
			cSequence(t, false, cParticle(t, "a", 1, 1)), nil, nil, nil)
		if err != nil {
			t.Fatalf("building the anonymous governing type: %v", err)
		}
		e, err := xsd.NewElementDeclarationOwningType(xsderr.Loc{}, id, xsd.QName{Local: "root"}, ct,
			nil, xsd.NewGlobalScope(), nil, false, nil, nil, nil, false, nil, nil)
		if err != nil {
			t.Fatalf("building the owning root element declaration: %v", err)
		}
		b := xsd.NewSchemaBuilder()
		b.AddElement(e)
		schema, err := b.Finalize()
		if err != nil {
			t.Fatalf("finalizing the anonymous content schema: %v", err)
		}
		return schema
	}

	wantSilence(t, cAssess(t, anonymous(), cRoot("a")), "the anonymous type's particle takes the sequence")

	got := cAssess(t, anonymous(), cRoot("c"))
	wantContentCharge(t, got, "cvc-complex-content", "1", loc(2, 1))
}

// A DESCENDANT's [[children]] are assessed against nothing: its governing type
// would come from the particle it is ·attributed to·, which this package does
// not thread into the descent.
func TestDescendantContentIsNotAssessed(t *testing.T) {
	schema := cSchema(t, cSequence(t, false, cParticle(t, "a", 1, 1)))
	root := cRoot("a")
	child := &testElement{
		name: xsd.QName{Local: "deep"},
		kids: []Child{TextChild(&testText{data: "anything", loc: loc(3, 1)})},
		loc:  loc(3, 1),
	}
	root.kids = append(root.kids, ElementChild(child))

	got := cAssess(t, schema, root)

	wantContentCharge(t, got, "cvc-complex-content", "1", loc(3, 1))
}
