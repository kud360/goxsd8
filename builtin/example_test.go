package builtin_test

import (
	"errors"
	"fmt"

	"github.com/kud360/goxsd8/builtin"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// exampleFullBackend is a stub value.Backend that maps every builtin primitive
// with a trivial identity mapping — just enough to satisfy Seed's precondition.
// A real program composes builtin/strict with value.Override instead.
type exampleFullBackend struct{}

func (exampleFullBackend) Mapping(typ xsd.QName) (value.Mapping, bool) {
	if typ.Space != xsd.XMLSchemaNS {
		return value.Mapping{}, false
	}
	for _, t := range builtin.Types {
		if t.IsPrimitive() && t.Name == typ.Local {
			return value.Mapping{
				Parse: func(lexical string, _ value.Context) (value.Value, error) { return lexical, nil },
			}, true
		}
	}
	return value.Mapping{}, false
}

// exampleEmptyBackend maps nothing, so Seed reports every primitive as missing.
type exampleEmptyBackend struct{}

func (exampleEmptyBackend) Mapping(xsd.QName) (value.Mapping, bool) {
	return value.Mapping{}, false
}

// ExampleSeed shows the consumption idiom: Seed the builtins from a backend,
// then index the returned components into a symbol table keyed by QName. No
// parser/symbol-table constructor exists yet (that is a later M4 surface), so a
// consumer keys the slice itself for now.
func ExampleSeed() {
	types, err := builtin.Seed(exampleFullBackend{})
	if err != nil {
		fmt.Println(err)
		return
	}

	sym := make(map[xsd.QName]*xsd.SimpleType, len(types))
	for _, t := range types {
		sym[t.Name()] = t
	}

	decimal := sym[xsd.QName{Space: xsd.XMLSchemaNS, Local: "decimal"}]
	fmt.Println("components:", len(types) == len(builtin.Types)+1)
	fmt.Println("first:", types[0].Name().Local)
	fmt.Println("decimal base:", decimal.Base().Name().Local)
	// Output:
	// components: true
	// first: anySimpleType
	// decimal base: anyAtomicType
}

// ExampleSeed_missingPrimitive shows detecting an under-provisioned backend:
// Seed fails with a typed *MissingPrimitivesError that errors.As recovers,
// listing EVERY unmapped primitive in Types order (not just the first).
func ExampleSeed_missingPrimitive() {
	_, err := builtin.Seed(exampleEmptyBackend{})

	var missing *builtin.MissingPrimitivesError
	if errors.As(err, &missing) {
		fmt.Printf("missing %d primitives; first is %s\n", len(missing.Missing), missing.Missing[0])
	}
	// Output:
	// missing 20 primitives; first is {http://www.w3.org/2001/XMLSchema}string
}
