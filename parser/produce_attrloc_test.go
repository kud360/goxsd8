package parser_test

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// TestProduceAttributeValueChargeLocated pins that a charge against one
// attribute's value is located at that attribute's name, not at its element's
// start tag (#3429, #3462): one row per charge, each faulted attribute written
// after another attribute and on a later line than the element's "<", so neither
// the element's position nor its first attribute's answers it. The two guard
// rows charge an element as a whole and stay at its "<" though the attributes
// they read sit on the next line: src-simple-type clause 1 against the second
// <maxLength>, and p-props-correct clause 2.1 against a <sequence> whose
// minOccurs exceeds its maxOccurs.
//
// A row with doc set is produced from that whole document, for an attribute of
// <schema> or one needing a binding wrap does not write; the rest are body
// wrapped by wrap.
func TestProduceAttributeValueChargeLocated(t *testing.T) {
	const schemaOpen = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"`
	// A slice, not a map: subtest order is output (STYLE D2).
	cases := []struct {
		name string
		body string
		doc  string
		rule xsderr.Rule
		line int
		col  int
	}{
		{
			name: "rejectInvalidID non-NCName",
			body: "\n" + `<xs:element name="e"` + "\n" + `  type="xs:string" id="1"/>`,
			rule: "cvc-datatype-valid", line: 3, col: 20,
		},
		{
			name: "rejectInvalidID duplicate",
			body: "\n" + `<xs:element name="e" id="a" type="xs:string"/>` + "\n" +
				`<xs:element name="f"` + "\n" + `   type="xs:string" id="a"/>`,
			rule: "cvc-id", line: 4, col: 21,
		},
		{
			name: "rejectInvalidDerivationSet block",
			body: "\n" + `<xs:element name="e"` + "\n" + ` type="xs:string" block="foo"/>`,
			rule: "cvc-datatype-valid", line: 3, col: 19,
		},
		{
			name: "declarationName non-NCName",
			body: "\n" + `<xs:element type="xs:string"` + "\n" + `    name="a:b"/>`,
			rule: "cvc-datatype-valid", line: 3, col: 5,
		},
		{
			name: "rejectXmlnsName",
			body: "\n" + `<xs:attribute type="xs:string"` + "\n" + `  name="xmlns"/>`,
			rule: "no-xmlns", line: 3, col: 3,
		},
		{
			name: "facetCountValue nonNegativeInteger",
			body: "\n" + `<xs:simpleType name="T"><xs:restriction base="xs:string">` + "\n" +
				`<xs:maxLength id="m"` + "\n" + `  value="-1"/></xs:restriction></xs:simpleType>`,
			rule: "cvc-datatype-valid", line: 4, col: 3,
		},
		{
			name: "facetCountValue positiveInteger",
			body: "\n" + `<xs:simpleType name="T"><xs:restriction base="xs:decimal">` + "\n" +
				`<xs:totalDigits id="t"` + "\n" + ` value="0"/></xs:restriction></xs:simpleType>`,
			rule: "cvc-datatype-valid", line: 4, col: 2,
		},
		{
			name: "facetFixed",
			body: "\n" + `<xs:simpleType name="T"><xs:restriction base="xs:string">` + "\n" +
				`<xs:maxLength value="1"` + "\n" + `    fixed="yes"/></xs:restriction></xs:simpleType>`,
			rule: "cvc-datatype-valid", line: 4, col: 5,
		},
		{
			name: "guard: src-simple-type clause 1 at the element",
			body: "\n" + `<xs:simpleType name="T"><xs:restriction base="xs:string">` + "\n" +
				`<xs:maxLength value="1"/>  <xs:maxLength` + "\n" + `  value="2"/></xs:restriction></xs:simpleType>`,
			rule: "src-simple-type", line: 3, col: 28,
		},
		{
			name: "attributeUseToken use",
			body: "\n" + `<xs:complexType name="T"><xs:sequence/><xs:attribute name="a"` + "\n" +
				`  use="bogus"/></xs:complexType>`,
			rule: "cvc-datatype-valid", line: 3, col: 3,
		},
		{
			name: "allOccursGrammar <all> minOccurs",
			body: "\n" + `<xs:complexType name="T"><xs:all id="g"` + "\n" +
				`  minOccurs="x"><xs:element name="a" type="xs:string"/></xs:all></xs:complexType>`,
			rule: "cvc-datatype-valid", line: 3, col: 3,
		},
		{
			name: "allOccursGrammar <all> maxOccurs unbounded",
			body: "\n" + `<xs:complexType name="T"><xs:all id="g"` + "\n" +
				`   maxOccurs="unbounded"><xs:element name="a" type="xs:string"/></xs:all></xs:complexType>`,
			rule: "cvc-datatype-valid", line: 3, col: 4,
		},
		{
			name: "allOccursGrammar <all> maxOccurs outside {0,1}",
			body: "\n" + `<xs:complexType name="T"><xs:all id="g"` + "\n" +
				`    maxOccurs="2"><xs:element name="a" type="xs:string"/></xs:all></xs:complexType>`,
			rule: "cvc-datatype-valid", line: 3, col: 5,
		},
		{
			name: "boolAttr nillable",
			body: "\n" + `<xs:element name="e"` + "\n" + `  nillable="yes"/>`,
			rule: "cvc-datatype-valid", line: 3, col: 3,
		},
		{
			name: "boolAttr abstract",
			body: "\n" + `<xs:complexType name="T"` + "\n" + `   abstract="yes"/>`,
			rule: "cvc-datatype-valid", line: 3, col: 4,
		},
		{
			name: "boolAttr mixed",
			body: "\n" + `<xs:complexType name="T"` + "\n" + `    mixed="yes"/>`,
			rule: "cvc-datatype-valid", line: 3, col: 5,
		},
		{
			name: "boolAttr appliesToEmpty",
			body: "\n" + `<xs:defaultOpenContent mode="interleave"` + "\n" +
				`  appliesToEmpty="yes"><xs:any/></xs:defaultOpenContent><xs:complexType name="T"/>`,
			rule: "cvc-datatype-valid", line: 3, col: 3,
		},
		{
			// defaultAttributesRef reads defaultAttributesApply whether or not
			// <schema> carries defaultAttributes; this row writes one, the next
			// does not.
			name: "boolAttr defaultAttributesApply",
			doc: schemaOpen + ` defaultAttributes="ag"><xs:attributeGroup name="ag"/>` + "\n" +
				`<xs:complexType name="T"` + "\n" + `   defaultAttributesApply="yes"/></xs:schema>`,
			rule: "cvc-datatype-valid", line: 3, col: 4,
		},
		{
			name: "boolAttr defaultAttributesApply without schema defaultAttributes",
			doc: schemaOpen + ">\n" +
				`<xs:complexType name="C"` + "\n" + `  defaultAttributesApply="yes"/></xs:schema>`,
			rule: "cvc-datatype-valid", line: 3, col: 3,
		},
		{
			name: "boolAttr inheritable",
			body: "\n" + `<xs:complexType name="T"><xs:sequence/><xs:attribute name="a"` + "\n" +
				`    inheritable="yes"/></xs:complexType>`,
			rule: "cvc-datatype-valid", line: 3, col: 5,
		},
		{
			name: "formChoiceQualified form",
			body: "\n" + `<xs:complexType name="T"><xs:sequence><xs:element name="a"` + "\n" +
				`  form="bogus"/></xs:sequence></xs:complexType>`,
			rule: "cvc-datatype-valid", line: 3, col: 3,
		},
		{
			name: "checkFormDefaults elementFormDefault",
			doc:  schemaOpen + "\n" + `  elementFormDefault="bogus"></xs:schema>`,
			rule: "cvc-datatype-valid", line: 2, col: 3,
		},
		{
			name: "checkFormDefaults attributeFormDefault",
			doc:  schemaOpen + "\n" + `   attributeFormDefault="bogus"></xs:schema>`,
			rule: "cvc-datatype-valid", line: 2, col: 4,
		},
		{
			name: "occursOf minOccurs",
			body: "\n" + `<xs:complexType name="T"><xs:sequence id="s"` + "\n" +
				`  minOccurs="x"><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`,
			rule: "cvc-datatype-valid", line: 3, col: 3,
		},
		{
			name: "occursOf maxOccurs",
			body: "\n" + `<xs:complexType name="T"><xs:sequence id="s"` + "\n" +
				`   maxOccurs="y"><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`,
			rule: "cvc-datatype-valid", line: 3, col: 4,
		},
		{
			name: "processContentsOf processContents",
			body: "\n" + `<xs:complexType name="T"><xs:sequence><xs:any namespace="##any"` + "\n" +
				`  processContents="bogus"/></xs:sequence></xs:complexType>`,
			rule: "cvc-datatype-valid", line: 3, col: 3,
		},
		{
			name: "disallowedNameKeywordOf notQName",
			body: "\n" + `<xs:complexType name="T"><xs:sequence><xs:any processContents="lax"` + "\n" +
				`   notQName="##bogus"/></xs:sequence></xs:complexType>`,
			rule: "cvc-datatype-valid", line: 3, col: 4,
		},
		{
			name: "conditionalDecimal vc:minVersion",
			doc: schemaOpen + ` xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning">` + "\n" +
				`<xs:element name="e"` + "\n" + `  vc:minVersion="x"/></xs:schema>`,
			rule: "src-cip", line: 3, col: 3,
		},
		{
			name: "conditionalDecimal vc:maxVersion",
			doc: schemaOpen + ` xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning">` + "\n" +
				`<xs:element name="e"` + "\n" + `   vc:maxVersion="x"/></xs:schema>`,
			rule: "src-cip", line: 3, col: 4,
		},
		{
			name: "guard: p-props-correct clause 2.1 at the element",
			body: "\n" + `<xs:complexType name="T"><xs:sequence minOccurs="2"` + "\n" +
				`  maxOccurs="1"><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`,
			rule: "p-props-correct", line: 2, col: 26,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := tc.doc
			if doc == "" {
				doc = wrap("", tc.body)
			}
			_, err := produce(t, doc)
			assertRule(t, err, tc.rule)
			loc, ok := xsderr.LocOf(err)
			if !ok {
				t.Fatalf("error %v carries no position (STYLE E3)", err)
			}
			if loc.URI != produceURI || loc.Line != tc.line || loc.Col != tc.col {
				t.Fatalf("position = %s:%d:%d, want %s:%d:%d (%v)",
					loc.URI, loc.Line, loc.Col, produceURI, tc.line, tc.col, err)
			}
		})
	}
}

// TestParseRedefineSelfReferenceOccursChargeLocated is the one #3462 charge
// outside a single produced document: checkSelfReferenceOccurs reads the
// redefining <group>'s self-reference, whose own minOccurs fails
// xs:nonNegativeInteger, and charges that attribute rather than the <group ref>
// carrying it.
func TestParseRedefineSelfReferenceOccursChargeLocated(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", "\n"+`<xs:redefine schemaLocation="lib.xsd"><xs:group name="g"><xs:sequence><xs:group ref="tns:g"`+"\n"+
			`  minOccurs="x"/></xs:sequence></xs:group></xs:redefine>`),
		"lib.xsd": wrap("urn:a", `<xs:group name="g"><xs:sequence>`+
			`<xs:element name="a" type="xs:string"/></xs:sequence></xs:group>`),
	})
	assertRule(t, err, "cvc-datatype-valid")
	loc, ok := xsderr.LocOf(err)
	if !ok {
		t.Fatalf("error %v carries no position (STYLE E3)", err)
	}
	if loc.URI != "main.xsd" || loc.Line != 3 || loc.Col != 3 {
		t.Fatalf("position = %s:%d:%d, want main.xsd:3:3 (%v)", loc.URI, loc.Line, loc.Col, err)
	}
}

// TestProduceQNameAttributeChargeLocated pins that a charge against a
// QName-valued attribute is located at that attribute's name, not at its
// element's start tag (#3463): one row per charging site, each faulted
// attribute written after another attribute and on a later line than the
// element's "<". A list-valued attribute (memberTypes, notQName,
// substitutionGroup, vc:typeAvailable) is located at the attribute, not at the
// faulted item, which each such row writes second. The guard row charges an
// element as a whole and stays at its "<" though the attribute it reads sits on
// the next line: src-identity-constraint clause 5 against a <unique ref> naming
// a <key>.
func TestProduceQNameAttributeChargeLocated(t *testing.T) {
	const vcOpen = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning">`
	// A slice, not a map: subtest order is output (STYLE D2).
	cases := []struct {
		name   string
		target string
		body   string
		doc    string
		rule   xsderr.Rule
		line   int
		col    int
		// msg, when set, is the opening of the message after its rule, pinning
		// the attribute name the charge reports.
		msg string
	}{
		{
			name: "bindQName lexical type",
			body: "\n" + `<xs:element name="e"` + "\n" + `  type="xs:"/>`,
			rule: "cvc-datatype-valid", line: 3, col: 3,
			msg: `<element> type value "xs:" `,
		},
		{
			name: "bindQName unbound prefix type",
			body: "\n" + `<xs:element name="e"` + "\n" + `   type="nope:T"/>`,
			rule: "src-resolve", line: 3, col: 4,
		},
		{
			name: "bindQName lexical memberTypes item",
			body: "\n" + `<xs:simpleType name="T"><xs:union id="u"` + "\n" +
				`  memberTypes="xs:string xs:"/></xs:simpleType>`,
			rule: "cvc-datatype-valid", line: 3, col: 3,
			msg: `<union> memberTypes value "xs:" `,
		},
		{
			name: "bindQName unbound prefix notQName item",
			body: "\n" + `<xs:complexType name="T"><xs:sequence><xs:any processContents="lax"` + "\n" +
				`    notQName="xs:string nope:a"/></xs:sequence></xs:complexType>`,
			rule: "src-resolve", line: 3, col: 5,
		},
		{
			name: "bindQName unbound prefix substitutionGroup item",
			body: "\n" + `<xs:element name="h" type="xs:string"/><xs:element name="e"` + "\n" +
				`  substitutionGroup="h nope:h"/>`,
			rule: "src-resolve", line: 3, col: 3,
		},
		{
			name: "bindQName unbound prefix itemType",
			body: "\n" + `<xs:simpleType name="T"><xs:list id="l"` + "\n" + ` itemType="nope:T"/></xs:simpleType>`,
			rule: "src-resolve", line: 3, col: 2,
		},
		{
			name: "bindQName unbound prefix keyref refer",
			body: "\n" + `<xs:element name="e" type="xs:string"><xs:keyref name="r"` + "\n" +
				`  refer="nope:k"><xs:selector xpath="."/><xs:field xpath="."/></xs:keyref></xs:element>`,
			rule: "src-resolve", line: 3, col: 3,
		},
		{
			name:   "licensedNamespace clause 4.1",
			target: "urn:a",
			body:   "\n" + `<xs:element name="e"` + "\n" + `  type="T"/>`,
			rule:   "src-resolve", line: 3, col: 3,
		},
		{
			name: "licensedNamespace clause 4.2",
			body: "\n" + `<xs:element name="e" xmlns:u="urn:u"` + "\n" + `   type="u:T"/>`,
			rule: "src-resolve", line: 3, col: 4,
		},
		{
			name: "resolveBaseType clause 1.1",
			body: "\n" + `<xs:complexType name="T"><xs:complexContent><xs:extension id="x"` + "\n" +
				`  base="Missing"/></xs:complexContent></xs:complexType>`,
			rule: "src-resolve", line: 3, col: 3,
		},
		{
			name: "referencedIdentityConstraint clause 1.7",
			body: "\n" + `<xs:element name="e" type="xs:string"><xs:key id="k"` + "\n" +
				`   ref="nokey"/></xs:element>`,
			rule: "src-resolve", line: 3, col: 4,
		},
		{
			name: "conditionalQNames lexical vc:typeAvailable item",
			doc: vcOpen + "\n" + `<xs:element name="e"` + "\n" +
				`  vc:typeAvailable="xs:string 23"/></xs:schema>`,
			rule: "src-cip", line: 3, col: 3,
		},
		{
			name: "conditionalQNames unbound prefix vc:facetAvailable item",
			doc: vcOpen + "\n" + `<xs:element name="e"` + "\n" +
				`    vc:facetAvailable="xs:length nope:length"/></xs:schema>`,
			rule: "src-cip", line: 3, col: 5,
		},
		{
			name: "guard: src-identity-constraint clause 5 at the element",
			body: "\n" + `<xs:element name="e" type="xs:string"><xs:key name="k"><xs:selector xpath="."/><xs:field xpath="."/></xs:key>` +
				`<xs:unique` + "\n" + `  ref="k"/></xs:element>`,
			rule: "src-identity-constraint", line: 2, col: 110,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := tc.doc
			if doc == "" {
				doc = wrap(tc.target, tc.body)
			}
			_, err := produce(t, doc)
			assertRule(t, err, tc.rule)
			loc, ok := xsderr.LocOf(err)
			if !ok {
				t.Fatalf("error %v carries no position (STYLE E3)", err)
			}
			if loc.URI != produceURI || loc.Line != tc.line || loc.Col != tc.col {
				t.Fatalf("position = %s:%d:%d, want %s:%d:%d (%v)",
					loc.URI, loc.Line, loc.Col, produceURI, tc.line, tc.col, err)
			}
			if want := "[" + string(tc.rule) + "] " + tc.msg; tc.msg != "" && !strings.Contains(err.Error(), want) {
				t.Fatalf("error %v does not open with %q", err, want)
			}
		})
	}
}

// TestParseRedefineQNameChargeLocated pins the #3463 charges a <redefine>
// reaches first: namesSelf reads a redefining type's base and namesSelfRef a
// redefining group's self-reference before either is produced, and each
// unbound prefix is charged at that base or ref attribute, written on a later
// line than its element's "<" and after its id.
func TestParseRedefineQNameChargeLocated(t *testing.T) {
	lib := wrap("urn:a", `<xs:simpleType name="s"><xs:restriction base="xs:string"/></xs:simpleType>`+
		`<xs:complexType name="c"><xs:sequence/></xs:complexType>`+
		`<xs:group name="g"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:group>`+
		`<xs:attributeGroup name="ag"/>`)
	// A slice, not a map: subtest order is output (STYLE D2).
	cases := []struct{ name, body string }{
		{"namesSelf simple base", `<xs:simpleType name="s"><xs:restriction id="r"` + "\n" +
			`  base="nope:s"/></xs:simpleType>`},
		{"namesSelf complex base", `<xs:complexType name="c"><xs:complexContent><xs:extension id="x"` + "\n" +
			`  base="nope:c"/></xs:complexContent></xs:complexType>`},
		{"namesSelfRef group ref", `<xs:group name="g"><xs:sequence><xs:group id="q"` + "\n" +
			`  ref="nope:g"/></xs:sequence></xs:group>`},
		{"namesSelfRef attributeGroup ref", `<xs:attributeGroup name="ag"><xs:attributeGroup id="q"` + "\n" +
			`  ref="nope:ag"/></xs:attributeGroup>`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseMap(t, "main.xsd", map[string]string{
				"main.xsd": wrap("urn:a", "\n"+`<xs:redefine schemaLocation="lib.xsd">`+tc.body+`</xs:redefine>`),
				"lib.xsd":  lib,
			})
			assertRule(t, err, "src-resolve")
			loc, ok := xsderr.LocOf(err)
			if !ok {
				t.Fatalf("error %v carries no position (STYLE E3)", err)
			}
			if loc.URI != "main.xsd" || loc.Line != 3 || loc.Col != 3 {
				t.Fatalf("position = %s:%d:%d, want main.xsd:3:3 (%v)", loc.URI, loc.Line, loc.Col, err)
			}
		})
	}
}
