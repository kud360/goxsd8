package goshape

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// TestTableSlot pins TableSlot over a table of two alternatives, of types a
// and b, and a default of type d: positions 0 and 1 are AlternativeSlot 1 and
// 2 holding a and b, position 2 the DefaultSlot holding d, and positions -1
// and 3 no slot.
func TestTableSlot(t *testing.T) {
	alternative := func(test *xsd.XPathExpression, local string) xsd.TypeAlternative {
		ta, err := xsd.NewTypeAlternative(xsderr.Loc{}, test, xsd.TypeDefinitionRef{Name: xsd.QName{Local: local}})
		if err != nil {
			t.Fatal(err)
		}
		return ta
	}
	test := xsd.NewXPathExpression("@k", nil, nil, nil)
	table, err := xsd.NewTypeTable(xsderr.Loc{}, []xsd.TypeAlternative{alternative(&test, "a"), alternative(&test, "b")}, alternative(nil, "d"))
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		index int
		slot  Slot
		local string // the held type's name; "" for ok false
	}{
		{0, AlternativeSlot(1), "a"},
		{1, AlternativeSlot(2), "b"},
		{2, DefaultSlot{}, "d"},
		{-1, nil, ""},
		{3, nil, ""},
	} {
		slot, ref, ok := TableSlot(table, row.index)
		if row.local == "" {
			if ok || slot != nil || ref != nil {
				t.Errorf("TableSlot(%d) = %v, %v, %t; want nil, nil, false", row.index, slot, ref, ok)
			}
			continue
		}
		named, isRef := ref.(xsd.TypeDefinitionRef)
		if !ok || slot != row.slot || !isRef || named.Name.Local != row.local {
			t.Errorf("TableSlot(%d) = %v, %v, %t; want %v holding %s, true", row.index, slot, ref, ok, row.slot, row.local)
		}
	}
}
