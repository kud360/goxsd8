package value_test

import (
	"fmt"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
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
	const xsdNS = "http://www.w3.org/2001/XMLSchema"
	decimal := xsd.QName{Space: xsdNS, Local: "decimal"}
	str := xsd.QName{Space: xsdNS, Local: "string"}

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
