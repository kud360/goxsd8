package backendtest_test

import (
	"fmt"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/value/backendtest"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// oneType is a value.Backend that maps exactly one QName — enough to stand in
// for a real backend while demonstrating composition.
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

// ExampleRun composes a base backend with a custom one via value.Override and
// hands the result to backendtest.Run. Composition is the headline consumer
// story: back only the types you specialize, inherit the rest, and one Run
// verifies the whole composed backend.
//
// Run needs a *testing.T, so the runnable body below shows the round-trip Run
// checks for one vector; inside a test the call is simply:
//
//	backendtest.Run(t, backend)
func ExampleRun() {
	boolean := xsd.QName{Space: xsd.XMLSchemaNS, Local: "boolean"}
	str := xsd.QName{Space: xsd.XMLSchemaNS, Local: "string"}

	// base covers xs:string; a custom backend covers xs:boolean.
	base := oneType{typ: str, mapping: value.Mapping{
		Parse: func(lexical string, _ value.Context) (value.Value, error) { return lexical, nil },
	}}
	booleans := oneType{typ: boolean, mapping: value.Mapping{
		Parse: func(lexical string, _ value.Context) (value.Value, error) {
			switch lexical {
			case "true", "1":
				return true, nil
			case "false", "0":
				return false, nil
			}
			return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{}, "boolean: %q not in lexical space", lexical)
		},
		Canonical: func(v value.Value) (string, error) {
			if v.(bool) {
				return "true", nil
			}
			return "false", nil
		},
	}}

	// Override yields booleans' mapping for xs:boolean and base's for the rest.
	backend := value.Override(base, booleans)

	// In a test, one call — backendtest.Run(t, backend) — drives every vector;
	// here we show the lexical→value→canonical round-trip it verifies for one
	// boolean lexical. Run itself needs a *testing.T, which a runnable Example
	// cannot construct, so this reference just pins the call Run's godoc shows.
	_ = backendtest.Run
	m, _ := backend.Mapping(boolean)
	v, _ := m.Parse("1", nil)
	c, _ := m.Canonical(v)
	fmt.Printf("boolean %q -> %s\n", "1", c)
	// Output:
	// boolean "1" -> true
}
