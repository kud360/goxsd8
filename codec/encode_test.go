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
	"slices"
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
// or a fault of backend: nil, item 6's ErrInvalidValue error, a plain error
// without item 2's or item 3's text, or item 7's decline.
func appendChecked[T any](t *testing.T, dst []byte, schema *xsd.Schema, backend value.Backend, name xsd.QName, v T) ([]byte, error) {
	t.Helper()
	out, err := Append(dst, schema, backend, name, v)
	checked := CheckAppend[T](schema, name)
	if checked == nil {
		class := staticClass(err)
		static := class == "plain" && (strings.HasPrefix(err.Error(), "codec: no top-level element declaration") || strings.Contains(err.Error(), "{abstract} true"))
		if static || err != nil && class != "plain" && class != "invalid" && class != "decline "+constructNoCanonical.name() {
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
	if want := ErrTypeMismatch.Error() + ": *codec.shapes is a pointer to codec.shapes, which matches; use codec.shapes in its place"; !errors.Is(err, ErrTypeMismatch) || err.Error() != want {
		t.Errorf("Append[*shapes] = %v, want ErrTypeMismatch %q", err, want)
	}
	_, err = appendChecked(t, nil, schema, strict.New(), rootName, struct{ One string }{})
	if !errors.Is(err, ErrTypeMismatch) {
		t.Errorf("Append[struct{One string}] = %v, want ErrTypeMismatch", err)
	}
}

// TestCheckAppend pins CheckAppend's answers by Append's items: a plain
// error for a nil schema; item 4's decline for an xs:float declaration and
// T float32, which CheckDecode accepts; nil for README's order.xsd and
// order.Order, whose zero value Append refuses by item 6, and item 5's
// ErrTypeMismatch for *order.Order; item 2's and item 3's plain errors; and
// item 3 before item 4 on an abstract xs:float declaration. appendChecked
// holds each answer to Append's.
func TestCheckAppend(t *testing.T) {
	schema := schemaOf(t, `<xs:element name="i" type="xs:float"/><xs:element name="abs" type="xs:string" abstract="true"/>`+
		`<xs:element name="absb" type="xs:float" abstract="true"/>`)
	plainError(t, CheckAppend[string](nil, sname("i")), "codec: CheckAppend: nil *xsd.Schema")

	if err := CheckDecode[float32](schema, sname("i")); err != nil {
		t.Fatalf("CheckDecode[float32] = %v, want nil", err)
	}
	err := CheckAppend[float32](schema, sname("i"))
	_ = asDecline(t, err, "codec does not encode an atomic simple type whose primitive is other than xs:string, xs:boolean, xs:decimal, xs:double, xs:duration, xs:dateTime, xs:time, xs:date, xs:gYearMonth, xs:gYear, xs:gMonthDay, xs:gDay or xs:gMonth: ")
	_, _ = appendChecked(t, nil, schema, strict.New(), sname("i"), float32(0))

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
		t.Run("an xs:double "+doubleLexical(f)+" a pattern rejects", func(t *testing.T) {
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
		{"list of float", "an atomic simple type whose primitive is other than xs:string, xs:boolean, xs:decimal, xs:double, xs:duration, xs:dateTime, xs:time, xs:date, xs:gYearMonth, xs:gYear, xs:gMonthDay, xs:gDay or xs:gMonth", simpleOf(`<xs:list itemType="xs:float"/>`)},
		{"list of union", "a union simple type", simpleOf(`<xs:list><xs:simpleType><xs:union memberTypes="xs:string xs:int"/></xs:simpleType></xs:list>`)},
		{"IDREFS", "a simple type derived from xs:ID, xs:IDREF or xs:ENTITY", `<xs:element name="%[1]s" type="xs:IDREFS"/>`},
		{"list assertions", "a simple type with an assertions facet", simpleOf(`<xs:restriction base="xs:NMTOKENS"><xs:assertion test="true()"/></xs:restriction>`)},
		{"inherited list assertions", "a simple type with an assertions facet", `<xs:simpleType name="assertedList%[1]s"><xs:restriction base="xs:NMTOKENS"><xs:assertion test="true()"/></xs:restriction></xs:simpleType>` + simpleOf(`<xs:restriction base="s:assertedList%[1]s"><xs:maxLength value="3"/></xs:restriction>`)},
		{"list item assertions", "a simple type with an assertions facet", `<xs:simpleType name="assertedItem%[1]s"><xs:restriction base="xs:string"><xs:assertion test="true()"/></xs:restriction></xs:simpleType>` + simpleOf(`<xs:list itemType="s:assertedItem%[1]s"/>`)},
		{"union", "a union simple type", simpleOf(`<xs:union memberTypes="xs:string xs:int"/>`)},
		{"hexBinary", "an atomic simple type whose primitive is other than xs:string, xs:boolean, xs:decimal, xs:double, xs:duration, xs:dateTime, xs:time, xs:date, xs:gYearMonth, xs:gYear, xs:gMonthDay, xs:gDay or xs:gMonth", `<xs:element name="%[1]s" type="xs:hexBinary"/>`},
		{"float", "an atomic simple type whose primitive is other than xs:string, xs:boolean, xs:decimal, xs:double, xs:duration, xs:dateTime, xs:time, xs:date, xs:gYearMonth, xs:gYear, xs:gMonthDay, xs:gDay or xs:gMonth", `<xs:element name="%[1]s" type="xs:float"/>`},
		{"anyURI attribute", "an atomic simple type whose primitive is other than xs:string, xs:boolean, xs:decimal, xs:double, xs:duration, xs:dateTime, xs:time, xs:date, xs:gYearMonth, xs:gYear, xs:gMonthDay, xs:gDay or xs:gMonth", complexOf(`<xs:attribute name="a" type="xs:anyURI"/>`)},
		{"ID", "a simple type derived from xs:ID, xs:IDREF or xs:ENTITY", `<xs:element name="%[1]s" type="xs:ID"/>`},
		{"IDREF particle", "a simple type derived from xs:ID, xs:IDREF or xs:ENTITY", sequence(`<xs:element name="a" type="xs:IDREF"/>`)},
		{"ENTITY restriction", "a simple type derived from xs:ID, xs:IDREF or xs:ENTITY", simpleOf(`<xs:restriction base="xs:ENTITY"/>`)},
		{"assertions", "a simple type with an assertions facet", simpleOf(`<xs:restriction base="xs:string"><xs:assertion test="true()"/></xs:restriction>`)},
		{"decimal assertions", "a simple type with an assertions facet", simpleOf(`<xs:restriction base="xs:decimal"><xs:assertion test="true()"/></xs:restriction>`)},
		{"boolean assertions", "a simple type with an assertions facet", simpleOf(`<xs:restriction base="xs:boolean"><xs:assertion test="true()"/></xs:restriction>`)},
		{"date assertions", "a simple type with an assertions facet", simpleOf(`<xs:restriction base="xs:date"><xs:assertion test="true()"/></xs:restriction>`)},
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
	schema := schemaOf(t, `<xs:element name="flt"><xs:simpleType><xs:list itemType="xs:float"/></xs:simpleType></xs:element>`+
		`<xs:simpleType name="dates"><xs:list><xs:simpleType><xs:restriction base="xs:hexBinary"/></xs:simpleType></xs:list></xs:simpleType>`+
		`<xs:element name="anon" type="s:dates"/>`)
	err := CheckAppend[strList](schema, sname("flt"))
	_ = asDecline(t, err, "codec does not encode an atomic simple type whose primitive is other than xs:string, xs:boolean, xs:decimal, xs:double, xs:duration, xs:dateTime, xs:time, xs:date, xs:gYearMonth, xs:gYear, xs:gMonthDay, xs:gDay or xs:gMonth: simple type the anonymous simple type of element {urn:s}flt declared at s.xsd:1:")
	if !strings.Contains(err.Error(), ", through the list type's {item type definition} xs:float; ") {
		t.Errorf("decline %q does not name the item xs:float", err)
	}
	err = CheckAppend[strList](schema, sname("anon"))
	_ = asDecline(t, err, "codec does not encode an atomic simple type whose primitive is other than xs:string, xs:boolean, xs:decimal, xs:double, xs:duration, xs:dateTime, xs:time, xs:date, xs:gYearMonth, xs:gYear, xs:gMonthDay, xs:gDay or xs:gMonth: simple type {urn:s}dates defined at s.xsd:1:")
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
// such value has another lexical; and a backend's mapping of
// xs:positiveInteger rejecting 0 is item 6, a verdict of the governing
// mapping's Parse, never a backend fault.
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
