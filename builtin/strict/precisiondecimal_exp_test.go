package strict

import (
	"testing"
	"time"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// pdDeadline bounds each huge-exponent row: a comparison that materialises
// 10^|scale| does not finish inside it (#1849), while one by adjusted exponent
// takes microseconds.
const pdDeadline = 10 * time.Second

// withinDeadline runs f and fails the test if it has not returned in pdDeadline.
// A row that times out leaves f running; the test binary exits with it.
func withinDeadline(t *testing.T, f func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		f()
	}()
	select {
	case <-done:
	case <-time.After(pdDeadline):
		t.Fatalf("did not finish within %s", pdDeadline)
	}
}

// pdValue parses lexical through strict's precisionDecimal mapping.
func pdValue(t *testing.T, lexical string) precisionDecimalVal {
	t.Helper()
	v, err := parsePrecisionDecimal(lexical, nil)
	if err != nil {
		t.Fatalf("Parse(%q): %v", lexical, err)
	}
	return v.(precisionDecimalVal)
}

// TestPrecisionDecimalCmpHugeExponent pins the §3.1 order on values whose
// exponents differ by about 10^9 (#1849): Cmp orders them by ·numericalValue·
// and finishes inside pdDeadline.
func TestPrecisionDecimalCmpHugeExponent(t *testing.T) {
	cases := []struct {
		a, b string
		want value.Ordering
	}{
		{"1E999999999", "10", value.Greater},
		{"10", "1E999999999", value.Less},
		{"1E-999999999", "10", value.Less},
		{"-1E999999999", "10", value.Less},
		{"-1E999999999", "-10", value.Less},
		{"-1E-999999999", "-10", value.Greater},
		{"0", "1E-999999999", value.Less},
		{"-0E999999999", "0E-999999999", value.Equal},
		{"1E999999999", "10E999999998", value.Equal},
		{"1.5E999999999", "15E999999998", value.Equal},
		{"1.5E999999999", "1.4999E999999999", value.Greater},
		{"9.9E999999998", "1E999999999", value.Less},
	}
	for _, c := range cases {
		t.Run(c.a+" vs "+c.b, func(t *testing.T) {
			a, b := pdValue(t, c.a), pdValue(t, c.b)
			var got value.Ordering
			withinDeadline(t, func() { got = a.Cmp(b) })
			if got != c.want {
				t.Errorf("Cmp(%s, %s) = %v, want %v", c.a, c.b, got, c.want)
			}
		})
	}
}

// TestPrecisionDecimalFacetsHugeExponent pins the bound and enumeration facets
// against a literal whose exponent dwarfs the facet's (#1849): each rejection
// charges its own rule (cvc-maxInclusive-valid, cvc-minInclusive-valid,
// cvc-enumeration-valid, §4.3) inside pdDeadline.
func TestPrecisionDecimalFacetsHugeExponent(t *testing.T) {
	pd := newPrim(t, "precisionDecimal")
	cases := []struct {
		name    string
		facet   xsd.Facet
		lexical string
		rule    xsderr.Rule
	}{
		{"maxInclusive 10", xsd.NewFacet(xsd.FacetMaxInclusive, []string{"10"}, false), "1E999999999", "cvc-maxInclusive-valid"},
		{"minInclusive 10", xsd.NewFacet(xsd.FacetMinInclusive, []string{"10"}, false), "1E-999999999", "cvc-minInclusive-valid"},
		{"enumeration 10", xsd.NewEnumerationFacet([]xsd.EnumerationMember{xsd.NewEnumerationMember("10", nil, nil)}), "1E999999999", "cvc-enumeration-valid"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			st := derive(t, "huge", pd, c.facet)
			var err error
			withinDeadline(t, func() { _, err = value.ValidateLexical(New(), noSchema{}, st, c.lexical, nil) })
			wantRule(t, err, c.rule)
		})
	}
}
