package parser

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/xsd"
)

// This test is package-internal because the mapping it pins is not observable
// from outside: §3.3.2.3 dcl.elt.local scopes an <element> within a named
// <group> to the MODEL GROUP DEFINITION, and xsd.Schema exposes no
// ModelGroupDefinition(QName) accessor to reach that component after Finalize
// (STYLE T5 — none is exported until a caller justifies it; see resolve.go's
// recorded follow-cost asymmetry). Calling produceModelGroupDefinition directly
// is the only way to read the produced declaration's {scope}.{parent}, and this
// branch is exactly where a silent bug would hide: it is the one place the
// threaded parent is a ModelGroupScopeParent rather than a
// ComplexTypeScopeParent.
func TestProduceModelGroupDefinitionScopesLocalElements(t *testing.T) {
	const doc = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:po">
		<xs:group name="g"><xs:sequence>
			<xs:element name="a" type="xs:string"/>
			<xs:choice><xs:element name="b" type="xs:string"/></xs:choice>
		</xs:sequence></xs:group>
	</xs:schema>`
	d, err := ReadDocument("mem://scope.xsd", strings.NewReader(doc))
	if err != nil {
		t.Fatalf("ReadDocument: %v", err)
	}
	builder := xsd.NewSchemaBuilder()
	sym, err := newSymbols(builder, strict.New())
	if err != nil {
		t.Fatalf("newSymbols: %v", err)
	}
	p := newProducer(d, "urn:po", nil, nil, nil, builder, sym)
	group := childElement(d.Root(), xsd.XMLSchemaNS, "group")
	if group == nil {
		t.Fatal("the test document has no top-level <group>")
	}

	name := xsd.QName{Space: "urn:po", Local: "g"}
	mgd, err := p.produceModelGroupDefinition(name, group)
	if err != nil {
		t.Fatalf("produceModelGroupDefinition: %v", err)
	}
	want := xsd.ModelGroupScopeParent{Name: name}

	parts := mgd.ModelGroup().Particles()
	if len(parts) != 2 {
		t.Fatalf("group g has %d particles, want 2", len(parts))
	}
	direct := scopeParentOfTerm(t, parts[0])
	if direct != want {
		t.Fatalf("element a {scope}.{parent} = %#v, want %#v", direct, want)
	}

	// A nested compositor inside the definition is not a scope boundary either.
	rt, ok := parts[1].Term().(xsd.ResolvedTerm)
	if !ok {
		t.Fatalf("the nested particle term is %T, want ResolvedTerm", parts[1].Term())
	}
	choice, ok := rt.Term.(xsd.ModelGroup)
	if !ok {
		t.Fatalf("the nested term is %T, want ModelGroup", rt.Term)
	}
	if deep := scopeParentOfTerm(t, choice.Particles()[0]); deep != want {
		t.Fatalf("element b {scope}.{parent} = %#v, want %#v", deep, want)
	}
}

// scopeParentOfTerm reads the {scope}.{parent} of the local element declaration
// a particle's {term} is, failing when the term is not one or the scope is
// global.
func scopeParentOfTerm(t *testing.T, p xsd.Particle) xsd.ElementScopeParent {
	t.Helper()
	rt, ok := p.Term().(xsd.ResolvedTerm)
	if !ok {
		t.Fatalf("particle term is %T, want an inline ResolvedTerm", p.Term())
	}
	ed, ok := rt.Term.(xsd.ElementDeclaration)
	if !ok {
		t.Fatalf("particle term is %T, want ElementDeclaration", rt.Term)
	}
	parent, ok := ed.Scope().Parent()
	if !ok {
		t.Fatalf("element %s has no {scope}.{parent}, want the containing model group definition", ed.Name())
	}
	return parent
}
