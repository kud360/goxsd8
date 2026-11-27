package codec

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/xsd"
)

// assertionSchema declares, in urn:s, r, a sequence of one a, xs:string,
// whose type asserts s:a = 'x'; b, a named type of the same content and
// assertion, and d, a restriction of b with no <xs:assert> of its own, which
// inherits b's (Structures §3.4.2 {assertions}, derivation-ok-restriction
// clause 5); p, of no {assertions}; and n, simple content of xs:int whose
// type asserts string-length(.) = 2, which a document spelling 1 as 01
// satisfies and Append's canonical 1 does not.
const assertionSchema = `
  <xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
    <xs:assert test="s:a = 'x'"/></xs:complexType></xs:element>
  <xs:complexType name="b"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
    <xs:assert test="s:a = 'x'"/></xs:complexType>
  <xs:element name="d"><xs:complexType><xs:complexContent><xs:restriction base="s:b">
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:restriction></xs:complexContent></xs:complexType></xs:element>
  <xs:element name="p"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType></xs:element>
  <xs:element name="n"><xs:complexType><xs:simpleContent><xs:extension base="xs:int">
    <xs:assert test="string-length(.) = 2"/></xs:extension></xs:simpleContent></xs:complexType></xs:element>`

// assertionA is codegen's type for assertionSchema's r, d and p, and
// assertionInt for its n.
type (
	assertionA   struct{ A string }
	assertionInt struct{ Value int32 }
)

// TestAppendAssertion pins Append for a complex type with {assertions} none
// of whose {test}s names the in-scope namespaces: a value the assertion holds
// for, on the type itself and inherited through a restriction, is written,
// accepted by validate and decoded back equal; one it rejects is item 8 of
// Append's doc, charged under cvc-assertion (§3.13.4.1) to assertion 1 of 1;
// CheckAppend returns nil for each. The rows fail with complexEncodes' decline
// of every type with {assertions} put back, and the inherited row's charge
// with complexEncodes reading only the members d does not inherit from b.
// n's row pins item 8's clause on the string value: the value 1 has a valid
// document, 01, which validate accepts, and Append's 1 is charged all the same.
func TestAppendAssertion(t *testing.T) {
	schema := schemaOf(t, assertionSchema)
	for _, name := range []string{"r", "d"} {
		t.Run(name, func(t *testing.T) {
			roundTrip(t, schema, sname(name), assertionA{A: "x"}, `<`+name+` xmlns="urn:s"><a>x</a></`+name+`>`, deepEqual[assertionA])
			got, err := appendChecked(t, []byte("pre"), schema, strict.New(), sname(name), assertionA{A: "y"})
			assessedCharge(t, got, err, "cvc-assertion", "", "assertion 1 of 1")
		})
	}
	t.Run("a string value Append spells canonically", func(t *testing.T) {
		accepts(t, schema, []byte(`<n xmlns="urn:s">01</n>`))
		got, err := appendChecked(t, []byte("pre"), schema, strict.New(), sname("n"), assertionInt{Value: 1})
		assessedCharge(t, got, err, "cvc-assertion", "", "assertion 1 of 1")
	})
}

// inScopeE declares, in urn:s, e, of empty content and an xs:int attribute
// x, whose type asserts the {test} it is formatted with; the raw* forms
// declare r beside it, whose content Append writes raw: a strict wildcard,
// a lax one, or an element a of type xs:anyType itself, each with a required
// xs:QName attribute q, and rawStrictNoQ a strict wildcard alone.
const (
	inScopeE     = `<xs:element name="e"><xs:complexType><xs:sequence/><xs:attribute name="x" type="xs:int"/><xs:assert test="%s"/></xs:complexType></xs:element>`
	rawStrictQ   = `<xs:element name="r"><xs:complexType><xs:sequence><xs:any namespace="##targetNamespace"/></xs:sequence><xs:attribute name="q" type="xs:QName" use="required"/></xs:complexType></xs:element>`
	rawLaxQ      = `<xs:element name="r"><xs:complexType><xs:sequence><xs:any namespace="##targetNamespace" processContents="lax"/></xs:sequence><xs:attribute name="q" type="xs:QName" use="required"/></xs:complexType></xs:element>`
	rawAnyTypeQ  = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a"/></xs:sequence><xs:attribute name="q" type="xs:QName" use="required"/></xs:complexType></xs:element>`
	rawStrictNoQ = `<xs:element name="r"><xs:complexType><xs:sequence><xs:any namespace="##targetNamespace"/></xs:sequence></xs:complexType></xs:element>`
)

// assertionWildQ is codegen's type for rawStrictQ's and rawLaxQ's r, and
// assertionAnyTypeQ for rawAnyTypeQ's.
type (
	assertionWildQ struct {
		Any anyElement
		Q   tQName
	}
	assertionAnyTypeQ struct {
		A anyType
		Q tQName
	}
)

// TestAppendInScopeAssertionRaw pins item 8 of Append's doc for a {test}
// reading the in-scope namespaces of an element in content Append writes
// raw, where a top-level declaration governs it at run time. The value
// decoded from a valid document whose q is spelled x:l is charged under
// cvc-assertion, below a strict wildcard, a lax one and an xs:anyType
// element alike, since Append spells q with a fresh p1 it declares on r,
// which not(in-scope-prefixes(.) = 'p1') reads at e: the charge comes from
// Append's binding, not from the value. The positive reader over a decoded
// value is written and decoded back equal, since Append re-declares the
// value's a, and a {test} reading no in-scope namespace still charges a
// hand-built value the assertion rejects.
func TestAppendInScopeAssertionRaw(t *testing.T) {
	const negated = "not(in-scope-prefixes(.) = 'p1')"
	const doc = `<r xmlns="urn:s" xmlns:x="urn:x" q="x:l"><e/></r>`
	t.Run("strict wildcard", func(t *testing.T) {
		rawCharged[assertionWildQ](t, rawStrictQ+fmt.Sprintf(inScopeE, negated), doc)
	})
	t.Run("lax wildcard", func(t *testing.T) {
		rawCharged[assertionWildQ](t, rawLaxQ+fmt.Sprintf(inScopeE, negated), doc)
	})
	t.Run("xs:anyType element", func(t *testing.T) {
		rawCharged[assertionAnyTypeQ](t, rawAnyTypeQ+fmt.Sprintf(inScopeE, negated), `<r xmlns="urn:s" xmlns:x="urn:x" q="x:l"><a><e/></a></r>`)
	})
	t.Run("a positive reader", func(t *testing.T) {
		schema := schemaOf(t, rawStrictNoQ+fmt.Sprintf(inScopeE, "in-scope-prefixes(.) = 'a'"))
		v, err := Decode[wildOther](schema, strict.New(), []byte(`<r xmlns="urn:s"><e xmlns:a="urn:a"/></r>`))
		if err != nil {
			t.Fatal(err)
		}
		data := roundTrip(t, schema, sname("r"), *v, "", deepEqual[wildOther])
		back, err := Decode[wildOther](schema, strict.New(), data)
		if err != nil || !reflect.DeepEqual(*back, *v) {
			t.Fatalf("Decode(%s) = %#v, %v, want %#v", data, back, err, *v)
		}
	})
	t.Run("a reader of no in-scope namespace", func(t *testing.T) {
		schema := schemaOf(t, rawStrictNoQ+fmt.Sprintf(inScopeE, "@x &gt; 0"))
		v := wildOther{Any: anyElement{Space: "urn:s", Local: "e", Content: anyType{Attributes: []anyAttribute{{"", "x", "0"}}, Namespaces: []anyNamespace{{"", "urn:s"}}}}}
		got, err := appendChecked(t, []byte("pre"), schema, strict.New(), sname("r"), v)
		assessedCharge(t, got, err, "cvc-assertion", "", "")
	})
}

// rawCharged fails t unless validate accepts data against decls' schema and
// the T Decode returns for it is item 8 of Append's doc as r, under
// cvc-assertion (assessedCharge), no text of validate's asserted.
func rawCharged[T any](t *testing.T, decls, data string) {
	t.Helper()
	schema := schemaOf(t, decls)
	accepts(t, schema, []byte(data))
	v, err := Decode[T](schema, strict.New(), []byte(data))
	if err != nil {
		t.Fatalf("Decode(%s): %v", data, err)
	}
	got, err := appendChecked(t, []byte("pre"), schema, strict.New(), sname("r"), *v)
	assessedCharge(t, got, err, "cvc-assertion", "", "")
}

// TestAppendAssertionUndecided pins item 9 of Append's doc for {assertions}:
// over ExampleUndecidedError's schema (undecidedSchema), whose {test} the
// XPath subset declines, a value is refused with dst unextended and an error
// errors.As reaches as an *UndecidedError for cvc-assertion, which errors.Is
// reports as errors.ErrUnsupported and not as ErrInvalidValue; CheckAppend
// returns nil for it.
func TestAppendAssertionUndecided(t *testing.T) {
	schema := schemaOf(t, undecidedSchema)
	got, err := appendChecked(t, []byte("pre"), schema, strict.New(), sname("r"), assertionA{A: "x"})
	if string(got) != "pre" {
		t.Errorf("Append returned %q, want dst unextended", got)
	}
	var ue *UndecidedError
	if !errors.As(err, &ue) || !errors.Is(err, errors.ErrUnsupported) || errors.Is(err, ErrInvalidValue) {
		t.Fatalf("Append returned %v, want an *UndecidedError that errors.Is reports as errors.ErrUnsupported and not ErrInvalidValue", err)
	}
	if rule := ue.Unevaluated().Rule(); rule != "cvc-assertion" {
		t.Errorf("Unevaluated().Rule() = %q, want cvc-assertion", rule)
	}
}

// TestAssertionsAssess pins elementEncodes' bool, whether Append assesses
// what it writes, for {assertions}: true for r and for d, whose assertion is
// inherited, and false for p, of no {assertions}, which fails under a
// complexEncodes that sets w.assess for every complex type.
func TestAssertionsAssess(t *testing.T) {
	schema := schemaOf(t, assertionSchema)
	for _, row := range []struct {
		name string
		want bool
	}{{"r", true}, {"d", true}, {"p", false}} {
		decl, ok := schema.Element(sname(row.name))
		if !ok {
			t.Fatalf("no declaration %s", row.name)
		}
		assess, err := elementEncodes(schema, decl)
		if err != nil || assess != row.want {
			t.Errorf("elementEncodes(%s) = %v, %v, want %v, nil", row.name, assess, err, row.want)
		}
	}
}

// TestNamesInScopeNamespaces pins namesInScopeNamespaces over {test} texts:
// each of the three function names, prefixed or not, and the namespace axis,
// with and without white space before "::", match; the axis with an XPath
// comment before "::", which the package doc's "# What encodes" leaves out,
// namespace-uri and a name test namespace, in a comparison and as a path, do
// not.
func TestNamesInScopeNamespaces(t *testing.T) {
	for _, row := range []struct {
		test string
		want bool
	}{
		{"in-scope-prefixes(.) = 'a'", true},
		{"fn:in-scope-prefixes(s:a)", true},
		{"namespace-uri-for-prefix('a', .) = 'urn:a'", true},
		{"resolve-QName('a:b', .) = xs:QName('s:b')", true},
		{"count(namespace::*) = 2", true},
		{"count(./namespace ::*) = 2", true},
		{"count(namespace (: c :) ::*) = 2", false},
		{"namespace-uri(.) = 'urn:s'", false},
		{"s:namespace = 'x'", false},
		{"count(s:namespace) = 0", false},
		{"s:a = 'x'", false},
	} {
		got := namesInScopeNamespaces(xsd.NewXPathExpression(row.test, nil, nil, nil))
		if got != row.want {
			t.Errorf("namesInScopeNamespaces(%q) = %v, want %v", row.test, got, row.want)
		}
	}
}
