package codegen

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin"
	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/loader"
	xsdparser "github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// parseString parses doc as the one schema document at location s.xsd.
func parseString(t *testing.T, doc string) *xsd.Schema {
	t.Helper()
	resolve := loader.ResolverFunc(func(_, location string) (io.ReadCloser, string, error) {
		if location != "s.xsd" {
			return nil, "", fmt.Errorf("no document %q", location)
		}
		return io.NopCloser(strings.NewReader(doc)), location, nil
	})
	schema, err := xsdparser.Parse("s.xsd", xsdparser.WithResolver(resolve))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return schema
}

// TestGenerateDeclines pins each construct outside the slice: Generate
// returns a *DeclineError of the row's kind, wrapping errors.ErrUnsupported,
// whose text names the construct and its Loc, and writes nothing. Each body
// is one line, so every Loc is on line 2.
func TestGenerateDeclines(t *testing.T) {
	const str = `type="xs:string"`
	tests := []struct {
		name, body, detail string
		kind               construct
	}{
		{"empty choice", `<xs:element name="e"><xs:complexType><xs:choice/></xs:complexType></xs:element>`,
			"complex type (anonymous) of element e whose choice has no {particles}", constructEmptyChoice},
		{"choice of vacuous elements", `<xs:element name="e"><xs:complexType><xs:choice><xs:element name="a" minOccurs="0" maxOccurs="0" ` + str + `/></xs:choice></xs:complexType></xs:element>`,
			"complex type (anonymous) of element e whose choice has no {particles}", constructEmptyChoice},
		{"emptiable choice", `<xs:element name="e"><xs:complexType><xs:choice><xs:element name="a" ` + str + `/><xs:element name="b" minOccurs="0" ` + str + `/></xs:choice></xs:complexType></xs:element>`,
			"complex type (anonymous) of element e whose choice is emptiable: its alternative element b has {min occurs} 0", constructEmptiableChoice},
		{"optional choice", `<xs:element name="e"><xs:complexType><xs:choice minOccurs="0"><xs:element name="a" ` + str + `/></xs:choice></xs:complexType></xs:element>`,
			"whose content particle occurs 0..1 times", constructContentOccurs},
		{"repeated choice", `<xs:element name="e"><xs:complexType><xs:choice maxOccurs="2"><xs:element name="a" ` + str + `/></xs:choice></xs:complexType></xs:element>`,
			"whose content particle occurs 1..2 times", constructContentOccurs},
		{"group in choice", `<xs:element name="e"><xs:complexType><xs:choice><xs:element name="a" ` + str + `/><xs:sequence><xs:element name="b" ` + str + `/></xs:sequence></xs:choice></xs:complexType></xs:element>`,
			"a nested sequence model group particle", constructNestedGroup},
		{"emptiable group in choice", `<xs:element name="e"><xs:complexType><xs:choice><xs:element name="a" ` + str + `/><xs:sequence><xs:element name="b" minOccurs="0" ` + str + `/></xs:sequence></xs:choice></xs:complexType></xs:element>`,
			"a nested sequence model group particle", constructNestedGroup},
		{"element ref in choice", `<xs:element name="h" ` + str + `/><xs:element name="e"><xs:complexType><xs:choice><xs:element name="a" ` + str + `/><xs:element ref="h"/></xs:choice></xs:complexType></xs:element>`,
			"an <element ref> particle to h", constructElementRef},
		{"group ref in choice", `<xs:group name="g"><xs:sequence><xs:element name="b" ` + str + `/></xs:sequence></xs:group><xs:element name="e"><xs:complexType><xs:choice><xs:element name="a" ` + str + `/><xs:group ref="g"/></xs:choice></xs:complexType></xs:element>`,
			"a <group ref> particle to g", constructGroupRef},
		{"wildcard in choice", `<xs:element name="e"><xs:complexType><xs:choice><xs:element name="a" ` + str + `/><xs:any/></xs:choice></xs:complexType></xs:element>`,
			"a wildcard particle", constructWildcard},
		{"mixed choice", `<xs:element name="e"><xs:complexType mixed="true"><xs:choice><xs:element name="a" ` + str + `/></xs:choice></xs:complexType></xs:element>`,
			"with mixed content", constructMixedContent},
		{"open content choice", `<xs:element name="e"><xs:complexType><xs:openContent><xs:any/></xs:openContent><xs:choice><xs:element name="a" ` + str + `/></xs:choice></xs:complexType></xs:element>`,
			"with {open content}", constructOpenContent},
		{"extension of a choice", `<xs:complexType name="b"><xs:choice><xs:element name="a" ` + str + `/></xs:choice></xs:complexType><xs:element name="e"><xs:complexType><xs:complexContent><xs:extension base="b"><xs:choice><xs:element name="c" ` + str + `/></xs:choice></xs:extension></xs:complexContent></xs:complexType></xs:element>`,
			"derived by extension", constructDerivation},
		{"choice alternative type", `<xs:element name="e"><xs:complexType><xs:choice><xs:element name="a" type="xs:decimal"/></xs:choice></xs:complexType></xs:element>`,
			"element a with {type definition} {http://www.w3.org/2001/XMLSchema}decimal, whose {primitive type definition} is", constructUncoveredPrimitive},
		{"all", `<xs:element name="e"><xs:complexType><xs:all><xs:element name="a" ` + str + `/></xs:all></xs:complexType></xs:element>`,
			"with an all compositor", constructAll},
		{"nested group", `<xs:element name="e"><xs:complexType><xs:sequence><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:sequence></xs:complexType></xs:element>`,
			"a nested sequence model group particle", constructNestedGroup},
		{"group ref particle", `<xs:group name="g"><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:group><xs:element name="e"><xs:complexType><xs:sequence><xs:group ref="g"/></xs:sequence></xs:complexType></xs:element>`,
			"a <group ref> particle to g", constructGroupRef},
		{"group ref content", `<xs:group name="g"><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:group><xs:element name="e"><xs:complexType><xs:group ref="g"/></xs:complexType></xs:element>`,
			"whose content particle is a <group ref>", constructContentGroupRef},
		{"element ref", `<xs:element name="h" ` + str + `/><xs:element name="e"><xs:complexType><xs:sequence><xs:element ref="h"/></xs:sequence></xs:complexType></xs:element>`,
			"an <element ref> particle to h", constructElementRef},
		{"wildcard", `<xs:element name="e"><xs:complexType><xs:sequence><xs:any/></xs:sequence></xs:complexType></xs:element>`,
			"a wildcard particle", constructWildcard},
		{"attribute", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence><xs:attribute name="t"/></xs:complexType></xs:element>`,
			"attribute t of complex type (anonymous) of element e with {type definition} {http://www.w3.org/2001/XMLSchema}anySimpleType, a special type", constructSpecialType},
		{"attribute anyAtomicType", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence><xs:attribute name="t" type="xs:anyAtomicType"/></xs:complexType></xs:element>`,
			"attribute t of complex type (anonymous) of element e with {type definition} {http://www.w3.org/2001/XMLSchema}anyAtomicType, a special type", constructSpecialType},
		{"attribute list of uncovered primitive", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence><xs:attribute name="t"><xs:simpleType><xs:list itemType="xs:decimal"/></xs:simpleType></xs:attribute></xs:complexType></xs:element>`,
			"attribute t of complex type (anonymous) of element e with {type definition} (anonymous), whose {item type definition} is {http://www.w3.org/2001/XMLSchema}decimal, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}decimal", constructUncoveredPrimitive},
		{"attribute union", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence><xs:attribute name="t"><xs:simpleType><xs:union memberTypes="xs:string xs:int"/></xs:simpleType></xs:attribute></xs:complexType></xs:element>`,
			"attribute t of complex type (anonymous) of element e with {type definition} (anonymous), a union type", constructUnionType},
		{"attribute uncovered primitive", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence><xs:attribute name="t" type="xs:decimal"/></xs:complexType></xs:element>`,
			"attribute t of complex type (anonymous) of element e with {type definition} {http://www.w3.org/2001/XMLSchema}decimal, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}decimal", constructUncoveredPrimitive},
		{"attribute QName", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence><xs:attribute name="t" type="xs:QName"/></xs:complexType></xs:element>`,
			"attribute t of complex type (anonymous) of element e with {type definition} {http://www.w3.org/2001/XMLSchema}QName, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}QName", constructUncoveredPrimitive},
		{"attribute default", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence><xs:attribute name="t" default="d" ` + str + `/></xs:complexType></xs:element>`,
			"attribute t of complex type (anonymous) of element e with a default effective value constraint (§3.5.4 key-evc)", constructAttributeValueConstraint},
		{"required attribute fixed", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence><xs:attribute name="t" use="required" fixed="f" ` + str + `/></xs:complexType></xs:element>`,
			"attribute t of complex type (anonymous) of element e with a fixed effective value constraint (§3.5.4 key-evc)", constructAttributeValueConstraint},
		// The two ref rows carry the value constraint on the global declaration
		// only, so they fail when the decline reads the use's own {value
		// constraint} rather than its effective one (§3.5.4 key-evc).
		{"attribute ref default", `<xs:attribute name="t" default="d" ` + str + `/>` + `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence><xs:attribute ref="t"/></xs:complexType></xs:element>`,
			"attribute t of complex type (anonymous) of element e with a default effective value constraint (§3.5.4 key-evc)", constructAttributeValueConstraint},
		{"attribute ref fixed", `<xs:attribute name="t" fixed="f" ` + str + `/>` + `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence><xs:attribute ref="t"/></xs:complexType></xs:element>`,
			"attribute t of complex type (anonymous) of element e with a fixed effective value constraint (§3.5.4 key-evc)", constructAttributeValueConstraint},
		// The empty-content rows reach attributeField with no particle field
		// before it, and contentGroup's checks ahead of its content-type switch.
		{"empty type attribute default", `<xs:element name="e"><xs:complexType><xs:attribute name="t" default="d" ` + str + `/></xs:complexType></xs:element>`,
			"attribute t of complex type (anonymous) of element e with a default effective value constraint (§3.5.4 key-evc)", constructAttributeValueConstraint},
		{"empty type attribute ref fixed", `<xs:attribute name="t" fixed="f" ` + str + `/>` + `<xs:element name="e"><xs:complexType><xs:attribute ref="t"/></xs:complexType></xs:element>`,
			"attribute t of complex type (anonymous) of element e with a fixed effective value constraint (§3.5.4 key-evc)", constructAttributeValueConstraint},
		{"empty type abstract", `<xs:complexType name="c" abstract="true"/><xs:element name="e" type="c"/>`,
			"complex type c of element e with {abstract} true", constructComplexTypeAbstract},
		{"empty type assertion", `<xs:element name="e"><xs:complexType><xs:assert test="true()"/></xs:complexType></xs:element>`,
			"with {assertions}", constructAssertions},
		{"empty type attribute wildcard", `<xs:element name="e"><xs:complexType><xs:anyAttribute/></xs:complexType></xs:element>`,
			"with an {attribute wildcard}", constructAttributeWildcard},
		{"empty type extension", `<xs:complexType name="b"/><xs:element name="e"><xs:complexType><xs:complexContent><xs:extension base="b"/></xs:complexContent></xs:complexType></xs:element>`,
			"derived by extension", constructDerivation},
		{"empty type restriction of another base", `<xs:complexType name="b"/><xs:element name="e"><xs:complexType><xs:complexContent><xs:restriction base="b"/></xs:complexContent></xs:complexType></xs:element>`,
			"a restriction of b rather than of xs:anyType", constructRestrictionBase},
		{"mixed empty", `<xs:element name="e"><xs:complexType mixed="true"/></xs:element>`,
			"with mixed content", constructMixedContent},
		{"default open content applies to empty", `<xs:defaultOpenContent appliesToEmpty="true"><xs:any/></xs:defaultOpenContent><xs:element name="e"><xs:complexType/></xs:element>`,
			"with {open content}", constructOpenContent},
		{"choice with attribute", `<xs:element name="e"><xs:complexType><xs:choice><xs:element name="a" ` + str + `/></xs:choice><xs:attribute name="t" ` + str + `/></xs:complexType></xs:element>`,
			"complex type (anonymous) of element e whose content is a choice, with {attribute uses}", constructChoiceAttributeUses},
		{"attribute wildcard", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence><xs:anyAttribute/></xs:complexType></xs:element>`,
			"with an {attribute wildcard}", constructAttributeWildcard},
		{"mixed", `<xs:element name="e"><xs:complexType mixed="true"><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:complexType></xs:element>`,
			"with mixed content", constructMixedContent},
		{"simple content", `<xs:element name="e"><xs:complexType><xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent></xs:complexType></xs:element>`,
			"derived by extension", constructDerivation},
		{"simple content restriction", `<xs:complexType name="b"><xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent></xs:complexType><xs:element name="e"><xs:complexType><xs:simpleContent><xs:restriction base="b"/></xs:simpleContent></xs:complexType></xs:element>`,
			"a restriction of b rather than of xs:anyType", constructRestrictionBase},
		{"simple content restriction of anyType", `<xs:element name="e"><xs:complexType><xs:simpleContent><xs:restriction base="xs:anyType"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:restriction></xs:simpleContent></xs:complexType></xs:element>`,
			"complex type (anonymous) of element e with simple content", constructSimpleContent},
		{"open content", `<xs:element name="e"><xs:complexType><xs:openContent><xs:any/></xs:openContent><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:complexType></xs:element>`,
			"with {open content}", constructOpenContent},
		{"default open content", `<xs:defaultOpenContent><xs:any/></xs:defaultOpenContent><xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:complexType></xs:element>`,
			"with {open content}", constructOpenContent},
		{"extension", `<xs:complexType name="b"><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:complexType><xs:element name="e"><xs:complexType><xs:complexContent><xs:extension base="b"/></xs:complexContent></xs:complexType></xs:element>`,
			"derived by extension", constructDerivation},
		{"restriction of another base", `<xs:complexType name="b"><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:complexType><xs:element name="e"><xs:complexType><xs:complexContent><xs:restriction base="b"><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:restriction></xs:complexContent></xs:complexType></xs:element>`,
			"a restriction of b rather than of xs:anyType", constructRestrictionBase},
		{"outer occurrence", `<xs:element name="e"><xs:complexType><xs:sequence minOccurs="0"><xs:element name="a" ` + str + `/></xs:sequence></xs:complexType></xs:element>`,
			"whose content particle occurs 0..1 times", constructContentOccurs},
		{"nillable", `<xs:element name="e" nillable="true" ` + str + `/>`,
			"element e with {nillable} true", constructElementNillable},
		{"abstract element", `<xs:element name="e" abstract="true" ` + str + `/>`,
			"element e with {abstract} true", constructElementAbstract},
		{"value constraint", `<xs:element name="e" default="d" ` + str + `/>`,
			"element e with a {value constraint}", constructElementValueConstraint},
		{"type table", `<xs:element name="e" ` + str + `><xs:alternative test="true()" type="xs:token"/></xs:element>`,
			"element e with a {type table}", constructTypeTable},
		{"substitution group", `<xs:element name="h" ` + str + `/><xs:element name="e" substitutionGroup="h" ` + str + `/>`,
			"element e with {substitution group affiliations}", constructSubstitutionGroup},
		{"assertion", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence><xs:assert test="true()"/></xs:complexType></xs:element>`,
			"with {assertions}", constructAssertions},
		{"abstract complex type", `<xs:complexType name="c" abstract="true"><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:complexType><xs:element name="e" type="c"/>`,
			"complex type c of element e with {abstract} true", constructComplexTypeAbstract},
		{"anyType", `<xs:element name="e"/>`,
			"element e with {type definition} xs:anyType", constructAnyTypeElement},
		{"anySimpleType", `<xs:element name="e" type="xs:anySimpleType"/>`,
			"{http://www.w3.org/2001/XMLSchema}anySimpleType, a special type", constructSpecialType},
		{"anyAtomicType", `<xs:element name="e" type="xs:anyAtomicType"/>`,
			"{http://www.w3.org/2001/XMLSchema}anyAtomicType, a special type", constructSpecialType},
		// The list rows pin "# Coverage"'s list decline, and that its message
		// names the list and then its {item type definition}.
		{"list of named union", `<xs:simpleType name="u"><xs:union memberTypes="xs:string xs:int"/></xs:simpleType><xs:simpleType name="l"><xs:list itemType="u"/></xs:simpleType><xs:element name="e" type="l"/>`,
			"element e with {type definition} l, whose {item type definition} is u, a union type", constructUnionType},
		{"list of anonymous union", `<xs:element name="e"><xs:simpleType><xs:list><xs:simpleType><xs:union memberTypes="xs:string xs:int"/></xs:simpleType></xs:list></xs:simpleType></xs:element>`,
			"element e with {type definition} (anonymous), whose {item type definition} is (anonymous), a union type", constructUnionType},
		{"restriction of a list of union", `<xs:simpleType name="u"><xs:union memberTypes="xs:string xs:int"/></xs:simpleType><xs:simpleType name="l"><xs:list itemType="u"/></xs:simpleType><xs:simpleType name="r"><xs:restriction base="l"><xs:maxLength value="2"/></xs:restriction></xs:simpleType><xs:element name="e" type="r"/>`,
			"element e with {type definition} r, whose {item type definition} is u, a union type", constructUnionType},
		{"list of uncovered primitive", `<xs:simpleType name="l"><xs:list itemType="xs:decimal"/></xs:simpleType><xs:element name="e" type="l"/>`,
			"element e with {type definition} l, whose {item type definition} is {http://www.w3.org/2001/XMLSchema}decimal, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}decimal", constructUncoveredPrimitive},
		{"union", `<xs:element name="e"><xs:simpleType><xs:union memberTypes="xs:string xs:int"/></xs:simpleType></xs:element>`,
			"element e with {type definition} (anonymous), a union type", constructUnionType},
		{"uncovered primitive", `<xs:element name="e" type="xs:decimal"/>`,
			"whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}decimal", constructUncoveredPrimitive},
		{"integer", `<xs:element name="e" type="xs:integer"/>`,
			"element e with {type definition} {http://www.w3.org/2001/XMLSchema}integer, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}decimal", constructUncoveredPrimitive},
		{"positiveInteger", `<xs:element name="e" type="xs:positiveInteger"/>`,
			"element e with {type definition} {http://www.w3.org/2001/XMLSchema}positiveInteger, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}decimal", constructUncoveredPrimitive},
		{"nonNegativeInteger", `<xs:element name="e" type="xs:nonNegativeInteger"/>`,
			"element e with {type definition} {http://www.w3.org/2001/XMLSchema}nonNegativeInteger, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}decimal", constructUncoveredPrimitive},
		{"nonPositiveInteger", `<xs:element name="e" type="xs:nonPositiveInteger"/>`,
			"element e with {type definition} {http://www.w3.org/2001/XMLSchema}nonPositiveInteger, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}decimal", constructUncoveredPrimitive},
		{"negativeInteger", `<xs:element name="e" type="xs:negativeInteger"/>`,
			"element e with {type definition} {http://www.w3.org/2001/XMLSchema}negativeInteger, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}decimal", constructUncoveredPrimitive},
		{"float", `<xs:element name="e" type="xs:float"/>`,
			"element e with {type definition} {http://www.w3.org/2001/XMLSchema}float, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}float", constructUncoveredPrimitive},
		{"double", `<xs:element name="e" type="xs:double"/>`,
			"element e with {type definition} {http://www.w3.org/2001/XMLSchema}double, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}double", constructUncoveredPrimitive},
		// The value-cycle rows each give Go types that would hold one another
		// as struct values.
		{"value cycle direct", `<xs:complexType name="c"><xs:sequence><xs:element name="x" type="c"/></xs:sequence></xs:complexType><xs:element name="e" type="c"/>`,
			"local element x reaching complex type c again through elements that each occur exactly once at s.xsd:2:39", constructValueCycle},
		{"value cycle mutual", `<xs:complexType name="c1"><xs:sequence><xs:element name="x" type="c2"/></xs:sequence></xs:complexType><xs:complexType name="c2"><xs:sequence><xs:element name="y" type="c1"/></xs:sequence></xs:complexType><xs:element name="e" type="c1"/>`,
			"local element x reaching complex type c2 again through elements that each occur exactly once", constructValueCycle},
		{"value cycle through anonymous type", `<xs:complexType name="c"><xs:sequence><xs:element name="x"><xs:complexType><xs:sequence><xs:element name="y" type="c"/></xs:sequence></xs:complexType></xs:element></xs:sequence></xs:complexType><xs:element name="e" type="c"/>`,
			"local element y reaching complex type c again through elements that each occur exactly once", constructValueCycle},
		{"value cycle in choice branch", `<xs:complexType name="t1"><xs:sequence><xs:element name="b" type="t2"/></xs:sequence></xs:complexType><xs:complexType name="t2"><xs:sequence><xs:element name="c" type="t1"/></xs:sequence></xs:complexType><xs:element name="e"><xs:complexType><xs:choice><xs:element name="a" type="t1"/></xs:choice></xs:complexType></xs:element>`,
			"local element c reaching complex type t1 again through elements that each occur exactly once", constructValueCycle},
		// x's walk emits z through the pointer field p before it reaches the
		// value field v, and z's walk meets x already stored: a check carried
		// by the emitting walk, which never walks a stored type again, lets
		// the cycle v, w through.
		{"value cycle behind a pointer", `<xs:complexType name="x"><xs:sequence><xs:element name="p" type="z" minOccurs="0"/><xs:element name="v" type="z"/></xs:sequence></xs:complexType><xs:complexType name="z"><xs:sequence><xs:element name="w" type="x"/></xs:sequence></xs:complexType><xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" type="x"/></xs:sequence></xs:complexType></xs:element>`,
			"local element w reaching complex type x again through elements that each occur exactly once", constructValueCycle},
		{"local element property", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="l" nillable="true" ` + str + `/></xs:sequence></xs:complexType></xs:element>`,
			"element l with {nillable} true", constructElementNillable},
		{"nested struct", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="l"><xs:complexType><xs:all><xs:element name="a" ` + str + `/></xs:all></xs:complexType></xs:element></xs:sequence></xs:complexType></xs:element>`,
			"complex type (anonymous) of element l with an all compositor", constructAll},
		{"nested choice", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="l"><xs:complexType><xs:choice><xs:element name="a" ` + str + `/><xs:any/></xs:choice></xs:complexType></xs:element></xs:sequence></xs:complexType></xs:element>`,
			"a wildcard particle in complex type (anonymous) of element l", constructWildcard},
		{"branch struct", `<xs:element name="e"><xs:complexType><xs:choice><xs:element name="l"><xs:complexType><xs:all><xs:element name="a" ` + str + `/></xs:all></xs:complexType></xs:element></xs:choice></xs:complexType></xs:element>`,
			"complex type (anonymous) of element l with an all compositor", constructAll},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := parseString(t, "<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\">\n"+tt.body+"\n</xs:schema>\n")
			out := filepath.Join(t.TempDir(), "out")
			err := Generate([]Target{{Schema: schema, Dir: out, Package: "p"}})
			if err == nil {
				t.Fatal("Generate succeeded, want a decline")
			}
			if !errors.Is(err, errors.ErrUnsupported) {
				t.Errorf("error %v does not wrap errors.ErrUnsupported", err)
			}
			var xe *xsderr.Error
			if errors.As(err, &xe) {
				t.Errorf("error %v is an *xsderr.Error, want a plain error", err)
			}
			var de *DeclineError
			if !errors.As(err, &de) {
				t.Fatalf("error %v is not a *DeclineError", err)
			}
			if got, want := de.Construct(), (Construct{tt.kind}); got != want {
				t.Errorf("Construct() = %v, want %v", got, want)
			}
			for _, want := range []string{"codegen: target 0: ", tt.detail, " at s.xsd:2:", "a generator limitation, not a schema fault"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not contain %q", err, want)
				}
			}
			if strings.Contains(err.Error(), "%!") {
				t.Errorf("error %q holds a misused format verb", err)
			}
			if _, err := os.Stat(out); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("Dir %s exists after a decline (stat: %v)", out, err)
			}
		})
	}
}

// TestGenerateDeclinesVacuousParticle pins the decline of a particle whose
// {max occurs} is 0. No schema document yields one, since an <element> with
// minOccurs=maxOccurs=0 maps to no component at all (§3.3.2), but
// xsd.NewParticle accepts a vacuous Occurs, so the schema is built through the
// constructors.
func TestGenerateDeclinesVacuousParticle(t *testing.T) {
	ctName := xsd.QName{Local: "c"}
	scope, err := xsd.NewLocalScope(xsderr.Loc{}, xsd.ComplexTypeScopeParent{Name: ctName})
	if err != nil {
		t.Fatal(err)
	}
	local, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "a"},
		xsd.TypeDefinitionRef{Name: stringPrimitive}, nil, scope, nil, false, nil, nil, nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	vacuous, err := xsd.NewOccurs(xsderr.Loc{}, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	inner, err := xsd.NewParticle(xsderr.Loc{}, vacuous, xsd.ResolvedTerm{Term: local})
	if err != nil {
		t.Fatal(err)
	}
	group, err := xsd.NewModelGroup(xsderr.Loc{}, xsd.CompositorSequence, []xsd.Particle{inner})
	if err != nil {
		t.Fatal(err)
	}
	outer, err := xsd.NewParticle(xsderr.Loc{}, exactlyOnce, xsd.ResolvedTerm{Term: group})
	if err != nil {
		t.Fatal(err)
	}
	ct, err := xsd.NewComplexType(xsderr.Loc{}, ctName, anyTypeName, nil, xsd.DerivationRestriction, false,
		nil, nil, nil, xsd.ElementContent{Particle: outer}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "e"},
		xsd.TypeDefinitionRef{Name: ctName}, nil, xsd.NewGlobalScope(), nil, false, nil, nil, nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	// The builtins and an xs:anyType stand-in are what the local element's
	// type and c's base resolve against.
	anyType, err := xsd.NewComplexType(xsderr.Loc{}, anyTypeName, anyTypeName, nil, xsd.DerivationRestriction, false,
		nil, nil, nil, xsd.EmptyContent{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	seeded, err := builtin.Seed(strict.New())
	if err != nil {
		t.Fatal(err)
	}
	b := xsd.NewSchemaBuilder()
	for _, st := range seeded {
		b.AddType(st)
	}
	b.AddType(anyType)
	b.AddType(ct)
	b.AddElement(e)
	schema, err := b.Finalize()
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	out := filepath.Join(t.TempDir(), "out")
	err = Generate([]Target{{Schema: schema, Dir: out, Package: "p"}})
	if !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("Generate = %v, want a decline wrapping errors.ErrUnsupported", err)
	}
	const want = "codegen: target 0: a particle of element a with {max occurs} 0 in complex type c of element e at "
	if !strings.HasPrefix(err.Error(), want) {
		t.Errorf("error %q does not start with %q", err, want)
	}
	var de *DeclineError
	if !errors.As(err, &de) || de.Construct() != (Construct{constructVacuousParticle}) {
		t.Errorf("Generate = %v, want a *DeclineError of Construct %v", err, Construct{constructVacuousParticle})
	}
	if _, err := os.Stat(out); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Dir %s exists after a decline (stat: %v)", out, err)
	}
}

// TestDeclineErrorAs pins that errors.As retrieves the *DeclineError from the
// error Generate returns for a named and an unnamed target, with its Construct
// and Loc as values, and that the target's name opens the error outside the
// *DeclineError's own text.
func TestDeclineErrorAs(t *testing.T) {
	schema := parseString(t, "<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\">\n<xs:element name=\"e\" type=\"xs:decimal\"/>\n</xs:schema>\n")
	tests := []struct {
		name, targetName, opening string
	}{
		{"unnamed", "", "codegen: target 0: "},
		{"named", "first", "first: "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Generate([]Target{{Schema: schema, Dir: filepath.Join(t.TempDir(), "out"), Package: "p", Name: tt.targetName}})
			var de *DeclineError
			if !errors.As(err, &de) {
				t.Fatalf("Generate = %v, want an error errors.As retrieves a *DeclineError from", err)
			}
			if got, want := de.Construct(), (Construct{constructUncoveredPrimitive}); got != want {
				t.Errorf("Construct() = %v, want %v", got, want)
			}
			if loc := de.Loc(); loc.URI != "s.xsd" || loc.Line != 2 {
				t.Errorf("Loc() = %+v, want URI s.xsd, Line 2", loc)
			}
			if want := tt.opening + de.Error(); err.Error() != want {
				t.Errorf("error %q, want %q", err, want)
			}
			const detail = "element e with {type definition} {http://www.w3.org/2001/XMLSchema}decimal, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}decimal at "
			if !strings.HasPrefix(de.Error(), detail+de.Loc().String()+" is not supported by codegen") {
				t.Errorf("DeclineError.Error() = %q, want it to open %q followed by its Loc", de.Error(), detail)
			}
		})
	}
}

// TestConstructString pins that every construct's String is non-empty and
// distinct, and that the zero Construct and one past the last render a
// diagnostic form.
func TestConstructString(t *testing.T) {
	names := make(map[string]construct)
	for c := construct(1); int(c) < len(constructs); c++ {
		name := Construct{c}.String()
		if name == "" {
			t.Errorf("Construct{%d}.String() is empty", c)
		}
		if prior, ok := names[name]; ok {
			t.Errorf("Construct{%d}.String() = %q, the same as Construct{%d}'s", c, name, prior)
		}
		names[name] = c
		if constructs[c].format == "" {
			t.Errorf("construct %d (%s) has no format", c, name)
		}
	}
	if got := len(names); got != int(constructWildcard) {
		t.Errorf("walked %d constructs, want %d", got, constructWildcard)
	}
	for _, c := range []construct{0, construct(len(constructs))} {
		if got, want := (Construct{c}).String(), fmt.Sprintf("Construct(%d)", c); got != want {
			t.Errorf("Construct{%d}.String() = %q, want %q", c, got, want)
		}
	}
}
