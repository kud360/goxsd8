package strict_test

import (
	"testing"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsderr"
)

func TestHexBinaryParseCanonicalAndLength(t *testing.T) {
	m := mappingFor(t, "hexBinary")
	// hexBinaryMap accepts lowercase a–f; f-hexBinaryCanonical uppercases (E.4.1).
	// Len is the OCTET count (§4.3.1.3 clause 1.2), i.e. half the hex-digit count,
	// not the lexical character count.
	cases := []struct {
		lex   string
		canon string
		octet int
	}{
		{"", "", 0},
		{"0FB7", "0FB7", 2},
		{"0fb7", "0FB7", 2},
		{"deadBEEF", "DEADBEEF", 4},
		{"ff", "FF", 1},
	}
	for _, c := range cases {
		v, err := m.Parse(c.lex, nil)
		if err != nil {
			t.Errorf("Parse(%q): unexpected error %v", c.lex, err)
			continue
		}
		got, err := m.Canonical(v)
		if err != nil {
			t.Errorf("Canonical(%q): unexpected error %v", c.lex, err)
			continue
		}
		if got != c.canon {
			t.Errorf("Canonical(Parse(%q)) = %q, want %q", c.lex, got, c.canon)
		}
		l, ok := v.(value.Lengthed)
		if !ok {
			t.Fatalf("hexBinary value %q does not implement value.Lengthed", c.lex)
		}
		if l.Len() != c.octet {
			t.Errorf("Len(%q) = %d octets, want %d", c.lex, l.Len(), c.octet)
		}
	}
}

func TestHexBinaryReject(t *testing.T) {
	m := mappingFor(t, "hexBinary")
	// Odd-length and non-hex lexicals are outside the space (nt-hexBinary,
	// §3.3.15.2); a whitespace-bearing literal never reaches Parse pre-collapsed
	// but must still be rejected by the anchored production.
	for _, lex := range []string{"F", "0FB", "0G", "gg", "0F B7", " 0F"} {
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

func TestBase64BinaryParseCanonicalAndLength(t *testing.T) {
	m := mappingFor(t, "base64Binary")
	// Len is the decoded OCTET count (§4.3.1.3 clause 1.2), never the base64
	// character count; canonical is the whitespace-free encoding (§3.3.16.2).
	cases := []struct {
		lex   string
		canon string
		octet int
	}{
		{"", "", 0},
		{"AQID", "AQID", 3},
		{"AQI=", "AQI=", 2},
		{"AQ==", "AQ==", 1},
		{"TWFu", "TWFu", 3},
		{"AQIDBA==", "AQIDBA==", 4},
	}
	for _, c := range cases {
		v, err := m.Parse(c.lex, nil)
		if err != nil {
			t.Errorf("Parse(%q): unexpected error %v", c.lex, err)
			continue
		}
		got, err := m.Canonical(v)
		if err != nil {
			t.Errorf("Canonical(%q): unexpected error %v", c.lex, err)
			continue
		}
		if got != c.canon {
			t.Errorf("Canonical(Parse(%q)) = %q, want %q", c.lex, got, c.canon)
		}
		l, ok := v.(value.Lengthed)
		if !ok {
			t.Fatalf("base64Binary value %q does not implement value.Lengthed", c.lex)
		}
		if l.Len() != c.octet {
			t.Errorf("Len(%q) = %d octets, want %d", c.lex, l.Len(), c.octet)
		}
	}
}

func TestBase64BinaryReject(t *testing.T) {
	m := mappingFor(t, "base64Binary")
	// The grammar rejects a non-multiple-of-four count ("AQI"), misplaced padding
	// ("A===", "AB=C") and — the constraint a naive decoder misses — a final
	// character outside the restricted B16char/B04char subset for its padding
	// width: "AQJ=" (single '=' needs B16char) and "AB==" (double '=' needs
	// B04char) both discard non-zero bits (§3.3.16.2).
	for _, lex := range []string{"AQI", "A===", "AB=C", "AQJ=", "AB==", "A", "==="} {
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

func TestBinaryEqNotOrdered(t *testing.T) {
	for _, local := range []string{"hexBinary", "base64Binary"} {
		m := mappingFor(t, local)
		var a, a2, b value.Value
		switch local {
		case "hexBinary":
			a, _ = m.Parse("0FB7", nil)
			a2, _ = m.Parse("0fb7", nil) // same octets, case-insensitive input
			b, _ = m.Parse("0FB8", nil)
		case "base64Binary":
			a, _ = m.Parse("AQID", nil)
			a2, _ = m.Parse("AQID", nil)
			b, _ = m.Parse("AQIE", nil)
		}
		eq, ok := a.(value.Eq)
		if !ok {
			t.Fatalf("%s value does not implement value.Eq", local)
		}
		if !eq.Eq(a2) {
			t.Errorf("%s: Eq of equal octet sequences = false, want true", local)
		}
		if eq.Eq(b) {
			t.Errorf("%s: Eq of distinct octet sequences = true, want false", local)
		}
		if eq.Eq(42) {
			t.Errorf("%s: Eq(binary, int) = true, want false", local)
		}
		// ordered=false (§3.3.15/§3.3.16): neither type may be value.Ordered.
		if _, ok := a.(value.Ordered); ok {
			t.Errorf("%s value implements value.Ordered; it must not", local)
		}
	}
}

func TestBinaryCanonicalForeign(t *testing.T) {
	for _, local := range []string{"hexBinary", "base64Binary"} {
		m := mappingFor(t, local)
		_, err := m.Canonical(42)
		if err == nil {
			t.Fatalf("%s Canonical(foreign): want error, got nil", local)
		}
		if rule, ok := xsderr.RuleOf(err); !ok || rule != "cvc-datatype-valid" {
			t.Errorf("%s Canonical(foreign): rule = %q (ok=%v), want cvc-datatype-valid", local, rule, ok)
		}
	}
}
