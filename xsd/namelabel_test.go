package xsd

import "testing"

// TestTypeLabelsSpellTheSchemaNamespaceXS pins the three type labels this
// package's messages name a type by — typeDefinitionLabel, complexTypeLabel and
// simpleTypeLabel — rendering an XML Schema namespace name through
// [QName.Display]: each row fails if its helper renders [QName.String] instead
// (#3688).
func TestTypeLabelsSpellTheSchemaNamespaceXS(t *testing.T) {
	decimal := &SimpleType{name: QName{Space: XMLSchemaNS, Local: "decimal"}}
	for _, tc := range []struct {
		helper, got, want string
	}{
		{"typeDefinitionLabel", typeDefinitionLabel(decimal), "xs:decimal"},
		{"complexTypeLabel", complexTypeLabel(QName{Space: XMLSchemaNS, Local: "anyType"}), "complex type xs:anyType"},
		{"simpleTypeLabel", simpleTypeLabel(decimal), "simple type xs:decimal"},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %q, want %q", tc.helper, tc.got, tc.want)
		}
	}
}
