package codec

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/validate"
	"github.com/kud360/goxsd8/validate/xmlsrc"
	"github.com/kud360/goxsd8/xsd"
)

// root declares s:r of an anonymous complex type whose content is body.
func root(body string) string {
	return `<xs:element name="r"><xs:complexType>` + body + `</xs:complexType></xs:element>`
}

// sequenceOf is a root whose content is a sequence of particles.
func sequenceOf(particles string) string {
	return root(`<xs:sequence>` + particles + `</xs:sequence>`)
}

// TestDecodeDeclines pins one row per construct Decode declines: each
// schema and instance is valid, so validation accepts, and Decode returns
// the construct's *DeclineError at the declining element's instance Loc,
// wrapping errors.ErrUnsupported, and no value. Each row decodes into a
// struct with no field, so a decline is never reported as the T check's
// caller fault: the declines precede it.
func TestDecodeDeclines(t *testing.T) {
	const a = `<s:a>x</s:a>`
	rows := []struct {
		name      string
		decls     string
		attrs     string
		content   string
		construct construct
		line, col int
	}{
		{"type table", `<xs:complexType name="rt"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>
			<xs:element name="r" type="s:rt"><xs:alternative test="@k" type="s:rt"/></xs:element>`, "", a, constructTypeTable, 1, 1},
		{"nillable", `<xs:element name="r" nillable="true"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType></xs:element>`,
			"", a, constructNillable, 1, 1},
		{"element default", sequenceOf(`<xs:element name="a" type="xs:string" default="d"/>`), "", a, constructElementDefault, 1, 1},
		{"element fixed", sequenceOf(`<xs:element name="a" type="xs:string" fixed="x"/>`), "", a, constructElementFixed, 1, 1},
		{"simple root", `<xs:element name="r" type="xs:string"/>`, "", "x", constructSimpleRoot, 1, 1},
		{"empty content", root(``), "", "", constructEmptyContent, 1, 1},
		{"simple content", root(`<xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent>`), "", "x", constructSimpleContent, 1, 1},
		{"mixed content", `<xs:element name="r"><xs:complexType mixed="true"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType></xs:element>`,
			"", a, constructMixedContent, 1, 1},
		{"open content", root(`<xs:openContent><xs:any namespace="##other" processContents="lax"/></xs:openContent><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>`),
			"", a, constructOpenContent, 1, 1},
		{"group ref content", `<xs:group name="g"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:group>` + root(`<xs:group ref="s:g"/>`),
			"", a, constructGroupRefContent, 1, 1},
		{"choice content", root(`<xs:choice><xs:element name="a" type="xs:string"/></xs:choice>`), "", a, constructChoiceContent, 1, 1},
		{"all content", root(`<xs:all><xs:element name="a" type="xs:string"/></xs:all>`), "", a, constructAllContent, 1, 1},
		{"content sequence occurring twice", root(`<xs:sequence maxOccurs="2"><xs:element name="a" type="xs:string"/></xs:sequence>`), "", a, constructSequenceOccurs, 1, 1},
		{"nested model group", sequenceOf(`<xs:choice><xs:element name="a" type="xs:string"/></xs:choice>`), "", a, constructNestedGroup, 1, 1},
		{"nested <group ref>", `<xs:group name="g"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:group>` + sequenceOf(`<xs:group ref="s:g"/>`),
			"", a, constructNestedGroup, 1, 1},
		{"wildcard particle", sequenceOf(`<xs:any processContents="lax"/>`), "", a, constructWildcardParticle, 1, 1},
		{"element ref", `<xs:element name="a" type="xs:string"/>` + sequenceOf(`<xs:element ref="s:a"/>`), "", a, constructElementRef, 1, 1},
		{"local element of complex type", `<xs:complexType name="ct"><xs:sequence/></xs:complexType>` + sequenceOf(`<xs:element name="a" type="s:ct"/>`),
			"", `<s:a/>`, constructComplexElement, 1, 1},
		{"local element of xs:anyType", sequenceOf(`<xs:element name="a"/>`), "", a, constructComplexElement, 1, 1},
		{"particles sharing an expanded name", sequenceOf(`<xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string"/><xs:element name="a" type="xs:string"/>`),
			"", a + `<s:b>y</s:b>` + a, constructSharedName, 1, 1},
		{"list", sequenceOf(`<xs:element name="a"><xs:simpleType><xs:list itemType="xs:string"/></xs:simpleType></xs:element>`), "", a, constructList, 1, 1},
		{"union", sequenceOf(`<xs:element name="a"><xs:simpleType><xs:union memberTypes="xs:string xs:int"/></xs:simpleType></xs:element>`), "", a, constructUnion, 1, 1},
		{"xs:anySimpleType", sequenceOf(`<xs:element name="a" type="xs:anySimpleType"/>`), "", a, constructSpecial, 1, 1},
		{"xs:anyAtomicType", sequenceOf(`<xs:element name="a" type="xs:anyAtomicType"/>`), "", a, constructSpecial, 1, 1},
		{"xs:anyURI", sequenceOf(`<xs:element name="a" type="xs:anyURI"/>`), "", a, constructAnyURI, 1, 1},
		{"other primitive", sequenceOf(`<xs:element name="a" type="xs:int"/>`), "", `<s:a>1</s:a>`, constructOtherPrimitive, 1, 1},
		{"attribute default", root(`<xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence><xs:attribute name="b" type="xs:string" default="d"/>`),
			"", a, constructAttributeDefault, 1, 1},
		{"attribute fixed", root(`<xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence><xs:attribute name="b" type="xs:string" fixed="f"/>`),
			"", a, constructAttributeFixed, 1, 1},
		{"attribute use's type", root(`<xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence><xs:attribute name="b" type="xs:boolean"/>`),
			` b="true"`, a, constructOtherPrimitive, 1, 1},
		{"attribute use in the xsi namespace", `<xs:import namespace="http://www.w3.org/2001/XMLSchema-instance"/>` +
			root(`<xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence><xs:attribute ref="xsi:nil"/>`), "", a, constructXsiAttribute, 1, 1},
		{"attribute wildcard", root(`<xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence><xs:anyAttribute processContents="lax"/>`),
			"", a, constructAttributeWildcard, 1, 1},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			schema := schemaOf(t, row.decls)
			data := instanceOf(row.attrs, row.content)
			mustAccept(t, schema, data)
			got, err := Decode[struct{}](schema, strict.New(), data)
			if got != nil {
				t.Fatalf("Decode = %v, want no value", got)
			}
			assertDecline(t, err, row.construct, row.line, row.col)
		})
	}
}

// TestDecodeXsiTypeDeclines pins the xsi:type decline (#3202's ruling,
// refused permanently where the selected type is concrete): an element
// carrying xsi:type, the document element or a child, declines at its own
// instance Loc after the T check, even where the xsi:type names the declared
// type itself. The child row is the one whose decoding into the base's field
// would lose the derived type's value: xs:token collapses what xs:string
// preserves.
func TestDecodeXsiTypeDeclines(t *testing.T) {
	schema := schemaOf(t, `<xs:complexType name="rt"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>
		<xs:element name="r" type="s:rt"/>`)
	rows := []struct {
		name      string
		attrs     string
		content   string
		line, col int
	}{
		{"the document element, naming its declared type", ` xsi:type="s:rt"`, `<s:a>x</s:a>`, 1, 1},
		{"a child, naming a type derived from its concrete one", "", `<s:a xsi:type="xs:token"> x  y </s:a>`, 1, 120},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			data := instanceOf(row.attrs, row.content)
			mustAccept(t, schema, data)
			got, err := Decode[struct{ A string }](schema, strict.New(), data)
			if got != nil {
				t.Fatalf("Decode = %v, want no value", got)
			}
			assertDecline(t, err, constructXsiType, row.line, row.col)
		})
	}
}

// TestDecodeUndeclaredRootXsiType pins that a document element no top-level
// declaration declares, which validate accepts through its xsi:type alone
// (§3.3.4.6 cvc-assess-elt), declines as xsi:type at its own Loc.
func TestDecodeUndeclaredRootXsiType(t *testing.T) {
	schema := schemaOf(t, `<xs:complexType name="rt"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`)
	data := instanceOf(` xsi:type="s:rt"`, `<s:a>x</s:a>`)
	mustAccept(t, schema, data)
	got, err := Decode[struct{ A string }](schema, strict.New(), data)
	if got != nil {
		t.Fatalf("Decode = %v, want no value", got)
	}
	assertDecline(t, err, constructXsiType, 1, 1)
}

// mustAccept fails the test unless validate accepts data against schema.
func mustAccept(t *testing.T, schema *xsd.Schema, data []byte) {
	t.Helper()
	v, err := validate.New(schema, strict.New())
	if err != nil {
		t.Fatal(err)
	}
	res, err := xmlsrc.Validate(v, bytes.NewReader(data))
	if err != nil || res.Err() != nil || res.Standing() != validate.NotProvenInvalid {
		t.Fatalf("the row's instance is not accepted: %v %v %v", err, res.Violations(), res.Unevaluated())
	}
}

// assertDecline fails the test unless err is the *DeclineError of
// construct c at instance line and col, wrapping errors.ErrUnsupported.
func assertDecline(t *testing.T, err error, c construct, line, col int) {
	t.Helper()
	var de *DeclineError
	if !errors.As(err, &de) {
		t.Fatalf("Decode error = %v, want a *DeclineError", err)
	}
	if de.construct != c {
		t.Errorf("construct = %s, want %s", de.construct.name(), c.name())
	}
	if de.Loc().Line != line || de.Loc().Col != col {
		t.Errorf("Loc = %s, want %d:%d", de.Loc(), line, col)
	}
	if !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("error %v does not wrap errors.ErrUnsupported", err)
	}
	if want := de.Loc().String() + ": codec does not decode " + c.name() + ": "; !strings.HasPrefix(err.Error(), want) {
		t.Errorf("error %q does not open with %q", err, want)
	}
}
