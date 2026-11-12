package goshape

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// TestBranchOccurs pins the branch occurrence codegen/doc.go "# Complex
// types" states: a {min occurs} 0 reads as 1 under a {max occurs} of 1, 2 or
// unbounded, which is kept, and an occurrence with {min occurs} above 0 or
// {max occurs} 0 is returned unchanged.
func TestBranchOccurs(t *testing.T) {
	bounded := func(min, max int) xsd.Occurs {
		o, err := xsd.NewOccurs(xsderr.Loc{}, min, max)
		if err != nil {
			t.Fatal(err)
		}
		return o
	}
	unbounded := func(min int) xsd.Occurs {
		o, err := xsd.NewUnboundedOccurs(xsderr.Loc{}, min)
		if err != nil {
			t.Fatal(err)
		}
		return o
	}
	rows := []struct {
		name     string
		in, want xsd.Occurs
	}{
		{"{0,1}", bounded(0, 1), bounded(1, 1)},
		{"{0,2}", bounded(0, 2), bounded(1, 2)},
		{"{0,unbounded}", unbounded(0), unbounded(1)},
		{"{1,1}", bounded(1, 1), bounded(1, 1)},
		{"{2,unbounded}", unbounded(2), unbounded(2)},
		{"{0,0}", bounded(0, 0), bounded(0, 0)},
	}
	for _, row := range rows {
		if got := BranchOccurs(row.in); got != row.want {
			t.Errorf("BranchOccurs(%s) = %+v, want %+v", row.name, got, row.want)
		}
	}
}
