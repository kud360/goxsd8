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
	if !strings.Contains(got[0].Msg, string(rule)+" clause "+clause) {
		t.Errorf("Msg = %q, want it to name %s clause %s inline (STYLE E4)", got[0].Msg, rule, clause)
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

// simpleContentSchema declares "root" over a NAMED complex type whose {content
// type} is simple, with the builtin named typ as its {simple type definition}.
// The builtins are SEEDED so the component the {content type} carries is the
// real one and its facets are the spec's.
func simpleContentSchema(t *testing.T, typ xsd.QName) *xsd.Schema {
	t.Helper()
	seeded, err := builtin.Seed(testBackend())
	if err != nil {
		t.Fatalf("seeding the builtin types: %v", err)
	}
	var st *xsd.SimpleType
	for _, s := range seeded {
		if s.Name() == typ {
			st = s
			break
		}
	}
	if st == nil {
		t.Fatalf("builtin.Seed produced no %s", typ)
	}
	ct, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Local: "RootType"}, xsd.QName{}, nil,
		xsd.DerivationRestriction, false, nil, nil, nil,
		xsd.SimpleContent{SimpleType: st}, nil, nil, nil)
	if err != nil {
		t.Fatalf("building the governing type: %v", err)
	}
	return cSchemaFrom(t, ct, func(b *xsd.SchemaBuilder) {
		for _, s := range seeded {
			b.AddType(s)
		}
	})
}

// A simple {content type} admits character data but no element information
// item [[children]] (clause 1.2). An element child is charged at its own
// position and the ·initial value· half is NOT charged behind it, whatever the
// character data holds: one element carries at most one content charge.
func TestSimpleContentChargesAnElementChild(t *testing.T) {
	schema := simpleContentSchema(t, integerType())

	wantSilence(t, cAssess(t, schema, cRoot("#42")), "clause 1.2 restricts element children, not character data")

	wantContentCharge(t, cAssess(t, schema, cRoot("a")), "cvc-complex-type", "1.2", loc(2, 1))
	wantContentCharge(t, cAssess(t, schema, cRoot("#abc", "a")), "cvc-complex-type", "1.2", loc(3, 1))
}

// The other half of clause 1.2: the ·initial value· is ·valid· with respect to
// the {content type}.{simple type definition} as String Valid (§3.16.4) defines
// it. The charge sits on the CONTAINING element, the value belonging to no one
// of its runs.
func TestSimpleContentValidatesTheInitialValue(t *testing.T) {
	schema := simpleContentSchema(t, integerType())

	wantSilence(t, cAssess(t, schema, cRoot("#42")), "42 is an xs:integer")
	// String Valid clause 1 normalizes before Datatype Valid runs, so xs:integer's
	// collapsing whiteSpace facet takes the padding off first.
	wantSilence(t, cAssess(t, schema, cRoot("#  42 ")),
		"whiteSpace normalization precedes datatype validation (String Valid clause 1)")

	got := cAssess(t, schema, cRoot("#abc"))
	wantContentCharge(t, got, "cvc-complex-type", "1.2", loc(1, 1))
	if !strings.Contains(got[0].Msg, "String Valid") {
		t.Errorf("Msg = %q, want the String Valid delegation named", got[0].Msg)
	}
}

// An element with NO character information item [[child]] DECLINES rather than
// being charged for the empty string: cvc-elt clause 5.1 has a declaration's
// {value constraint} supply the ·initial value· of an empty element, so the
// charge needs a dispatch this check cannot make (#716).
func TestSimpleContentDeclinesAnEmptyElement(t *testing.T) {
	schema := simpleContentSchema(t, integerType())

	wantSilence(t, cAssess(t, schema, cRoot()),
		"cvc-elt clause 5.1 may validate a default in place of the empty ·initial value·")
	// White space IS a character information item [[child]], so an element
	// carrying only white space is decided — and xs:integer's collapsing
	// whiteSpace facet leaves nothing an xs:integer admits.
	wantContentCharge(t, cAssess(t, schema, cRoot("#  ")), "cvc-complex-type", "1.2", loc(1, 1))
}

// The ·initial value· is the [[character code]] of EVERY character information
// item [[child]], concatenated in order — never the first run alone. An adapter
// splits the runs wherever a comment or processing instruction sits between
// them, neither of which is a [[child]] this engine sees.
func TestSimpleContentConcatenatesEveryTextRun(t *testing.T) {
	schema := simpleContentSchema(t, integerType())

	// Neither run is an xs:integer by itself; their concatenation is.
	wantSilence(t, cAssess(t, schema, cRoot("#-", "#42")), "the ·initial value· is -42")

	// The mirror: each run is an xs:integer by itself and the concatenation is not.
	wantContentCharge(t, cAssess(t, schema, cRoot("#4", "#-2")), "cvc-complex-type", "1.2", loc(1, 1))
}

// An ungoverned {simple type definition} DECLINES rather than charging: no
// backend maps xs:anySimpleType, and value.ValidateLexical reports that under
// cvc-datatype-valid exactly as it reports a real rejection (#774).
func TestSimpleContentDeclinesAnUngovernedType(t *testing.T) {
	schema := simpleContentSchema(t, xsd.QName{Space: xsd.XMLSchemaNS, Local: "anySimpleType"})

	wantSilence(t, cAssess(t, schema, cRoot("#anything at all")),
		"an ungoverned type is a backend gap, not a verdict about the ·initial value·")
}

// One element carries at most one content charge: the first, at the offending
// position. A second unadmitted child is not matched against a position the
// document never reached.
func TestContentChargesOneElementOnce(t *testing.T) {
	schema := cSchema(t, cSequence(t, false, cParticle(t, "a", 1, 1)))

	got := cAssess(t, schema, cRoot("c", "d", "#text"))

	wantContentCharge(t, got, "cvc-complex-content", "1", loc(2, 1))
}

// cvc-complex-type clause 1 applies only to an element that is not ·nilled·, so
// a ·nilled· element runs no clause 1 at all — its EMPTY [[children]] are not
// matched against the {content type} that would otherwise require an <a>.
func TestNilledElementRunsNoContentCheck(t *testing.T) {
	wantSilence(t, cAssess(t, nillableSchema(t, cSequence(t, false, cParticle(t, "a", 1, 1)), nil),
		nilledRoot(t, "true")), "a ·nilled· element is exempt from cvc-complex-type clause 1")
}

// An element carrying xsi:nil = FALSE is not ·nilled·, so clause 1 applies to it
// exactly as it does to an element carrying no xsi:nil at all: the presence of
// the attribute is not the decision, its ·actual value· is.
func TestNonNilledElementStillRunsTheContentCheck(t *testing.T) {
	got := cAssess(t, nillableSchema(t, cSequence(t, false, cParticle(t, "a", 1, 1)), nil),
		nilledRoot(t, "false"))

	wantContentCharge(t, got, "cvc-complex-content", "1", loc(1, 1))
}

// nillableSchema declares "root" over content with {nillable} true, and with the
// given {value constraint}.
func nillableSchema(t *testing.T, content xsd.ContentType, vc *xsd.ValueConstraint) *xsd.Schema {
	t.Helper()
	name := xsd.QName{Local: "RootType"}
	ct, err := xsd.NewComplexType(xsderr.Loc{}, name, xsd.QName{}, nil,
		xsd.DerivationRestriction, false, nil, nil, nil, content, nil, nil, nil)
	if err != nil {
		t.Fatalf("building the governing type: %v", err)
	}
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "root"},
		xsd.TypeDefinitionRef{Name: name}, nil, xsd.NewGlobalScope(), vc, true, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("building the root element declaration: %v", err)
	}
	b := xsd.NewSchemaBuilder()
	b.AddType(ct)
	b.AddElement(e)
	schema, err := b.Finalize()
	if err != nil {
		t.Fatalf("finalizing the nillable schema: %v", err)
	}
	return schema
}

// nilledRoot is an empty <root xsi:nil="..."/>.
func nilledRoot(t *testing.T, lexical string, kids ...string) *testElement {
	t.Helper()
	e := cRoot(kids...)
	e.attrs = []Attribute{&testAttribute{
		name:  xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: "nil"},
		value: lexical, loc: loc(1, 10)}}
	return e
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

// A child the content model REJECTED is ·attributed to· no particle, so its own
// [[children]] are assessed against nothing: the one charge is the parent's, at
// the child's position, and the subtree below it draws none.
func TestRejectedChildAttributesItsSubtreeToNothing(t *testing.T) {
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
