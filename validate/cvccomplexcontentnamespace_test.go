package validate_test

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/validate"
	"github.com/kud360/goxsd8/validate/xmlsrc"
)

// These rows pin the clause a cvc-complex-content charge at a rejected item
// adds where the item's local name matches an expected element declaration
// whose namespace name differs (namespaceMismatch), as the whole Error() text
// of a schema and instance read through parser and xmlsrc.

// nsOrderXSD is a urn:o schema with no elementFormDefault and the local item
// inside order's anonymous complex type: item is in no namespace.
const nsOrderXSD = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:o">
  <xs:element name="order"><xs:complexType><xs:sequence>
    <xs:element name="item" type="xs:string"/>
  </xs:sequence></xs:complexType></xs:element>
</xs:schema>`

// nsCharge is the one charge xmlsrc reports for instance against the schema
// docs serves at "o.xsd", as its Error() text.
func nsCharge(t *testing.T, docs map[string]string, instance string) string {
	t.Helper()
	backend := strict.New()
	schema, err := parser.Parse("o.xsd", parser.WithBackend(backend), parser.WithResolver(loader.Map(docs)))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	v, err := validate.New(schema, backend)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	res, err := xmlsrc.Validate(v, strings.NewReader(instance), xmlsrc.WithURI("o.xml"))
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	got := res.Violations()
	if len(got) != 1 {
		t.Fatalf("Violations() = %v, want exactly one", got)
	}
	return got[0].Error()
}

// nsChain is the clause's dcl.elt.local half for the local declaration item.
const nsChain = `, and element declaration item is local, so it takes a {target namespace} only from its own targetNamespace attribute, else from form="qualified" or the schema's elementFormDefault="qualified", and those two take the targetNamespace of its schema document if it has one`

func TestContentChargeNamesANamespaceMismatch(t *testing.T) {
	for _, tc := range []struct {
		name     string
		docs     map[string]string
		instance string
		want     string
	}{
		// The Fix row: a local declaration in no namespace, under an
		// anonymous {scope}.{parent}, gets the mismatch and the chain.
		{"an unqualified local declaration", map[string]string{"o.xsd": nsOrderXSD},
			`<order xmlns="urn:o"><item>a</item></order>`,
			`o.xml:1:22: [cvc-complex-content] element {urn:o}item, a child of {urn:o}order, is not allowed at its position: no particle of the parent's {content type} admits it there, where the content model expects element declaration item; the child's local name matches element declaration item, which is in no namespace, while the child is in namespace "urn:o"` + nsChain + `, but cvc-complex-content clause 1 requires the [[children]] to be ·valid· with respect to that {content type}`},
		// Guard (a): no expected declaration has the child's local name.
		{"no local name matches", map[string]string{"o.xsd": nsOrderXSD},
			`<order xmlns="urn:o"><other>a</other></order>`,
			`o.xml:1:22: [cvc-complex-content] element {urn:o}other, a child of {urn:o}order, is not allowed at its position: no particle of the parent's {content type} admits it there, where the content model expects element declaration item, but cvc-complex-content clause 1 requires the [[children]] to be ·valid· with respect to that {content type}`},
		// Guard (b): a qualified declaration gets the mismatch, not the chain.
		{"a qualified local declaration", map[string]string{"o.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:o" elementFormDefault="qualified">
  <xs:element name="order"><xs:complexType><xs:sequence>
    <xs:element name="item" type="xs:string"/>
  </xs:sequence></xs:complexType></xs:element>
</xs:schema>`},
			`<order xmlns="urn:o"><item xmlns="urn:x">a</item></order>`,
			`o.xml:1:22: [cvc-complex-content] element {urn:x}item, a child of {urn:o}order, is not allowed at its position: no particle of the parent's {content type} admits it there, where the content model expects element declaration {urn:o}item; the child's local name matches element declaration {urn:o}item, which is in namespace "urn:o", while the child is in namespace "urn:x", but cvc-complex-content clause 1 requires the [[children]] to be ·valid· with respect to that {content type}`},
		// Guard (c): the {open content} charge carries the same clause.
		{"under an open content wildcard", map[string]string{"o.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:o">
  <xs:element name="order"><xs:complexType>
    <xs:openContent mode="interleave"><xs:any namespace="urn:open" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:element name="item" type="xs:string"/></xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`},
			`<order xmlns="urn:o"><item>a</item></order>`,
			`o.xml:1:22: [cvc-complex-content] element {urn:o}item, a child of {urn:o}order, is not allowed at its position: no particle of the parent's {content type} admits it there, and neither does that {content type}'s {open content} wildcard, where the content model expects one of element declaration item, open content wildcard enumeration "urn:open"; the child's local name matches element declaration item, which is in no namespace, while the child is in namespace "urn:o"` + nsChain + `, but cvc-complex-content clause 3 requires the [[children]] to be ·valid· with respect to that {content type}`},
		// Guard (d): item's {scope}.{parent} is the type T named in no
		// namespace, so its schema document has no targetNamespace and the
		// chain is withheld, though the parent element is in urn:o.
		{"a local declaration of a no-namespace named type", map[string]string{
			"o.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:o">
  <xs:import schemaLocation="n.xsd"/>
  <xs:element name="order" type="T"/>
</xs:schema>`,
			"n.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="T"><xs:sequence>
    <xs:element name="item" type="xs:string"/>
  </xs:sequence></xs:complexType>
</xs:schema>`},
			`<order xmlns="urn:o"><item>a</item></order>`,
			`o.xml:1:22: [cvc-complex-content] element {urn:o}item, a child of {urn:o}order, is not allowed at its position: no particle of the parent's {content type} admits it there, where the content model expects element declaration item; the child's local name matches element declaration item, which is in no namespace, while the child is in namespace "urn:o", but cvc-complex-content clause 1 requires the [[children]] to be ·valid· with respect to that {content type}`},
		// The same for a {scope}.{parent} that is a model group definition
		// named in no namespace, reached through a <group ref>.
		{"a local declaration of a no-namespace model group", map[string]string{
			"o.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:o">
  <xs:import schemaLocation="n.xsd"/>
  <xs:element name="order"><xs:complexType><xs:group ref="G"/></xs:complexType></xs:element>
</xs:schema>`,
			"n.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="G"><xs:sequence>
    <xs:element name="item" type="xs:string"/>
  </xs:sequence></xs:group>
</xs:schema>`},
			`<order xmlns="urn:o"><item>a</item></order>`,
			`o.xml:1:22: [cvc-complex-content] element {urn:o}item, a child of {urn:o}order, is not allowed at its position: no particle of the parent's {content type} admits it there, where the content model expects element declaration item; the child's local name matches element declaration item, which is in no namespace, while the child is in namespace "urn:o", but cvc-complex-content clause 1 requires the [[children]] to be ·valid· with respect to that {content type}`},
		// Two expected declarations share the child's local name, each in a
		// namespace other than the child's: both are named.
		{"two matching declarations", map[string]string{"o.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:o" xmlns:o="urn:o">
  <xs:element name="item" type="xs:string"/>
  <xs:element name="order"><xs:complexType><xs:choice>
    <xs:element name="item" type="xs:string"/>
    <xs:element ref="o:item"/>
  </xs:choice></xs:complexType></xs:element>
</xs:schema>`},
			`<order xmlns="urn:o"><item xmlns="urn:x">a</item></order>`,
			`o.xml:1:22: [cvc-complex-content] element {urn:x}item, a child of {urn:o}order, is not allowed at its position: no particle of the parent's {content type} admits it there, where the content model expects one of element declaration item, element declaration {urn:o}item; the child's local name matches element declaration item, which is in no namespace, and element declaration {urn:o}item, which is in namespace "urn:o", while the child is in namespace "urn:x"` + nsChain + `, but cvc-complex-content clause 1 requires the [[children]] to be ·valid· with respect to that {content type}`},
		// A matching declaration past maxExpectedListed is named by the
		// clause though the expected list only counts it.
		{"a matching declaration past the bound", map[string]string{"o.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:o">
  <xs:element name="order"><xs:complexType><xs:choice>
    <xs:element name="a"/><xs:element name="b"/><xs:element name="c"/>
    <xs:element name="d"/><xs:element name="e"/><xs:element name="item"/>
  </xs:choice></xs:complexType></xs:element>
</xs:schema>`},
			`<order xmlns="urn:o"><item>a</item></order>`,
			`o.xml:1:22: [cvc-complex-content] element {urn:o}item, a child of {urn:o}order, is not allowed at its position: no particle of the parent's {content type} admits it there, where the content model expects one of element declaration a, element declaration b, element declaration c, element declaration d, element declaration e and 1 more; the child's local name matches element declaration item, which is in no namespace, while the child is in namespace "urn:o"` + nsChain + `, but cvc-complex-content clause 1 requires the [[children]] to be ·valid· with respect to that {content type}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := nsCharge(t, tc.docs, tc.instance); got != tc.want {
				t.Errorf("Error() =\n\t%s\nwant\n\t%s", got, tc.want)
			}
		})
	}
}
