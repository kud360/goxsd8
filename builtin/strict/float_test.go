package strict_test

import (
	"testing"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsderr"
)

// parseVal parses lex through local's mapping, failing the test on rejection.
func parseVal(t *testing.T, local, lex string) value.Value {
	t.Helper()
	v, err := mappingFor(t, local).Parse(lex, nil)
	if err != nil {
		t.Fatalf("%s: Parse(%q) unexpected error: %v", local, lex, err)
	}
	return v
}

// TestFloatingParseCanonical checks the round-trip for both float and double,
// including the special values and the signed-zero canonical forms (§3.3.4.2/
// §3.3.5.2). The canonical strings are hand-verified against scientificCanonicalMap
// so this bites independently of the generated backendtest vectors.
func TestFloatingParseCanonical(t *testing.T) {
	cases := []struct{ lex, canon string }{
		{"INF", "INF"}, {"+INF", "INF"}, {"-INF", "-INF"}, {"NaN", "NaN"},
		{"0", "0.0E0"}, {"-0", "-0.0E0"},
		{"1", "1.0E0"}, {"-1", "-1.0E0"}, {"100", "1.0E2"},
		{"1.5E1", "1.5E1"}, {"15", "1.5E1"}, {".5", "5.0E-1"},
		{"3.14", "3.14E0"}, {"-0.001", "-1.0E-3"},
	}
	for _, local := range []string{"float", "double"} {
		m := mappingFor(t, local)
		for _, c := range cases {
			v, err := m.Parse(c.lex, nil)
			if err != nil {
				t.Errorf("%s: Parse(%q): %v", local, c.lex, err)
				continue
			}
			got, err := m.Canonical(v)
			if err != nil {
				t.Errorf("%s: Canonical(%q): %v", local, c.lex, err)
				continue
			}
			if got != c.canon {
				t.Errorf("%s: Canonical(Parse(%q)) = %q, want %q", local, c.lex, got, c.canon)
			}
		}
	}
}

// TestFloatingOverflowUnderflowValid proves out-of-range numerals are VALID: an
// overflow maps to ±INF and an underflow to a signed zero (floatingPointRound),
// never a cvc-datatype-valid rejection.
func TestFloatingOverflowUnderflowValid(t *testing.T) {
	cases := map[string]map[string]string{
		"float":  {"1e40": "INF", "-1e40": "-INF", "1e-50": "0.0E0", "-1e-50": "-0.0E0"},
		"double": {"1e400": "INF", "-1e400": "-INF", "1e-400": "0.0E0", "-1e-400": "-0.0E0"},
	}
	for local, m := range cases {
		mp := mappingFor(t, local)
		for lex, wantCanon := range m {
			v, err := mp.Parse(lex, nil)
			if err != nil {
				t.Errorf("%s: Parse(%q): out-of-range must be valid, got %v", local, lex, err)
				continue
			}
			got, _ := mp.Canonical(v)
			if got != wantCanon {
				t.Errorf("%s: Canonical(Parse(%q)) = %q, want %q", local, lex, got, wantCanon)
			}
		}
	}
}

// TestFloatingReject proves lexicals outside the space are rejected as
// cvc-datatype-valid — notably +NaN/-NaN (the special grammar admits only bare
// NaN, though +INF is legal), case variants, whitespace and dangling exponents.
func TestFloatingReject(t *testing.T) {
	for _, local := range []string{"float", "double"} {
		m := mappingFor(t, local)
		for _, lex := range []string{"+NaN", "-NaN", "Infinity", "inf", "nan", "", " 1", "1 ", "1.5e", "++1", "1.0.0", "0x1", "E5"} {
			_, err := m.Parse(lex, nil)
			if err == nil {
				t.Errorf("%s: Parse(%q): want cvc-datatype-valid rejection, got nil", local, lex)
				continue
			}
			if rule, ok := xsderr.RuleOf(err); !ok || rule != "cvc-datatype-valid" {
				t.Errorf("%s: Parse(%q): rule = %q (ok=%v), want cvc-datatype-valid", local, lex, rule, ok)
			}
		}
	}
}

// TestFloatingEqVsIdentity is the load-bearing distinction (§3.3.4.1/§2.2.2):
// Eq and Identical genuinely DISAGREE on NaN and signed zero. Each assertion
// bites via mutation — swapping Eq for Identical (or vice versa) flips a result.
func TestFloatingEqVsIdentity(t *testing.T) {
	for _, local := range []string{"float", "double"} {
		nan := parseVal(t, local, "NaN")
		nan2 := parseVal(t, local, "NaN")
		posZero := parseVal(t, local, "0")
		negZero := parseVal(t, local, "-0")
		one := parseVal(t, local, "1")

		eq := nan.(value.Eq)
		id := nan.(value.Identical)

		// Equality: NaN ≠ NaN, but +0 = -0.
		if eq.Eq(nan2) {
			t.Errorf("%s: Eq(NaN, NaN) = true, want false", local)
		}
		if !posZero.(value.Eq).Eq(negZero) {
			t.Errorf("%s: Eq(+0, -0) = false, want true", local)
		}
		if posZero.(value.Eq).Eq(one) {
			t.Errorf("%s: Eq(+0, 1) = true, want false", local)
		}

		// Identity: NaN identical to itself, but +0 NOT identical to -0.
		if !id.Identical(nan2) {
			t.Errorf("%s: Identical(NaN, NaN) = false, want true", local)
		}
		if posZero.(value.Identical).Identical(negZero) {
			t.Errorf("%s: Identical(+0, -0) = true, want false", local)
		}
		if !posZero.(value.Identical).Identical(parseVal(t, local, "0")) {
			t.Errorf("%s: Identical(+0, +0) = false, want true", local)
		}
	}
}

// TestFloatingPartialOrder proves the partial order (§3.3.4.3/§3.3.5.3): finite
// ordering, ±INF bounds, ±0 equal, and NaN INCOMPARABLE with everything including
// itself — the outcome the bounds-facet stage relies on.
func TestFloatingPartialOrder(t *testing.T) {
	for _, local := range []string{"float", "double"} {
		ord := func(a, b string) value.Ordering {
			return parseVal(t, local, a).(value.Ordered).Cmp(parseVal(t, local, b))
		}
		checks := []struct {
			a, b string
			want value.Ordering
		}{
			{"1", "2", value.Less},
			{"2", "1", value.Greater},
			{"1", "1", value.Equal},
			{"0", "-0", value.Equal},
			{"-INF", "1", value.Less},
			{"INF", "1", value.Greater},
			{"-INF", "INF", value.Less},
			{"NaN", "1", value.Incomparable},
			{"1", "NaN", value.Incomparable},
			{"NaN", "NaN", value.Incomparable},
			{"NaN", "INF", value.Incomparable},
		}
		for _, c := range checks {
			if got := ord(c.a, c.b); got != c.want {
				t.Errorf("%s: Cmp(%s, %s) = %v, want %v", local, c.a, c.b, got, c.want)
			}
		}
	}
}

// TestFloatingCrossType proves float and double are DISTINCT value spaces: a
// float compared/equated/identified against a double is Incomparable/false, so a
// mixed comparison never fabricates an order (rf-ordered).
func TestFloatingCrossType(t *testing.T) {
	f := parseVal(t, "float", "1")
	d := parseVal(t, "double", "1")
	if got := f.(value.Ordered).Cmp(d); got != value.Incomparable {
		t.Errorf("Cmp(float 1, double 1) = %v, want Incomparable", got)
	}
	if f.(value.Eq).Eq(d) {
		t.Error("Eq(float 1, double 1) = true, want false (distinct value spaces)")
	}
	if f.(value.Identical).Identical(d) {
		t.Error("Identical(float 1, double 1) = true, want false")
	}
}

// TestFloatingCapabilities pins the capability surface: float/double are
// Ordered/Eq/Identical/Canonical and deliberately NOT Lengthed/DigitCounted/
// Scaled (cos-applicable-facets §4.1.5).
func TestFloatingCapabilities(t *testing.T) {
	for _, local := range []string{"float", "double"} {
		v := parseVal(t, local, "1.5")
		if _, ok := v.(value.Ordered); !ok {
			t.Errorf("%s: value is not value.Ordered", local)
		}
		if _, ok := v.(value.Identical); !ok {
			t.Errorf("%s: value is not value.Identical", local)
		}
		if _, ok := v.(value.Canonical); !ok {
			t.Errorf("%s: value is not value.Canonical", local)
		}
		if _, ok := v.(value.Lengthed); ok {
			t.Errorf("%s: value implements value.Lengthed; it must not", local)
		}
		if _, ok := v.(value.DigitCounted); ok {
			t.Errorf("%s: value implements value.DigitCounted; it must not", local)
		}
		if _, ok := v.(value.Scaled); ok {
			t.Errorf("%s: value implements value.Scaled; it must not", local)
		}
	}
}

// TestFloatingCanonicalForeign proves the Canonical wrapper rejects a foreign
// value as an *xsderr.Error rather than panicking (warden guardrail), and does
// not accept a double where a float is expected (or vice versa).
func TestFloatingCanonicalForeign(t *testing.T) {
	floatM := mappingFor(t, "float")
	doubleM := mappingFor(t, "double")
	d := parseVal(t, "double", "1")
	f := parseVal(t, "float", "1")

	for _, c := range []struct {
		m   value.Mapping
		v   value.Value
		who string
	}{
		{floatM, 42, "float/int"},
		{floatM, d, "float/double"},
		{doubleM, "x", "double/string"},
		{doubleM, f, "double/float"},
	} {
		_, err := c.m.Canonical(c.v)
		if err == nil {
			t.Errorf("%s: Canonical(foreign) = nil error, want rejection", c.who)
			continue
		}
		if rule, ok := xsderr.RuleOf(err); !ok || rule != "cvc-datatype-valid" {
			t.Errorf("%s: Canonical(foreign): rule = %q (ok=%v), want cvc-datatype-valid", c.who, rule, ok)
		}
	}
}
