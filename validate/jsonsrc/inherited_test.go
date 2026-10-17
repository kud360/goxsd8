package jsonsrc

import (
	"slices"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/validate/xmlsrc"
	"github.com/kud360/goxsd8/xsderr"
)

// inheritSchema declares, in urn:i, the top-level attribute ia, inheritable;
// T, an optional child m of xs:int, the attribute a and a lax ##local
// {attribute wildcard}; and U, which restricts T to no child and the
// attribute m of xs:string. Under T a key "m" is the child, whose "x" charges
// cvc-type, and under U the attribute; an infoset built under the type the
// engine does not select charges otherwise — a child m under U charges
// cvc-complex-type, and an attribute m under T, which its wildcard admits,
// nothing — so each row's verdict tells the two selections apart. tu is of
// type T and selects U where its a is "u", ti where its i:ia is "u", and tb
// where its i:ib is "u"; no top-level ib exists. p holds an optional p and
// then an optional tu, and carries a, inheritable; pd carries a, inheritable,
// defaulting to "u"; pn carries a, not inheritable, and pnd carries it
// defaulting to "u"; each holds an optional tu. pw holds an optional ti and an
// optional tb under a lax {attribute wildcard} of urn:i, and pk holds an
// optional ti under a skip one. pt is of type P0, which holds an optional tu,
// and PT extends P0 with a, inheritable. po holds an optional x of type T that
// selects U where its a is "u", and has a lax interleave {open content}; ps is
// po with a skip one. mid holds an optional tu. H holds an optional mid and
// carries a, and H2 extends it with ib, inheritable, defaulting to "v"; rw is
// of type H and selects H2 where it counts an a, a {test} outside the §3.12.6
// compiler's subset. big's {content type} is one xsd.Schema.ContentMatcher
// declines (its GAP(xsd) ceiling), holding mids.
const inheritSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:i" xmlns:i="urn:i" elementFormDefault="qualified">
  <xs:attribute name="ia" type="xs:string" inheritable="true"/>
  <xs:complexType name="T">
    <xs:sequence><xs:element name="m" type="xs:int" minOccurs="0"/></xs:sequence>
    <xs:attribute name="a" type="xs:string"/>
    <xs:anyAttribute namespace="##local" processContents="lax"/>
  </xs:complexType>
  <xs:complexType name="U">
    <xs:complexContent>
      <xs:restriction base="i:T"><xs:sequence/><xs:attribute name="m" type="xs:string"/></xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="tu" type="i:T"><xs:alternative test="@a = 'u'" type="i:U"/></xs:element>
  <xs:element name="ti" type="i:T"><xs:alternative test="@i:ia = 'u'" type="i:U"/></xs:element>
  <xs:element name="tb" type="i:T"><xs:alternative test="@i:ib = 'u'" type="i:U"/></xs:element>
  <xs:element name="p">
    <xs:complexType>
      <xs:sequence><xs:element ref="i:p" minOccurs="0"/><xs:element ref="i:tu" minOccurs="0"/></xs:sequence>
      <xs:attribute name="a" type="xs:string" inheritable="true"/>
    </xs:complexType>
  </xs:element>
  <xs:element name="pd">
    <xs:complexType>
      <xs:sequence><xs:element ref="i:tu" minOccurs="0"/></xs:sequence>
      <xs:attribute name="a" type="xs:string" default="u" inheritable="true"/>
    </xs:complexType>
  </xs:element>
  <xs:element name="pn">
    <xs:complexType>
      <xs:sequence><xs:element ref="i:tu" minOccurs="0"/></xs:sequence>
      <xs:attribute name="a" type="xs:string"/>
    </xs:complexType>
  </xs:element>
  <xs:element name="pnd">
    <xs:complexType>
      <xs:sequence><xs:element ref="i:tu" minOccurs="0"/></xs:sequence>
      <xs:attribute name="a" type="xs:string" default="u"/>
    </xs:complexType>
  </xs:element>
  <xs:element name="pw">
    <xs:complexType>
      <xs:sequence><xs:element ref="i:ti" minOccurs="0"/><xs:element ref="i:tb" minOccurs="0"/></xs:sequence>
      <xs:anyAttribute namespace="##targetNamespace" processContents="lax"/>
    </xs:complexType>
  </xs:element>
  <xs:element name="pk">
    <xs:complexType>
      <xs:sequence><xs:element ref="i:ti" minOccurs="0"/></xs:sequence>
      <xs:anyAttribute namespace="##targetNamespace" processContents="skip"/>
    </xs:complexType>
  </xs:element>
  <xs:complexType name="P0"><xs:sequence><xs:element ref="i:tu" minOccurs="0"/></xs:sequence></xs:complexType>
  <xs:complexType name="PT">
    <xs:complexContent><xs:extension base="i:P0"><xs:attribute name="a" type="xs:string" inheritable="true"/></xs:extension></xs:complexContent>
  </xs:complexType>
  <xs:element name="pt" type="i:P0"/>
  <xs:element name="po">
    <xs:complexType>
      <xs:openContent mode="interleave"><xs:any processContents="lax"/></xs:openContent>
      <xs:sequence><xs:element name="x" type="i:T" minOccurs="0"><xs:alternative test="@a = 'u'" type="i:U"/></xs:element></xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:element name="ps">
    <xs:complexType>
      <xs:openContent mode="interleave"><xs:any processContents="skip"/></xs:openContent>
      <xs:sequence><xs:element name="x" type="i:T" minOccurs="0"><xs:alternative test="@a = 'u'" type="i:U"/></xs:element></xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:element name="mid"><xs:complexType><xs:sequence><xs:element ref="i:tu" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>
  <xs:complexType name="H">
    <xs:sequence><xs:element ref="i:mid" minOccurs="0"/></xs:sequence>
    <xs:attribute name="a" type="xs:string"/>
  </xs:complexType>
  <xs:complexType name="H2">
    <xs:complexContent><xs:extension base="i:H"><xs:attribute name="ib" type="xs:string" default="v" inheritable="true"/></xs:extension></xs:complexContent>
  </xs:complexType>
  <xs:element name="rw" type="i:H"><xs:alternative test="count(@a) > 0" type="i:H2"/></xs:element>
  <xs:element name="big">
    <xs:complexType>
      <xs:sequence>
        <xs:sequence maxOccurs="1000"><xs:element ref="i:mid" maxOccurs="unbounded"/></xs:sequence>
        <xs:element name="b" type="xs:string"/>
        <xs:sequence maxOccurs="1000"><xs:element name="c" maxOccurs="unbounded"/></xs:sequence>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

// TestInheritedSelected pins the [inherited attributes] the selection of a
// {type table} below the root reads (walker.handDown, walker.selector): an
// ancestor's attribute that is ·potentially inherited· (key-p-inherited)
// decides the type tu or ti selects, and so whether m is its attribute or its
// child, as the engine selects it for each XML twin (key-cta-ta-select
// clause 1.1.3). The rows read, in turn: a specified attribute whose use is
// inheritable (clause 3.1); one ·defaulted· from such a use; one whose use is
// not inheritable, specified and ·defaulted·; the nearer of two same-named
// inheritable ones (e-inherited_attributes clause 2), in either order; the
// element's own attribute over an inherited one of its name
// (key-cta-ta-select clause 1.1.3); an attribute an {attribute wildcard}
// admits, whose top-level declaration is inheritable (clause 3.2); the same
// attribute ·skipped· by a skip wildcard; one a lax wildcard admits with no
// declaration; and an attribute whose use the type a "$type" names, which
// ·overrides· the declared one, makes inheritable.
func TestInheritedSelected(t *testing.T) {
	v := validatorOver(t, inheritSchema)
	const i = `xmlns="urn:i" xmlns:i="urn:i" `
	for _, tc := range []struct {
		name, json, xml string
		want            []xsderr.Rule
	}{
		{"an inheritable attribute", `{"p":{"a":"u","tu":{"m":"x"}}}`,
			`<p ` + i + `a="u"><tu m="x"/></p>`, none},
		{"a defaulted inheritable attribute", `{"pd":{"tu":{"m":"x"}}}`,
			`<pd ` + i + `><tu m="x"/></pd>`, none},
		{"an attribute that is not inheritable", `{"pn":{"a":"u","tu":{"m":"x"}}}`,
			`<pn ` + i + `a="u"><tu><m>x</m></tu></pn>`, typ},
		{"a defaulted attribute that is not inheritable", `{"pnd":{"tu":{"m":"x"}}}`,
			`<pnd ` + i + `><tu><m>x</m></tu></pnd>`, typ},
		{"the nearer of two inheritable attributes", `{"p":{"a":"z","p":{"a":"u","tu":{"m":"x"}}}}`,
			`<p ` + i + `a="z"><p a="u"><tu m="x"/></p></p>`, none},
		{"the nearer of two inheritable attributes, the farther selecting", `{"p":{"a":"u","p":{"a":"z","tu":{"m":"x"}}}}`,
			`<p ` + i + `a="u"><p a="z"><tu><m>x</m></tu></p></p>`, typ},
		{"the element's own attribute over an inherited one", `{"p":{"a":"u","tu":{"a":"z","m":"x"}}}`,
			`<p ` + i + `a="u"><tu a="z"><m>x</m></tu></p>`, typ},
		{"a wildcard's attribute whose declaration is inheritable", `{"pw":{"$xmlns":{"i":"urn:i"},"i:ia":"u","ti":{"m":"x"}}}`,
			`<pw ` + i + `i:ia="u"><ti m="x"/></pw>`, none},
		{"a skipped attribute", `{"pk":{"$xmlns":{"i":"urn:i"},"i:ia":"u","ti":{"m":"x"}}}`,
			`<pk ` + i + `i:ia="u"><ti><m>x</m></ti></pk>`, typ},
		{"a wildcard's attribute with no declaration", `{"pw":{"$xmlns":{"i":"urn:i"},"i:ib":"u","tb":{"m":"x"}}}`,
			`<pw ` + i + `i:ib="u"><tb><m>x</m></tb></pw>`, typ},
		{"an inheritable attribute of the type a \"$type\" names", `{"pt":{"$xmlns":{"i":"urn:i"},"$type":"i:PT","a":"u","tu":{"m":"x"}}}`,
			`<pt ` + i + xsiDecl + ` xsi:type="i:PT" a="u"><tu m="x"/></pt>`, none},
	} {
		t.Run(tc.name, func(t *testing.T) { twin(t, v, tc.json, tc.xml, tc.want) })
	}
}

// TestNoGoverningDeclaration pins walker.selector's arm for an element its
// parent's ·governing type definition· leaves with no ·governing element
// declaration·: the second x of po is ·attributed to· the lax {open content}
// and governed by its ·locally declared type· T (key-governing-ed clause 4.3,
// key-governing-type-elem clause 7), and the second x of ps is ·skipped· by
// the skip one, so the engine reads no {type table} for either, and the
// mapping classifies the members under the declaration's T alone: a an
// attribute and m a child, where U would take m as an attribute.
func TestNoGoverningDeclaration(t *testing.T) {
	v := validatorOver(t, inheritSchema)
	const i = `xmlns="urn:i" `
	for _, tc := range []struct {
		name, json, xml string
		want            []xsderr.Rule
	}{
		{"a locally declared type", `{"po":{"x":[{},{"a":"u","m":"x"}]}}`,
			`<po ` + i + `><x/><x a="u"><m>x</m></x></po>`, typ},
		{"skipped", `{"ps":{"x":[{},{"a":"u","m":"x"}]}}`,
			`<ps ` + i + `><x/><x a="u"><m>x</m></x></ps>`, none},
	} {
		t.Run(tc.name, func(t *testing.T) { twin(t, v, tc.json, tc.xml, tc.want) })
	}
}

// TestInheritedUndetermined pins walker.classify's GAP(jsonsrc) markers: where
// the mapping determines no [inherited attributes] for tu, it reads no
// selection of its {type table} (walker.selector), and m, a child under T and
// an attribute under U, declines, where the engine walks each XML twin's
// subtree undecided. rw's selection is withheld, and the types it may select,
// H and H2, hand down different attributes, H2 a ·defaulted· ib
// (walker.handDown), so mid hands down none determined; big's content model
// is one xsd.Schema.ContentMatcher declines, so mid is attributed to nothing.
func TestInheritedUndetermined(t *testing.T) {
	v := validatorOver(t, inheritSchema)
	for _, tc := range []struct {
		name, json, xml string
		opens           string
	}{
		{"below a withheld selection", `{"rw":{"a":"1","mid":{"tu":{"a":"u","m":"x"}}}}`,
			`<rw xmlns="urn:i" a="1"><mid><tu a="u" m="x"/></mid></rw>`,
			`jsonsrc: instance.json:1:37: member "m" of {urn:i}tu is not classified alike`},
		{"below an unattributed parent", `{"big":{"mid":{"tu":{"a":"u","m":"x"}},"b":""}}`,
			`<big xmlns="urn:i"><mid><tu a="u" m="x"/></mid><b/></big>`,
			`jsonsrc: instance.json:1:30: member "m" of {urn:i}tu is not classified alike`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			xres, err := xmlsrc.Validate(v, strings.NewReader(tc.xml), xmlsrc.WithURI("instance.xml"))
			if err != nil || xres.Err() != nil {
				t.Fatalf("xmlsrc.Validate(%s): %v / %v", tc.xml, err, xres.Err())
			}
			if got := rules(xres); !slices.Equal(got, none) || len(xres.Unevaluated()) == 0 {
				t.Fatalf("XML twin %s charges %v, %d unevaluated; want none charged and its subtree undecided", tc.xml, got, len(xres.Unevaluated()))
			}
			res, err := Validate(v, strings.NewReader(tc.json), WithURI(jsonURI))
			if err == nil {
				t.Fatalf("Validate(%s) = %v violations, nil error; want the decline", tc.json, rules(res))
			}
			if !strings.HasPrefix(err.Error(), tc.opens) {
				t.Errorf("Validate(%s) = %q, want it to open %q", tc.json, err, tc.opens)
			}
		})
	}
}
