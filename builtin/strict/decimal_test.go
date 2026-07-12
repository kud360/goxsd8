package strict_test

import (
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

func decimalMapping(t *testing.T) value.Mapping {
	t.Helper()
	m, ok := strict.New().Mapping(xsd.QName{Space: xsd.XMLSchemaNS, Local: "decimal"})
	if !ok {
		t.Fatal("strict backend does not map xs:decimal")
	}
	return m
}

func parseDec(t *testing.T, m value.Mapping, lexical string) value.Value {
	t.Helper()
	v, err := m.Parse(lexical, nil)
	if err != nil {
		t.Fatalf("Parse(%q): unexpected error %v", lexical, err)
	}
	return v
}

func TestDecimalCanonical(t *testing.T) {
	m := decimalMapping(t)
	// f-decimalCanmap examples (§3.3.3.2).
	cases := map[string]string{
		"+1.0":   "1",
		"010.20": "10.2",
		"0":      "0",
		"-0":     "0",
		"0.0":    "0",
		"5.":     "5",
		".5":     "0.5",
		"0.05":   "0.05",
		"-1.230": "-1.23",
		"100":    "100",
		"1.00":   "1",
	}
	for lex, want := range cases {
		v := parseDec(t, m, lex)
		got, err := m.Canonical(v)
		if err != nil {
			t.Errorf("Canonical(%q): unexpected error %v", lex, err)
			continue
		}
		if got != want {
			t.Errorf("Canonical(%q) = %q, want %q", lex, got, want)
		}
	}
}

func TestDecimalReject(t *testing.T) {
	m := decimalMapping(t)
	// decimal has no exponent production; whitespace and bare "." are out of
	// the lexical space (decimal-lexical-representation, §3.3.3.1).
	for _, lex := range []string{"1E2", "1e2", ".", "", "+", "-", " 1", "1 ", "1.2.3", "abc", "0x1", "+-1"} {
		_, err := m.Parse(lex, nil)
		if err == nil {
			t.Errorf("Parse(%q): want lexical-space error, got nil", lex)
			continue
		}
		if rule, ok := xsderr.RuleOf(err); !ok || rule != "cvc-datatype-valid" {
			t.Errorf("Parse(%q): rule = %q (ok=%v), want cvc-datatype-valid", lex, rule, ok)
		}
	}
}

func TestDecimalOrderAndEq(t *testing.T) {
	m := decimalMapping(t)
	// Total order (§3.3.3.2): value comparison ignores lexical scale.
	one := parseDec(t, m, "1.0").(value.Ordered)
	same := parseDec(t, m, "1.00")
	bigger := parseDec(t, m, "10")
	smaller := parseDec(t, m, "-3.5")

	if got := one.Cmp(same); got != value.Equal {
		t.Errorf("Cmp(1.0, 1.00) = %v, want Equal", got)
	}
	if got := one.Cmp(bigger); got != value.Less {
		t.Errorf("Cmp(1.0, 10) = %v, want Less", got)
	}
	if got := one.Cmp(smaller); got != value.Greater {
		t.Errorf("Cmp(1.0, -3.5) = %v, want Greater", got)
	}
	if !one.Eq(same) {
		t.Error("Eq(1.0, 1.00) = false, want true")
	}
	if one.Eq(bigger) {
		t.Error("Eq(1.0, 10) = true, want false")
	}
	// A foreign value is Incomparable / unequal, not a panic.
	if got := one.Cmp("not a decimal"); got != value.Incomparable {
		t.Errorf("Cmp(decimal, string) = %v, want Incomparable", got)
	}
	if one.Eq(42) {
		t.Error("Eq(decimal, int) = true, want false")
	}
}

func TestDecimalDigitCounts(t *testing.T) {
	m := decimalMapping(t)
	// VALUE-based digit counts (rf-totalDigits/rf-fractionDigits): counted on
	// the minimal form, not the raw lexical.
	cases := []struct {
		lex             string
		total, fraction int
	}{
		{"010.20", 3, 1}, // minimal 102, scale 1
		{"1.00", 1, 0},   // minimal 1, scale 0
		{"0", 1, 0},      // zero convention
		{"0.05", 1, 2},   // minimal 5, scale 2
		{"-1.230", 3, 2}, // minimal -123, scale 2
		{"100", 3, 0},    // integer trailing zeros are significant
	}
	for _, c := range cases {
		v := parseDec(t, m, c.lex).(value.DigitCounted)
		if got := v.TotalDigits(); got != c.total {
			t.Errorf("TotalDigits(%q) = %d, want %d", c.lex, got, c.total)
		}
		if got := v.FractionDigits(); got != c.fraction {
			t.Errorf("FractionDigits(%q) = %d, want %d", c.lex, got, c.fraction)
		}
	}
}

func TestDecimalNotScaled(t *testing.T) {
	m := decimalMapping(t)
	// decimal collapses precision, so it must NOT be Scaled (unlike
	// precisionDecimal); the capability assertion must fail.
	v := parseDec(t, m, "1.0")
	if _, ok := v.(value.Scaled); ok {
		t.Error("decimal value implements value.Scaled; it must not")
	}
}

func TestDecimalCanonicalForeign(t *testing.T) {
	m := decimalMapping(t)
	_, err := m.Canonical("not a decimal")
	if err == nil {
		t.Fatal("Canonical(foreign): want error, got nil")
	}
	if rule, ok := xsderr.RuleOf(err); !ok || rule != "cvc-datatype-valid" {
		t.Errorf("Canonical(foreign): rule = %q (ok=%v), want cvc-datatype-valid", rule, ok)
	}
}
