package xsd_test

import (
	"fmt"
	"io"
	"strings"

	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/xsd"
)

// Example_contentModelRefs is Example_contentModelTraversal's sibling over a
// PARSED schema: it walks from a global element's complex type to the particles
// whose terms are an ElementDeclarationRef and a ModelGroupRef, prints each
// reference's Name and Loc, and follows each by the read-time lookup its type's
// doc names. Each Loc is the position parser.Parse read the ref= attribute at.
func Example_contentModelRefs() {
	const doc = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:po="urn:example:po" targetNamespace="urn:example:po">
  <xs:element name="purchaseOrder" type="po:PurchaseOrderType"/>
  <xs:element name="billTo" type="xs:string"/>
  <xs:group name="nameGroup">
    <xs:sequence><xs:element name="first" type="xs:string"/></xs:sequence>
  </xs:group>
  <xs:complexType name="PurchaseOrderType">
    <xs:sequence>
      <xs:element ref="po:billTo"/>
      <xs:group ref="po:nameGroup"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>
`
	resolve := loader.ResolverFunc(func(_, location string) (io.ReadCloser, string, error) {
		if location != "po.xsd" {
			return nil, "", fmt.Errorf("no document %q", location)
		}
		return io.NopCloser(strings.NewReader(doc)), location, nil
	})
	s, err := parser.Parse("po.xsd", parser.WithResolver(resolve))
	if err != nil {
		panic(err)
	}

	root, ok := s.Element(xsd.QName{Space: "urn:example:po", Local: "purchaseOrder"})
	if !ok {
		panic("purchaseOrder not declared")
	}
	rootType, byName := root.TypeDefinition().(xsd.TypeDefinitionRef)
	if !byName {
		panic("purchaseOrder's type is not a by-name reference")
	}
	def, ok := s.Type(rootType.Name)
	if !ok {
		panic("PurchaseOrderType not declared")
	}
	ct, ok := def.(xsd.ComplexType)
	if !ok {
		panic("PurchaseOrderType is not complex")
	}
	content, ok := ct.ContentType().(xsd.ElementContent)
	if !ok {
		panic("PurchaseOrderType has no particle")
	}
	top, ok := content.Particle.Term().(xsd.ResolvedTerm)
	if !ok {
		panic("PurchaseOrderType's particle is not an inline group")
	}
	seq, ok := top.Term.(xsd.ModelGroup)
	if !ok {
		panic("PurchaseOrderType's term is not a model group")
	}

	for _, p := range seq.Particles() {
		switch v := p.Term().(type) {
		case xsd.ElementDeclarationRef:
			target, found := s.Element(v.Name)
			if !found {
				panic("Finalize admitted an unresolvable element ref")
			}
			fmt.Println("element ref", v.Name, "at", v.Loc.String(), "->", target.Name().Local)
		case xsd.ModelGroupRef:
			target, found := s.ModelGroup(v.Name)
			if !found {
				panic("Finalize admitted an unresolvable group ref")
			}
			fmt.Println("group ref", v.Name, "at", v.Loc.String(), "->", target.ModelGroup().Compositor())
		case xsd.ResolvedTerm:
			fmt.Println("inline term")
		}
	}

	// Output:
	// element ref {urn:example:po}billTo at po.xsd:10:19 -> billTo
	// group ref {urn:example:po}nameGroup at po.xsd:11:17 -> sequence
}
