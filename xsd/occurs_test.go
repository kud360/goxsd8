package xsd_test

import (
	"errors"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

func TestNewOccursValid(t *testing.T) {
	cases := []struct {
		name     string
		min, max int
	}{
		{"one-one", 1, 1},
		{"zero-zero-vacuous", 0, 0},
		{"widened", 2, 5},
		{"equal-large", 7, 7},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			o, err := xsd.NewOccurs(c.min, c.max)
			if err != nil {
				t.Fatalf("NewOccurs(%d, %d) unexpected error: %v", c.min, c.max, err)
			}
			if o.Min() != c.min {
				t.Errorf("Min() = %d, want %d", o.Min(), c.min)
			}
			if o.IsUnbounded() {
				t.Errorf("IsUnbounded() = true, want false")
			}
			got, ok := o.Max()
			if !ok || got != c.max {
				t.Errorf("Max() = (%d, %v), want (%d, true)", got, ok, c.max)
			}
		})
	}
}

func TestNewOccursRejectsInvertedRange(t *testing.T) {
	_, err := xsd.NewOccurs(3, 2)
	if err == nil {
		t.Fatal("NewOccurs(3, 2) succeeded, want p-props-correct error")
	}
	assertRule(t, err, "p-props-correct")
}

func TestNewOccursRejectsNegativeMin(t *testing.T) {
	_, err := xsd.NewOccurs(-1, 5)
	if err == nil {
		t.Fatal("NewOccurs(-1, 5) succeeded, want p-props-correct error")
	}
	assertRule(t, err, "p-props-correct")
}

func TestNewOccursRejectsNegativeMax(t *testing.T) {
	_, err := xsd.NewOccurs(0, -1)
	if err == nil {
		t.Fatal("NewOccurs(0, -1) succeeded, want p-props-correct error")
	}
	assertRule(t, err, "p-props-correct")
}

func TestNewUnboundedOccursValid(t *testing.T) {
	for _, min := range []int{0, 1, 42} {
		o, err := xsd.NewUnboundedOccurs(min)
		if err != nil {
			t.Fatalf("NewUnboundedOccurs(%d) unexpected error: %v", min, err)
		}
		if o.Min() != min {
			t.Errorf("Min() = %d, want %d", o.Min(), min)
		}
		if !o.IsUnbounded() {
			t.Errorf("IsUnbounded() = false, want true")
		}
		if got, ok := o.Max(); ok {
			t.Errorf("Max() = (%d, true), want (_, false) for unbounded", got)
		}
	}
}

func TestNewUnboundedOccursRejectsNegativeMin(t *testing.T) {
	_, err := xsd.NewUnboundedOccurs(-1)
	if err == nil {
		t.Fatal("NewUnboundedOccurs(-1) succeeded, want p-props-correct error")
	}
	assertRule(t, err, "p-props-correct")
}

func TestOccursPermitsBounded(t *testing.T) {
	o, err := xsd.NewOccurs(2, 5)
	if err != nil {
		t.Fatalf("NewOccurs(2, 5): %v", err)
	}
	cases := []struct {
		n    int
		want bool
	}{
		{-1, false},
		{0, false},
		{1, false},
		{2, true}, // inclusive at min
		{3, true},
		{5, true}, // inclusive at max
		{6, false},
	}
	for _, c := range cases {
		if got := o.Permits(c.n); got != c.want {
			t.Errorf("(2..5).Permits(%d) = %v, want %v", c.n, got, c.want)
		}
	}
}

func TestOccursPermitsUnbounded(t *testing.T) {
	o, err := xsd.NewUnboundedOccurs(2)
	if err != nil {
		t.Fatalf("NewUnboundedOccurs(2): %v", err)
	}
	cases := []struct {
		n    int
		want bool
	}{
		{-1, false},
		{1, false},
		{2, true}, // inclusive at min
		{3, true},
		{1_000_000, true}, // no upper bound
	}
	for _, c := range cases {
		if got := o.Permits(c.n); got != c.want {
			t.Errorf("(2..unbounded).Permits(%d) = %v, want %v", c.n, got, c.want)
		}
	}
}

func TestOccursString(t *testing.T) {
	mustBounded := func(min, max int) xsd.Occurs {
		o, err := xsd.NewOccurs(min, max)
		if err != nil {
			t.Fatalf("NewOccurs(%d, %d): %v", min, max, err)
		}
		return o
	}
	mustUnbounded := func(min int) xsd.Occurs {
		o, err := xsd.NewUnboundedOccurs(min)
		if err != nil {
			t.Fatalf("NewUnboundedOccurs(%d): %v", min, err)
		}
		return o
	}
	cases := []struct {
		name string
		o    xsd.Occurs
		want string
	}{
		{"point", mustBounded(1, 1), "1"},
		{"zero-point", mustBounded(0, 0), "0"},
		{"range", mustBounded(2, 5), "2..5"},
		{"unbounded", mustUnbounded(1), "1..unbounded"},
		{"unbounded-zero", mustUnbounded(0), "0..unbounded"},
		{"zero-value", xsd.Occurs{}, "0"},
	}
	for _, c := range cases {
		if got := c.o.String(); got != c.want {
			t.Errorf("%s: String() = %q, want %q", c.name, got, c.want)
		}
	}
}

func assertRule(t *testing.T, err error, want xsderr.Rule) {
	t.Helper()
	var e *xsderr.Error
	if !errors.As(err, &e) {
		t.Fatalf("error %v is not an *xsderr.Error", err)
	}
	if got, ok := xsderr.RuleOf(err); !ok || got != want {
		t.Errorf("RuleOf = (%q, %v), want (%q, true)", got, ok, want)
	}
}
