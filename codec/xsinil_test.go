package codec

import (
	"bytes"
	"reflect"
	"testing"
)

// The xsi:nil-use schemas, each declaring s:r, not nillable, of a struct type
// one of whose attribute uses is xsi:nil's, to which codegen gives no field
// (codegen/doc.go "# Attributes"): nilUseSchema's r holds the element a;
// nilNestedSchema's r holds n, whose own anonymous type holds b beside the
// use; nilWildcardSchema's r adds a lax {attribute wildcard} to
// nilUseSchema's.
const (
	nilUseSchema = `<xs:import namespace="http://www.w3.org/2001/XMLSchema-instance"/>` +
		`<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence><xs:attribute ref="xsi:nil"/></xs:complexType></xs:element>`
	nilNestedSchema = `<xs:import namespace="http://www.w3.org/2001/XMLSchema-instance"/>` +
		`<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="n"><xs:complexType><xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence><xs:attribute ref="xsi:nil"/></xs:complexType></xs:element></xs:sequence></xs:complexType></xs:element>`
	nilWildcardSchema = `<xs:import namespace="http://www.w3.org/2001/XMLSchema-instance"/>` +
		`<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence><xs:attribute ref="xsi:nil"/><xs:anyAttribute processContents="lax"/></xs:complexType></xs:element>`
)

// The types codegen emits for those schemas' s:r, which
// TestXsiNilUseTypesAreCodegens pins: nilUse nilUseSchema's R, nilNested
// nilNestedSchema's R, its N nilNestedN, and nilWildcard nilWildcardSchema's
// R.
type (
	nilUse      struct{ A string }
	nilNested   struct{ N nilNestedN }
	nilNestedN  struct{ B string }
	nilWildcard struct {
		A            string
		AnyAttribute []anyAttribute
	}
)

// TestXsiNilUseTypesAreCodegens pins the types above to those codegen emits
// for each xsi:nil-use schema, in shape, so that the rows decoding into them
// decode into codegen's types.
func TestXsiNilUseTypesAreCodegens(t *testing.T) {
	rows := []struct {
		decls string
		want  []string
	}{
		{nilUseSchema, []string{"type R struct {\n\tA string\n}\n"}},
		{nilNestedSchema, []string{"type R struct {\n\tN N\n}\n", "type N struct {\n\tB string\n}\n"}},
		{nilWildcardSchema, []string{"type R struct {\n\tA            string\n\tAnyAttribute []AnyAttribute\n}\n"}},
	}
	for _, row := range rows {
		gen := generated(t, schemaOf(t, row.decls))
		for _, decl := range row.want {
			if !bytes.Contains(gen, []byte(decl)) {
				t.Errorf("codegen emits\n%s\nwhich does not declare\n%s", gen, decl)
			}
		}
	}
}

// TestDecodeXsiNilUse pins that a struct type's attribute use of xsi:nil
// gives no attribute field and no decline (goshape.FieldAttributeUses, as
// codegen reads it): the document element's type, a nested particle's, and
// one beside an {attribute wildcard}, whose AnyAttribute holds the
// instance's other attribute. No row's instance carries xsi:nil: its
// declaration is not nillable, which admits none (cvc-elt clause 3.1).
func TestDecodeXsiNilUse(t *testing.T) {
	rows := []struct {
		name  string
		decls string
		attrs string
		body  string
		into  target
		want  any
	}{
		{"the document element's type", nilUseSchema, "", "<s:a>x</s:a>", into[nilUse](), nilUse{A: "x"}},
		{"a nested particle's type", nilNestedSchema, "", "<s:n><s:b>y</s:b></s:n>", into[nilNested](), nilNested{N: nilNestedN{B: "y"}}},
		{"beside an attribute wildcard", nilWildcardSchema, ` k="v"`, "<s:a>x</s:a>", into[nilWildcard](),
			nilWildcard{A: "x", AnyAttribute: []anyAttribute{{Local: "k", Value: "v"}}}},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			schema := schemaOf(t, row.decls)
			data := instanceOf(row.attrs, row.body)
			mustAccept(t, schema, data)
			got, err := row.into.decode(schema, data)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, row.want) {
				t.Errorf("Decode = %#v, want %#v", got, row.want)
			}
			if err := row.into.check(schema); err != nil {
				t.Errorf("CheckDecode = %v, want nil for a row Decode decodes", err)
			}
		})
	}
}
