package codec

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/xsd"
)

// spR to spRef are the types codegen emits for the special-type rows of
// TestDecodeSpecial, each row pinning its own declaration: spR a document
// element's of xs:anySimpleType or xs:anyAtomicType, spAttr an attribute
// use's, optional, and spDefaulted one with a default, spValue simple
// content's, spValueAttr simple content with an attribute use, and spRef a
// sequence of an <element ref> to a top-level h and a local a.
type (
	spR         string
	spAttr      struct{ K *string }
	spDefaulted struct{ K string }
	spValue     struct{ Value string }
	spValueAttr struct {
		Value string
		K     *string
	}
	spH   string
	spRef struct {
		H spH
		A string
	}
)

// spSpaced is a lexical carrying a tab, a line feed and runs of spaces, which
// replace or collapse would change.
const spSpaced = " a \t\n b  "

// TestDecodeSpecial pins the lift of the special types, xs:anySimpleType and
// xs:anyAtomicType (Datatypes §2.4.2 dt-special): an item of either decodes
// into the string codegen gives it (codegen's package doc, "# Simple types"),
// holding its characters unchanged, the ·normalized value· of the preserve
// case (Structures §3.1.4 key-nv), for a document element, an attribute use
// whose declaration has no type, so xs:anySimpleType (§3.2.2.2
// dcl.att.local), present and absent with a default, one typed
// xs:anySimpleType, simple content extending either type (§3.4.2.2
// dcl.ctd.ctsc), an <element ref> to a head of xs:anySimpleType and a
// particle of xs:anyAtomicType; CheckDecode returns nil for each T.
func TestDecodeSpecial(t *testing.T) {
	rootOfAnyAtomic := `<xs:element name="r" type="xs:anyAtomicType"/>`
	rows := []struct {
		name, decls, attrs, content string
		emits                       []string
		decode                      func(schema *xsd.Schema, data []byte) (any, error)
		want                        any
	}{
		{"document element of xs:anySimpleType", `<xs:element name="r" type="xs:anySimpleType"/>`, "", spSpaced,
			[]string{"type R string\n"}, specialAs[spR], spR(spSpaced)},
		{"document element of xs:anyAtomicType", rootOfAnyAtomic, "", spSpaced,
			[]string{"type R string\n"}, specialAs[spR], spR(spSpaced)},
		{"empty document element of xs:anySimpleType with a default", `<xs:element name="r" type="xs:anySimpleType" default=" d&#9;"/>`, "", "",
			[]string{"type R string\n"}, specialAs[spR], spR(" d\t")},
		{"attribute use of no type, present", root(`<xs:attribute name="k"/>`), ` k=" x&#9;y&#10;z "`, "",
			[]string{"type R struct {\n\tK *string\n}\n"}, specialAs[spAttr], spAttr{K: new(" x\ty\nz ")}},
		{"attribute use of no type, absent with a default", root(`<xs:attribute name="k" default=" d "/>`), "", "",
			[]string{"type R struct {\n\tK string\n}\n"}, specialAs[spDefaulted], spDefaulted{K: " d "}},
		{"attribute use of xs:anySimpleType", root(`<xs:attribute name="k" type="xs:anySimpleType"/>`), ` k="1.0&#9;"`, "",
			[]string{"type R struct {\n\tK *string\n}\n"}, specialAs[spAttr], spAttr{K: new("1.0\t")}},
		{"simple content extending xs:anySimpleType", root(`<xs:simpleContent><xs:extension base="xs:anySimpleType"/></xs:simpleContent>`), "", spSpaced,
			[]string{"type R struct {\n\tValue string\n}\n"}, specialAs[spValue], spValue{Value: spSpaced}},
		{"simple content extending xs:anyAtomicType, with an attribute use", root(`<xs:simpleContent><xs:extension base="xs:anyAtomicType"><xs:attribute name="k" type="xs:string"/></xs:extension></xs:simpleContent>`), ` k="v"`, spSpaced,
			[]string{"type R struct {\n\tValue string\n\tK     *string\n}\n"}, specialAs[spValueAttr], spValueAttr{Value: spSpaced, K: new("v")}},
		{"element ref to a head of xs:anySimpleType, then a particle of xs:anyAtomicType", `<xs:element name="h" type="xs:anySimpleType"/>` + sequenceOf(`<xs:element ref="s:h"/><xs:element name="a" type="xs:anyAtomicType"/>`), "", `<s:h>` + spSpaced + `</s:h><s:a>` + spSpaced + `</s:a>`,
			[]string{"type H string\n", "type R struct {\n\tH H\n\tA string\n}\n"}, specialAs[spRef], spRef{H: spSpaced, A: spSpaced}},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			schema := schemaOf(t, row.decls)
			gen := generated(t, schema)
			for _, decl := range row.emits {
				if !bytes.Contains(gen, []byte(decl)) {
					t.Errorf("codegen emits\n%s\nwhich does not declare\n%s", gen, decl)
				}
			}
			data := instanceOf(row.attrs, row.content)
			mustAccept(t, schema, data)
			got, err := row.decode(schema, data)
			if err != nil || !reflect.DeepEqual(got, row.want) {
				t.Errorf("Decode = %#v, %v, want %#v", got, err, row.want)
			}
		})
	}

	schema := schemaOf(t, rootOfAnyAtomic)
	_, err := Decode[int](schema, strict.New(), instanceOf("", "1"))
	if !errors.Is(err, ErrTypeMismatch) {
		t.Errorf("Decode[int] = %v, want ErrTypeMismatch", err)
	}
	assertProbeMismatch(t, CheckDecode[int](schema, rootName), err)
}

// specialAs is a TestDecodeSpecial row's decode: CheckDecode[T]'s error for
// schema's s:r where it returns one, and otherwise the value Decode[T]
// returns for data, T the type codegen emits.
func specialAs[T any](schema *xsd.Schema, data []byte) (any, error) {
	if err := CheckDecode[T](schema, rootName); err != nil {
		return nil, fmt.Errorf("CheckDecode: %w", err)
	}
	got, err := Decode[T](schema, strict.New(), data)
	if err != nil {
		return nil, err
	}
	return *got, nil
}

// TestDecodeSpecialXsiType pins that an element of a special type carrying
// xsi:type still declines as one carrying xsi:type, its string holding no
// instance-specified type (Structures §3.3.4.3 cvc-elt clause 4): the
// document element, and a particle.
func TestDecodeSpecialXsiType(t *testing.T) {
	schema := schemaOf(t, `<xs:element name="r" type="xs:anySimpleType"/>`)
	data := instanceOf(` xsi:type="xs:string"`, "x")
	mustAccept(t, schema, data)
	_, err := Decode[spR](schema, strict.New(), data)
	assertDecline(t, err, constructXsiType, 1, 1)

	schema = schemaOf(t, sequenceOf(`<xs:element name="a" type="xs:anyAtomicType"/>`))
	data = instanceOf("", `<s:a xsi:type="xs:string">x</s:a>`)
	mustAccept(t, schema, data)
	_, err = Decode[struct{ A string }](schema, strict.New(), data)
	assertDecline(t, err, constructXsiType, 1, bytes.Index(data, []byte(`<s:a`))+1)
}
