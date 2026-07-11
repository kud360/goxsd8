package strict_test

import (
	"fmt"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// ExampleNew parses two decimal lexicals, discovers the value's ordering
// capability by asserting value.Ordered (no type switch over an unexported
// type), compares them, and shows the canonical round-trip +1.0 → "1".
func ExampleNew() {
	backend := strict.New()

	m, _ := backend.Mapping(xsd.QName{Space: xsd.XMLSchemaNS, Local: "decimal"})

	v, err := m.Parse("+1.0", nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	// decimal values satisfy value.Ordered; assert the capability, then compare.
	other, _ := m.Parse("1.00", nil)
	fmt.Println(v.(value.Ordered).Cmp(other) == value.Equal)

	c, _ := m.Canonical(v)
	fmt.Println(c)

	// Output:
	// true
	// 1
}
