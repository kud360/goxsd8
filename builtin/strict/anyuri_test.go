package strict_test

import (
	"testing"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsderr"
)

func TestAnyURIIdentityAndLength(t *testing.T) {
	m := mappingFor(t, "anyURI")
	// The lexical mapping is the identity (§3.3.17.2) over every Char* sequence;
	// no URI-syntax validation (§3.3.17.2: the spec disclaims RFC 3986/3987
	// checking), so Parse accepts anything verbatim — including strings no URI
	// parser would accept, matching xs:string's permissiveness.
	for _, s := range []string{
		"", "http://example.com/", " relative/path ", "not a uri at all",
		"%zz-not-valid-percent-encoding", "café", "𝔘nicode",
	} {
		v, err := m.Parse(s, nil)
		if err != nil {
			t.Errorf("Parse(%q): unexpected error %v", s, err)
			continue
		}
		got, err := m.Canonical(v)
		if err != nil {
			t.Errorf("Canonical(%q): unexpected error %v", s, err)
			continue
		}
		if got != s {
			t.Errorf("Canonical(Parse(%q)) = %q, want identity", s, got)
		}
	}

	// Len is character (codepoint) count, not bytes (§4.3.1–3).
	cases := map[string]int{
		"":        0,
		"http://": 7,
		"café":    4, // é is one codepoint but two UTF-8 bytes
		"𝔘nicode": 7, // 𝔘 is one codepoint but four UTF-8 bytes
	}
	for s, want := range cases {
		v, _ := m.Parse(s, nil)
		l, ok := v.(value.Lengthed)
		if !ok {
			t.Fatalf("anyURI value %q does not implement value.Lengthed", s)
		}
		if got := l.Len(); got != want {
			t.Errorf("Len(%q) = %d, want %d", s, got, want)
		}
	}
}

func TestAnyURIEqNotOrdered(t *testing.T) {
	m := mappingFor(t, "anyURI")
	// §3.3.17.2 Note: distinct character sequences are distinct values, even if
	// they denote "equivalent" URIs (trailing slash, percent-encoding case, …).
	a, _ := m.Parse("http://example.com", nil)
	a2, _ := m.Parse("http://example.com", nil)
	b, _ := m.Parse("http://example.com/", nil)

	eq := a.(value.Eq)
	if !eq.Eq(a2) {
		t.Error(`Eq("http://example.com", "http://example.com") = false, want true`)
	}
	if eq.Eq(b) {
		t.Error(`Eq must not treat trailing-slash-equivalent URIs as equal (§3.3.17.2 Note)`)
	}
	if eq.Eq(42) {
		t.Error("Eq(anyURI, int) = true, want false")
	}
	// ordered=false (§3.3.17.3): anyURI must NOT be value.Ordered.
	if _, ok := a.(value.Ordered); ok {
		t.Error("anyURI value implements value.Ordered; it must not")
	}
}

func TestAnyURICanonicalForeign(t *testing.T) {
	m := mappingFor(t, "anyURI")
	_, err := m.Canonical(42)
	if err == nil {
		t.Fatal("Canonical(foreign): want error, got nil")
	}
	if rule, ok := xsderr.RuleOf(err); !ok || rule != "cvc-datatype-valid" {
		t.Errorf("Canonical(foreign): rule = %q (ok=%v), want cvc-datatype-valid", rule, ok)
	}
}
