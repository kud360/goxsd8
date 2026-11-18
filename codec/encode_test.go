package codec

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/codec/testdata/readme/order"
	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
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

// roundTrip appends v as the document element name of schema, requires the
// bytes want when want is not empty, validate's acceptance of them, and
// Decode's T equal to v by eq, and returns the bytes.
func roundTrip[T any](t *testing.T, schema *xsd.Schema, name xsd.QName, v T, want string, eq func(a, b T) bool) []byte {
	t.Helper()
	data, err := appendChecked(t, nil, schema, strict.New(), name, v)
	if err != nil {
		t.Fatalf("Append: %v", err)
	}
	if want != "" && string(data) != want {
		t.Fatalf("Append wrote\n%s\nwant\n%s", data, want)
	}
	accepts(t, schema, data)
	got, err := Decode[T](schema, strict.New(), data)
	if err != nil {
		t.Fatalf("Decode(%s): %v", data, err)
	}
	if !eq(*got, v) {
		t.Fatalf("Decode(%s) = %#v, want %#v", data, *got, v)
	}
	return data
}

// appendChecked returns Append's answer for its arguments and fails t
// unless CheckAppend[T]'s answer for schema and name agrees with it: a
// non-nil CheckAppend error is Append's, of equal Error text and equal class
// (staticClass), and a nil one leaves Append nil, an item 6 or item 7 error
// or a fault of backend: nil, a plain error without item 2's or item 3's
// text, or item 7's decline.
func appendChecked[T any](t *testing.T, dst []byte, schema *xsd.Schema, backend value.Backend, name xsd.QName, v T) ([]byte, error) {
	t.Helper()
	out, err := Append(dst, schema, backend, name, v)
	checked := CheckAppend[T](schema, name)
	if checked == nil {
		class := staticClass(err)
		static := class == "plain" && (strings.HasPrefix(err.Error(), "codec: no top-level element declaration") || strings.Contains(err.Error(), "{abstract} true"))
		if static || err != nil && class != "plain" && class != "decline "+constructNoCanonical.name() {
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
// ErrTypeMismatch, "plain" for every other non-nil error, and "nil".
func staticClass(err error) string {
	var d *DeclineError
	switch {
	case err == nil:
		return "nil"
	case errors.As(err, &d):
		return "decline " + d.Construct().String()
	case errors.Is(err, ErrTypeMismatch):
		return "mismatch"
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
// is TestAppendReadmeOrder's.
func TestAppendNamespaces(t *testing.T) {
	type child struct{ A string }
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
// error of Append's doc: no *DeclineError, wrapping neither
// ErrTypeMismatch nor errors.ErrUnsupported, whose text holds each of
// want.
func plainError(t *testing.T, err error, want ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("no error returned, want a plain error")
	}
	var d *DeclineError
	if errors.As(err, &d) || errors.Is(err, errors.ErrUnsupported) || errors.Is(err, ErrTypeMismatch) {
		t.Fatalf("returned %v, want a plain error", err)
	}
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
	if want := ErrTypeMismatch.Error() + ": *codec.shapes is a pointer to codec.shapes, which matches; use codec.shapes in its place"; !errors.Is(err, ErrTypeMismatch) || err.Error() != want {
		t.Errorf("Append[*shapes] = %v, want ErrTypeMismatch %q", err, want)
	}
	_, err = appendChecked(t, nil, schema, strict.New(), rootName, struct{ One string }{})
	if !errors.Is(err, ErrTypeMismatch) {
		t.Errorf("Append[struct{One string}] = %v, want ErrTypeMismatch", err)
	}
}

// TestCheckAppend pins CheckAppend's answers by Append's items: a plain
// error for a nil schema; item 4's decline for an xs:integer declaration and
// T big.Int, which CheckDecode accepts; nil for README's order.xsd and
// order.Order, whose zero value Append refuses by item 6, and item 5's
// ErrTypeMismatch for *order.Order; item 2's and item 3's plain errors; and
// item 3 before item 4 on an abstract xs:boolean declaration. appendChecked
// holds each answer to Append's.
func TestCheckAppend(t *testing.T) {
	schema := schemaOf(t, `<xs:element name="i" type="xs:integer"/><xs:element name="abs" type="xs:string" abstract="true"/>`+
		`<xs:element name="absb" type="xs:boolean" abstract="true"/>`)
	plainError(t, CheckAppend[string](nil, sname("i")), "codec: CheckAppend: nil *xsd.Schema")

	if err := CheckDecode[big.Int](schema, sname("i")); err != nil {
		t.Fatalf("CheckDecode[big.Int] = %v, want nil", err)
	}
	err := CheckAppend[big.Int](schema, sname("i"))
	_ = asDecline(t, err, "codec does not encode an atomic simple type whose primitive is other than xs:string or xs:double: ")
	_, _ = appendChecked(t, nil, schema, strict.New(), sname("i"), big.Int{})

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

// TestAppendInvalidValues pins item 6 of Append's doc: a plain error naming
// the Go path, for each value no valid document holds, and for the first in
// document order, an element's attributes before its content, though an
// item 7 value comes earlier.
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
			plainError(t, err, row.want...)
		})
	}
	t.Run("a token attribute not collapsed", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("ta"), tokenAttr{T: "  a  b"})
		plainError(t, err, "tokenAttr.T", `form "a b"`)
	})
	t.Run("a repeated field outside its occurrence", func(t *testing.T) {
		readme := filepath.Join("..", "testdata", "readme")
		orderSchema, err := parser.Parse("order.xsd", parser.WithResolver(loader.Dir(readme)))
		if err != nil {
			t.Fatal(err)
		}
		_, err = appendChecked(t, nil, orderSchema, strict.New(), xsd.QName{Space: "urn:o", Local: "order"}, order.Order{})
		plainError(t, err, "Order.Item holds 0 items", "1..unbounded")
	})
	t.Run("an xs:double a bound rejects", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("np"), dblRoot(1))
		plainError(t, err, "dblRoot", "{urn:s}nonPositive")
		if rule, _ := xsderr.RuleOf(err); rule != "cvc-maxInclusive-valid" {
			t.Errorf("rule %q, want cvc-maxInclusive-valid", rule)
		}
	})
	for _, f := range []float64{math.NaN(), math.Inf(-1)} {
		t.Run("an xs:double "+doubleLexical(f)+" a pattern rejects", func(t *testing.T) {
			_, err := appendChecked(t, nil, schema, strict.New(), sname("p"), dblRoot(f))
			plainError(t, err, "dblRoot", "{urn:s}one")
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
		plainError(t, err, "strRoot", "{urn:s}onlyA")
		if rule, _ := xsderr.RuleOf(err); rule != "cvc-pattern-valid" {
			t.Errorf("rule %q, want cvc-pattern-valid", rule)
		}
	})
	t.Run("item 6 after an earlier item 7", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("order2"), order2{P: 1, Np: 1})
		plainError(t, err, "order2.Np")
	})
	t.Run("an attribute's item 6 before its element's", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("order2"), order2{P: 1, Np: 1, Oa: str("b")})
		plainError(t, err, "order2.Oa")
	})
}

// The backends of TestAppendNoCanonical and TestAppendBackendFaults, each
// strict's with one mapping replaced.
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
// error naming the value's Go path.
func TestAppendBackendFaults(t *testing.T) {
	schema := schemaOf(t, encodeSchema)
	_, err := appendChecked(t, nil, schema, doubleRejectingParse, sname("d"), dblRoot(1))
	plainError(t, err, "dblRoot", `doubleRep form "1E+00"`, "rejected")
	_, err = appendChecked(t, nil, schema, noMappings{}, sname("d"), dblRoot(1))
	plainError(t, err, "dblRoot", "no mapping governing")
	_, err = appendChecked(t, nil, schema, noMappings{}, sname("str"), strRoot("v"))
	plainError(t, err, "strRoot")
}

// asDecline fails t unless err is an Append or CheckAppend *DeclineError that wraps
// errors.ErrUnsupported, charged to no Loc, whose text opens with prefix,
// and returns it.
func asDecline(t *testing.T, err error, prefix string) *DeclineError {
	t.Helper()
	var d *DeclineError
	if !errors.As(err, &d) || !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("returned %v, want a *DeclineError wrapping errors.ErrUnsupported", err)
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
// element particle, and a wildcard particle, which declines so before it
// declines as a wildcard.
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
		{"wildcard particle", `<xs:any processContents="lax"/>`, false},
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

// TestAppendDeclines pins item 4 of Append's doc for each construct of the
// closed set it declines: each row's two declarations, r and q, decline with
// equal Constructs, named by the opening of the decline's text, and rows of
// different constructs with different ones; no bytes are written. A row's
// value is struct{}{}, or the one appendOf passes for it, which item 5
// accepts.
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
		{"nested sequence", "a nested model group particle", sequence(`<xs:sequence>` + a + `</xs:sequence>`)},
		{"nested group ref", "a nested model group particle", `<xs:group name="g%[1]s"><xs:sequence>` + a + `</xs:sequence></xs:group>` + sequence(`<xs:group ref="s:g%[1]s"/>`)},
		{"type table", "an element declaration with a {type table}", `<xs:element name="%[1]s" type="xs:string"><xs:alternative test="@a" type="xs:token"/></xs:element>`},
		{"element value constraint", "a value constraint, default or fixed", `<xs:element name="%[1]s" type="xs:string" default="x"/>`},
		{"attribute value constraint", "a value constraint, default or fixed", complexOf(`<xs:attribute name="a" type="xs:string" default="x"/>`)},
		{"identity constraint", "an element declaration with {identity-constraint definitions}", `<xs:element name="%[1]s"><xs:complexType><xs:sequence>` + a + `</xs:sequence></xs:complexType><xs:key name="k%[1]s"><xs:selector xpath="s:a"/><xs:field xpath="."/></xs:key></xs:element>`},
		{"anyType", "an element declaration of type xs:anyType", `<xs:element name="%[1]s"/>`},
		{"anyType particle", "an element declaration of type xs:anyType", sequence(`<xs:element name="a"/>`)},
		{"abstract type", "an element declaration whose {type definition} is a complex type with {abstract} true", abstractType},
		{"simple content", "a complex type with simple content", complexOf(`<xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent>`)},
		{"mixed", "a complex type with mixed content", `<xs:element name="%[1]s"><xs:complexType mixed="true"><xs:sequence>` + a + `</xs:sequence></xs:complexType></xs:element>`},
		{"open content", "an {open content}", complexOf(`<xs:openContent><xs:any namespace="##other" processContents="lax"/></xs:openContent><xs:sequence>` + a + `</xs:sequence>`)},
		{"all group", "an all group", complexOf(`<xs:all>` + a + `</xs:all>`)},
		{"choice", "a construct codegen holds in a sealed interface: a choice, a flattened particle or a substitution group", complexOf(`<xs:choice>` + a + `<xs:element name="b" type="xs:string"/></xs:choice>`)},
		{"flattened", "a construct codegen holds in a sealed interface: a choice, a flattened particle or a substitution group", complexOf(`<xs:sequence maxOccurs="unbounded"><xs:element name="a" type="xs:string" minOccurs="0"/></xs:sequence>`)},
		{"occurrence field", "a model group particle codegen holds in an occurrence field", complexOf(`<xs:sequence maxOccurs="2">` + a + `<xs:element name="b" type="xs:string"/></xs:sequence>`)},
		{"wildcard", "a wildcard particle", sequence(`<xs:any processContents="lax"/>`)},
		{"element ref", "an <element ref> particle", `<xs:element name="x%[1]s" type="xs:string"/>` + sequence(`<xs:element ref="s:x%[1]s"/>`)},
		{"complex particle", "an element particle of complex type", sequence(`<xs:element name="a"><xs:complexType/></xs:element>`)},
		{"qualified attribute", "an attribute use whose declaration has a {target namespace}", complexOf(`<xs:attribute name="a" type="xs:string" form="qualified"/>`)},
		{"attribute wildcard", "an {attribute wildcard}", complexOf(`<xs:anyAttribute processContents="lax"/>`)},
		{"complex assertions", "a complex type with {assertions}", complexOf(asserted)},
		{"restricted complex assertions", "a complex type with {assertions}", `<xs:complexType name="ab%[1]s">` + asserted + `</xs:complexType>` +
			`<xs:element name="%[1]s"><xs:complexType><xs:complexContent><xs:restriction base="s:ab%[1]s"><xs:sequence>` + a + `</xs:sequence></xs:restriction></xs:complexContent></xs:complexType></xs:element>`},
		{"special type", "xs:anySimpleType or xs:anyAtomicType as a simple type", `<xs:element name="%[1]s" type="xs:anySimpleType"/>`},
		{"list of decimal", "an atomic simple type whose primitive is other than xs:string or xs:double", simpleOf(`<xs:list itemType="xs:decimal"/>`)},
		{"list of union", "a union simple type", simpleOf(`<xs:list><xs:simpleType><xs:union memberTypes="xs:string xs:int"/></xs:simpleType></xs:list>`)},
		{"IDREFS", "a simple type derived from xs:ID, xs:IDREF or xs:ENTITY", `<xs:element name="%[1]s" type="xs:IDREFS"/>`},
		{"list assertions", "a simple type with an assertions facet", simpleOf(`<xs:restriction base="xs:NMTOKENS"><xs:assertion test="true()"/></xs:restriction>`)},
		{"inherited list assertions", "a simple type with an assertions facet", `<xs:simpleType name="assertedList%[1]s"><xs:restriction base="xs:NMTOKENS"><xs:assertion test="true()"/></xs:restriction></xs:simpleType>` + simpleOf(`<xs:restriction base="s:assertedList%[1]s"><xs:maxLength value="3"/></xs:restriction>`)},
		{"list item assertions", "a simple type with an assertions facet", `<xs:simpleType name="assertedItem%[1]s"><xs:restriction base="xs:string"><xs:assertion test="true()"/></xs:restriction></xs:simpleType>` + simpleOf(`<xs:list itemType="s:assertedItem%[1]s"/>`)},
		{"union", "a union simple type", simpleOf(`<xs:union memberTypes="xs:string xs:int"/>`)},
		{"boolean", "an atomic simple type whose primitive is other than xs:string or xs:double", `<xs:element name="%[1]s" type="xs:boolean"/>`},
		{"anyURI attribute", "an atomic simple type whose primitive is other than xs:string or xs:double", complexOf(`<xs:attribute name="a" type="xs:anyURI"/>`)},
		{"ID", "a simple type derived from xs:ID, xs:IDREF or xs:ENTITY", `<xs:element name="%[1]s" type="xs:ID"/>`},
		{"IDREF particle", "a simple type derived from xs:ID, xs:IDREF or xs:ENTITY", sequence(`<xs:element name="a" type="xs:IDREF"/>`)},
		{"ENTITY restriction", "a simple type derived from xs:ID, xs:IDREF or xs:ENTITY", simpleOf(`<xs:restriction base="xs:ENTITY"/>`)},
		{"assertions", "a simple type with an assertions facet", simpleOf(`<xs:restriction base="xs:string"><xs:assertion test="true()"/></xs:restriction>`)},
		{"inherited assertions", "a simple type with an assertions facet", `<xs:simpleType name="asserted%[1]s"><xs:restriction base="xs:string"><xs:assertion test="true()"/></xs:restriction></xs:simpleType>` + simpleOf(`<xs:restriction base="s:asserted%[1]s"><xs:maxLength value="3"/></xs:restriction>`)},
		{"particle value constraint", "a value constraint, default or fixed", sequence(`<xs:element name="a" type="xs:string" fixed="x"/>`)},
		{"particle type table", "an element declaration with a {type table}", sequence(`<xs:element name="a" type="xs:string"><xs:alternative test="@b" type="xs:token"/></xs:element>`)},
		{"particle identity constraint", "an element declaration with {identity-constraint definitions}", sequence(`<xs:element name="a" type="xs:string"><xs:unique name="u%[1]s"><xs:selector xpath="."/><xs:field xpath="."/></xs:unique></xs:element>`)},
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
	seen := map[string]Construct{}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			schema := schemaOf(t, fmt.Sprintf(row.decl, "r")+fmt.Sprintf(row.decl, "q"))
			dst := []byte("pre")
			var constructs []Construct
			for _, name := range []string{"r", "q"} {
				got, err := appendChecked(t, dst, schema, strict.New(), sname(name), struct{}{})
				if v, has := appendOf[row.name]; has {
					got, err = v(t, dst, schema, sname(name))
				}
				d := asDecline(t, err, "codec does not encode "+row.construct+": ")
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
// repeatedly ([][]V). Each document is accepted by validate and decoded back
// into the value, nil and empty one value.
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
// under item types that admit the item, an item's own check answering first
// and an item's own item 7 named at the item's path.
func TestAppendListInvalid(t *testing.T) {
	schema := schemaOf(t, encodeListSchema)
	for _, row := range []struct {
		name, elem string
		v          strList
		rule       xsderr.Rule
		want       []string
	}{
		{"an item holding a space", "sl", strList{"a", "b c"}, "", []string{"strList[1]", `"b c"`, "empty or holds white space", "through the list type's {item type definition} {http://www.w3.org/2001/XMLSchema}string"}},
		{"an empty item", "sl", strList{""}, "", []string{"strList[0]", `""`, "empty or holds white space"}},
		{"an item holding TAB", "sl", strList{"a\tb"}, "", []string{"strList[0]", "empty or holds white space"}},
		{"an item holding LF", "sl", strList{"a\nb"}, "", []string{"strList[0]", "empty or holds white space"}},
		{"an item holding CR", "sl", strList{"a\rb"}, "", []string{"strList[0]", "empty or holds white space"}},
		{"a token item holding a space", "tl", strList{"a b"}, "", []string{"strList[0]", "empty or holds white space"}},
		{"a non-Char item", "sl", strList{"a", "\x00"}, "", []string{"strList[1]", "XML 1.0 Char"}},
		{"a token item not collapsed", "tl", strList{" a"}, "", []string{"strList[0]", `whiteSpace-normalized form "a"`}},
		{"an empty NMTOKENS", "nm", strList{}, "cvc-minLength-valid", []string{"strList", "{http://www.w3.org/2001/XMLSchema}NMTOKENS"}},
		{"length", "pair", strList{"a"}, "cvc-length-valid", []string{"strList", `written "a"`, "{urn:s}pair"}},
		{"minLength", "some", nil, "cvc-minLength-valid", []string{"strList", "{urn:s}some"}},
		{"maxLength", "single", strList{"a", "b"}, "cvc-maxLength-valid", []string{"strList", `written "a b"`}},
		{"enumeration", "ab", strList{"a"}, "cvc-enumeration-valid", []string{"strList", "{urn:s}ab"}},
		{"a string list's pattern", "sx", strList{"a"}, "cvc-pattern-valid", []string{"strList", `written "a"`, "{urn:s}onlyX"}},
	} {
		t.Run(row.name, func(t *testing.T) {
			_, err := appendChecked(t, nil, schema, strict.New(), sname(row.elem), row.v)
			plainError(t, err, row.want...)
			if !strings.HasPrefix(err.Error(), "codec: Append: "+row.want[0]+" holds ") {
				t.Errorf("error %q does not open with the path %s", err, row.want[0])
			}
			if rule, _ := xsderr.RuleOf(err); rule != row.rule {
				t.Errorf("rule %q, want %q", rule, row.rule)
			}
		})
	}
	t.Run("a double list's empty value its pattern rejects", func(t *testing.T) {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("dp"), dblList{})
		plainError(t, err, "dblList", "{urn:s}twoDoubles")
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
			plainError(t, err, "dblList", "{urn:s}twoDoubles")
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
	schema := schemaOf(t, `<xs:element name="dec"><xs:simpleType><xs:list itemType="xs:decimal"/></xs:simpleType></xs:element>`+
		`<xs:simpleType name="booleans"><xs:list><xs:simpleType><xs:restriction base="xs:boolean"/></xs:simpleType></xs:list></xs:simpleType>`+
		`<xs:element name="anon" type="s:booleans"/>`)
	err := CheckAppend[strList](schema, sname("dec"))
	_ = asDecline(t, err, "codec does not encode an atomic simple type whose primitive is other than xs:string or xs:double: simple type the anonymous simple type of element {urn:s}dec declared at s.xsd:1:")
	if !strings.Contains(err.Error(), ", through the list type's {item type definition} {http://www.w3.org/2001/XMLSchema}decimal; ") {
		t.Errorf("decline %q does not name the item xs:decimal", err)
	}
	err = CheckAppend[strList](schema, sname("anon"))
	_ = asDecline(t, err, "codec does not encode an atomic simple type whose primitive is other than xs:string or xs:double: simple type {urn:s}booleans defined at s.xsd:1:")
	if !strings.Contains(err.Error(), ", through the list type's anonymous {item type definition}, a type defined at s.xsd:1:") {
		t.Errorf("decline %q does not name the anonymous item by its Loc", err)
	}
}
