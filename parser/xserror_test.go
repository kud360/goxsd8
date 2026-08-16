package parser_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// ·xs:error· (§3.16.7.3) is present in every schema by definition, so naming it
// on a declaration resolves like any other builtin. This is the plainest
// statement of the defect: the reference used to be charged src-resolve clause
// 1.1 because no component carried the name (#821).
func TestParseElementTypedXSError(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("", `<xs:element name="a" type="xs:error"/>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	ed, ok := s.Element(xsd.QName{Local: "a"})
	if !ok {
		t.Fatal("element declaration a not found")
	}
	if got, want := declaredTypeName(t, ed.TypeDefinition()), (xsd.QName{Space: xsdNS, Local: "error"}); got != want {
		t.Errorf("a {type definition} = %v, want %v", got, want)
	}
}

// An <attribute> naming xs:error resolves on the same terms — the shape
// saxonData/VC/vc014.xsd takes.
func TestProduceAttributeTypedXSError(t *testing.T) {
	if _, err := produce(t, wrap("", `<xs:attribute name="a" type="xs:error"/>`)); err != nil {
		t.Fatalf("Produce: %v", err)
	}
}

// xs:error's {final} is all four derivation keywords, so every way of deriving
// from it is rejected by the ordinary machinery — no branch anywhere tests for
// its identity. Restriction and complex extension are charged against the
// {base type definition}; list and union against the item and member slots.
func TestProduceDerivationFromXSErrorRejected(t *testing.T) {
	cases := []struct {
		name string
		body string
		want xsderr.Rule
	}{
		{"restriction", `<xs:simpleType name="D"><xs:restriction base="xs:error"/></xs:simpleType>`, "st-props-correct"},
		{"list", `<xs:simpleType name="D"><xs:list itemType="xs:error"/></xs:simpleType>`, "cos-st-restricts"},
		{"union", `<xs:simpleType name="D"><xs:union memberTypes="xs:error"/></xs:simpleType>`, "cos-st-restricts"},
		{"extension", `<xs:complexType name="D"><xs:simpleContent><xs:extension base="xs:error"><xs:attribute name="k" type="xs:string"/></xs:extension></xs:simpleContent></xs:complexType>`, "cos-ct-extends"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, wrap("", tc.body))
			assertRule(t, err, tc.want)
		})
	}
}
