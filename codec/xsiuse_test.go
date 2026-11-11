package codec

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/xsd"
)

// The xsi:-use schemas, each declaring s:r of a struct type one of whose
// attribute uses references a built-in xsi: declaration other than xsi:nil
// (§3.2.7): useRequiredSchema's r holds a and a required
// xsi:noNamespaceSchemaLocation use; useOptionalSchema's r holds a and an
// optional xsi:schemaLocation use; useDefaultSchema's r holds a and an
// xsi:type use with default="xs:date", useFixedSchema's one with
// fixed="xs:integer", and useBareSchema's one with no value constraint;
// useNestedSchema's r holds n, whose own anonymous type holds b beside an
// xsi:schemaLocation use; useGroupSchema's r takes a required
// xsi:noNamespaceSchemaLocation use through an attribute group reference;
// useExtensionSchema's r is of ext, which extends base, holding a and an
// xsi:schemaLocation use, by the element c.
const (
	xsiImport         = `<xs:import namespace="http://www.w3.org/2001/XMLSchema-instance"/>`
	useRequiredSchema = xsiImport +
		`<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence><xs:attribute ref="xsi:noNamespaceSchemaLocation" use="required"/></xs:complexType></xs:element>`
	useOptionalSchema = xsiImport +
		`<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence><xs:attribute ref="xsi:schemaLocation"/></xs:complexType></xs:element>`
	useDefaultSchema = xsiImport +
		`<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence><xs:attribute ref="xsi:type" default="xs:date"/></xs:complexType></xs:element>`
	useFixedSchema = xsiImport +
		`<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence><xs:attribute ref="xsi:type" fixed="xs:integer"/></xs:complexType></xs:element>`
	useBareSchema = xsiImport +
		`<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence><xs:attribute ref="xsi:type"/></xs:complexType></xs:element>`
	useNestedSchema = xsiImport +
		`<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="n"><xs:complexType><xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence><xs:attribute ref="xsi:schemaLocation"/></xs:complexType></xs:element></xs:sequence></xs:complexType></xs:element>`
	useGroupSchema = xsiImport +
		`<xs:attributeGroup name="g"><xs:attribute ref="xsi:noNamespaceSchemaLocation" use="required"/></xs:attributeGroup>` +
		`<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence><xs:attributeGroup ref="s:g"/></xs:complexType></xs:element>`
	useExtensionSchema = xsiImport +
		`<xs:complexType name="base"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence><xs:attribute ref="xsi:schemaLocation"/></xs:complexType>` +
		`<xs:complexType name="ext"><xs:complexContent><xs:extension base="s:base"><xs:sequence><xs:element name="c" type="xs:string"/></xs:sequence></xs:extension></xs:complexContent></xs:complexType>` +
		`<xs:element name="r" type="s:ext"/>`
)

// useDefaultAttributesDoc is a schema document like schemaOf's whose
// defaultAttributes names the attribute group g, holding an optional
// xsi:schemaLocation use, so that r's type takes the use with no attribute
// group reference of its own (§3.4.2.4 dcl.ctd.attuses).
const useDefaultAttributesDoc = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:s="urn:s" targetNamespace="urn:s" elementFormDefault="qualified" defaultAttributes="s:g">` +
	xsiImport +
	`<xs:attributeGroup name="g"><xs:attribute ref="xsi:schemaLocation"/></xs:attributeGroup>` +
	`<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType></xs:element>` +
	`</xs:schema>`

// The types codegen emits for those schemas' s:r, which
// TestXsiUseTypesAreCodegens pins: useRequired for useRequiredSchema and
// useGroupSchema, useOptional for useOptionalSchema and
// useDefaultAttributesDoc, useType for useDefaultSchema, useFixedSchema and
// useBareSchema, useNested for useNestedSchema, its N useNestedN, and
// useExtension for useExtensionSchema.
type (
	useRequired struct {
		A                         string
		NoNamespaceSchemaLocation string
	}
	useOptional struct {
		A              string
		SchemaLocation *[]string
	}
	useType struct {
		A    string
		Type *tQName
	}
	useNested  struct{ N useNestedN }
	useNestedN struct {
		B              string
		SchemaLocation *[]string
	}
	useExtension struct {
		A              string
		C              string
		SchemaLocation *[]string
	}
)

// TestXsiUseTypesAreCodegens pins the types above to those codegen emits for
// each xsi:-use schema, in shape, so that the rows decoding into them decode
// into codegen's types: a built-in xsi: use's value constraint never makes
// its field bare (codegen/doc.go "# Attributes").
func TestXsiUseTypesAreCodegens(t *testing.T) {
	const (
		required = "type R struct {\n\tA                         string\n\tNoNamespaceSchemaLocation string\n}\n"
		optional = "type R struct {\n\tA              string\n\tSchemaLocation *[]string\n}\n"
		typ      = "type R struct {\n\tA    string\n\tType *QName\n}\n"
	)
	rows := []struct {
		schema *xsd.Schema
		want   []string
	}{
		{schemaOf(t, useRequiredSchema), []string{required}},
		{schemaOf(t, useOptionalSchema), []string{optional}},
		{schemaOf(t, useDefaultSchema), []string{typ}},
		{schemaOf(t, useFixedSchema), []string{typ}},
		{schemaOf(t, useBareSchema), []string{typ}},
		{schemaOf(t, useNestedSchema), []string{"type R struct {\n\tN N\n}\n", "type N struct {\n\tB              string\n\tSchemaLocation *[]string\n}\n"}},
		{schemaOf(t, useGroupSchema), []string{required}},
		{schemaOf(t, useExtensionSchema), []string{"type R struct {\n\tA              string\n\tC              string\n\tSchemaLocation *[]string\n}\n"}},
		{parseString(t, useDefaultAttributesDoc), []string{optional}},
	}
	for _, row := range rows {
		gen := generated(t, row.schema)
		for _, decl := range row.want {
			if !bytes.Contains(gen, []byte(decl)) {
				t.Errorf("codegen emits\n%s\nwhich does not declare\n%s", gen, decl)
			}
		}
	}
}

// TestDecodeXsiUse pins that a struct type's attribute use of xsi:type,
// xsi:schemaLocation or xsi:noNamespaceSchemaLocation decodes into the field
// codegen gives it: the field holds the instance's attribute of that name,
// which satisfies a required use (cvc-complex-type clause 3), and an absent
// one's field is nil whatever the use's value constraint, since a built-in
// xsi: use is never a defaulted attribute (§3.4.4.2 key-dflt-att clause 4).
// The xsi:type rows' instances carry no xsi:type: one carried declines
// (TestDecodeXsiUseXsiTypeDeclines).
func TestDecodeXsiUse(t *testing.T) {
	const a = "<s:a>x</s:a>"
	rows := []struct {
		name   string
		schema *xsd.Schema
		attrs  string
		body   string
		into   target
		want   any
	}{
		{"a required xsi:noNamespaceSchemaLocation use, carried", schemaOf(t, useRequiredSchema), ` xsi:noNamespaceSchemaLocation="n.xsd"`, a, into[useRequired](),
			useRequired{A: "x", NoNamespaceSchemaLocation: "n.xsd"}},
		{"an optional xsi:schemaLocation use, absent", schemaOf(t, useOptionalSchema), "", a, into[useOptional](), useOptional{A: "x"}},
		{"an optional xsi:schemaLocation use, carried", schemaOf(t, useOptionalSchema), ` xsi:schemaLocation="urn:s s.xsd"`, a, into[useOptional](),
			useOptional{A: "x", SchemaLocation: &[]string{"urn:s", "s.xsd"}}},
		{"an xsi:type use with a default, absent", schemaOf(t, useDefaultSchema), "", a, into[useType](), useType{A: "x"}},
		{"an xsi:type use with a fixed value, absent", schemaOf(t, useFixedSchema), "", a, into[useType](), useType{A: "x"}},
		{"an xsi:type use with no value constraint, absent", schemaOf(t, useBareSchema), "", a, into[useType](), useType{A: "x"}},
		{"a nested particle's type, its element carrying xsi:schemaLocation", schemaOf(t, useNestedSchema), "", `<s:n xsi:schemaLocation="urn:s s.xsd"><s:b>y</s:b></s:n>`, into[useNested](),
			useNested{N: useNestedN{B: "y", SchemaLocation: &[]string{"urn:s", "s.xsd"}}}},
		{"a use through an attribute group reference", schemaOf(t, useGroupSchema), ` xsi:noNamespaceSchemaLocation="n.xsd"`, a, into[useRequired](),
			useRequired{A: "x", NoNamespaceSchemaLocation: "n.xsd"}},
		{"a use an extension inherits", schemaOf(t, useExtensionSchema), ` xsi:schemaLocation="urn:s s.xsd"`, a + "<s:c>z</s:c>", into[useExtension](),
			useExtension{A: "x", C: "z", SchemaLocation: &[]string{"urn:s", "s.xsd"}}},
		{"a use the schema's defaultAttributes brings", parseString(t, useDefaultAttributesDoc), ` xsi:schemaLocation="urn:s s.xsd"`, a, into[useOptional](),
			useOptional{A: "x", SchemaLocation: &[]string{"urn:s", "s.xsd"}}},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			data := instanceOf(row.attrs, row.body)
			mustAccept(t, row.schema, data)
			got, err := row.into.decode(row.schema, data)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, row.want) {
				t.Errorf("Decode = %#v, want %#v", got, row.want)
			}
			if err := row.into.check(row.schema); err != nil {
				t.Errorf("CheckDecode = %v, want nil for a row Decode decodes", err)
			}
		})
	}
}

// TestDecodeXsiUseXsiTypeDeclines pins that an element carrying xsi:type
// declines as xsi:type even where its type has an xsi:type use, whose field
// T holds: the document element's, and a child's, each of a named type with
// that use.
func TestDecodeXsiUseXsiTypeDeclines(t *testing.T) {
	schema := schemaOf(t, xsiImport+
		`<xs:complexType name="rt"><xs:sequence><xs:element name="n" type="s:nt"/></xs:sequence><xs:attribute ref="xsi:type"/></xs:complexType>`+
		`<xs:complexType name="nt"><xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence><xs:attribute ref="xsi:type"/></xs:complexType>`+
		`<xs:element name="r" type="s:rt"/>`)
	type nt struct {
		B    string
		Type *tQName
	}
	type rt struct {
		N    nt
		Type *tQName
	}
	rows := []struct {
		name      string
		attrs     string
		content   string
		line, col int
	}{
		{"the document element", ` xsi:type="s:rt"`, `<s:n><s:b>y</s:b></s:n>`, 1, 1},
		{"a child", "", `<s:n xsi:type="s:nt"><s:b>y</s:b></s:n>`, 1, 120},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			data := instanceOf(row.attrs, row.content)
			mustAccept(t, schema, data)
			got, err := Decode[rt](schema, strict.New(), data)
			if got != nil {
				t.Fatalf("Decode = %v, want no value", got)
			}
			assertDecline(t, err, constructXsiType, row.line, row.col)
		})
	}
}
