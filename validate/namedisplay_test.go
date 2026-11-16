package validate

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

// TestChargeNamesUserTypesByTheirOwnNamespace pins how a validate charge spells
// a type outside the XML Schema namespace, beside the xs:decimal its delegated
// cause names: a type in a target namespace as {namespace}local, a no-namespace
// one bare (#3688). The first row fails if [xsd.QName.Display] prefixes every
// namespace, the second if it spells a no-namespace name {}local.
func TestChargeNamesUserTypesByTheirOwnNamespace(t *testing.T) {
	const decimalCause = `[cvc-datatype-valid] "abc" is not in the lexical space of xs:decimal, which cvc-datatype-valid clause 2.1 requires it to be in`
	for _, tc := range []struct {
		name, tns string
		root      xsd.QName
		want      string
	}{
		{"target namespace", ` targetNamespace="urn:g" xmlns="urn:g" elementFormDefault="qualified"`, xsd.QName{Space: "urn:g", Local: "root"},
			`element {urn:g}root's value "abc" is not a valid {urn:g}T, which cvc-type clause 3.1.3 requires: ` + decimalCause},
		{"no namespace", ``, xsd.QName{Local: "root"},
			`element root's value "abc" is not a valid T, which cvc-type clause 3.1.3 requires: ` + decimalCause},
	} {
		t.Run(tc.name, func(t *testing.T) {
			schema := parsedSchema(t, map[string]string{"main.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"` + tc.tns + `>
  <xs:simpleType name="T"><xs:restriction base="xs:decimal"/></xs:simpleType>
  <xs:element name="root" type="T"/>
</xs:schema>`})
			root := &testElement{name: tc.root, loc: loc(1, 1),
				kids: []Child{TextChild(&testText{data: "abc", loc: loc(1, 7)})}}
			got := cAssess(t, schema, root)
			tWantCharge(t, got, "3.1.3", loc(1, 1))
			if got[0].Msg != tc.want {
				t.Errorf("Msg =\n\t%s\nwant\n\t%s", got[0].Msg, tc.want)
			}
		})
	}
}
