package value_test

import (
	"fmt"
	"strings"
	"time"

	"github.com/kud360/goxsd8/builtin"
	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xpath"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// oneType is a trivial value.Backend that maps exactly one QName. A real
// backend parses lexical forms into typed values implementing the value
// capabilities; this stub only needs to demonstrate composition.
type oneType struct {
	typ     xsd.QName
	mapping value.Mapping
}

func (b oneType) Mapping(typ xsd.QName) (value.Mapping, bool) {
	if typ == b.typ {
		return b.mapping, true
	}
	return value.Mapping{}, false
}

// ExampleOverride backs only xs:decimal with a custom "money" mapping while
// inheriting every other type — here xs:string — from the base backend.
func ExampleOverride() {
	// A QName is an expanded name {namespace}local; the XSD builtins live in
	// the XML Schema namespace.
	decimal := xsd.QName{Space: xsd.XMLSchemaNS, Local: "decimal"}
	str := xsd.QName{Space: xsd.XMLSchemaNS, Local: "string"}

	base := oneType{
		typ: str,
		mapping: value.Mapping{
			Parse: func(lexical string, _ value.Context) (value.Value, error) {
				return lexical, nil
			},
		},
	}
	money := oneType{
		typ: decimal,
		mapping: value.Mapping{
			Parse: func(lexical string, _ value.Context) (value.Value, error) {
				return "money(" + lexical + ")", nil
			},
		},
	}

	// Override yields money's mapping for xs:decimal and base's for the rest.
	backend := value.Override(base, money)

	if m, ok := backend.Mapping(decimal); ok {
		v, _ := m.Parse("3.14", nil)
		fmt.Printf("decimal -> %v\n", v)
	}
	if m, ok := backend.Mapping(str); ok {
		v, _ := m.Parse("hello", nil)
		fmt.Printf("string  -> %v\n", v)
	}

	// Output:
	// decimal -> money(3.14)
	// string  -> hello
}

// builtinsOnly is an xsd.TypeResolver that resolves no name: the seeded
// builtins link their {base type definition} chains by pointer, so a type
// drawn from builtin.Seed needs no by-name lookup.
type builtinsOnly struct{}

func (builtinsOnly) Type(xsd.QName) (xsd.TypeDefinition, bool) { return nil, false }

// ExampleValidateLexical reaches the value of a type its backend does not map
// directly. builtin/strict maps xs:decimal but not xs:integer, so
// Backend.Mapping answers false for xs:integer; ValidateLexical runs the whole
// facet pipeline and parses the literal under xs:decimal's mapping, the nearest
// mapped ancestor. The evaluator is the one package validate passes.
func ExampleValidateLexical() {
	backend := strict.New()
	seeded, err := builtin.Seed(backend)
	if err != nil {
		fmt.Println(err)
		return
	}
	var integer *xsd.SimpleType
	for _, st := range seeded {
		if st.Name() == (xsd.QName{Space: xsd.XMLSchemaNS, Local: "integer"}) {
			integer = st
		}
	}

	_, mapped := backend.Mapping(integer.Name())
	fmt.Println("mapped directly:", mapped)

	v, err := value.ValidateLexical(backend, builtinsOnly{}, integer, "42", nil, xpath.FacetAssertions(time.Time{}))
	if err != nil {
		fmt.Println(err)
		return
	}
	if c, ok := v.(value.Canonical); ok {
		fmt.Println("value:", c.Canonical())
	}

	// Output:
	// mapped directly: false
	// value: 42
}

// evenOnly is an evaluator that knows one {test}, `$value mod 2 = 0`, and
// declines every other one, saying why.
type evenOnly struct{}

func (evenOnly) Evaluate(_ value.Backend, _ xsd.TypeResolver, _ *xsd.SimpleType, test xsd.XPathExpression, v value.Value) value.AssertionAnswer {
	c, ok := v.(value.Canonical)
	if test.Expression() != "$value mod 2 = 0" || !ok {
		return value.AssertionDeclinedBecause("this evaluator knows only one test")
	}
	lexical := c.Canonical()
	return value.AssertionDecided(strings.ContainsAny(lexical[len(lexical)-1:], "02468"))
}

// ExampleAssertionEvaluator runs a caller-written evaluator through
// ValidateLexical on two restrictions of xs:integer, each carrying an
// assertions facet. The evaluator decides the {test} it recognises and
// declines the other, and IsAssertionDeclined tells that decline apart from a
// rejection.
//
// A caller holds an xsd.XPathExpression either by building one with
// xsd.NewXPathExpression, as here, or by reading Assertion.Test() off the
// assertions facet of a parsed type. Package validate installs its own
// evaluator, so one written here is reached only by calling ValidateLexical,
// ValidatingType or ConstraintMatches directly.
func ExampleAssertionEvaluator() {
	backend := strict.New()
	seeded, err := builtin.Seed(backend)
	if err != nil {
		fmt.Println(err)
		return
	}
	var integer *xsd.SimpleType
	for _, st := range seeded {
		if st.Name() == (xsd.QName{Space: xsd.XMLSchemaNS, Local: "integer"}) {
			integer = st
		}
	}

	// asserting restricts xs:integer to {urn:ex}local with one assertion.
	asserting := func(local, test string) (*xsd.SimpleType, error) {
		facet := xsd.NewAssertionsFacet([]xsd.Assertion{
			xsd.NewAssertion(xsd.NewXPathExpression(test, nil, nil, nil)),
		})
		return xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:ex", Local: local},
			xsd.RestrictionDerivation{}, xsd.OwnedSimpleType{Definition: integer}, []xsd.Facet{facet}, nil)
	}
	even, err := asserting("even", "$value mod 2 = 0")
	if err != nil {
		fmt.Println(err)
		return
	}
	small, err := asserting("small", "$value lt 100")
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, call := range []struct {
		st      *xsd.SimpleType
		lexical string
	}{{even, "42"}, {even, "7"}, {small, "42"}} {
		_, err := value.ValidateLexical(backend, builtinsOnly{}, call.st, call.lexical, nil, evenOnly{})
		switch {
		case err == nil:
			fmt.Printf("%s %s: valid\n", call.st.Name().Local, call.lexical)
		case value.IsAssertionDeclined(err):
			fmt.Printf("%s %s: undecided\n%v\n", call.st.Name().Local, call.lexical, err)
		case value.IsDatatypeVerdict(err):
			fmt.Printf("%s %s: invalid\n", call.st.Name().Local, call.lexical)
		default:
			fmt.Printf("%s %s: a fault of the type or backend: %v\n", call.st.Name().Local, call.lexical, err)
		}
	}

	// Output:
	// even 42: valid
	// even 7: invalid
	// small 42: undecided
	// ?: [cvc-assertions-valid] assertion 1 of 1 in the {value} of the assertions facet of the simple type {urn:ex}small, whose {test} is "$value lt 100", was not evaluated because this evaluator knows only one test, so it is undecided whether the value is facet-valid with respect to it, which cvc-assertions-valid requires
}
