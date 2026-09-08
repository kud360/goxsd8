package parser_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

// These tests pin the two mapping stages §4.1.4 fixes for every xs:boolean
// schema attribute the producer reads — mixed, abstract, nillable, inheritable,
// appliesToEmpty and defaultAttributesApply — in the order it fixes them:
// whiteSpace is collapse for xs:boolean (§3.3.2.3) and is applied BEFORE
// lexical-space membership is tested, and booleanRep admits exactly 'true',
// 'false', '1' and '0' (§3.3.2.2).
//
// Both directions bite, which is what separates this file from
// produce_collapsetrim_test.go's pure false-ACCEPT table. A padded literal must
// be READ (`mixed=" 1 "` is the value true, the shape saxonData/Open/open013.xsd
// writes), and a literal outside those four must be CHARGED rather than quietly
// read as false — §4.1.4 states no fallback clause. Reverting boolAttr to its
// raw `v == "true" || v == "1"` compare fails every case below: the positives
// because the padding is no longer trimmed, the rejections because an
// out-of-space lexical silently becomes false.

// TestProduceBooleanAttributePaddedActualValue reads each of the six attribute
// families with §4.3.6 whitespace on both sides of the literal and asserts the
// property that literal maps to. Every row would produce the FALSE property
// under a raw compare, so each check is an inequality a revert breaks.
func TestProduceBooleanAttributePaddedActualValue(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2).
	cases := []struct {
		name  string
		doc   string
		check func(*testing.T, *xsd.Schema)
	}{
		{
			// The named conformance case's own shape: saxonData/Open/open013.xsd
			// writes mixed=" 1 " on a <complexType> with no <complexContent>, and a
			// raw compare produced element-only content, which then charged
			// cvc-complex-type clause 1.1 against a document the suite calls valid.
			name: `mixed=" 1 " on <complexType> (produceImplicitContent)`,
			doc: wrap("urn:x", `<xs:complexType name="T" mixed=" 1 ">`+
				`<xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`),
			check: func(t *testing.T, s *xsd.Schema) {
				if got := contentTypeOf(t, s, xq("T")).Variety(); got != xsd.ContentMixed {
					t.Errorf("{content type} variety = %v, want mixed — §3.4.2.3.3 clause 1.2's ·effective mixed· is the ·actual value· of mixed", got)
				}
			},
		},
		{
			// The <complexContent> half of the clause 1.1/1.2 pair. The padded value
			// is read on the <complexContent>, which clause 1.1 lets win, and the
			// ·effective content· is non-empty so clause 4.2.3 takes ·effective mixed·
			// rather than copying the base's {content type} wholesale. Read as false
			// the extension would not even produce: cos-ct-extends clause
			// 1.4.3.2.2.1 forbids an element-only extension of a mixed base.
			name: `mixed=" true " on <complexContent> (produceComplexContent)`,
			doc: wrap("urn:x", `<xs:complexType name="B" mixed="true"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`+
				`<xs:complexType name="T"><xs:complexContent mixed=" true ">`+
				`<xs:extension base="tns:B"><xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence></xs:extension></xs:complexContent></xs:complexType>`),
			check: func(t *testing.T, s *xsd.Schema) {
				if got := contentTypeOf(t, s, xq("T")).Variety(); got != xsd.ContentMixed {
					t.Errorf("{content type} variety = %v, want mixed — clause 1.1 takes <complexContent>'s mixed", got)
				}
			},
		},
		{
			name: `abstract=" true " on a top-level <element> (produceElement)`,
			doc:  wrap("urn:x", `<xs:element name="e" type="xs:string" abstract=" true "/>`),
			check: func(t *testing.T, s *xsd.Schema) {
				ed, ok := s.Element(xq("e"))
				if !ok {
					t.Fatal("element e not found")
				}
				if !ed.Abstract() {
					t.Error("{abstract} = false, want true — §3.3.2.1 dcl.elt.common reads the ·actual value·")
				}
			},
		},
		{
			name: `abstract=" 1 " on a <complexType> (produceImplicitContent)`,
			doc:  wrap("urn:x", `<xs:complexType name="T" abstract=" 1 "><xs:sequence/></xs:complexType>`),
			check: func(t *testing.T, s *xsd.Schema) {
				if !topComplexTypeIn(t, s, xq("T")).Abstract() {
					t.Error("{abstract} = false, want true — §3.4.2.2 ctd-abstract reads the ·actual value·")
				}
			},
		},
		{
			// <simpleContent> is the third of the three <complexType> paths, and the
			// only one whose abstract= is read after a base type has been resolved.
			name: `abstract=" true " on a <simpleContent> <complexType> (produceSimpleContent)`,
			doc: wrap("urn:x", `<xs:complexType name="T" abstract=" true "><xs:simpleContent>`+
				`<xs:extension base="xs:string"/></xs:simpleContent></xs:complexType>`),
			check: func(t *testing.T, s *xsd.Schema) {
				if !topComplexTypeIn(t, s, xq("T")).Abstract() {
					t.Error("{abstract} = false, want true")
				}
			},
		},
		{
			// One document, both nillable sites: the top-level form in produceElement
			// and the local form in produceLocalElement.
			name: `nillable=" 1 " on the top-level and local <element> forms`,
			doc: wrap("urn:x", `<xs:element name="top" type="xs:string" nillable=" 1 "/>`+
				`<xs:complexType name="T"><xs:sequence>`+
				`<xs:element name="local" type="xs:string" nillable=" 1 "/></xs:sequence></xs:complexType>`),
			check: func(t *testing.T, s *xsd.Schema) {
				top, ok := s.Element(xq("top"))
				if !ok {
					t.Fatal("element top not found")
				}
				if !top.Nillable() {
					t.Error("top-level {nillable} = false, want true")
				}
				group := groupTermOf(t, elementContentOf(t, s, xq("T")).Particle)
				if !elementTermOf(t, group.Particles()[0]).Nillable() {
					t.Error("local {nillable} = false, want true")
				}
			},
		},
		{
			// One <attribute> feeds BOTH inheritable sites: produceAttributeUse maps
			// the Attribute Use's {inheritable} (§3.5.1 au-inheritable) and
			// produceLocalAttribute the Attribute Declaration's (§3.2.1
			// ad-inheritable), each reading this same padded literal.
			name: `inheritable=" true " on a local <attribute>`,
			doc: wrap("urn:x", `<xs:complexType name="T"><xs:sequence/>`+
				`<xs:attribute name="a" type="xs:string" inheritable=" true "/></xs:complexType>`),
			check: func(t *testing.T, s *xsd.Schema) {
				uses := topComplexTypeIn(t, s, xq("T")).AttributeUses()
				if len(uses) != 1 {
					t.Fatalf("{attribute uses} = %d, want 1", len(uses))
				}
				if !uses[0].Inheritable() {
					t.Error("Attribute Use {inheritable} = false, want true (§3.5.1)")
				}
				local, ok := uses[0].AttributeDeclaration().(xsd.LocalAttributeDeclaration)
				if !ok {
					t.Fatalf("{attribute declaration} is %T, want a local declaration", uses[0].AttributeDeclaration())
				}
				if !local.Declaration.Inheritable() {
					t.Error("Attribute Declaration {inheritable} = false, want true (§3.2.1)")
				}
			},
		},
		{
			// §3.4.2.3.3 clause 5.2.2: an EMPTY ·explicit content type· picks the
			// document's <defaultOpenContent> up only on appliesToEmpty = true.
			name: `appliesToEmpty=" true " on <defaultOpenContent> (wildcardElement)`,
			doc: wrap("urn:x", `<xs:defaultOpenContent appliesToEmpty=" true "><xs:any/></xs:defaultOpenContent>`+
				`<xs:complexType name="T"><xs:sequence/></xs:complexType>`),
			check: func(t *testing.T, s *xsd.Schema) {
				if openContentOfType(t, s, "T") == nil {
					t.Error("{open content} is absent, want the document's default — clause 5.2.2")
				}
			},
		},
		{
			// §3.4.2.4: only an ·actual value· of false opts out, so a padded " true "
			// must FOLD the default group. A raw compare read it as false and
			// suppressed the fold.
			name: `defaultAttributesApply=" true " on <complexType> (foldDefaultAttributes)`,
			doc:  defaultAttributesSchema(`<xs:complexType name="T" defaultAttributesApply=" true "><xs:sequence/></xs:complexType>`),
			check: func(t *testing.T, s *xsd.Schema) {
				if !hasAttrUse(topComplexTypeIn(t, s, xq("T")).AttributeUses(), "da") {
					t.Error("the <schema defaultAttributes> group was not folded, want it folded — defaultAttributesApply's ·actual value· is true")
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := produce(t, tc.doc)
			if err != nil {
				t.Fatalf("Produce: %v", err)
			}
			tc.check(t, s)
		})
	}
}

// TestProduceBooleanAttributeOutOfLexicalSpaceRejected charges every family for a
// literal outside booleanRep. cvc-datatype-valid (§4.1.4) states no fallback
// clause, so none of these may be read as false.
//
// The wantMsg of each row is the whole opening of boolAttr's message — element
// name, then attribute name, then the offending lexical — so the assertion pins
// the argument ORDER and not merely the presence of three strings (#1048).
//
// The rows reach thirteen of boolAttr's fourteen call sites. The fourteenth,
// produceLocalAttribute's inheritable, has no document that reaches it with a
// bad lexical: its only caller is produceAttributeUse, which reads inheritable
// off the SAME <attribute> element first and returns on the fault, so the row
// above for a local <attribute> charges at that site and this one never sees an
// invalid literal. Its propagation is still written out, because a call that
// drops the error is the STYLE S3 fault whether or not a test can reach it.
func TestProduceBooleanAttributeOutOfLexicalSpaceRejected(t *testing.T) {
	cases := []struct {
		name    string
		doc     string
		wantMsg string
	}{
		{
			name: `mixed="yes"`,
			doc: wrap("urn:x", `<xs:complexType name="T" mixed="yes">`+
				`<xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`),
			wantMsg: `<complexType> mixed value "yes"`,
		},
		{
			name: `mixed="True" on <complexContent>`,
			doc: wrap("urn:x", `<xs:complexType name="B"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`+
				`<xs:complexType name="T"><xs:complexContent mixed="True">`+
				`<xs:extension base="tns:B"><xs:sequence/></xs:extension></xs:complexContent></xs:complexType>`),
			wantMsg: `<complexContent> mixed value "True"`,
		},
		{
			// produceSimpleContent's own mixed site, reached before src-ct clause 1
			// has an ·actual value· to test: an out-of-space lexical is not the value
			// true, so the datatype fault is charged rather than clause 1.
			name: `mixed="yes" on a <simpleContent> <complexType>`,
			doc: wrap("urn:x", `<xs:complexType name="T" mixed="yes"><xs:simpleContent>`+
				`<xs:extension base="xs:string"/></xs:simpleContent></xs:complexType>`),
			wantMsg: `<complexType> mixed value "yes"`,
		},
		{
			name: `abstract="yes" on a <simpleContent> <complexType>`,
			doc: wrap("urn:x", `<xs:complexType name="T" abstract="yes"><xs:simpleContent>`+
				`<xs:extension base="xs:string"/></xs:simpleContent></xs:complexType>`),
			wantMsg: `<complexType> abstract value "yes"`,
		},
		{
			// The <complexType> half of the clause 1.1/1.2 mixed pair, charged at the
			// <complexType>'s own position and not the <complexContent>'s.
			name: `mixed="yes" on a <complexType> with a <complexContent>`,
			doc: wrap("urn:x", `<xs:complexType name="B"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`+
				`<xs:complexType name="T" mixed="yes"><xs:complexContent>`+
				`<xs:extension base="tns:B"><xs:sequence/></xs:extension></xs:complexContent></xs:complexType>`),
			wantMsg: `<complexType> mixed value "yes"`,
		},
		{
			name: `abstract="yes" on a <complexContent> <complexType>`,
			doc: wrap("urn:x", `<xs:complexType name="B"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`+
				`<xs:complexType name="T" abstract="yes"><xs:complexContent>`+
				`<xs:extension base="tns:B"><xs:sequence/></xs:extension></xs:complexContent></xs:complexType>`),
			wantMsg: `<complexType> abstract value "yes"`,
		},
		{
			name:    `abstract="yes" on a top-level <element>`,
			doc:     wrap("urn:x", `<xs:element name="e" type="xs:string" abstract="yes"/>`),
			wantMsg: `<element> abstract value "yes"`,
		},
		{
			name:    `abstract="TRUE" on a <complexType>`,
			doc:     wrap("urn:x", `<xs:complexType name="T" abstract="TRUE"><xs:sequence/></xs:complexType>`),
			wantMsg: `<complexType> abstract value "TRUE"`,
		},
		{
			name:    `nillable="TRUE" on a top-level <element>`,
			doc:     wrap("urn:x", `<xs:element name="e" type="xs:string" nillable="TRUE"/>`),
			wantMsg: `<element> nillable value "TRUE"`,
		},
		{
			name: `nillable="2" on a local <element>`,
			doc: wrap("urn:x", `<xs:complexType name="T"><xs:sequence>`+
				`<xs:element name="a" type="xs:string" nillable="2"/></xs:sequence></xs:complexType>`),
			wantMsg: `<element> nillable value "2"`,
		},
		{
			name: `inheritable="Y" on a local <attribute>`,
			doc: wrap("urn:x", `<xs:complexType name="T"><xs:sequence/>`+
				`<xs:attribute name="a" type="xs:string" inheritable="Y"/></xs:complexType>`),
			wantMsg: `<attribute> inheritable value "Y"`,
		},
		{
			name:    `appliesToEmpty="yes" on <defaultOpenContent>`,
			doc:     wrap("urn:x", `<xs:defaultOpenContent appliesToEmpty="yes"><xs:any/></xs:defaultOpenContent>`+`<xs:complexType name="T"><xs:sequence/></xs:complexType>`),
			wantMsg: `<defaultOpenContent> appliesToEmpty value "yes"`,
		},
		{
			name:    `defaultAttributesApply="no" on a <complexType>`,
			doc:     defaultAttributesSchema(`<xs:complexType name="T" defaultAttributesApply="no"><xs:sequence/></xs:complexType>`),
			wantMsg: `<complexType> defaultAttributesApply value "no"`,
		},
		{
			// The character-class boundary, per #324's precedent. U+00A0 is not
			// §4.3.6 whitespace and collapse PRESERVES it, so the padded literal is
			// not a member of booleanRep and must be charged — where the wider class
			// strings.TrimSpace cuts would read it as true.
			name: `mixed="&#xA0;1" — U+00A0 is not §4.3.6 whitespace`,
			doc: wrap("urn:x", `<xs:complexType name="T" mixed="&#xA0;1">`+
				`<xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`),
			wantMsg: `<complexType> mixed value "\u00a01"`,
		},
		{
			// The same boundary one character over: U+2028 LINE SEPARATOR is not #xA.
			name:    `nillable="1&#x2028;" — U+2028 is not §4.3.6 whitespace`,
			doc:     wrap("urn:x", `<xs:element name="e" type="xs:string" nillable="1&#x2028;"/>`),
			wantMsg: `<element> nillable value "1\u2028"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, tc.doc)
			mustRule(t, err, "cvc-datatype-valid", tc.wantMsg, "lexical space of xs:boolean")
		})
	}
}
