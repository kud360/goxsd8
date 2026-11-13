package parser_test

import (
	"errors"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// TestAttributeUseRejectionsCiteTheAttribute pins, end to end through Parse,
// that a finalize-time rejection of an attribute use is charged at the
// attribute and not at the enclosing <xs:complexType> or <xs:attributeGroup>:
// au-props-correct (§3.5.6) clauses 2 and 3 at the use's own <xs:attribute>
// start tag, which the producer threads into xsd.NewAttributeUse, and
// src-resolve (§3.17.6.2) for a dangling <attribute ref> at its ref= attribute,
// which the producer threads into the xsd.AttributeDeclarationRef. The
// inherited row's base lives in an included document, so the charge must name
// that document, not the deriving type's.
func TestAttributeUseRejectionsCiteTheAttribute(t *testing.T) {
	const head = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">` + "\n"
	const tail = "\n</xs:schema>"
	for _, tc := range []struct {
		name string
		docs map[string]string
		rule xsderr.Rule
		want xsderr.Loc
	}{
		{"clause 2 under an anonymous complex type", map[string]string{"main.xsd": head +
			`<xs:element name="r"><xs:complexType>` + "\n" +
			`  <xs:attribute name="k" type="xs:int" default="abc"/>` + "\n" +
			`</xs:complexType></xs:element>` + tail},
			"au-props-correct", xsderr.Loc{URI: "main.xsd", Line: 3, Col: 3}},
		{"clause 3 under an attribute group", map[string]string{"main.xsd": head +
			`<xs:attribute name="g" type="xs:int" fixed="1"/>` + "\n" +
			`<xs:attributeGroup name="ag">` + "\n" +
			`  <xs:attribute ref="g"` + "\n" +
			`     default="1"/>` + "\n" +
			`</xs:attributeGroup>` + tail},
			"au-props-correct", xsderr.Loc{URI: "main.xsd", Line: 4, Col: 3}},
		{"clause 2 on a use inherited from an included base", map[string]string{
			"main.xsd": head +
				`<xs:include schemaLocation="part.xsd"/>` + "\n" +
				`<xs:complexType name="D"><xs:complexContent><xs:extension base="B"/></xs:complexContent></xs:complexType>` + tail,
			"part.xsd": head +
				`<xs:complexType name="B">` + "\n" +
				`    <xs:attribute name="k" type="xs:int" default="abc"/>` + "\n" +
				`</xs:complexType>` + tail},
			"au-props-correct", xsderr.Loc{URI: "part.xsd", Line: 3, Col: 5}},
		{"src-resolve of a dangling <attribute ref> in a complex type", map[string]string{"main.xsd": head +
			`<xs:complexType name="C">` + "\n" +
			`  <xs:attribute ref="nosuch"/>` + "\n" +
			`</xs:complexType>` + tail},
			"src-resolve", xsderr.Loc{URI: "main.xsd", Line: 3, Col: 17}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseMap(t, "main.xsd", tc.docs)
			var xe *xsderr.Error
			if !errors.As(err, &xe) {
				t.Fatalf("Parse error = %v, want an *xsderr.Error", err)
			}
			if xe.Rule != tc.rule {
				t.Fatalf("rule = %q, want %q (%v)", xe.Rule, tc.rule, err)
			}
			if xe.Loc != tc.want {
				t.Fatalf("loc = %s, want %s (%v)", xe.Loc, tc.want, err)
			}
		})
	}
}
