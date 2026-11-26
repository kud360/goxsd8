package codec

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/codec/testdata/readme/order"
	"github.com/kud360/goxsd8/internal/goshape"
	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/parser/xmltree"
	"github.com/kud360/goxsd8/validate"
	"github.com/kud360/goxsd8/validate/xmlsrc"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// accepts fails t unless validate accepts data against schema on strict's
// backend: no source fault and Standing NotProvenInvalid.
func accepts(t *testing.T, schema *xsd.Schema, data []byte) {
	t.Helper()
	v, err := validate.New(schema, strict.New())
	if err != nil {
		t.Fatal(err)
	}
	res, err := xmlsrc.Validate(v, bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if res.Err() != nil || res.Standing() != validate.NotProvenInvalid {
		t.Fatalf("validate of %s: Err %v, Standing %v, violations %v", data, res.Err(), res.Standing(), res.Violations())
	}
}

// roundTrip appends v as the document element name of schema under opts,
// requires the bytes want when want is not empty, validate's acceptance of
// them, and Decode's T under opts equal to v by eq, and returns the bytes.
func roundTrip[T any](t *testing.T, schema *xsd.Schema, name xsd.QName, v T, want string, eq func(a, b T) bool, opts ...Option) []byte {
	t.Helper()
	data, err := appendChecked(t, nil, schema, strict.New(), name, v, opts...)
	if err != nil {
		t.Fatalf("Append: %v", err)
	}
	if want != "" && string(data) != want {
		t.Fatalf("Append wrote\n%s\nwant\n%s", data, want)
	}
	accepts(t, schema, data)
	got, err := Decode[T](schema, strict.New(), data, opts...)
	if err != nil {
		t.Fatalf("Decode(%s): %v", data, err)
	}
	if !eq(unscoped(*got).(T), unscoped(v).(T)) {
		t.Fatalf("Decode(%s) = %#v, want %#v", data, *got, v)
	}
	return data
}

// appendChecked returns Append's answer for its arguments and fails t
// unless CheckAppend[T]'s answer for schema, name and opts agrees with it: a
// non-nil CheckAppend error is Append's, of equal Error text and equal class
// (staticClass), and a nil one leaves Append nil, an item 6 or item 7 error
// or a fault of backend: nil, item 6's ErrInvalidValue error, a plain error
// without item 2's or item 3's text, or item 7's decline, by
// constructNoCanonical, constructQNameSpelling, constructValueConstraint,
// constructXsiAttribute, constructStrictAttributeWildcard,
// constructGovernedContent or constructLocallyTypedMatch.
func appendChecked[T any](t *testing.T, dst []byte, schema *xsd.Schema, backend value.Backend, name xsd.QName, v T, opts ...Option) ([]byte, error) {
	t.Helper()
	out, err := Append(dst, schema, backend, name, v, opts...)
	checked := CheckAppend[T](schema, name, opts...)
	if checked == nil {
		class := staticClass(err)
		static := class == "plain" && (strings.HasPrefix(err.Error(), "codec: no top-level element declaration") || strings.Contains(err.Error(), "{abstract} true"))
		if static || err != nil && class != "plain" && class != "invalid" && class != "decline "+constructNoCanonical.name() && class != "decline "+constructQNameSpelling.name() && class != "decline "+constructValueConstraint.name() && class != "decline "+constructXsiAttribute.name() && class != "decline "+constructStrictAttributeWildcard.name() && class != "decline "+constructGovernedContent.name() && class != "decline "+constructLocallyTypedMatch.name() {
			t.Errorf("CheckAppend = nil, but Append returned %v, %s", err, class)
		}
		return out, err
	}
	if err == nil || err.Error() != checked.Error() || staticClass(err) != staticClass(checked) {
		t.Errorf("CheckAppend = %v, %s; Append returned %v, %s", checked, staticClass(checked), err, staticClass(err))
	}
	return out, err
}

// staticClass names err's class among Append's errors: "decline " and its
// construct for a *DeclineError, "mismatch" for an error errors.Is reports as
// ErrTypeMismatch, "invalid" for one errors.Is reports as ErrInvalidValue,
// "plain" for every other non-nil error, and "nil".
func staticClass(err error) string {
	var d *DeclineError
	switch {
	case err == nil:
		return "nil"
	case errors.As(err, &d):
		return "decline " + d.Construct().String()
	case errors.Is(err, ErrTypeMismatch):
		return "mismatch"
	case errors.Is(err, ErrInvalidValue):
		return "invalid"
	}
	return "plain"
}

// deepEqual is reflect.DeepEqual over T.
func deepEqual[T any](a, b T) bool { return reflect.DeepEqual(a, b) }

// TestAppendReadmeOrder pins README's fixture round trip: the Order Decode
// returns for order.xml encodes to a document validate accepts and Decode
// turns back into an equal Order, and an Order of one item and a kind is
// written exactly as the package doc's "# What encodes" states.
func TestAppendReadmeOrder(t *testing.T) {
	readme := filepath.Join("..", "testdata", "readme")
	schema, err := parser.Parse("order.xsd", parser.WithResolver(loader.Dir(readme)))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(readme, "order.xml"))
	if err != nil {
		t.Fatal(err)
	}
	o, err := Decode[order.Order](schema, strict.New(), data)
	if err != nil {
		t.Fatal(err)
	}
	name := xsd.QName{Space: "urn:o", Local: "order"}
	roundTrip(t, schema, name, *o, `<order xmlns="urn:o" kind="x"><item>a</item><item>b</item></order>`, deepEqual[order.Order])
	kind := "k"
	roundTrip(t, schema, name, order.Order{Item: []string{"a"}, Kind: &kind}, `<order xmlns="urn:o" kind="k"><item>a</item></order>`, deepEqual[order.Order])
}

// TestAppendShapes pins one row per Go shape of Decode's first slice over
// shapesSchema: a required element and an xs:token element (string), an
// optional one (*string) present and absent, a repeated one ([]string) of
// two items and of none, a required attribute (string) and an optional one
// (*string) present and absent. Each writes the bytes shown, which validate
// accepts and Decode turns back into the value.
func TestAppendShapes(t *testing.T) {
	schema := schemaOf(t, shapesSchema)
	str := func(s string) *string { return &s }
	for _, row := range []struct {
		name string
		v    shapes
		want string
	}{
		{"required and token elements, required attribute", shapes{One: "x", Tok: "t", Req: "r"},
			`<r xmlns="urn:s" req="r"><one>x</one><tok>t</tok></r>`},
		{"optional element present", shapes{One: "x", Opt: str("o"), Tok: "t", Req: "r"},
			`<r xmlns="urn:s" req="r"><one>x</one><opt>o</opt><tok>t</tok></r>`},
		{"repeated element of two items", shapes{One: "x", Rep: []string{"1", " 2 "}, Tok: "t", Req: "r"},
			`<r xmlns="urn:s" req="r"><one>x</one><rep>1</rep><rep> 2 </rep><tok>t</tok></r>`},
		{"optional attribute present", shapes{One: "x", Tok: "t", Req: "r", Opt2: str("a b")},
			`<r xmlns="urn:s" req="r" opt="a b"><one>x</one><tok>t</tok></r>`},
		{"every field present, an empty string an empty-element tag", shapes{One: "", Opt: str("o"), Rep: []string{"1"}, Tok: "a b", Req: "", Opt2: str("o2")},
			`<r xmlns="urn:s" req="" opt="o2"><one/><opt>o</opt><rep>1</rep><tok>a b</tok></r>`},
	} {
		t.Run(row.name, func(t *testing.T) {
			roundTrip(t, schema, rootName, row.v, row.want, deepEqual[shapes])
		})
	}
}

// encodeSchema declares, in urn:s, a document element of each further
// shape the package doc's "# What encodes" admits: a simple type, xs:double
// and xs:string, empty content, xs:double in an element and an attribute, a
// token attribute, and xs:double and xs:string restrictions by facets.
const encodeSchema = `
  <xs:element name="d" type="xs:double"/>
  <xs:element name="str" type="xs:string"/>
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="a" type="xs:string"/>
    </xs:complexType>
  </xs:element>
  <xs:element name="ds">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="v" type="xs:double"/>
      </xs:sequence>
      <xs:attribute name="w" type="xs:double" use="required"/>
    </xs:complexType>
  </xs:element>
  <xs:element name="ta">
    <xs:complexType>
      <xs:attribute name="t" type="xs:token" use="required"/>
    </xs:complexType>
  </xs:element>
  <xs:simpleType name="one">
    <xs:restriction base="xs:double">
      <xs:pattern value="1"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="oneBelowZero">
    <xs:restriction base="xs:double">
      <xs:pattern value="1"/>
      <xs:maxInclusive value="0"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="nonPositive">
    <xs:restriction base="xs:double">
      <xs:maxInclusive value="0"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="onlyA">
    <xs:restriction base="xs:string">
      <xs:pattern value="a"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="p" type="s:one"/>
  <xs:element name="pb" type="s:oneBelowZero"/>
  <xs:element name="np" type="s:nonPositive"/>
  <xs:element name="oa" type="s:onlyA"/>
  <xs:element name="order2">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="p" type="s:one"/>
        <xs:element name="np" type="s:nonPositive"/>
      </xs:sequence>
      <xs:attribute name="oa" type="s:onlyA"/>
    </xs:complexType>
  </xs:element>`

// The T codegen emits for encodeSchema's document elements, which Append's
// item 5 holds each to.
type (
	dblRoot   float64
	strRoot   string
	emptyRoot struct{ A *string }
	doubles   struct {
		V float64
		W float64
	}
	tokenAttr struct{ T string }
	order2    struct {
		P  float64
		Np float64
		Oa *string
	}
)

// sname is the expanded name of local in urn:s.
func sname(local string) xsd.QName { return xsd.QName{Space: "urn:s", Local: local} }

// sameDouble reports whether a and b are one float64, every NaN one.
func sameDouble[T ~float64](a, b T) bool {
	return math.Float64bits(float64(a)) == math.Float64bits(float64(b)) || math.IsNaN(float64(a)) && math.IsNaN(float64(b))
}

// TestAppendDouble pins the canonical form of xs:double: 1 is written
// 1.0E0, the form strict's Canonical renders, never strconv's 1; the
// infinities, NaN, of Go's payload or another, and negative zero as INF,
// -INF, NaN and -0.0E0; each accepted by validate and decoded back to the
// same float64, compared by its bits, every NaN equal. A float64 in an
// element and an attribute takes the same route.
func TestAppendDouble(t *testing.T) {
	schema := schemaOf(t, encodeSchema)
	for _, row := range []struct {
		v    float64
		want string
	}{
		{1, "1.0E0"},
		{math.Inf(1), "INF"},
		{math.Inf(-1), "-INF"},
		{math.NaN(), "NaN"},
		{math.Float64frombits(0x7FF4000000000000), "NaN"},
		{math.Copysign(0, -1), "-0.0E0"},
		{0.1, "1.0E-1"},
		{1e21, "1.0E21"},
		{5e-324, "5.0E-324"},
	} {
		t.Run(row.want, func(t *testing.T) {
			roundTrip(t, schema, sname("d"), dblRoot(row.v), `<d xmlns="urn:s">`+row.want+`</d>`, sameDouble[dblRoot])
		})
	}
	t.Run("not Go's own form", func(t *testing.T) {
		data, err := appendChecked(t, nil, schema, strict.New(), sname("d"), dblRoot(1))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) == `<d xmlns="urn:s">`+strconv.FormatFloat(1, 'g', -1, 64)+`</d>` {
			t.Errorf("Append wrote %s, strconv.FormatFloat's 1", data)
		}
	})
	t.Run("element and attribute", func(t *testing.T) {
		roundTrip(t, schema, sname("ds"), doubles{V: 2.5, W: -1}, `<ds xmlns="urn:s" w="-1.0E0"><v>2.5E0</v></ds>`, deepEqual[doubles])
	})
}

// TestAppendSimpleAndEmpty pins the two shapes "# What encodes" admits
// beyond Decode's first slice: a document element of simple type, an empty
// string as an empty-element tag, and one of empty content.
func TestAppendSimpleAndEmpty(t *testing.T) {
	schema := schemaOf(t, encodeSchema)
	roundTrip(t, schema, sname("str"), strRoot("v w"), `<str xmlns="urn:s">v w</str>`, deepEqual[strRoot])
	roundTrip(t, schema, sname("str"), strRoot(""), `<str xmlns="urn:s"/>`, deepEqual[strRoot])
	roundTrip(t, schema, sname("e"), emptyRoot{}, `<e xmlns="urn:s"/>`, deepEqual[emptyRoot])
	v := "x"
	roundTrip(t, schema, sname("e"), emptyRoot{A: &v}, `<e xmlns="urn:s" a="x"/>`, deepEqual[emptyRoot])
}

// TestAppendEscapes pins the escapes "# What encodes" states: '<', '&' and
// '>' everywhere, so "]]>" is never written; CR everywhere; '"', TAB and LF
// in an attribute value; TAB and LF literal in content and the apostrophe
// literal everywhere. Each document round-trips, the attribute's TAB, LF and CR
// surviving attribute-value normalization and the content's CR end-of-line
// handling.
func TestAppendEscapes(t *testing.T) {
	schema := schemaOf(t, shapesSchema)
	v := shapes{One: "a<b&c>d]]>e\r\tf\ng'", Tok: "t", Req: "x\ty\nz\r\"'<&>]]>"}
	roundTrip(t, schema, rootName, v,
		`<r xmlns="urn:s" req="x&#x9;y&#xA;z&#xD;&quot;'&lt;&amp;&gt;]]&gt;"><one>a&lt;b&amp;c&gt;d]]&gt;e&#xD;`+"\tf\ng'"+`</one><tok>t</tok></r>`,
		deepEqual[shapes])
}

// TestAppendNamespaces pins the default bindings "# What encodes" states:
// a child of an unqualified declaration under a document element in urn:u
// carries xmlns="", no namespace writes no declaration, and a child in
// another namespace than its parent's, here a local element of a type
// imported from urn:b, carries that namespace's. README's qualified child
// is TestAppendReadmeOrder's. It pins too how "# What encodes" names an
// attribute use's attribute of a qualified declaration: pN:Local, never under
// the element's default namespace though that is the attribute's, one
// declaration serving two attributes, an xs:QName value in the attribute's
// namespace on an element in none, whichever of the two is written first,
// and a child's attribute in the namespace an ancestor declared a prefix for;
// xml:Local, undeclared, for the XML namespace; and pN:Local for a location
// hint's use, the one XSI-namespace use that writes.
func TestAppendNamespaces(t *testing.T) {
	type child struct{ A string }
	type qualified struct{ A, B string }
	t.Run("qualified attributes", func(t *testing.T) {
		schema := schemaOf(t, `<xs:element name="r"><xs:complexType>
		  <xs:attribute name="a" type="xs:string" form="qualified" use="required"/>
		  <xs:attribute name="b" type="xs:string" form="qualified" use="required"/></xs:complexType></xs:element>`)
		roundTrip(t, schema, sname("r"), qualified{A: "x", B: "y"}, `<r xmlns="urn:s" xmlns:p1="urn:s" p1:a="x" p1:b="y"/>`, deepEqual[qualified])
	})
	t.Run("qualified attribute beside an xs:QName value", func(t *testing.T) {
		type nameFirst struct {
			A string
			Q tQName
		}
		type valueFirst struct {
			Q tQName
			A string
		}
		type pair struct {
			N nameFirst
			V valueFirst
		}
		schema := schemaOf(t, `<xs:element name="r"><xs:complexType><xs:sequence>
		  <xs:element name="n" form="unqualified"><xs:complexType>
		    <xs:attribute name="a" type="xs:string" form="qualified" use="required"/>
		    <xs:attribute name="q" type="xs:QName" use="required"/></xs:complexType></xs:element>
		  <xs:element name="v" form="unqualified"><xs:complexType>
		    <xs:attribute name="q" type="xs:QName" use="required"/>
		    <xs:attribute name="a" type="xs:string" form="qualified" use="required"/></xs:complexType></xs:element>
		</xs:sequence></xs:complexType></xs:element>`)
		z := tQName{Space: "urn:s", Local: "z"}
		v := pair{N: nameFirst{A: "w", Q: z}, V: valueFirst{Q: z, A: "w"}}
		roundTrip(t, schema, sname("r"), v,
			`<r xmlns="urn:s"><n xmlns="" xmlns:p1="urn:s" p1:a="w" q="p1:z"/><v xmlns="" xmlns:p1="urn:s" q="p1:z" p1:a="w"/></r>`, deepEqual[pair])
	})
	t.Run("qualified attribute under an ancestor's prefix", func(t *testing.T) {
		type inner struct{ B string }
		type outer struct {
			C inner
			A string
		}
		schema := schemaOf(t, `<xs:element name="r"><xs:complexType><xs:sequence>
		  <xs:element name="c" form="unqualified"><xs:complexType><xs:attribute name="b" type="xs:string" form="qualified" use="required"/></xs:complexType></xs:element>
		</xs:sequence><xs:attribute name="a" type="xs:string" form="qualified" use="required"/></xs:complexType></xs:element>`)
		roundTrip(t, schema, sname("r"), outer{C: inner{B: "y"}, A: "x"}, `<r xmlns="urn:s" xmlns:p1="urn:s" p1:a="x"><c xmlns="" p1:b="y"/></r>`, deepEqual[outer])
	})
	t.Run("XML namespace attribute", func(t *testing.T) {
		type spaced struct{ Space *string }
		schema := parseString(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:s">
		  <xs:import namespace="http://www.w3.org/XML/1998/namespace"/>
		  <xs:element name="r"><xs:complexType><xs:attribute ref="xml:space"/></xs:complexType></xs:element>
		</xs:schema>`)
		preserve := "preserve"
		roundTrip(t, schema, sname("r"), spaced{Space: &preserve}, `<r xmlns="urn:s" xml:space="preserve"/>`, deepEqual[spaced])
	})
	t.Run("xsi:noNamespaceSchemaLocation attribute use", func(t *testing.T) {
		type hinted struct{ NoNamespaceSchemaLocation *string }
		schema := schemaOf(t, `<xs:element name="r"><xs:complexType><xs:attribute ref="xsi:noNamespaceSchemaLocation"/></xs:complexType></xs:element>`)
		location := "x.xsd"
		roundTrip(t, schema, sname("r"), hinted{NoNamespaceSchemaLocation: &location},
			`<r xmlns="urn:s" xmlns:p1="http://www.w3.org/2001/XMLSchema-instance" p1:noNamespaceSchemaLocation="x.xsd"/>`, deepEqual[hinted])
	})
	t.Run("xsi:schemaLocation attribute use", func(t *testing.T) {
		type hinted struct{ SchemaLocation []string }
		schema := schemaOf(t, `<xs:element name="r"><xs:complexType><xs:attribute ref="xsi:schemaLocation" use="required"/></xs:complexType></xs:element>`)
		roundTrip(t, schema, sname("r"), hinted{SchemaLocation: []string{"urn:x", "x.xsd"}},
			`<r xmlns="urn:s" xmlns:p1="http://www.w3.org/2001/XMLSchema-instance" p1:schemaLocation="urn:x x.xsd"/>`, deepEqual[hinted])
	})
	t.Run("unqualified child", func(t *testing.T) {
		schema := parseString(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:u">
		  <xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType></xs:element>
		</xs:schema>`)
		roundTrip(t, schema, xsd.QName{Space: "urn:u", Local: "r"}, child{A: "x"}, `<r xmlns="urn:u"><a xmlns="">x</a></r>`, deepEqual[child])
	})
	t.Run("no namespace", func(t *testing.T) {
		schema := parseString(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType></xs:element>
		</xs:schema>`)
		roundTrip(t, schema, xsd.QName{Local: "r"}, child{A: "x"}, `<r><a>x</a></r>`, deepEqual[child])
	})
	t.Run("imported namespace", func(t *testing.T) {
		schema, err := parser.Parse("a.xsd", parser.WithResolver(loader.Map(map[string]string{
			"a.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:b="urn:b" targetNamespace="urn:a">
			  <xs:import namespace="urn:b" schemaLocation="b.xsd"/>
			  <xs:element name="r" type="b:t"/>
			</xs:schema>`,
			"b.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:b" elementFormDefault="qualified">
			  <xs:complexType name="t"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>
			</xs:schema>`,
		})))
		if err != nil {
			t.Fatal(err)
		}
		roundTrip(t, schema, xsd.QName{Space: "urn:a", Local: "r"}, child{A: "x"}, `<r xmlns="urn:a"><a xmlns="urn:b">x</a></r>`, deepEqual[child])
	})
}

// TestAppendGroupRef pins a content sequence codegen splices through
// <group ref>, which "# What encodes" admits as it does an inline one.
func TestAppendGroupRef(t *testing.T) {
	schema := schemaOf(t, `<xs:group name="g"><xs:sequence><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:double" minOccurs="0"/></xs:sequence></xs:group>
	  <xs:element name="r"><xs:complexType><xs:group ref="s:g"/></xs:complexType></xs:element>`)
	type grouped struct {
		A string
		B *float64
	}
	b := 2.0
	roundTrip(t, schema, rootName, grouped{A: "x", B: &b}, `<r xmlns="urn:s"><a>x</a><b>2.0E0</b></r>`, deepEqual[grouped])
}

// The <element ref> types Append writes beside decode_test.go's: refX,
// refNested and refOuter are refNamespaceSchema's X, its complex N and R;
// refSelf is refSelfSchema's R, whose optional ref names r itself.
type (
	refX      string
	refNested struct{ A string }
	refOuter  struct {
		X refX
		N refNested
	}
	refSelf struct {
		A string
		R *refSelf
	}
)

// refSelfSchema declares s:r of a local a and an optional <element ref> back
// to r, which its own complex type reaches again.
const refSelfSchema = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/><xs:element ref="s:r" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>`

// TestAppendElementRef pins "# What encodes"' <element ref> particle
// (§3.3.2.4 ref.elt.global): a ref to a top-level declaration whose field
// codegen gives no sealed interface is written as an element of the
// declaration's own expanded name (§3.9.4.3 cvc-accept clause 2.3.1), which
// validate accepts and Decode returns equal: decode_test.go's refSchema in
// each occurrence shape, present and absent; a head blocking substitution,
// and one whose one member is abstract, examined as a local declaration; a
// choice's ref alternative, once and repeated; a ref into another target
// namespace, beside a ref of complex type; and a ref from a declaration's own
// type back to it. TestAppendSubstitutionGroup pins a ref whose field codegen
// gives a sealed interface.
func TestAppendElementRef(t *testing.T) {
	b := refB("1 2")
	t.Run("one, optional and repeated, present", func(t *testing.T) {
		roundTrip(t, schemaOf(t, refSchema), rootName, refs{A: "x", B: &b, C: []refC{"p", "q"}},
			`<r xmlns="urn:s"><a>x</a><b>1 2</b><c>p</c><c>q</c></r>`, deepEqual[refs])
	})
	t.Run("optional and repeated absent", func(t *testing.T) {
		roundTrip(t, schemaOf(t, refSchema), rootName, refs{A: "x"}, `<r xmlns="urn:s"><a>x</a></r>`, deepEqual[refs])
	})
	t.Run("a head blocking substitution", func(t *testing.T) {
		roundTrip(t, schemaOf(t, blockedSchema), rootName, refHolder{H: "x"}, `<r xmlns="urn:s"><h>x</h></r>`, deepEqual[refHolder])
	})
	t.Run("a head whose one member is abstract", func(t *testing.T) {
		roundTrip(t, schemaOf(t, abstractMemberSchema), rootName, refHolder{H: "x"}, `<r xmlns="urn:s"><h>x</h></r>`, deepEqual[refHolder])
	})
	t.Run("a choice's ref alternative", func(t *testing.T) {
		roundTrip(t, schemaOf(t, refChoiceSchema), rootName, refChoice(refChoiceA2{A: "v"}), `<r xmlns="urn:s"><A>v</A></r>`, deepEqual[refChoice], refChoiceBranches)
	})
	t.Run("a choice's repeated ref alternative", func(t *testing.T) {
		roundTrip(t, schemaOf(t, refChoiceSchema), rootName, refChoice(refChoiceK{K: []refK{"u v", "w"}}), `<r xmlns="urn:s"><k>u v</k><k>w</k></r>`, deepEqual[refChoice], refChoiceBranches)
	})
	t.Run("another target namespace and a complex type", func(t *testing.T) {
		schema, err := parser.Parse("a.xsd", parser.WithResolver(loader.Map(map[string]string{
			"a.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:b="urn:b" targetNamespace="urn:a">
			  <xs:import namespace="urn:b" schemaLocation="b.xsd"/>
			  <xs:element name="n"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType></xs:element>
			  <xs:element name="r"><xs:complexType><xs:sequence><xs:element ref="b:x"/><xs:element ref="n" xmlns="urn:a"/></xs:sequence></xs:complexType></xs:element>
			</xs:schema>`,
			"b.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:b">
			  <xs:element name="x" type="xs:string"/>
			</xs:schema>`,
		})))
		if err != nil {
			t.Fatal(err)
		}
		roundTrip(t, schema, xsd.QName{Space: "urn:a", Local: "r"}, refOuter{X: "v", N: refNested{A: "w"}},
			`<r xmlns="urn:a"><x xmlns="urn:b">v</x><n><a xmlns="">w</a></n></r>`, deepEqual[refOuter])
	})
	t.Run("a ref back to its own declaration", func(t *testing.T) {
		roundTrip(t, schemaOf(t, refSelfSchema), rootName, refSelf{A: "x", R: &refSelf{A: "y", R: &refSelf{A: "z"}}},
			`<r xmlns="urn:s"><a>x</a><r><a>y</a><r><a>z</a></r></r></r>`, deepEqual[refSelf])
	})
}

// The substitution-group types Append writes beside decode_test.go's: fc* are
// flatGroupSchema's R, its flattened choice's interface and branches, and the
// head's interface and branches, h as grpA and m as grpM; wcHolder is
// wildGroupSchema's R, beside abstractOneSchema's interface; rec* are
// recursiveGroupSchema's H and M, of one complex type that reaches h again,
// and its R, interface and branches; blk* are derivationGroupSchema's R,
// interface and branches, h and n as grpA; xn* are the cross-namespace
// schema's R, interface and branches; and memberless is the R of an abstract
// head whose group holds no other declaration, referenced once.
type (
	fcHolder  struct{ Choice []fcChoice }
	fcChoice  interface{ isFcChoice() }
	fcChoiceA struct{ A string }
	fcChoiceH struct{ H fcGroup }
	fcGroup   interface{ isFcGroup() }
	fcGroupH  struct{ H grpA }
	fcGroupM  struct{ M grpM }
	wcHolder  struct {
		H   abs1Group
		Any *anyElement
	}
	recH struct {
		V string
		H recGroup
	}
	recM       recH
	recHolder  struct{ H recGroup }
	recGroup   interface{ isRecGroup() }
	recGroupH  struct{ H recH }
	recGroupM  struct{ M recM }
	blkHolder  struct{ H blkGroup }
	blkGroup   interface{ isBlkGroup() }
	blkGroupH  struct{ H grpA }
	blkGroupN  struct{ N grpA }
	xnHolder   struct{ H xnGroup }
	xnGroup    interface{ isXnGroup() }
	xnGroupH   struct{ H grpA }
	xnGroupM   struct{ M grpA }
	memberless struct{ H aloneGroup }
)

func (fcChoiceA) isFcChoice() {}
func (fcChoiceH) isFcChoice() {}
func (fcGroupH) isFcGroup()   {}
func (fcGroupM) isFcGroup()   {}
func (recGroupH) isRecGroup() {}
func (recGroupM) isRecGroup() {}
func (blkGroupH) isBlkGroup() {}
func (blkGroupN) isBlkGroup() {}
func (xnGroupH) isXnGroup()   {}
func (xnGroupM) isXnGroup()   {}

// The substitution-group schemas Append writes beside decode_test.go's:
// flatGroupSchema's head h, with the member m, is an alternative of r's
// flattened choice beside a local a; wildGroupSchema's abstract h, with the
// member m, is referenced ahead of a skip wildcard, so that r's children are
// attributed by a Matcher (attributionCheck); recursiveGroupSchema's h and m,
// of the complex type ct, each optionally hold h again; and
// derivationGroupSchema's h blocks restriction, so that of its members m, of
// a restriction of xs:string, and n, of xs:string, only n is substitutable
// (cos-equiv-derived-ok-rec clause 2.3).
const (
	flatGroupSchema = `<xs:element name="h" type="xs:string"/><xs:element name="m" type="xs:string" substitutionGroup="s:h"/>` +
		`<xs:element name="r"><xs:complexType><xs:choice maxOccurs="unbounded"><xs:element name="a" type="xs:string"/><xs:element ref="s:h"/></xs:choice></xs:complexType></xs:element>`
	wildGroupSchema = `<xs:element name="h" type="xs:string" abstract="true"/><xs:element name="m" type="xs:string" substitutionGroup="s:h"/>` +
		`<xs:element name="r"><xs:complexType><xs:sequence><xs:element ref="s:h"/><xs:any processContents="skip" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>`
	recursiveGroupSchema = `<xs:complexType name="ct"><xs:sequence><xs:element name="v" type="xs:string"/><xs:element ref="s:h" minOccurs="0"/></xs:sequence></xs:complexType>` +
		`<xs:element name="h" type="s:ct"/><xs:element name="m" type="s:ct" substitutionGroup="s:h"/>` +
		`<xs:element name="r"><xs:complexType><xs:sequence><xs:element ref="s:h"/></xs:sequence></xs:complexType></xs:element>`
	derivationGroupSchema = `<xs:simpleType name="t"><xs:restriction base="xs:string"/></xs:simpleType>` +
		`<xs:element name="h" type="xs:string" block="restriction"/><xs:element name="m" type="s:t" substitutionGroup="s:h"/><xs:element name="n" type="xs:string" substitutionGroup="s:h"/>` +
		`<xs:element name="r"><xs:complexType><xs:sequence><xs:element ref="s:h"/></xs:sequence></xs:complexType></xs:element>`
)

// TestAppendSubstitutionGroup pins "# What encodes"' <element ref> particle
// whose field codegen gives the sealed interface over its head's substitution
// group: the branch a value of it holds is written as an element of its
// member's own expanded name (§3.9.4.3 cvc-accept clause 2.3.2, §3.3.4.3
// cvc-elt clause 1), which validate accepts and Decode returns equal: a head
// of {abstract} false, by its own branch and its member's; an abstract head
// with two members; the field repeated and optional, by heads and members, and
// the optional absent; a member in another target namespace than its head's;
// a member of complex type whose type reaches the head again; a head blocking
// one member's derivation, by the other; a member whose own block is #all; a
// head whose final is #all; a head ref among a flattened choice's
// alternatives, its slice holding the head's branch and a member's in document
// order; a member in content a Matcher attributes, taken at its head's
// particle; and, under {min occurs} 0, an abstract head whose group holds no
// other declaration, its interface nil, writing nothing.
func TestAppendSubstitutionGroup(t *testing.T) {
	t.Run("a head of {abstract} false, the head's branch", func(t *testing.T) {
		roundTrip(t, schemaOf(t, groupSchema), rootName, grpHolder{A: grpAGroupA{A: "x"}}, `<r xmlns="urn:s"><a>x</a></r>`, deepEqual[grpHolder], grpBranches)
	})
	t.Run("a head of {abstract} false, the member's branch", func(t *testing.T) {
		roundTrip(t, schemaOf(t, groupSchema), rootName, grpHolder{A: grpAGroupM{M: "y"}}, `<r xmlns="urn:s"><m>y</m></r>`, deepEqual[grpHolder], grpBranches)
	})
	t.Run("an abstract head with two members", func(t *testing.T) {
		roundTrip(t, schemaOf(t, abstractTwoSchema), rootName, abs2Holder{H: abs2GroupM{M: "x"}}, `<r xmlns="urn:s"><m>x</m></r>`, deepEqual[abs2Holder], abs2Branches)
		roundTrip(t, schemaOf(t, abstractTwoSchema), rootName, abs2Holder{H: abs2GroupN{N: "1 2"}}, `<r xmlns="urn:s"><n>1 2</n></r>`, deepEqual[abs2Holder], abs2Branches)
	})
	t.Run("repeated and optional, by heads and members", func(t *testing.T) {
		roundTrip(t, schemaOf(t, occursGroupSchema), rootName,
			occHolder{A: occAGroupX{X: "1"}, B: occBGroupB{B: "2"}, C: []occCGroup{occCGroupC{C: "3"}, occCGroupZ{Z: "4"}, occCGroupC{C: "5"}}},
			`<r xmlns="urn:s"><x>1</x><b>2</b><c>3</c><z>4</z><c>5</c></r>`, deepEqual[occHolder], occBranches...)
	})
	t.Run("optional and repeated absent", func(t *testing.T) {
		roundTrip(t, schemaOf(t, occursGroupSchema), rootName, occHolder{A: occAGroupA{A: "1"}}, `<r xmlns="urn:s"><a>1</a></r>`, deepEqual[occHolder], occBranches...)
	})
	t.Run("a member in another target namespace", func(t *testing.T) {
		schema, err := parser.Parse("a.xsd", parser.WithResolver(loader.Map(map[string]string{
			"a.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:b="urn:b" targetNamespace="urn:a">
			  <xs:import namespace="urn:b" schemaLocation="b.xsd"/>
			  <xs:element name="m" type="xs:string" substitutionGroup="b:h"/>
			  <xs:element name="r"><xs:complexType><xs:sequence><xs:element ref="b:h"/></xs:sequence></xs:complexType></xs:element>
			</xs:schema>`,
			"b.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:b">
			  <xs:element name="h" type="xs:string"/>
			</xs:schema>`,
		})))
		if err != nil {
			t.Fatal(err)
		}
		r, branches := xsd.QName{Space: "urn:a", Local: "r"}, WithBranches[xnGroup](xnGroupH{}, xnGroupM{})
		roundTrip(t, schema, r, xnHolder{H: xnGroupM{M: "x"}}, `<r xmlns="urn:a"><m>x</m></r>`, deepEqual[xnHolder], branches)
		roundTrip(t, schema, r, xnHolder{H: xnGroupH{H: "y"}}, `<r xmlns="urn:a"><h xmlns="urn:b">y</h></r>`, deepEqual[xnHolder], branches)
	})
	t.Run("a member whose type reaches the head again", func(t *testing.T) {
		v := recHolder{H: recGroupM{M: recM{V: "1", H: recGroupH{H: recH{V: "2", H: recGroupM{M: recM{V: "3"}}}}}}}
		roundTrip(t, schemaOf(t, recursiveGroupSchema), rootName, v, `<r xmlns="urn:s"><m><v>1</v><h><v>2</v><m><v>3</v></m></h></m></r>`,
			deepEqual[recHolder], WithBranches[recGroup](recGroupH{}, recGroupM{}))
	})
	t.Run("a head blocking one member's derivation, by the other", func(t *testing.T) {
		roundTrip(t, schemaOf(t, derivationGroupSchema), rootName, blkHolder{H: blkGroupN{N: "x"}}, `<r xmlns="urn:s"><n>x</n></r>`,
			deepEqual[blkHolder], WithBranches[blkGroup](blkGroupH{}, blkGroupN{}))
	})
	t.Run("a member whose block is #all", func(t *testing.T) {
		schema := schemaOf(t, strings.Replace(groupSchema, `substitutionGroup="s:a"`, `block="#all" substitutionGroup="s:a"`, 1))
		roundTrip(t, schema, rootName, grpHolder{A: grpAGroupM{M: "y"}}, `<r xmlns="urn:s"><m>y</m></r>`, deepEqual[grpHolder], grpBranches)
	})
	t.Run("a head whose final is #all", func(t *testing.T) {
		schema := schemaOf(t, strings.Replace(groupSchema, `<xs:element name="a" type="xs:string"/>`, `<xs:element name="a" type="xs:string" final="#all"/>`, 1))
		roundTrip(t, schema, rootName, grpHolder{A: grpAGroupM{M: "y"}}, `<r xmlns="urn:s"><m>y</m></r>`, deepEqual[grpHolder], grpBranches)
	})
	t.Run("a flattened choice's head ref alternative", func(t *testing.T) {
		v := fcHolder{Choice: []fcChoice{fcChoiceH{H: fcGroupM{M: "1"}}, fcChoiceA{A: "2"}, fcChoiceH{H: fcGroupH{H: "3"}}}}
		roundTrip(t, schemaOf(t, flatGroupSchema), rootName, v, `<r xmlns="urn:s"><m>1</m><a>2</a><h>3</h></r>`, deepEqual[fcHolder],
			WithBranches[fcChoice](fcChoiceA{}, fcChoiceH{}), WithBranches[fcGroup](fcGroupH{}, fcGroupM{}))
	})
	t.Run("a member taken at its head's particle by a Matcher", func(t *testing.T) {
		roundTrip(t, schemaOf(t, wildGroupSchema), rootName, wcHolder{H: abs1GroupM{M: "x"}, Any: &anyElement{Space: "urn:o", Local: "w"}},
			`<r xmlns="urn:s"><m>x</m><w xmlns="urn:o"/></r>`, deepEqual[wcHolder], abs1Branches)
	})
	for _, s := range aloneSchemas {
		t.Run("the interface alone, nil, optional, "+s.name, func(t *testing.T) {
			roundTrip(t, schemaOf(t, s.decls), rootName, aloneHolder{A: "x"}, `<r xmlns="urn:s"><a>x</a></r>`, deepEqual[aloneHolder])
		})
	}
}

// TestAppendSubstitutionGroupInvalid pins item 6 of Append's doc for the
// sealed interface over a substitution group: nil where nil does not mean
// absent, as a required field of a head of {abstract} false, as an item of
// its slice, and as the required field of an abstract head whose group holds
// no other declaration, which has no branch; and a value of a dynamic type
// the WithBranches set places at no branch, a pointer to a branch. Each opens
// with the Go path of the field and writes nothing.
func TestAppendSubstitutionGroupInvalid(t *testing.T) {
	const nilled = " is nil, the sealed interface over the substitution group of element "
	rows := []struct {
		name   string
		append func(t *testing.T) ([]byte, error)
		prefix string
	}{
		{"a required field nil", func(t *testing.T) ([]byte, error) {
			return appendChecked(t, nil, schemaOf(t, groupSchema), strict.New(), rootName, grpHolder{}, grpBranches)
		}, "codec: Append: grpHolder.A" + nilled + "{urn:s}a declared at "},
		{"an item of the slice nil", func(t *testing.T) ([]byte, error) {
			v := occHolder{A: occAGroupA{A: "1"}, C: []occCGroup{occCGroupC{C: "3"}, nil}}
			return appendChecked(t, nil, schemaOf(t, occursGroupSchema), strict.New(), rootName, v, occBranches...)
		}, "codec: Append: occHolder.C[1]" + nilled + "{urn:s}c declared at "},
		{"an abstract head with no branch, required", func(t *testing.T) ([]byte, error) {
			schema := schemaOf(t, `<xs:element name="h" type="xs:string" abstract="true"/>`+sequenceOf(`<xs:element ref="s:h"/>`))
			return appendChecked(t, nil, schema, strict.New(), rootName, memberless{})
		}, "codec: Append: memberless.H" + nilled + "{urn:s}h declared at "},
		{"a pointer to a branch", func(t *testing.T) ([]byte, error) {
			return appendChecked(t, nil, schemaOf(t, groupSchema), strict.New(), rootName, grpHolder{A: &grpAGroupM{M: "y"}}, grpBranches)
		}, "codec: Append: grpHolder.A holds a *codec.grpAGroupM, which the WithBranches set of codec.grpAGroup places at no branch of the substitution group of element {urn:s}a declared at "},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			got, err := row.append(t)
			invalidChoice(t, err, row.prefix)
			if got != nil {
				t.Errorf("Append wrote %q", got)
			}
		})
	}
}

// TestAppendDst pins that Append appends to dst, and on error returns dst
// unextended.
func TestAppendDst(t *testing.T) {
	schema := schemaOf(t, encodeSchema)
	dst := []byte("pre")
	got, err := appendChecked(t, dst, schema, strict.New(), sname("str"), strRoot("v"))
	if err != nil || string(got) != `pre<str xmlns="urn:s">v</str>` {
		t.Errorf("Append(pre) = %q, %v", got, err)
	}
	got, err = appendChecked(t, dst, schema, strict.New(), sname("p"), dblRoot(1))
	if err == nil || string(got) != "pre" || len(got) != 3 {
		t.Errorf("a declined Append(pre) = %q, %v; want pre and an error", got, err)
	}
}

// plainError fails t unless err, Append's or CheckAppend's, is a plain
// error of Append's doc: no *DeclineError, wrapping none of
// ErrTypeMismatch, ErrInvalidValue and errors.ErrUnsupported, whose text
// holds each of want.
func plainError(t *testing.T, err error, want ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("no error returned, want a plain error")
	}
	var d *DeclineError
	if errors.As(err, &d) || errors.Is(err, errors.ErrUnsupported) || errors.Is(err, ErrTypeMismatch) || errors.Is(err, ErrInvalidValue) {
		t.Fatalf("returned %v, want a plain error", err)
	}
	holds(t, err, want...)
}

// invalidValue fails t unless err is item 6's error of Append's doc: one
// errors.Is reports as ErrInvalidValue, no *DeclineError, wrapping neither
// ErrTypeMismatch nor errors.ErrUnsupported, whose text holds each of want
// and not ErrInvalidValue's own.
func invalidValue(t *testing.T, err error, want ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("no error returned, want ErrInvalidValue")
	}
	var d *DeclineError
	if !errors.Is(err, ErrInvalidValue) || errors.As(err, &d) || errors.Is(err, errors.ErrUnsupported) || errors.Is(err, ErrTypeMismatch) {
		t.Fatalf("returned %v, want an error errors.Is reports as ErrInvalidValue alone", err)
	}
	if strings.Contains(err.Error(), ErrInvalidValue.Error()) {
		t.Errorf("error %q holds ErrInvalidValue's own text", err)
	}
	holds(t, err, want...)
}

// noRule fails t if err carries an *xsderr.Error, as item 6's error of
// Append's doc does in none but its last case.
func noRule(t *testing.T, err error) {
	t.Helper()
	if rule, ok := xsderr.RuleOf(err); ok {
		t.Errorf("rule %q, want none", rule)
	}
}

// holds fails t unless err's text holds each of want.
func holds(t *testing.T, err error, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(err.Error(), w) {
			t.Errorf("error %q does not hold %q", err, w)
		}
	}
}

// TestAppendPlainErrors pins items 1 to 3 and 5 of Append's doc.
func TestAppendPlainErrors(t *testing.T) {
	schema := schemaOf(t, shapesSchema+`<xs:element name="abs" type="xs:string" abstract="true"/>`)
	_, err := Append[shapes](nil, nil, strict.New(), rootName, shapes{})
	plainError(t, err, "nil *xsd.Schema")
	_, err = Append[shapes](nil, schema, nil, rootName, shapes{})
	plainError(t, err, "nil value.Backend")
	_, err = appendChecked[shapes](t, nil, schema, strict.New(), sname("none"), shapes{})
	plainError(t, err, "{urn:s}none")
	_, err = appendChecked(t, nil, schema, strict.New(), sname("abs"), strRoot("x"))
	plainError(t, err, "{abstract} true")
	_, err = appendChecked(t, nil, schema, strict.New(), rootName, &shapes{})
	if want := ErrTypeMismatch.Error() + ": *codec.shapes is a pointer to codec.shapes, which matches; use codec.shapes in its place; the document element's declaration is element {urn:s}r declared at s.xsd:2:3"; !errors.Is(err, ErrTypeMismatch) || err.Error() != want {
		t.Errorf("Append[*shapes] = %v, want ErrTypeMismatch %q", err, want)
	}
	_, err = appendChecked(t, nil, schema, strict.New(), rootName, struct{ One string }{})
	if !errors.Is(err, ErrTypeMismatch) {
		t.Errorf("Append[struct{One string}] = %v, want ErrTypeMismatch", err)
	}
}

// TestCheckAppend pins CheckAppend's answers by Append's items: a plain
// error for a nil schema; item 4's decline for an xs:ENTITY declaration and T
// string, which CheckDecode accepts; nil for an xs:precisionDecimal
// declaration and T its datatype struct under WithPrecisionDecimalArms, and
// item 5's ErrTypeMismatch with none; nil for an xs:float declaration and T
// float32; nil for README's order.xsd and order.Order, whose zero value
// Append refuses by item 6, and item 5's ErrTypeMismatch for *order.Order;
// item 2's and item 3's plain errors; and item 3 before item 4 on an
// abstract xs:ENTITY declaration. appendChecked holds each answer to Append's.
func TestCheckAppend(t *testing.T) {
	schema := schemaOf(t, `<xs:element name="i" type="xs:ENTITY"/><xs:element name="abs" type="xs:string" abstract="true"/>`+
		`<xs:element name="absb" type="xs:ENTITY" abstract="true"/><xs:element name="f" type="xs:float"/><xs:element name="pd" type="xs:precisionDecimal"/>`)
	plainError(t, CheckAppend[string](nil, sname("i")), "codec: CheckAppend: nil *xsd.Schema")

	if err := CheckDecode[string](schema, sname("i")); err != nil {
		t.Fatalf("CheckDecode[string] = %v, want nil", err)
	}
	err := CheckAppend[string](schema, sname("i"))
	_ = asDecline(t, err, "codec does not encode a simple type derived from xs:ENTITY: ")
	_, _ = appendChecked(t, nil, schema, strict.New(), sname("i"), "x")
	if err := CheckAppend[tPrecisionDecimal](schema, sname("pd"), pdArms); err != nil {
		t.Errorf("CheckAppend[tPrecisionDecimal] = %v, want nil", err)
	}
	if err := CheckAppend[tPrecisionDecimal](schema, sname("pd")); !errors.Is(err, ErrTypeMismatch) {
		t.Errorf("CheckAppend[tPrecisionDecimal] with no WithPrecisionDecimalArms = %v, want ErrTypeMismatch", err)
	}
	_, _ = appendChecked(t, nil, schema, strict.New(), sname("pd"), tPrecisionDecimal{})
	if err := CheckAppend[float32](schema, sname("f")); err != nil {
		t.Errorf("CheckAppend[float32] = %v, want nil", err)
	}

	plainError(t, CheckAppend[string](schema, sname("none")), "codec: no top-level element declaration has the expanded name", "{urn:s}none")
	_, _ = appendChecked(t, nil, schema, strict.New(), sname("none"), "x")
	plainError(t, CheckAppend[string](schema, sname("abs")), "{abstract} true")
	_, _ = appendChecked(t, nil, schema, strict.New(), sname("abs"), "x")
	plainError(t, CheckAppend[string](schema, sname("absb")), "{abstract} true")
	_, _ = appendChecked(t, nil, schema, strict.New(), sname("absb"), "x")

	readme := filepath.Join("..", "testdata", "readme")
	orderSchema, err := parser.Parse("order.xsd", parser.WithResolver(loader.Dir(readme)))
	if err != nil {
		t.Fatal(err)
	}
	name := xsd.QName{Space: "urn:o", Local: "order"}
	if err := CheckAppend[order.Order](orderSchema, name); err != nil {
		t.Errorf("CheckAppend[order.Order] = %v, want nil", err)
	}
	_, _ = appendChecked(t, nil, orderSchema, strict.New(), name, order.Order{})
	err = CheckAppend[*order.Order](orderSchema, name)
	if !errors.Is(err, ErrTypeMismatch) {
		t.Errorf("CheckAppend[*order.Order] = %v, want ErrTypeMismatch", err)
	}
	_, _ = appendChecked(t, nil, orderSchema, strict.New(), name, &order.Order{})
	_, _ = appendChecked(t, nil, orderSchema, strict.New(), name, (*order.Order)(nil))
}

// TestAppendInvalidValues pins item 6 of Append's doc: an error errors.Is
// reports as ErrInvalidValue naming the Go path, for each value no valid
// document holds, a backend Parse's verdict carrying no xsderr rule among
// them, and for the first in document order, an element's attributes before
// its content, though an item 7 value comes earlier.
func TestAppendInvalidValues(t *testing.T) {
	schema := schemaOf(t, shapesSchema+encodeSchema)
	tail := func(s shapes) shapes {
		if s.Tok == "" {
			s.Tok = "t"
		}
		return s
	}
	str := func(s string) *string { return &s }
	for _, row := range []struct {
		name string
		v    shapes
		want []string
	}{
		{"a token element not collapsed", shapes{Tok: "  a  b"}, []string{"shapes.Tok", `"  a  b"`, `whiteSpace-normalized form "a b"`}},
		{"a normalizedString attribute holding TAB", tail(shapes{Opt2: str("a\tb")}), []string{"shapes.Opt2", `form "a b"`}},
		{"a normalizedString attribute holding LF", tail(shapes{Opt2: str("a\nb")}), []string{"shapes.Opt2"}},
		{"a normalizedString attribute holding CR", tail(shapes{Opt2: str("a\rb")}), []string{"shapes.Opt2"}},
		{"U+0000", tail(shapes{One: "a\x00b"}), []string{"shapes.One", "byte offset 1", "XML 1.0 Char"}},
		{"U+FFFE", tail(shapes{One: "a￾"}), []string{"shapes.One", "byte offset 1"}},
		{"bytes that are not UTF-8", tail(shapes{Rep: []string{"ok", "\xffb"}}), []string{"shapes.Rep[1]", "byte offset 0"}},
		{"an attribute before an earlier field's element", shapes{One: "\x00", Tok: "  t", Req: "\x01"}, []string{"shapes.Req"}},
	} {
		t.Run(row.name, func(t *testing.T) {
			_, err := appendChecked(t, nil, schema, strict.New(), rootName, row.v)
			invalidValue(t, err, row.want...)
			noRule(t, err)
		})
	}
	t.Run("a token attribute not collapsed", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("ta"), tokenAttr{T: "  a  b"})
		invalidValue(t, err, "tokenAttr.T", `form "a b"`)
		noRule(t, err)
	})
	t.Run("a repeated field outside its occurrence", func(t *testing.T) {
		readme := filepath.Join("..", "testdata", "readme")
		orderSchema, err := parser.Parse("order.xsd", parser.WithResolver(loader.Dir(readme)))
		if err != nil {
			t.Fatal(err)
		}
		_, err = appendChecked(t, nil, orderSchema, strict.New(), xsd.QName{Space: "urn:o", Local: "order"}, order.Order{})
		invalidValue(t, err, "Order.Item holds 0 items", "1..unbounded")
		noRule(t, err)
	})
	t.Run("an xs:double a bound rejects", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("np"), dblRoot(1))
		invalidValue(t, err, "dblRoot", "{urn:s}nonPositive")
		if rule, _ := xsderr.RuleOf(err); rule != "cvc-maxInclusive-valid" {
			t.Errorf("rule %q, want cvc-maxInclusive-valid", rule)
		}
	})
	for _, f := range []float64{math.NaN(), math.Inf(-1)} {
		t.Run("an xs:double "+doubleLexical(f, 64)+" a pattern rejects", func(t *testing.T) {
			_, err := appendChecked(t, nil, schema, strict.New(), sname("p"), dblRoot(f))
			invalidValue(t, err, "dblRoot", "{urn:s}one")
			if !strings.HasPrefix(err.Error(), "codec: Append: dblRoot holds ") {
				t.Errorf("error %q does not open with the path dblRoot", err)
			}
			if rule, _ := xsderr.RuleOf(err); rule != "cvc-pattern-valid" {
				t.Errorf("rule %q, want cvc-pattern-valid", rule)
			}
		})
	}
	t.Run("an xs:string a pattern rejects", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("oa"), strRoot("b"))
		invalidValue(t, err, "strRoot", "{urn:s}onlyA")
		if rule, _ := xsderr.RuleOf(err); rule != "cvc-pattern-valid" {
			t.Errorf("rule %q, want cvc-pattern-valid", rule)
		}
	})
	t.Run("an xs:string a backend's Parse rejects", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, stringRejectingParse, sname("str"), strRoot("v"))
		invalidValue(t, err, "strRoot", "rejected")
		if rule, ok := xsderr.RuleOf(err); ok {
			t.Errorf("rule %q, want none", rule)
		}
	})
	t.Run("item 6 after an earlier item 7", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("order2"), order2{P: 1, Np: 1})
		invalidValue(t, err, "order2.Np")
	})
	t.Run("an attribute's item 6 before its element's", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("order2"), order2{P: 1, Np: 1, Oa: str("b")})
		invalidValue(t, err, "order2.Oa")
	})
}

// The backends of TestAppendNoCanonical, TestAppendBackendFaults and
// TestAppendInvalidValues, each strict's with one mapping replaced.
var (
	doubleNilCanonical = overDatatype("double", func(base value.Mapping) value.Mapping {
		return value.Mapping{Parse: base.Parse}
	})
	doubleFailingCanonical = overDatatype("double", func(base value.Mapping) value.Mapping {
		return value.Mapping{Parse: base.Parse, Canonical: func(value.Value) (string, error) { return "", errors.New("no canonical form") }}
	})
	doubleOtherCanonical = overDatatype("double", func(base value.Mapping) value.Mapping {
		return value.Mapping{Parse: base.Parse, Canonical: func(value.Value) (string, error) { return "2.0E0", nil }}
	})
	stringNilCanonical = overDatatype("string", func(base value.Mapping) value.Mapping {
		return value.Mapping{Parse: base.Parse}
	})
	stringFailingCanonical = overDatatype("string", func(base value.Mapping) value.Mapping {
		return value.Mapping{Parse: base.Parse, Canonical: func(value.Value) (string, error) { return "", errors.New("no canonical form") }}
	})
	doubleRejectingParse = overDatatype("double", func(base value.Mapping) value.Mapping {
		return value.Mapping{Parse: func(string, value.Context) (value.Value, error) { return nil, errors.New("rejected") }, Canonical: base.Canonical}
	})
	stringRejectingParse = overDatatype("string", func(base value.Mapping) value.Mapping {
		return value.Mapping{Parse: func(string, value.Context) (value.Value, error) { return nil, errors.New("rejected") }, Canonical: base.Canonical}
	})
	doubleRejectingCanonical = overDatatype("double", func(base value.Mapping) value.Mapping {
		return value.Mapping{Parse: func(s string, ctx value.Context) (value.Value, error) {
			if s == "1.0E0" {
				return nil, errors.New("rejects canonical")
			}
			return base.Parse(s, ctx)
		}, Canonical: base.Canonical}
	})
	doubleSpacedCanonical = overDatatype("double", func(base value.Mapping) value.Mapping {
		return value.Mapping{Parse: func(s string, ctx value.Context) (value.Value, error) {
			if s == "1.0E0" {
				return nil, errors.New("rejects collapsed canonical")
			}
			return base.Parse(strings.TrimLeft(s, " "), ctx)
		}, Canonical: func(v value.Value) (string, error) {
			c, err := base.Canonical(v)
			return " " + c, err
		}}
	})
)

// TestAppendNoCanonical pins item 7 of Append's doc: a value with no
// canonical representation declines, through a pattern facet rejecting an
// xs:double's canonical representation, a bound rejecting it too, a nil or
// failing Canonical, or a Canonical whose form does not map back; the
// decline names the Go path and the value, and its Construct is one.
func TestAppendNoCanonical(t *testing.T) {
	schema := schemaOf(t, encodeSchema)
	var first Construct
	for _, row := range []struct {
		name    string
		backend value.Backend
		elem    string
		v       any
		want    []string
	}{
		{"a pattern rejecting the canonical form", strict.New(), "p", dblRoot(1), []string{"dblRoot", "holding 1", `"1.0E0"`, "{urn:s}one"}},
		{"a pattern and a bound rejecting it", strict.New(), "pb", dblRoot(1), []string{"dblRoot", "{urn:s}oneBelowZero"}},
		{"a pattern rejecting INF, which +INF also spells", strict.New(), "p", dblRoot(math.Inf(1)), []string{"dblRoot", "holding +Inf", `"INF"`}},
		{"a nil Canonical", doubleNilCanonical, "d", dblRoot(1), []string{"dblRoot", "no canonical mapping"}},
		{"a failing Canonical", doubleFailingCanonical, "d", dblRoot(1), []string{"dblRoot", "no canonical form"}},
		{"a Canonical not mapping back", doubleOtherCanonical, "d", dblRoot(1), []string{`"2.0E0" does not map back`}},
		{"a string's nil Canonical", stringNilCanonical, "str", strRoot("v"), []string{"strRoot", `holding "v"`}},
		{"a string's failing Canonical", stringFailingCanonical, "str", strRoot("v"), []string{"strRoot", "no canonical form"}},
	} {
		t.Run(row.name, func(t *testing.T) {
			var err error
			switch v := row.v.(type) {
			case dblRoot:
				_, err = appendChecked(t, nil, schema, row.backend, sname(row.elem), v)
			case strRoot:
				_, err = appendChecked(t, nil, schema, row.backend, sname(row.elem), v)
			}
			d := asDecline(t, err, "codec does not encode a value with no canonical representation: ")
			for _, w := range row.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("%q does not hold %q", err, w)
				}
			}
			if first == (Construct{}) {
				first = d.Construct()
			}
			if d.Construct() != first {
				t.Errorf("Construct %v, want %v", d.Construct(), first)
			}
		})
	}
}

// TestAppendBackendFaults pins the backend faults of Append's doc: a plain
// error naming the value's Go path, the xs:string one plain though it wraps
// value's *xsderr.Error.
func TestAppendBackendFaults(t *testing.T) {
	schema := schemaOf(t, encodeSchema)
	_, err := appendChecked(t, nil, schema, doubleRejectingParse, sname("d"), dblRoot(1))
	plainError(t, err, "dblRoot", `doubleRep form "1E+00"`, "rejected")
	_, err = appendChecked(t, nil, schema, doubleRejectingCanonical, sname("d"), dblRoot(1))
	plainError(t, err, "dblRoot", "canonical representation", `"1.0E0"`)
	_, err = appendChecked(t, nil, schema, doubleSpacedCanonical, sname("d"), dblRoot(1))
	plainError(t, err, `codec: Append: dblRoot holds 1, of simple type xs:double, whose canonical representation " 1.0E0" the backend's Parse rejects: rejects collapsed canonical`)
	_, err = appendChecked(t, nil, schema, noMappings{}, sname("d"), dblRoot(1))
	plainError(t, err, "dblRoot", "no mapping governing")
	_, err = appendChecked(t, nil, schema, noMappings{}, sname("str"), strRoot("v"))
	plainError(t, err, "strRoot", "the backend cannot assess")
	if rule, _ := xsderr.RuleOf(err); rule != "cvc-datatype-valid" {
		t.Errorf("rule %q, want cvc-datatype-valid", rule)
	}
}

// asDecline fails t unless err is an Append or CheckAppend *DeclineError that wraps
// errors.ErrUnsupported and not ErrInvalidValue, charged to no Loc, whose
// text opens with prefix, and returns it.
func asDecline(t *testing.T, err error, prefix string) *DeclineError {
	t.Helper()
	var d *DeclineError
	if !errors.As(err, &d) || !errors.Is(err, errors.ErrUnsupported) || errors.Is(err, ErrInvalidValue) {
		t.Fatalf("returned %v, want a *DeclineError wrapping errors.ErrUnsupported and not ErrInvalidValue", err)
	}
	if !strings.HasPrefix(err.Error(), prefix) || !strings.HasSuffix(err.Error(), "; an encoder refusal, not a schema or value fault") {
		t.Errorf("decline %q does not open with %q and close as an encoder refusal", err, prefix)
	}
	if _, ok := d.Loc(); ok {
		t.Errorf("decline %q is charged to a Loc", err)
	}
	return d
}

// TestAppendVacuousParticle pins the decline of a particle with {max
// occurs} 0, which no schema document yields (§3.3.2, §3.10.2, and
// §3.4.2.3.3 dcl.ctd.ctcc.common clause 2.1.4 for a content {particle}), so
// each schema is built from a parsed one with type v's particle rebuilt
// through xsd.NewParticle with Occurs{0,0}: the content {particle}, an
// element particle, and a strict wildcard particle, which declines so before
// it declines as a strict wildcard.
func TestAppendVacuousParticle(t *testing.T) {
	none, err := xsd.NewOccurs(xsderr.Loc{}, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		name, particle string
		content        bool
	}{
		{"content particle", `<xs:element name="a" type="xs:string"/>`, true},
		{"element particle", `<xs:element name="a" type="xs:string"/>`, false},
		{"wildcard particle", `<xs:any/>`, false},
	} {
		t.Run(row.name, func(t *testing.T) {
			parsed := schemaOf(t, `<xs:complexType name="v"><xs:sequence>`+row.particle+`</xs:sequence></xs:complexType><xs:element name="r" type="s:v"/>`)
			v := xsd.QName{Space: "urn:s", Local: "v"}
			declared, ok := parsed.Type(v)
			if !ok {
				t.Fatalf("the parsed schema declares no type %s", v)
			}
			content := declared.(xsd.ComplexType).ContentType().(xsd.ElementContent)
			group := content.Particle.Term().(xsd.ResolvedTerm).Term.(xsd.ModelGroup)
			occurs := content.Particle.Occurs()
			if row.content {
				occurs = none
			}
			if !row.content {
				vacuous, err := xsd.NewParticle(xsderr.Loc{}, none, group.Particles()[0].Term())
				if err != nil {
					t.Fatal(err)
				}
				group, err = xsd.NewModelGroup(xsderr.Loc{}, xsd.CompositorSequence, []xsd.Particle{vacuous})
				if err != nil {
					t.Fatal(err)
				}
			}
			particle, err := xsd.NewParticle(xsderr.Loc{}, occurs, xsd.ResolvedTerm{Term: group})
			if err != nil {
				t.Fatal(err)
			}
			ct, err := xsd.NewComplexType(declared.Loc(), v, xsd.QName{Space: xsd.XMLSchemaNS, Local: "anyType"}, nil, xsd.DerivationRestriction, false,
				nil, nil, nil, xsd.ElementContent{Particle: particle}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			_, err = appendChecked(t, nil, replacedSchema(t, parsed, ct), strict.New(), rootName, struct{}{})
			_ = asDecline(t, err, "codec does not encode a particle with {max occurs} 0: ")
		})
	}
}

// TestAppendAbstractParticle pins declEncodes' {abstract} check (cvc-elt
// clause 2) at a particle: the parser rejects abstract on a local <element>,
// so type v's one particle, a of xs:string, is rebuilt through
// xsd.NewElementDeclaration with {abstract} true, and Append and CheckAppend
// decline it naming a, where without the check they would write <a>.
func TestAppendAbstractParticle(t *testing.T) {
	parsed := schemaOf(t, `<xs:complexType name="v"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType><xs:element name="r" type="s:v"/>`)
	v := xsd.QName{Space: "urn:s", Local: "v"}
	declared, ok := parsed.Type(v)
	if !ok {
		t.Fatalf("the parsed schema declares no type %s", v)
	}
	scope, err := xsd.NewLocalScope(xsderr.Loc{}, xsd.ComplexTypeScopeParent{Name: v})
	if err != nil {
		t.Fatal(err)
	}
	a, err := xsd.NewElementDeclaration(xsderr.Loc{}, sname("a"), xsd.TypeDefinitionRef{Name: xsd.QName{Space: xsd.XMLSchemaNS, Local: "string"}}, nil, scope, nil, false, nil, nil, nil, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	p, err := xsd.NewParticle(xsderr.Loc{}, goshape.ExactlyOnce(), xsd.ResolvedTerm{Term: a})
	if err != nil {
		t.Fatal(err)
	}
	group, err := xsd.NewModelGroup(xsderr.Loc{}, xsd.CompositorSequence, []xsd.Particle{p})
	if err != nil {
		t.Fatal(err)
	}
	particle, err := xsd.NewParticle(xsderr.Loc{}, goshape.ExactlyOnce(), xsd.ResolvedTerm{Term: group})
	if err != nil {
		t.Fatal(err)
	}
	ct, err := xsd.NewComplexType(declared.Loc(), v, xsd.QName{Space: xsd.XMLSchemaNS, Local: "anyType"}, nil, xsd.DerivationRestriction, false,
		nil, nil, nil, xsd.ElementContent{Particle: particle}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = appendChecked(t, nil, replacedSchema(t, parsed, ct), strict.New(), rootName, struct{ A string }{A: "x"})
	_ = asDecline(t, err, "codec does not encode an element declaration with {abstract} true: element {urn:s}a declared at ")
}

// encodePrimitiveName is constructEncodePrimitive's name, spelled out where
// its code generates it from encodedPrimitives, so that a change to the list
// fails here.
const encodePrimitiveName = "an atomic simple type whose primitive is other than xs:string, xs:boolean, xs:decimal, xs:float, xs:double, xs:duration, xs:dateTime, xs:time, xs:date, xs:gYearMonth, xs:gYear, xs:gMonthDay, xs:gDay, xs:gMonth, xs:hexBinary, xs:base64Binary, xs:anyURI, xs:QName, xs:NOTATION or xs:precisionDecimal"

// identityName is constructIdentityConstraint's name.
const identityName = "an identity constraint whose {selector} or {fields}, or whose {referenced key}'s, lie outside the path subset icpath compiles (Structures §3.11.6.2 ·selector subset·, §3.11.6.3 ·field subset·)"

// absentAffiliationName is constructAbsentAffiliation's name.
const absentAffiliationName = "an element declaration with an ·absent· member of its {substitution group affiliations}"

// entityTypeName is constructEntityType's name.
const entityTypeName = "a simple type derived from xs:ENTITY"

// TestAppendDeclines pins item 4 of Append's doc for each construct of the
// closed set it declines: each row's two declarations, r and q, decline with
// equal Constructs, named by the opening of the decline's text, and rows of
// different constructs with different ones; no bytes are written. A row
// subjectOf names also opens its subject so after the construct, which is how
// the two nested particles inside a nested occurrence field's item fail where
// Decode's in-item decline leaks through in Append's place. A row's
// value is struct{}{}, or the one appendOf passes for it, and no row's value
// reaches item 5: Append finds item 4 before it binds T (encodeStatic). A row
// whose declarations reference s:money is read through mintedSchema, its
// primitive the one no primitive builtin/strict defines.
func TestAppendDeclines(t *testing.T) {
	sequence := func(body string) string {
		return `<xs:element name="%[1]s"><xs:complexType><xs:sequence>` + body + `</xs:sequence></xs:complexType></xs:element>`
	}
	complexOf := func(body string) string {
		return `<xs:element name="%[1]s"><xs:complexType>` + body + `</xs:complexType></xs:element>`
	}
	simpleOf := func(body string) string {
		return `<xs:element name="%[1]s"><xs:simpleType>` + body + `</xs:simpleType></xs:element>`
	}
	const a = `<xs:element name="a" type="xs:string"/>`
	// abstractMember is codegen's struct for the one member, d, of r's
	// abstract {type definition}, b.
	type abstractMember struct{ A, C, X string }
	const abstractType = `<xs:complexType name="b%[1]s" abstract="true"><xs:sequence>` + a + `</xs:sequence></xs:complexType>` +
		`<xs:complexType name="d%[1]s"><xs:complexContent><xs:extension base="s:b%[1]s"><xs:sequence><xs:element name="c" type="xs:string"/></xs:sequence>` +
		`<xs:attribute name="x" type="xs:string" use="required"/></xs:extension></xs:complexContent></xs:complexType>` +
		`<xs:element name="%[1]s" type="s:b%[1]s"/>`
	const asserted = `<xs:sequence>` + a + `</xs:sequence><xs:assert test="false()"/>`
	rows := []struct {
		name      string
		construct string
		decl      string
	}{
		{"nillable", "an element declaration with {nillable} true", `<xs:element name="%[1]s" type="xs:string" nillable="true"/>`},
		{"nillable particle", "an element declaration with {nillable} true", sequence(`<xs:element name="a" type="xs:string" nillable="true"/>`)},
		{"type table", "an element declaration with a {type table}", `<xs:element name="%[1]s" type="xs:string"><xs:alternative test="@a" type="xs:token"/></xs:element>`},
		{"identity constraint", identityName, `<xs:element name="%[1]s"><xs:complexType><xs:sequence>` + a + `</xs:sequence></xs:complexType><xs:key name="k%[1]s"><xs:selector xpath="s:a[1]"/><xs:field xpath="."/></xs:key></xs:element>`},
		{"identity constraint field", identityName, `<xs:element name="%[1]s"><xs:complexType><xs:sequence>` + a + `</xs:sequence></xs:complexType><xs:unique name="k%[1]s"><xs:selector xpath="s:a"/><xs:field xpath="."/><xs:field xpath="s:a[1]"/></xs:unique></xs:element>`},
		{"keyref referenced key", identityName, `<xs:element name="h%[1]s"><xs:complexType><xs:sequence>` + a + `</xs:sequence></xs:complexType><xs:key name="hk%[1]s"><xs:selector xpath="s:a[1]"/><xs:field xpath="."/></xs:key></xs:element>` +
			`<xs:element name="%[1]s"><xs:complexType><xs:sequence>` + a + `</xs:sequence></xs:complexType><xs:keyref name="kr%[1]s" refer="s:hk%[1]s"><xs:selector xpath="s:a"/><xs:field xpath="."/></xs:keyref></xs:element>`},
		{"absent affiliation", absentAffiliationName, `<xs:element name="%[1]s" type="xs:string" substitutionGroup="s:nope%[1]s"/>`},
		{"ref absent affiliation", absentAffiliationName, `<xs:element name="x%[1]s" type="xs:string" substitutionGroup="s:nope%[1]s"/>` +
			sequence(`<xs:element ref="s:x%[1]s"/>`)},
		{"member type table", "an element declaration with a {type table}", `<xs:element name="x%[1]s" type="xs:string"/><xs:element name="m%[1]s" type="xs:string" substitutionGroup="s:x%[1]s"><xs:alternative test="@a" type="xs:token"/></xs:element>` +
			sequence(`<xs:element ref="s:x%[1]s"/>`)},
		{"member absent affiliation", absentAffiliationName, `<xs:element name="x%[1]s" type="xs:string"/><xs:element name="m%[1]s" type="xs:string" substitutionGroup="s:x%[1]s s:nope%[1]s"/>` +
			sequence(`<xs:element ref="s:x%[1]s"/>`)},
		{"nillable anyType", "an element declaration with {nillable} true", `<xs:element name="%[1]s" nillable="true"/>`},
		{"anyType particle identity constraint", identityName, sequence(`<xs:element name="a"><xs:unique name="u%[1]s"><xs:selector xpath="."/><xs:field xpath="..[1]"/></xs:unique></xs:element>`)},
		{"abstract type", "an element declaration whose {type definition} is a complex type with {abstract} true", abstractType},
		{"abstract particle type", "an element declaration whose {type definition} is a complex type with {abstract} true", `<xs:complexType name="ab%[1]s" abstract="true"/>` + sequence(`<xs:element name="a" type="s:ab%[1]s"/>`)},
		{"simple content", entityTypeName, complexOf(`<xs:simpleContent><xs:extension base="xs:ENTITY"/></xs:simpleContent>`)},
		{"simple content before attribute uses", entityTypeName, complexOf(`<xs:simpleContent><xs:extension base="xs:ENTITY"><xs:attribute name="a" type="xs:string" form="qualified"/></xs:extension></xs:simpleContent>`)},
		{"mixed", "a complex type with mixed content", `<xs:element name="%[1]s"><xs:complexType mixed="true"><xs:sequence>` + a + `</xs:sequence></xs:complexType></xs:element>`},
		{"open content", "an {open content}", complexOf(`<xs:openContent><xs:any namespace="##other" processContents="lax"/></xs:openContent><xs:sequence>` + a + `</xs:sequence>`)},
		{"flattened", "an element declaration with {nillable} true", complexOf(`<xs:sequence maxOccurs="unbounded"><xs:element name="a" type="xs:string" minOccurs="0" nillable="true"/></xs:sequence>`)},
		{"flattened untracked", "a content model the content matcher does not track", complexOf(`<xs:sequence maxOccurs="21845"><xs:element name="a" type="xs:string" maxOccurs="2"/></xs:sequence>`)},
		{"nested flattened particle in an occurrence field", "a nested model group particle", complexOf(`<xs:sequence maxOccurs="2">` + a +
			`<xs:sequence maxOccurs="unbounded"><xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence></xs:sequence>`)},
		{"nested occurrence field in an occurrence field", "a nested model group particle", complexOf(`<xs:sequence maxOccurs="2">` + a +
			`<xs:choice maxOccurs="2"><xs:element name="b" type="xs:string"/><xs:element name="c" type="xs:string"/></xs:choice></xs:sequence>`)},
		{"nested occurrence field as an item's model-group alternative", "a nested model group particle", complexOf(`<xs:sequence maxOccurs="2">` + a +
			`<xs:choice><xs:element name="b" type="xs:string"/><xs:sequence maxOccurs="2"><xs:element name="c" type="xs:string"/></xs:sequence></xs:choice></xs:sequence>`)},
		{"nested occurrence field in a nested occurrence field's item", "a nested model group particle", sequence(a + `<xs:sequence maxOccurs="2"><xs:element name="b" type="xs:string"/>` +
			`<xs:sequence maxOccurs="2"><xs:element name="c" type="xs:string"/></xs:sequence></xs:sequence>`)},
		{"nested flattened particle in a nested occurrence field's item", "a nested model group particle", sequence(a + `<xs:sequence maxOccurs="2"><xs:element name="b" type="xs:string"/>` +
			`<xs:sequence minOccurs="0" maxOccurs="unbounded"><xs:element name="c" type="xs:string" minOccurs="0"/></xs:sequence></xs:sequence>`)},
		{"nested flattened untracked", "a content model the content matcher does not track", sequence(a + `<xs:sequence maxOccurs="21845"><xs:element name="c" type="xs:string" maxOccurs="2"/></xs:sequence>`)},
		{"strict wildcard", "a wildcard particle with {process contents} strict", sequence(`<xs:any/>`)},
		{"ref type table", "an element declaration with a {type table}", `<xs:element name="x%[1]s" type="xs:string"><xs:alternative test="@a" type="xs:token"/></xs:element>` + sequence(`<xs:element ref="s:x%[1]s"/>`)},
		{"ref simple content", entityTypeName, `<xs:element name="x%[1]s"><xs:complexType><xs:simpleContent><xs:extension base="xs:ENTITY"/></xs:simpleContent></xs:complexType></xs:element>` +
			sequence(`<xs:element ref="s:x%[1]s"/>`)},
		{"complex particle", "a wildcard particle with {process contents} strict", sequence(`<xs:element name="a"><xs:complexType><xs:sequence><xs:any/></xs:sequence></xs:complexType></xs:element>`)},
		{"particle type sharing its element's name", "a wildcard particle with {process contents} strict", `<xs:complexType name="%[1]s"><xs:sequence><xs:any/></xs:sequence></xs:complexType>` +
			sequence(`<xs:element name="c" type="s:%[1]s"/>`)},
		{"xsi:type attribute use", "an attribute use whose declaration's {target namespace} is the XSI namespace, other than xsi:schemaLocation's and xsi:noNamespaceSchemaLocation's, or the one the prefix xmlns is bound to", complexOf(`<xs:attribute ref="xsi:type"/>`)},
		{"complex assertions", "a complex type with {assertions}", complexOf(asserted)},
		{"restricted complex assertions", "a complex type with {assertions}", `<xs:complexType name="ab%[1]s">` + asserted + `</xs:complexType>` +
			`<xs:element name="%[1]s"><xs:complexType><xs:complexContent><xs:restriction base="s:ab%[1]s"><xs:sequence>` + a + `</xs:sequence></xs:restriction></xs:complexContent></xs:complexType></xs:element>`},
		{"list of a minted primitive", encodePrimitiveName, simpleOf(`<xs:list itemType="s:money"/>`)},
		{"list of union of an ENTITY member", entityTypeName, simpleOf(`<xs:list><xs:simpleType><xs:union memberTypes="xs:int xs:ENTITY"/></xs:simpleType></xs:list>`)},
		{"ENTITIES", entityTypeName, `<xs:element name="%[1]s" type="xs:ENTITIES"/>`},
		{"list assertions", "a simple type with an assertions facet", simpleOf(`<xs:restriction base="xs:NMTOKENS"><xs:assertion test="true()"/></xs:restriction>`)},
		{"inherited list assertions", "a simple type with an assertions facet", `<xs:simpleType name="assertedList%[1]s"><xs:restriction base="xs:NMTOKENS"><xs:assertion test="true()"/></xs:restriction></xs:simpleType>` + simpleOf(`<xs:restriction base="s:assertedList%[1]s"><xs:maxLength value="3"/></xs:restriction>`)},
		{"list item assertions", "a simple type with an assertions facet", `<xs:simpleType name="assertedItem%[1]s"><xs:restriction base="xs:string"><xs:assertion test="true()"/></xs:restriction></xs:simpleType>` + simpleOf(`<xs:list itemType="s:assertedItem%[1]s"/>`)},
		{"union of an ENTITY member", entityTypeName, simpleOf(`<xs:union memberTypes="xs:int xs:ENTITY"/>`)},
		{"union of a minted member", encodePrimitiveName, simpleOf(`<xs:union memberTypes="xs:int s:money"/>`)},
		{"union assertions", "a simple type with an assertions facet", simpleOf(`<xs:restriction><xs:simpleType><xs:union memberTypes="xs:int xs:date"/></xs:simpleType><xs:assertion test="true()"/></xs:restriction>`)},
		{"member union assertions", "a simple type with an assertions facet", `<xs:simpleType name="assertedUnion%[1]s"><xs:restriction><xs:simpleType><xs:union memberTypes="xs:int"/></xs:simpleType><xs:assertion test="true()"/></xs:restriction></xs:simpleType>` +
			simpleOf(`<xs:union memberTypes="s:assertedUnion%[1]s xs:date"/>`)},
		{"minted primitive", encodePrimitiveName, `<xs:element name="%[1]s" type="s:money"/>`},
		{"minted primitive attribute", encodePrimitiveName, complexOf(`<xs:attribute name="a" type="s:money"/>`)},
		{"ENTITY", entityTypeName, `<xs:element name="%[1]s" type="xs:ENTITY"/>`},
		{"ENTITY particle", entityTypeName, sequence(`<xs:element name="a" type="xs:ENTITY"/>`)},
		{"ENTITY restriction", entityTypeName, simpleOf(`<xs:restriction base="xs:ENTITY"/>`)},
		{"assertions", "a simple type with an assertions facet", simpleOf(`<xs:restriction base="xs:string"><xs:assertion test="true()"/></xs:restriction>`)},
		{"decimal assertions", "a simple type with an assertions facet", simpleOf(`<xs:restriction base="xs:decimal"><xs:assertion test="true()"/></xs:restriction>`)},
		{"boolean assertions", "a simple type with an assertions facet", simpleOf(`<xs:restriction base="xs:boolean"><xs:assertion test="true()"/></xs:restriction>`)},
		{"date assertions", "a simple type with an assertions facet", simpleOf(`<xs:restriction base="xs:date"><xs:assertion test="true()"/></xs:restriction>`)},
		{"inherited assertions", "a simple type with an assertions facet", `<xs:simpleType name="asserted%[1]s"><xs:restriction base="xs:string"><xs:assertion test="true()"/></xs:restriction></xs:simpleType>` + simpleOf(`<xs:restriction base="s:asserted%[1]s"><xs:maxLength value="3"/></xs:restriction>`)},
		{"particle type table", "an element declaration with a {type table}", sequence(`<xs:element name="a" type="xs:string"><xs:alternative test="@b" type="xs:token"/></xs:element>`)},
		{"particle identity constraint", identityName, sequence(`<xs:element name="a" type="xs:string"><xs:unique name="u%[1]s"><xs:selector xpath="."/><xs:field xpath="..[1]"/></xs:unique></xs:element>`)},
	}
	appendOf := map[string]func(t *testing.T, dst []byte, schema *xsd.Schema, name xsd.QName) ([]byte, error){
		"abstract type": func(t *testing.T, dst []byte, schema *xsd.Schema, name xsd.QName) ([]byte, error) {
			return appendChecked(t, dst, schema, strict.New(), name, abstractMember{A: "a", C: "c", X: "x"})
		},
		"complex assertions": func(t *testing.T, dst []byte, schema *xsd.Schema, name xsd.QName) ([]byte, error) {
			return appendChecked(t, dst, schema, strict.New(), name, struct{ A string }{A: "a"})
		},
		"restricted complex assertions": func(t *testing.T, dst []byte, schema *xsd.Schema, name xsd.QName) ([]byte, error) {
			return appendChecked(t, dst, schema, strict.New(), name, struct{ A string }{A: "a"})
		},
	}
	// subjectOf is the subject a row's decline names after its construct, where
	// the row pins it: a particle inside a nested occurrence field's item by
	// its place, which Decode's text for the same particle does not open with.
	nestedInItem := "particle 2 of particle 2 of the content {particle} of "
	subjectOf := map[string]string{
		"nested occurrence field in a nested occurrence field's item":   nestedInItem,
		"nested flattened particle in a nested occurrence field's item": nestedInItem,
	}
	seen := map[string]Construct{}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			decls := fmt.Sprintf(row.decl, "r") + fmt.Sprintf(row.decl, "q")
			schemaFor := schemaOf
			if strings.Contains(decls, "s:money") {
				schemaFor = mintedSchema
			}
			schema := schemaFor(t, decls)
			dst := []byte("pre")
			var constructs []Construct
			for _, name := range []string{"r", "q"} {
				got, err := appendChecked(t, dst, schema, strict.New(), sname(name), struct{}{})
				if v, has := appendOf[row.name]; has {
					got, err = v(t, dst, schema, sname(name))
				}
				d := asDecline(t, err, "codec does not encode "+row.construct+": "+subjectOf[row.name])
				if string(got) != "pre" {
					t.Errorf("Append wrote %q", got)
				}
				constructs = append(constructs, d.Construct())
			}
			if constructs[0] != constructs[1] {
				t.Errorf("r and q decline by %v and %v", constructs[0], constructs[1])
			}
			for other, c := range seen {
				if (other == row.construct) != (c == constructs[0]) {
					t.Errorf("Construct %v against %q's %v", constructs[0], other, c)
				}
			}
			seen[row.construct] = constructs[0]
		})
	}
	// An attribute use whose declaration is in the namespace the prefix xmlns
	// is bound to, which no parser rule charges, declines by
	// constructReservedAttributeUse, as the xsi:type row does.
	t.Run("xmlns namespace attribute use", func(t *testing.T) {
		const reserved = "an attribute use whose declaration's {target namespace} is the XSI namespace, other than xsi:schemaLocation's and xsi:noNamespaceSchemaLocation's, or the one the prefix xmlns is bound to"
		schema, err := parser.Parse("a.xsd", parser.WithResolver(loader.Map(map[string]string{
			"a.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:n="http://www.w3.org/2000/xmlns/" targetNamespace="urn:a">
			  <xs:import namespace="http://www.w3.org/2000/xmlns/" schemaLocation="n.xsd"/>
			  <xs:element name="r"><xs:complexType><xs:attribute ref="n:a"/></xs:complexType></xs:element>
			</xs:schema>`,
			"n.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="http://www.w3.org/2000/xmlns/">
			  <xs:attribute name="a" type="xs:string"/>
			</xs:schema>`,
		})))
		if err != nil {
			t.Fatal(err)
		}
		_, err = appendChecked(t, nil, schema, strict.New(), xsd.QName{Space: "urn:a", Local: "r"}, struct{}{})
		d := asDecline(t, err, "codec does not encode "+reserved+": ")
		if want := (Construct{constructReservedAttributeUse}); d.Construct() != want {
			t.Errorf("Construct %v, want %v", d.Construct(), want)
		}
	})
}

// encodeListSchema declares, in urn:s, document elements of the list types "#
// What encodes" admits: xs:NMTOKENS, user lists over xs:double, xs:string and
// xs:token, in an element, a required and an optional attribute and three
// particles, once, optional and repeated, of lists over xs:string and
// xs:double; and restrictions of them by a pattern, length, minLength,
// maxLength and enumeration facet.
const encodeListSchema = `
  <xs:simpleType name="doubles"><xs:list itemType="xs:double"/></xs:simpleType>
  <xs:simpleType name="strings"><xs:list itemType="xs:string"/></xs:simpleType>
  <xs:simpleType name="tokens"><xs:list itemType="xs:token"/></xs:simpleType>
  <xs:simpleType name="one"><xs:restriction base="xs:double"><xs:pattern value="1"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="ones"><xs:list itemType="s:one"/></xs:simpleType>
  <xs:simpleType name="twoDoubles"><xs:restriction base="s:doubles"><xs:pattern value="1\.0 2"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="onlyX"><xs:restriction base="s:strings"><xs:pattern value="x"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="pair"><xs:restriction base="s:strings"><xs:length value="2"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="some"><xs:restriction base="s:strings"><xs:minLength value="1"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="single"><xs:restriction base="s:strings"><xs:maxLength value="1"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="ab"><xs:restriction base="s:strings"><xs:enumeration value="a b"/></xs:restriction></xs:simpleType>
  <xs:element name="nm" type="xs:NMTOKENS"/>
  <xs:element name="dl" type="s:doubles"/>
  <xs:element name="sl" type="s:strings"/>
  <xs:element name="tl" type="s:tokens"/>
  <xs:element name="ol" type="s:ones"/>
  <xs:element name="dp" type="s:twoDoubles"/>
  <xs:element name="sx" type="s:onlyX"/>
  <xs:element name="pair" type="s:pair"/>
  <xs:element name="some" type="s:some"/>
  <xs:element name="single" type="s:single"/>
  <xs:element name="ab" type="s:ab"/>
  <xs:element name="da">
    <xs:complexType><xs:attribute name="a" type="s:doubles" use="required"/></xs:complexType>
  </xs:element>
  <xs:element name="oda">
    <xs:complexType><xs:attribute name="a" type="s:doubles"/></xs:complexType>
  </xs:element>
  <xs:element name="lp">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="once" type="s:strings"/>
        <xs:element name="opt" type="s:doubles" minOccurs="0"/>
        <xs:element name="rep" type="s:strings" maxOccurs="2"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>`

// The T codegen emits for encodeListSchema's document elements.
type (
	strList     []string
	dblList     []float64
	listAttr    struct{ A []float64 }
	optListAttr struct{ A *[]float64 }
	listParts   struct {
		Once []string
		Opt  *[]float64
		Rep  [][]string
	}
)

// sameLists reports whether a and b are one value, codegen's reading of a
// list field that is not a pointer: nil and the empty slice one value, the
// empty list, at any depth.
func sameLists[T any](a, b T) bool { return sameValue(reflect.ValueOf(a), reflect.ValueOf(b)) }

// sameValue is sameLists over reflect.Values of one type.
func sameValue(a, b reflect.Value) bool {
	switch a.Kind() {
	case reflect.Slice:
		if a.Len() != b.Len() {
			return false
		}
		for i := range a.Len() {
			if !sameValue(a.Index(i), b.Index(i)) {
				return false
			}
		}
		return true
	case reflect.Pointer:
		if a.IsNil() || b.IsNil() {
			return a.IsNil() == b.IsNil()
		}
		return sameValue(a.Elem(), b.Elem())
	case reflect.Struct:
		for i := range a.NumField() {
			if !sameValue(a.Field(i), b.Field(i)) {
				return false
			}
		}
		return true
	default:
		return a.Interface() == b.Interface()
	}
}

// TestAppendLists pins the list form "# What encodes" states, the empty list
// as nil and as an empty slice, in an element, a required and an optional
// attribute, and a particle occurring once ([]V), optionally (*[]V) and
// repeatedly ([][]V), and xs:string items holding white space other than S
// written as themselves. Each document is accepted by validate and decoded
// back into the value, nil and empty one value.
func TestAppendLists(t *testing.T) {
	schema := schemaOf(t, encodeListSchema)
	dbl := func(f ...float64) *[]float64 { return &f }
	t.Run("NMTOKENS element", func(t *testing.T) {
		roundTrip(t, schema, sname("nm"), strList{"a", "b:c", "1"}, `<nm xmlns="urn:s">a b:c 1</nm>`, sameLists[strList])
	})
	t.Run("token and string items", func(t *testing.T) {
		roundTrip(t, schema, sname("tl"), strList{"a", "b"}, `<tl xmlns="urn:s">a b</tl>`, sameLists[strList])
		roundTrip(t, schema, sname("sl"), strList{"<&>"}, `<sl xmlns="urn:s">&lt;&amp;&gt;</sl>`, sameLists[strList])
	})
	t.Run("string items holding white space S is not", func(t *testing.T) {
		// U+00A0 and U+0085 are unicode.IsSpace's and no list separator,
		// which is XML's S (Datatypes §2.4.1.2, cvc-datatype-valid 2.2).
		roundTrip(t, schema, sname("sl"), strList{"a b", "\u0085"}, "<sl xmlns=\"urn:s\">a b \u0085</sl>", sameLists[strList])
	})
	t.Run("double element", func(t *testing.T) {
		roundTrip(t, schema, sname("dl"), dblList{1, -0.5, math.Inf(1)}, `<dl xmlns="urn:s">1.0E0 -5.0E-1 INF</dl>`, sameLists[dblList])
	})
	t.Run("double attribute", func(t *testing.T) {
		roundTrip(t, schema, sname("da"), listAttr{A: []float64{1, 2.5}}, `<da xmlns="urn:s" a="1.0E0 2.5E0"/>`, sameLists[listAttr])
	})
	for _, row := range []struct {
		name string
		v    []string
	}{{"nil", nil}, {"empty", []string{}}} {
		t.Run("empty list element, "+row.name, func(t *testing.T) {
			roundTrip(t, schema, sname("sl"), strList(row.v), `<sl xmlns="urn:s"/>`, sameLists[strList])
		})
	}
	for _, row := range []struct {
		name string
		v    []float64
	}{{"nil", nil}, {"empty", []float64{}}} {
		t.Run("empty list attribute, "+row.name, func(t *testing.T) {
			roundTrip(t, schema, sname("da"), listAttr{A: row.v}, `<da xmlns="urn:s" a=""/>`, sameLists[listAttr])
		})
	}
	t.Run("optional attribute", func(t *testing.T) {
		roundTrip(t, schema, sname("oda"), optListAttr{}, `<oda xmlns="urn:s"/>`, sameLists[optListAttr])
		roundTrip(t, schema, sname("oda"), optListAttr{A: dbl()}, `<oda xmlns="urn:s" a=""/>`, sameLists[optListAttr])
		roundTrip(t, schema, sname("oda"), optListAttr{A: dbl(3)}, `<oda xmlns="urn:s" a="3.0E0"/>`, sameLists[optListAttr])
	})
	for _, row := range []struct {
		name string
		v    listParts
		want string
	}{
		{"once of two items, repeated of two", listParts{Once: []string{"x", "y"}, Rep: [][]string{{"a", "b"}, {"c"}}},
			`<lp xmlns="urn:s"><once>x y</once><rep>a b</rep><rep>c</rep></lp>`},
		{"optional empty, repeated empty item", listParts{Once: []string{"x"}, Opt: dbl(), Rep: [][]string{{}}},
			`<lp xmlns="urn:s"><once>x</once><opt/><rep/></lp>`},
		{"once empty, optional of one item", listParts{Opt: dbl(1), Rep: [][]string{nil, {"d"}}},
			`<lp xmlns="urn:s"><once/><opt>1.0E0</opt><rep/><rep>d</rep></lp>`},
	} {
		t.Run("particles, "+row.name, func(t *testing.T) {
			roundTrip(t, schema, sname("lp"), row.v, row.want, sameLists[listParts])
		})
	}
}

// TestAppendListInvalid pins the list cases of items 6 and 7 of Append's doc,
// under item types that admit the item, an item's own check answering first,
// before a backend fault of the list's own, and an item's own item 7 named at
// the item's path.
func TestAppendListInvalid(t *testing.T) {
	schema := schemaOf(t, encodeListSchema)
	for _, row := range []struct {
		name, elem string
		v          strList
		rule       xsderr.Rule
		want       []string
	}{
		{"an item holding a space", "sl", strList{"a", "b c"}, "", []string{"strList[1]", `"b c"`, "empty or holds white space", "through the list type's {item type definition} xs:string"}},
		{"an empty item", "sl", strList{""}, "", []string{"strList[0]", `""`, "empty or holds white space"}},
		{"an item holding TAB", "sl", strList{"a\tb"}, "", []string{"strList[0]", "empty or holds white space"}},
		{"an item holding LF", "sl", strList{"a\nb"}, "", []string{"strList[0]", "empty or holds white space"}},
		{"an item holding CR", "sl", strList{"a\rb"}, "", []string{"strList[0]", "empty or holds white space"}},
		{"a token item holding a space", "tl", strList{"a b"}, "", []string{"strList[0]", "empty or holds white space"}},
		{"a non-Char item", "sl", strList{"a", "\x00"}, "", []string{"strList[1]", "XML 1.0 Char"}},
		{"a token item not collapsed", "tl", strList{" a"}, "", []string{"strList[0]", `whiteSpace-normalized form "a"`}},
		{"an empty NMTOKENS", "nm", strList{}, "cvc-minLength-valid", []string{"strList", "xs:NMTOKENS"}},
		{"length", "pair", strList{"a"}, "cvc-length-valid", []string{"strList", `written "a"`, "{urn:s}pair"}},
		{"minLength", "some", nil, "cvc-minLength-valid", []string{"strList", "{urn:s}some"}},
		{"maxLength", "single", strList{"a", "b"}, "cvc-maxLength-valid", []string{"strList", `written "a b"`}},
		{"enumeration", "ab", strList{"a"}, "cvc-enumeration-valid", []string{"strList", "{urn:s}ab"}},
		{"a string list's pattern", "sx", strList{"a"}, "cvc-pattern-valid", []string{"strList", `written "a"`, "{urn:s}onlyX"}},
	} {
		t.Run(row.name, func(t *testing.T) {
			_, err := appendChecked(t, nil, schema, strict.New(), sname(row.elem), row.v)
			invalidValue(t, err, row.want...)
			if !strings.HasPrefix(err.Error(), "codec: Append: "+row.want[0]+" holds ") {
				t.Errorf("error %q does not open with the path %s", err, row.want[0])
			}
			if rule, _ := xsderr.RuleOf(err); rule != row.rule {
				t.Errorf("rule %q, want %q", rule, row.rule)
			}
		})
	}
	t.Run("a dirty item under a backend that cannot assess the list", func(t *testing.T) {
		// The item's own error answers, not the list-level fault of a
		// backend with no mapping: s.st's facets are checked only once every
		// item is clean.
		_, err := appendChecked(t, nil, schema, noMappings{}, sname("sl"), strList{"\x00"})
		invalidValue(t, err, "strList[0]", "XML 1.0 Char")
		if !strings.HasPrefix(err.Error(), "codec: Append: strList[0] holds ") {
			t.Errorf("error %q does not open with the path strList[0]", err)
		}
	})
	t.Run("a double list's empty value its pattern rejects", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("dp"), dblList{})
		invalidValue(t, err, "dblList", "{urn:s}twoDoubles")
		if rule, _ := xsderr.RuleOf(err); rule != "cvc-pattern-valid" {
			t.Errorf("rule %q, want cvc-pattern-valid", rule)
		}
	})
	for _, row := range []struct {
		name string
		v    dblList
	}{
		{"a double list of NaN its pattern rejects", dblList{math.NaN()}},
		{"a double list of -INF its pattern rejects", dblList{math.Inf(-1)}},
		{"a double list of NaN and -INF its pattern rejects", dblList{math.NaN(), math.Inf(-1)}},
	} {
		t.Run(row.name, func(t *testing.T) {
			_, err := appendChecked(t, nil, schema, strict.New(), sname("dp"), row.v)
			invalidValue(t, err, "dblList", "{urn:s}twoDoubles")
			if !strings.HasPrefix(err.Error(), "codec: Append: dblList holds ") {
				t.Errorf("error %q does not open with the path dblList", err)
			}
			if rule, _ := xsderr.RuleOf(err); rule != "cvc-pattern-valid" {
				t.Errorf("rule %q, want cvc-pattern-valid", rule)
			}
		})
	}
	t.Run("a double list holding NaN and 1 its pattern rejects", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("dp"), dblList{math.NaN(), 1})
		_ = asDecline(t, err, "codec does not encode a value with no canonical representation: simple type {urn:s}twoDoubles")
		if !strings.Contains(err.Error(), `a pattern facet rejects its canonical representation "NaN 1.0E0"`) {
			t.Errorf("decline %q does not name the join", err)
		}
	})
	t.Run("a double list's pattern rejecting its canonical join", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("dp"), dblList{1, 2})
		_ = asDecline(t, err, "codec does not encode a value with no canonical representation: simple type {urn:s}twoDoubles")
		if !strings.Contains(err.Error(), `at dblList holding [1 2]: a pattern facet rejects its canonical representation "1.0E0 2.0E0"`) {
			t.Errorf("decline %q does not name the path, value and join", err)
		}
	})
	t.Run("an item's own pattern rejecting an xs:double", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("ol"), dblList{1})
		_ = asDecline(t, err, "codec does not encode a value with no canonical representation: simple type {urn:s}ones")
		if !strings.Contains(err.Error(), `through the list type's {item type definition} {urn:s}one`) || !strings.Contains(err.Error(), "at dblList[0] holding 1") {
			t.Errorf("decline %q does not name the item type and the item's path", err)
		}
	})
}

// TestAppendListDeclineSubject pins the detail of a list's item decline: the
// list type, then its {item type definition}, by its QName where it has one
// and its schema Loc where it is located.
func TestAppendListDeclineSubject(t *testing.T) {
	schema := schemaOf(t, `<xs:element name="flt"><xs:simpleType><xs:list itemType="xs:ENTITY"/></xs:simpleType></xs:element>`+
		`<xs:simpleType name="dates"><xs:list><xs:simpleType><xs:restriction base="xs:ENTITY"/></xs:simpleType></xs:list></xs:simpleType>`+
		`<xs:element name="anon" type="s:dates"/>`)
	err := CheckAppend[strList](schema, sname("flt"))
	_ = asDecline(t, err, "codec does not encode a simple type derived from xs:ENTITY: simple type the anonymous simple type of element {urn:s}flt declared at s.xsd:1:")
	if !strings.Contains(err.Error(), ", through the list type's {item type definition} xs:ENTITY; ") {
		t.Errorf("decline %q does not name the item xs:ENTITY", err)
	}
	err = CheckAppend[strList](schema, sname("anon"))
	_ = asDecline(t, err, "codec does not encode a simple type derived from xs:ENTITY: simple type {urn:s}dates defined at s.xsd:1:")
	if !strings.Contains(err.Error(), ", through the list type's anonymous {item type definition}, a type defined at s.xsd:1:") {
		t.Errorf("decline %q does not name the anonymous item by its Loc", err)
	}
}

// encodeDecimalSchema declares, in urn:s, document elements of the types
// whose primitive is xs:decimal or xs:boolean, one per Go type codegen gives
// them that the rows below hold, a list over xs:integer, an element and an
// attribute of complex type, and restrictions by a bound, a pattern on an
// atom and on a list, and an assertion.
const encodeDecimalSchema = `
  <xs:simpleType name="integers"><xs:list itemType="xs:integer"/></xs:simpleType>
  <xs:simpleType name="booleans"><xs:list itemType="xs:boolean"/></xs:simpleType>
  <xs:simpleType name="small"><xs:restriction base="xs:decimal"><xs:maxInclusive value="10"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="onePoint"><xs:restriction base="xs:decimal"><xs:pattern value="1\.0"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="cents"><xs:restriction base="xs:decimal"><xs:pattern value="\d+\.\d\d"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="digitBool"><xs:restriction base="xs:boolean"><xs:pattern value="1|0"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="bits"><xs:restriction base="s:booleans"><xs:pattern value="[01]( [01])*"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="plusOnes"><xs:restriction base="s:integers"><xs:pattern value="\+1( \+1)*"/></xs:restriction></xs:simpleType>
  <xs:element name="dec" type="xs:decimal"/>
  <xs:element name="int" type="xs:integer"/>
  <xs:element name="pos" type="xs:positiveInteger"/>
  <xs:element name="long" type="xs:long"/>
  <xs:element name="ulong" type="xs:unsignedLong"/>
  <xs:element name="b" type="xs:boolean"/>
  <xs:element name="ints" type="s:integers"/>
  <xs:element name="small" type="s:small"/>
  <xs:element name="onePoint" type="s:onePoint"/>
  <xs:element name="cents" type="s:cents"/>
  <xs:element name="digitBool" type="s:digitBool"/>
  <xs:element name="bits" type="s:bits"/>
  <xs:element name="plusOnes" type="s:plusOnes"/>
  <xs:element name="ea">
    <xs:complexType>
      <xs:sequence><xs:element name="v" type="xs:decimal"/></xs:sequence>
      <xs:attribute name="a" type="xs:boolean" use="required"/>
      <xs:attribute name="n" type="xs:byte"/>
    </xs:complexType>
  </xs:element>`

// The T codegen emits for encodeDecimalSchema's document elements, but for
// the ones held in big.Rat, big.Int, int64, uint64 and bool themselves.
type (
	intList  []big.Int
	boolList []bool
	decParts struct {
		V big.Rat
		A bool
		N *int8
	}
)

// sameRat and sameInt report whether a and b are one value, by Cmp.
func sameRat(a, b big.Rat) bool { return a.Cmp(&b) == 0 }
func sameInt(a, b big.Int) bool { return a.Cmp(&b) == 0 }

// sameInts is sameInt over each item of a and b.
func sameInts(a, b intList) bool { return slices.EqualFunc(a, b, sameInt) }

// sameParts is sameRat over V and == over A and *N.
func sameParts(a, b decParts) bool {
	return sameRat(a.V, b.V) && a.A == b.A && (a.N == nil) == (b.N == nil) && (a.N == nil || *a.N == *b.N)
}

// TestAppendDecimalAndBoolean pins the canonical forms of the types whose
// primitive is xs:decimal or xs:boolean, one row per Go type codegen gives
// them that these rows hold: a big.Rat not an integer, with its shortest
// exact expansion (f-decCanFragMap), and an integer one with no decimal point
// (f-decimalCanmap through f-noDecCanMap); a big.Int beyond int64; a signed
// bounded integer, its minimum; a uint64 above math.MaxInt64 under
// xs:unsignedLong; a bool, true and false; a list over xs:integer; and an
// element and an attribute of complex type. Each is accepted by validate and
// decoded back equal, big.Rat and big.Int by Cmp.
func TestAppendDecimalAndBoolean(t *testing.T) {
	schema := schemaOf(t, encodeDecimalSchema)
	for _, row := range []struct {
		r    *big.Rat
		want string
	}{
		{big.NewRat(3, 2), "1.5"},
		{big.NewRat(-1, 2), "-0.5"},
		{big.NewRat(1, 20), "0.05"},
		{big.NewRat(4, 2), "2"},
		{big.NewRat(-7, 1), "-7"},
		{new(big.Rat), "0"},
	} {
		t.Run("decimal "+row.want, func(t *testing.T) {
			roundTrip(t, schema, sname("dec"), *row.r, `<dec xmlns="urn:s">`+row.want+`</dec>`, sameRat)
		})
	}
	t.Run("integer beyond int64", func(t *testing.T) {
		i, _ := new(big.Int).SetString("-12345678901234567890", 10)
		roundTrip(t, schema, sname("int"), *i, `<int xmlns="urn:s">-12345678901234567890</int>`, sameInt)
	})
	t.Run("long minimum", func(t *testing.T) {
		roundTrip(t, schema, sname("long"), int64(math.MinInt64), `<long xmlns="urn:s">-9223372036854775808</long>`, deepEqual[int64])
	})
	t.Run("unsignedLong above math.MaxInt64", func(t *testing.T) {
		roundTrip(t, schema, sname("ulong"), uint64(math.MaxUint64), `<ulong xmlns="urn:s">18446744073709551615</ulong>`, deepEqual[uint64])
	})
	for _, b := range []bool{true, false} {
		t.Run("boolean "+strconv.FormatBool(b), func(t *testing.T) {
			roundTrip(t, schema, sname("b"), b, `<b xmlns="urn:s">`+strconv.FormatBool(b)+`</b>`, deepEqual[bool])
		})
	}
	t.Run("list of integer", func(t *testing.T) {
		roundTrip(t, schema, sname("ints"), intList{*big.NewInt(1), *big.NewInt(-20), {}}, `<ints xmlns="urn:s">1 -20 0</ints>`, sameInts)
	})
	t.Run("element and attributes", func(t *testing.T) {
		n := int8(-3)
		roundTrip(t, schema, sname("ea"), decParts{V: *big.NewRat(5, 4), A: true, N: &n}, `<ea xmlns="urn:s" a="true" n="-3"><v>1.25</v></ea>`, sameParts)
		roundTrip(t, schema, sname("ea"), decParts{V: *big.NewRat(10, 1)}, `<ea xmlns="urn:s" a="false"><v>10</v></ea>`, sameParts)
	})
}

// The backends of TestAppendDecimalFaults and TestAppendDecimalInvalid, each
// strict's with xs:decimal's mapping replaced, but for positiveRejectsZero,
// which adds one for xs:positiveInteger.
var (
	decimalRejectingParse = overDatatype("decimal", func(base value.Mapping) value.Mapping {
		return value.Mapping{Parse: func(string, value.Context) (value.Value, error) { return nil, errors.New("rejected") }, Canonical: base.Canonical}
	})
	decimalPointedCanonical = overDatatype("decimal", func(base value.Mapping) value.Mapping {
		return value.Mapping{Parse: base.Parse, Canonical: func(v value.Value) (string, error) {
			c, err := base.Canonical(v)
			return c + ".0", err
		}}
	})
	decimalSpacedCanonical = overDatatype("decimal", func(base value.Mapping) value.Mapping {
		return value.Mapping{Parse: base.Parse, Canonical: func(v value.Value) (string, error) {
			c, err := base.Canonical(v)
			return " " + c + ".0", err
		}}
	})
	decimalOtherCanonical = overDatatype("decimal", func(base value.Mapping) value.Mapping {
		return value.Mapping{Parse: base.Parse, Canonical: func(value.Value) (string, error) { return "2", nil }}
	})
	decimalNilCanonical = overDatatype("decimal", func(base value.Mapping) value.Mapping {
		return value.Mapping{Parse: base.Parse}
	})
	decimalFailingCanonical = overDatatype("decimal", func(base value.Mapping) value.Mapping {
		return value.Mapping{Parse: base.Parse, Canonical: func(value.Value) (string, error) { return "", errors.New("no canonical form") }}
	})
	positiveRejectsZero = func() value.Backend {
		decimal, _ := strict.New().Mapping(decimalName)
		return value.Override(strict.New(), oneMapping{name: xsd.QName{Space: xsd.XMLSchemaNS, Local: "positiveInteger"}, m: value.Mapping{
			Parse: func(s string, ctx value.Context) (value.Value, error) {
				if s == "0" {
					return nil, errors.New("not positive")
				}
				return decimal.Parse(s, ctx)
			},
			Canonical: decimal.Canonical,
		}})
	}()
)

// decimalOnce is strict's backend, but that it maps xs:decimal at the first
// ask alone, so that value.ValidateLexical, asking after Append's own lookup
// of the primitive's mapping, finds no governing mapping.
type decimalOnce struct{ asked *bool }

func (d decimalOnce) Mapping(typ xsd.QName) (value.Mapping, bool) {
	if typ == decimalName {
		if *d.asked {
			return value.Mapping{}, false
		}
		*d.asked = true
	}
	return strict.New().Mapping(typ)
}

// TestAppendDecimalInvalid pins items 6 and 7 of Append's doc for the types
// whose primitive is xs:decimal or xs:boolean: a value a bound rejects and a
// big.Rat outside the decimal value space, 1/3, are item 6, each error
// opening with its Go path; a value whose canonical representation a
// pattern rejects is item 7, as an atom, a decimal and a boolean, and as a
// list under a list-level pattern, a boolean and an integer list, since every
// such value has another lexical; item 7's example, 3/2 under the pattern
// \d+\.\d\d, is the "cents" row, and Decode of its 1.50 returns 3/2; and a
// backend's mapping of xs:positiveInteger rejecting 0 is item 6, a verdict of
// the governing mapping's Parse, never a backend fault.
func TestAppendDecimalInvalid(t *testing.T) {
	schema := schemaOf(t, encodeDecimalSchema)
	t.Run("a bound", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("small"), *big.NewRat(11, 1))
		invalidValue(t, err, "{urn:s}small")
		if !strings.HasPrefix(err.Error(), "codec: Append: Rat holds 11, which simple type {urn:s}small") {
			t.Errorf("error %q does not open with the path, value and type", err)
		}
		if rule, _ := xsderr.RuleOf(err); rule != "cvc-maxInclusive-valid" {
			t.Errorf("rule %q, want cvc-maxInclusive-valid", rule)
		}
	})
	t.Run("outside the decimal value space", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("dec"), *big.NewRat(1, 3))
		invalidValue(t, err, "outside the decimal value space")
		noRule(t, err)
		if !strings.HasPrefix(err.Error(), "codec: Append: Rat holds 1/3, of simple type xs:decimal, ") {
			t.Errorf("error %q does not open with the path, value and type", err)
		}
	})
	t.Run("a two-place pattern admitting 1.50", func(t *testing.T) {
		got, err := Decode[big.Rat](schema, strict.New(), []byte(`<s:cents xmlns:s="urn:s">1.50</s:cents>`))
		if err != nil {
			t.Fatalf("Decode: %v", err)
		}
		if !sameRat(*got, *big.NewRat(3, 2)) {
			t.Errorf("Decode = %s, want 3/2", got.RatString())
		}
	})
	t.Run("a positiveInteger mapping rejecting 0", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, positiveRejectsZero, sname("pos"), big.Int{})
		invalidValue(t, err, "not positive")
		if !strings.HasPrefix(err.Error(), "codec: Append: Int holds 0, which simple type xs:positiveInteger rejects") {
			t.Errorf("error %q does not open with the path, value and type", err)
		}
	})
	for _, row := range []struct {
		name, elem string
		v          any
		at         string
	}{
		{"a decimal's pattern", "onePoint", *big.NewRat(1, 1), "at Rat holding 1: "},
		{"a decimal's two-place pattern", "cents", *big.NewRat(3, 2), "at Rat holding 1.5: "},
		{"a boolean's pattern", "digitBool", true, "at bool holding true: "},
		{"a boolean list's pattern", "bits", boolList{true, false}, "at boolList holding [true false]: "},
		{"an integer list's pattern", "plusOnes", intList{*big.NewInt(1), *big.NewInt(-2)}, "at intList holding [1 -2]: "},
	} {
		t.Run(row.name, func(t *testing.T) {
			var err error
			switch v := row.v.(type) {
			case big.Rat:
				_, err = appendChecked(t, nil, schema, strict.New(), sname(row.elem), v)
			case bool:
				_, err = appendChecked(t, nil, schema, strict.New(), sname(row.elem), v)
			case boolList:
				_, err = appendChecked(t, nil, schema, strict.New(), sname(row.elem), v)
			case intList:
				_, err = appendChecked(t, nil, schema, strict.New(), sname(row.elem), v)
			}
			_ = asDecline(t, err, "codec does not encode a value with no canonical representation: simple type {urn:s}"+row.elem+" defined at s.xsd:")
			holds(t, err, row.at+"a pattern facet rejects its canonical representation")
		})
	}
}

// TestAppendDecimalFaults pins the backend faults and the backend's item 7
// declines of Append's doc for a type whose primitive is xs:decimal or
// xs:boolean: no mapping of the primitive, the primitive's Parse rejecting
// the spelling, a Canonical rendering other text than the canonical
// representation, whether that text maps back to the value ("1.0", " 1.0")
// or not ("2"), and no governing mapping inside value.ValidateLexical are
// plain errors opening with the value's Go path; a nil or failing Canonical
// is item 7.
func TestAppendDecimalFaults(t *testing.T) {
	schema := schemaOf(t, encodeDecimalSchema)
	one := *big.NewRat(1, 1)
	_, err := appendChecked(t, nil, schema, noMappings{}, sname("b"), true)
	plainError(t, err, "codec: Append: bool, of simple type xs:boolean: the backend has no mapping of its primitive xs:boolean")
	_, err = appendChecked(t, nil, schema, decimalRejectingParse, sname("dec"), one)
	plainError(t, err, `codec: Append: Rat holds 1, of simple type xs:decimal, whose canonical spelling the backend's Parse of xs:decimal rejects: rejected`)
	for _, row := range []struct {
		name     string
		backend  value.Backend
		rendered string
	}{
		{"a spaced Canonical", decimalSpacedCanonical, `" 1.0"`},
		{"a pointed Canonical", decimalPointedCanonical, `"1.0"`},
		{"a Canonical not mapping back", decimalOtherCanonical, `"2"`},
	} {
		_, err = appendChecked(t, nil, schema, row.backend, sname("dec"), one)
		plainError(t, err, "codec: Append: Rat holds 1, of simple type xs:decimal, which the backend's Canonical renders "+row.rendered+", other text than its canonical representation (Datatypes Appendix E, the canonical mapping of xs:decimal)")
	}
	_, err = appendChecked(t, nil, schema, decimalOnce{asked: new(bool)}, sname("dec"), one)
	plainError(t, err, "codec: Append: Rat holds 1, of simple type xs:decimal, which the backend cannot assess: ")
	for _, row := range []struct {
		name    string
		backend value.Backend
		want    string
	}{
		{"a nil Canonical", decimalNilCanonical, "no canonical mapping"},
		{"a failing Canonical", decimalFailingCanonical, "no canonical form"},
	} {
		t.Run(row.name, func(t *testing.T) {
			_, err := appendChecked(t, nil, schema, row.backend, sname("dec"), one)
			_ = asDecline(t, err, "codec does not encode a value with no canonical representation: simple type xs:decimal, at Rat holding 1: ")
			holds(t, err, row.want)
		})
	}
}

// encodeTemporalSchema declares, in urn:s, a document element of each date,
// time and duration builtin codegen gives a datatype struct, a list over
// xs:date, an element and an attribute of complex type, and restrictions of
// xs:date by a bound and by explicitTimezone, and of xs:dateTime, xs:date,
// xs:gDay and the list by a pattern that rejects the canonical
// representation of the values the rows hold.
const encodeTemporalSchema = `
  <xs:element name="dt" type="xs:dateTime"/>
  <xs:element name="dts" type="xs:dateTimeStamp"/>
  <xs:element name="tm" type="xs:time"/>
  <xs:element name="da" type="xs:date"/>
  <xs:element name="ym" type="xs:gYearMonth"/>
  <xs:element name="gy" type="xs:gYear"/>
  <xs:element name="md" type="xs:gMonthDay"/>
  <xs:element name="gd" type="xs:gDay"/>
  <xs:element name="gm" type="xs:gMonth"/>
  <xs:element name="du" type="xs:duration"/>
  <xs:element name="dtd" type="xs:dayTimeDuration"/>
  <xs:element name="ymd" type="xs:yearMonthDuration"/>
  <xs:simpleType name="dates"><xs:list itemType="xs:date"/></xs:simpleType>
  <xs:simpleType name="early"><xs:restriction base="xs:date"><xs:maxInclusive value="2000-01-01"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="zoned"><xs:restriction base="xs:date"><xs:explicitTimezone value="required"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="pointed"><xs:restriction base="xs:dateTime"><xs:pattern value=".*\.0"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="plusZero"><xs:restriction base="xs:date"><xs:pattern value=".*\+00:00"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="minusYear"><xs:restriction base="xs:date"><xs:pattern value="-.*"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="plusZeroDay"><xs:restriction base="xs:gDay"><xs:pattern value=".*\+00:00"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="plusZeroDates"><xs:restriction base="s:dates"><xs:pattern value="\S*\+00:00( \S*\+00:00)*"/></xs:restriction></xs:simpleType>
  <xs:element name="dl" type="s:dates"/>
  <xs:element name="early" type="s:early"/>
  <xs:element name="zoned" type="s:zoned"/>
  <xs:element name="pointed" type="s:pointed"/>
  <xs:element name="plusZero" type="s:plusZero"/>
  <xs:element name="minusYear" type="s:minusYear"/>
  <xs:element name="plusZeroDay" type="s:plusZeroDay"/>
  <xs:element name="plusZeroDates" type="s:plusZeroDates"/>
  <xs:element name="ta">
    <xs:complexType>
      <xs:sequence><xs:element name="v" type="xs:dateTime"/></xs:sequence>
      <xs:attribute name="a" type="xs:date" use="required"/>
    </xs:complexType>
  </xs:element>`

// The T codegen emits for encodeTemporalSchema's list and complex document
// elements; the simple ones take temporal_test.go's datatype structs.
type (
	dateList  []tDate
	tempParts struct {
		V tDateTime
		A tDate
	}
)

// sameSpelled reports whether a and b are one value, by spellValue.
func sameSpelled[T any](a, b T) bool {
	return spellValue(reflect.ValueOf(a)) == spellValue(reflect.ValueOf(b))
}

// bigYear is the big.Int s spells in base 10.
func bigYear(s string) big.Int {
	var y big.Int
	y.SetString(s, 10)
	return y
}

// offset is a *int16 holding n.
func offset(n int16) *int16 { return &n }

// TestAppendTemporals pins the canonical forms of the date, time and
// duration builtins (Datatypes §E.3.5's canonical mappings, §E.2
// f-durationCanMap and f-yearMonthDurationCanMap), one row or more per
// datatype struct codegen gives them, the derived dateTimeStamp,
// dayTimeDuration and yearMonthDuration among them: a timezoneOffset as Z,
// and as a sign, hours and minutes; a year of four digits, zero-padded, and
// beyond them; a fractional second with no trailing zero; and a duration's
// fields with every zero one omitted. A list over xs:date and an element and
// an attribute of complex type take the same route. Each is accepted by
// validate and decoded back equal.
func TestAppendTemporals(t *testing.T) {
	schema := schemaOf(t, encodeTemporalSchema)
	el := func(name, text string) string { return `<` + name + ` xmlns="urn:s">` + text + `</` + name + `>` }
	for _, row := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"dateTime with an offset", func(t *testing.T) {
			roundTrip(t, schema, sname("dt"), tDateTime{Year: bigYear("2002"), Month: 10, Day: 10, Hour: 12, TimezoneOffset: offset(-300)},
				el("dt", "2002-10-10T12:00:00-05:00"), sameSpelled[tDateTime])
		}},
		{"dateTime with a fractional second and no offset", func(t *testing.T) {
			roundTrip(t, schema, sname("dt"), tDateTime{Year: bigYear("2002"), Month: 10, Day: 10, Hour: 12, Minute: 7, Second: *big.NewRat(1, 2)},
				el("dt", "2002-10-10T12:07:00.5"), sameSpelled[tDateTime])
		}},
		{"dateTime of a negative year at +14:00", func(t *testing.T) {
			roundTrip(t, schema, sname("dt"), tDateTime{Year: bigYear("-44"), Month: 3, Day: 15, Hour: 23, Minute: 59, Second: *big.NewRat(59999, 1000), TimezoneOffset: offset(840)},
				el("dt", "-0044-03-15T23:59:59.999+14:00"), sameSpelled[tDateTime])
		}},
		{"dateTime of a year beyond 9999 at Z", func(t *testing.T) {
			roundTrip(t, schema, sname("dt"), tDateTime{Year: bigYear("123456789012345678901234"), Month: 1, Day: 1, TimezoneOffset: offset(0)},
				el("dt", "123456789012345678901234-01-01T00:00:00Z"), sameSpelled[tDateTime])
		}},
		{"dateTimeStamp", func(t *testing.T) {
			roundTrip(t, schema, sname("dts"), tDateTimeStamp{Year: bigYear("2002"), Month: 10, Day: 10, Hour: 12, TimezoneOffset: -840},
				el("dts", "2002-10-10T12:00:00-14:00"), sameSpelled[tDateTimeStamp])
		}},
		{"time", func(t *testing.T) {
			roundTrip(t, schema, sname("tm"), tTime{Hour: 13, Minute: 20, Second: *big.NewRat(121, 4)}, el("tm", "13:20:30.25"), sameSpelled[tTime])
		}},
		{"time at Z", func(t *testing.T) {
			roundTrip(t, schema, sname("tm"), tTime{Hour: 1, Second: *big.NewRat(5, 1), TimezoneOffset: offset(0)}, el("tm", "01:00:05Z"), sameSpelled[tTime])
		}},
		{"date of February 29", func(t *testing.T) {
			roundTrip(t, schema, sname("da"), tDate{Year: bigYear("2000"), Month: 2, Day: 29, TimezoneOffset: offset(780)}, el("da", "2000-02-29+13:00"), sameSpelled[tDate])
		}},
		{"date of year 0", func(t *testing.T) {
			roundTrip(t, schema, sname("da"), tDate{Month: 1, Day: 1}, el("da", "0000-01-01"), sameSpelled[tDate])
		}},
		{"gYearMonth", func(t *testing.T) {
			roundTrip(t, schema, sname("ym"), tGYearMonth{Year: bigYear("1999"), Month: 5, TimezoneOffset: offset(330)}, el("ym", "1999-05+05:30"), sameSpelled[tGYearMonth])
		}},
		{"gYear negative", func(t *testing.T) {
			roundTrip(t, schema, sname("gy"), tGYear{Year: bigYear("-1")}, el("gy", "-0001"), sameSpelled[tGYear])
		}},
		{"gYear beyond 9999 at Z", func(t *testing.T) {
			roundTrip(t, schema, sname("gy"), tGYear{Year: bigYear("12345"), TimezoneOffset: offset(0)}, el("gy", "12345Z"), sameSpelled[tGYear])
		}},
		{"gMonthDay", func(t *testing.T) {
			roundTrip(t, schema, sname("md"), tGMonthDay{Month: 2, Day: 29}, el("md", "--02-29"), sameSpelled[tGMonthDay])
		}},
		{"gDay", func(t *testing.T) {
			roundTrip(t, schema, sname("gd"), tGDay{Day: 31, TimezoneOffset: offset(-180)}, el("gd", "---31-03:00"), sameSpelled[tGDay])
		}},
		{"gMonth", func(t *testing.T) {
			roundTrip(t, schema, sname("gm"), tGMonth{Month: 5}, el("gm", "--05"), sameSpelled[tGMonth])
		}},
		{"duration of every field", func(t *testing.T) {
			roundTrip(t, schema, sname("du"), tDuration{Months: *big.NewInt(14), Seconds: *big.NewRat(187569, 2)}, el("du", "P1Y2M1DT2H3M4.5S"), sameSpelled[tDuration])
		}},
		{"duration negative", func(t *testing.T) {
			roundTrip(t, schema, sname("du"), tDuration{Months: *big.NewInt(-1), Seconds: *big.NewRat(-60, 1)}, el("du", "-P1MT1M"), sameSpelled[tDuration])
		}},
		{"duration zero", func(t *testing.T) {
			roundTrip(t, schema, sname("du"), tDuration{}, el("du", "PT0S"), sameSpelled[tDuration])
		}},
		{"duration of months alone", func(t *testing.T) {
			roundTrip(t, schema, sname("du"), tDuration{Months: *big.NewInt(12)}, el("du", "P1Y"), sameSpelled[tDuration])
		}},
		{"dayTimeDuration", func(t *testing.T) {
			roundTrip(t, schema, sname("dtd"), tDayTimeDuration{Seconds: *big.NewRat(129600, 1)}, el("dtd", "P1DT12H"), sameSpelled[tDayTimeDuration])
		}},
		{"dayTimeDuration below a second", func(t *testing.T) {
			roundTrip(t, schema, sname("dtd"), tDayTimeDuration{Seconds: *big.NewRat(-1, 2)}, el("dtd", "-PT0.5S"), sameSpelled[tDayTimeDuration])
		}},
		{"yearMonthDuration", func(t *testing.T) {
			roundTrip(t, schema, sname("ymd"), tYearMonthDuration{Months: *big.NewInt(-3)}, el("ymd", "-P3M"), sameSpelled[tYearMonthDuration])
		}},
		{"list of date", func(t *testing.T) {
			roundTrip(t, schema, sname("dl"), dateList{{Year: bigYear("2000"), Month: 1, Day: 1}, {Year: bigYear("2000"), Month: 1, Day: 2, TimezoneOffset: offset(0)}},
				el("dl", "2000-01-01 2000-01-02Z"), sameSpelled[dateList])
		}},
		{"element and attribute", func(t *testing.T) {
			roundTrip(t, schema, sname("ta"), tempParts{V: tDateTime{Year: bigYear("2001"), Month: 12, Day: 31, Hour: 23, Minute: 59, Second: *big.NewRat(59, 1)}, A: tDate{Year: bigYear("2001"), Month: 1, Day: 1, TimezoneOffset: offset(60)}},
				`<ta xmlns="urn:s" a="2001-01-01+01:00"><v>2001-12-31T23:59:59</v></ta>`, sameSpelled[tempParts])
		}},
	} {
		t.Run(row.name, row.run)
	}
}

// TestAppendTemporalOutsideValueSpace pins item 6 of Append's doc for a
// datatype struct value no value maps to (codegen/doc.go "# Simple types"),
// one row per kind of property outside its range each struct can hold, the
// first in field order answering: an error errors.Is reports as
// ErrInvalidValue, carrying no *xsderr.Error, opening with the Go path, the
// value and its type, and naming the property, its value and its bound. The
// check precedes the backend's Parse and Canonical: under a backend whose
// mapping of the primitive rejects every lexical and renders none, the error
// is item 6 still.
func TestAppendTemporalOutsideValueSpace(t *testing.T) {
	schema := schemaOf(t, encodeTemporalSchema)
	day := tDate{Year: bigYear("2001"), Month: 2, Day: 29}
	with := func(f func(*tDateTime)) tDateTime {
		c := tDateTime{Year: bigYear("2001"), Month: 1, Day: 1}
		f(&c)
		return c
	}
	for _, row := range []struct {
		name, elem string
		v          any
		want       string
	}{
		{"a month of 13", "da", tDate{Year: bigYear("2000"), Month: 13, Day: 1}, "the month 13 is outside 1 to 12 (§D.2.1)"},
		{"a month of 0 before a day of 0", "da", tDate{Year: bigYear("2000")}, "the month 0 is outside 1 to 12 (§D.2.1)"},
		{"February 29 of 2001", "da", day, "the day 29 is outside 1 to 28 (§D.2.1)"},
		{"a day of 0", "dt", with(func(c *tDateTime) { c.Day = 0 }), "the day 0 is outside 1 to 31 (§D.2.1)"},
		{"an hour of 24", "dt", with(func(c *tDateTime) { c.Hour = 24 }), "the hour 24 is outside 0 to 23 (§D.2.1)"},
		{"a minute of 60", "dt", with(func(c *tDateTime) { c.Minute = 60 }), "the minute 60 is outside 0 to 59 (§D.2.1)"},
		{"a second of 60", "dt", with(func(c *tDateTime) { c.Second = *big.NewRat(60, 1) }), "the second 60 is outside 0 to 60, 60 excluded (§D.2.1)"},
		{"a negative second", "dt", with(func(c *tDateTime) { c.Second = *big.NewRat(-1, 2) }), "the second -1/2 is outside 0 to 60, 60 excluded (§D.2.1)"},
		{"a second outside the decimal value space", "dt", with(func(c *tDateTime) { c.Second = *big.NewRat(1, 3) }), "the second 1/3: its denominator in lowest terms is no 2^m·5^n, so it lies outside the decimal value space (Datatypes §3.3.3.1)"},
		{"an offset beyond 840", "dt", with(func(c *tDateTime) { c.TimezoneOffset = offset(841) }), "the timezoneOffset 841 is outside -840 to 840 minutes (§D.2.1)"},
		{"an hour before an offset", "dt", with(func(c *tDateTime) { c.Hour, c.TimezoneOffset = 24, offset(-841) }), "the hour 24 is outside"},
		{"a dateTimeStamp offset below -840", "dts", tDateTimeStamp{Year: bigYear("1"), Month: 1, Day: 1, TimezoneOffset: -841}, "the timezoneOffset -841 is outside"},
		{"a time second outside the decimal value space", "tm", tTime{Second: *big.NewRat(1, 3)}, "the second 1/3: its denominator"},
		{"a time offset", "tm", tTime{TimezoneOffset: offset(900)}, "the timezoneOffset 900 is outside"},
		{"a date offset", "da", tDate{Year: bigYear("1"), Month: 1, Day: 1, TimezoneOffset: offset(-900)}, "the timezoneOffset -900 is outside"},
		{"a gYearMonth month", "ym", tGYearMonth{Month: 0}, "the month 0 is outside 1 to 12"},
		{"a gYearMonth offset", "ym", tGYearMonth{Month: 1, TimezoneOffset: offset(841)}, "the timezoneOffset 841 is outside"},
		{"a gYear offset", "gy", tGYear{TimezoneOffset: offset(841)}, "the timezoneOffset 841 is outside"},
		{"a gMonthDay of February 30", "md", tGMonthDay{Month: 2, Day: 30}, "the day 30 is outside 1 to 29 (§D.2.1)"},
		{"a gMonthDay of April 31", "md", tGMonthDay{Month: 4, Day: 31}, "the day 31 is outside 1 to 30 (§D.2.1)"},
		{"a gMonthDay month", "md", tGMonthDay{Month: 13, Day: 1}, "the month 13 is outside 1 to 12"},
		{"a gMonthDay offset", "md", tGMonthDay{Month: 1, Day: 1, TimezoneOffset: offset(841)}, "the timezoneOffset 841 is outside"},
		{"a gDay of 32", "gd", tGDay{Day: 32}, "the day 32 is outside 1 to 31 (§D.2.1)"},
		{"a gDay of 0", "gd", tGDay{}, "the day 0 is outside 1 to 31 (§D.2.1)"},
		{"a gDay offset", "gd", tGDay{Day: 1, TimezoneOffset: offset(841)}, "the timezoneOffset 841 is outside"},
		{"a gMonth of 13", "gm", tGMonth{Month: 13}, "the month 13 is outside 1 to 12 (§D.2.1)"},
		{"a gMonth offset", "gm", tGMonth{Month: 1, TimezoneOffset: offset(841)}, "the timezoneOffset 841 is outside"},
		{"a duration of opposite signs", "du", tDuration{Months: *big.NewInt(1), Seconds: *big.NewRat(-1, 1)}, "the Months 1 and the Seconds -1 are of opposite signs (§3.3.6.1)"},
		{"a duration's seconds outside the decimal value space", "du", tDuration{Seconds: *big.NewRat(1, 3)}, "the Seconds 1/3: its denominator in lowest terms is no 2^m·5^n"},
		{"a dayTimeDuration's seconds outside the decimal value space", "dtd", tDayTimeDuration{Seconds: *big.NewRat(-1, 3)}, "the Seconds -1/3: its denominator"},
	} {
		t.Run(row.name, func(t *testing.T) {
			for _, backend := range []value.Backend{strict.New(), overPrimitive(row.elem)} {
				err := appendValue(t, schema, backend, row.elem, row.v)
				invalidValue(t, err, row.want)
				noRule(t, err)
				typ := reflect.TypeOf(row.v).Name()
				if !strings.HasPrefix(err.Error(), "codec: Append: "+typ+" holds {") || !strings.Contains(err.Error(), "}, of simple type xs:") {
					t.Errorf("error %q does not open with the path %s, the value and its type", err, typ)
				}
			}
		})
	}
	t.Run("the value shown", func(t *testing.T) {
		err := appendValue(t, schema, strict.New(), "da", day)
		const want = "codec: Append: tDate holds {Year 2001, Month 2, Day 29, TimezoneOffset nil}, of simple type xs:date, no value of its datatype: the day 29 is outside 1 to 28 (§D.2.1)"
		if err == nil || err.Error() != want {
			t.Errorf("error %v, want %q", err, want)
		}
	})
	t.Run("a list item", func(t *testing.T) {
		err := appendValue(t, schema, strict.New(), "dl", dateList{{Year: bigYear("1"), Month: 1, Day: 1}, {Month: 13, Day: 1}})
		invalidValue(t, err, "the month 13 is outside 1 to 12")
		if !strings.HasPrefix(err.Error(), "codec: Append: dateList[1] holds {") {
			t.Errorf("error %q does not open with the item's path", err)
		}
	})
}

// temporalPrimitives names, by encodeTemporalSchema's document element, the
// local name of its type's primitive.
var temporalPrimitives = map[string]string{
	"dt": "dateTime", "dts": "dateTime", "tm": "time", "da": "date", "ym": "gYearMonth", "gy": "gYear",
	"md": "gMonthDay", "gd": "gDay", "gm": "gMonth", "du": "duration", "dtd": "duration", "dl": "date",
}

// overPrimitive is strict's backend with the mapping of the primitive of
// elem's type (temporalPrimitives) rejecting every lexical and rendering no
// form.
func overPrimitive(elem string) value.Backend {
	return overDatatype(temporalPrimitives[elem], func(value.Mapping) value.Mapping {
		return value.Mapping{
			Parse:     func(string, value.Context) (value.Value, error) { return nil, errors.New("rejected") },
			Canonical: func(value.Value) (string, error) { return "", errors.New("no canonical form") },
		}
	})
}

// appendValue is appendChecked's error for v, of one of the T
// encodeTemporalSchema's document elements take, as the document element
// elem.
func appendValue(t *testing.T, schema *xsd.Schema, backend value.Backend, elem string, v any) error {
	t.Helper()
	var err error
	switch v := v.(type) {
	case tDateTime:
		_, err = appendChecked(t, nil, schema, backend, sname(elem), v)
	case tDateTimeStamp:
		_, err = appendChecked(t, nil, schema, backend, sname(elem), v)
	case tTime:
		_, err = appendChecked(t, nil, schema, backend, sname(elem), v)
	case tDate:
		_, err = appendChecked(t, nil, schema, backend, sname(elem), v)
	case tGYearMonth:
		_, err = appendChecked(t, nil, schema, backend, sname(elem), v)
	case tGYear:
		_, err = appendChecked(t, nil, schema, backend, sname(elem), v)
	case tGMonthDay:
		_, err = appendChecked(t, nil, schema, backend, sname(elem), v)
	case tGDay:
		_, err = appendChecked(t, nil, schema, backend, sname(elem), v)
	case tGMonth:
		_, err = appendChecked(t, nil, schema, backend, sname(elem), v)
	case tDuration:
		_, err = appendChecked(t, nil, schema, backend, sname(elem), v)
	case tDayTimeDuration:
		_, err = appendChecked(t, nil, schema, backend, sname(elem), v)
	case tYearMonthDuration:
		_, err = appendChecked(t, nil, schema, backend, sname(elem), v)
	case dateList:
		_, err = appendChecked(t, nil, schema, backend, sname(elem), v)
	default:
		t.Fatalf("appendValue holds no %T", v)
	}
	return err
}

// TestAppendTemporalInvalid pins items 6 and 7 of Append's doc for the date,
// time and duration builtins: a value a bound or explicitTimezone rejects is
// item 6 carrying the facet's rule; a value whose canonical representation a
// pattern rejects is item 7 where it has another lexical, a dateTime always,
// a date or gDay only where its timezoneOffset is 0 or a date's year is 0,
// as an atom and in a list, and item 6, carrying cvc-pattern-valid, where it
// has none; and the zero yearMonthDuration, which strict's Canonical refuses
// (§3.4.26.1's note), is item 7.
func TestAppendTemporalInvalid(t *testing.T) {
	schema := schemaOf(t, encodeTemporalSchema)
	for _, row := range []struct {
		name, elem string
		v          any
		rule       xsderr.Rule
	}{
		{"a bound", "early", tDate{Year: bigYear("2001"), Month: 1, Day: 1}, "cvc-maxInclusive-valid"},
		{"explicitTimezone required", "zoned", tDate{Year: bigYear("2001"), Month: 1, Day: 1}, "cvc-explicitTimezone-valid"},
		{"a pattern on a date with no offset", "plusZero", tDate{Year: bigYear("2001"), Month: 1, Day: 1}, "cvc-pattern-valid"},
		{"a pattern on a date at +05:00", "plusZero", tDate{Year: bigYear("2001"), Month: 1, Day: 1, TimezoneOffset: offset(300)}, "cvc-pattern-valid"},
		{"a pattern on a date of year 1", "minusYear", tDate{Year: bigYear("1"), Month: 1, Day: 1}, "cvc-pattern-valid"},
		{"a pattern on a gDay with no offset", "plusZeroDay", tGDay{Day: 1}, "cvc-pattern-valid"},
		{"a pattern on a list of dates with no offset 0", "plusZeroDates", dateList{{Year: bigYear("1"), Month: 1, Day: 1}, {Year: bigYear("2"), Month: 1, Day: 1, TimezoneOffset: offset(60)}}, "cvc-pattern-valid"},
	} {
		t.Run(row.name, func(t *testing.T) {
			err := appendValue(t, schema, strict.New(), row.elem, row.v)
			invalidValue(t, err, "{urn:s}"+row.elem)
			if typ := reflect.TypeOf(row.v).Name(); !strings.HasPrefix(err.Error(), "codec: Append: "+typ+" holds ") {
				t.Errorf("error %q does not open with the path %s", err, typ)
			}
			if rule, _ := xsderr.RuleOf(err); rule != row.rule {
				t.Errorf("rule %q, want %q", rule, row.rule)
			}
		})
	}
	for _, row := range []struct {
		name, elem string
		v          any
		at         string
	}{
		{"a pattern on a dateTime", "pointed", tDateTime{Year: bigYear("2001"), Month: 1, Day: 1}, `at tDateTime holding 2001-01-01T00:00:00: a pattern facet rejects its canonical representation "2001-01-01T00:00:00"`},
		{"a pattern on a date at Z", "plusZero", tDate{Year: bigYear("2001"), Month: 1, Day: 1, TimezoneOffset: offset(0)}, "at tDate holding 2001-01-01Z: a pattern facet"},
		{"a pattern on a date of year 0", "minusYear", tDate{Month: 1, Day: 1}, "at tDate holding 0000-01-01: a pattern facet"},
		{"a pattern on a gDay at Z", "plusZeroDay", tGDay{Day: 1, TimezoneOffset: offset(0)}, "at tGDay holding ---01Z: a pattern facet"},
		{"a pattern on a list holding a date at Z", "plusZeroDates", dateList{{Year: bigYear("1"), Month: 1, Day: 1}, {Year: bigYear("2"), Month: 1, Day: 1, TimezoneOffset: offset(0)}}, "at dateList holding [0001-01-01 0002-01-01Z]: a pattern facet"},
		{"the zero yearMonthDuration", "ymd", tYearMonthDuration{}, "at tYearMonthDuration holding P0M: yearMonthDuration: zero value has no canonical representation"},
	} {
		t.Run(row.name, func(t *testing.T) {
			err := appendValue(t, schema, strict.New(), row.elem, row.v)
			_ = asDecline(t, err, "codec does not encode a value with no canonical representation: simple type ")
			holds(t, err, row.at)
		})
	}
}

// TestAppendTemporalFaults pins the backend faults of Append's doc for the
// date, time and duration builtins: no mapping of the primitive, read from
// the type, a dateTimeStamp's xs:dateTime and a dayTimeDuration's
// xs:duration; the primitive's Parse rejecting the canonical spelling; and a
// Canonical rendering other text, each a plain error opening with the value's
// Go path.
func TestAppendTemporalFaults(t *testing.T) {
	schema := schemaOf(t, encodeTemporalSchema)
	err := appendValue(t, schema, noMappingOf{xsd.QName{Space: xsd.XMLSchemaNS, Local: "dateTime"}}, "dts", tDateTimeStamp{Year: bigYear("1"), Month: 1, Day: 1})
	plainError(t, err, "codec: Append: tDateTimeStamp, of simple type xs:dateTimeStamp: the backend has no mapping of its primitive xs:dateTime")
	err = appendValue(t, schema, noMappingOf{xsd.QName{Space: xsd.XMLSchemaNS, Local: "duration"}}, "dtd", tDayTimeDuration{})
	plainError(t, err, "codec: Append: tDayTimeDuration, of simple type xs:dayTimeDuration: the backend has no mapping of its primitive xs:duration")
	rejecting := overDatatype("date", func(base value.Mapping) value.Mapping {
		return value.Mapping{Parse: func(string, value.Context) (value.Value, error) { return nil, errors.New("rejected") }, Canonical: base.Canonical}
	})
	err = appendValue(t, schema, rejecting, "da", tDate{Year: bigYear("1"), Month: 1, Day: 1})
	plainError(t, err, "codec: Append: tDate holds 0001-01-01, of simple type xs:date, whose canonical spelling the backend's Parse of xs:date rejects: rejected")
	offsetCanonical := overDatatype("date", func(base value.Mapping) value.Mapping {
		return value.Mapping{Parse: base.Parse, Canonical: func(v value.Value) (string, error) {
			c, err := base.Canonical(v)
			return strings.TrimSuffix(c, "Z") + "+00:00", err
		}}
	})
	err = appendValue(t, schema, offsetCanonical, "da", tDate{Year: bigYear("1"), Month: 1, Day: 1, TimezoneOffset: offset(0)})
	plainError(t, err, `codec: Append: tDate holds 0001-01-01Z, of simple type xs:date, which the backend's Canonical renders "0001-01-01+00:00", other text than its canonical representation (Datatypes Appendix E, the canonical mapping of xs:date)`)
}

// noMappingOf is strict's backend, but that it maps no name.
type noMappingOf struct{ name xsd.QName }

// Mapping is strict's, but for name, which it does not map.
func (n noMappingOf) Mapping(typ xsd.QName) (value.Mapping, bool) {
	if typ == n.name {
		return value.Mapping{}, false
	}
	return strict.New().Mapping(typ)
}

// encodeBinarySchema declares, in urn:s, document elements of xs:hexBinary,
// xs:base64Binary, xs:anyURI and xs:float, an attribute of xs:anyURI, lists
// over xs:float and xs:hexBinary, and restrictions of each by a pattern, a
// length, a maxLength or a bound.
const encodeBinarySchema = `
  <xs:simpleType name="hexCD"><xs:restriction base="xs:hexBinary"><xs:pattern value="CD"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="hexTwo"><xs:restriction base="xs:hexBinary"><xs:length value="2"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="b64Foo"><xs:restriction base="xs:base64Binary"><xs:pattern value="Zm9v"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="uriShort"><xs:restriction base="xs:anyURI"><xs:maxLength value="3"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="floatOne"><xs:restriction base="xs:float"><xs:pattern value="1"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="floatNonPositive"><xs:restriction base="xs:float"><xs:maxInclusive value="0"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="floats"><xs:list itemType="xs:float"/></xs:simpleType>
  <xs:simpleType name="hexes"><xs:list itemType="xs:hexBinary"/></xs:simpleType>
  <xs:simpleType name="floatOnes"><xs:restriction base="s:floats"><xs:pattern value="1"/></xs:restriction></xs:simpleType>
  <xs:element name="hb" type="xs:hexBinary"/>
  <xs:element name="bb" type="xs:base64Binary"/>
  <xs:element name="u" type="xs:anyURI"/>
  <xs:element name="ua"><xs:complexType><xs:attribute name="a" type="xs:anyURI" use="required"/></xs:complexType></xs:element>
  <xs:element name="f" type="xs:float"/>
  <xs:element name="fl" type="s:floats"/>
  <xs:element name="hl" type="s:hexes"/>
  <xs:element name="hcd" type="s:hexCD"/>
  <xs:element name="htwo" type="s:hexTwo"/>
  <xs:element name="bfoo" type="s:b64Foo"/>
  <xs:element name="ushort" type="s:uriShort"/>
  <xs:element name="fone" type="s:floatOne"/>
  <xs:element name="fnp" type="s:floatNonPositive"/>
  <xs:element name="flp" type="s:floatOnes"/>`

// The Go types the rows below hold encodeBinarySchema's values in, beside
// binary_test.go's binaryRoot and binaryListRoot: ua's, u's, f's and fl's.
type (
	uriAttr struct{ A string }
	uriRoot string
	fltRoot float32
	fltList []float32
)

// sameFloat reports whether a and b are one float32, every NaN one.
func sameFloat[T ~float32](a, b T) bool {
	return math.Float32bits(float32(a)) == math.Float32bits(float32(b)) || a != a && b != b
}

// sameBytes is bytes.Equal over T, so that nil and empty are one value.
func sameBytes[T ~[]byte](a, b T) bool { return bytes.Equal(a, b) }

// TestAppendBinaryAndURI pins the canonical forms of xs:hexBinary, two
// upper-case hex digits per octet (f-hexBinaryCanonical); of xs:base64Binary,
// padded Base64 with no white space (Canonical-base64Binary); and of
// xs:anyURI, the value's own characters, neither normalized nor
// percent-escaped (§3.3.17.2's identity mapping); each in an element, the
// empty value as an empty-element tag, an xs:anyURI attribute and a list of
// xs:hexBinary too, accepted by validate and decoded back equal.
func TestAppendBinaryAndURI(t *testing.T) {
	schema := schemaOf(t, encodeBinarySchema)
	for _, row := range []struct {
		elem string
		v    binaryRoot
		want string
	}{
		{"hb", binaryRoot{0x0F, 0xB7}, `<hb xmlns="urn:s">0FB7</hb>`},
		{"hb", binaryRoot{0xab, 0xcd, 0x09}, `<hb xmlns="urn:s">ABCD09</hb>`},
		{"hb", binaryRoot{}, `<hb xmlns="urn:s"/>`},
		{"bb", binaryRoot("foo"), `<bb xmlns="urn:s">Zm9v</bb>`},
		{"bb", binaryRoot{0xff}, `<bb xmlns="urn:s">/w==</bb>`},
		{"bb", binaryRoot{0xff, 0xfe}, `<bb xmlns="urn:s">//4=</bb>`},
		{"bb", binaryRoot("a value of more than 57 octets, which RFC 2045 would break into lines"), `<bb xmlns="urn:s">` + base64.StdEncoding.EncodeToString([]byte("a value of more than 57 octets, which RFC 2045 would break into lines")) + `</bb>`},
		{"bb", nil, `<bb xmlns="urn:s"/>`},
	} {
		t.Run(row.elem+" "+row.want, func(t *testing.T) {
			roundTrip(t, schema, sname(row.elem), row.v, row.want, sameBytes[binaryRoot])
		})
	}
	for _, s := range []string{"http://example.com/a%20b?q=é#Frag", "HTTP://Example.COM/./a/../b", "", "a b", "ü"} {
		t.Run("anyURI "+strconv.Quote(s), func(t *testing.T) {
			want := `<u xmlns="urn:s">` + s + `</u>`
			if s == "" {
				want = `<u xmlns="urn:s"/>`
			}
			roundTrip(t, schema, sname("u"), uriRoot(s), want, deepEqual[uriRoot])
		})
	}
	t.Run("anyURI attribute", func(t *testing.T) {
		roundTrip(t, schema, sname("ua"), uriAttr{A: "#x y"}, `<ua xmlns="urn:s" a="#x y"/>`, deepEqual[uriAttr])
	})
	t.Run("hexBinary list", func(t *testing.T) {
		roundTrip(t, schema, sname("hl"), binaryListRoot{{0xab}, {0x01, 0x02}}, `<hl xmlns="urn:s">AB 0102</hl>`, deepEqual[binaryListRoot])
	})
}

// TestAppendSpecial pins the special types, xs:anySimpleType and
// xs:anyAtomicType (Datatypes §2.4.2 dt-special), written as the string each
// holds, unchanged under either, for a document element of each, an
// attribute use of xs:anyAtomicType and one whose declaration has no type, so
// xs:anySimpleType (Structures §3.2.2.2 dcl.att.local), holding TAB, LF and
// CR, simple content extending xs:anySimpleType and an <element ref> to a
// head of it: each written as shown, accepted by validate and decoded back
// equal. The xs:anyAtomicType document element's row fails under a writer
// that collapses its white space. A string holding U+0000 under either is
// item 6 of Append's doc, nothing written.
func TestAppendSpecial(t *testing.T) {
	t.Run("document element of xs:anySimpleType", func(t *testing.T) {
		schema := schemaOf(t, `<xs:element name="r" type="xs:anySimpleType"/>`)
		roundTrip(t, schema, sname("r"), spR(spSpaced), `<r xmlns="urn:s">`+spSpaced+`</r>`, deepEqual[spR])
	})
	t.Run("document element of xs:anyAtomicType", func(t *testing.T) {
		schema := schemaOf(t, `<xs:element name="r" type="xs:anyAtomicType"/>`)
		roundTrip(t, schema, sname("r"), spR(spSpaced), `<r xmlns="urn:s">`+spSpaced+`</r>`, deepEqual[spR])
	})
	t.Run("attribute use of xs:anyAtomicType", func(t *testing.T) {
		schema := schemaOf(t, root(`<xs:attribute name="k" type="xs:anyAtomicType"/>`))
		roundTrip(t, schema, sname("r"), spAttr{K: new(" 1.0  x ")}, `<r xmlns="urn:s" k=" 1.0  x "/>`, deepEqual[spAttr])
	})
	t.Run("attribute use of no type", func(t *testing.T) {
		schema := schemaOf(t, root(`<xs:attribute name="k"/>`))
		roundTrip(t, schema, sname("r"), spAttr{K: new(" x\ty\nz\r ")}, `<r xmlns="urn:s" k=" x&#x9;y&#xA;z&#xD; "/>`, deepEqual[spAttr])
	})
	t.Run("simple content extending xs:anySimpleType", func(t *testing.T) {
		schema := schemaOf(t, root(`<xs:simpleContent><xs:extension base="xs:anySimpleType"/></xs:simpleContent>`))
		roundTrip(t, schema, sname("r"), spValue{Value: spSpaced}, `<r xmlns="urn:s">`+spSpaced+`</r>`, deepEqual[spValue])
	})
	t.Run("element ref to a head of xs:anySimpleType", func(t *testing.T) {
		schema := schemaOf(t, `<xs:element name="h" type="xs:anySimpleType"/>`+sequenceOf(`<xs:element ref="s:h"/><xs:element name="a" type="xs:anyAtomicType"/>`))
		roundTrip(t, schema, sname("r"), spRef{H: spSpaced, A: "\r"}, `<r xmlns="urn:s"><h>`+spSpaced+`</h><a>&#xD;</a></r>`, deepEqual[spRef])
	})
	for _, typ := range []string{"anySimpleType", "anyAtomicType"} {
		t.Run("U+0000 under xs:"+typ, func(t *testing.T) {
			schema := schemaOf(t, `<xs:element name="r" type="xs:`+typ+`"/>`)
			got, err := appendChecked(t, []byte("pre"), schema, strict.New(), sname("r"), spR("a\x00"))
			invalidValue(t, err, `codec: Append: spR holds "a\x00", of simple type xs:`+typ+`, whose byte offset 1 starts no XML 1.0 Char`)
			if string(got) != "pre" {
				t.Errorf("Append wrote %q", got)
			}
		})
	}
}

// TestAppendFloat pins xs:float down double's path at 32 bits: each value,
// the infinities, NaN, the zeros, the largest finite float32 and its
// smallest subnormal among them, is written as the backend's Canonical
// renders it and decoded back to the same float32 by its bits, every NaN
// equal. A list of xs:float takes the same route. With mapsBack reading at 64
// bits, the rows 0.1, 3.4028235e+38, 1e-45 and the list fail.
func TestAppendFloat(t *testing.T) {
	schema := schemaOf(t, encodeBinarySchema)
	for _, f := range []float32{1, 0.1, -2.5, float32(math.Inf(1)), float32(math.Inf(-1)), float32(math.NaN()), float32(math.Copysign(0, -1)), 0, math.MaxFloat32, math.SmallestNonzeroFloat32, 16777216} {
		t.Run(strconv.FormatFloat(float64(f), 'g', -1, 32), func(t *testing.T) {
			roundTrip(t, schema, sname("f"), fltRoot(f), "", sameFloat[fltRoot])
		})
	}
	t.Run("list", func(t *testing.T) {
		roundTrip(t, schema, sname("fl"), fltList{0.1, float32(math.Inf(-1)), 3}, "", func(a, b fltList) bool {
			return slices.EqualFunc(a, b, sameFloat[float32])
		})
	})
}

// TestAppendBinaryInvalid pins items 6 and 7 of Append's doc for xs:hexBinary,
// xs:base64Binary, xs:float and xs:anyURI: a pattern's rejection of an
// xs:hexBinary value is item 7 where one of its octets holds a hex digit A to
// F and item 6 where none does; of an xs:base64Binary value item 7 but for
// the empty value, item 6; of an xs:float value, or a list of xs:float, item
// 7 but for NaN, item 6; a length, maxLength or bound rejection, an xs:anyURI
// that is not its own whiteSpace-normalized form or holds no XML 1.0 Char,
// and an empty xs:hexBinary list item, item 6. The rows "a hexBinary pattern,
// no letter" and "a base64Binary pattern, the empty value" fail where
// otherLexicalOf answers true for xs:hexBinary and xs:base64Binary
// respectively, and "an xs:float list's pattern" where it answers false for
// xs:float. Where hexLetter reads only the low nibble, "a hexBinary pattern,
// a high-nibble letter" fails; only the high nibble, "a low-nibble letter"
// and "a letter after the first octet"; only the first octet, "a letter after
// the first octet".
func TestAppendBinaryInvalid(t *testing.T) {
	schema := schemaOf(t, encodeBinarySchema)
	for _, row := range []struct {
		name, elem string
		v          binaryRoot
		want       string
		rule       xsderr.Rule
	}{
		{"a hexBinary pattern, no letter", "hcd", binaryRoot{0x12}, "codec: Append: binaryRoot holds 12, which simple type {urn:s}hexCD", "cvc-pattern-valid"},
		{"a base64Binary pattern, the empty value", "bfoo", nil, "codec: Append: binaryRoot holds , which simple type {urn:s}b64Foo", "cvc-pattern-valid"},
		{"a hexBinary length", "htwo", binaryRoot{0xAB}, "codec: Append: binaryRoot holds AB, which simple type {urn:s}hexTwo", "cvc-length-valid"},
	} {
		t.Run(row.name, func(t *testing.T) {
			_, err := appendChecked(t, nil, schema, strict.New(), sname(row.elem), row.v)
			invalidValue(t, err)
			if !strings.HasPrefix(err.Error(), row.want) {
				t.Errorf("error %q, want the prefix %q", err, row.want)
			}
			if rule, _ := xsderr.RuleOf(err); rule != row.rule {
				t.Errorf("rule %q, want %s", rule, row.rule)
			}
		})
	}
	for _, row := range []struct {
		name, elem string
		v          binaryRoot
		want       string
	}{
		{"a hexBinary pattern, a letter", "hcd", binaryRoot{0xAB}, "codec does not encode a value with no canonical representation: simple type {urn:s}hexCD defined at s.xsd:"},
		{"a hexBinary pattern, a low-nibble letter", "hcd", binaryRoot{0x1A}, "codec does not encode a value with no canonical representation: simple type {urn:s}hexCD defined at s.xsd:"},
		{"a hexBinary pattern, a high-nibble letter", "hcd", binaryRoot{0xA1}, "codec does not encode a value with no canonical representation: simple type {urn:s}hexCD defined at s.xsd:"},
		{"a hexBinary pattern, a letter after the first octet", "hcd", binaryRoot{0x12, 0x3A}, "codec does not encode a value with no canonical representation: simple type {urn:s}hexCD defined at s.xsd:"},
		{"a base64Binary pattern, a non-empty value", "bfoo", binaryRoot("bar"), "codec does not encode a value with no canonical representation: simple type {urn:s}b64Foo defined at s.xsd:"},
	} {
		t.Run(row.name, func(t *testing.T) {
			_, err := appendChecked(t, nil, schema, strict.New(), sname(row.elem), row.v)
			_ = asDecline(t, err, row.want)
			holds(t, err, "a pattern facet rejects its canonical representation")
		})
	}
	t.Run("an xs:float pattern", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("fone"), fltRoot(1))
		_ = asDecline(t, err, "codec does not encode a value with no canonical representation: simple type {urn:s}floatOne defined at s.xsd:")
		holds(t, err, "at fltRoot holding 1: a pattern facet rejects its canonical representation")
	})
	t.Run("an xs:float NaN a pattern rejects", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("fone"), fltRoot(float32(math.NaN())))
		invalidValue(t, err)
		if !strings.HasPrefix(err.Error(), "codec: Append: fltRoot holds NaN, which simple type {urn:s}floatOne") {
			t.Errorf("error %q does not open with the path, value and type", err)
		}
	})
	t.Run("an xs:float list's pattern", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("flp"), fltList{1})
		_ = asDecline(t, err, "codec does not encode a value with no canonical representation: simple type {urn:s}floatOnes defined at s.xsd:")
		holds(t, err, "at fltList holding [1]: a pattern facet rejects its canonical representation")
	})
	t.Run("an xs:float list of NaN its pattern rejects", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("flp"), fltList{float32(math.NaN())})
		invalidValue(t, err)
		if !strings.HasPrefix(err.Error(), "codec: Append: fltList holds [NaN], written \"NaN\", which simple type {urn:s}floatOnes") {
			t.Errorf("error %q does not open with the path, value and type", err)
		}
	})
	t.Run("an xs:float a bound rejects", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("fnp"), fltRoot(1))
		invalidValue(t, err)
		if !strings.HasPrefix(err.Error(), "codec: Append: fltRoot holds 1, which simple type {urn:s}floatNonPositive") {
			t.Errorf("error %q does not open with the path, value and type", err)
		}
		if rule, _ := xsderr.RuleOf(err); rule != "cvc-maxInclusive-valid" {
			t.Errorf("rule %q, want cvc-maxInclusive-valid", rule)
		}
	})
	t.Run("an xs:anyURI a maxLength rejects", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("ushort"), uriRoot("abcd"))
		invalidValue(t, err)
		if !strings.HasPrefix(err.Error(), `codec: Append: uriRoot holds "abcd", which simple type {urn:s}uriShort`) {
			t.Errorf("error %q does not open with the path, value and type", err)
		}
		if rule, _ := xsderr.RuleOf(err); rule != "cvc-maxLength-valid" {
			t.Errorf("rule %q, want cvc-maxLength-valid", rule)
		}
	})
	t.Run("an xs:anyURI not collapsed", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("u"), uriRoot(" a"))
		invalidValue(t, err)
		if want := `codec: Append: uriRoot holds " a", of simple type xs:anyURI, which differs from its whiteSpace-normalized form "a"`; err.Error() != want {
			t.Errorf("error %q, want %q", err, want)
		}
		noRule(t, err)
	})
	t.Run("an xs:anyURI holding U+0000", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("u"), uriRoot("a\x00"))
		invalidValue(t, err)
		if want := `codec: Append: uriRoot holds "a\x00", of simple type xs:anyURI, whose byte offset 1 starts no XML 1.0 Char (Datatypes §3.3.1.1, §3.3.17.1, §3.2.1.2, §3.2.2.2)`; err.Error() != want {
			t.Errorf("error %q, want %q", err, want)
		}
		noRule(t, err)
	})
	t.Run("an empty xs:hexBinary list item", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("hl"), binaryListRoot{{0xab}, {}})
		invalidValue(t, err)
		if want := `codec: Append: binaryListRoot[1] holds "", of simple type {urn:s}hexes defined at s.xsd:9:3, through the list type's {item type definition} xs:hexBinary, which is empty or holds white space, so no list item's lexical representation spells it (Datatypes §2.4.1.2)`; err.Error() != want {
			t.Errorf("error %q, want %q", err, want)
		}
		noRule(t, err)
	})
}

// nestedEncodeSchema declares, in urn:s, document elements whose element
// particles are of a complex type, which "# What encodes" admits: kid, named,
// with element-only content and a required attribute; ch, named, whose
// content is an emptiable choice, a struct holding its sealed interface, and
// hard, named, a bare sealed interface over a choice none of whose
// alternatives is emptiable; r, a sequence of kid once (n), optional (no) and
// repeated (nr), of an anonymous type once (a), optional (ao) and repeated
// (ar), of hard once (ci), optional (co) and repeated (cr), and of ch once
// (ce); alt, a choice of kid (s), of kid repeated (ss), of hard (i) and of
// hard repeated (ii); tree, of node, whose particles o and p hold node itself,
// optional and repeated; rec, of c1, ctZ003's shape
// (msData/complexType/ctZ003.xsd), a choice of e1, of c1, occurring 0 to 10
// times; lp, of loop, a bare sealed interface over a choice of z, a string,
// and l, of loop itself, repeated; and grp, the group g, whose one particle
// gc, optional, has an anonymous type holding g again, keyed by its
// groupPath.
const nestedEncodeSchema = `
  <xs:complexType name="kid">
    <xs:sequence><xs:element name="v" type="xs:string"/></xs:sequence>
    <xs:attribute name="k" type="xs:string" use="required"/>
  </xs:complexType>
  <xs:complexType name="ch">
    <xs:choice><xs:element name="x" type="xs:string"/><xs:element name="y" type="xs:double" minOccurs="0"/></xs:choice>
  </xs:complexType>
  <xs:complexType name="hard">
    <xs:choice><xs:element name="x" type="xs:string"/><xs:element name="y" type="xs:double"/></xs:choice>
  </xs:complexType>
  <xs:element name="r">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="n" type="s:kid"/>
        <xs:element name="no" type="s:kid" minOccurs="0"/>
        <xs:element name="nr" type="s:kid" minOccurs="0" maxOccurs="unbounded"/>
        <xs:element name="a"><xs:complexType><xs:sequence><xs:element name="w" type="xs:double"/></xs:sequence></xs:complexType></xs:element>
        <xs:element name="ao" minOccurs="0"><xs:complexType><xs:sequence><xs:element name="w" type="xs:double"/></xs:sequence></xs:complexType></xs:element>
        <xs:element name="ar" maxOccurs="2"><xs:complexType><xs:sequence><xs:element name="w" type="xs:double"/></xs:sequence></xs:complexType></xs:element>
        <xs:element name="ci" type="s:hard"/>
        <xs:element name="co" type="s:hard" minOccurs="0"/>
        <xs:element name="cr" type="s:hard" minOccurs="0" maxOccurs="2"/>
        <xs:element name="ce" type="s:ch"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:element name="alt">
    <xs:complexType>
      <xs:choice>
        <xs:element name="s" type="s:kid"/>
        <xs:element name="ss" type="s:kid" maxOccurs="2"/>
        <xs:element name="i" type="s:hard"/>
        <xs:element name="ii" type="s:hard" maxOccurs="2"/>
      </xs:choice>
    </xs:complexType>
  </xs:element>
  <xs:complexType name="node">
    <xs:sequence>
      <xs:element name="o" type="s:node" minOccurs="0"/>
      <xs:element name="p" type="s:node" minOccurs="0" maxOccurs="unbounded"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="tree" type="s:node"/>
  <xs:complexType name="c1">
    <xs:choice><xs:element name="e1" type="s:c1" minOccurs="0" maxOccurs="10"/></xs:choice>
  </xs:complexType>
  <xs:element name="rec" type="s:c1"/>
  <xs:complexType name="loop">
    <xs:choice><xs:element name="z" type="xs:string"/><xs:element name="l" type="s:loop" maxOccurs="2"/></xs:choice>
  </xs:complexType>
  <xs:element name="lp" type="s:loop"/>
  <xs:group name="g">
    <xs:sequence>
      <xs:element name="gc" minOccurs="0"><xs:complexType><xs:group ref="s:g"/></xs:complexType></xs:element>
    </xs:sequence>
  </xs:group>
  <xs:element name="grp"><xs:complexType><xs:group ref="s:g"/></xs:complexType></xs:element>`

// The T codegen emits for nestedEncodeSchema's document elements, in shape,
// which TestNestedEncodeTypesAreCodegens pins: neR for r, its anonymous
// types' structs all neW; neAlt and its branches for alt; neNode for tree and
// node; neC1 for rec and c1; neLoop and its branches for lp and loop; neGc for
// grp and gc.
type (
	neR struct {
		N  neKid
		No *neKid
		Nr []neKid
		A  neW
		Ao *neW
		Ar []neW
		Ci neHard
		Co neHard
		Cr []neHard
		Ce neCh
	}
	neKid      struct{ V, K string }
	neW        struct{ W float64 }
	neHard     interface{ isNeHard() }
	neHardX    struct{ X string }
	neHardY    struct{ Y float64 }
	neCh       struct{ Choice neChChoice }
	neChChoice interface{ isNeChChoice() }
	neChX      struct{ X string }
	neChY      struct{ Y float64 }
	neAlt      interface{ isNeAlt() }
	neAltS     struct{ S neKid }
	neAltSs    struct{ Ss []neKid }
	neAltI     struct{ I neHard }
	neAltIi    struct{ Ii []neHard }
	neNode     struct {
		O *neNode
		P []neNode
	}
	neC1       struct{ Choice neC1Choice }
	neC1Choice interface{ isNeC1Choice() }
	neC1E1     struct{ E1 []neC1 }
	neLoop     interface{ isNeLoop() }
	neLoopZ    struct{ Z string }
	neLoopL    struct{ L []neLoop }
	neGc       struct{ Gc *neGc }
)

func (neHardX) isNeHard()    {}
func (neHardY) isNeHard()    {}
func (neChX) isNeChChoice()  {}
func (neChY) isNeChChoice()  {}
func (neAltS) isNeAlt()      {}
func (neAltSs) isNeAlt()     {}
func (neAltI) isNeAlt()      {}
func (neAltIi) isNeAlt()     {}
func (neC1E1) isNeC1Choice() {}
func (neLoopZ) isNeLoop()    {}
func (neLoopL) isNeLoop()    {}

// nestedBranches is every branch set nestedEncodeSchema's interfaces take.
var nestedBranches = []Option{
	WithBranches[neHard](neHardX{}, neHardY{}),
	WithBranches[neChChoice](neChX{}, neChY{}),
	WithBranches[neAlt](neAltS{}, neAltSs{}, neAltI{}, neAltIi{}),
	WithBranches[neC1Choice](neC1E1{}),
	WithBranches[neLoop](neLoopZ{}, neLoopL{}),
}

// TestNestedEncodeTypesAreCodegens pins the types above to those codegen
// emits for nestedEncodeSchema, in shape.
func TestNestedEncodeTypesAreCodegens(t *testing.T) {
	gen := generated(t, schemaOf(t, nestedEncodeSchema))
	for _, decl := range []string{
		"type R struct {\n\tN  Kid\n\tNo *Kid\n\tNr []Kid\n\tA  A\n\tAo *Ao\n\tAr []Ar\n\tCi Hard\n\tCo Hard\n\tCr []Hard\n\tCe Ch\n}\n",
		"type Kid struct {\n\tV string\n\tK string\n}\n",
		"type A struct {\n\tW float64\n}\n",
		"type Ao struct {\n\tW float64\n}\n",
		"type Ar struct {\n\tW float64\n}\n",
		"type Hard interface {\n\tisHard()\n}\n",
		"type HardX struct {\n\tX string\n}\n",
		"type HardY struct {\n\tY float64\n}\n",
		"type Ch struct {\n\tChoice ChChoice\n}\n",
		"type ChChoiceX struct {\n\tX string\n}\n",
		"type ChChoiceY struct {\n\tY float64\n}\n",
		"type AltS struct {\n\tS Kid\n}\n",
		"type AltSs struct {\n\tSs []Kid\n}\n",
		"type AltI struct {\n\tI Hard\n}\n",
		"type AltIi struct {\n\tIi []Hard\n}\n",
		"type Tree struct {\n\tO *Node\n\tP []Node\n}\n",
		"type Node struct {\n\tO *Node\n\tP []Node\n}\n",
		"type Rec struct {\n\tChoice C1Choice\n}\n",
		"type C1ChoiceE1 struct {\n\tE1 []C1\n}\n",
		"type C1 struct {\n\tChoice C1Choice\n}\n",
		"type LpL struct {\n\tL []Loop\n}\n",
		"type LoopZ struct {\n\tZ string\n}\n",
		"type LoopL struct {\n\tL []Loop\n}\n",
		"type Grp struct {\n\tGc *Gc\n}\n",
		"type Gc struct {\n\tGc *Gc\n}\n",
	} {
		if !bytes.Contains(gen, []byte(decl)) {
			t.Errorf("codegen emits\n%s\nwhich does not declare\n%s", gen, decl)
		}
	}
}

// TestAppendComplexParticle pins the element particle of a complex type that
// "# What encodes" admits, its content assessed against its own type
// (cvc-complex-type, cvc-complex-content), in each shape item 6 of Decode's
// doc holds it to: a named and an anonymous type's struct S once, *S
// optional, present and absent, and []S repeated; a bare sealed interface I
// once, optional, nil meaning absent, and []I repeated; a struct holding an
// emptiable choice, its empty match an empty-element tag; a choice's
// alternative into its branch's field, S, []S, I and []I; and recursive
// types, a struct through *S and []S, ctZ003's c1 through []S, a bare
// interface through []I, and a model group definition's anonymous type
// through its groupPath, each walked once by item 4 and written to the
// value's finite depth. Each value is written, validate accepts it, and
// Decode returns it, a shared *S written at each of its sites.
func TestAppendComplexParticle(t *testing.T) {
	schema := schemaOf(t, nestedEncodeSchema)
	kid := func(v string) neKid { return neKid{V: v, K: "k" + v} }
	shared := &neNode{P: []neNode{{}}}
	for _, row := range []struct {
		name string
		elem string
		v    any
		want string
	}{
		{"each shape once, the optional ones absent", "r", neR{N: kid("1"), A: neW{W: 1}, Ar: []neW{{W: 2}}, Ci: neHardX{X: "x"}, Ce: neCh{}},
			`<r xmlns="urn:s"><n k="k1"><v>1</v></n><a><w>1.0E0</w></a><ar><w>2.0E0</w></ar><ci><x>x</x></ci><ce/></r>`},
		{"optional ones present, repeated ones of two", "r", neR{N: kid("1"), No: &neKid{V: "2", K: "k2"}, Nr: []neKid{kid("3"), kid("4")}, A: neW{W: 1}, Ao: &neW{W: 2},
			Ar: []neW{{W: 3}, {W: 4}}, Ci: neHardY{Y: 1}, Co: neHardX{X: "o"}, Cr: []neHard{neHardX{X: "r"}, neHardY{Y: 2}}, Ce: neCh{Choice: neChY{Y: 5}}}, ""},
		{"alternative S", "alt", neAlt(neAltS{S: kid("s")}), `<alt xmlns="urn:s"><s k="ks"><v>s</v></s></alt>`},
		{"alternative []S", "alt", neAlt(neAltSs{Ss: []neKid{kid("1"), kid("2")}}), ""},
		{"alternative I", "alt", neAlt(neAltI{I: neHardX{X: "i"}}), `<alt xmlns="urn:s"><i><x>i</x></i></alt>`},
		{"alternative []I", "alt", neAlt(neAltIi{Ii: []neHard{neHardY{Y: 1}, neHardX{X: "2"}}}), ""},
		{"a struct holding itself through *S and []S", "tree", neNode{O: &neNode{P: []neNode{{}, {O: &neNode{}}}}, P: []neNode{{}}},
			`<tree xmlns="urn:s"><o><p/><p><o/></p></o><p/></tree>`},
		{"a shared *S", "tree", neNode{O: shared, P: []neNode{{O: shared}}}, `<tree xmlns="urn:s"><o><p/></o><p><o><p/></o></p></tree>`},
		{"ctZ003's c1", "rec", neC1{Choice: neC1E1{E1: []neC1{{Choice: neC1E1{E1: []neC1{{}}}}, {}}}}, `<rec xmlns="urn:s"><e1><e1/></e1><e1/></rec>`},
		{"a bare interface holding itself through []I", "lp", neLoop(neLoopL{L: []neLoop{neLoopZ{Z: "a"}, neLoopL{L: []neLoop{neLoopZ{Z: "b"}}}}}),
			`<lp xmlns="urn:s"><l><z>a</z></l><l><l><z>b</z></l></l></lp>`},
		{"a group's anonymous type holding itself", "grp", neGc{Gc: &neGc{Gc: &neGc{}}}, `<grp xmlns="urn:s"><gc><gc/></gc></grp>`},
	} {
		t.Run(row.name, func(t *testing.T) {
			// The rows' Ts differ, and roundTrip takes each as its own T.
			switch v := row.v.(type) {
			case neR:
				roundTrip(t, schema, sname(row.elem), v, row.want, deepEqual[neR], nestedBranches...)
			case neAlt:
				roundTrip(t, schema, sname(row.elem), v, row.want, deepEqual[neAlt], nestedBranches...)
			case neNode:
				roundTrip(t, schema, sname(row.elem), v, row.want, deepEqual[neNode], nestedBranches...)
			case neC1:
				roundTrip(t, schema, sname(row.elem), v, row.want, deepEqual[neC1], nestedBranches...)
			case neLoop:
				roundTrip(t, schema, sname(row.elem), v, row.want, deepEqual[neLoop], nestedBranches...)
			case neGc:
				roundTrip(t, schema, sname(row.elem), v, row.want, deepEqual[neGc], nestedBranches...)
			default:
				t.Fatalf("row of %T", row.v)
			}
		})
	}
}

// TestAppendParticleTypeSharingElementName pins that a named complex type
// sharing the expanded name of a top-level element declaration is its own
// unit, apart from that element's: r, of an anonymous type, holds c of the
// complex type r, a sequence of v, an xs:string, and r's value is written,
// validate accepts it, and Decode returns it.
func TestAppendParticleTypeSharingElementName(t *testing.T) {
	schema := schemaOf(t, `<xs:complexType name="r"><xs:sequence><xs:element name="v" type="xs:string"/></xs:sequence></xs:complexType>`+
		`<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="c" type="s:r"/></xs:sequence></xs:complexType></xs:element>`)
	// namesake and its C are codegen's R and R2.
	type namesake struct{ C struct{ V string } }
	if gen := generated(t, schema); !bytes.Contains(gen, []byte("type R struct {\n\tC R2\n}\n")) || !bytes.Contains(gen, []byte("type R2 struct {\n\tV string\n}\n")) {
		t.Fatalf("codegen emits\n%s\nwhich does not declare R{C R2} and R2{V string}", gen)
	}
	v := namesake{}
	v.C.V = "x"
	roundTrip(t, schema, rootName, v, `<r xmlns="urn:s"><c><v>x</v></c></r>`, deepEqual[namesake])
}

// TestAppendComplexParticleInvalid pins item 6 of Append's doc at an element
// particle of complex type: the nil bare interface of a particle held at
// {min occurs} 1, whose choice no alternative of is emptiable; and a value
// that holds itself, through *S (v.O = &v), []S (s[0].P = s) and []I (s[0] a
// branch holding s), the error naming the Go path where the value is reached
// again and the one it is reached from. None carries an xsderr rule.
func TestAppendComplexParticleInvalid(t *testing.T) {
	schema := schemaOf(t, nestedEncodeSchema)
	t.Run("nil I of a particle held once", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("r"), neR{Ar: []neW{{}}}, nestedBranches...)
		invalidChoice(t, err, "codec: Append: neR.Ci is nil, the empty match, which the content choice of the {type definition} of element {urn:s}ci declared at ", "none of its 2 alternatives is emptiable")
	})
	t.Run("through *S", func(t *testing.T) {
		var v neNode
		v.O = &v
		_, err := appendChecked(t, nil, schema, strict.New(), sname("tree"), v, nestedBranches...)
		invalidChoice(t, err, "codec: Append: neNode.O.O is the value at neNode.O, whose element is still being written, a value that holds itself, which no finite document holds")
	})
	t.Run("through []S", func(t *testing.T) {
		s := make([]neNode, 1)
		s[0].P = s
		_, err := appendChecked(t, nil, schema, strict.New(), sname("tree"), neNode{P: s}, nestedBranches...)
		invalidChoice(t, err, "codec: Append: neNode.P[0].P[0] is the value at neNode.P[0], whose element is still being written")
	})
	t.Run("through []I", func(t *testing.T) {
		s := make([]neLoop, 1)
		s[0] = neLoopL{L: s}
		_, err := appendChecked(t, nil, schema, strict.New(), sname("lp"), neLoop(neLoopL{L: s}), nestedBranches...)
		invalidChoice(t, err, "codec: Append: neLoop.(neLoopL).L[0].(neLoopL).L[0] is the value at neLoop.(neLoopL).L[0], whose element is still being written")
	})
}

// encodeQNameSchema declares, in urn:s, document elements of xs:QName (q); of
// nt, an enumeration of xs:NOTATION over the notation png, gif being declared
// and not enumerated (n); of qe, an enumeration of xs:QName over s:x (qe); of
// unprefixed, a pattern admitting only an unprefixed lexical (up); of a list
// of xs:QName (l) and its restriction by the same pattern (ul); an element
// whose attribute a is of xs:QName (at); an element of that attribute and of
// repeated xs:QName children c (pc); and an element of two unqualified
// children in no namespace, c of xs:QName and p of prefixed, a pattern
// admitting only a prefixed lexical (nons).
const encodeQNameSchema = `<xs:notation name="png" public="image/png"/><xs:notation name="gif" public="image/gif"/>
  <xs:simpleType name="nt"><xs:restriction base="xs:NOTATION"><xs:enumeration value="s:png"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="qe"><xs:restriction base="xs:QName"><xs:enumeration value="s:x"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="unprefixed"><xs:restriction base="xs:QName"><xs:pattern value="[^:]+"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="prefixed"><xs:restriction base="xs:QName"><xs:pattern value="p\d+:.+"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="qnames"><xs:list itemType="xs:QName"/></xs:simpleType>
  <xs:simpleType name="unprefixedList"><xs:restriction base="s:qnames"><xs:pattern value="[^:]+( [^:]+)*"/></xs:restriction></xs:simpleType>
  <xs:element name="q" type="xs:QName"/>
  <xs:element name="n" type="s:nt"/>
  <xs:element name="qe" type="s:qe"/>
  <xs:element name="up" type="s:unprefixed"/>
  <xs:element name="l" type="s:qnames"/>
  <xs:element name="ul" type="s:unprefixedList"/>
  <xs:element name="at"><xs:complexType><xs:sequence/><xs:attribute name="a" type="xs:QName"/></xs:complexType></xs:element>
  <xs:element name="pc"><xs:complexType><xs:sequence><xs:element name="c" type="xs:QName" maxOccurs="unbounded"/></xs:sequence>
    <xs:attribute name="a" type="xs:QName"/></xs:complexType></xs:element>
  <xs:element name="nons"><xs:complexType><xs:sequence><xs:element name="c" type="xs:QName" form="unqualified"/>
    <xs:element name="p" type="s:prefixed" form="unqualified"/></xs:sequence></xs:complexType></xs:element>`

// qnameList, qnameAt, qnameParent and qnameNoNS are the types codegen emits
// for encodeQNameSchema's l, at, pc and nons, which item 5 of Append's doc
// holds them to.
type (
	qnameList   []tQName
	qnameAt     struct{ A *tQName }
	qnameParent struct {
		C []tQName
		A *tQName
	}
	qnameNoNS struct{ C, P tQName }
)

// TestAppendQNames pins how Append spells an xs:QName or xs:NOTATION value,
// as the package doc's "# What encodes" states, each document accepted by
// validate and decoded back equal: unprefixed in its element's own namespace,
// the absent one included; xml:Local for the XML namespace, undeclared; and
// otherwise under a prefix p1, p2, … declared on the start tag of the element
// it is written on, its namespace name escaped, reused where an ancestor's or
// an earlier value's start tag declares it in scope, and declared again by a
// sibling, whose scope the earlier sibling's declaration has left; in an
// attribute, an element, a list holding a prefixed and an unprefixed item,
// and an xs:NOTATION enumeration.
func TestAppendQNames(t *testing.T) {
	schema := schemaOf(t, encodeQNameSchema)
	for _, row := range []struct {
		name string
		v    tQName
		want string
	}{
		{"own namespace", tQName{Space: "urn:s", Local: "x"}, `<q xmlns="urn:s">x</q>`},
		{"another namespace", tQName{Space: "urn:o", Local: "y"}, `<q xmlns="urn:s" xmlns:p1="urn:o">p1:y</q>`},
		{"the XML namespace", tQName{Space: xmltree.XMLNamespaceURI, Local: "lang"}, `<q xmlns="urn:s">xml:lang</q>`},
		{"an escaped namespace name", tQName{Space: `urn:a&b"c`, Local: "y"}, `<q xmlns="urn:s" xmlns:p1="urn:a&amp;b&quot;c">p1:y</q>`},
	} {
		t.Run(row.name, func(t *testing.T) {
			roundTrip(t, schema, sname("q"), row.v, row.want, deepEqual[tQName])
		})
	}
	t.Run("NOTATION", func(t *testing.T) {
		roundTrip(t, schema, sname("n"), tQName{Space: "urn:s", Local: "png"}, `<n xmlns="urn:s">png</n>`, deepEqual[tQName])
	})
	t.Run("attribute", func(t *testing.T) {
		roundTrip(t, schema, sname("at"), qnameAt{A: &tQName{Space: "urn:o", Local: "z"}}, `<at xmlns="urn:s" xmlns:p1="urn:o" a="p1:z"/>`, deepEqual[qnameAt])
	})
	t.Run("parent and child", func(t *testing.T) {
		v := qnameParent{C: []tQName{{Space: "urn:o", Local: "b"}, {Space: "urn:t", Local: "c"}}, A: &tQName{Space: "urn:o", Local: "a"}}
		roundTrip(t, schema, sname("pc"), v, `<pc xmlns="urn:s" xmlns:p1="urn:o" a="p1:a"><c>p1:b</c><c xmlns:p2="urn:t">p2:c</c></pc>`, deepEqual[qnameParent])
	})
	t.Run("siblings", func(t *testing.T) {
		v := qnameParent{C: []tQName{{Space: "urn:t", Local: "a"}, {Space: "urn:t", Local: "b"}}}
		roundTrip(t, schema, sname("pc"), v, `<pc xmlns="urn:s"><c xmlns:p1="urn:t">p1:a</c><c xmlns:p1="urn:t">p1:b</c></pc>`, deepEqual[qnameParent])
	})
	t.Run("no namespace", func(t *testing.T) {
		v := qnameNoNS{C: tQName{Local: "x"}, P: tQName{Space: "urn:o", Local: "a"}}
		roundTrip(t, schema, sname("nons"), v, `<nons xmlns="urn:s"><c xmlns="">x</c><p xmlns="" xmlns:p1="urn:o">p1:a</p></nons>`, deepEqual[qnameNoNS])
	})
	t.Run("list, one prefixed", func(t *testing.T) {
		roundTrip(t, schema, sname("l"), qnameList{{Space: "urn:o", Local: "a"}, {Space: "urn:s", Local: "b"}}, `<l xmlns="urn:s" xmlns:p1="urn:o">p1:a b</l>`, deepEqual[qnameList])
	})
	t.Run("list, one prefix twice", func(t *testing.T) {
		roundTrip(t, schema, sname("l"), qnameList{{Space: "urn:o", Local: "a"}, {Space: "urn:o", Local: "b"}}, `<l xmlns="urn:s" xmlns:p1="urn:o">p1:a p1:b</l>`, deepEqual[qnameList])
	})
}

// TestAppendQNameInvalid pins items 6 and 7 of Append's doc and a backend
// fault for an xs:QName or xs:NOTATION value: a namespace name holding no XML
// 1.0 Char or the xmlns namespace, a Local the mapping's Parse rejects, a
// list item spelled as another expanded name, a value outside an enumeration
// of xs:QName or of xs:NOTATION, and a pattern's rejection of the spelling of
// a value of no namespace, item 6; a pattern's rejection of the spelling of a
// value of a namespace, an atom or a list, and a value of no namespace
// written where the default namespace is not absent, item 7 by
// constructQNameSpelling; and a backend with no xs:QName mapping, a plain
// error.
func TestAppendQNameInvalid(t *testing.T) {
	schema := schemaOf(t, encodeQNameSchema)
	for _, row := range []struct {
		name, elem string
		v          any
		want       string
		rule       xsderr.Rule
	}{
		{"a non-Char namespace name", "q", tQName{Space: "urn:\x00", Local: "x"}, "codec: Append: tQName holds {Space urn:\x00, Local x}, of simple type xs:QName, whose namespace name's byte offset 4 starts no XML 1.0 Char", ""},
		{"the xmlns namespace", "q", tQName{Space: "http://www.w3.org/2000/xmlns/", Local: "x"}, "codec: Append: tQName holds {Space http://www.w3.org/2000/xmlns/, Local x}, of simple type xs:QName, whose namespace name is the one the prefix xmlns is bound to", ""},
		{"a Local that is no NCName", "q", tQName{Space: "urn:s", Local: "1x"}, `codec: Append: tQName holds {Space urn:s, Local 1x}, of simple type xs:QName, written "1x", which the backend's Parse of xs:QName rejects`, "cvc-datatype-valid"},
		{"a Local spelling another name", "l", qnameList{{Space: "urn:o", Local: "a"}, {Space: "urn:s", Local: "p1:x"}}, `codec: Append: qnameList[1] holds {Space urn:s, Local p1:x}, of simple type {urn:s}qnames defined at s.xsd:6:3, through the list type's {item type definition} xs:QName, written "p1:x", which does not resolve back to it: it resolves to {Space urn:o, Local x}`, ""},
		{"a QName enumeration", "qe", tQName{Space: "urn:s", Local: "y"}, `codec: Append: tQName holds {Space urn:s, Local y}, written "y", which simple type {urn:s}qe`, "cvc-enumeration-valid"},
		{"a NOTATION enumeration", "n", tQName{Space: "urn:s", Local: "gif"}, `codec: Append: tQName holds {Space urn:s, Local gif}, written "gif", which simple type {urn:s}nt`, "cvc-enumeration-valid"},
		{"a pattern, no namespace", "nons", qnameNoNS{C: tQName{Local: "x"}, P: tQName{Local: "a"}}, `codec: Append: qnameNoNS.P holds {Space , Local a}, written "a", which simple type {urn:s}prefixed`, "cvc-pattern-valid"},
	} {
		t.Run(row.name, func(t *testing.T) {
			var err error
			switch v := row.v.(type) {
			case tQName:
				_, err = appendChecked(t, nil, schema, strict.New(), sname(row.elem), v)
			case qnameList:
				_, err = appendChecked(t, nil, schema, strict.New(), sname(row.elem), v)
			case qnameNoNS:
				_, err = appendChecked(t, nil, schema, strict.New(), sname(row.elem), v)
			}
			invalidValue(t, err)
			if !strings.HasPrefix(err.Error(), row.want) {
				t.Errorf("error %q, want the prefix %q", err, row.want)
			}
			if rule, _ := xsderr.RuleOf(err); rule != row.rule {
				t.Errorf("rule %q, want %q", rule, row.rule)
			}
		})
	}
	const spelling = "codec does not encode an xs:QName or xs:NOTATION value with no spelling Append writes where it is written: "
	t.Run("no namespace under a default", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("q"), tQName{Local: "x"})
		_ = asDecline(t, err, spelling+"simple type xs:QName, at tQName holding {Space , Local x}: its namespace name is absent where the default namespace in scope, its element's own, is urn:s")
	})
	t.Run("a pattern, prefixed", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("up"), tQName{Space: "urn:o", Local: "a"})
		_ = asDecline(t, err, spelling+"simple type {urn:s}unprefixed defined at s.xsd:")
		holds(t, err, `at tQName holding {Space urn:o, Local a}: a pattern facet rejects its spelling "p1:a"`)
	})
	t.Run("a list pattern", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("ul"), qnameList{{Space: "urn:s", Local: "a"}, {Space: "urn:o", Local: "b"}})
		_ = asDecline(t, err, spelling+"simple type {urn:s}unprefixedList defined at s.xsd:")
		holds(t, err, `a pattern facet rejects its spelling "a p1:b"`)
	})
	t.Run("no mapping of xs:QName", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, noMappingOf{xsd.QName{Space: xsd.XMLSchemaNS, Local: "QName"}}, sname("q"), tQName{Space: "urn:s", Local: "x"})
		plainError(t, err, "codec: Append: tQName, of simple type xs:QName: the backend has no mapping of its primitive xs:QName")
	})
}

// encodePDSchema declares, in urn:s, document elements of xs:precisionDecimal
// types: h, of xs:precisionDecimal itself; hl, a list of it; pa, a required
// attribute of it; pat, a restriction by the pattern \d+\.\d\d, which admits
// 3.00 but not 3, INF, -INF or NaN; max, a restriction by maxInclusive 5; and
// pl, a list of it restricted by the pattern NaN.
const encodePDSchema = `<xs:element name="h" type="xs:precisionDecimal"/>
<xs:element name="hl"><xs:simpleType><xs:list itemType="xs:precisionDecimal"/></xs:simpleType></xs:element>
<xs:element name="pa"><xs:complexType><xs:attribute name="a" type="xs:precisionDecimal" use="required"/></xs:complexType></xs:element>
<xs:element name="pat"><xs:simpleType><xs:restriction base="xs:precisionDecimal"><xs:pattern value="\d+\.\d\d"/></xs:restriction></xs:simpleType></xs:element>
<xs:element name="max"><xs:simpleType><xs:restriction base="xs:precisionDecimal"><xs:maxInclusive value="5"/></xs:restriction></xs:simpleType></xs:element>
<xs:element name="pl"><xs:simpleType><xs:restriction><xs:simpleType><xs:list itemType="xs:precisionDecimal"/></xs:simpleType><xs:pattern value="NaN"/></xs:restriction></xs:simpleType></xs:element>`

// pdNumerical returns the tPrecisionDecimal of the numerical arm of
// coefficient, scale and negative.
func pdNumerical(coefficient, scale int64, negative bool) tPrecisionDecimal {
	n := tPDNumerical{Negative: negative}
	n.Coefficient.SetInt64(coefficient)
	n.Scale.SetInt64(scale)
	return tPrecisionDecimal{n}
}

// samePD reports whether a and b, of a type over tPrecisionDecimal or a
// struct or slice of them, hold one value by arm, spellDecimal's, so that NaN
// equals NaN, which value equality never makes it (precisionDecimal §3.1).
func samePD[T any](a, b T) bool {
	return fmt.Sprint(pdSpelled(reflect.ValueOf(a))) == fmt.Sprint(pdSpelled(reflect.ValueOf(b)))
}

// pdSpelled spells v as samePD compares it: a slice by its items, a struct of
// one field Value by spellDecimal, and any other struct by its fields.
func pdSpelled(v reflect.Value) []string {
	if v.Kind() == reflect.Slice {
		var out []string
		for i := range v.Len() {
			out = append(out, pdSpelled(v.Index(i))...)
		}
		return out
	}
	if v.Type().Field(0).Name == "Value" {
		return []string{spellDecimal(v)}
	}
	var out []string
	for i := range v.NumField() {
		out = append(out, pdSpelled(v.Field(i))...)
	}
	return out
}

// TestAppendPrecisionDecimal pins xs:precisionDecimal's write: a finite value
// keeps its scale (3.00, precisionDecimal §6 f-precDecCanmap step 4), and
// INF, -INF and NaN take their roles from the WithPrecisionDecimalArms set,
// each validated and decoded back under the same set to the same arm; a list
// and an attribute write each value so. Under a set passing the infinities in
// each other's roles, the positiveInfinity arm writes -INF, the misuse
// WithPrecisionDecimalArms' doc states, and decodes back to that arm. With no
// set, CheckAppend and Append return item 5's ErrTypeMismatch.
func TestAppendPrecisionDecimal(t *testing.T) {
	schema := schemaOf(t, encodePDSchema)
	for _, row := range []struct {
		name string
		v    tPrecisionDecimal
		want string
	}{
		{"finite", pdNumerical(300, 2, false), `<h xmlns="urn:s">3.00</h>`},
		{"positiveInfinity", tPrecisionDecimal{tPDPositiveInfinity{}}, `<h xmlns="urn:s">INF</h>`},
		{"negativeInfinity", tPrecisionDecimal{tPDNegativeInfinity{}}, `<h xmlns="urn:s">-INF</h>`},
		{"notANumber", tPrecisionDecimal{tPDNotANumber{}}, `<h xmlns="urn:s">NaN</h>`},
	} {
		t.Run(row.name, func(t *testing.T) {
			_ = roundTrip(t, schema, sname("h"), row.v, row.want, samePD, pdArms)
		})
	}
	_ = roundTrip(t, schema, sname("hl"), tHl{pdNumerical(3, -2, false), {tPDNotANumber{}}, {tPDNegativeInfinity{}}}, `<hl xmlns="urn:s">3E2 NaN -INF</hl>`, samePD, pdArms)
	_ = roundTrip(t, schema, sname("pa"), struct{ A tPrecisionDecimal }{pdNumerical(15, 1, false)}, `<pa xmlns="urn:s" a="1.5"/>`, samePD, pdArms)

	swapped := WithPrecisionDecimalArms[tPDValue](tPDNumerical{}, tPDNegativeInfinity{}, tPDPositiveInfinity{}, tPDNotANumber{})
	_ = roundTrip(t, schema, sname("h"), tPrecisionDecimal{tPDPositiveInfinity{}}, `<h xmlns="urn:s">-INF</h>`, samePD, swapped)

	if err := CheckAppend[tPrecisionDecimal](schema, sname("h")); !errors.Is(err, ErrTypeMismatch) {
		t.Errorf("CheckAppend with no WithPrecisionDecimalArms = %v, want ErrTypeMismatch", err)
	}
	_, _ = appendChecked(t, nil, schema, strict.New(), sname("h"), tPrecisionDecimal{tPDNotANumber{}})
}

// TestAppendPrecisionDecimalInvalid pins item 6 of Append's doc for a
// PrecisionDecimal: a nil Value, a numerical arm of negative Coefficient, a
// pointer to an arm, which the set supplies in no role, and a list item of a
// nil Value, each named by the Value's Go path and carrying no rule; a
// maxInclusive facet's rejection, carrying cvc-maxInclusive-valid; and a
// pattern's rejection of -INF and of NaN, which have no other lexical, and a
// list's pattern rejecting a list of -INF alone.
func TestAppendPrecisionDecimalInvalid(t *testing.T) {
	schema := schemaOf(t, encodePDSchema)
	for _, row := range []struct {
		name string
		elem string
		v    tPrecisionDecimal
		want string
	}{
		{"a nil Value", "h", tPrecisionDecimal{}, `codec: Append: tPrecisionDecimal.Value is nil, of simple type xs:precisionDecimal, which no precisionDecimal value maps to`},
		{"a negative Coefficient", "h", pdNumerical(-3, 0, false), `codec: Append: tPrecisionDecimal.Value holds the Coefficient -3, of simple type xs:precisionDecimal, negative`},
		{"a pointer to an arm", "h", tPrecisionDecimal{&tPDNumerical{}}, `codec: Append: tPrecisionDecimal.Value holds a *codec.tPDNumerical, of simple type xs:precisionDecimal, which the WithPrecisionDecimalArms set of codec.tPDValue supplies in no role`},
	} {
		t.Run(row.name, func(t *testing.T) {
			_, err := appendChecked(t, nil, schema, strict.New(), sname(row.elem), row.v, pdArms)
			invalidValue(t, err)
			if !strings.HasPrefix(err.Error(), row.want) {
				t.Errorf("error %q does not open with %q", err, row.want)
			}
			noRule(t, err)
		})
	}
	t.Run("a list item's nil Value", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("hl"), tHl{pdNumerical(3, 0, false), {}}, pdArms)
		invalidValue(t, err)
		if want := "codec: Append: tHl[1].Value is nil, of simple type "; !strings.HasPrefix(err.Error(), want) {
			t.Errorf("error %q does not open with %q", err, want)
		}
	})
	t.Run("a maxInclusive facet", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("max"), pdNumerical(6, 0, false), pdArms)
		invalidValue(t, err, "codec: Append: tPrecisionDecimal holds 6E0, which simple type the anonymous simple type of element {urn:s}max")
		if rule, _ := xsderr.RuleOf(err); rule != "cvc-maxInclusive-valid" {
			t.Errorf("rule %q, want cvc-maxInclusive-valid", rule)
		}
	})
	t.Run("a list's pattern rejecting [-INF]", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("pl"), tHl{{tPDNegativeInfinity{}}}, pdArms)
		invalidValue(t, err, `codec: Append: tHl holds [-INF], written "-INF", which simple type`)
	})
	for _, v := range []tPrecisionDecimal{{tPDNegativeInfinity{}}, {tPDNotANumber{}}} {
		t.Run("a pattern rejecting "+spellDecimal(reflect.ValueOf(v)), func(t *testing.T) {
			_, err := appendChecked(t, nil, schema, strict.New(), sname("pat"), v, pdArms)
			invalidValue(t, err, "codec: Append: tPrecisionDecimal holds "+spellDecimal(reflect.ValueOf(v))+", which simple type the anonymous simple type of element {urn:s}pat")
			if rule, _ := xsderr.RuleOf(err); rule != rulePatternValid {
				t.Errorf("rule %q, want %s", rule, rulePatternValid)
			}
		})
	}
}

// pdRendering returns strict's backend with its xs:precisionDecimal mapping's
// Canonical replaced by one rendering text for every value.
func pdRendering(text string) value.Backend {
	return overDatatype("precisionDecimal", func(base value.Mapping) value.Mapping {
		return value.Mapping{Parse: base.Parse, Canonical: func(value.Value) (string, error) { return text, nil }}
	})
}

// The other backends of TestAppendPrecisionDecimalBackend, each strict's with
// its xs:precisionDecimal mapping replaced: a nil and a failing Canonical,
// and one whose Parse rejects every lexical.
var (
	pdNilCanonical = overDatatype("precisionDecimal", func(base value.Mapping) value.Mapping {
		return value.Mapping{Parse: base.Parse}
	})
	pdFailingCanonical = overDatatype("precisionDecimal", func(base value.Mapping) value.Mapping {
		return value.Mapping{Parse: base.Parse, Canonical: func(value.Value) (string, error) { return "", errors.New("no canonical form") }}
	})
	pdRejectingParse = overDatatype("precisionDecimal", func(base value.Mapping) value.Mapping {
		return value.Mapping{Parse: func(string, value.Context) (value.Value, error) { return nil, errors.New("rejected") }, Canonical: base.Canonical}
	})
)

// TestAppendPrecisionDecimalBackend pins the write's reading of the backend:
// Canonical's text is written where it maps back to the value, as 300E-2 does
// to 3.00 though it is not f-precDecCanmap's, so no backend fault is charged
// against a canonical string of the codec's own; it is item 7 where it does
// not map back, as 3 does not, nor 300E-1, of another Scale alone, nor -3.00,
// of another sign alone, or where Canonical is nil or fails, and where a
// pattern rejects a numerical value or INF, each of which has another
// lexical, or a list holding one beside -INF; and a Parse rejecting the
// value's pDecimalRep is a backend fault.
func TestAppendPrecisionDecimalBackend(t *testing.T) {
	schema := schemaOf(t, encodePDSchema)
	got, err := appendChecked(t, nil, schema, pdRendering("300E-2"), sname("h"), pdNumerical(300, 2, false), pdArms)
	if want := `<h xmlns="urn:s">300E-2</h>`; err != nil || string(got) != want {
		t.Errorf("Append = %q, %v, want %q", got, err, want)
	}
	for _, row := range []struct {
		name    string
		backend value.Backend
		elem    string
		v       tPrecisionDecimal
		want    string
	}{
		{"a Canonical not mapping back", pdRendering("3"), "h", pdNumerical(300, 2, false), `simple type xs:precisionDecimal, at tPrecisionDecimal holding 300E-2: its canonical representation "3" does not map back to it`},
		{"a Canonical of another Scale", pdRendering("300E-1"), "h", pdNumerical(300, 2, false), `its canonical representation "300E-1" does not map back to it`},
		{"a Canonical of another sign", pdRendering("-3.00"), "h", pdNumerical(300, 2, false), `its canonical representation "-3.00" does not map back to it`},
		{"a Canonical of another arm", pdRendering("-INF"), "h", tPrecisionDecimal{tPDPositiveInfinity{}}, `its canonical representation "-INF" does not map back to it`},
		{"a nil Canonical", pdNilCanonical, "h", pdNumerical(300, 2, false), "at tPrecisionDecimal holding 300E-2: its governing mapping has no canonical mapping"},
		{"a failing Canonical", pdFailingCanonical, "h", tPrecisionDecimal{tPDNotANumber{}}, "at tPrecisionDecimal holding NaN: no canonical form"},
		{"a pattern rejecting a numerical value", strict.New(), "pat", pdNumerical(3, 0, false), `at tPrecisionDecimal holding 3E0: a pattern facet rejects its canonical representation "3"`},
		{"a pattern rejecting INF, which +INF also spells", strict.New(), "pat", tPrecisionDecimal{tPDPositiveInfinity{}}, `at tPrecisionDecimal holding INF: a pattern facet rejects its canonical representation "INF"`},
	} {
		t.Run(row.name, func(t *testing.T) {
			_, err := appendChecked(t, nil, schema, row.backend, sname(row.elem), row.v, pdArms)
			_ = asDecline(t, err, "codec does not encode a value with no canonical representation: ")
			holds(t, err, row.want)
		})
	}
	t.Run("a list's pattern rejecting [-INF 3E0]", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("pl"), tHl{{tPDNegativeInfinity{}}, pdNumerical(3, 0, false)}, pdArms)
		_ = asDecline(t, err, "codec does not encode a value with no canonical representation: ")
		holds(t, err, `at tHl holding [-INF 3]: a pattern facet rejects its canonical representation "-INF 3"`)
	})
	_, err = appendChecked(t, nil, schema, pdRejectingParse, sname("h"), pdNumerical(300, 2, false), pdArms)
	plainError(t, err)
	if want := "codec: Append: tPrecisionDecimal holds 300E-2, of simple type xs:precisionDecimal, whose pDecimalRep form the backend's Parse rejects: rejected"; err.Error() != want {
		t.Errorf("error %q, want %q", err, want)
	}
}

// encodeUnionSchema declares, beside unionTypes, document elements of the
// union types "# What encodes" admits: root, of u; list, a list of u; num;
// nest; wl, withList; parts, of u in a local element occurring once,
// optionally and repeatedly and in an optional and a required attribute;
// leading, a restriction of u whose pattern admits only a lexical with a
// leading digit 0, which no canonical xs:int representation but 0's has;
// five, a restriction of u enumerating 5; str, a restriction by the pattern
// a of a union of xs:string alone, whose values have no other lexical; qn, a
// union of xs:int and xs:QName; nq, a union of xs:Name and xs:QName, whose
// xs:Name member accepts every prefixed spelling of an xs:QName value first;
// qnn, a union of xs:QName and xs:Name; and wlp, a restriction of withList by
// the pattern 0.*, which rejects the canonical representation of its list
// member's values but for one whose first item is 0.
const encodeUnionSchema = unionTypes + `
<xs:element name="root" type="s:u"/>
<xs:element name="list"><xs:simpleType><xs:list itemType="s:u"/></xs:simpleType></xs:element>
<xs:element name="num" type="s:num"/>
<xs:element name="nest" type="s:nest"/>
<xs:element name="wl" type="s:withList"/>
<xs:element name="parts"><xs:complexType><xs:sequence>
<xs:element name="a" type="s:u"/><xs:element name="opt" type="s:u" minOccurs="0"/><xs:element name="many" type="s:u" maxOccurs="unbounded"/>
</xs:sequence><xs:attribute name="at" type="s:u"/><xs:attribute name="req" type="s:u" use="required"/></xs:complexType></xs:element>
<xs:element name="leading"><xs:simpleType><xs:restriction base="s:u"><xs:pattern value="0\d+"/></xs:restriction></xs:simpleType></xs:element>
<xs:element name="five"><xs:simpleType><xs:restriction base="s:u"><xs:enumeration value="5"/></xs:restriction></xs:simpleType></xs:element>
<xs:element name="str"><xs:simpleType><xs:restriction><xs:simpleType><xs:union memberTypes="xs:string"/></xs:simpleType><xs:pattern value="a"/></xs:restriction></xs:simpleType></xs:element>
<xs:element name="qn"><xs:simpleType><xs:union memberTypes="xs:int xs:QName"/></xs:simpleType></xs:element>
<xs:element name="nq"><xs:simpleType><xs:union memberTypes="xs:Name xs:QName"/></xs:simpleType></xs:element>
<xs:element name="qnn"><xs:simpleType><xs:union memberTypes="xs:QName xs:Name"/></xs:simpleType></xs:element>
<xs:element name="wlp"><xs:simpleType><xs:restriction base="s:withList"><xs:pattern value="0.*"/></xs:restriction></xs:simpleType></xs:element>`

// unionParts is codegen's struct for encodeUnionSchema's parts, its particle
// fields before its attribute fields; unionStr and its variant are str's
// interface, and unionQn, unionNq and unionQnn and their variants qn's, nq's
// and qnn's, which TestEncodeUnionTypesAreCodegens pins.
type (
	unionParts struct {
		A    unionU
		Opt  unionU
		Many []unionU
		At   unionU
		Req  unionU
	}
	unionStr       interface{ isStr() }
	unionStrString string
	unionQn        interface{ isQn() }
	unionQnInt     int32
	unionQnQName   struct{ Space, Local string }
	unionNq        interface{ isNq() }
	unionNqName    string
	unionNqQName   struct{ Space, Local string }
	unionQnn       interface{ isQnn() }
	unionQnnQName  struct{ Space, Local string }
	unionQnnName   string
)

func (unionStrString) isStr() {}
func (unionQnInt) isQn()      {}
func (unionQnQName) isQn()    {}
func (unionNqName) isNq()     {}
func (unionNqQName) isNq()    {}
func (unionQnnQName) isQnn()  {}
func (unionQnnName) isQnn()   {}

// encodeUnionVariants is unionVariants with str's, qn's, nq's and qnn's sets.
var encodeUnionVariants = append(slices.Clone(unionVariants),
	WithBranches[unionStr](unionStrString("")), WithBranches[unionQn](unionQnInt(0), unionQnQName{}),
	WithBranches[unionNq](unionNqName(""), unionNqQName{}), WithBranches[unionQnn](unionQnnQName{}, unionQnnName("")))

// TestEncodeUnionTypesAreCodegens pins the types above to those codegen
// emits for encodeUnionSchema, in shape.
func TestEncodeUnionTypesAreCodegens(t *testing.T) {
	gen := generated(t, schemaOf(t, encodeUnionSchema))
	for _, decl := range []string{
		"type Parts struct {\n\tA    U\n\tOpt  U\n\tMany []U\n\tAt   U\n\tReq  U\n}\n",
		"type StrUnionString string\n", "type QnUnionInt int32\n", "type QnUnionQName QName\n",
		"type NqUnionName string\n", "type NqUnionQName QName\n", "type QnnUnionQName QName\n", "type QnnUnionName string\n",
	} {
		if !bytes.Contains(gen, []byte(decl)) {
			t.Errorf("codegen emits\n%s\nwhich does not declare\n%s", gen, decl)
		}
	}
}

// sameUnion reports whether a and b are one value as spellUnion spells it,
// each variant named.
func sameUnion[T any](a, b T) bool {
	return spellUnion(reflect.ValueOf(&a).Elem()) == spellUnion(reflect.ValueOf(&b).Elem())
}

// unionDateOf is the unionDate of y-m-d with no timezone.
func unionDateOf(y int64, m, d uint8) unionDate {
	return unionDate{Year: *big.NewInt(y), Month: m, Day: d}
}

// TestAppendUnions pins the union lift (Datatypes §2.4.1.3): a value of a
// union's sealed interface is written as its variant's member value is, in
// that member's canonical representation, and validate accepts it and Decode
// returns the same variant, at every site a union reaches: a document
// element, each of two members with disjoint lexical spaces, a list of a
// union, items of both members in one list, a nested union's inner basic
// member and its own, xs:decimal 3/2 under num, where xs:int rejects 1.5, a
// list member, a list member of a union of a list of a union, an element
// particle occurring once, optionally, absent, and repeatedly, an optional
// attribute absent and present and a required one, and an xs:QName member,
// its prefix declared on the element's start tag, under qn and under qnn,
// whose xs:QName member precedes xs:Name and so is the ·active basic member·
// of the prefixed spelling (dt-active-basic-member). CheckAppend returns nil
// for each.
func TestAppendUnions(t *testing.T) {
	schema := schemaOf(t, encodeUnionSchema)
	opts := encodeUnionVariants
	t.Run("root, the xs:int member", func(t *testing.T) {
		roundTrip(t, schema, sname("root"), unionRoot(unionUInt(5)), `<root xmlns="urn:s">5</root>`, sameUnion[unionRoot], opts...)
	})
	t.Run("root, the xs:date member", func(t *testing.T) {
		roundTrip(t, schema, sname("root"), unionRoot(unionUDate(unionDateOf(2000, 1, 2))), `<root xmlns="urn:s">2000-01-02</root>`, sameUnion[unionRoot], opts...)
	})
	t.Run("a list of a union, items of both members", func(t *testing.T) {
		roundTrip(t, schema, sname("list"), unionListRoot{unionUInt(5), unionUDate(unionDateOf(2000, 1, 2))}, `<list xmlns="urn:s">5 2000-01-02</list>`, sameUnion[unionListRoot], opts...)
	})
	t.Run("xs:decimal 3/2 beside xs:int", func(t *testing.T) {
		roundTrip(t, schema, sname("num"), unionNum(unionNumDecimal(*big.NewRat(3, 2))), `<num xmlns="urn:s">1.5</num>`, sameUnion[unionNum], opts...)
	})
	t.Run("a nested union's inner member and its own", func(t *testing.T) {
		roundTrip(t, schema, sname("nest"), unionNest(unionNestInt(6)), `<nest xmlns="urn:s">6</nest>`, sameUnion[unionNest], opts...)
		roundTrip(t, schema, sname("nest"), unionNest(unionNestBoolean(true)), `<nest xmlns="urn:s">true</nest>`, sameUnion[unionNest], opts...)
	})
	t.Run("a list member", func(t *testing.T) {
		roundTrip(t, schema, sname("wl"), unionWithList(unionWithListList{1, 2}), `<wl xmlns="urn:s">1 2</wl>`, sameUnion[unionWithList], opts...)
		roundTrip(t, schema, sname("wl"), unionWithList(unionWithListBoolean(false)), `<wl xmlns="urn:s">false</wl>`, sameUnion[unionWithList], opts...)
	})
	t.Run("a list member of a union of a list of a union", func(t *testing.T) {
		lm := schemaOf(t, unionListMemberRoot)
		roundTrip(t, lm, sname("r"), unionLM(unionLMUnionList{unionUInt(5), unionUDate(unionDateOf(2000, 1, 1))}), `<r xmlns="urn:s">5 2000-01-01</r>`, sameUnion[unionLM], opts...)
		roundTrip(t, lm, sname("r"), unionLM(unionLMUnionDate(unionDateOf(2000, 1, 1))), `<r xmlns="urn:s">2000-01-01</r>`, sameUnion[unionLM], opts...)
	})
	t.Run("particles and attributes", func(t *testing.T) {
		roundTrip(t, schema, sname("parts"), unionParts{A: unionUInt(1), Many: []unionU{unionUDate(unionDateOf(2000, 1, 2)), unionUInt(3)}, Req: unionUInt(7)},
			`<parts xmlns="urn:s" req="7"><a>1</a><many>2000-01-02</many><many>3</many></parts>`, sameUnion[unionParts], opts...)
		roundTrip(t, schema, sname("parts"), unionParts{A: unionUInt(1), Opt: unionUInt(2), Many: []unionU{unionUInt(3)}, At: unionUDate(unionDateOf(2000, 1, 4)), Req: unionUInt(7)},
			`<parts xmlns="urn:s" at="2000-01-04" req="7"><a>1</a><opt>2</opt><many>3</many></parts>`, sameUnion[unionParts], opts...)
	})
	t.Run("an xs:QName member", func(t *testing.T) {
		roundTrip(t, schema, sname("qn"), unionQn(unionQnQName{Space: "urn:o", Local: "a"}), `<qn xmlns="urn:s" xmlns:p1="urn:o">p1:a</qn>`, sameUnion[unionQn], opts...)
		roundTrip(t, schema, sname("qn"), unionQn(unionQnQName{Space: "urn:s", Local: "a"}), `<qn xmlns="urn:s">a</qn>`, sameUnion[unionQn], opts...)
		roundTrip(t, schema, sname("qnn"), unionQnn(unionQnnQName{Space: "urn:o", Local: "a"}), `<qnn xmlns="urn:s" xmlns:p1="urn:o">p1:a</qnn>`, sameUnion[unionQnn], opts...)
	})
}

// TestAppendUnionsOfOneGoType pins the lift where basic members share a Go
// type (oneGoTypeSchema): each variant is written where no earlier member
// accepts its text, a list-of-a-union member's items among them, the variants
// placed at their members by WithBranches' supplied order.
func TestAppendUnionsOfOneGoType(t *testing.T) {
	v := ogt{A: ogtStString("a b"), B: ogtLsString("ABC"), C: ogtLisString("A1"), D: ogtLlList2{"A"}, E: ogtLuList2{ogtIBBoolean(true)}}
	roundTrip(t, schemaOf(t, oneGoTypeSchema), sname("r"), v, `<r xmlns="urn:s"><a>a b</a><b>ABC</b><c>A1</c><d>A</d><e>true</e></r>`, sameUnion[ogt], ogtVariants...)
}

// TestAppendUnionInvalid pins item 6 of Append's doc for a union: the nil
// interface where a value is required, at the document element, a required
// attribute, a list item and a repeated particle's item; a dynamic type the
// WithBranches set places at no basic member, a stray implementing the
// interface and a pointer to a variant; an enumeration of the union itself
// rejecting the active member's value (cvc-datatype-valid clause 3); a union
// pattern rejecting a string, which has no other lexical; and a member's own
// rejection, under the member's subject.
func TestAppendUnionInvalid(t *testing.T) {
	schema := schemaOf(t, encodeUnionSchema)
	date := unionUDate(unionDateOf(2000, 1, 2))
	for _, row := range []struct {
		name string
		elem string
		v    any
		want []string
	}{
		{"a nil root", "root", nil, []string{"codec: Append: unionRoot is nil, of simple type {urn:s}u defined at ", "where a value is required"}},
		{"a nil required attribute", "parts", unionParts{A: date, Many: []unionU{date}}, []string{"codec: Append: unionParts.Req is nil, of simple type {urn:s}u"}},
		{"a nil list item", "list", unionListRoot{date, nil}, []string{"codec: Append: unionListRoot[1] is nil, of simple type ", "through the list type's {item type definition} {urn:s}u"}},
		{"a nil repeated item", "parts", unionParts{A: date, Many: []unionU{nil}, Req: date}, []string{"codec: Append: unionParts.Many[0] is nil"}},
		{"a stray", "root", unionRoot(unionUOther(1)), []string{"codec: Append: unionRoot holds a codec.unionUOther, which the WithBranches set of codec.unionRoot places at no basic member of simple type {urn:s}u"}},
		{"a pointer to a variant", "root", unionRoot(new(unionUInt)), []string{"codec: Append: unionRoot holds a *codec.unionUInt, which the WithBranches set"}},
		{"an enumeration of the union", "five", unionRoot(unionUInt(6)), []string{`codec: Append: unionRoot holds a codec.unionUInt written "6", which simple type the anonymous simple type of element {urn:s}five`, "cvc-enumeration-valid"}},
		{"a union pattern rejecting a string", "str", unionStr(unionStrString("b")), []string{`codec: Append: unionStr holds a codec.unionStrString written "b", which simple type the anonymous simple type of element {urn:s}str`, "cvc-pattern-valid"}},
		{"a member's own rejection", "qn", unionQn(unionQnQName{Space: "urn:o", Local: "1a"}), []string{"codec: Append: unionQn.(unionQnQName) holds", "through the union's basic member xs:QName"}},
	} {
		t.Run(row.name, func(t *testing.T) {
			var err error
			switch v := row.v.(type) {
			case nil:
				// A nil unionRoot held in any is the nil any.
				_, err = appendChecked(t, nil, schema, strict.New(), sname(row.elem), unionRoot(nil), encodeUnionVariants...)
			case unionRoot:
				_, err = appendChecked(t, nil, schema, strict.New(), sname(row.elem), v, encodeUnionVariants...)
			case unionParts:
				_, err = appendChecked(t, nil, schema, strict.New(), sname(row.elem), v, encodeUnionVariants...)
			case unionListRoot:
				_, err = appendChecked(t, nil, schema, strict.New(), sname(row.elem), v, encodeUnionVariants...)
			case unionStr:
				_, err = appendChecked(t, nil, schema, strict.New(), sname(row.elem), v, encodeUnionVariants...)
			case unionQn:
				_, err = appendChecked(t, nil, schema, strict.New(), sname(row.elem), v, encodeUnionVariants...)
			}
			invalidValue(t, err, row.want...)
			if !strings.HasPrefix(err.Error(), row.want[0]) {
				t.Errorf("error %q does not open with %q", err, row.want[0])
			}
		})
	}
}

// TestAppendUnionNoCanonical pins item 7 of Append's doc for a union: a
// variant whose canonical representation an earlier member accepts, its
// ·active basic member· (Datatypes §2.4.1.3 dt-active-basic-member, §4.1.4
// cvc-datatype-valid clause 2.3), so that Decode returns another variant
// though a valid document holds the value: xs:decimal 1 under num, written 1,
// which xs:int takes, though 1.0, which validate accepts, decodes into the
// xs:decimal variant; ls's xs:string variant, which lower takes; and a
// list-of-union member's item, which the earlier list member takes; and a
// pattern of the union itself rejecting the canonical representation of a
// value with another lexical, as 05 spells xs:int 5, and as 01 2 spells a
// list member's value [1 2]; and, by constructQNameSpelling, nq's xs:QName
// variant, whose spelling p1:a xs:Name accepts first, so that it is xs:Name's
// value Decode returns. Each row opens with its construct's decline.
func TestAppendUnionNoCanonical(t *testing.T) {
	schema := schemaOf(t, encodeUnionSchema)
	ogtSchema := schemaOf(t, oneGoTypeSchema)
	const noCanonical = "codec does not encode a value with no canonical representation: "
	for _, row := range []struct {
		name   string
		err    func(t *testing.T) error
		prefix string
		want   []string
	}{
		{"xs:decimal 1 beside xs:int", func(t *testing.T) error {
			accepts(t, schema, []byte(`<num xmlns="urn:s">1.0</num>`))
			_, err := appendChecked(t, nil, schema, strict.New(), sname("num"), unionNum(unionNumDecimal(*big.NewRat(1, 1))), encodeUnionVariants...)
			return err
		}, noCanonical, []string{`simple type {urn:s}num defined at `,
			`, at unionNum holding a codec.unionNumDecimal written "1": the ·active basic member· of its canonical representation "1" is xs:int, not its variant's member xs:decimal`}},
		{"the string variant lower takes", func(t *testing.T) error {
			v := ogt{A: ogtStString("a"), B: ogtLsString("abc"), C: ogtLisString("A"), D: ogtLlList2{"A"}, E: ogtLuList2{ogtIBBoolean(true)}}
			_, err := appendChecked(t, nil, ogtSchema, strict.New(), sname("r"), v, ogtVariants...)
			return err
		}, noCanonical, []string{`at ogt.B holding a codec.ogtLsString written "abc": the ·active basic member· of its canonical representation "abc" is {urn:s}lower, not its variant's member xs:string`}},
		{"a list-of-union member's item the earlier list member takes", func(t *testing.T) error {
			v := ogt{A: ogtStString("a"), B: ogtLsString("A"), C: ogtLisString("A"), D: ogtLlList2{"A"}, E: ogtLuList2{ogtIBInt(1)}}
			_, err := appendChecked(t, nil, ogtSchema, strict.New(), sname("r"), v, ogtVariants...)
			return err
		}, noCanonical, []string{`at ogt.E holding a codec.ogtLuList2 written "1": the ·active basic member· of its canonical representation "1" is the anonymous simple type defined at `}},
		{"a union pattern rejecting a canonical representation", func(t *testing.T) error {
			accepts(t, schema, []byte(`<leading xmlns="urn:s">05</leading>`))
			_, err := appendChecked(t, nil, schema, strict.New(), sname("leading"), unionRoot(unionUInt(5)), encodeUnionVariants...)
			return err
		}, noCanonical, []string{`at unionRoot holding a codec.unionUInt written "5": a pattern facet rejects its canonical representation "5"`}},
		{"a union pattern rejecting a list member's text", func(t *testing.T) error {
			accepts(t, schema, []byte(`<wlp xmlns="urn:s">01 2</wlp>`))
			_, err := appendChecked(t, nil, schema, strict.New(), sname("wlp"), unionWithList(unionWithListList{1, 2}), encodeUnionVariants...)
			return err
		}, noCanonical, []string{`at unionWithList holding a codec.unionWithListList written "1 2": a pattern facet rejects its canonical representation "1 2"`}},
		{"an xs:QName variant xs:Name shadows", func(t *testing.T) error {
			accepts(t, schema, []byte(`<nq xmlns="urn:s" xmlns:p1="urn:o">p1:a</nq>`))
			_, err := appendChecked(t, nil, schema, strict.New(), sname("nq"), unionNq(unionNqQName{Space: "urn:o", Local: "a"}), encodeUnionVariants...)
			return err
		}, "codec does not encode an xs:QName or xs:NOTATION value with no spelling Append writes where it is written: ",
			[]string{`at unionNq holding a codec.unionNqQName written "p1:a": the ·active basic member· of its spelling "p1:a" is xs:Name, not its variant's member xs:QName`}},
	} {
		t.Run(row.name, func(t *testing.T) {
			err := row.err(t)
			_ = asDecline(t, err, row.prefix)
			holds(t, err, row.want...)
		})
	}
}

// TestAppendUnionDeclineSubject pins the detail of a union's item 4 decline:
// a basic member's construct names the member after the union, as Decode's
// does, and a member union's assertions facet names that union, which
// goshape.BasicMembers flattens away.
func TestAppendUnionDeclineSubject(t *testing.T) {
	schema := schemaOf(t, `<xs:element name="id"><xs:simpleType><xs:union memberTypes="xs:int xs:ENTITY"/></xs:simpleType></xs:element>
<xs:simpleType name="asserted"><xs:restriction><xs:simpleType><xs:union memberTypes="xs:int"/></xs:simpleType><xs:assertion test="true()"/></xs:restriction></xs:simpleType>
<xs:element name="mu"><xs:simpleType><xs:union memberTypes="s:asserted xs:date"/></xs:simpleType></xs:element>`)
	err := CheckAppend[unionU](schema, sname("id"))
	_ = asDecline(t, err, "codec does not encode a simple type derived from xs:ENTITY: simple type the anonymous simple type of element {urn:s}id declared at s.xsd:")
	holds(t, err, ", through the union's basic member xs:ENTITY;")
	err = CheckAppend[unionU](schema, sname("mu"))
	_ = asDecline(t, err, "codec does not encode a simple type with an assertions facet: simple type the anonymous simple type of element {urn:s}mu declared at s.xsd:")
	holds(t, err, ", through the union's member union {urn:s}asserted, a type defined at s.xsd:")
}

// TestCheckAppendUnionMismatch pins item 5 of Append's doc for a union's
// interface, which item 4 now admits: no WithBranches set for it is
// ErrTypeMismatch, from Append and CheckAppend alike.
func TestCheckAppendUnionMismatch(t *testing.T) {
	schema := schemaOf(t, encodeUnionSchema)
	_, err := appendChecked(t, nil, schema, strict.New(), sname("root"), unionRoot(unionUInt(5)))
	if !errors.Is(err, ErrTypeMismatch) || !strings.Contains(err.Error(), "no WithBranches supplies the variants of codec.unionRoot, the union's interface of T, where codegen emits 2") {
		t.Errorf("Append = %v, want item 5's ErrTypeMismatch for the union's missing set", err)
	}
}

// The skip-wildcard schemas Append encodes, each declaring s:r, beside
// wildcard_test.go's seqSkipSchema: skipManySchema's wildcard repeats twice
// at most after a; skipOptionalSchema is seqOptionalSchema's sequence(a, b?,
// any?) under skip; skipLeadingSchema's optional wildcard precedes a;
// skipTargetSchema's is over ##targetNamespace and skipNotSchema's disallows
// s:x; skipPickSchema's choice holds a and the wildcard, skipEmptySchema's a
// and an optional wildcard, and skipAttrSchema's is skipPickSchema's beside
// an attribute x. Their s:r types are wildcard_test.go's seqRepeated,
// seqOptional, seqLeading, seqWild for both constrained wildcards, pickWild
// for skipPickSchema, and skipEmpty and skipAttr, which
// TestWildcardTypesAreCodegens pins.
const (
	skipManySchema     = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/><xs:any processContents="skip" maxOccurs="2"/></xs:sequence></xs:complexType></xs:element>`
	skipOptionalSchema = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string" minOccurs="0"/><xs:any processContents="skip" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>`
	skipLeadingSchema  = `<xs:element name="r"><xs:complexType><xs:sequence><xs:any processContents="skip" minOccurs="0"/><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType></xs:element>`
	skipTargetSchema   = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/><xs:any namespace="##targetNamespace" processContents="skip"/></xs:sequence></xs:complexType></xs:element>`
	skipNotSchema      = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/><xs:any notQName="s:x" processContents="skip"/></xs:sequence></xs:complexType></xs:element>`
	skipPickSchema     = `<xs:element name="r"><xs:complexType><xs:choice><xs:element name="a" type="xs:string"/><xs:any processContents="skip"/></xs:choice></xs:complexType></xs:element>`
	skipEmptySchema    = `<xs:element name="r"><xs:complexType><xs:choice><xs:element name="a" type="xs:string"/><xs:any processContents="skip" minOccurs="0"/></xs:choice></xs:complexType></xs:element>`
	skipAttrSchema     = `<xs:element name="r"><xs:complexType><xs:choice><xs:element name="a" type="xs:string"/><xs:any processContents="skip"/></xs:choice><xs:attribute name="x" type="xs:string"/></xs:complexType></xs:element>`
)

// The lax-wildcard schemas, each declaring s:r: laxGovernedSchema is
// seqLaxSchema, sequence(a, any), beside a top-level element w and a
// top-level attribute t, each of xs:int, which govern an item so named at any
// depth of the match; laxTargetSchema is sequence(a, any) over
// ##targetNamespace; and laxPickBaseSchema and laxFlatBaseSchema restrict
// laxBase, sequence(c?, any*), by pickWildSchema's choice(a, any) and by
// flatPickSchema's flattened choice(a+, any), repeated, so that a match named
// c has its ·locally declared type· from the base type alone (§3.4.6.4
// key-ldt-elem clause 3). Their s:r types are wildcard_test.go's seqWild for
// the first two and pickWild, and flatten_test.go's flatPick, which
// TestWildcardTypesAreCodegens and TestFlattenTypesAreCodegens pin.
const (
	laxGovernedSchema = seqLaxSchema + `<xs:element name="w" type="xs:int"/><xs:attribute name="t" type="xs:int"/>`
	laxTargetSchema   = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/><xs:any namespace="##targetNamespace" processContents="lax"/></xs:sequence></xs:complexType></xs:element>`
	laxBase           = `<xs:complexType name="lb"><xs:sequence><xs:element name="c" type="xs:string" minOccurs="0"/><xs:any processContents="lax" minOccurs="0" maxOccurs="unbounded"/></xs:sequence></xs:complexType>`
	laxPickBaseSchema = laxBase + `<xs:element name="r"><xs:complexType><xs:complexContent><xs:restriction base="s:lb"><xs:choice><xs:element name="a" type="xs:string"/><xs:any processContents="lax"/></xs:choice></xs:restriction></xs:complexContent></xs:complexType></xs:element>`
	laxFlatBaseSchema = laxBase + `<xs:element name="r"><xs:complexType><xs:complexContent><xs:restriction base="s:lb"><xs:choice maxOccurs="unbounded"><xs:element name="a" type="xs:string" maxOccurs="unbounded"/><xs:any processContents="lax"/></xs:choice></xs:restriction></xs:complexContent></xs:complexType></xs:element>`
)

// skipAttr and skipEmpty are the types codegen emits for skipAttrSchema's and
// skipEmptySchema's s:r, each choice in the struct's first field, of one
// interface's shape.
type (
	skipAttr struct {
		Choice skipAttrChoice
		X      *string
	}
	skipEmpty      struct{ Choice skipAttrChoice }
	skipAttrChoice interface{ isSkipAttrChoice() }
	skipAttrA      struct{ A string }
	skipAttrAny    struct{ Any anyElement }
)

func (skipAttrA) isSkipAttrChoice()   {}
func (skipAttrAny) isSkipAttrChoice() {}

// skipAttrBranches is skipAttrChoice's branch set.
var skipAttrBranches = WithBranches[skipAttrChoice](skipAttrA{}, skipAttrAny{})

// TestAppendWildcard pins the wildcard particle with {process contents} skip "# What encodes"
// admits: each element its field holds is written raw, its attributes in held order, an
// unqualified one unprefixed, one in the XML namespace as xml:, undeclared, and every other one
// under the prefix an open start tag declares for its namespace, or a fresh pN declared on its
// element's own, though the namespace be the element's; then its Text, then each child followed by
// its Tail; an element in the XML namespace as xml:Local, moving no default namespace, and one in
// the XSI namespace under a default declaration of it, an attribute in it being item 7 but for
// xsi:schemaLocation and xsi:noNamespaceSchemaLocation, written on a skip match and on a lax one
// as any other attribute is; both rows fail with rawAttribute's hint exemption removed, and the
// lax one, not the skip one, with rawAttributes asking encoder.governed of a hint. Each document
// validate accepts and Decode turns back into the value; an optional wildcard holding
// nothing writes nothing, and so does the nil interface of a choice whose wildcard alternative has
// {min occurs} 0, which is emptiable though its branch reads it as 1 (choiceEmptiable). The rows
// of one and several held elements are the ones the static decline failed before the slice
// (constructWildcard). A skip match named as the element particle beside it is written, no
// ·locally declared type· read for it, and so is a lax match no declaration governs, at the
// top and with a child at depth, unqualified and foreign attributes on each, and one holding a
// child named as that element particle, which has no locally declared type below the match
// (§3.4.6.4 key-ldt-elem clause 1); the three lax rows decline constructWildcard with
// particleEncodes' narrowing to strict undone.
func TestAppendWildcard(t *testing.T) {
	t.Run("one held element", func(t *testing.T) {
		roundTrip(t, schemaOf(t, seqSkipSchema), rootName, seqWild{A: "x", Any: anyElement{Space: "urn:o", Local: "w"}},
			`<r xmlns="urn:s"><a>x</a><w xmlns="urn:o"/></r>`, deepEqual[seqWild])
	})
	t.Run("several held elements, raw content", func(t *testing.T) {
		v := seqRepeated{A: "x", Any: []anyElement{
			{Space: "urn:o", Local: "p", Content: anyType{
				Attributes: []anyAttribute{{"", "k", "v\t1"}, {"urn:o", "q", "w"}, {xmltree.XMLNamespaceURI, "lang", "en"}, {"urn:z", "z", `<&"`}},
				Text:       "t ",
				Children: []anyChild{
					{Element: anyElement{Local: "c", Content: anyType{Text: "c\r"}}, Tail: " tail"},
					{Element: anyElement{Space: "urn:o", Local: "d", Content: anyType{Attributes: []anyAttribute{{"urn:z", "y", "1"}}}}},
				},
			}},
			{Space: "urn:s", Local: "a"},
		}}
		roundTrip(t, schemaOf(t, skipManySchema), rootName, v,
			`<r xmlns="urn:s"><a>x</a><p xmlns="urn:o" k="v&#x9;1" xmlns:p1="urn:o" p1:q="w" xml:lang="en" xmlns:p2="urn:z" p2:z="&lt;&amp;&quot;">t <c xmlns="">c&#xD;</c> tail<d p2:y="1"/></p><a/></r>`,
			deepEqual[seqRepeated])
	})
	t.Run("elements in the XML namespace and in none", func(t *testing.T) {
		v := seqRepeated{A: "x", Any: []anyElement{
			{Space: xmltree.XMLNamespaceURI, Local: "e", Content: anyType{Children: []anyChild{{Element: anyElement{Space: "urn:s", Local: "in"}}}}},
			{Local: "n", Content: anyType{Children: []anyChild{{Element: anyElement{Space: "urn:s", Local: "in"}}}}},
		}}
		roundTrip(t, schemaOf(t, skipManySchema), rootName, v,
			`<r xmlns="urn:s"><a>x</a><xml:e><in/></xml:e><n xmlns=""><in xmlns="urn:s"/></n></r>`, deepEqual[seqRepeated])
	})
	t.Run("an element in the XSI namespace, under a default declaration", func(t *testing.T) {
		roundTrip(t, schemaOf(t, seqSkipSchema), rootName, seqWild{A: "x", Any: anyElement{Space: xsd.XMLSchemaInstanceNS, Local: "w"}},
			`<r xmlns="urn:s"><a>x</a><w xmlns="http://www.w3.org/2001/XMLSchema-instance"/></r>`, deepEqual[seqWild])
	})
	t.Run("an optional wildcard holding nothing", func(t *testing.T) {
		roundTrip(t, schemaOf(t, skipOptionalSchema), rootName, seqOptional{A: "x"}, `<r xmlns="urn:s"><a>x</a></r>`, deepEqual[seqOptional])
	})
	t.Run("a choice's wildcard alternative", func(t *testing.T) {
		roundTrip(t, schemaOf(t, skipPickSchema), rootName, pickWild(pickWildAny{Any: anyElement{Space: "urn:o", Local: "w"}}),
			`<r xmlns="urn:s"><w xmlns="urn:o"/></r>`, deepEqual[pickWild], pickWildBranches)
	})
	t.Run("the empty match of a choice of an optional wildcard", func(t *testing.T) {
		roundTrip(t, schemaOf(t, skipEmptySchema), rootName, skipEmpty{}, `<r xmlns="urn:s"/>`, deepEqual[skipEmpty], skipAttrBranches)
	})
	t.Run("a content choice's wildcard alternative beside an attribute", func(t *testing.T) {
		x := "v"
		roundTrip(t, schemaOf(t, skipAttrSchema), rootName, skipAttr{Choice: skipAttrAny{Any: anyElement{Space: "urn:o", Local: "w"}}, X: &x},
			`<r xmlns="urn:s" x="v"><w xmlns="urn:o"/></r>`, deepEqual[skipAttr], skipAttrBranches)
	})
	t.Run("a skip match named as the element particle beside it", func(t *testing.T) {
		roundTrip(t, schemaOf(t, seqSkipSchema), rootName, seqWild{A: "x", Any: anyElement{Space: "urn:s", Local: "a", Content: anyType{Text: "y"}}},
			`<r xmlns="urn:s"><a>x</a><a>y</a></r>`, deepEqual[seqWild])
	})
	lax := schemaOf(t, laxGovernedSchema)
	t.Run("a lax match no declaration governs", func(t *testing.T) {
		roundTrip(t, lax, rootName, seqWild{A: "x", Any: anyElement{Space: "urn:o", Local: "w", Content: anyType{Attributes: []anyAttribute{{"", "t", "v"}}, Text: "y"}}},
			`<r xmlns="urn:s"><a>x</a><w xmlns="urn:o" t="v">y</w></r>`, deepEqual[seqWild])
	})
	t.Run("a lax match holding a child no declaration governs", func(t *testing.T) {
		v := seqWild{A: "x", Any: anyElement{Space: "urn:o", Local: "p", Content: anyType{Children: []anyChild{
			{Element: anyElement{Space: "urn:o", Local: "w", Content: anyType{Attributes: []anyAttribute{{"urn:o", "t", "v"}}, Text: "z"}}, Tail: "tail"},
		}}}}
		roundTrip(t, lax, rootName, v, `<r xmlns="urn:s"><a>x</a><p xmlns="urn:o"><w xmlns:p1="urn:o" p1:t="v">z</w>tail</p></r>`, deepEqual[seqWild])
	})
	t.Run("a lax match holding a child named as the element particle beside it", func(t *testing.T) {
		v := seqWild{A: "x", Any: anyElement{Space: "urn:o", Local: "p", Content: anyType{Children: []anyChild{{Element: anyElement{Space: "urn:s", Local: "a", Content: anyType{Text: "y"}}}}}}}
		roundTrip(t, lax, rootName, v, `<r xmlns="urn:s"><a>x</a><p xmlns="urn:o"><a xmlns="urn:s">y</a></p></r>`, deepEqual[seqWild])
	})
	t.Run("a skip match holding xsi:schemaLocation", func(t *testing.T) {
		v := seqWild{A: "x", Any: anyElement{Space: "urn:o", Local: "w", Content: anyType{Attributes: []anyAttribute{{xsd.XMLSchemaInstanceNS, "schemaLocation", "urn:x x.xsd"}}}}}
		roundTrip(t, schemaOf(t, seqSkipSchema), rootName, v,
			`<r xmlns="urn:s"><a>x</a><w xmlns="urn:o" xmlns:p1="http://www.w3.org/2001/XMLSchema-instance" p1:schemaLocation="urn:x x.xsd"/></r>`, deepEqual[seqWild])
	})
	t.Run("a lax match holding xsi:noNamespaceSchemaLocation", func(t *testing.T) {
		v := seqWild{A: "x", Any: anyElement{Space: "urn:o", Local: "w", Content: anyType{Attributes: []anyAttribute{{xsd.XMLSchemaInstanceNS, "noNamespaceSchemaLocation", "n.xsd"}}}}}
		roundTrip(t, lax, rootName, v,
			`<r xmlns="urn:s"><a>x</a><w xmlns="urn:o" xmlns:p1="http://www.w3.org/2001/XMLSchema-instance" p1:noNamespaceSchemaLocation="n.xsd"/></r>`, deepEqual[seqWild])
	})
}

// TestAppendWildcardInvalid pins item 6 of Append's doc for a skip wildcard's field: a
// slice outside the particle's occurrence; an element the content model admits at no
// particle where it is written, outside the {namespace constraint} or among the
// {disallowed names}; one the content model attributes to an element particle ahead of
// the wildcard (cvc-accept), in a sequence, a choice and a content choice; and the raw
// content no document holds: a Local that is no NCName, at any depth, a Space that is
// the xmlns namespace or holds a non-Char, an unqualified attribute named xmlns, two
// attributes of one expanded name, and a Value, Text or Tail holding a non-Char; and a
// held element whose Children slice holds itself (encoder.enter). None carries an
// *xsderr.Error. A lax match outside its {namespace constraint} is item 6 too, dst
// returned unextended. Then item 7: xsi:type and xsi:nil, at any depth of a skip or a
// lax match, decline by constructXsiAttribute, behind an item 6 error found
// after it, and xsi:banana on a skip and on a lax match declines by
// constructXsiAttribute too, the two rows failing under a writer that declines
// only xsi:type and xsi:nil; and in a lax match an element or attribute a top-level
// declaration governs, the match itself, a child and an attribute, declines by
// constructGovernedContent, and a match with a ·locally declared type· (§3.3.4.6
// key-governing-type-elem clause 7) declines by constructLocallyTypedMatch: named as
// the element particle beside it in a sequence, and as an element only the base type's
// content model contains, through a choice's and a flattened choice's branch, each
// failing with writeAt.typ left unthreaded on its path; a top-level declaration
// governs, by constructGovernedContent, where both govern.
func TestAppendWildcardInvalid(t *testing.T) {
	many, one := schemaOf(t, skipManySchema), schemaOf(t, seqSkipSchema)
	w := anyElement{Space: "urn:o", Local: "w"}
	held := func(c anyType) seqWild {
		return seqWild{A: "x", Any: anyElement{Space: "urn:o", Local: "w", Content: c}}
	}
	for _, row := range []struct {
		name string
		run  func(t *testing.T) error
		want string
	}{
		{"three held elements of a wildcard of {max occurs} 2", func(t *testing.T) error {
			_, err := appendChecked(t, nil, many, strict.New(), rootName, seqRepeated{A: "x", Any: []anyElement{w, w, w}})
			return err
		}, "codec: Append: seqRepeated.Any holds 3 items, outside the occurrence"},
		{"outside the {namespace constraint}", func(t *testing.T) error {
			_, err := appendChecked(t, nil, schemaOf(t, skipTargetSchema), strict.New(), rootName, seqWild{A: "x", Any: w})
			return err
		}, "codec: Append: seqWild.Any holds element {urn:o}w, which no particle of the content model of "},
		{"a {disallowed names} QName", func(t *testing.T) error {
			_, err := appendChecked(t, nil, schemaOf(t, skipNotSchema), strict.New(), rootName, seqWild{A: "x", Any: anyElement{Space: "urn:s", Local: "x"}})
			return err
		}, "codec: Append: seqWild.Any holds element {urn:s}x, which no particle of the content model of "},
		{"a name the element particle after the wildcard takes", func(t *testing.T) error {
			_, err := appendChecked(t, nil, schemaOf(t, skipLeadingSchema), strict.New(), rootName, seqLeading{Any: &anyElement{Space: "urn:s", Local: "a"}, A: "x"})
			return err
		}, "codec: Append: seqLeading.Any holds element {urn:s}a, which the content model of "},
		{"a name the choice's element alternative takes", func(t *testing.T) error {
			_, err := appendChecked(t, nil, schemaOf(t, skipPickSchema), strict.New(), rootName, pickWild(pickWildAny{Any: anyElement{Space: "urn:s", Local: "a"}}), pickWildBranches)
			return err
		}, "codec: Append: pickWild.(pickWildAny).Any holds element {urn:s}a, which the content model of "},
		{"a name a content choice's element alternative takes", func(t *testing.T) error {
			_, err := appendChecked(t, nil, schemaOf(t, skipAttrSchema), strict.New(), rootName, skipAttr{Choice: skipAttrAny{Any: anyElement{Space: "urn:s", Local: "a"}}}, skipAttrBranches)
			return err
		}, "codec: Append: skipAttr.Choice.(skipAttrAny).Any holds element {urn:s}a, which the content model of "},
		{"a Local that is no NCName", func(t *testing.T) error {
			_, err := appendChecked(t, nil, one, strict.New(), rootName, seqWild{A: "x", Any: anyElement{Space: "urn:o", Local: "1w"}})
			return err
		}, `codec: Append: seqWild.Any holds the local name "1w", which is no NCName`},
		{"a Local holding a byte that is not UTF-8", func(t *testing.T) error {
			_, err := appendChecked(t, nil, one, strict.New(), rootName, seqWild{A: "x", Any: anyElement{Space: "urn:o", Local: "w\xff"}})
			return err
		}, `codec: Append: seqWild.Any holds the local name "w\xff", which is no NCName`},
		{"an empty Local", func(t *testing.T) error {
			_, err := appendChecked(t, nil, one, strict.New(), rootName, seqWild{A: "x", Any: anyElement{Space: "urn:o"}})
			return err
		}, `codec: Append: seqWild.Any holds the local name "", which is no NCName`},
		{"a child's Local holding a colon", func(t *testing.T) error {
			_, err := appendChecked(t, nil, one, strict.New(), rootName, held(anyType{Children: []anyChild{{Element: anyElement{Local: "a:b"}}}}))
			return err
		}, `codec: Append: seqWild.Any.Content.Children[0].Element holds the local name "a:b", which is no NCName`},
		{"a Space that is the xmlns namespace", func(t *testing.T) error {
			_, err := appendChecked(t, nil, one, strict.New(), rootName, seqWild{A: "x", Any: anyElement{Space: xmlnsNamespace, Local: "w"}})
			return err
		}, "codec: Append: seqWild.Any holds {http://www.w3.org/2000/xmlns/}w, whose namespace name is the one the prefix xmlns is bound to"},
		{"a Space holding a non-Char", func(t *testing.T) error {
			_, err := appendChecked(t, nil, one, strict.New(), rootName, seqWild{A: "x", Any: anyElement{Space: "urn:\xff", Local: "w"}})
			return err
		}, `codec: Append: seqWild.Any holds the namespace name "urn:\xff", whose byte offset 4 starts no XML 1.0 Char`},
		{"an attribute's Local that is no NCName", func(t *testing.T) error {
			_, err := appendChecked(t, nil, one, strict.New(), rootName, held(anyType{Attributes: []anyAttribute{{"", "-k", "v"}}}))
			return err
		}, `codec: Append: seqWild.Any.Content.Attributes[0] holds the local name "-k", which is no NCName`},
		{"an attribute named xmlns", func(t *testing.T) error {
			_, err := appendChecked(t, nil, one, strict.New(), rootName, held(anyType{Attributes: []anyAttribute{{"", "xmlns", "urn:q"}}}))
			return err
		}, "codec: Append: seqWild.Any.Content.Attributes[0] holds the attribute xmlns, a namespace declaration"},
		{"an attribute in the xmlns namespace", func(t *testing.T) error {
			_, err := appendChecked(t, nil, one, strict.New(), rootName, held(anyType{Attributes: []anyAttribute{{xmlnsNamespace, "q", "urn:q"}}}))
			return err
		}, "codec: Append: seqWild.Any.Content.Attributes[0] holds {http://www.w3.org/2000/xmlns/}q, whose namespace name is the one the prefix xmlns is bound to"},
		{"two attributes of one expanded name", func(t *testing.T) error {
			_, err := appendChecked(t, nil, one, strict.New(), rootName, held(anyType{Attributes: []anyAttribute{{"urn:q", "k", "1"}, {"", "j", "2"}, {"urn:q", "k", "3"}}}))
			return err
		}, "codec: Append: seqWild.Any.Content.Attributes[2] holds the attribute {urn:q}k, as item 0 before it does"},
		{"a Value holding a non-Char", func(t *testing.T) error {
			_, err := appendChecked(t, nil, one, strict.New(), rootName, held(anyType{Attributes: []anyAttribute{{"", "k", "a\x00"}}}))
			return err
		}, `codec: Append: seqWild.Any.Content.Attributes[0].Value holds "a\x00", whose byte offset 1 starts no XML 1.0 Char`},
		{"a Text holding a non-Char", func(t *testing.T) error {
			_, err := appendChecked(t, nil, one, strict.New(), rootName, held(anyType{Text: "\x01"}))
			return err
		}, `codec: Append: seqWild.Any.Content.Text holds "\x01", whose byte offset 0 starts no XML 1.0 Char`},
		{"a Tail holding a non-Char", func(t *testing.T) error {
			_, err := appendChecked(t, nil, one, strict.New(), rootName, held(anyType{Children: []anyChild{{Element: w, Tail: "ok\xff"}}}))
			return err
		}, `codec: Append: seqWild.Any.Content.Children[0].Tail holds "ok\xff", whose byte offset 2 starts no XML 1.0 Char`},
		{"an XSI attribute, then a Text holding a non-Char", func(t *testing.T) error {
			_, err := appendChecked(t, nil, one, strict.New(), rootName, held(anyType{Attributes: []anyAttribute{{xsd.XMLSchemaInstanceNS, "type", "xs:byte"}}, Text: "\x01"}))
			return err
		}, `codec: Append: seqWild.Any.Content.Text holds "\x01"`},
		{"a Children slice that holds itself", func(t *testing.T) error {
			c := make([]anyChild, 1)
			c[0].Element = anyElement{Local: "c"}
			c[0].Element.Content.Children = c
			_, err := appendChecked(t, nil, one, strict.New(), rootName, held(anyType{Children: c}))
			return err
		}, "codec: Append: seqWild.Any.Content.Children[0].Element.Content.Children[0].Element.Content is the value at seqWild.Any.Content.Children[0].Element.Content, whose element is still being written, a value that holds itself, which no finite document holds"},
	} {
		t.Run(row.name, func(t *testing.T) {
			err := row.run(t)
			invalidValue(t, err, row.want)
			if !strings.HasPrefix(err.Error(), row.want) {
				t.Errorf("error %q does not open with %q", err, row.want)
			}
			noRule(t, err)
		})
	}
	t.Run("a lax match outside the {namespace constraint}", func(t *testing.T) {
		dst := []byte("<prior/>")
		out, err := appendChecked(t, dst, schemaOf(t, laxTargetSchema), strict.New(), rootName, seqWild{A: "x", Any: w})
		want := "codec: Append: seqWild.Any holds element {urn:o}w, which no particle of the content model of "
		invalidValue(t, err, want)
		if !strings.HasPrefix(err.Error(), want) {
			t.Errorf("error %q does not open with %q", err, want)
		}
		if !bytes.Equal(out, dst) {
			t.Errorf("Append returned %q, want dst %q unextended", out, dst)
		}
	})
	lax := schemaOf(t, laxGovernedSchema)
	laxHeld := func(c anyType) seqWild {
		return seqWild{A: "x", Any: anyElement{Space: "urn:o", Local: "p", Content: c}}
	}
	const match = ", in the match of wildcard particle 1 of the {type definition} of element {urn:s}r declared at "
	for _, row := range []struct {
		name      string
		schema    *xsd.Schema
		construct construct
		v         seqWild
		want      []string
	}{
		{"xsi:type on the match", one, constructXsiAttribute, held(anyType{Attributes: []anyAttribute{{"", "k", "v"}, {xsd.XMLSchemaInstanceNS, "type", "xs:byte"}}}),
			[]string{"attribute xsi:type at seqWild.Any.Content.Attributes[1]" + match}},
		{"xsi:nil on a child of the match", one, constructXsiAttribute, held(anyType{Children: []anyChild{{Element: anyElement{Local: "c", Content: anyType{Attributes: []anyAttribute{{xsd.XMLSchemaInstanceNS, "nil", "true"}}}}}}}),
			[]string{"attribute xsi:nil at seqWild.Any.Content.Children[0].Element.Content.Attributes[0]" + match}},
		{"xsi:banana on the match", one, constructXsiAttribute, held(anyType{Attributes: []anyAttribute{{xsd.XMLSchemaInstanceNS, "banana", "1"}}}),
			[]string{"attribute xsi:banana at seqWild.Any.Content.Attributes[0]" + match}},
		{"a lax match a top-level declaration governs", lax, constructGovernedContent, seqWild{A: "x", Any: anyElement{Space: "urn:s", Local: "w", Content: anyType{Text: "abc"}}},
			[]string{"element {urn:s}w at seqWild.Any" + match, ", governed by element {urn:s}w declared at "}},
		{"a lax match's child a top-level declaration governs", lax, constructGovernedContent, laxHeld(anyType{Children: []anyChild{{Element: anyElement{Space: "urn:s", Local: "w", Content: anyType{Text: "abc"}}}}}),
			[]string{"element {urn:s}w at seqWild.Any.Content.Children[0].Element" + match, ", governed by element {urn:s}w declared at "}},
		{"a lax match's attribute a top-level declaration governs", lax, constructGovernedContent, laxHeld(anyType{Attributes: []anyAttribute{{"urn:s", "t", "abc"}}}),
			[]string{"attribute {urn:s}t at seqWild.Any.Content.Attributes[0]" + match, ", governed by attribute {urn:s}t declared at "}},
		{"xsi:type on a lax match", lax, constructXsiAttribute, laxHeld(anyType{Attributes: []anyAttribute{{xsd.XMLSchemaInstanceNS, "type", "xs:int"}}, Text: "abc"}),
			[]string{"attribute xsi:type at seqWild.Any.Content.Attributes[0]" + match}},
		{"xsi:banana on a lax match", lax, constructXsiAttribute, laxHeld(anyType{Attributes: []anyAttribute{{xsd.XMLSchemaInstanceNS, "banana", "1"}}}),
			[]string{"attribute xsi:banana at seqWild.Any.Content.Attributes[0]" + match}},
		{"a lax match named as the element particle beside it", lax, constructLocallyTypedMatch, seqWild{A: "1", Any: anyElement{Space: "urn:s", Local: "a", Content: anyType{Text: "hello"}}},
			[]string{"element {urn:s}a at seqWild.Any" + match, ", governed by its locally declared type in the {type definition} of element {urn:s}r declared at "}},
		{"a lax match a top-level declaration and its locally declared type both govern", schemaOf(t, seqLaxSchema+`<xs:element name="a" type="xs:string"/>`), constructGovernedContent,
			seqWild{A: "1", Any: anyElement{Space: "urn:s", Local: "a", Content: anyType{Text: "hello"}}},
			[]string{"element {urn:s}a at seqWild.Any" + match, ", governed by element {urn:s}a declared at "}},
	} {
		t.Run(row.name, func(t *testing.T) {
			_, err := appendChecked(t, nil, row.schema, strict.New(), rootName, row.v)
			_ = asDecline(t, err, "codec does not encode "+row.construct.name()+": "+row.want[0])
			holds(t, err, row.want[1:]...)
		})
	}
	const local = ", governed by its locally declared type in the {type definition} of element {urn:s}r declared at "
	c := anyElement{Space: "urn:s", Local: "c", Content: anyType{Text: "hello"}}
	t.Run("a choice's lax match named as an element of the base type", func(t *testing.T) {
		_, err := appendChecked(t, nil, schemaOf(t, laxPickBaseSchema), strict.New(), rootName, pickWild(pickWildAny{Any: c}), pickWildBranches)
		_ = asDecline(t, err, "codec does not encode "+constructLocallyTypedMatch.name()+": element {urn:s}c at pickWild.(pickWildAny).Any, in the match of ")
		holds(t, err, local)
	})
	t.Run("a flattened lax match named as an element of the base type", func(t *testing.T) {
		_, err := appendChecked(t, nil, schemaOf(t, laxFlatBaseSchema), strict.New(), rootName, flatPick{Choice: []flatPickItem{flatPickAny{Any: c}}}, flatPickBranches)
		_ = asDecline(t, err, "codec does not encode "+constructLocallyTypedMatch.name()+": element {urn:s}c at flatPick.Choice[0].(flatPickAny).Any, in the match of ")
		holds(t, err, local)
	})
}

// governedAnySchema declares, beside s:r of type xs:anyType itself, the
// top-level declarations that govern an item so named in its content: q of
// xs:QName, i of xs:int, and the attribute qa of xs:QName.
const governedAnySchema = `<xs:element name="r"/><xs:element name="q" type="xs:QName"/><xs:element name="i" type="xs:int"/><xs:attribute name="qa" type="xs:QName"/>`

// Schemas beside anyattribute_test.go's, each of whose s:r is of awOne's
// shape: awNotSchema's skip wildcard names s:no in its {disallowed names},
// awDefinedSchema's lax one ##defined beside a top-level attribute g, and
// awLaxGovernedSchema's lax one sits beside a top-level attribute t.
const (
	awNotSchema         = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="e" type="xs:string"/></xs:sequence><xs:anyAttribute notQName="s:no" processContents="skip"/></xs:complexType></xs:element>`
	awDefinedSchema     = `<xs:attribute name="g" type="xs:int"/><xs:element name="r"><xs:complexType><xs:sequence><xs:element name="e" type="xs:string"/></xs:sequence><xs:anyAttribute notQName="##defined" processContents="lax"/></xs:complexType></xs:element>`
	awLaxGovernedSchema = `<xs:attribute name="t" type="xs:int"/><xs:element name="r"><xs:complexType><xs:sequence><xs:element name="e" type="xs:string"/></xs:sequence><xs:anyAttribute processContents="lax"/></xs:complexType></xs:element>`
)

// TestAppendAttributeWildcard pins the {attribute wildcard} "# What encodes"
// admits: each attribute its field holds written after the attribute uses'
// attributes, in held order, raw, as a wildcard particle's match's are, under
// a skip wildcard, the effective one an extension unites with its base's
// (awUnionSchema), and one a top-level declaration names, which governs
// nothing a skip wildcard admits (§3.10.4.1 key-skipped), and a lax one no
// top-level declaration governs, a use's attribute ahead, on the document
// element, beside simple content and on a particle's element; and an empty
// field writes no attribute, whatever the {process contents}, strict
// included. Each document validate accepts and Decode turns back into the
// value. Every row fails with complexEncodes' static decline of a type with
// an {attribute wildcard} put back.
func TestAppendAttributeWildcard(t *testing.T) {
	one := int32(1)
	t.Run("a skip wildcard's attribute in an admitted namespace", func(t *testing.T) {
		roundTrip(t, schemaOf(t, awUnionSchema), rootName, awOne{E: "v", AnyAttribute: []anyAttribute{{"urn:x", "q", "1"}, {"urn:b", "p", `<&"`}}},
			`<r xmlns="urn:s" xmlns:p1="urn:x" p1:q="1" xmlns:p2="urn:b" p2:p="&lt;&amp;&quot;"><e>v</e></r>`, deepEqual[awOne])
	})
	t.Run("a lax wildcard's undeclared attributes after a use's", func(t *testing.T) {
		roundTrip(t, schemaOf(t, awUseSchema+`<xs:attribute name="t" type="xs:int"/>`), rootName, awUse{E: "v", A: &one, AnyAttribute: []anyAttribute{{"", "y", "3"}, {"urn:p", "x", "2"}, {xmltree.XMLNamespaceURI, "lang", "en"}}},
			`<r xmlns="urn:s" a="1" y="3" xmlns:p1="urn:p" p1:x="2" xml:lang="en"><e>v</e></r>`, deepEqual[awUse])
	})
	t.Run("a skip wildcard's attribute a top-level declaration names", func(t *testing.T) {
		roundTrip(t, schemaOf(t, awNotSchema+`<xs:attribute name="t" type="xs:int"/>`), rootName, awOne{E: "v", AnyAttribute: []anyAttribute{{"urn:s", "t", "abc"}}},
			`<r xmlns="urn:s" xmlns:p1="urn:s" p1:t="abc"><e>v</e></r>`, deepEqual[awOne])
	})
	t.Run("an attribute in the element's own namespace", func(t *testing.T) {
		roundTrip(t, schemaOf(t, awUseSchema), rootName, awUse{E: "v", AnyAttribute: []anyAttribute{{"urn:s", "y", "3"}}},
			`<r xmlns="urn:s" xmlns:p1="urn:s" p1:y="3"><e>v</e></r>`, deepEqual[awUse])
	})
	t.Run("beside simple content", func(t *testing.T) {
		v := awSimple{}
		v.A.Value = "v"
		v.A.AnyAttribute = []anyAttribute{{"", "w", "1"}}
		roundTrip(t, schemaOf(t, awSimpleSchema), rootName, v, `<r xmlns="urn:s"><a w="1">v</a></r>`, deepEqual[awSimple])
	})
	t.Run("on a particle's element", func(t *testing.T) {
		v := awNested{}
		v.A.B = "x"
		v.A.AnyAttribute = []anyAttribute{{"", "w", "1"}}
		roundTrip(t, schemaOf(t, awNestedSchema), rootName, v, `<r xmlns="urn:s"><a w="1"><b>x</b></a></r>`, deepEqual[awNested])
	})
	for _, row := range []struct {
		name, decls string
		v           awOne
		want        string
	}{
		{"an empty skip field", awUnionSchema, awOne{E: "v"}, `<r xmlns="urn:s"><e>v</e></r>`},
		{"an empty strict field", awStrictSchema, awOne{E: "v"}, `<r xmlns="urn:s"><e>v</e></r>`},
		{"an empty lax field, defaultAttributes's", awDefaultDoc, awOne{E: "v", AnyAttribute: []anyAttribute{}}, `<r xmlns="urn:s"><e>v</e></r>`},
	} {
		t.Run(row.name, func(t *testing.T) {
			roundTrip(t, awSchema(t, row.decls), rootName, row.v, row.want, func(a, b awOne) bool { return a.E == b.E && len(a.AnyAttribute) == 0 && len(b.AnyAttribute) == 0 })
		})
	}
	t.Run("an empty field of empty content", func(t *testing.T) {
		roundTrip(t, schemaOf(t, awEmptySchema), rootName, awEmpty{}, `<r xmlns="urn:s"/>`, deepEqual[awEmpty])
	})
}

// TestAppendAttributeWildcardInvalid pins item 6 of Append's doc for an
// {attribute wildcard}'s field, dst returned unextended: an attribute whose
// expanded name an attribute use names, its field nil and set (§3.4.4.2
// cvc-complex-type clause 2.1), which Decode never returns; one the wildcard
// does not admit (clause 2.2.2, §3.10.4.1 cvc-wildcard), its namespace outside
// the {namespace constraint}, an XSI one other than the four clause 2 excepts
// among them, its name in {disallowed names}, and one ##defined excludes,
// though a lax wildcard would decline it as governed; rawAttribute's own, a
// Local that is no NCName and two attributes of one expanded name; and one
// found after a strict wildcard's decline, which item 6 outranks. Then item 7:
// a strict wildcard's attribute, governed by a top-level declaration or by
// none, declines by constructStrictAttributeWildcard; a lax wildcard's
// governed one by constructGovernedContent; and an XSI attribute, xsi:type
// under a wildcard whose {namespace constraint} excludes the XSI namespace
// and xsi:schemaLocation under a skip, a lax and a strict wildcard among
// them, by constructXsiAttribute; the three xsi:schemaLocation rows fail
// with attributeWildcardHold.writesHints answering true. Each item 6 row
// returns nil, having written the attribute, under a writer that writes the
// field unchecked.
func TestAppendAttributeWildcardInvalid(t *testing.T) {
	one := int32(1)
	held := func(a ...anyAttribute) awOne { return awOne{E: "v", AnyAttribute: a} }
	const field = ", in the {attribute wildcard} field of the {type definition} of element {urn:s}r declared at "
	for _, row := range []struct {
		name, decls string
		v           any
		want        string
	}{
		{"a use's expanded name, its field nil", awUseSchema, awUse{E: "v", AnyAttribute: []anyAttribute{{"", "a", "1"}}},
			"codec: Append: awUse.AnyAttribute[0] holds the attribute a, whose expanded name an attribute use of the {type definition} of element {urn:s}r declared at "},
		{"a use's expanded name, its field set", awUseSchema, awUse{E: "v", A: &one, AnyAttribute: []anyAttribute{{"", "a", "1"}}},
			"codec: Append: awUse.AnyAttribute[0] holds the attribute a, whose expanded name an attribute use of the {type definition} of element {urn:s}r declared at "},
		{"a namespace outside the {namespace constraint}", awUnionSchema, held(anyAttribute{"urn:x", "q", "1"}, anyAttribute{"urn:o", "q", "1"}),
			"codec: Append: awOne.AnyAttribute[1] holds the attribute {urn:o}q, which the {attribute wildcard} of the {type definition} of element {urn:s}r declared at "},
		{"an XSI attribute outside the {namespace constraint}", awUnionSchema, held(anyAttribute{xsd.XMLSchemaInstanceNS, "banana", "1"}),
			"codec: Append: awOne.AnyAttribute[0] holds the attribute xsi:banana, which the {attribute wildcard} of "},
		{"a {disallowed names} QName", awNotSchema, held(anyAttribute{"urn:s", "no", "1"}),
			"codec: Append: awOne.AnyAttribute[0] holds the attribute {urn:s}no, which the {attribute wildcard} of "},
		{"a name ##defined excludes", awDefinedSchema, held(anyAttribute{"urn:s", "g", "1"}),
			"codec: Append: awOne.AnyAttribute[0] holds the attribute {urn:s}g, which the {attribute wildcard} of "},
		{"a Local that is no NCName", awUseSchema, awUse{E: "v", AnyAttribute: []anyAttribute{{"", "-k", "1"}}},
			`codec: Append: awUse.AnyAttribute[0] holds the local name "-k", which is no NCName`},
		{"two attributes of one expanded name", awUseSchema, awUse{E: "v", AnyAttribute: []anyAttribute{{"urn:q", "k", "1"}, {"urn:q", "k", "2"}}},
			"codec: Append: awUse.AnyAttribute[1] holds the attribute {urn:q}k, as item 0 before it does"},
		{"an invalid attribute after a strict one", awStrictSchema, held(anyAttribute{"urn:s", "g", "5"}, anyAttribute{"urn:o", "q", "1"}),
			"codec: Append: awOne.AnyAttribute[1] holds the attribute {urn:o}q, which the {attribute wildcard} of "},
	} {
		t.Run(row.name, func(t *testing.T) {
			dst := []byte("<prior/>")
			var out []byte
			var err error
			// The row's T is its v's own type, so that CheckAppend[T] reads it.
			switch v := row.v.(type) {
			case awUse:
				out, err = appendChecked(t, dst, schemaOf(t, row.decls), strict.New(), rootName, v)
			case awOne:
				out, err = appendChecked(t, dst, schemaOf(t, row.decls), strict.New(), rootName, v)
			}
			invalidValue(t, err, row.want)
			if !strings.HasPrefix(err.Error(), row.want) {
				t.Errorf("error %q does not open with %q", err, row.want)
			}
			noRule(t, err)
			if !bytes.Equal(out, dst) {
				t.Errorf("Append returned %q, want dst %q unextended", out, dst)
			}
		})
	}
	for _, row := range []struct {
		name, decls string
		construct   construct
		v           awOne
		want        []string
	}{
		{"a strict wildcard's attribute a top-level declaration governs", awStrictSchema, constructStrictAttributeWildcard, held(anyAttribute{"urn:s", "g", "5"}),
			[]string{"attribute {urn:s}g at awOne.AnyAttribute[0]" + field}},
		{"a strict wildcard's attribute no declaration governs", awStrictSchema, constructStrictAttributeWildcard, held(anyAttribute{"urn:s", "h", "5"}),
			[]string{"attribute {urn:s}h at awOne.AnyAttribute[0]" + field}},
		{"xsi:type under a skip wildcard", awNotSchema, constructXsiAttribute, held(anyAttribute{xsd.XMLSchemaInstanceNS, "type", "xs:int"}),
			[]string{"attribute xsi:type at awOne.AnyAttribute[0]" + field}},
		{"xsi:type under a wildcard excluding the XSI namespace", awUnionSchema, constructXsiAttribute, held(anyAttribute{xsd.XMLSchemaInstanceNS, "type", "xs:int"}),
			[]string{"attribute xsi:type at awOne.AnyAttribute[0]" + field}},
		{"an XSI attribute the wildcard admits", awNotSchema, constructXsiAttribute, held(anyAttribute{xsd.XMLSchemaInstanceNS, "banana", "1"}),
			[]string{"attribute xsi:banana at awOne.AnyAttribute[0]" + field}},
		{"xsi:schemaLocation under a skip wildcard", awNotSchema, constructXsiAttribute, held(anyAttribute{xsd.XMLSchemaInstanceNS, "schemaLocation", "urn:x x.xsd"}),
			[]string{"attribute xsi:schemaLocation at awOne.AnyAttribute[0]" + field}},
		{"xsi:schemaLocation under a lax wildcard", awLaxGovernedSchema, constructXsiAttribute, held(anyAttribute{xsd.XMLSchemaInstanceNS, "schemaLocation", "urn:x x.xsd"}),
			[]string{"attribute xsi:schemaLocation at awOne.AnyAttribute[0]" + field}},
		{"xsi:schemaLocation under a strict wildcard", awStrictSchema, constructXsiAttribute, held(anyAttribute{xsd.XMLSchemaInstanceNS, "schemaLocation", "urn:x x.xsd"}),
			[]string{"attribute xsi:schemaLocation at awOne.AnyAttribute[0]" + field}},
		{"a lax wildcard's attribute a top-level declaration governs", awLaxGovernedSchema, constructGovernedContent, held(anyAttribute{"", "y", "3"}, anyAttribute{"urn:s", "t", "abc"}),
			[]string{"attribute {urn:s}t at awOne.AnyAttribute[1]" + field, ", governed by attribute {urn:s}t declared at "}},
	} {
		t.Run(row.name, func(t *testing.T) {
			dst := []byte("<prior/>")
			out, err := appendChecked(t, dst, schemaOf(t, row.decls), strict.New(), rootName, row.v)
			_ = asDecline(t, err, "codec does not encode "+row.construct.name()+": "+row.want[0])
			holds(t, err, row.want[1:]...)
			if !bytes.Equal(out, dst) {
				t.Errorf("Append returned %q, want dst %q unextended", out, dst)
			}
		})
	}
}

// TestAppendAnyType pins the element declaration of type xs:anyType itself
// "# What encodes" admits, the document element's and a particle's, written
// raw as a skip wildcard's match is: the document element's content, into its
// named type and into AnyType itself, holding text, an unqualified attribute,
// a qualified one and an xml:lang, and nested children; a particle's once,
// local, as an <element ref> into its named type, and beside another;
// repeated and optional, an absent optional writing nothing; and a choice's
// local and <element ref> alternatives; and xsi:schemaLocation on the
// document element and xsi:noNamespaceSchemaLocation on a descendant. Each
// document validate accepts and Decode turns back into the value. Every row
// declines by item 4 with the arm removed (elementEncodes' xs:anyType case
// returning a decline). The two XSI rows, and no other row here, fail with
// rawAttribute's hint exemption removed, and with rawAttributes asking
// encoder.governed of a hint, which Schema.Attribute resolves to its
// built-in declaration.
func TestAppendAnyType(t *testing.T) {
	content := anyType{
		Attributes: []anyAttribute{{"", "k", "v"}, {"urn:z", "y", "1"}, {xmltree.XMLNamespaceURI, "lang", "en"}},
		Text:       "t ",
		Children: []anyChild{
			{Element: anyElement{Space: "urn:s", Local: "c", Content: anyType{Text: "in"}}, Tail: "out"},
			{Element: anyElement{Space: "urn:o", Local: "d", Content: anyType{Children: []anyChild{{Element: anyElement{Local: "e"}}}}}},
		},
	}
	const written = ` k="v" xmlns:p1="urn:z" p1:y="1" xml:lang="en">t <c>in</c>out<d xmlns="urn:o"><e xmlns=""/></d>`
	t.Run("the document element", func(t *testing.T) {
		roundTrip(t, schemaOf(t, rootAnySchema), rootName, rootAny(content), `<r xmlns="urn:s"`+written+`</r>`, deepEqual[rootAny])
	})
	t.Run("the document element of type xs:anyType, into AnyType itself", func(t *testing.T) {
		roundTrip(t, schemaOf(t, typedRootAnySchema), rootName, content, `<r xmlns="urn:s"`+written+`</r>`, deepEqual[anyType])
	})
	t.Run("an empty document element", func(t *testing.T) {
		roundTrip(t, schemaOf(t, rootAnySchema), rootName, rootAny{}, `<r xmlns="urn:s"/>`, deepEqual[rootAny])
	})
	t.Run("a particle once", func(t *testing.T) {
		roundTrip(t, schemaOf(t, oneAnySchema), rootName, oneAny{A: content}, `<r xmlns="urn:s"><a`+written+`</a></r>`, deepEqual[oneAny])
	})
	t.Run("an <element ref> particle", func(t *testing.T) {
		roundTrip(t, schemaOf(t, refAnySchema), rootName, refAny{F: free{Text: "x"}}, `<r xmlns="urn:s"><f>x</f></r>`, deepEqual[refAny])
	})
	t.Run("two particles", func(t *testing.T) {
		roundTrip(t, schemaOf(t, pairAnySchema), rootName, pairAny{A: anyType{Text: "1"}, B: content}, `<r xmlns="urn:s"><a>1</a><b`+written+`</b></r>`, deepEqual[pairAny])
	})
	t.Run("a repeated particle and an optional one", func(t *testing.T) {
		roundTrip(t, schemaOf(t, occursAnySchema), rootName, occursAny{A: []anyType{{Text: "1"}, {}}, B: &anyType{Text: "2"}}, `<r xmlns="urn:s"><a>1</a><a/><b>2</b></r>`, deepEqual[occursAny])
	})
	t.Run("an absent optional particle", func(t *testing.T) {
		roundTrip(t, schemaOf(t, occursAnySchema), rootName, occursAny{A: []anyType{{}}}, `<r xmlns="urn:s"><a/></r>`, deepEqual[occursAny])
	})
	t.Run("a choice's local alternative", func(t *testing.T) {
		roundTrip(t, schemaOf(t, pickAnySchema), rootName, pickAny(pickAnyA{A: anyType{Text: "x"}}), `<r xmlns="urn:s"><a>x</a></r>`, deepEqual[pickAny], pickAnyBranches)
	})
	t.Run("a choice's <element ref> alternative", func(t *testing.T) {
		roundTrip(t, schemaOf(t, pickRefAnySchema), rootName, pickRefAny(pickRefAnyF{F: free{Text: "x"}}), `<r xmlns="urn:s"><f>x</f></r>`, deepEqual[pickRefAny], pickRefAnyBranches)
	})
	t.Run("xsi:schemaLocation on the document element", func(t *testing.T) {
		roundTrip(t, schemaOf(t, rootAnySchema), rootName, rootAny{Attributes: []anyAttribute{{xsd.XMLSchemaInstanceNS, "schemaLocation", "urn:x x.xsd"}}},
			`<r xmlns="urn:s" xmlns:p1="http://www.w3.org/2001/XMLSchema-instance" p1:schemaLocation="urn:x x.xsd"/>`, deepEqual[rootAny])
	})
	t.Run("xsi:noNamespaceSchemaLocation on a descendant", func(t *testing.T) {
		v := rootAny{Children: []anyChild{{Element: anyElement{Local: "c", Content: anyType{Attributes: []anyAttribute{{xsd.XMLSchemaInstanceNS, "noNamespaceSchemaLocation", "n.xsd"}}}}}}}
		roundTrip(t, schemaOf(t, rootAnySchema), rootName, v,
			`<r xmlns="urn:s"><c xmlns="" xmlns:p1="http://www.w3.org/2001/XMLSchema-instance" p1:noNamespaceSchemaLocation="n.xsd"/></r>`, deepEqual[rootAny])
	})
}

// TestAppendAnyTypeGuards pins what Append does with the content of an
// element of type xs:anyType itself beyond what it writes, each row but the
// last two answered, before Append encoded such an element, by item 4's
// decline of its declaration. Item 6 of Append's doc: a held
// Local that is no NCName, at depth, a Text holding a non-Char, a held
// attribute named xmlns, and a held element's self-holding Children. Item 7:
// xsi:type, on the element and at depth, xsi:nil, and xsi:banana on the
// element decline by constructXsiAttribute, each xsi:type row written, and
// charged by validate by cvc-attribute, its prefix unbound, under a writer
// that admits every XSI attribute and asks encoder.governed of none, and the
// xsi:banana row failing under a writer that declines only xsi:type and
// xsi:nil; and an element whose expanded name a top-level element declaration
// names, or an attribute but for the two schema location hints whose expanded
// name a top-level attribute declaration names, at any depth, under an
// ungoverned child too, declines by constructGovernedContent, whatever its
// value, the XSI check first on one attribute, since the schema declares
// xsi:type: a governed xs:QName child Decode returns from a valid document, a
// governed xs:int child built invalid against its declaration, a governed
// attribute on the element and on a descendant, and a governed child of a
// particle's element. An item 6 error inside a governed element's content, and
// one in a later sibling, wins over that decline. A skip wildcard's match
// holding a governed name is still written, and a type derived from xs:anyType
// still encodes as its own complex type.
func TestAppendAnyTypeGuards(t *testing.T) {
	governed := schemaOf(t, governedAnySchema)
	child := func(e anyElement) []anyChild { return []anyChild{{Element: e}} }
	q := anyElement{Space: "urn:s", Local: "q", Content: anyType{Text: "s:x"}}
	for _, row := range []struct {
		name string
		v    rootAny
		want string
	}{
		{"a held Local that is no NCName, at depth", rootAny{Children: child(anyElement{Local: "c", Content: anyType{Children: child(anyElement{Local: "1c"})}})},
			`codec: Append: rootAny.Children[0].Element.Content.Children[0].Element holds the local name "1c", which is no NCName`},
		{"a Text holding a non-Char", rootAny{Text: "\x01"}, `codec: Append: rootAny.Text holds "\x01", whose byte offset 0 starts no XML 1.0 Char`},
		{"an attribute named xmlns", rootAny{Attributes: []anyAttribute{{"", "xmlns", "urn:q"}}}, "codec: Append: rootAny.Attributes[0] holds the attribute xmlns, a namespace declaration"},
		{"a governed element holding a non-Char", rootAny{Children: child(anyElement{Space: "urn:s", Local: "q", Content: anyType{Text: "\x01"}})},
			`codec: Append: rootAny.Children[0].Element.Content.Text holds "\x01"`},
		{"a governed element, then a sibling's Tail holding a non-Char", rootAny{Children: []anyChild{{Element: q}, {Element: anyElement{Local: "c"}, Tail: "\xff"}}},
			`codec: Append: rootAny.Children[1].Tail holds "\xff"`},
	} {
		t.Run(row.name, func(t *testing.T) {
			_, err := appendChecked(t, nil, governed, strict.New(), rootName, row.v)
			invalidValue(t, err, row.want)
			if !strings.HasPrefix(err.Error(), row.want) {
				t.Errorf("error %q does not open with %q", err, row.want)
			}
			noRule(t, err)
		})
	}
	t.Run("a held element whose Children slice holds itself", func(t *testing.T) {
		c := make([]anyChild, 1)
		c[0].Element = anyElement{Local: "c"}
		c[0].Element.Content.Children = c
		_, err := appendChecked(t, nil, governed, strict.New(), rootName, rootAny{Children: c})
		invalidValue(t, err, "codec: Append: rootAny.Children[0].Element.Content.Children[0].Element.Content is the value at rootAny.Children[0].Element.Content, whose element is still being written")
	})
	const in = "in the content of element {urn:s}r declared at "
	decoded, err := Decode[rootAny](governed, strict.New(), instanceOf("", `<s:q>s:x</s:q>`))
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		name, construct string
		v               rootAny
		want            []string
	}{
		{"xsi:type on the element", constructXsiAttribute.name(), rootAny{Attributes: []anyAttribute{{"", "k", "v"}, {xsd.XMLSchemaInstanceNS, "type", "xs:string"}}},
			[]string{"attribute xsi:type at rootAny.Attributes[1], " + in}},
		{"xsi:type at depth", constructXsiAttribute.name(), rootAny{Children: child(anyElement{Local: "c", Content: anyType{Attributes: []anyAttribute{{xsd.XMLSchemaInstanceNS, "type", "x:t"}}}})},
			[]string{"attribute xsi:type at rootAny.Children[0].Element.Content.Attributes[0], " + in}},
		{"xsi:banana on the element", constructXsiAttribute.name(), rootAny{Attributes: []anyAttribute{{xsd.XMLSchemaInstanceNS, "banana", "1"}}},
			[]string{"attribute xsi:banana at rootAny.Attributes[0], " + in}},
		{"a governed xs:QName child decoded from a valid document", constructGovernedContent.name(), *decoded,
			[]string{"element {urn:s}q at rootAny.Children[0].Element, " + in, ", governed by element {urn:s}q declared at "}},
		{"a governed xs:int child built invalid against its declaration", constructGovernedContent.name(), rootAny{Children: child(anyElement{Space: "urn:s", Local: "i", Content: anyType{Text: "abc"}})},
			[]string{"element {urn:s}i at rootAny.Children[0].Element, " + in, ", governed by element {urn:s}i declared at "}},
		{"a governed element under an ungoverned child", constructGovernedContent.name(), rootAny{Children: child(anyElement{Local: "c", Content: anyType{Children: child(q)}})},
			[]string{"element {urn:s}q at rootAny.Children[0].Element.Content.Children[0].Element, " + in}},
		{"a governed attribute on the element", constructGovernedContent.name(), rootAny{Attributes: []anyAttribute{{"urn:s", "qa", "s:x"}}},
			[]string{"attribute {urn:s}qa at rootAny.Attributes[0], " + in, ", governed by attribute {urn:s}qa declared at "}},
		{"a governed attribute on a descendant", constructGovernedContent.name(), rootAny{Children: child(anyElement{Local: "c", Content: anyType{Attributes: []anyAttribute{{"urn:s", "qa", "s:x"}}}})},
			[]string{"attribute {urn:s}qa at rootAny.Children[0].Element.Content.Attributes[0], " + in}},
		{"an XSI attribute ahead of a governed element", constructXsiAttribute.name(), rootAny{Attributes: []anyAttribute{{xsd.XMLSchemaInstanceNS, "nil", "true"}}, Children: child(q)},
			[]string{"attribute xsi:nil at rootAny.Attributes[0], " + in}},
	} {
		t.Run(row.name, func(t *testing.T) {
			_, err := appendChecked(t, nil, governed, strict.New(), rootName, row.v)
			_ = asDecline(t, err, "codec does not encode "+row.construct+": "+row.want[0])
			holds(t, err, row.want[1:]...)
		})
	}
	t.Run("a governed child of a particle's element", func(t *testing.T) {
		schema := schemaOf(t, oneAnySchema+`<xs:element name="q" type="xs:QName"/>`)
		_, err := appendChecked(t, nil, schema, strict.New(), rootName, oneAny{A: anyType{Children: child(q)}})
		_ = asDecline(t, err, "codec does not encode "+constructGovernedContent.name()+": element {urn:s}q at oneAny.A.Children[0].Element, in the content of element {urn:s}a declared at ")
	})
	t.Run("a skip wildcard's match holding a governed name", func(t *testing.T) {
		v := seqWild{A: "x", Any: anyElement{Space: "urn:o", Local: "w", Content: anyType{Children: child(anyElement{Space: "urn:s", Local: "r", Content: anyType{Text: "s:x"}})}}}
		roundTrip(t, schemaOf(t, seqSkipSchema), rootName, v, `<r xmlns="urn:s"><a>x</a><w xmlns="urn:o"><r xmlns="urn:s">s:x</r></w></r>`, deepEqual[seqWild])
	})
	t.Run("a type derived from xs:anyType", func(t *testing.T) {
		roundTrip(t, schemaOf(t, derivedAnySchema), rootName, derivedAny{A: struct{ B string }{B: "x"}}, `<r xmlns="urn:s"><a><b>x</b></a></r>`, deepEqual[derivedAny])
	})
}

// encodeSimpleContentSchema declares, in urn:s, beside simpleContentTypes:
// note, an extension of xs:string with the attribute lang; qv, an extension of
// xs:QName with the attribute k; mr, a restriction of the mixed emptiable mx
// by a child <simpleType> restricting xs:int (Structures §3.4.2.2
// dcl.ctd.ctsc case 2); a document element of each of note, rich, capped,
// tagged, nums, qv and mr; and ps, a sequence of particles of note, rich and
// capped, once, optional and repeated.
const encodeSimpleContentSchema = simpleContentTypes + `<xs:complexType name="note"><xs:simpleContent><xs:extension base="xs:string"><xs:attribute name="lang" type="xs:string"/></xs:extension></xs:simpleContent></xs:complexType>
<xs:complexType name="qv"><xs:simpleContent><xs:extension base="xs:QName"><xs:attribute name="k" type="xs:string"/></xs:extension></xs:simpleContent></xs:complexType>
<xs:complexType name="mx" mixed="true"><xs:sequence><xs:element name="e" type="xs:string" minOccurs="0"/></xs:sequence></xs:complexType>
<xs:complexType name="mr"><xs:simpleContent><xs:restriction base="s:mx"><xs:simpleType><xs:restriction base="xs:int"/></xs:simpleType></xs:restriction></xs:simpleContent></xs:complexType>
<xs:element name="note" type="s:note"/>
<xs:element name="rich" type="s:rich"/>
<xs:element name="capped" type="s:capped"/>
<xs:element name="tagged" type="s:tagged"/>
<xs:element name="nums" type="s:nums"/>
<xs:element name="qv" type="s:qv"/>
<xs:element name="mr" type="s:mr"/>
<xs:element name="ps"><xs:complexType><xs:sequence>
<xs:element name="n" type="s:note"/><xs:element name="no" type="s:note" minOccurs="0"/><xs:element name="nr" type="s:note" maxOccurs="unbounded"/>
<xs:element name="x" type="s:rich"/><xs:element name="xo" type="s:rich" minOccurs="0"/><xs:element name="xr" type="s:rich" maxOccurs="unbounded"/>
<xs:element name="c" type="s:capped"/><xs:element name="co" type="s:capped" minOccurs="0"/><xs:element name="cr" type="s:capped" maxOccurs="unbounded"/>
</xs:sequence></xs:complexType></xs:element>`

// scNote, scQName, scMixedRestriction and scParticles are the types codegen
// emits for encodeSimpleContentSchema's note, qv, mr and ps, which
// TestEncodeSimpleContentTypesAreCodegens pins; rich and capped are
// simplecontent_test.go's scRich and scPrice.
type (
	scNote struct {
		Value string
		Lang  *string
	}
	scQName struct {
		Value tQName
		K     *string
	}
	scMixedRestriction struct{ Value int32 }
	scParticles        struct {
		N  scNote
		No *scNote
		Nr []scNote
		X  scRich
		Xo *scRich
		Xr []scRich
		C  scPrice
		Co *scPrice
		Cr []scPrice
	}
)

// TestEncodeSimpleContentTypesAreCodegens pins the types above to those
// codegen emits for encodeSimpleContentSchema, in shape (codegen/doc.go "#
// Complex types").
func TestEncodeSimpleContentTypesAreCodegens(t *testing.T) {
	gen := generated(t, schemaOf(t, encodeSimpleContentSchema))
	for _, decl := range []string{
		"type Note struct {\n\tValue string\n\tLang  *string\n}\n",
		"type Qv struct {\n\tValue QName\n\tK     *string\n}\n",
		"type Mr struct {\n\tValue int32\n}\n",
		"type Ps struct {\n\tN  Note2\n\tNo *Note2\n\tNr []Note2\n\tX  Rich2\n\tXo *Rich2\n\tXr []Rich2\n\tC  Capped2\n\tCo *Capped2\n\tCr []Capped2\n}\n",
	} {
		if !bytes.Contains(gen, []byte(decl)) {
			t.Errorf("codegen emits\n%s\nwhich does not declare\n%s", gen, decl)
		}
	}
}

// TestAppendSimpleContent pins the simple-content bullet of "# What encodes"
// (Structures §3.4.4.2 cvc-complex-type clause 1.2, §3.4.2.2 dcl.ctd.ctsc):
// an element of a complex type with simple content is written with its
// attributes, then its value field's text as its character content, each
// document accepted by validate and decoded back equal: an
// extension of xs:string with an attribute, an extension of a complex type
// with simple content, and a restriction adding a facet, each as the document
// element and as a particle once, optional and repeated; an extension of a
// union and of a list; an empty xs:string value, written <x/>, and a CR,
// written &#xD;; an xs:QName value in another namespace, its prefix declared
// on the start tag after the attributes; and a restriction of a mixed
// emptiable base by a child <simpleType> (case 2).
func TestAppendSimpleContent(t *testing.T) {
	schema := schemaOf(t, encodeSimpleContentSchema)
	str := func(s string) *string { return &s }
	i32 := func(i int32) *int32 { return &i }
	t.Run("an extension of xs:string with an attribute", func(t *testing.T) {
		roundTrip(t, schema, sname("note"), scNote{Value: "a<b", Lang: str("en")}, `<note xmlns="urn:s" lang="en">a&lt;b</note>`, deepEqual[scNote])
	})
	t.Run("an empty value", func(t *testing.T) {
		roundTrip(t, schema, sname("note"), scNote{}, `<note xmlns="urn:s"/>`, deepEqual[scNote])
	})
	t.Run("white space", func(t *testing.T) {
		roundTrip(t, schema, sname("note"), scNote{Value: " a\r\n\tb "}, "<note xmlns=\"urn:s\"> a&#xD;\n\tb </note>", deepEqual[scNote])
	})
	t.Run("an extension of a complex type with simple content", func(t *testing.T) {
		roundTrip(t, schema, sname("rich"), scRich{Value: testRat(t, "12.5"), K: i32(3), Cur: str("EUR")}, `<rich xmlns="urn:s" k="3" cur="EUR">12.5</rich>`, deepEqual[scRich])
	})
	t.Run("a restriction adding a facet", func(t *testing.T) {
		roundTrip(t, schema, sname("capped"), scPrice{Value: testRat(t, "100")}, `<capped xmlns="urn:s">100</capped>`, deepEqual[scPrice])
	})
	t.Run("an extension of a union", func(t *testing.T) {
		roundTrip(t, schema, sname("tagged"), scTagged{Value: unionUInt(5), Value2: str("v")}, `<tagged xmlns="urn:s" value="v">5</tagged>`, deepEqual[scTagged], WithBranches[unionU](unionUInt(0), unionUDate{}))
	})
	t.Run("an extension of a list", func(t *testing.T) {
		roundTrip(t, schema, sname("nums"), scNums{Value: []int32{1, 2}, N: i32(2)}, `<nums xmlns="urn:s" n="2">1 2</nums>`, deepEqual[scNums])
	})
	t.Run("an xs:QName in another namespace", func(t *testing.T) {
		roundTrip(t, schema, sname("qv"), scQName{Value: tQName{Space: "urn:o", Local: "x"}, K: str("EUR")}, `<qv xmlns="urn:s" k="EUR" xmlns:p1="urn:o">p1:x</qv>`, deepEqual[scQName])
	})
	t.Run("a restriction of a mixed emptiable base", func(t *testing.T) {
		roundTrip(t, schema, sname("mr"), scMixedRestriction{Value: -7}, `<mr xmlns="urn:s">-7</mr>`, deepEqual[scMixedRestriction])
	})
	t.Run("particles once, optional and repeated", func(t *testing.T) {
		v := scParticles{
			N: scNote{Value: "a", Lang: str("en")}, No: &scNote{Value: "b"}, Nr: []scNote{{Value: "c"}, {Value: "d", Lang: str("fr")}},
			X: scRich{Value: testRat(t, "1"), Cur: str("x")}, Xo: &scRich{Value: testRat(t, "2"), K: i32(4)}, Xr: []scRich{{Value: testRat(t, "3")}},
			C: scPrice{Value: testRat(t, "5")}, Co: &scPrice{Value: testRat(t, "6"), Cur: str("y")}, Cr: []scPrice{{Value: testRat(t, "7")}, {Value: testRat(t, "8")}},
		}
		roundTrip(t, schema, sname("ps"), v, `<ps xmlns="urn:s"><n lang="en">a</n><no>b</no><nr>c</nr><nr lang="fr">d</nr>`+
			`<x cur="x">1</x><xo k="4">2</xo><xr>3</xr><c>5</c><co cur="y">6</co><cr>7</cr><cr>8</cr></ps>`, deepEqual[scParticles])
	})
	t.Run("particles, the optional absent", func(t *testing.T) {
		v := scParticles{N: scNote{Value: "a"}, Nr: []scNote{{}}, X: scRich{Value: testRat(t, "1")}, Xr: []scRich{{Value: testRat(t, "3")}}, C: scPrice{Value: testRat(t, "5")}, Cr: []scPrice{{Value: testRat(t, "7")}}}
		roundTrip(t, schema, sname("ps"), v, `<ps xmlns="urn:s"><n>a</n><nr/><x>1</x><xr>3</xr><c>5</c><cr>7</cr></ps>`, deepEqual[scParticles])
	})
}

// TestAppendSimpleContentInvalid pins item 6 of Append's doc for simple
// content, each error naming the value field's Go path and the {content
// type}'s {simple type definition} after its complex type: a value a
// restriction's facet rejects, as the document element and as a particle; the
// nil interface of a union value field, which never means absent; and a value
// invalid in an attribute and in the value field, the attribute's error
// reported, an element's attributes before its content.
func TestAppendSimpleContentInvalid(t *testing.T) {
	schema := schemaOf(t, encodeSimpleContentSchema)
	bad := "\x00"
	t.Run("a facet of a restriction", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("capped"), scPrice{Value: testRat(t, "101")})
		invalidValue(t, err)
		const want = "codec: Append: scPrice.Value holds 101, which the {type definition} of element {urn:s}capped declared at s.xsd:16:1, a type defined at s.xsd:6:1, through its {content type}'s anonymous {simple type definition}, a type defined at s.xsd:6:49 rejects: "
		if !strings.HasPrefix(err.Error(), want) {
			t.Errorf("error %q, want the prefix %q", err, want)
		}
		if rule, _ := xsderr.RuleOf(err); rule != "cvc-maxInclusive-valid" {
			t.Errorf("rule %q, want cvc-maxInclusive-valid", rule)
		}
	})
	t.Run("a facet of a restriction, a particle", func(t *testing.T) {
		v := scParticles{N: scNote{Value: "a"}, Nr: []scNote{{}}, X: scRich{Value: testRat(t, "1")}, Xr: []scRich{{Value: testRat(t, "3")}}, C: scPrice{Value: testRat(t, "5")}, Cr: []scPrice{{Value: testRat(t, "7")}, {Value: testRat(t, "200")}}}
		_, err := appendChecked(t, nil, schema, strict.New(), sname("ps"), v)
		invalidValue(t, err)
		const want = "codec: Append: scParticles.Cr[1].Value holds 200, which the {type definition} of element {urn:s}cr declared at s.xsd:24:92, a type defined at s.xsd:6:1, through its {content type}'s anonymous {simple type definition}, a type defined at s.xsd:6:49 rejects: "
		if !strings.HasPrefix(err.Error(), want) {
			t.Errorf("error %q, want the prefix %q", err, want)
		}
	})
	t.Run("a nil union value field", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("tagged"), scTagged{}, WithBranches[unionU](unionUInt(0), unionUDate{}))
		invalidValue(t, err)
		const want = "codec: Append: scTagged.Value is nil, of the {type definition} of element {urn:s}tagged declared at "
		if !strings.HasPrefix(err.Error(), want) {
			t.Errorf("error %q, want the prefix %q", err, want)
		}
		holds(t, err, "through its {content type}'s {simple type definition} {urn:s}u")
	})
	t.Run("attributes before content", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("capped"), scPrice{Value: testRat(t, "101"), Cur: &bad})
		invalidValue(t, err)
		const want = "codec: Append: scPrice.Cur "
		if !strings.HasPrefix(err.Error(), want) {
			t.Errorf("error %q, want the prefix %q", err, want)
		}
	})
}

// TestAppendSimpleContentDeclineText pins the message of a decline of simple
// content's {simple type definition}, named after the complex type it is the
// {content type} of, as TestDeclineErrorTextSimpleContent pins CheckDecode's:
// for a named one, xs:ENTITY an extension's base, and for the anonymous one a
// restriction carrying an assertions facet gives.
func TestAppendSimpleContentDeclineText(t *testing.T) {
	named := schemaOf(t, root(`<xs:simpleContent><xs:extension base="xs:ENTITY"><xs:attribute name="k" type="xs:string"/></xs:extension></xs:simpleContent>`))
	const wantNamed = "codec does not encode a simple type derived from xs:ENTITY: the {type definition} of element {urn:s}r declared at s.xsd:1:181, a type defined at s.xsd:1:202, through its {content type}'s {simple type definition} xs:ENTITY; an encoder refusal, not a schema or value fault"
	if err := CheckAppend[struct{}](named, rootName); err == nil || err.Error() != wantNamed {
		t.Errorf("CheckAppend = %v, want %q", err, wantNamed)
	}
	anon := schemaOf(t, `<xs:complexType name="b"><xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent></xs:complexType>`+
		root(`<xs:simpleContent><xs:restriction base="s:b"><xs:assertion test="true()"/></xs:restriction></xs:simpleContent>`))
	const wantAnon = "codec does not encode a simple type with an assertions facet: the {type definition} of element {urn:s}r declared at s.xsd:1:292, a type defined at s.xsd:1:313, through its {content type}'s anonymous {simple type definition}, a type defined at s.xsd:1:347; an encoder refusal, not a schema or value fault"
	err := CheckAppend[struct{}](anon, rootName)
	if err == nil || err.Error() != wantAnon {
		t.Errorf("CheckAppend = %v, want %q", err, wantAnon)
	}
	var de *DeclineError
	if !errors.As(err, &de) || de.Construct() != (Construct{constructAssertions}) {
		t.Errorf("Construct = %v, want %s", err, constructAssertions.name())
	}
}

// The encode flatten schemas, each declaring s:r, whose content {particle}
// codegen flattens, beside flatten_test.go's: efPairSchema's sequence(a?, b?)
// occurring exactly twice, read into flatSeq; efOnceSchema's choice(a,
// sequence(b?)) occurring exactly once, flattened by its emptiable
// model-group alternative, read into flatOpt; efSkipSchema's sequence(a+,
// any?) repeated, its wildcard ##other of {process contents} skip, read into
// flatWild; efRefSchema's sequence(x?, b?) repeated, x an <element ref> to a
// top-level declaration that heads no substitution group; and efKeySchema's
// sequence(g?, b?) repeated, g a local declaration with a unique gu over its
// k children.
const (
	efPairSchema = `<xs:element name="r"><xs:complexType><xs:sequence minOccurs="2" maxOccurs="2"><xs:element name="a" type="xs:string" minOccurs="0"/><xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>`
	efOnceSchema = `<xs:element name="r"><xs:complexType><xs:choice><xs:element name="a" type="xs:string"/><xs:sequence><xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence></xs:choice></xs:complexType></xs:element>`
	efSkipSchema = `<xs:element name="r"><xs:complexType><xs:sequence maxOccurs="unbounded"><xs:element name="a" type="xs:string" maxOccurs="unbounded"/><xs:any namespace="##other" minOccurs="0" processContents="skip"/></xs:sequence></xs:complexType></xs:element>`
	efRefSchema  = `<xs:element name="x" type="xs:string"/><xs:element name="r"><xs:complexType><xs:sequence maxOccurs="unbounded"><xs:element ref="s:x" minOccurs="0"/><xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>`
	efKeySchema  = `<xs:element name="r"><xs:complexType><xs:sequence maxOccurs="unbounded"><xs:element name="g" minOccurs="0"><xs:complexType><xs:sequence><xs:element name="k" type="xs:string" maxOccurs="2"/></xs:sequence></xs:complexType>` +
		`<xs:unique name="gu"><xs:selector xpath="s:k"/><xs:field xpath="."/></xs:unique></xs:element><xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>`
)

// The types codegen emits for efRefSchema's and efKeySchema's s:r, in shape,
// which flatten_test.go's TestFlattenTypesAreCodegens pins, as it pins
// flatSeq's, flatOpt's and flatWild's for efPairSchema, efOnceSchema and
// efSkipSchema: efRef, efRefX, efRefB and efX, the named type of top-level
// x; efKey, efKeyG, efKeyB and efKeyGroup, g's struct.
type (
	efRef     struct{ Sequence []efRefItem }
	efRefItem interface{ isEfRefItem() }
	efRefX    struct{ X efX }
	efRefB    struct{ B string }
	efX       string

	efKey      struct{ Sequence []efKeyItem }
	efKeyItem  interface{ isEfKeyItem() }
	efKeyG     struct{ G efKeyGroup }
	efKeyB     struct{ B string }
	efKeyGroup struct{ K []string }
)

func (efRefX) isEfRefItem() {}
func (efRefB) isEfRefItem() {}
func (efKeyG) isEfKeyItem() {}
func (efKeyB) isEfKeyItem() {}

// The branch sets of efRefSchema's and efKeySchema's interfaces.
var (
	efRefBranches = WithBranches[efRefItem](efRefX{}, efRefB{})
	efKeyBranches = WithBranches[efKeyItem](efKeyG{}, efKeyB{})
)

// TestAppendFlattened pins "# What encodes"' flattened content {particle}, in
// each shape of reach its rows name: each item of the slice is written, in
// slice order, as the element of its branch's basic particle (Structures
// §3.8.4.2 key-vpath), validate accepts the document, and Decode under the
// same sets returns the value. The rows of efPairSchema hold three items and
// one under a sequence occurring exactly twice, since the slice holds one
// item per child, not per occurrence (§3.9.4.3 cvc-accept clause 3.1), and
// efOnceSchema's empty slice is the empty match. Each row fails with
// contentEncodes' Flattened case declining as it did before.
func TestAppendFlattened(t *testing.T) {
	t.Run("a repeated sequence of element particles", func(t *testing.T) {
		v := flatSeq{Sequence: []flatSeqItem{flatSeqA{A: "1"}, flatSeqB{B: "2"}, flatSeqB{B: "3"}, flatSeqA{A: "4"}}}
		roundTrip(t, schemaOf(t, flatSeqSchema), rootName, v, `<r xmlns="urn:s"><a>1</a><b>2</b><b>3</b><a>4</a></r>`, deepEqual[flatSeq], flatSeqBranches)
	})
	t.Run("a sequence occurring twice, three items", func(t *testing.T) {
		v := flatSeq{Sequence: []flatSeqItem{flatSeqA{A: "1"}, flatSeqA{A: "2"}, flatSeqB{B: "3"}}}
		roundTrip(t, schemaOf(t, efPairSchema), rootName, v, `<r xmlns="urn:s"><a>1</a><a>2</a><b>3</b></r>`, deepEqual[flatSeq], flatSeqBranches)
	})
	t.Run("a sequence occurring twice, one item", func(t *testing.T) {
		v := flatSeq{Sequence: []flatSeqItem{flatSeqB{B: "1"}}}
		roundTrip(t, schemaOf(t, efPairSchema), rootName, v, `<r xmlns="urn:s"><b>1</b></r>`, deepEqual[flatSeq], flatSeqBranches)
	})
	t.Run("a <group ref> occurring twice", func(t *testing.T) {
		v := flatSeq{Sequence: []flatSeqItem{flatSeqA{A: "1"}, flatSeqB{B: "2"}, flatSeqA{A: "3"}}}
		roundTrip(t, schemaOf(t, flatRefSchema), rootName, v, `<r xmlns="urn:s"><a>1</a><b>2</b><a>3</a></r>`, deepEqual[flatSeq], flatSeqBranches)
	})
	t.Run("a repeated choice with a repeating edge, beside attributes", func(t *testing.T) {
		c, q := "c", "q"
		v := flatChoice{Choice: []flatChoiceItem{flatChoiceA{A: "x"}, flatChoiceA{A: "y"}, flatChoiceB{B: 1}, flatChoiceA{A: "z"}, flatChoiceB{B: 2}}, Sequence: &q, Choice2: &c}
		roundTrip(t, schemaOf(t, flatChoiceSchema), rootName, v, `<r xmlns="urn:s" sequence="q" choice="c"><a>x</a><a>y</a><b>1</b><a>z</a><b>2</b></r>`, deepEqual[flatChoice], flatChoiceBranches)
	})
	t.Run("a choice occurring once, its element alternative", func(t *testing.T) {
		v := flatOpt{Choice: []flatOptItem{flatOptA{A: "v"}}}
		roundTrip(t, schemaOf(t, efOnceSchema), rootName, v, `<r xmlns="urn:s"><a>v</a></r>`, deepEqual[flatOpt], flatOptBranches)
	})
	t.Run("a choice occurring once, its emptiable sequence's element", func(t *testing.T) {
		v := flatOpt{Choice: []flatOptItem{flatOptB{B: "v"}}}
		roundTrip(t, schemaOf(t, efOnceSchema), rootName, v, `<r xmlns="urn:s"><b>v</b></r>`, deepEqual[flatOpt], flatOptBranches)
	})
	t.Run("a choice occurring once, the empty match", func(t *testing.T) {
		roundTrip(t, schemaOf(t, efOnceSchema), rootName, flatOpt{}, `<r xmlns="urn:s"/>`, deepEqual[flatOpt], flatOptBranches)
	})
	t.Run("an optional choice with an emptiable model-group alternative", func(t *testing.T) {
		roundTrip(t, schemaOf(t, flatOptSchema), rootName, flatOpt{Choice: []flatOptItem{flatOptB{B: "v"}}}, `<r xmlns="urn:s"><b>v</b></r>`, deepEqual[flatOpt], flatOptBranches)
	})
	t.Run("a skip wildcard in the reach", func(t *testing.T) {
		v := flatWild{Sequence: []flatWildItem{flatWildA{A: "1"}, flatWildA{A: "2"}, flatWildAny{Any: anyElement{Space: "urn:o", Local: "w"}}, flatWildA{A: "3"}}}
		roundTrip(t, schemaOf(t, efSkipSchema), rootName, v, `<r xmlns="urn:s"><a>1</a><a>2</a><w xmlns="urn:o"/><a>3</a></r>`, deepEqual[flatWild], flatWildBranches)
	})
	t.Run("an <element ref> alternative", func(t *testing.T) {
		v := efRef{Sequence: []efRefItem{efRefX{X: "1"}, efRefB{B: "2"}, efRefX{X: "3"}}}
		roundTrip(t, schemaOf(t, efRefSchema), rootName, v, `<r xmlns="urn:s"><x>1</x><b>2</b><x>3</x></r>`, deepEqual[efRef], efRefBranches)
	})
	t.Run("two branches of one name, a tie", func(t *testing.T) {
		v := flatTie{Sequence: []flatTieItem{flatTieR1{R1: "1"}, FlatTieItemR2{R2: "2"}, FlatTieItemR22{R2: "3"}, flatTieR3{R3: "4"}}}
		roundTrip(t, schemaOf(t, flatTieSchema), rootName, v, `<r xmlns="urn:s"><r1>1</r1><r2>2</r2><r2>3</r2><r3>4</r3></r>`, deepEqual[flatTie], flatTieBranches)
	})
	t.Run("a declaration with an identity constraint in the reach", func(t *testing.T) {
		v := efKey{Sequence: []efKeyItem{efKeyG{G: efKeyGroup{K: []string{"1", "2"}}}, efKeyB{B: "b"}, efKeyG{G: efKeyGroup{K: []string{"1"}}}}}
		roundTrip(t, schemaOf(t, efKeySchema), rootName, v, `<r xmlns="urn:s"><g><k>1</k><k>2</k></g><b>b</b><g><k>1</k></g></r>`, deepEqual[efKey], efKeyBranches)
	})
}

// TestAppendFlattenedInvalid pins item 6 of Append's doc for a flattened
// content {particle}'s slice, each error with no xsderr rule and opening with
// the Go path of the item it names: a nil item, which Decode never returns;
// an item of a dynamic type the WithBranches set placed at no alternative, a
// pointer to a branch among them; an item the content model admits at no
// particle where it is written, a second item of a choice occurring once and
// a name outside the skip wildcard's {namespace constraint} among them
// (§3.9.4.3 cvc-accept, §3.10.4.1 cvc-wildcard); an item the content model
// attributes to a particle other than its branch's, the tie's g1 r2 where
// g2's branch holds it (§3.4.4.4 key-att-to); and a slice after whose last
// item the content model is not accepting (§3.8.4.3 cvc-model-group). The
// reached declaration's identity constraint is assessed, so a duplicate
// value in one g is item 8's error instead.
func TestAppendFlattenedInvalid(t *testing.T) {
	seqS := schemaOf(t, flatSeqSchema)
	t.Run("a nil item", func(t *testing.T) {
		_, err := appendChecked(t, nil, seqS, strict.New(), rootName, flatSeq{Sequence: []flatSeqItem{flatSeqA{A: "1"}, nil}}, flatSeqBranches)
		invalidChoice(t, err, "codec: Append: flatSeq.Sequence[1] is nil, an item of the flattened content of the {type definition} of element {urn:s}r declared at ", "so that Decode returns no such item")
	})
	t.Run("a pointer to a branch", func(t *testing.T) {
		_, err := appendChecked(t, nil, seqS, strict.New(), rootName, flatSeq{Sequence: []flatSeqItem{&flatSeqA{A: "1"}}}, flatSeqBranches)
		invalidChoice(t, err, "codec: Append: flatSeq.Sequence[0] holds a *codec.flatSeqA, which the WithBranches set of codec.flatSeqItem places at no alternative of the flattened content of the {type definition} of element {urn:s}r declared at ", "so that Decode returns no such value")
	})
	t.Run("a second item of a choice occurring once", func(t *testing.T) {
		_, err := appendChecked(t, nil, schemaOf(t, efOnceSchema), strict.New(), rootName, flatOpt{Choice: []flatOptItem{flatOptA{A: "1"}, flatOptA{A: "2"}}}, flatOptBranches)
		invalidChoice(t, err, "codec: Append: flatOpt.Choice[1].(flatOptA).A holds element {urn:s}a, which no particle of the content model of the {type definition} of element {urn:s}r declared at ", "admits where it is written")
	})
	t.Run("a name outside the skip wildcard's {namespace constraint}", func(t *testing.T) {
		v := flatWild{Sequence: []flatWildItem{flatWildA{A: "1"}, flatWildAny{Any: anyElement{Space: "urn:s", Local: "w"}}}}
		_, err := appendChecked(t, nil, schemaOf(t, efSkipSchema), strict.New(), rootName, v, flatWildBranches)
		invalidChoice(t, err, "codec: Append: flatWild.Sequence[1].(flatWildAny).Any holds element {urn:s}w, which no particle of the content model of ", "cvc-wildcard")
	})
	t.Run("an item attributed to another alternative's particle", func(t *testing.T) {
		v := flatTie{Sequence: []flatTieItem{flatTieR1{R1: "1"}, FlatTieItemR22{R2: "2"}}}
		_, err := appendChecked(t, nil, schemaOf(t, flatTieSchema), strict.New(), rootName, v, flatTieBranches)
		invalidChoice(t, err, "codec: Append: flatTie.Sequence[1].(FlatTieItemR22).R2 holds element {urn:s}r2, which the content model of the {type definition} of element {urn:s}r declared at ", "attributes to a particle other than the one whose field holds it")
	})
	t.Run("a slice the content model does not accept at its end", func(t *testing.T) {
		v := flatShared{Sequence: []flatSharedItem{flatSharedC{C: "1"}}}
		_, err := appendChecked(t, nil, schemaOf(t, flatSharedSchema), strict.New(), rootName, v, flatSharedBranches)
		invalidChoice(t, err, "codec: Append: flatShared holds content the content model of the {type definition} of element {urn:s}r declared at ", "it ends short of a particle it must satisfy")
	})
	t.Run("an identity constraint in the reach, assessed", func(t *testing.T) {
		v := efKey{Sequence: []efKeyItem{efKeyG{G: efKeyGroup{K: []string{"1", "1"}}}}}
		_, err := appendChecked(t, nil, schemaOf(t, efKeySchema), strict.New(), rootName, v, efKeyBranches)
		if !errors.Is(err, ErrInvalidValue) || !strings.Contains(err.Error(), "{urn:s}gu") {
			t.Errorf("Append = %v, want item 8's ErrInvalidValue naming the unique {urn:s}gu", err)
		}
	})
}

// TestAppendFlattenedDeclineSubject pins the subject of a decline item 4 of
// Append's doc charges inside a flattened content {particle}: the basic
// particle by its index among the slice's alternatives, in reach order, as
// "alternative 2 of the flattened content {particle} of" the type; and,
// beside it, a choice occurring exactly once that codegen does not flatten
// naming its alternative as "alternative 2 of the content {particle} of" the
// type, so that routing that hold through the flattened arm fails here.
func TestAppendFlattenedDeclineSubject(t *testing.T) {
	rows := []struct {
		name, decls, want string
	}{
		{"flattened", strings.Replace(flatWildSchema, ` processContents="lax"`, "", 1), "codec does not encode a wildcard particle with {process contents} strict: alternative 2 of the flattened content {particle} of the {type definition} of element {urn:s}r declared at "},
		{"once choice", `<xs:element name="r"><xs:complexType><xs:choice><xs:element name="a" type="xs:string"/><xs:any/></xs:choice></xs:complexType></xs:element>`,
			"codec does not encode a wildcard particle with {process contents} strict: alternative 2 of the content {particle} of the {type definition} of element {urn:s}r declared at "},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			err := CheckAppend[struct{}](schemaOf(t, row.decls), rootName)
			_ = asDecline(t, err, row.want)
		})
	}
}

// vcSchema declares the value constraints TestAppendValueConstraint writes
// under: ed, an xs:int element with a default, and ef, one with a fixed
// value; es and fs, xs:string elements with default and fixed "abc"; el, a
// list of xs:int with default "1 2"; dec, an xs:decimal element fixed "1.0";
// nan, an xs:double one fixed "NaN"; at, a complex type of an xs:string
// attribute a with a default and an xs:int attribute f with a fixed value;
// un, one of an attribute of the union u with a default; and ad and af,
// xs:anyType elements with a default and a fixed value.
const vcSchema = unionTypes + `<xs:element name="ed" type="xs:int" default="7"/>
<xs:element name="ef" type="xs:int" fixed="7"/>
<xs:element name="es" type="xs:string" default="abc"/>
<xs:element name="fs" type="xs:string" fixed="abc"/>
<xs:element name="el" default="1 2"><xs:simpleType><xs:list itemType="xs:int"/></xs:simpleType></xs:element>
<xs:element name="dec" type="xs:decimal" fixed="1.0"/>
<xs:element name="nan" type="xs:double" fixed="NaN"/>
<xs:element name="at"><xs:complexType><xs:attribute name="a" type="xs:string" default="x"/><xs:attribute name="f" type="xs:int" fixed="7"/></xs:complexType></xs:element>
<xs:element name="un"><xs:complexType><xs:attribute name="dflt" type="s:u" default="7"/></xs:complexType></xs:element>
<xs:element name="ad" default="x"/>
<xs:element name="af" fixed="x"/>`

// vcGlobalSchema declares, in no namespace, the top-level attribute g of
// xs:int fixed "7" and the element r whose complex type uses it by ref and
// carries no value constraint of its own, so that only g's declaration
// constrains it (Structures §3.2.4.1 cvc-attribute clause 4).
const vcGlobalSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:attribute name="g" type="xs:int" fixed="7"/>
<xs:element name="r"><xs:complexType><xs:attribute ref="g"/></xs:complexType></xs:element></xs:schema>`

// The Go types codegen emits for vcSchema's and vcGlobalSchema's
// declarations: a bare V for each attribute use with a value constraint
// (codegen's package doc, "# Attributes").
type (
	vcInt     int32
	vcString  string
	vcList    []int32
	vcDecimal big.Rat
	vcDouble  float64
	vcAttrs   struct {
		A string
		F int32
	}
	vcUnion  struct{ Dflt unionU }
	vcGlobal struct{ G int32 }
)

// appendRow returns the Append, through appendChecked on strict's backend,
// of v as root of schema under opts, for a row of TestAppendValueConstraint.
func appendRow[T any](schema *xsd.Schema, root xsd.QName, v T, opts ...Option) func(t *testing.T, dst []byte) ([]byte, error) {
	return func(t *testing.T, dst []byte) ([]byte, error) {
		t.Helper()
		return appendChecked(t, dst, schema, strict.New(), root, v, opts...)
	}
}

// TestAppendValueConstraint pins Append on a declaration with a value
// constraint, default or fixed, which it writes as it is held and never
// omits (items 6 and 7 of Append's doc): an element's value, a simple
// content's and an attribute's each written, validated and decoded equal,
// an equal value spelled other than the constraint, 1 for "1.0", and NaN
// for "NaN", among them; a value no valid document holds under a fixed one,
// at an element, a simple content, an attribute use and an attribute
// declaration alone, the empty text and the content of an xs:anyType
// element among them, item 6; the empty text under an element default whose
// value it is not, item 7; and a nil union under an attribute default, item
// 6, since Decode supplies the default there. Each error leaves dst
// unextended.
func TestAppendValueConstraint(t *testing.T) {
	schema := schemaOf(t, vcSchema)
	global := parseString(t, vcGlobalSchema)
	var one, four big.Rat
	one.SetInt64(1)
	four.SetInt64(4)
	schemaNS := "urn:schema"
	for _, row := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"element default, another value", func(t *testing.T) {
			roundTrip(t, schema, sname("ed"), vcInt(5), `<ed xmlns="urn:s">5</ed>`, deepEqual[vcInt])
		}},
		{"element default, its value", func(t *testing.T) {
			roundTrip(t, schema, sname("ed"), vcInt(7), `<ed xmlns="urn:s">7</ed>`, deepEqual[vcInt])
		}},
		{"element fixed", func(t *testing.T) {
			roundTrip(t, schema, sname("ef"), vcInt(7), `<ef xmlns="urn:s">7</ef>`, deepEqual[vcInt])
		}},
		{"string default", func(t *testing.T) {
			roundTrip(t, schema, sname("es"), vcString("abc"), `<es xmlns="urn:s">abc</es>`, deepEqual[vcString])
		}},
		{"string fixed", func(t *testing.T) {
			roundTrip(t, schema, sname("fs"), vcString("abc"), `<fs xmlns="urn:s">abc</fs>`, deepEqual[vcString])
		}},
		{"list default", func(t *testing.T) {
			roundTrip(t, schema, sname("el"), vcList{1, 2}, `<el xmlns="urn:s">1 2</el>`, sameLists[vcList])
		}},
		{"decimal fixed 1.0 holding 1", func(t *testing.T) {
			roundTrip(t, schema, sname("dec"), vcDecimal(one), `<dec xmlns="urn:s">1</dec>`, func(a, b vcDecimal) bool { return sameRat(big.Rat(a), big.Rat(b)) })
		}},
		{"double fixed NaN", func(t *testing.T) {
			roundTrip(t, schema, sname("nan"), vcDouble(math.NaN()), `<nan xmlns="urn:s">NaN</nan>`, sameDouble[vcDouble])
		}},
		{"simple content fixed", func(t *testing.T) {
			roundTrip(t, schemaOf(t, scFixedRoot), sname("r"), scPrice{Value: four}, `<r xmlns="urn:s">4</r>`, func(a, b scPrice) bool { return sameRat(a.Value, b.Value) && reflect.DeepEqual(a.Cur, b.Cur) })
		}},
		// f's fixed xs:QName p:fix, p bound to urn:schema in the schema
		// document, is spelled p1:fix under the prefix Append declares on f's
		// start tag, which the comparison resolves.
		{"QName fixed", func(t *testing.T) {
			roundTrip(t, schemaOf(t, qnameSchema), sname("r"), qnames{
				Q: tQName{Local: "q", Space: "urn:s"}, L: []tQName{}, N: tQName{Space: "urn:s", Local: "png"}, H: qnameHead{Space: "urn:s", Local: "h"},
				D: tQName{Space: schemaNS, Local: "d"}, F: tQName{Space: schemaNS, Local: "fix"}, Da: tQName{Space: schemaNS, Local: "att"},
			}, "", deepEqual[qnames])
		}},
		{"attribute default and fixed", func(t *testing.T) {
			roundTrip(t, schema, sname("at"), vcAttrs{A: "y", F: 7}, `<at xmlns="urn:s" a="y" f="7"/>`, deepEqual[vcAttrs])
		}},
		{"attribute default, its value", func(t *testing.T) {
			roundTrip(t, schema, sname("at"), vcAttrs{A: "x", F: 7}, `<at xmlns="urn:s" a="x" f="7"/>`, deepEqual[vcAttrs])
		}},
		{"attribute default, empty", func(t *testing.T) {
			roundTrip(t, schema, sname("at"), vcAttrs{A: "", F: 7}, `<at xmlns="urn:s" a="" f="7"/>`, deepEqual[vcAttrs])
		}},
		{"attribute declaration fixed", func(t *testing.T) {
			roundTrip(t, global, xsd.QName{Local: "r"}, vcGlobal{G: 7}, `<r g="7"/>`, deepEqual[vcGlobal])
		}},
		{"union attribute default", func(t *testing.T) {
			roundTrip(t, schema, sname("un"), vcUnion{Dflt: unionUInt(3)}, `<un xmlns="urn:s" dflt="3"/>`, deepEqual[vcUnion], unionVariants...)
		}},
		{"anyType default text", func(t *testing.T) {
			roundTrip(t, schema, sname("ad"), anyType{Text: "y"}, `<ad xmlns="urn:s">y</ad>`, deepEqual[anyType])
		}},
		{"anyType default child", func(t *testing.T) {
			roundTrip(t, schema, sname("ad"), anyType{Children: []anyChild{{Element: anyElement{Local: "c"}}}}, `<ad xmlns="urn:s"><c xmlns=""/></ad>`, deepEqual[anyType])
		}},
		{"anyType fixed text", func(t *testing.T) {
			roundTrip(t, schema, sname("af"), anyType{Text: "x"}, `<af xmlns="urn:s">x</af>`, deepEqual[anyType])
		}},
		{"anyType fixed empty", func(t *testing.T) {
			roundTrip(t, schema, sname("af"), anyType{}, `<af xmlns="urn:s"/>`, deepEqual[anyType])
		}},
		{"anyType fixed attributes", func(t *testing.T) {
			roundTrip(t, schema, sname("af"), anyType{Attributes: []anyAttribute{{Local: "k", Value: "v"}}, Text: "x"}, `<af xmlns="urn:s" k="v">x</af>`, deepEqual[anyType])
		}},
	} {
		t.Run(row.name, row.run)
	}
	var five big.Rat
	five.SetInt64(5)
	for _, row := range []struct {
		name string
		app  func(t *testing.T, dst []byte) ([]byte, error)
		want []string
	}{
		{"fixed element", appendRow(schema, sname("ef"), vcInt(8)), []string{"codec: Append: vcInt, of simple type xs:int, is written ", `"8", whose value is neither equal nor identical to "7", the {value} of the fixed {value constraint} of element {urn:s}ef`, "cvc-elt clause 5.2.2.2.2"}},
		{"fixed element, empty", appendRow(schema, sname("fs"), vcString("")), []string{"codec: Append: vcString, of simple type xs:string, is written ", `"", whose value is neither equal nor identical to "abc"`, "cvc-elt clause 5.1, §3.3.5.4 sic-eltType"}},
		{"fixed simple content", appendRow(schemaOf(t, scFixedRoot), sname("r"), scPrice{Value: five}), []string{"codec: Append: scPrice.Value, of ", `is written "5", whose value is neither equal nor identical to "4", the {value} of the fixed {value constraint} of element {urn:s}r`, "cvc-elt clause 5.2.2.2.2"}},
		{"fixed attribute use", appendRow(schema, sname("at"), vcAttrs{A: "y", F: 8}), []string{"codec: Append: vcAttrs.F, of simple type xs:int, is written ", `"8", whose value is neither equal nor identical to "7", the {value} of the fixed {value constraint} of the attribute use of attribute f`, "§3.5.4 cvc-au"}},
		{"fixed attribute declaration", appendRow(global, xsd.QName{Local: "r"}, vcGlobal{G: 8}), []string{"codec: Append: vcGlobal.G, of simple type xs:int, is written ", `"8", whose value is neither equal nor identical to "7", the {value} of the fixed {value constraint} of attribute g`, "§3.2.4.1 cvc-attribute clause 4"}},
		{"nil union under a default", appendRow(schema, sname("un"), vcUnion{}, unionVariants...), []string{"codec: Append: vcUnion.Dflt is nil, of simple type {urn:s}u", "where a value is required"}},
		{"anyType fixed, another text", appendRow(schema, sname("af"), anyType{Text: "y"}), []string{`codec: Append: anyType.Text holds "y", which does not match "x", the {lexical form} of the fixed {value constraint} of element {urn:s}af`, "cvc-elt clause 5.2.2.2.1"}},
		{"anyType fixed child", appendRow(schema, sname("af"), anyType{Text: "x", Children: []anyChild{{Element: anyElement{Local: "c"}}}}), []string{`codec: Append: anyType.Children holds 1, though the fixed {value constraint} "x" of element {urn:s}af`, "admits no element child", "cvc-elt clause 5.2.2.1"}},
		{"anyType fixed child, Text no Char", appendRow(schema, sname("af"), anyType{Text: "\x00", Children: []anyChild{{Element: anyElement{Local: "c"}}}}), []string{`codec: Append: anyType.Text holds "\x00", whose byte offset 0 starts no XML 1.0 Char`}},
	} {
		t.Run(row.name, func(t *testing.T) {
			got, err := row.app(t, []byte("pre"))
			invalidValue(t, err, row.want...)
			if !strings.HasPrefix(err.Error(), row.want[0]) {
				t.Errorf("error %q does not open with %q", err, row.want[0])
			}
			noRule(t, err)
			if string(got) != "pre" {
				t.Errorf("Append wrote %q", got)
			}
		})
	}
	for _, row := range []struct {
		name string
		app  func(t *testing.T, dst []byte) ([]byte, error)
		want string
	}{
		{"empty string under a default", appendRow(schema, sname("es"), vcString("")), `simple type xs:string, at vcString holding the value written "": its value is neither equal nor identical to "abc", the {value} of the default {value constraint} of element {urn:s}es`},
		{"empty list under a default", appendRow(schema, sname("el"), vcList{}), `at vcList holding the value written "": its value is neither equal nor identical to "1 2", the {value} of the default {value constraint} of element {urn:s}el`},
	} {
		t.Run(row.name, func(t *testing.T) {
			got, err := row.app(t, []byte("pre"))
			d := asDecline(t, err, "codec does not encode "+constructValueConstraint.name()+": ")
			holds(t, err, row.want, "cvc-elt clause 5.1, §3.3.5.4 sic-eltType")
			if d.Construct() != (Construct{constructValueConstraint}) {
				t.Errorf("Construct %v, want constructValueConstraint's", d.Construct())
			}
			if string(got) != "pre" {
				t.Errorf("Append wrote %q", got)
			}
		})
	}
}

// The all schemas Append writes beside all_test.go's, each declaring s:r:
// allBASchema's all(b, a), its particles declared b then a;
// allSkipFirstSchema's all(any, a), a skip wildcard of ##any declared ahead of
// the element particle a it matches, and allOccSkipFirstSchema's the same all
// group with {min occurs} 0, held in an occurrence field; allEmptySchema's
// <xs:all/>; allGroupSchema's all(<element ref> to the head a, b), the head's
// field the sealed interface over its substitution group; allUnionSchema's
// all(b of the union u, a); allPDSchema's all(p of xs:precisionDecimal, a);
// and allIDSchema's all(e of xs:ID, ref of xs:IDREF).
const (
	allBASchema           = `<xs:element name="r"><xs:complexType><xs:all><xs:element name="b" type="xs:int"/><xs:element name="a" type="xs:string"/></xs:all></xs:complexType></xs:element>`
	allSkipFirstSchema    = `<xs:element name="r"><xs:complexType><xs:all><xs:any processContents="skip"/><xs:element name="a" type="xs:string"/></xs:all></xs:complexType></xs:element>`
	allOccSkipFirstSchema = `<xs:element name="r"><xs:complexType><xs:all minOccurs="0"><xs:any processContents="skip"/><xs:element name="a" type="xs:string"/></xs:all></xs:complexType></xs:element>`
	allSkipManySchema     = `<xs:element name="r"><xs:complexType><xs:all><xs:any processContents="skip" maxOccurs="2"/><xs:element name="a" type="xs:string" maxOccurs="2"/></xs:all></xs:complexType></xs:element>`
	allEmptySchema        = `<xs:element name="r"><xs:complexType><xs:all/></xs:complexType></xs:element>`
	allGroupSchema        = `<xs:element name="a" type="xs:string"/><xs:element name="m" type="xs:string" substitutionGroup="s:a"/>
		<xs:element name="r"><xs:complexType><xs:all><xs:element ref="s:a"/><xs:element name="b" type="xs:string"/></xs:all></xs:complexType></xs:element>`
	allUnionSchema = unionTypes + `<xs:element name="r"><xs:complexType><xs:all><xs:element name="b" type="s:u"/><xs:element name="a" type="xs:string"/></xs:all></xs:complexType></xs:element>`
	allPDSchema    = `<xs:element name="r"><xs:complexType><xs:all><xs:element name="p" type="xs:precisionDecimal"/><xs:element name="a" type="xs:string"/></xs:all></xs:complexType></xs:element>`
	allIDSchema    = `<xs:element name="r"><xs:complexType><xs:all><xs:element name="e" type="xs:ID"/><xs:element name="ref" type="xs:IDREF"/></xs:all></xs:complexType></xs:element>`
)

// The types codegen emits for the schemas above, which item 5 of Append's
// doc holds each to: allSkipFirst is allSkipFirstSchema's R and the item of
// allOccSkipFirstSchema's occurrence field.
type (
	allBA struct {
		B int32
		A string
	}
	allSkipFirst struct {
		Any anyElement
		A   string
	}
	allOccSkipFirst struct{ All *allSkipFirst }
	allSkipMany     struct {
		Any []anyElement
		A   []string
	}
	allGroup struct {
		A grpAGroup
		B string
	}
	allUnion struct {
		B unionU
		A string
	}
	allPD struct {
		P tPrecisionDecimal
		A string
	}
	allID struct{ E, Ref string }
)

// TestAppendAll pins the all group "# What encodes" admits, spliced and held
// in an occurrence field: each row is written as shown, validate accepts it
// and Decode returns it equal. Element particles are written in {particles}
// order, all(b, a)'s b first; a <group ref>, a repeated particle, an
// emptiable all holding nothing and an extension's merged all write so too,
// and <xs:all/>, whose explicit content is empty (§3.4.2.3.3 clause 2.1.2),
// is empty content, which Append wrote before the all group's lift; a
// wildcard particle's element follows the element particles' whatever its
// place in {particles}: all(a, any ##other), all(a, <group ref> to
// all(any)), and all(any skip, a), whose value Decode returns for
// <a>x</a><a>y</a>, Any holding the second <a>, is written <a>x</a><a>y</a>,
// spliced and in an occurrence field, and so is all(any skip, a), each of
// {max occurs} 2, whose value Decode returns for three <a> holds the first two
// in A: with boundSequence.write writing its members in member order
// (writeMembers) the two spliced skip rows fail and no other row of the
// package, and with boundSequenceItem.writeItem doing so the occurrence row
// alone. A value Decode returns for a permuted document re-encodes in
// {particles} order. A sealed-interface field (a substitution group's, a
// union's, xs:precisionDecimal's) and an xs:ID and xs:IDREF pair, which Append
// assesses (cvc-id), are written inside an all group as anywhere else.
func TestAppendAll(t *testing.T) {
	w := anyElement{Space: "urn:o", Local: "w"}
	second := anyElement{Space: "urn:s", Local: "a", Content: anyType{Text: "y"}}
	t.Run("all(a, b)", func(t *testing.T) {
		roundTrip(t, schemaOf(t, allABSchema), rootName, allAB{A: "x", B: 1}, `<r xmlns="urn:s"><a>x</a><b>1</b></r>`, deepEqual[allAB])
	})
	t.Run("all(b, a), declared b then a", func(t *testing.T) {
		roundTrip(t, schemaOf(t, allBASchema), rootName, allBA{B: 1, A: "x"}, `<r xmlns="urn:s"><b>1</b><a>x</a></r>`, deepEqual[allBA])
	})
	t.Run("a <group ref> to an all group", func(t *testing.T) {
		roundTrip(t, schemaOf(t, allRefSchema), rootName, allAB{A: "x", B: 4}, `<r xmlns="urn:s"><a>x</a><b>4</b></r>`, deepEqual[allAB])
	})
	t.Run("a repeated particle", func(t *testing.T) {
		roundTrip(t, schemaOf(t, allRepSchema), rootName, allRep{A: []string{"p", "q"}, B: 3}, `<r xmlns="urn:s"><a>p</a><a>q</a><b>3</b></r>`, deepEqual[allRep])
	})
	t.Run("an emptiable spliced all holding nothing", func(t *testing.T) {
		roundTrip(t, schemaOf(t, allSplicedSchema), rootName, allSpliced{}, `<r xmlns="urn:s"/>`, deepEqual[allSpliced])
	})
	t.Run("an extension merging two alls", func(t *testing.T) {
		roundTrip(t, schemaOf(t, allExtSchema), rootName, allAB{A: "x", B: 6}, `<r xmlns="urn:s"><a>x</a><b>6</b></r>`, deepEqual[allAB])
	})
	t.Run("<xs:all/>", func(t *testing.T) {
		roundTrip(t, schemaOf(t, allEmptySchema), rootName, struct{}{}, `<r xmlns="urn:s"/>`, deepEqual[struct{}])
	})
	t.Run("all(a, any ##other)", func(t *testing.T) {
		roundTrip(t, schemaOf(t, allWildSchema), rootName, allWild{A: "x", Any: w}, `<r xmlns="urn:s"><a>x</a><w xmlns="urn:o"/></r>`, deepEqual[allWild])
	})
	t.Run("a wildcard inside a nested all", func(t *testing.T) {
		roundTrip(t, schemaOf(t, allNestedWildSchema), rootName, allWild{A: "x", Any: w}, `<r xmlns="urn:s"><a>x</a><w xmlns="urn:o"/></r>`, deepEqual[allWild])
	})
	t.Run("all(any skip, a), the value Decode returns", func(t *testing.T) {
		schema := schemaOf(t, allSkipFirstSchema)
		v := decodedAs(t, schema, "<s:a>x</s:a><s:a>y</s:a>", allSkipFirst{Any: second, A: "x"})
		roundTrip(t, schema, rootName, v, `<r xmlns="urn:s"><a>x</a><a>y</a></r>`, deepEqual[allSkipFirst])
	})
	t.Run("all(any skip, a) in an occurrence field, the value Decode returns", func(t *testing.T) {
		schema := schemaOf(t, allOccSkipFirstSchema)
		v := decodedAs(t, schema, "<s:a>x</s:a><s:a>y</s:a>", allOccSkipFirst{All: &allSkipFirst{Any: second, A: "x"}})
		roundTrip(t, schema, rootName, v, `<r xmlns="urn:s"><a>x</a><a>y</a></r>`, deepEqual[allOccSkipFirst])
	})
	t.Run("all(any skip, a), each of {max occurs} 2, the value Decode returns", func(t *testing.T) {
		schema := schemaOf(t, allSkipManySchema)
		third := anyElement{Space: "urn:s", Local: "a", Content: anyType{Text: "3"}}
		v := decodedAs(t, schema, "<s:a>1</s:a><s:a>2</s:a><s:a>3</s:a>", allSkipMany{Any: []anyElement{third}, A: []string{"1", "2"}})
		roundTrip(t, schema, rootName, v, `<r xmlns="urn:s"><a>1</a><a>2</a><a>3</a></r>`, deepEqual[allSkipMany])
	})
	t.Run("a value decoded from a permuted document", func(t *testing.T) {
		schema := schemaOf(t, allABSchema)
		v := decodedAs(t, schema, "<s:b>1</s:b><s:a>x</s:a>", allAB{A: "x", B: 1})
		roundTrip(t, schema, rootName, v, `<r xmlns="urn:s"><a>x</a><b>1</b></r>`, deepEqual[allAB])
	})
	t.Run("an occurrence field holding one item", func(t *testing.T) {
		roundTrip(t, schemaOf(t, allOccABSchema), rootName, allOccAB{All: &allOccABItem{A: "x", B: 1}}, `<r xmlns="urn:s"><a>x</a><b>1</b></r>`, deepEqual[allOccAB])
	})
	t.Run("an occurrence field holding nothing", func(t *testing.T) {
		roundTrip(t, schemaOf(t, allOccASchema), rootName, allOccA{}, `<r xmlns="urn:s"/>`, deepEqual[allOccA])
	})
	t.Run("an occurrence field's repeated particle", func(t *testing.T) {
		roundTrip(t, schemaOf(t, allOccRepSchema), rootName, allOccRep{All: &allOccRepItem{A: []string{"p", "q"}, B: 3}}, `<r xmlns="urn:s"><a>p</a><a>q</a><b>3</b></r>`, deepEqual[allOccRep])
	})
	t.Run("an occurrence field's nested all", func(t *testing.T) {
		roundTrip(t, schemaOf(t, allOccNestedSchema), rootName, allOccAB{All: &allOccABItem{A: "x", B: 4}}, `<r xmlns="urn:s"><a>x</a><b>4</b></r>`, deepEqual[allOccAB])
	})
	t.Run("an occurrence field of an extension merging two alls", func(t *testing.T) {
		roundTrip(t, schemaOf(t, allOccExtSchema), rootName, allOccAB{All: &allOccABItem{A: "x", B: 6}}, `<r xmlns="urn:s"><a>x</a><b>6</b></r>`, deepEqual[allOccAB])
	})
	t.Run("a substitution group's interface", func(t *testing.T) {
		roundTrip(t, schemaOf(t, allGroupSchema), rootName, allGroup{A: grpAGroupM{M: "y"}, B: "b"}, `<r xmlns="urn:s"><m>y</m><b>b</b></r>`, deepEqual[allGroup], grpBranches)
	})
	t.Run("a union's interface", func(t *testing.T) {
		roundTrip(t, schemaOf(t, allUnionSchema), rootName, allUnion{B: unionUDate(unionDateOf(2000, 1, 2)), A: "x"}, `<r xmlns="urn:s"><b>2000-01-02</b><a>x</a></r>`, sameUnion[allUnion], unionVariants...)
	})
	t.Run("an xs:precisionDecimal", func(t *testing.T) {
		roundTrip(t, schemaOf(t, allPDSchema), rootName, allPD{P: tPrecisionDecimal{tPDNotANumber{}}, A: "x"}, `<r xmlns="urn:s"><p>NaN</p><a>x</a></r>`, func(a, b allPD) bool {
			return samePD(a.P, b.P) && a.A == b.A
		}, pdArms)
	})
	t.Run("an xs:ID and an xs:IDREF", func(t *testing.T) {
		roundTrip(t, schemaOf(t, allIDSchema), rootName, allID{E: "i", Ref: "i"}, `<r xmlns="urn:s"><e>i</e><ref>i</ref></r>`, deepEqual[allID])
	})
}

// decodedAs returns the value Decode returns for the document s:r of schema
// holding body, failing t unless validate accepts it and the value is want.
func decodedAs[T any](t *testing.T, schema *xsd.Schema, body string, want T) T {
	t.Helper()
	data := instanceOf("", body)
	mustAccept(t, schema, data)
	got, err := Decode[T](schema, strict.New(), data)
	if err != nil {
		t.Fatal(err)
	}
	v := unscoped(*got).(T)
	if !reflect.DeepEqual(v, want) {
		t.Fatalf("Decode = %#v, want %#v", v, want)
	}
	return v
}

// TestAppendAllInvalid pins item 6 of Append's doc for an all group's
// required particle whose field is unset, spliced and in an occurrence
// field's item: a slice of {min occurs} 1 holding no item and a nil
// substitution group interface. Each error opens as shown and Append returns
// dst unextended; with writeOccurring's length check removed the two slice
// rows fail, Append returning no error, and no other row of this test or
// TestAppendAll does. Then item 8 for cvc-id: an xs:IDREF naming no xs:ID.
func TestAppendAllInvalid(t *testing.T) {
	pre := []byte("pre")
	for _, row := range []struct {
		name string
		run  func(t *testing.T) ([]byte, error)
		want string
	}{
		{"a repeated particle holding no item", func(t *testing.T) ([]byte, error) {
			return appendChecked(t, pre, schemaOf(t, allRepSchema), strict.New(), rootName, allRep{B: 3})
		}, "codec: Append: allRep.A holds 0 items, outside the occurrence 1..unbounded of the particle of "},
		{"an occurrence field's repeated particle holding no item", func(t *testing.T) ([]byte, error) {
			return appendChecked(t, pre, schemaOf(t, allOccRepSchema), strict.New(), rootName, allOccRep{All: &allOccRepItem{B: 3}})
		}, "codec: Append: allOccRep.All.A holds 0 items, outside the occurrence 1..unbounded of the particle of "},
		{"a nil substitution group interface", func(t *testing.T) ([]byte, error) {
			return appendChecked(t, pre, schemaOf(t, allGroupSchema), strict.New(), rootName, allGroup{B: "b"}, grpBranches)
		}, "codec: Append: allGroup.A is nil"},
	} {
		t.Run(row.name, func(t *testing.T) {
			got, err := row.run(t)
			invalidValue(t, err)
			if !strings.HasPrefix(err.Error(), row.want) {
				t.Errorf("error %q does not open with %q", err, row.want)
			}
			if string(got) != "pre" {
				t.Errorf("Append returned %q, want dst unextended", got)
			}
		})
	}
	t.Run("an xs:IDREF naming no xs:ID", func(t *testing.T) {
		got, err := appendChecked(t, pre, schemaOf(t, allIDSchema), strict.New(), rootName, allID{E: "i", Ref: "z"})
		assessedCharge(t, got, err, "cvc-id", "1", `the id "z"`)
	})
}
