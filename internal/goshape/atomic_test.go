package goshape

import (
	"math"
	"testing"
)

// TestDatatypeFields holds that every Atomic Datatype answers for has a
// datatype struct of at least one field, AtomicPrecisionDecimal aside, whose
// numerical arm NumericalFields gives instead, and that no other Atomic has
// any. Every value of Atomic's range is enumerated through Datatype itself, so
// a datatypeFields entry deleted or emptied fails here.
func TestDatatypeFields(t *testing.T) {
	for i := 1; i <= math.MaxUint8; i++ {
		a := Atomic(i)
		name, ok := a.Datatype()
		fields := a.Fields()
		if !ok || a == AtomicPrecisionDecimal {
			if fields != nil {
				t.Errorf("Atomic %d (datatype %v, %t) has fields %v, want none", a, name, ok, fields)
			}
			continue
		}
		if len(fields) == 0 {
			t.Errorf("the datatype struct of %v (Atomic %d) has no field", name, a)
		}
		for _, f := range fields {
			if f.Name == "" || f.Type == nil {
				t.Errorf("the datatype struct of %v has the field %+v", name, f)
			}
		}
	}
	if len(NumericalFields()) == 0 {
		t.Error("PrecisionDecimal's numerical arm has no field")
	}
}
