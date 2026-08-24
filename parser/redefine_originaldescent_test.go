package parser_test

import "testing"

// This file pins the containment edge that only a <redefine> creates: the
// src-expredef clause 1.1 ORIGINAL, whose {name} is ·absent· and which clause 1.2
// puts in the redefining type's {base type definition} and nowhere else. It is in
// no index and no other slot names it, so a finalize walk that skips {base type
// definition} cannot reach it or anything nested inside it (#843).
//
// Every case below redefines by RESTRICTION deliberately. An <extension>'s
// {content type} contains the base's particle (§3.4.2.3.2), so an element
// declaration inside an extended original stays reachable through the redefining
// type's own content model and would pass whether or not the base slot is walked.
// A restriction writes its content model out fresh, so the original's is reachable
// ONLY down the base slot — which is what makes these tests able to fail.

// TestRedefineOriginalElementDefaultCharged is e-props-correct (§3.3.6.1) clause 2
// inside a redefine original: the original's local <element> defaults an xs:int to
// a non-numeral, which is not a valid default with respect to its {type
// definition}.
func TestRedefineOriginalElementDefaultCharged(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:redefine schemaLocation="lib.xsd">`+
			`<xs:complexType name="T"><xs:complexContent>`+
			`<xs:restriction base="tns:T"><xs:sequence>`+
			`<xs:element name="e" type="xs:int"/>`+
			`</xs:sequence></xs:restriction>`+
			`</xs:complexContent></xs:complexType>`+
			`</xs:redefine>`+
			`<xs:element name="root" type="tns:T"/>`),
		"lib.xsd": wrap("urn:a", `<xs:complexType name="T"><xs:sequence>`+
			`<xs:element name="e" type="xs:int" default="notAnInt"/>`+
			`</xs:sequence></xs:complexType>`),
	})
	mustRule(t, err, "e-props-correct", "clause 2")
}

// TestRedefineOriginalTypeTableCharged is e-props-correct clause 7 inside a
// redefine original: the original's local <element> is typed xs:string and carries
// an <alternative> typed xs:int, which is neither ·validly substitutable· for
// xs:string nor ·xs:error·.
//
// The redefinition restricts the original's emptiable content model AWAY, so the
// offending declaration exists in the original alone. Repeating it in the
// restriction would have charged clause 7 against the redefining type instead and
// passed with the base slot unwalked.
func TestRedefineOriginalTypeTableCharged(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:redefine schemaLocation="lib.xsd">`+
			`<xs:complexType name="T"><xs:complexContent>`+
			`<xs:restriction base="tns:T"><xs:sequence/></xs:restriction>`+
			`</xs:complexContent></xs:complexType>`+
			`</xs:redefine>`+
			`<xs:element name="root" type="tns:T"/>`),
		"lib.xsd": wrap("urn:a", `<xs:complexType name="T"><xs:sequence>`+
			`<xs:element name="e" type="xs:string" minOccurs="0">`+
			`<xs:alternative test="true()" type="xs:int"/>`+
			`</xs:element>`+
			`</xs:sequence></xs:complexType>`),
	})
	mustRule(t, err, "e-props-correct", "clause 7")
}

// TestRedefineOriginalAttributeUseFixedCharged is au-props-correct (§3.5.6) clause
// 3 inside a redefine original: the original's attribute use contradicts the fixed
// {value constraint} of the global declaration it references.
func TestRedefineOriginalAttributeUseFixedCharged(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:redefine schemaLocation="lib.xsd">`+
			`<xs:complexType name="T"><xs:complexContent>`+
			`<xs:restriction base="tns:T"><xs:sequence/>`+
			`<xs:attribute ref="tns:a" fixed="1"/>`+
			`</xs:restriction>`+
			`</xs:complexContent></xs:complexType>`+
			`</xs:redefine>`+
			`<xs:element name="root" type="tns:T"/>`),
		"lib.xsd": wrap("urn:a", `<xs:attribute name="a" type="xs:int" fixed="1"/>`+
			`<xs:complexType name="T"><xs:sequence/>`+
			`<xs:attribute ref="tns:a" default="2"/>`+
			`</xs:complexType>`),
	})
	mustRule(t, err, "au-props-correct", "clause 3")
}
