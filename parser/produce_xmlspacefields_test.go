package parser_test

import (
	"slices"
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

// The rows below pin that every list-typed schema attribute is delimited on XML
// white space alone, xml.md [3] S (cvc-datatype-valid, Datatypes §4.1.4 clause
// 2.2; §4.3.6), and never on the other Unicode spaces strings.Fields breaks on.
// What a non-S space inside a token means depends on the item type:
//
//   - U+00A0 and U+2028 are no NameChars, so a QName item holding one is no
//     QName. Each such row pairs two prefixed names, so the one token carries
//     two colons and qnameLexical charges cvc-datatype-valid.
//   - U+1680 is an NCName character (NameStartChar #x37F-#x1FFF), so a QName
//     item holding one is ONE valid name, which strings.Fields cut in two.
//   - xs:anyURI admits every XML Char, so a namespace item holding U+00A0 or
//     U+2028 is one valid namespace name, charged nothing.
//
// Every row fails with its site's split reverted to strings.Fields.

// wildcardOfCT returns the <xs:any> wildcard that complex type CT in target
// holds as its sequence's one particle.
func wildcardOfCT(t *testing.T, s *xsd.Schema, target string) xsd.Wildcard {
	t.Helper()
	td, ok := s.Type(xsd.QName{Space: target, Local: "CT"})
	if !ok {
		t.Fatalf("complexType CT not found")
	}
	ps := topGroup(t, td.(xsd.ComplexType)).Particles()
	if len(ps) != 1 {
		t.Fatalf("particles = %d, want 1", len(ps))
	}
	wc, ok := ps[0].Term().(xsd.ResolvedTerm).Term.(xsd.Wildcard)
	if !ok {
		t.Fatalf("term = %T, want Wildcard", ps[0].Term().(xsd.ResolvedTerm).Term)
	}
	return wc
}

// TestProduceUnionMemberTypesSplitOnXMLSpace pins unionMembers' split of
// memberTypes= (§3.16.2.4 map.std.union case 1a).
func TestProduceUnionMemberTypesSplitOnXMLSpace(t *testing.T) {
	for _, sp := range []string{"&#xA0;", "&#x2028;"} {
		t.Run("two QNames joined by "+sp+" are one token, no QName", func(t *testing.T) {
			_, err := produce(t, wrap("", `<xs:simpleType name="U"><xs:union memberTypes="xs:string`+sp+`xs:int"/></xs:simpleType>`))
			assertRule(t, err, "cvc-datatype-valid")
		})
	}
	t.Run("a U+1680 inside a name is one member", func(t *testing.T) {
		body := `<xs:simpleType name="U"><xs:union memberTypes="tns:a&#x1680;b"/></xs:simpleType>` +
			`<xs:simpleType name="a&#x1680;b"><xs:restriction base="xs:string"/></xs:simpleType>`
		s, err := produce(t, wrap("urn:x", body))
		if err != nil {
			t.Fatalf("Produce: %v", err)
		}
		u := mustSimpleType(t, s, xsd.QName{Space: "urn:x", Local: "U"})
		want := []xsd.QName{{Space: "urn:x", Local: "a b"}}
		if got := memberNames(mustMembers(t, s, u)); !slices.Equal(got, want) {
			t.Errorf("U {member type definitions} = %v, want %v", got, want)
		}
	})
}

// TestProduceSubstitutionGroupSplitOnXMLSpace pins
// substitutionGroupAffiliations' split of substitutionGroup= (§3.3.2.1
// {substitution group affiliations}).
func TestProduceSubstitutionGroupSplitOnXMLSpace(t *testing.T) {
	for _, sp := range []string{"&#xA0;", "&#x2028;"} {
		t.Run("two QNames joined by "+sp+" are one token, no QName", func(t *testing.T) {
			_, err := produce(t, wrap("urn:x", `<xs:element name="m" substitutionGroup="tns:h`+sp+`tns:k"/>`))
			assertRule(t, err, "cvc-datatype-valid")
		})
	}
	t.Run("a U+1680 inside a name is one affiliation", func(t *testing.T) {
		body := `<xs:element name="h&#x1680;k"/><xs:element name="m" substitutionGroup="tns:h&#x1680;k"/>`
		s, err := produce(t, wrap("urn:x", body))
		if err != nil {
			t.Fatalf("Produce: %v", err)
		}
		m, ok := s.Element(xsd.QName{Space: "urn:x", Local: "m"})
		if !ok {
			t.Fatalf("element m not found")
		}
		want := []xsd.QName{{Space: "urn:x", Local: "h k"}}
		if got := m.SubstitutionGroupAffiliationNames(); !slices.Equal(got, want) {
			t.Errorf("m {substitution group affiliations} = %v, want %v", got, want)
		}
	})
}

// TestProduceSubstitutionGroupHeadTypeSplitOnXMLSpace pins
// substitutionGroupHeadType's split of a HEAD's own substitutionGroup=, which
// it reads to follow §3.3.2.1 dcl.elt.common {type definition} case 3 up a
// chain: m takes mid's type, and mid has only its first affiliation's, xs:int.
// Split at the U+1680, mid's first item is tns:h, an ·absent· head, and m gets
// xs:anyType, which e-props-correct clause 4 then refuses against mid's xs:int.
//
// No U+00A0 row is observable here: mid's own substitutionGroup= reaches
// substitutionGroupAffiliations first, which charges it (the test above).
func TestProduceSubstitutionGroupHeadTypeSplitOnXMLSpace(t *testing.T) {
	body := `<xs:element name="h&#x1680;k" type="xs:int"/>` +
		`<xs:element name="mid" substitutionGroup="tns:h&#x1680;k"/>` +
		`<xs:element name="m" substitutionGroup="tns:mid"/>`
	s, err := produce(t, wrap("urn:x", body))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	m, ok := s.Element(xsd.QName{Space: "urn:x", Local: "m"})
	if !ok {
		t.Fatalf("element m not found")
	}
	if got, want := declaredTypeName(t, m.TypeDefinition()), (xsd.QName{Space: xsdNS, Local: "int"}); got != want {
		t.Errorf("m {type definition} = %s, want %s", got, want)
	}
}

// TestProduceWildcardNamespaceSplitOnXMLSpace pins namespaceVarietyAndSet's
// split of namespace= and notNamespace= (§3.10.2.2 {namespaces}): a list of
// xs:anyURI, so a U+00A0 or U+2028 between two URIs leaves one valid namespace
// name.
func TestProduceWildcardNamespaceSplitOnXMLSpace(t *testing.T) {
	for _, tc := range []struct {
		attr, sp, want string
		variety        xsd.NamespaceConstraintVariety
	}{
		{"namespace", "&#xA0;", "urn:a urn:b", xsd.NamespaceConstraintEnumeration},
		{"namespace", "&#x2028;", "urn:a urn:b", xsd.NamespaceConstraintEnumeration},
		{"notNamespace", "&#xA0;", "urn:a urn:b", xsd.NamespaceConstraintNot},
		{"notNamespace", "&#x2028;", "urn:a urn:b", xsd.NamespaceConstraintNot},
	} {
		t.Run(tc.attr+" joined by "+tc.sp, func(t *testing.T) {
			body := `<xs:complexType name="CT"><xs:sequence>` +
				`<xs:any ` + tc.attr + `="urn:a` + tc.sp + `urn:b" processContents="skip"/>` +
				`</xs:sequence></xs:complexType>`
			s, err := produce(t, wrap("", body))
			if err != nil {
				t.Fatalf("Produce: %v", err)
			}
			nc := wildcardOfCT(t, s, "").NamespaceConstraint()
			if got := nc.Variety(); got != tc.variety {
				t.Errorf("{variety} = %s, want %s", got, tc.variety)
			}
			if got, want := nc.Namespaces(), []xsd.Namespace{xsd.NamespaceName(tc.want)}; !slices.Equal(got, want) {
				t.Errorf("{namespaces} = %q, want %q", got, want)
			}
		})
	}
}

// TestProduceNotQNameSplitOnXMLSpace pins disallowedNames' split of notQName=
// (§3.10.2.2 {disallowed names}).
func TestProduceNotQNameSplitOnXMLSpace(t *testing.T) {
	for _, sp := range []string{"&#xA0;", "&#x2028;"} {
		t.Run("two QNames joined by "+sp+" are one token, no QName", func(t *testing.T) {
			body := `<xs:complexType name="CT"><xs:sequence><xs:any notQName="tns:a` + sp + `tns:b"/></xs:sequence></xs:complexType>`
			_, err := produce(t, wrap("urn:x", body))
			assertRule(t, err, "cvc-datatype-valid")
		})
	}
	t.Run("a U+1680 inside a name is one disallowed name", func(t *testing.T) {
		body := `<xs:complexType name="CT"><xs:sequence><xs:any notQName="tns:a&#x1680;b"/></xs:sequence></xs:complexType>`
		s, err := produce(t, wrap("urn:x", body))
		if err != nil {
			t.Fatalf("Produce: %v", err)
		}
		nc := wildcardOfCT(t, s, "urn:x").NamespaceConstraint()
		if nc.AllowsName(xsd.QName{Space: "urn:x", Local: "a b"}) {
			t.Errorf("AllowsName admitted {urn:x}a\\u1680b, the one notQName member")
		}
		if !nc.AllowsName(xsd.QName{Space: "urn:x", Local: "a"}) {
			t.Errorf("AllowsName refused {urn:x}a, which notQName does not name")
		}
	})
}
