package parser

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/xsd"
)

// This test is package-internal because the agreement it pins is not observable
// from outside: xsd.Schema exposes no AttributeGroup(QName) accessor (STYLE T5 —
// none is exported until a caller justifies it), so the top-level Attribute Group
// Definition component that run() hands to AddAttributeGroup cannot be read back
// after Finalize and compared with the copy folded into a referencing complex
// type. Driving the two producers directly is the only way to put the two
// foldings side by side.
//
// The two foldings are §3.6.2.1's {attribute uses} of the group itself and
// §3.4.2.4 clause c-add2's contribution of that same property to a complex type
// that names the group. c-add2 is a union of the ALREADY-RESOLVED component
// property, so the two must agree — an assembly holding a {urn:x}a in G's own
// component and a {}a in T's would be exactly the #368 bug, and would go
// unnoticed by any test that reads only one of the two sites.
func TestAttributeGroupComponentAndInlineFoldAgree(t *testing.T) {
	const xs = `xmlns:xs="http://www.w3.org/2001/XMLSchema"`
	// A chameleon: no targetNamespace of its own, its own attributeFormDefault,
	// and an unqualified type= naming a sibling — every property under test is
	// decided by THIS document, so a fold under root.xsd's producer disagrees.
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
	basep := newProducer(baseDoc, "urn:x", nil, builder, sym)
	rootp := newProducer(rootDoc, "urn:x", nil, builder, sym)
	basep.prescan()
	rootp.prescan()

	group := childElement(basep.schemaElem, xsd.XMLSchemaNS, "attributeGroup")
	ag, err := basep.buildAttributeGroup(xsd.QName{Space: "urn:x", Local: "G"}, group)
	if err != nil {
		t.Fatalf("buildAttributeGroup: %v", err)
	}
	inlined, _, _, err := rootp.produceAttributeUses(childElement(rootp.schemaElem, xsd.XMLSchemaNS, "complexType"))
	if err != nil {
		t.Fatalf("produceAttributeUses: %v", err)
	}

	own := attributeUseNames(t, ag.AttributeUses())
	folded := attributeUseNames(t, inlined)
	if len(own) != 1 || own[0] != "{urn:x}a:{urn:x}Local" {
		t.Fatalf("{urn:x}G's own {attribute uses} = %v, want one {urn:x}a typed {urn:x}Local", own)
	}
	if len(folded) != len(own) || folded[0] != own[0] {
		t.Errorf("complex type T folded %v but {urn:x}G's own component holds %v — c-add2 unions the component's already-resolved property, so the two foldings must agree", folded, own)
	}
}

// attributeUseNames renders each attribute use as "name:type" so two foldings of
// one group compare as plain strings; a use that is not a local declaration with
// a by-name type fails the test, since neither shape can occur in this fixture.
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
