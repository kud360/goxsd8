package parser

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/xsd"
)

// This test is package-internal because what it pins is the PRODUCER's output
// before Finalize, which no exported surface shows: a complex type naming an
// <attributeGroup> is handed the REFERENCE, never a re-mapping of the group's
// source elements (#479), and the group's own component — built once, under the
// DECLARING document's producer — is the only place the group's attributes are
// declared. §3.4.2.4 clause c-add2 unions the ALREADY-RESOLVED component
// property, so after Finalize the referencing type holds exactly those
// declarations; TestParseAttributeGroupFoldedUnderItsOwnProducer pins that half
// through Parse.
//
// A chameleon fixture makes the distinction observable: every property under
// test is decided by the declaring document, so a re-mapping under root.xsd's
// producer would mint {}a typed {}Local — the #368 bug — where the component
// holds {urn:x}a typed {urn:x}Local.
func TestAttributeGroupReferenceIsNotRemapped(t *testing.T) {
	const xs = `xmlns:xs="http://www.w3.org/2001/XMLSchema"`
	const chameleon = `<xs:schema ` + xs + ` attributeFormDefault="qualified">` +
		`<xs:simpleType name="Local"><xs:restriction base="xs:string"/></xs:simpleType>` +
		`<xs:attributeGroup name="G"><xs:attribute name="a" type="Local"/></xs:attributeGroup>` +
		`</xs:schema>`
	const root = `<xs:schema ` + xs + ` targetNamespace="urn:x" xmlns:tns="urn:x">` +
		`<xs:complexType name="T"><xs:sequence/><xs:attributeGroup ref="tns:G"/></xs:complexType>` +
		`</xs:schema>`

	baseDoc, err := ReadDocument("mem://base.xsd", strings.NewReader(chameleon))
	if err != nil {
		t.Fatalf("ReadDocument(base): %v", err)
	}
	rootDoc, err := ReadDocument("mem://root.xsd", strings.NewReader(root))
	if err != nil {
		t.Fatalf("ReadDocument(root): %v", err)
	}
	builder := xsd.NewSchemaBuilder()
	sym, err := newSymbols(builder, strict.New())
	if err != nil {
		t.Fatalf("newSymbols: %v", err)
	}
	// base.xsd is <include>d by root.xsd, so its effective target namespace is
	// root's (§4.2.3 clause 2.3) while it declares none of its own — the chameleon
	// state, mirroring what compile() wires up.
	basep := newProducer(baseDoc, "urn:x", nil, nil, nil, builder, sym)
	rootp := newProducer(rootDoc, "urn:x", nil, nil, nil, builder, sym)
	if err := basep.prescan(); err != nil {
		t.Fatalf("prescan(base): %v", err)
	}
	if err := rootp.prescan(); err != nil {
		t.Fatalf("prescan(root): %v", err)
	}

	gName := xsd.QName{Space: "urn:x", Local: "G"}
	group := childElement(basep.schemaElem, xsd.XMLSchemaNS, "attributeGroup")
	ag, err := basep.buildAttributeGroup(gName, group)
	if err != nil {
		t.Fatalf("buildAttributeGroup: %v", err)
	}
	tName := xsd.QName{Space: "urn:x", Local: "T"}
	ctElem := childElement(rootp.schemaElem, xsd.XMLSchemaNS, "complexType")
	content, _, wildcard, err := rootp.produceAttributeUses(ctElem, ctElem,
		attributeScopeParentOf(namedComplexType{name: tName}))
	if err != nil {
		t.Fatalf("produceAttributeUses: %v", err)
	}

	if len(content) != 1 || content[0] != xsd.AttributeUseOrGroupRef(xsd.AttributeGroupRef{Name: gName}) {
		t.Fatalf("complex type T's attribute content = %#v, want the one reference to {urn:x}G and nothing mapped from G's body", content)
	}
	if wildcard != nil {
		t.Errorf("complex type T's local wildcard = %v, want none: T writes no <anyAttribute>, and G's is folded in at finalize", wildcard)
	}
	own := attributeUseNames(t, ag.AttributeUses())
	if len(own) != 1 || own[0] != "{urn:x}a:{urn:x}Local" {
		t.Fatalf("{urn:x}G's own {attribute uses} = %v, want one {urn:x}a typed {urn:x}Local", own)
	}
	// {scope}.{parent} (§3.2.1 sc_a): the <attribute> is a child of the
	// top-level <attributeGroup> and has no <complexType> ancestor, so §3.2.2.2
	// dcl.att.local makes its {parent} G, the one value every referrer inherits.
	want := xsd.AttributeScopeParent(xsd.AttributeGroupScopeParent{Name: gName})
	if got := attributeUseScopeParent(t, ag.AttributeUses()[0]); got != want {
		t.Errorf("{urn:x}G's own component scopes {urn:x}a to %#v, want %#v", got, want)
	}
}

// TestReferencedAttributeGroupAttributesScopedToGroup is the direct case on the
// ordinary (non-chameleon, single-document) path, read after Finalize: a complex
// type C references attribute group G, and the attribute G contributes to C's
// folded {attribute uses} reports G as its {scope}.{parent}, never C.
func TestReferencedAttributeGroupAttributesScopedToGroup(t *testing.T) {
	const doc = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:x" xmlns:tns="urn:x">` +
		`<xs:attributeGroup name="G"><xs:attribute name="a" type="xs:string"/></xs:attributeGroup>` +
		`<xs:complexType name="C"><xs:sequence/><xs:attribute name="own" type="xs:string"/><xs:attributeGroup ref="tns:G"/></xs:complexType>` +
		`</xs:schema>`
	uses := complexTypeAttributeUses(t, doc, "C")
	if len(uses) != 2 {
		t.Fatalf("complex type C folded %d attribute uses, want 2 (its own and G's)", len(uses))
	}
	// Document order: C's own <attribute> precedes the <attributeGroup ref>.
	if got, want := attributeUseScopeParent(t, uses[0]), xsd.AttributeScopeParent(xsd.AttributeComplexTypeScopeParent{Name: xsd.QName{Space: "urn:x", Local: "C"}}); got != want {
		t.Errorf("C's own attribute is scoped to %#v, want %#v", got, want)
	}
	if got, want := attributeUseScopeParent(t, uses[1]), xsd.AttributeScopeParent(xsd.AttributeGroupScopeParent{Name: xsd.QName{Space: "urn:x", Local: "G"}}); got != want {
		t.Errorf("the attribute reached through <attributeGroup ref> is scoped to %#v, want %#v — §3.2.2.2 dcl.att.local reads {parent} off the <attribute>'s OWN ancestor axis, which holds no <complexType>", got, want)
	}
}

// complexTypeAttributeUses produces and finalizes a single-document schema and
// returns the {attribute uses} of its named top-level complex type.
func complexTypeAttributeUses(t *testing.T, doc, name string) []xsd.AttributeUse {
	t.Helper()
	parsed, err := ReadDocument("mem://scope.xsd", strings.NewReader(doc))
	if err != nil {
		t.Fatalf("ReadDocument: %v", err)
	}
	s, err := Produce(parsed, strict.New())
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	def, ok := s.Type(xsd.QName{Space: "urn:x", Local: name})
	if !ok {
		t.Fatalf("no top-level type {urn:x}%s in the fixture", name)
	}
	ct, ok := def.(xsd.ComplexType)
	if !ok {
		t.Fatalf("{urn:x}%s is a %T, want a complex type", name, def)
	}
	return ct.AttributeUses()
}

// attributeUseScopeParent reports the {scope}.{parent} of a use's sibling local
// Attribute Declaration; a use of any other shape, or a declaration with no
// {parent}, fails the test — neither can occur in these fixtures.
func attributeUseScopeParent(t *testing.T, u xsd.AttributeUse) xsd.AttributeScopeParent {
	t.Helper()
	local, ok := u.AttributeDeclaration().(xsd.LocalAttributeDeclaration)
	if !ok {
		t.Fatalf("attribute use declaration is %T, want a local declaration", u.AttributeDeclaration())
	}
	parent, ok := local.Declaration.Scope().Parent()
	if !ok {
		t.Fatalf("attribute %s has an absent {scope}.{parent}, but §3.2.2.2 dcl.att.local makes it Required for a local declaration", local.Declaration.Name())
	}
	return parent
}

// attributeUseNames renders each attribute use as "name:type" so a use's
// expanded name and type compare as one plain string; a use that is not a local
// declaration with a by-name type fails the test, since neither shape can occur
// in this fixture.
func attributeUseNames(t *testing.T, uses []xsd.AttributeUse) []string {
	t.Helper()
	names := make([]string, 0, len(uses))
	for _, u := range uses {
		local, ok := u.AttributeDeclaration().(xsd.LocalAttributeDeclaration)
		if !ok {
			t.Fatalf("attribute use declaration is %T, want a local declaration", u.AttributeDeclaration())
		}
		ref, ok := local.Declaration.TypeDefinition().(xsd.TypeDefinitionRef)
		if !ok {
			t.Fatalf("attribute %s {type definition} is %T, want a by-name reference", local.Declaration.Name(), local.Declaration.TypeDefinition())
		}
		names = append(names, local.Declaration.Name().String()+":"+ref.Name.String())
	}
	return names
}
