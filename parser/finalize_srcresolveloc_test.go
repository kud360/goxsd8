package parser_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// TestFinalizeSrcResolveChargedAtNamingAttribute pins, end to end through
// Parse, that a finalize-time src-resolve rejection (§3.17.6.2) is charged at
// the QName-valued attribute naming the missing or wrong-kind component — the
// position the producer threads into the by-name reference's Loc — and not at
// the start tag of the element carrying it. Each naming attribute sits on a
// line of its own, so a row fails if the charge falls back to that element or
// to an enclosing component. msg pins which charge answered. memberTypes= is
// charged at the attribute's start: the reference carries no per-item position.
//
// The two guard rows pin charges that stay at a start tag: resolveKeyref's
// c-props-correct clause 2 charge faults the keyref, not its refer= name, and
// the <attributeGroup> reference §3.4.2.4 synthesizes from <schema
// defaultAttributes> is no attribute of the complex type it is charged against.
//
// The redefine row's dangling <element ref> sits in the <group> b.xsd declares
// and main.xsd redefines; the reference travels with the original into the
// redefinition, so the charge names b.xsd.
func TestFinalizeSrcResolveChargedAtNamingAttribute(t *testing.T) {
	const head = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">` + "\n"
	const tail = "\n</xs:schema>"
	one := func(body string) map[string]string { return map[string]string{"main.xsd": head + body + tail} }
	for _, tc := range []struct {
		name string
		docs map[string]string
		rule xsderr.Rule
		want xsderr.Loc
		msg  string
	}{
		{"element type=", one(`<xs:element name="e"` + "\n" +
			`  type="nope"/>`),
			"src-resolve", xsderr.Loc{URI: "main.xsd", Line: 3, Col: 3},
			"element declaration e {type definition} references type nope,"},
		{"attribute type= dangling", one(`<xs:attribute name="a"` + "\n" +
			`  type="nope"/>`),
			"src-resolve", xsderr.Loc{URI: "main.xsd", Line: 3, Col: 3},
			"attribute declaration a {type definition} references type nope,"},
		{"attribute type= naming a complex type", one(`<xs:complexType name="C"/>` + "\n" +
			`<xs:attribute name="a"` + "\n" +
			`  type="C"/>`),
			"src-resolve", xsderr.Loc{URI: "main.xsd", Line: 4, Col: 3},
			"attribute declaration a {type definition} references simple type C, but that expanded name is a complex type definition"},
		{"alternative type=", one(`<xs:element name="e">` + "\n" +
			`  <xs:alternative test="@k"` + "\n" +
			`    type="nope"/>` + "\n" +
			`</xs:element>`),
			"src-resolve", xsderr.Loc{URI: "main.xsd", Line: 4, Col: 5},
			"references type nope,"},
		{"simple base= dangling", one(`<xs:simpleType name="s">` + "\n" +
			`  <xs:restriction` + "\n" +
			`    base="nope"/>` + "\n" +
			`</xs:simpleType>`),
			"src-resolve", xsderr.Loc{URI: "main.xsd", Line: 4, Col: 5},
			"simple type s {base type definition} references simple type nope, but no type definition"},
		{"simple base= naming a complex type", one(`<xs:complexType name="C"/>` + "\n" +
			`<xs:simpleType name="s">` + "\n" +
			`  <xs:restriction` + "\n" +
			`    base="C"/>` + "\n" +
			`</xs:simpleType>`),
			"src-resolve", xsderr.Loc{URI: "main.xsd", Line: 5, Col: 5},
			"simple type s {base type definition} references simple type C, but that expanded name is a complex type definition"},
		{"itemType=", one(`<xs:simpleType name="l">` + "\n" +
			`  <xs:list` + "\n" +
			`    itemType="nope"/>` + "\n" +
			`</xs:simpleType>`),
			"src-resolve", xsderr.Loc{URI: "main.xsd", Line: 4, Col: 5},
			"simple type l {item type definition} references simple type nope,"},
		{"memberTypes=", one(`<xs:simpleType name="u">` + "\n" +
			`  <xs:union` + "\n" +
			`    memberTypes="xs:int nope"/>` + "\n" +
			`</xs:simpleType>`),
			"src-resolve", xsderr.Loc{URI: "main.xsd", Line: 4, Col: 5},
			"simple type u {member type definitions}[1] references simple type nope,"},
		{"element ref=", one(`<xs:complexType name="c">` + "\n" +
			`  <xs:sequence>` + "\n" +
			`    <xs:element` + "\n" +
			`      ref="nope"/>` + "\n" +
			`  </xs:sequence>` + "\n" +
			`</xs:complexType>`),
			"src-resolve", xsderr.Loc{URI: "main.xsd", Line: 5, Col: 7},
			"particle {term} <element ref> references element declaration nope,"},
		{"group ref=", one(`<xs:complexType name="c">` + "\n" +
			`  <xs:sequence>` + "\n" +
			`    <xs:group` + "\n" +
			`      ref="nope"/>` + "\n" +
			`  </xs:sequence>` + "\n" +
			`</xs:complexType>`),
			"src-resolve", xsderr.Loc{URI: "main.xsd", Line: 5, Col: 7},
			"particle {term} <group ref> references model group definition nope,"},
		{"attribute ref=", one(`<xs:complexType name="c">` + "\n" +
			`  <xs:attribute` + "\n" +
			`    ref="nope"/>` + "\n" +
			`</xs:complexType>`),
			"src-resolve", xsderr.Loc{URI: "main.xsd", Line: 4, Col: 5},
			"attribute use <attribute ref> references attribute declaration nope,"},
		{"attributeGroup ref=", one(`<xs:complexType name="c">` + "\n" +
			`  <xs:attributeGroup` + "\n" +
			`    ref="nope"/>` + "\n" +
			`</xs:complexType>`),
			"src-resolve", xsderr.Loc{URI: "main.xsd", Line: 4, Col: 5},
			"references attribute group definition nope "},
		{"keyref refer=", one(`<xs:element name="e">` + "\n" +
			`  <xs:complexType/>` + "\n" +
			`  <xs:keyref name="kr"` + "\n" +
			`    refer="nope">` + "\n" +
			`    <xs:selector xpath="."/>` + "\n" +
			`    <xs:field xpath="@a"/>` + "\n" +
			`  </xs:keyref>` + "\n" +
			`</xs:element>`),
			"src-resolve", xsderr.Loc{URI: "main.xsd", Line: 5, Col: 5},
			"keyref kr references identity constraint nope,"},
		{"guard: keyref c-props-correct clause 2 stays at the <keyref>", one(`<xs:element name="e">` + "\n" +
			`  <xs:complexType/>` + "\n" +
			`  <xs:key name="k"><xs:selector xpath="."/><xs:field xpath="@a"/></xs:key>` + "\n" +
			`  <xs:keyref name="kr"` + "\n" +
			`    refer="k">` + "\n" +
			`    <xs:selector xpath="."/>` + "\n" +
			`    <xs:field xpath="@a"/>` + "\n" +
			`    <xs:field xpath="@b"/>` + "\n" +
			`  </xs:keyref>` + "\n" +
			`</xs:element>`),
			"c-props-correct", xsderr.Loc{URI: "main.xsd", Line: 5, Col: 3},
			"c-props-correct clause 2 requires equal cardinality"},
		{"guard: the defaultAttributes reference stays at the complex type", map[string]string{"main.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"` + "\n" +
			`  defaultAttributes="nope">` + "\n" +
			`  <xs:complexType name="c"/>` + tail},
			"src-resolve", xsderr.Loc{URI: "main.xsd", Line: 3, Col: 3},
			"complex type c references attribute group definition nope "},
		{"redefine: element ref= in a redefined group's original", map[string]string{
			"main.xsd": head +
				`<xs:redefine schemaLocation="b.xsd">` + "\n" +
				`  <xs:group name="g">` + "\n" +
				`    <xs:sequence>` + "\n" +
				`      <xs:group ref="g"/>` + "\n" +
				`    </xs:sequence>` + "\n" +
				`  </xs:group>` + "\n" +
				`</xs:redefine>` + tail,
			"b.xsd": head +
				`<xs:group name="g">` + "\n" +
				`  <xs:sequence>` + "\n" +
				`    <xs:element` + "\n" +
				`      ref="nope"/>` + "\n" +
				`  </xs:sequence>` + "\n" +
				`</xs:group>` + tail},
			"src-resolve", xsderr.Loc{URI: "b.xsd", Line: 5, Col: 7},
			"particle {term} <element ref> references element declaration nope,"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseMap(t, "main.xsd", tc.docs)
			var xe *xsderr.Error
			if !errors.As(err, &xe) {
				t.Fatalf("Parse error = %v, want an *xsderr.Error", err)
			}
			if xe.Rule != tc.rule {
				t.Fatalf("rule = %q, want %q (%v)", xe.Rule, tc.rule, err)
			}
			if xe.Loc != tc.want {
				t.Errorf("loc = %s, want %s (%v)", xe.Loc, tc.want, err)
			}
			if !strings.Contains(err.Error(), tc.msg) {
				t.Errorf("error = %v, want it to contain %q", err, tc.msg)
			}
		})
	}
}
