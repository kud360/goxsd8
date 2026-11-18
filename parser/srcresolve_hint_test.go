package parser_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// TestSrcResolveNamesTheNearDeclaration pins, through Parse, the clause every
// src-resolve clause 1 charge for a QName naming no component appends: the
// nearest component of the kind its lookup seeks, in the missing name's
// namespace, under internal/nearname's rule (#3689). Each row pins the whole
// Msg, so a row naming no candidate fails if one is added.
//
// The guard rows each fail under a rule that drops one part: "kind" offers an
// element declaration to a type= and a complex type to a simple slot, a keyref
// to a refer= and a unique to a <key ref=>; "namespace" offers xs:string to a
// no-namespace type=; "threshold" offers xy to ab under a flat distance of 2.
// One row per site the clause reaches, each with a near declaration, fails if
// that site alone was not taught the clause. A row whose body opens <xs:schema>
// is the whole document.
func TestSrcResolveNamesTheNearDeclaration(t *testing.T) {
	const xs = "xs:"
	const head = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">`
	const tail = `</xs:schema>`
	const keyBody = `<xs:selector xpath="."/><xs:field xpath="@a"/>`
	for _, tc := range []struct {
		name, body, msg string
	}{
		{"(i) element type= xs:strng names xs:string",
			`<xs:element name="e" type="xs:strng"/>`,
			"element declaration e {type definition} references type " + xs + "strng, but no type definition with that expanded name is present in the schema; did you mean " + xs + "string? (src-resolve clause 1.1)"},
		{"(ii) nothing near adds no clause",
			`<xs:element name="e" type="nope"/>`,
			"element declaration e {type definition} references type nope, but no type definition with that expanded name is present in the schema (src-resolve clause 1.1)"},
		{"(iii) kind guard: an element declaration is no type",
			`<xs:element name="string"/><xs:element name="e" type="strng"/>`,
			"element declaration e {type definition} references type strng, but no type definition with that expanded name is present in the schema (src-resolve clause 1.1)"},
		{"(iv) namespace guard: xs:string is not in the absent namespace",
			`<xs:element name="e" type="strng"/>`,
			"element declaration e {type definition} references type strng, but no type definition with that expanded name is present in the schema (src-resolve clause 1.1)"},
		{"(v) threshold guard: ab is two edits from xy",
			`<xs:simpleType name="xy"><xs:restriction base="xs:string"/></xs:simpleType><xs:element name="e" type="ab"/>`,
			"element declaration e {type definition} references type ab, but no type definition with that expanded name is present in the schema (src-resolve clause 1.1)"},
		{"(vi) element ref= names the near declaration",
			`<xs:element name="foo"/><xs:complexType name="c"><xs:sequence><xs:element ref="fo"/></xs:sequence></xs:complexType>`,
			"particle {term} <element ref> references element declaration fo, but no element declaration with that expanded name is present in the schema; did you mean foo? (src-resolve clause 1.3)"},
		{"element type= names a complex type",
			`<xs:complexType name="foo"/><xs:element name="e" type="fo"/>`,
			"element declaration e {type definition} references type fo, but no type definition with that expanded name is present in the schema; did you mean foo? (src-resolve clause 1.1)"},
		{"element type= six runes allow two edits",
			`<xs:element name="e" type="xs:enyTyp"/>`,
			"element declaration e {type definition} references type " + xs + "enyTyp, but no type definition with that expanded name is present in the schema; did you mean " + xs + "anyType? (src-resolve clause 1.1)"},
		{"alternative type=",
			`<xs:simpleType name="foo"><xs:restriction base="xs:string"/></xs:simpleType><xs:element name="e"><xs:alternative test="@k" type="fo"/></xs:element>`,
			"element declaration e {type table}.{alternatives}[0].{type definition} references type fo, but no type definition with that expanded name is present in the schema; did you mean foo? (src-resolve clause 1.1)"},
		{"attribute type= names a simple type",
			`<xs:simpleType name="foo"><xs:restriction base="xs:string"/></xs:simpleType><xs:attribute name="a" type="fo"/>`,
			"attribute declaration a {type definition} references type fo, but no type definition with that expanded name is present in the schema; did you mean foo? (src-resolve clause 1.1)"},
		{"kind guard: attribute type= is offered no complex type",
			`<xs:complexType name="foo"/><xs:attribute name="a" type="fo"/>`,
			"attribute declaration a {type definition} references type fo, but no type definition with that expanded name is present in the schema (src-resolve clause 1.1)"},
		{"simple base=",
			`<xs:simpleType name="s"><xs:restriction base="xs:strng"/></xs:simpleType>`,
			"simple type s {base type definition} references simple type " + xs + "strng, but no type definition with that expanded name is present in the schema; did you mean " + xs + "string? (src-resolve clause 1.1)"},
		{"kind guard: simple base= is offered no complex type",
			`<xs:complexType name="foo"/><xs:simpleType name="s"><xs:restriction base="fo"/></xs:simpleType>`,
			"simple type s {base type definition} references simple type fo, but no type definition with that expanded name is present in the schema (src-resolve clause 1.1)"},
		{"itemType=",
			`<xs:simpleType name="l"><xs:list itemType="xs:it"/></xs:simpleType>`,
			"simple type l {item type definition} references simple type " + xs + "it, but no type definition with that expanded name is present in the schema; did you mean " + xs + "int? (src-resolve clause 1.1)"},
		{"memberTypes=",
			`<xs:simpleType name="u"><xs:union memberTypes="xs:int xs:strng"/></xs:simpleType>`,
			"simple type u {member type definitions}[1] references simple type " + xs + "strng, but no type definition with that expanded name is present in the schema; did you mean " + xs + "string? (src-resolve clause 1.1)"},
		{"complex base=",
			`<xs:complexType name="foo"/><xs:complexType name="c"><xs:complexContent><xs:extension base="fo"/></xs:complexContent></xs:complexType>`,
			"complex type c {base type definition} <extension base> references type fo, but no type definition with that expanded name is present in the schema; did you mean foo? (src-resolve clause 1.1)"},
		{"complex base= names xs:anyType",
			`<xs:complexType name="c"><xs:complexContent><xs:restriction base="xs:anyTyp"/></xs:complexContent></xs:complexType>`,
			"complex type c {base type definition} <restriction base> references type " + xs + "anyTyp, but no type definition with that expanded name is present in the schema; did you mean " + xs + "anyType? (src-resolve clause 1.1)"},
		{"tie: element type= names the earlier declaration",
			`<xs:simpleType name="fob"><xs:restriction base="xs:string"/></xs:simpleType><xs:complexType name="foa"/><xs:element name="e" type="fo"/>`,
			"element declaration e {type definition} references type fo, but no type definition with that expanded name is present in the schema; did you mean fob? (src-resolve clause 1.1)"},
		{"tie: complex base= names the earlier declaration",
			`<xs:simpleType name="fob"><xs:restriction base="xs:string"/></xs:simpleType><xs:complexType name="foa"/><xs:complexType name="c"><xs:simpleContent><xs:extension base="fo"/></xs:simpleContent></xs:complexType>`,
			"complex type c {base type definition} <extension base> references type fo, but no type definition with that expanded name is present in the schema; did you mean fob? (src-resolve clause 1.1)"},
		{"attribute ref= names a seeded xsi: declaration",
			`<xs:complexType name="c" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"><xs:attribute ref="xsi:typ"/></xs:complexType>`,
			"attribute use <attribute ref> references attribute declaration xsi:typ, but no attribute declaration with that expanded name is present in the schema; did you mean xsi:type? (src-resolve clause 1.2)"},
		{"attribute ref=",
			`<xs:attribute name="foo"/><xs:complexType name="c"><xs:attribute ref="fo"/></xs:complexType>`,
			"attribute use <attribute ref> references attribute declaration fo, but no attribute declaration with that expanded name is present in the schema; did you mean foo? (src-resolve clause 1.2)"},
		{"group ref=",
			`<xs:group name="foo"><xs:sequence/></xs:group><xs:complexType name="c"><xs:sequence><xs:group ref="fo"/></xs:sequence></xs:complexType>`,
			"particle {term} <group ref> references model group definition fo, but no model group definition with that expanded name is present in the schema; did you mean foo? (src-resolve clause 1.5)"},
		{"attributeGroup ref=",
			`<xs:attributeGroup name="foo"/><xs:complexType name="c"><xs:attributeGroup ref="fo"/></xs:complexType>`,
			"complex type c references attribute group definition fo — through an <attributeGroup ref> child, or, on a complex type, the reference §3.4.2.4 synthesizes from <schema defaultAttributes> — but no attribute group definition with that expanded name is present in the schema; did you mean foo? (src-resolve clause 1.4)"},
		{"defaultAttributes=",
			`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" defaultAttributes="fo"><xs:attributeGroup name="foo"/><xs:complexType name="c"/>` + tail,
			"complex type c references attribute group definition fo — through an <attributeGroup ref> child, or, on a complex type, the reference §3.4.2.4 synthesizes from <schema defaultAttributes> — but no attribute group definition with that expanded name is present in the schema; did you mean foo? (src-resolve clause 1.4)"},
		{"keyref refer=",
			`<xs:element name="e"><xs:complexType/><xs:key name="foo">` + keyBody + `</xs:key><xs:keyref name="kr" refer="fo">` + keyBody + `</xs:keyref></xs:element>`,
			"keyref kr references identity constraint fo, but no identity-constraint definition with that expanded name is present in the schema; did you mean foo? (src-resolve clause 1.7)"},
		{"kind guard: keyref refer= is offered no keyref",
			`<xs:element name="e"><xs:complexType/><xs:key name="k">` + keyBody + `</xs:key><xs:keyref name="foo" refer="k">` + keyBody + `</xs:keyref><xs:keyref name="kr" refer="fo">` + keyBody + `</xs:keyref></xs:element>`,
			"keyref kr references identity constraint fo, but no identity-constraint definition with that expanded name is present in the schema (src-resolve clause 1.7)"},
		{"key ref=",
			`<xs:element name="e"><xs:complexType/><xs:key name="foo">` + keyBody + `</xs:key></xs:element><xs:element name="f"><xs:complexType/><xs:key ref="fo"/></xs:element>`,
			`<key ref="fo"> resolves to no identity-constraint definition in the schema; did you mean "foo"? (src-resolve clause 1.7)`},
		{"kind guard: key ref= is offered no unique",
			`<xs:element name="e"><xs:complexType/><xs:unique name="foo">` + keyBody + `</xs:unique></xs:element><xs:element name="f"><xs:complexType/><xs:key ref="fo"/></xs:element>`,
			`<key ref="fo"> resolves to no identity-constraint definition in the schema (src-resolve clause 1.7)`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := tc.body
			if !strings.HasPrefix(doc, "<xs:schema") {
				doc = head + doc + tail
			}
			_, err := parseMap(t, "main.xsd", map[string]string{"main.xsd": doc})
			var xe *xsderr.Error
			if !errors.As(err, &xe) {
				t.Fatalf("Parse error = %v, want an *xsderr.Error", err)
			}
			if xe.Rule != "src-resolve" || xe.Msg != tc.msg {
				t.Errorf("got [%s] %s\nwant [src-resolve] %s", xe.Rule, xe.Msg, tc.msg)
			}
		})
	}
}
