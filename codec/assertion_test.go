package codec

import (
	"errors"
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
// with and without white space before "::", match; namespace-uri and a name
// test namespace, in a comparison and as a path, do not.
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
