package codegen

import (
	"errors"
	"fmt"
	"go/doc/comment"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"slices"
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
	const (
		str         = `type="xs:string"`
		pastDecline = `<xs:complexType name="m"><xs:sequence><xs:element name="q" type="t"/><xs:element name="r" type="xs:precisionDecimal"/></xs:sequence></xs:complexType><xs:complexType name="t"><xs:sequence><xs:element name="f1" type="m" minOccurs="0"/>%s</xs:sequence></xs:complexType>%s<xs:element name="root"><xs:complexType><xs:sequence><xs:element name="x" type="t"/></xs:sequence></xs:complexType></xs:element>`
		uncoveredR  = "element r with {type definition} {http://www.w3.org/2001/XMLSchema}precisionDecimal, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}precisionDecimal at s.xsd:2:70 "
	)
	tests := []struct {
		name, body, detail string
		kind               construct
	}{
		{"empty choice", `<xs:element name="e"><xs:complexType><xs:choice/></xs:complexType></xs:element>`,
			"complex type (anonymous) of element e whose choice has no {particles}", constructEmptyChoice},
		// An attribute makes the type a struct (shape) without lifting the
		// decline: its choice field's occurrence type still has no {particles}.
		{"empty choice with attribute", `<xs:element name="e"><xs:complexType><xs:choice/><xs:attribute name="t" ` + str + `/></xs:complexType></xs:element>`,
			"complex type (anonymous) of element e whose choice has no {particles}", constructEmptyChoice},
		{"choice of vacuous elements", `<xs:element name="e"><xs:complexType><xs:choice><xs:element name="a" minOccurs="0" maxOccurs="0" ` + str + `/></xs:choice></xs:complexType></xs:element>`,
			"complex type (anonymous) of element e whose choice has no {particles}", constructEmptyChoice},
		// A model group alternative whose term is emptiable declines, inline
		// or through a <group ref>.
		{"emptiable group in choice", `<xs:element name="e"><xs:complexType><xs:choice><xs:element name="a" ` + str + `/><xs:sequence><xs:element name="b" minOccurs="0" ` + str + `/></xs:sequence></xs:choice></xs:complexType></xs:element>`,
			"complex type (anonymous) of element e whose choice's alternative 2, a sequence model group particle, has an emptiable model group", constructEmptiableGroupAlternative},
		{"emptiable choice in choice", `<xs:element name="e"><xs:complexType><xs:choice><xs:element name="a" ` + str + `/><xs:choice><xs:element name="b" ` + str + `/><xs:element name="c" minOccurs="0" ` + str + `/></xs:choice></xs:choice></xs:complexType></xs:element>`,
			"complex type (anonymous) of element e whose choice's alternative 2, a choice model group particle, has an emptiable model group", constructEmptiableGroupAlternative},
		{"emptiable group ref in choice", `<xs:group name="g"><xs:sequence><xs:element name="b" minOccurs="0" ` + str + `/></xs:sequence></xs:group><xs:element name="e"><xs:complexType><xs:choice><xs:element name="a" ` + str + `/><xs:group ref="g"/></xs:choice></xs:complexType></xs:element>`,
			"complex type (anonymous) of element e whose choice's alternative 2, a sequence model group particle, has an emptiable model group", constructEmptiableGroupAlternative},
		// The alternative's own {min occurs} 0 does not lift the decline: the
		// gate reads the term.
		{"optional emptiable group in choice", `<xs:element name="e"><xs:complexType><xs:choice><xs:element name="a" ` + str + `/><xs:sequence minOccurs="0"><xs:element name="b" minOccurs="0" ` + str + `/></xs:sequence></xs:choice></xs:complexType></xs:element>`,
			"complex type (anonymous) of element e whose choice's alternative 2, a sequence model group particle, has an emptiable model group", constructEmptiableGroupAlternative},
		// A sequence repeating past 1 whose one occurrence is emptiable
		// declines, as the content particle or nested, inline or through a
		// <group ref>, whatever its {min occurs}, and nested with no
		// {particles}, an empty content <sequence> giving empty content
		// instead (§3.4.2.3.3 clause 2.1.2, key-expcon); it is met
		// where its field would be, ahead of z after it and of a declining
		// particle inside it.
		{"repeated emptiable sequence", `<xs:element name="e"><xs:complexType><xs:sequence minOccurs="0" maxOccurs="unbounded"><xs:element name="a" minOccurs="0" ` + str + `/></xs:sequence></xs:complexType></xs:element>`,
			"complex type (anonymous) of element e holding a sequence model group particle with {max occurs} above 1 whose sequence is emptiable at s.xsd:2:22 ", constructRepeatedEmptiableSequence},
		{"nested repeated emptiable sequence", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="h" ` + str + `/><xs:sequence maxOccurs="2"><xs:element name="a" minOccurs="0" ` + str + `/></xs:sequence><xs:element name="z" type="xs:precisionDecimal"/></xs:sequence></xs:complexType></xs:element>`,
			"complex type (anonymous) of element e holding a sequence model group particle with {max occurs} above 1 whose sequence is emptiable at s.xsd:2:22 ", constructRepeatedEmptiableSequence},
		{"repeated empty sequence", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="h" ` + str + `/><xs:sequence maxOccurs="2"/></xs:sequence></xs:complexType></xs:element>`,
			"complex type (anonymous) of element e holding a sequence model group particle with {max occurs} above 1 whose sequence is emptiable at s.xsd:2:22 ", constructRepeatedEmptiableSequence},
		{"repeated emptiable sequence before its particles", `<xs:element name="e"><xs:complexType><xs:sequence maxOccurs="unbounded"><xs:element name="a" minOccurs="0" type="xs:precisionDecimal"/></xs:sequence></xs:complexType></xs:element>`,
			"complex type (anonymous) of element e holding a sequence model group particle with {max occurs} above 1 whose sequence is emptiable at s.xsd:2:22 ", constructRepeatedEmptiableSequence},
		{"repeated emptiable sequence group ref", `<xs:group name="g"><xs:sequence><xs:element name="b" minOccurs="0" ` + str + `/></xs:sequence></xs:group><xs:element name="e"><xs:complexType><xs:group ref="g" maxOccurs="unbounded"/></xs:complexType></xs:element>`,
			"complex type (anonymous) of element e holding a sequence model group particle with {max occurs} above 1 whose sequence is emptiable at s.xsd:2:", constructRepeatedEmptiableSequence},
		{"mixed choice", `<xs:element name="e"><xs:complexType mixed="true"><xs:choice><xs:element name="a" ` + str + `/></xs:choice></xs:complexType></xs:element>`,
			"with mixed content", constructMixedContent},
		{"open content choice", `<xs:element name="e"><xs:complexType><xs:openContent><xs:any/></xs:openContent><xs:choice><xs:element name="a" ` + str + `/></xs:choice></xs:complexType></xs:element>`,
			"with {open content}", constructOpenContent},
		{"choice alternative type", `<xs:element name="e"><xs:complexType><xs:choice><xs:element name="a" type="xs:precisionDecimal"/></xs:choice></xs:complexType></xs:element>`,
			"element a with {type definition} {http://www.w3.org/2001/XMLSchema}precisionDecimal, whose {primitive type definition} is", constructUncoveredPrimitive},
		// A <group ref> content particle is checked as its referenced group
		// would be inline: an emptiable group alternative declines whatever the
		// ref's occurs.
		{"emptiable group alternative group ref content", `<xs:group name="g"><xs:choice><xs:element name="a" ` + str + `/><xs:sequence><xs:element name="b" minOccurs="0" ` + str + `/></xs:sequence></xs:choice></xs:group><xs:element name="e"><xs:complexType><xs:group ref="g" maxOccurs="1"/></xs:complexType></xs:element>`,
			"complex type (anonymous) of element e whose choice's alternative 2, a sequence model group particle, has an emptiable model group", constructEmptiableGroupAlternative},
		// The group-cycle rows reach e again through a <group ref> to the
		// group declaring it (legal: mg-props-correct clause 2 bounds group
		// containment alone), as e's content particle and inside an inline
		// sequence, and as a named type's content.
		{"group cycle through content particle", `<xs:group name="g"><xs:sequence><xs:element name="e"><xs:complexType><xs:group ref="g"/></xs:complexType></xs:element></xs:sequence></xs:group><xs:element name="r"><xs:complexType><xs:group ref="g"/></xs:complexType></xs:element>`,
			"local element e reaching element e of model group g again through elements that each occur exactly once at s.xsd:2:33 ", constructValueCycle},
		{"group cycle through inline sequence", `<xs:group name="g"><xs:sequence><xs:element name="e"><xs:complexType><xs:sequence><xs:group ref="g"/></xs:sequence></xs:complexType></xs:element></xs:sequence></xs:group><xs:element name="r"><xs:complexType><xs:sequence><xs:group ref="g"/></xs:sequence></xs:complexType></xs:element>`,
			"local element e reaching element e of model group g again through elements that each occur exactly once at s.xsd:2:33 ", constructValueCycle},
		{"group cycle through named type", `<xs:complexType name="t"><xs:group ref="g"/></xs:complexType><xs:group name="g"><xs:sequence><xs:element name="e" type="t"/></xs:sequence></xs:group><xs:element name="r" type="t"/>`,
			"local element e reaching complex type t again through elements that each occur exactly once", constructValueCycle},
		// In the ref rows below, h is declared after e. Where a row places a
		// later decline after the ref, the ref's check is shown to come
		// first.
		{"element ref to nillable", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element ref="h"/></xs:sequence></xs:complexType></xs:element><xs:element name="z" type="xs:precisionDecimal"/><xs:element name="h" nillable="true" ` + str + `/>`,
			"element h with {nillable} true at s.xsd:2:165 ", constructElementNillable},
		{"element ref in choice to nillable", `<xs:element name="e"><xs:complexType><xs:choice><xs:element name="a" ` + str + `/><xs:element ref="h"/></xs:choice></xs:complexType></xs:element><xs:element name="h" nillable="true" ` + str + `/>`,
			"element h with {nillable} true", constructElementNillable},
		// h's type is checked where e's ref meets it, ahead of z's decline.
		{"element ref to complex type outside the slice", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element ref="h"/><xs:element name="z" type="xs:precisionDecimal"/></xs:sequence></xs:complexType></xs:element><xs:element name="h"><xs:complexType mixed="true"><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:complexType></xs:element>`,
			"complex type (anonymous) of element h with mixed content", constructMixedContent},
		{"element ref to simple type outside the slice", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element ref="h"/><xs:element name="z" type="xs:precisionDecimal"/></xs:sequence></xs:complexType></xs:element><xs:element name="h" type="xs:precisionDecimal"/>`,
			"element h with {type definition} {http://www.w3.org/2001/XMLSchema}precisionDecimal, whose {primitive type definition} is", constructUncoveredPrimitive},
		{"attribute list of uncovered primitive", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence><xs:attribute name="t"><xs:simpleType><xs:list itemType="xs:precisionDecimal"/></xs:simpleType></xs:attribute></xs:complexType></xs:element>`,
			"attribute t of complex type (anonymous) of element e with {type definition} (anonymous), whose {item type definition} is {http://www.w3.org/2001/XMLSchema}precisionDecimal, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}precisionDecimal", constructUncoveredPrimitive},
		{"attribute union", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence><xs:attribute name="t"><xs:simpleType><xs:union memberTypes="xs:int xs:precisionDecimal"/></xs:simpleType></xs:attribute></xs:complexType></xs:element>`,
			"attribute t of complex type (anonymous) of element e with {type definition} (anonymous), whose basic member is {http://www.w3.org/2001/XMLSchema}precisionDecimal, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}precisionDecimal", constructUncoveredPrimitive},
		{"attribute uncovered primitive", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence><xs:attribute name="t" type="xs:precisionDecimal"/></xs:complexType></xs:element>`,
			"attribute t of complex type (anonymous) of element e with {type definition} {http://www.w3.org/2001/XMLSchema}precisionDecimal, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}precisionDecimal", constructUncoveredPrimitive},
		{"empty type abstract", `<xs:complexType name="c" abstract="true"/><xs:element name="e" type="c"/>`,
			"complex type c of element e with {abstract} true", constructComplexTypeAbstract},
		{"mixed empty", `<xs:element name="e"><xs:complexType mixed="true"/></xs:element>`,
			"with mixed content", constructMixedContent},
		{"default open content applies to empty", `<xs:defaultOpenContent appliesToEmpty="true"><xs:any/></xs:defaultOpenContent><xs:element name="e"><xs:complexType/></xs:element>`,
			"with {open content}", constructOpenContent},
		// Mixed content, checked first, still declines a type that has an
		// {attribute wildcard}.
		{"attribute wildcard with mixed content", `<xs:element name="e"><xs:complexType mixed="true"><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence><xs:anyAttribute/></xs:complexType></xs:element>`,
			"complex type (anonymous) of element e with mixed content", constructMixedContent},
		{"mixed", `<xs:element name="e"><xs:complexType mixed="true"><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:complexType></xs:element>`,
			"with mixed content", constructMixedContent},
		// The simple-content rows pin the value field's declines: its
		// {simple type definition} (§3.4.2.2 dcl.ctd.ctsc) declines by that
		// type's construct.
		{"simple content extension of union", `<xs:simpleType name="u"><xs:union memberTypes="xs:int xs:precisionDecimal"/></xs:simpleType><xs:element name="e"><xs:complexType><xs:simpleContent><xs:extension base="u"/></xs:simpleContent></xs:complexType></xs:element>`,
			"complex type (anonymous) of element e with {simple type definition} u, whose basic member is {http://www.w3.org/2001/XMLSchema}precisionDecimal, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}precisionDecimal", constructUncoveredPrimitive},
		{"simple content extension of precisionDecimal", `<xs:element name="e"><xs:complexType><xs:simpleContent><xs:extension base="xs:precisionDecimal"/></xs:simpleContent></xs:complexType></xs:element>`,
			"complex type (anonymous) of element e with {simple type definition} {http://www.w3.org/2001/XMLSchema}precisionDecimal, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}precisionDecimal at s.xsd:2:22 ", constructUncoveredPrimitive},
		{"open content", `<xs:element name="e"><xs:complexType><xs:openContent><xs:any/></xs:openContent><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:complexType></xs:element>`,
			"with {open content}", constructOpenContent},
		{"default open content", `<xs:defaultOpenContent><xs:any/></xs:defaultOpenContent><xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:complexType></xs:element>`,
			"with {open content}", constructOpenContent},
		// The two rows below carry a {value constraint} beside the declining
		// property, so they pin that the value constraint, accepted, is not
		// checked ahead of it.
		{"nillable", `<xs:element name="e" nillable="true" default="d" ` + str + `/>`,
			"element e with {nillable} true", constructElementNillable},
		{"abstract element", `<xs:element name="e" abstract="true" fixed="f" ` + str + `/>`,
			"element e with {abstract} true", constructElementAbstract},
		// An abstract head declines at its own turn, declared before its
		// member, and the decline is charged to the head: membership in an
		// abstract head's substitution group declines nothing (§3.3.6.4
		// key-eq; cvc-elt clause 2 reads the declaration's own {abstract}).
		{"abstract substitution group head", `<xs:element name="h" abstract="true" ` + str + `/><xs:element name="m" substitutionGroup="h" ` + str + `/>`,
			"element h with {abstract} true at s.xsd:2:1 ", constructElementAbstract},
		// The two rows below pin that a type with {assertions} still declines
		// for its other properties: mixed content, tested after {abstract},
		// and {abstract} true.
		{"mixed with assertion", `<xs:element name="e"><xs:complexType mixed="true"><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence><xs:assert test="true()"/></xs:complexType></xs:element>`,
			"complex type (anonymous) of element e with mixed content", constructMixedContent},
		{"abstract with assertion", `<xs:complexType name="c" abstract="true"><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence><xs:assert test="true()"/></xs:complexType><xs:element name="e" type="c"/>`,
			"complex type c of element e with {abstract} true", constructComplexTypeAbstract},
		{"abstract complex type", `<xs:complexType name="c" abstract="true"><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:complexType><xs:element name="e" type="c"/>`,
			"complex type c of element e with {abstract} true", constructComplexTypeAbstract},
		// The three guard rows pin that the xs:anyType arm keys on identity: c
		// copies xs:anyType's §3.4.7 shape, or derives from it by extension,
		// and still declines as the complex type it is.
		{"copy of anyType's shape", `<xs:complexType name="c" mixed="true"><xs:sequence><xs:any processContents="lax" minOccurs="0" maxOccurs="unbounded"/></xs:sequence><xs:anyAttribute processContents="lax"/></xs:complexType><xs:element name="e" type="c"/>`,
			"complex type c of element e with mixed content", constructMixedContent},
		{"copy of anyType's shape without anyAttribute", `<xs:complexType name="c" mixed="true"><xs:sequence><xs:any processContents="lax" minOccurs="0" maxOccurs="unbounded"/></xs:sequence></xs:complexType><xs:element name="e" type="c"/>`,
			"complex type c of element e with mixed content", constructMixedContent},
		{"extension of anyType", `<xs:complexType name="c"><xs:complexContent><xs:extension base="xs:anyType"/></xs:complexContent></xs:complexType><xs:element name="e" type="c"/>`,
			"complex type c of element e with mixed content", constructMixedContent},
		// The list rows pin "# Coverage"'s list decline, and that its message
		// names the list and then its {item type definition}.
		{"list of named union", `<xs:simpleType name="u"><xs:union memberTypes="xs:int xs:precisionDecimal"/></xs:simpleType><xs:simpleType name="l"><xs:list itemType="u"/></xs:simpleType><xs:element name="e" type="l"/>`,
			"element e with {type definition} l, whose {item type definition} is u, whose basic member is {http://www.w3.org/2001/XMLSchema}precisionDecimal, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}precisionDecimal", constructUncoveredPrimitive},
		{"list of anonymous union", `<xs:element name="e"><xs:simpleType><xs:list><xs:simpleType><xs:union memberTypes="xs:int xs:precisionDecimal"/></xs:simpleType></xs:list></xs:simpleType></xs:element>`,
			"element e with {type definition} (anonymous), whose {item type definition} is (anonymous), whose basic member is {http://www.w3.org/2001/XMLSchema}precisionDecimal, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}precisionDecimal", constructUncoveredPrimitive},
		{"restriction of a list of union", `<xs:simpleType name="u"><xs:union memberTypes="xs:int xs:precisionDecimal"/></xs:simpleType><xs:simpleType name="l"><xs:list itemType="u"/></xs:simpleType><xs:simpleType name="r"><xs:restriction base="l"><xs:maxLength value="2"/></xs:restriction></xs:simpleType><xs:element name="e" type="r"/>`,
			"element e with {type definition} r, whose {item type definition} is u, whose basic member is {http://www.w3.org/2001/XMLSchema}precisionDecimal, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}precisionDecimal", constructUncoveredPrimitive},
		{"list of uncovered primitive", `<xs:simpleType name="l"><xs:list itemType="xs:precisionDecimal"/></xs:simpleType><xs:element name="e" type="l"/>`,
			"element e with {type definition} l, whose {item type definition} is {http://www.w3.org/2001/XMLSchema}precisionDecimal, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}precisionDecimal", constructUncoveredPrimitive},
		// The union rows pin that a union declines exactly when a basic member
		// does, by the member's construct, naming that member after the union:
		// one listed second, and one reached through a nested union.
		{"union", `<xs:element name="e"><xs:simpleType><xs:union memberTypes="xs:int xs:precisionDecimal"/></xs:simpleType></xs:element>`,
			"element e with {type definition} (anonymous), whose basic member is {http://www.w3.org/2001/XMLSchema}precisionDecimal, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}precisionDecimal", constructUncoveredPrimitive},
		{"union of a union", `<xs:simpleType name="inner"><xs:union memberTypes="xs:int xs:precisionDecimal"/></xs:simpleType><xs:element name="e"><xs:simpleType><xs:union memberTypes="xs:boolean inner"/></xs:simpleType></xs:element>`,
			"element e with {type definition} (anonymous), whose basic member is {http://www.w3.org/2001/XMLSchema}precisionDecimal, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}precisionDecimal", constructUncoveredPrimitive},
		{"precisionDecimal", `<xs:element name="e" type="xs:precisionDecimal"/>`,
			"element e with {type definition} {http://www.w3.org/2001/XMLSchema}precisionDecimal, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}precisionDecimal", constructUncoveredPrimitive},
		// The derived rows reach an uncovered primitive along {base type
		// definition} chains of one to three steps.
		{"restriction of precisionDecimal", `<xs:simpleType name="d"><xs:restriction base="xs:precisionDecimal"/></xs:simpleType><xs:element name="e" type="d"/>`,
			"element e with {type definition} d, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}precisionDecimal", constructUncoveredPrimitive},
		{"anonymous restriction of precisionDecimal", `<xs:element name="e"><xs:simpleType><xs:restriction base="xs:precisionDecimal"><xs:totalDigits value="2"/></xs:restriction></xs:simpleType></xs:element>`,
			"element e with {type definition} (anonymous), whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}precisionDecimal", constructUncoveredPrimitive},
		{"restriction of a restriction of precisionDecimal", `<xs:simpleType name="d"><xs:restriction base="xs:precisionDecimal"/></xs:simpleType><xs:simpleType name="d2"><xs:restriction base="d"/></xs:simpleType><xs:element name="e" type="d2"/>`,
			"element e with {type definition} d2, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}precisionDecimal", constructUncoveredPrimitive},
		{"restriction of a restriction of a restriction of precisionDecimal", `<xs:simpleType name="d"><xs:restriction base="xs:precisionDecimal"/></xs:simpleType><xs:simpleType name="d2"><xs:restriction base="d"/></xs:simpleType><xs:simpleType name="d3"><xs:restriction base="d2"/></xs:simpleType><xs:element name="e" type="d3"/>`,
			"element e with {type definition} d3, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}precisionDecimal", constructUncoveredPrimitive},
		// The value-cycle rows each give Go types that would hold one another
		// as struct values.
		{"value cycle direct", `<xs:complexType name="c"><xs:sequence><xs:element name="x" type="c"/></xs:sequence></xs:complexType><xs:element name="e" type="c"/>`,
			"local element x reaching complex type c again through elements that each occur exactly once at s.xsd:2:39", constructValueCycle},
		{"value cycle mutual", `<xs:complexType name="c1"><xs:sequence><xs:element name="x" type="c2"/></xs:sequence></xs:complexType><xs:complexType name="c2"><xs:sequence><xs:element name="y" type="c1"/></xs:sequence></xs:complexType><xs:element name="e" type="c1"/>`,
			"local element y reaching complex type c1 again through elements that each occur exactly once", constructValueCycle},
		{"value cycle through anonymous type", `<xs:complexType name="c"><xs:sequence><xs:element name="x"><xs:complexType><xs:sequence><xs:element name="y" type="c"/></xs:sequence></xs:complexType></xs:element></xs:sequence></xs:complexType><xs:element name="e" type="c"/>`,
			"local element y reaching complex type c again through elements that each occur exactly once", constructValueCycle},
		{"value cycle in choice branch", `<xs:complexType name="t1"><xs:sequence><xs:element name="b" type="t2"/></xs:sequence></xs:complexType><xs:complexType name="t2"><xs:sequence><xs:element name="c" type="t1"/></xs:sequence></xs:complexType><xs:element name="e"><xs:complexType><xs:choice><xs:element name="a" type="t1"/></xs:choice></xs:complexType></xs:element>`,
			"local element c reaching complex type t1 again through elements that each occur exactly once", constructValueCycle},
		// An all group's member occurring exactly once is a value edge of the
		// emitting walk as a sequence's is, inline and spliced from an all
		// group nested through a <group ref> (cos-all-limited clause 1.3).
		{"value cycle through all", `<xs:complexType name="c"><xs:all><xs:element name="x" type="c"/></xs:all></xs:complexType><xs:element name="e" type="c"/>`,
			"local element x reaching complex type c again through elements that each occur exactly once at s.xsd:2:34", constructValueCycle},
		{"value cycle through nested all", `<xs:group name="g"><xs:all><xs:element name="x" type="c"/></xs:all></xs:group><xs:complexType name="c"><xs:all><xs:element name="y" ` + str + `/><xs:group ref="g"/></xs:all></xs:complexType><xs:element name="e" type="c"/>`,
			"local element x reaching complex type c again through elements that each occur exactly once at s.xsd:2:28", constructValueCycle},
		// The ref-cycle rows close a cycle through <element ref> particles,
		// which the emitting walk treats as leaves, so only the recursion check
		// of each ref finds it; each decline cites the Loc of the top-level
		// declaration the closing ref names.
		{"ref cycle through a named type", `<xs:element name="r" type="c"/><xs:complexType name="c"><xs:sequence><xs:element ref="r"/></xs:sequence></xs:complexType>`,
			"<element ref> particle to element r reaching element r again through elements that each occur exactly once at s.xsd:2:1 ", constructValueCycle},
		// The same holds for the recursion check (valueCycle), which alone
		// finds a cycle closed by an <element ref> particle: these rows fail
		// when it ends its walk at an all group.
		{"ref cycle through all", `<xs:element name="r" type="c"/><xs:complexType name="c"><xs:all><xs:element ref="r"/></xs:all></xs:complexType>`,
			"<element ref> particle to element r reaching element r again through elements that each occur exactly once at s.xsd:2:1 ", constructValueCycle},
		{"ref cycle through nested all", `<xs:element name="r" type="c"/><xs:group name="g"><xs:all><xs:element ref="r"/></xs:all></xs:group><xs:complexType name="c"><xs:all><xs:element name="y" ` + str + `/><xs:group ref="g"/></xs:all></xs:complexType>`,
			"<element ref> particle to element r reaching element r again through elements that each occur exactly once at s.xsd:2:1 ", constructValueCycle},
		{"ref cycle through anonymous types", `<xs:element name="a"><xs:complexType><xs:sequence><xs:element ref="b"/></xs:sequence></xs:complexType></xs:element><xs:element name="b"><xs:complexType><xs:sequence><xs:element ref="a"/></xs:sequence></xs:complexType></xs:element>`,
			"<element ref> particle to element a reaching element a again through elements that each occur exactly once at s.xsd:2:1 ", constructValueCycle},
		// g's own element node is new to the chain f starts, and the type t
		// it shares with f closes the cycle.
		{"ref cycle closing at a complex type", `<xs:element name="f" type="t"/><xs:complexType name="t"><xs:sequence><xs:element ref="g"/></xs:sequence></xs:complexType><xs:element name="g" type="t"/>`,
			"<element ref> particle to element g reaching complex type t again through elements that each occur exactly once", constructValueCycle},
		// Each ref sits in a nested sequence occurring exactly once, so the
		// recursion check finds the cycle only by walking a's and b's
		// particles as particleFields does (spliced).
		{"ref cycle through nested sequences", `<xs:element name="a"><xs:complexType><xs:sequence><xs:sequence><xs:element ref="b"/></xs:sequence></xs:sequence></xs:complexType></xs:element><xs:element name="b"><xs:complexType><xs:sequence><xs:sequence><xs:element ref="a"/></xs:sequence></xs:sequence></xs:complexType></xs:element>`,
			"<element ref> particle to element a reaching element a again through elements that each occur exactly once at s.xsd:2:1 ", constructValueCycle},
		// In the two extension rows each ref is spliced into a's or b's
		// fields from the base particle, or from the extension's own, of a
		// clause 4.2.3.3 sequence, so the recursion check finds the cycle
		// only by walking the particles spliced returns.
		{"ref cycle through base particles", `<xs:complexType name="ba"><xs:sequence><xs:element ref="b"/></xs:sequence></xs:complexType><xs:complexType name="bb"><xs:sequence><xs:element ref="a"/></xs:sequence></xs:complexType><xs:element name="a"><xs:complexType><xs:complexContent><xs:extension base="ba"><xs:sequence><xs:element name="s" ` + str + `/></xs:sequence></xs:extension></xs:complexContent></xs:complexType></xs:element><xs:element name="b"><xs:complexType><xs:complexContent><xs:extension base="bb"><xs:sequence><xs:element name="s" ` + str + `/></xs:sequence></xs:extension></xs:complexContent></xs:complexType></xs:element>`,
			"<element ref> particle to element a reaching element a again through elements that each occur exactly once", constructValueCycle},
		{"ref cycle through extension particles", `<xs:complexType name="be"><xs:sequence><xs:element name="s" ` + str + `/></xs:sequence></xs:complexType><xs:element name="a"><xs:complexType><xs:complexContent><xs:extension base="be"><xs:sequence><xs:element ref="b"/></xs:sequence></xs:extension></xs:complexContent></xs:complexType></xs:element><xs:element name="b"><xs:complexType><xs:complexContent><xs:extension base="be"><xs:sequence><xs:element ref="a"/></xs:sequence></xs:extension></xs:complexContent></xs:complexType></xs:element>`,
			"<element ref> particle to element a reaching element a again through elements that each occur exactly once", constructValueCycle},
		{"ref cycle through a local element", `<xs:element name="a"><xs:complexType><xs:sequence><xs:element name="l"><xs:complexType><xs:sequence><xs:element ref="a"/></xs:sequence></xs:complexType></xs:element></xs:sequence></xs:complexType></xs:element>`,
			"<element ref> particle to element a reaching element a again through elements that each occur exactly once", constructValueCycle},
		// The ref to a is checked for recursion where it is met, ahead of the
		// decline of z after it.
		{"ref cycle ahead of a later decline", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element ref="a"/><xs:element name="z" type="xs:precisionDecimal"/></xs:sequence></xs:complexType></xs:element><xs:element name="a"><xs:complexType><xs:sequence><xs:element ref="b"/></xs:sequence></xs:complexType></xs:element><xs:element name="b"><xs:complexType><xs:sequence><xs:element ref="a"/></xs:sequence></xs:complexType></xs:element>`,
			"<element ref> particle to element a reaching element a again through elements that each occur exactly once", constructValueCycle},
		// x's walk emits z through the pointer field p before it reaches the
		// value field v, and z's walk meets x already stored: a check carried
		// by the emitting walk, which never walks a stored type again, lets
		// the cycle v, w through.
		{"value cycle behind a pointer", `<xs:complexType name="x"><xs:sequence><xs:element name="p" type="z" minOccurs="0"/><xs:element name="v" type="z"/></xs:sequence></xs:complexType><xs:complexType name="z"><xs:sequence><xs:element name="w" type="x"/></xs:sequence></xs:complexType><xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" type="x"/></xs:sequence></xs:complexType></xs:element>`,
			"local element v reaching complex type z again through elements that each occur exactly once", constructValueCycle},
		// The emptiable group alternative under p precedes the cycle v closes
		// in particle order, so it is the decline returned.
		{"value cycle after an earlier decline", `<xs:complexType name="c"><xs:sequence><xs:element name="p" minOccurs="0"><xs:complexType><xs:choice><xs:sequence><xs:any minOccurs="0"/></xs:sequence></xs:choice></xs:complexType></xs:element><xs:element name="v" type="c"/></xs:sequence></xs:complexType><xs:element name="e"><xs:complexType><xs:sequence><xs:element name="l" type="c"/></xs:sequence></xs:complexType></xs:element>`,
			"complex type (anonymous) of element p of complex type c whose choice's alternative 1, a sequence model group particle, has an emptiable model group at ", constructEmptiableGroupAlternative},
		// In the three rows below, q reaches t while t's body is being
		// emitted, so the recursion check walks t again past f1: it passes
		// the wildcard as a leaf, and meets an element declaration property
		// and a complex type outside the slice as declines it leaves to the
		// emitting walk, which returns r's first.
		{"value cycle check past a wildcard", fmt.Sprintf(pastDecline, `<xs:any/>`, ""),
			uncoveredR, constructUncoveredPrimitive},
		{"value cycle check past an element property", fmt.Sprintf(pastDecline, `<xs:element name="g" nillable="true" `+str+`/>`, ""),
			uncoveredR, constructUncoveredPrimitive},
		{"value cycle check past a complex type", fmt.Sprintf(pastDecline, `<xs:element name="g" type="a"/>`, `<xs:complexType name="a" mixed="true"><xs:sequence><xs:element name="h" `+str+`/></xs:sequence></xs:complexType>`),
			uncoveredR, constructUncoveredPrimitive},
		// The same walk of t again meets the recursion s closes after f1, and
		// Generate's doc puts that recursion ahead of r's decline.
		{"value cycle check past a recursion", fmt.Sprintf(pastDecline, `<xs:element name="g" type="cyc"/>`, `<xs:complexType name="cyc"><xs:sequence><xs:element name="s" type="cyc"/></xs:sequence></xs:complexType>`),
			"local element s reaching complex type cyc again through elements that each occur exactly once", constructValueCycle},
		// e's walk reaches t, declared after f, before f's walk begins.
		{"named type body before a later top-level element", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="l" type="t"/></xs:sequence></xs:complexType></xs:element><xs:element name="f" type="xs:precisionDecimal"/><xs:complexType name="t"><xs:choice><xs:sequence><xs:any minOccurs="0"/></xs:sequence></xs:choice></xs:complexType>`,
			"complex type t whose choice's alternative 1, a sequence model group particle, has an emptiable model group at s.xsd:2:175 ", constructEmptiableGroupAlternative},
		{"named complex local body", `<xs:complexType name="c"><xs:choice><xs:sequence><xs:any minOccurs="0"/></xs:sequence></xs:choice></xs:complexType><xs:element name="e"><xs:complexType><xs:sequence><xs:element name="l" type="c"/></xs:sequence></xs:complexType></xs:element>`,
			"complex type c whose choice's alternative 1, a sequence model group particle, has an emptiable model group at s.xsd:2:1 ", constructEmptiableGroupAlternative},
		{"local element property", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="l" nillable="true" ` + str + `/></xs:sequence></xs:complexType></xs:element>`,
			"element l with {nillable} true", constructElementNillable},
		{"nested struct", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="l"><xs:complexType mixed="true"><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:complexType></xs:element></xs:sequence></xs:complexType></xs:element>`,
			"complex type (anonymous) of element l of element e with mixed content at ", constructMixedContent},
		{"nested choice", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="l"><xs:complexType><xs:choice><xs:element name="a" ` + str + `/><xs:sequence><xs:any minOccurs="0"/></xs:sequence></xs:choice></xs:complexType></xs:element></xs:sequence></xs:complexType></xs:element>`,
			"complex type (anonymous) of element l of element e whose choice's alternative 2, a sequence model group particle, has an emptiable model group at ", constructEmptiableGroupAlternative},
		{"branch struct", `<xs:element name="e"><xs:complexType><xs:choice><xs:element name="l"><xs:complexType mixed="true"><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:complexType></xs:element></xs:choice></xs:complexType></xs:element>`,
			"complex type (anonymous) of element l of element e with mixed content at ", constructMixedContent},
		// The {type table} rows pin that a decline in a type the table selects
		// names that type's slot: an anonymous complex type by its slot and
		// the place of the declaration, a simple type by its slot, charged to
		// the declaration.
		{"type table anonymous alternative", `<xs:element name="e"><xs:alternative test="@k"><xs:complexType mixed="true"><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:complexType></xs:alternative><xs:alternative type="xs:string"/></xs:element>`,
			"complex type (anonymous) of {type table} alternative 1 of element e with mixed content at ", constructMixedContent},
		{"type table anonymous default", `<xs:element name="e"><xs:alternative test="@k" type="xs:string"/><xs:alternative><xs:complexType><xs:sequence><xs:element name="l"><xs:complexType mixed="true"><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:complexType></xs:element></xs:sequence></xs:complexType></xs:alternative></xs:element>`,
			"complex type (anonymous) of element l of {default type definition} of element e with mixed content at ", constructMixedContent},
		{"type table simple alternative", `<xs:element name="e"><xs:alternative test="@k" type="xs:precisionDecimal"/><xs:alternative type="xs:string"/></xs:element>`,
			"{type table} alternative 1 of element e with {type definition} {http://www.w3.org/2001/XMLSchema}precisionDecimal, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}precisionDecimal at s.xsd:2:1 ", constructUncoveredPrimitive},
		{"type table local", `<xs:complexType name="c"><xs:sequence><xs:element name="x"><xs:alternative test="@k"><xs:complexType mixed="true"><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:complexType></xs:alternative><xs:alternative type="xs:string"/></xs:element></xs:sequence></xs:complexType><xs:element name="e" type="c"/>`,
			"complex type (anonymous) of {type table} alternative 1 of element x of complex type c with mixed content at ", constructMixedContent},
		// h's complex alternative is checked where e's ref meets it, ahead of
		// z's decline.
		{"element ref to type table outside the slice", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element ref="h"/><xs:element name="z" type="xs:precisionDecimal"/></xs:sequence></xs:complexType></xs:element><xs:element name="h"><xs:alternative test="@k"><xs:complexType mixed="true"><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:complexType></xs:alternative><xs:alternative type="xs:string"/></xs:element>`,
			"complex type (anonymous) of {type table} alternative 1 of element h with mixed content at ", constructMixedContent},
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

// TestGenerateDeclinesLocalAnonymousPlace pins that a decline met in a local
// element's anonymous complex type names the type's declaring place, so that
// two sites charged at one Loc, here the global attribute g every site
// references, print distinct texts and Generate reports each, in document
// order, while one site reached by two walks stays one element.
func TestGenerateDeclinesLocalAnonymousPlace(t *testing.T) {
	const (
		g    = `<xs:attribute name="g" type="xs:precisionDecimal"/>`
		anon = `<xs:complexType><xs:attribute ref="g"/></xs:complexType>`
		x    = `<xs:element name="x">` + anon + `</xs:element>`
		a    = `<xs:element name="a"><xs:complexType><xs:sequence>` + x + `</xs:sequence></xs:complexType></xs:element>`
	)
	top := func(name, content string) string {
		return `<xs:element name="` + name + `"><xs:complexType>` + content + `</xs:complexType></xs:element>`
	}
	seq := func(particle string) string { return `<xs:sequence>` + particle + `</xs:sequence>` }
	decline := func(subject string) string {
		return "codegen: target 0: attribute g of " + subject + " with {type definition} {http://www.w3.org/2001/XMLSchema}precisionDecimal, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}precisionDecimal at s.xsd:2:1 is not supported by codegen, a generator limitation, not a schema fault"
	}
	tests := []struct {
		name, body string
		want       []string
	}{
		{"two top-level elements", g + top("p", seq(x)) + top("q", seq(x)), []string{
			decline("complex type (anonymous) of element x of element p"),
			decline("complex type (anonymous) of element x of element q"),
		}},
		{"one model group definition", g + `<xs:group name="m">` + seq(x) + `</xs:group>` + top("p", `<xs:group ref="m"/>`) + top("q", `<xs:group ref="m"/>`), []string{
			decline("complex type (anonymous) of element x of model group m"),
		}},
		{"named complex type and top-level element", g + `<xs:complexType name="c">` + seq(x) + `</xs:complexType><xs:element name="r" type="c"/>` + top("p", seq(x)), []string{
			decline("complex type (anonymous) of element x of complex type c"),
			decline("complex type (anonymous) of element x of element p"),
		}},
		{"top-level and local element", g + `<xs:element name="x">` + anon + `</xs:element>` + top("p", seq(x)), []string{
			decline("complex type (anonymous) of element x"),
			decline("complex type (anonymous) of element x of element p"),
		}},
		{"two levels down", g + top("p", seq(a)) + top("q", seq(a)), []string{
			decline("complex type (anonymous) of element x of element a of element p"),
			decline("complex type (anonymous) of element x of element a of element q"),
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := parseString(t, "<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\">\n"+tt.body+"\n</xs:schema>\n")
			errs := elements(t, Generate([]Target{{Schema: schema, Dir: filepath.Join(t.TempDir(), "out"), Package: "p"}}))
			got := make([]string, len(errs))
			for i, err := range errs {
				got[i] = err.Error()
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("Generate's elements =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}

// TestGenerateDeclinesTypeTableSimpleAtOwnTurn pins Generate's doc on a ref
// to a declaration given a {type table}'s interface: the simple types the
// table selects are checked at the declaration's own turn, not where the ref
// is met, so e's turn returns z's decline, after the ref to h, and h's turn
// returns its alternative's.
func TestGenerateDeclinesTypeTableSimpleAtOwnTurn(t *testing.T) {
	const body = `<xs:element name="e"><xs:complexType><xs:sequence><xs:element ref="h"/><xs:element name="z" type="xs:precisionDecimal"/></xs:sequence></xs:complexType></xs:element>` +
		`<xs:element name="h"><xs:alternative test="@k" type="xs:precisionDecimal"/><xs:alternative type="xs:string"/></xs:element>`
	const tail = ", whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}precisionDecimal at s.xsd:2:"
	schema := parseString(t, "<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\">\n"+body+"\n</xs:schema>\n")
	errs := elements(t, Generate([]Target{{Schema: schema, Dir: filepath.Join(t.TempDir(), "out"), Package: "p"}}))
	want := []string{
		"codegen: target 0: element z with {type definition} {http://www.w3.org/2001/XMLSchema}precisionDecimal" + tail,
		"codegen: target 0: {type table} alternative 1 of element h with {type definition} {http://www.w3.org/2001/XMLSchema}precisionDecimal" + tail,
	}
	if len(errs) != len(want) {
		t.Fatalf("Generate returned %d elements %q, want %d", len(errs), errs, len(want))
	}
	for i, err := range errs {
		if !strings.HasPrefix(err.Error(), want[i]) {
			t.Errorf("element %d = %q, want the prefix %q", i, err, want[i])
		}
	}
}

// TestGenerateDeclinesVacuousParticle pins the decline of a particle whose
// {max occurs} is 0, naming its term: a local element, a wildcard, the
// sequence model group of a content particle, or a choice model group nested
// in it. No schema document yields one, since an <element> or <any> with
// minOccurs=maxOccurs=0 maps to no component at all (§3.3.2, §3.10.2) and a
// <sequence> or <choice> with maxOccurs=0 makes the content empty (§3.4.2.3.3
// dcl.ctd.ctcc.common 2.1.4), but xsd.NewParticle and xsd.NewComplexType
// accept a vacuous Occurs, so the schema is built through the constructors.
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
	anyNamespace, err := xsd.NewNamespaceConstraint(xsderr.Loc{}, xsd.NamespaceConstraintAny, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	wildcard, err := xsd.NewWildcard(xsderr.Loc{}, anyNamespace, xsd.ProcessLax)
	if err != nil {
		t.Fatal(err)
	}
	vacuous, err := xsd.NewOccurs(xsderr.Loc{}, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	localParticle, err := xsd.NewParticle(xsderr.Loc{}, exactlyOnce, xsd.ResolvedTerm{Term: local})
	if err != nil {
		t.Fatal(err)
	}
	nested, err := xsd.NewModelGroup(xsderr.Loc{}, xsd.CompositorChoice, []xsd.Particle{localParticle})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name         string
		term         xsd.Term
		inner, outer xsd.Occurs
		want         string
	}{
		{"element", local, vacuous, exactlyOnce, "codegen: target 0: a particle of element a with {max occurs} 0 in complex type c of element e at "},
		{"wildcard", wildcard, vacuous, exactlyOnce, "codegen: target 0: a particle of a wildcard with {max occurs} 0 in complex type c of element e at "},
		{"content particle", local, exactlyOnce, vacuous, "codegen: target 0: a particle of a sequence model group with {max occurs} 0 in complex type c of element e at "},
		{"nested group", nested, vacuous, exactlyOnce, "codegen: target 0: a particle of a choice model group with {max occurs} 0 in complex type c of element e at "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := vacuousSchema(t, ctName, tt.term, tt.inner, tt.outer)
			out := filepath.Join(t.TempDir(), "out")
			err := Generate([]Target{{Schema: schema, Dir: out, Package: "p"}})
			if !errors.Is(err, errors.ErrUnsupported) {
				t.Fatalf("Generate = %v, want a decline wrapping errors.ErrUnsupported", err)
			}
			if !strings.HasPrefix(err.Error(), tt.want) {
				t.Errorf("error %q does not start with %q", err, tt.want)
			}
			var de *DeclineError
			if !errors.As(err, &de) || de.Construct() != (Construct{constructVacuousParticle}) {
				t.Fatalf("Generate = %v, want a *DeclineError of Construct %v", err, Construct{constructVacuousParticle})
			}
			// Every component is built at the zero Loc, so DeclineError.Loc's
			// doc makes the decline's Loc zero.
			if loc := de.Loc(); loc != (xsderr.Loc{}) {
				t.Errorf("Loc() = %+v, want the zero xsderr.Loc", loc)
			}
			if _, err := os.Stat(out); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("Dir %s exists after a decline (stat: %v)", out, err)
			}
		})
	}
}

// vacuousSchema builds the schema TestGenerateDeclinesVacuousParticle
// generates from: a top-level element e of the complex type ctName, whose
// content particle, occurring outerOccurs times, is a sequence of one particle
// of term occurring innerOccurs times.
func vacuousSchema(t *testing.T, ctName xsd.QName, term xsd.Term, innerOccurs, outerOccurs xsd.Occurs) *xsd.Schema {
	t.Helper()
	inner, err := xsd.NewParticle(xsderr.Loc{}, innerOccurs, xsd.ResolvedTerm{Term: term})
	if err != nil {
		t.Fatal(err)
	}
	group, err := xsd.NewModelGroup(xsderr.Loc{}, xsd.CompositorSequence, []xsd.Particle{inner})
	if err != nil {
		t.Fatal(err)
	}
	outer, err := xsd.NewParticle(xsderr.Loc{}, outerOccurs, xsd.ResolvedTerm{Term: group})
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
	return schema
}

// TestGenerateEmptyUnion pins that a union with no {member type definitions},
// which §3.16.1 admits and xsd.NewSimpleType accepts though no schema document
// yields one (src-union-memberTypes-or-simpleTypes), generates its interface
// alone, which no variant implements, and so does v, whose one member is u.
func TestGenerateEmptyUnion(t *testing.T) {
	u, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Local: "u"}, xsd.UnionDerivation{},
		xsd.OwnedSimpleType{Definition: xsd.AnySimpleType()}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	v, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Local: "v"},
		xsd.UnionDerivation{Members: []xsd.SimpleTypeOrRef{xsd.OwnedSimpleType{Definition: u}}},
		xsd.OwnedSimpleType{Definition: xsd.AnySimpleType()}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "e"},
		xsd.TypeDefinitionRef{Name: u.Name()}, nil, xsd.NewGlobalScope(), nil, false, nil, nil, nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	f, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "f"},
		xsd.TypeDefinitionRef{Name: v.Name()}, nil, xsd.NewGlobalScope(), nil, false, nil, nil, nil, false, nil)
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
	b.AddType(u)
	b.AddType(v)
	b.AddElement(e)
	b.AddElement(f)
	schema, err := b.Finalize()
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	src := generateOne(t, Target{Schema: schema, Dir: filepath.Join(t.TempDir(), "out"), Package: "p"})
	typeCheck(t, src)
	for _, want := range []string{"type U interface {\n\tisU()\n}\n", "type V interface {\n\tisV()\n}\n"} {
		if !strings.Contains(string(src), want) {
			t.Errorf("source does not declare %q:\n%s", want, src)
		}
	}
	if strings.Contains(string(src), "func (") {
		t.Errorf("source declares a method, want no variant implementing isU or isV:\n%s", src)
	}
}

// TestDeclineErrorAs pins that errors.As retrieves the *DeclineError from the
// error Generate returns for a named and an unnamed target, with its Construct
// and Loc as values, and that the target's name opens the error outside the
// *DeclineError's own text.
func TestDeclineErrorAs(t *testing.T) {
	schema := parseString(t, "<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\">\n<xs:element name=\"e\" type=\"xs:precisionDecimal\"/>\n</xs:schema>\n")
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
			const detail = "element e with {type definition} {http://www.w3.org/2001/XMLSchema}precisionDecimal, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}precisionDecimal at "
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
	if got := len(names); got != int(constructValueCycle) {
		t.Errorf("walked %d constructs, want %d", got, constructValueCycle)
	}
	for _, c := range []construct{0, construct(len(constructs))} {
		if got, want := (Construct{c}).String(), fmt.Sprintf("Construct(%d)", c); got != want {
			t.Errorf("Construct{%d}.String() = %q, want %q", c, got, want)
		}
	}
}

// TestCoverageDeclinedList pins the package doc's "Declined today" list to
// constructs (STYLE D3): its items are the names Construct.String returns for
// constructs[1:], one item each, in index order.
func TestCoverageDeclinedList(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "doc.go", nil, parser.ParseComments|parser.PackageClauseOnly)
	if err != nil {
		t.Fatalf("parse doc.go: %v", err)
	}
	if file.Doc == nil {
		t.Fatal("doc.go has no package comment")
	}
	blocks := new(comment.Parser).Parse(file.Doc.Text()).Content
	label := slices.IndexFunc(blocks, func(b comment.Block) bool {
		p, ok := b.(*comment.Paragraph)
		return ok && strings.HasPrefix(plainText(p.Text), "Declined today")
	})
	if label < 0 || label+1 == len(blocks) {
		t.Fatal(`package doc has no "Declined today" paragraph followed by a block`)
	}
	list, ok := blocks[label+1].(*comment.List)
	if !ok {
		t.Fatalf(`the block after "Declined today" is a %T, want a *comment.List`, blocks[label+1])
	}
	var got []string
	for i, item := range list.Items {
		if len(item.Content) != 1 {
			t.Fatalf("Declined item %d holds %d blocks, want 1", i, len(item.Content))
		}
		p, ok := item.Content[0].(*comment.Paragraph)
		if !ok {
			t.Fatalf("Declined item %d is a %T, want a *comment.Paragraph", i, item.Content[0])
		}
		got = append(got, plainText(p.Text))
	}
	var want []string
	for _, c := range constructs[1:] {
		want = append(want, c.name)
	}
	if !slices.Equal(got, want) {
		t.Errorf("Declined today list = %q, want the constructs names in index order %q", got, want)
	}
}

// plainText is the text of texts' leading plain runs, followed, at the first
// run that is not plain, by that run's type in angle brackets, so that a link
// or other markup in a Declined item never compares equal to a name.
func plainText(texts []comment.Text) string {
	var b strings.Builder
	for _, text := range texts {
		plain, ok := text.(comment.Plain)
		if !ok {
			return b.String() + fmt.Sprintf("<%T>", text)
		}
		b.WriteString(string(plain))
	}
	return b.String()
}

// parsedDecline parses body as line 2 of the one schema document s.xsd, its
// <xs:schema> start tag declaring the xs and xsi prefixes, and returns the
// *DeclineError Generate returns for it.
func parsedDecline(t *testing.T, body string) *DeclineError {
	t.Helper()
	schema := parseString(t, "<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\" xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\">\n"+body+"\n</xs:schema>\n")
	err := Generate([]Target{{Schema: schema, Dir: filepath.Join(t.TempDir(), "out"), Package: "p"}})
	var de *DeclineError
	if !errors.As(err, &de) {
		t.Fatalf("Generate = %v, want an error errors.As retrieves a *DeclineError from", err)
	}
	return de
}

// TestConstructMapKey pins Construct's doc: a Construct is comparable and may
// key a map. Each Construct comes from a decline, never a literal: two empty
// choices declining at different elements share one key, and a nillable
// element's decline keys another.
func TestConstructMapKey(t *testing.T) {
	first := parsedDecline(t, `<xs:element name="e"><xs:complexType><xs:choice/></xs:complexType></xs:element>`).Construct()
	second := parsedDecline(t, `<xs:element name="f"><xs:complexType><xs:choice/></xs:complexType></xs:element>`).Construct()
	third := parsedDecline(t, `<xs:element name="n" nillable="true" type="xs:string"/>`).Construct()
	counts := make(map[Construct]int)
	for _, c := range []Construct{first, second, third} {
		counts[c]++
	}
	if got := len(counts); got != 2 {
		t.Errorf("map holds %d keys, want 2: %v", got, counts)
	}
	if got := counts[first]; got != 2 {
		t.Errorf("counts[%v] = %d, want 2", first, got)
	}
	if got := counts[third]; got != 1 {
		t.Errorf("counts[%v] = %d, want 1", third, got)
	}
	if first != second {
		t.Errorf("two empty-choice declines have Constructs %v and %v, want them ==", first, second)
	}
	if first == third {
		t.Errorf("an empty-choice and a nillable decline both have Construct %v, want them !=", first)
	}
}

// TestDeclineErrorLoc pins DeclineError.Loc's doc for parsed schemas: a
// decline charged to a declaration read from the document has that
// declaration's location.
func TestDeclineErrorLoc(t *testing.T) {
	tests := []struct {
		name, body string
		kind       construct
		want       xsderr.Loc
	}{
		{"nillable element", `<xs:element name="e" nillable="true" type="xs:string"/>`,
			constructElementNillable, xsderr.Loc{URI: "s.xsd", Line: 2, Col: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			de := parsedDecline(t, tt.body)
			if got, want := de.Construct(), (Construct{tt.kind}); got != want {
				t.Errorf("Construct() = %v, want %v", got, want)
			}
			if got := de.Loc(); got != tt.want {
				t.Errorf("Loc() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
