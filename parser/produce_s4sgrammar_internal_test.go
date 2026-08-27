package parser

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

// TestProduceComplexTypeNamelessBackstopNamesItsProduction pins the ONE
// s4s-grammar message no schema document can draw: produceComplexType's
// nameless-{name} guard, whose live entry paths run out at #305 (run takes every
// top-level name from topLevelName a step earlier) and #343 (bindQName rejects an
// empty base= lexical at the attribute). What it still backstops is a DIRECT
// in-package call, which is what this test makes — the guard's own doc says so,
// and the reason it is kept is that deleting it would make the verdict depend on
// whether the content happens to hold a local element (#206).
//
// The assertion is the production, not merely the rejection: the fault carries no
// Rule, so xs:topLevelComplexType is the only citation the message has (#975).
func TestProduceComplexTypeNamelessBackstopNamesItsProduction(t *testing.T) {
	doc, err := ReadDocument("mem://backstop.xsd", strings.NewReader(
		`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:x">`+
			`<xs:complexType name=""><xs:sequence/></xs:complexType></xs:schema>`))
	if err != nil {
		t.Fatalf("ReadDocument: %v", err)
	}
	el := childElement(doc.Root(), xsd.XMLSchemaNS, "complexType")
	if el == nil {
		t.Fatal("no <complexType> child of <schema>")
	}
	p := &producer{}
	_, err = p.produceComplexType(namedComplexType{name: xsd.QName{Space: "urn:x"}}, el)
	if err == nil {
		t.Fatal("produceComplexType succeeded on an empty {name}, want the grammar fault")
	}
	if !strings.Contains(err.Error(), "no usable name") ||
		!strings.Contains(err.Error(), "xs:topLevelComplexType") {
		t.Fatalf("error = %v, want the unusable-name fault naming xs:topLevelComplexType", err)
	}
}
