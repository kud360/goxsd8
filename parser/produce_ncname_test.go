package parser_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// TestProduceNonNCNameDeclarationNameRejected pins, END TO END from a schema
// DOCUMENT, that a declaration whose name attribute is not in the ·lexical
// space· of xs:NCName is rejected charged cvc-datatype-valid (Datatypes §4.1.4)
// at the declaration's own position — never registered as a {name} nobody can
// reference, which is what a colonized or digit-initial name became before #632.
//
// One row per CODE PATH into declarationName, not per bad name: the five
// top-level kinds routed through topLevelName, the top-level <simpleType> and
// <notation> that are not, the two local declaration forms, and the prohibited
// <attribute> that maps to no component at all (§3.4.2.4's Note) yet is still a
// name in a schema document. Reverting any one call site leaves its row failing.
func TestProduceNonNCNameDeclarationNameRejected(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2). Every body starts on
	// line 2 of the wrapped document and puts the offending declaration on the
	// line named by wantLine, so the charged position is pinned exactly.
	cases := []struct {
		name     string
		body     string
		wantLine int
	}{
		{
			name:     `top-level <element> name="a:b"`,
			body:     "\n" + `<xs:element name="a:b" type="xs:string"/>`,
			wantLine: 2,
		},
		{
			name:     `top-level <attribute> name="0"`,
			body:     "\n" + `<xs:attribute name="0" type="xs:string"/>`,
			wantLine: 2,
		},
		{
			name:     `top-level <complexType> name="1foo"`,
			body:     "\n" + `<xs:complexType name="1foo"><xs:sequence/></xs:complexType>`,
			wantLine: 2,
		},
		{
			name:     `top-level <group> name="-foo"`,
			body:     "\n" + `<xs:group name="-foo"><xs:sequence/></xs:group>`,
			wantLine: 2,
		},
		{
			name:     `top-level <attributeGroup> name=".x"`,
			body:     "\n" + `<xs:attributeGroup name=".x"/>`,
			wantLine: 2,
		},
		{
			// Outside topLevelName: run expands a top-level <simpleType>'s name
			// itself, and xsd.NewSimpleType carries no {name} guard (#523).
			name: `top-level <simpleType> name="nsk:Test"`,
			body: "\n" + `<xs:simpleType name="nsk:Test">` + "\n" +
				`<xs:restriction base="xs:string"/></xs:simpleType>`,
			wantLine: 2,
		},
		{
			// Outside topLevelName: produceNotation builds the component directly.
			name:     `<notation> name="foo:"`,
			body:     "\n" + `<xs:notation name="foo:" system="viewer.exe"/>`,
			wantLine: 2,
		},
		{
			// produceLocalElement. This is the path xsd.NewElementDeclaration's
			// e-props-correct clause 1 guard does NOT reach: that guard tests
			// emptiness, and ":bar" is not empty.
			name: `local <element> name=":bar"`,
			body: "\n" + `<xs:complexType name="CT"><xs:sequence>` + "\n" +
				`<xs:element name=":bar" type="xs:string"/>` + "\n" +
				`</xs:sequence></xs:complexType>`,
			wantLine: 3,
		},
		{
			// produceLocalAttribute, the a-props-correct clause 1 counterpart.
			name: `local <attribute> name="a:b:b"`,
			body: "\n" + `<xs:complexType name="CT"><xs:sequence/>` + "\n" +
				`<xs:attribute name="a:b:b" type="xs:string"/>` + "\n" +
				`</xs:complexType>`,
			wantLine: 3,
		},
		{
			// prohibitedAttributeNames: §3.4.2.4 clause 3.2.2's input, where the
			// <attribute> corresponds to no component and so has no constructor
			// guard behind it at all.
			name: `prohibited local <attribute> name="1x"`,
			body: "\n" + `<xs:complexType name="Base"><xs:sequence/></xs:complexType>` + "\n" +
				`<xs:complexType name="CT"><xs:complexContent>` + "\n" +
				`<xs:restriction base="Base"><xs:sequence/>` + "\n" +
				`<xs:attribute name="1x" use="prohibited"/>` + "\n" +
				`</xs:restriction></xs:complexContent></xs:complexType>`,
			wantLine: 5,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, wrap("", tc.body))
			assertRule(t, err, "cvc-datatype-valid")
			loc, ok := xsderr.LocOf(err)
			if !ok {
				t.Fatalf("error %v carries no position, want %s:%d:col (E3)", err, produceURI, tc.wantLine)
			}
			if loc.URI != produceURI || loc.Line != tc.wantLine || loc.Col == 0 {
				t.Fatalf("position = %s:%d:%d, want the offending declaration at %s:%d with a column",
					loc.URI, loc.Line, loc.Col, produceURI, tc.wantLine)
			}
		})
	}
}

// TestProduceXmlnsAttributeNameChargedNCName pins that an <attribute> named
// "xmlns:" or "xmlns:a" is charged the NCName lexical rule and NOT no-xmlns
// (§3.2.6.3). no-xmlns governs the bare string "xmlns", which IS an NCName; its
// own Note derives the xmlns:* prohibition FROM the NCName constraint, so a
// colonized name fails the lexical rule before no-xmlns could apply, and
// charging both would double-charge one fault (STYLE E2).
func TestProduceXmlnsAttributeNameChargedNCName(t *testing.T) {
	for _, name := range []string{"xmlns:", "xmlns:a"} {
		t.Run(name, func(t *testing.T) {
			_, err := produce(t, wrap("", `<xs:attribute name="`+name+`" type="xs:string"/>`))
			assertRule(t, err, "cvc-datatype-valid")
		})
	}
}

// TestProduceRedefiningNonNCNameRejected pins the <redefine> child, the one
// declaration-name path that mints a top-level {name} without passing through
// run's dispatch: produceRedefinition took the name straight off the entry key
// before #632. The lexical fault is charged ahead of src-expredef's closing
// requirement, so the redefined document's matching <group a:b> — which is
// present here, recorded as the original — does not change the verdict.
func TestProduceRedefiningNonNCNameRejected(t *testing.T) {
	docs := map[string]string{
		"main.xsd": wrap("", `<xs:redefine schemaLocation="base.xsd">`+"\n"+
			`<xs:group name="a:b"><xs:sequence/></xs:group>`+
			`</xs:redefine>`),
		"base.xsd": wrap("", `<xs:group name="a:b"><xs:sequence/></xs:group>`),
	}
	_, err := parseMap(t, "main.xsd", docs)
	assertRule(t, err, "cvc-datatype-valid")
}

// TestProduceDeclarationNameWhiteSpaceCollapsed pins the ·actual value· of a
// name attribute as the whiteSpace-normalized one: xs:NCName carries whiteSpace
// = collapse (Datatypes §3.4.7.1), so <element name="sub2-elem "> declares
// "sub2-elem" and is VALID. Rejecting the raw lexical instead would false-reject
// a schema the W3C suite declares valid for exactly this reason (addB193).
func TestProduceDeclarationNameWhiteSpaceCollapsed(t *testing.T) {
	s, err := produce(t, wrap("", "\n"+
		`<xs:element name="&#x9;sub2-elem&#xA; " type="xs:string"/>`))
	if err != nil {
		t.Fatalf("Produce: %v, want the collapsed name to be a valid xs:NCName", err)
	}
	if _, ok := s.Element(xsd.QName{Local: "sub2-elem"}); !ok {
		t.Fatalf("element {,sub2-elem} not declared; the name attribute's surrounding whitespace was not normalized away")
	}
}

// TestProduceNCNameDeclarationNamesAccepted guards the check against
// over-rejection: every character class the NCName production admits beyond a
// bare ASCII letter — '_' and ':'-free punctuation in a non-initial position, a
// digit in a non-initial position, and a non-ASCII NameStartChar — still
// declares a component.
func TestProduceNCNameDeclarationNamesAccepted(t *testing.T) {
	for _, name := range []string{"_foo.bar-1", "é", "A0"} {
		t.Run(name, func(t *testing.T) {
			s, err := produce(t, wrap("", `<xs:element name="`+name+`" type="xs:string"/>`))
			if err != nil {
				t.Fatalf("Produce: %v, want %q accepted as an xs:NCName", err, name)
			}
			if _, ok := s.Element(xsd.QName{Local: name}); !ok {
				t.Fatalf("element {,%s} not declared", name)
			}
		})
	}
}
