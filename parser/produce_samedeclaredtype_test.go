package parser_test

import "testing"

// These tests pin cos-ct-derived-ok clause 2.1 (§3.4.6.5) for the ·locally
// declared type· comparison of cos-ct-extends clause 1.6 (c-vs-ctd-e): where a
// base and its extension reach ONE element declaration, the two {type
// definition}s are one component, anonymous or not (the no-identity Note's "two
// element declarations ... discovered to be the same declaration"), so the
// extension is accepted (#1973).

// An <element ref> to a global declaration owning an inline complexType, in a
// base extended by a type adding only an attribute — common/xsts.xsd's
// schemaDocumentRef/ref shape.
func TestProduceExtensionSharedRefAnonymousTypeAccepted(t *testing.T) {
	_, err := produce(t, wrap("urn:t", `
	<xs:element name="a">
	  <xs:complexType><xs:sequence><xs:element name="x" minOccurs="0"/></xs:sequence></xs:complexType>
	</xs:element>
	<xs:complexType name="B">
	  <xs:sequence><xs:element ref="tns:a" minOccurs="0"/></xs:sequence>
	</xs:complexType>
	<xs:complexType name="T">
	  <xs:complexContent><xs:extension base="tns:B"><xs:attribute name="k"/></xs:extension></xs:complexContent>
	</xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v, want an extension sharing B's one declaration of a accepted", err)
	}
}

// B declares a LOCAL <element name="a"> with an inline complexType, and T
// extends B adding c: T's content model reaches B's own local declaration, so a's
// ·locally declared type· within T is the one within B.
func TestProduceExtensionInheritedLocalAnonymousTypeAccepted(t *testing.T) {
	_, err := produce(t, wrap("", `
	<xs:complexType name="B">
	  <xs:sequence>
	    <xs:element name="a">
	      <xs:complexType><xs:sequence><xs:element name="x" minOccurs="0"/></xs:sequence></xs:complexType>
	    </xs:element>
	  </xs:sequence>
	</xs:complexType>
	<xs:complexType name="T">
	  <xs:complexContent><xs:extension base="B"><xs:sequence><xs:element name="c"/></xs:sequence></xs:extension></xs:complexContent>
	</xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v, want an extension inheriting B's local declaration of a accepted", err)
	}
}

// e-props-correct clause 7 still rejects an anonymous alternative type not
// derived from the element's anonymous declared type. The alternative's inline
// type carries the same {context} as the declared one, so this fails if the
// declaration identity above is ever decided inside ·validly substitutable·
// rather than at the ·locally declared type· comparison alone.
func TestProduceTypeTableAnonymousAlternativeOfAnonymousDeclaredTypeCharged(t *testing.T) {
	_, err := produce(t, wrap("", `
	<xs:element name="e">
	  <xs:complexType><xs:sequence/></xs:complexType>
	  <xs:alternative test="@k='i'"><xs:complexType><xs:sequence><xs:element name="z"/></xs:sequence></xs:complexType></xs:alternative>
	</xs:element>`))
	if err == nil {
		t.Fatal("Produce succeeded, want e-props-correct clause 7 for an anonymous alternative type not derived from the anonymous declared type")
	}
	assertRule(t, err, "e-props-correct")
}
