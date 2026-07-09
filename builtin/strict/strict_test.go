package strict_test

import (
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// xsdNS is the XML Schema namespace the cohort types live in.
const xsdNS = "http://www.w3.org/2001/XMLSchema"

func TestBackendCoverage(t *testing.T) {
	backend := strict.New()
	for _, local := range []string{"decimal", "boolean", "string"} {
		m, ok := backend.Mapping(xsd.QName{Space: xsdNS, Local: local})
		if !ok {
			t.Errorf("Mapping(xs:%s): ok=false, want true", local)
			continue
		}
		if m.Parse == nil {
			t.Errorf("Mapping(xs:%s): Parse is nil", local)
		}
		if m.Canonical == nil {
			t.Errorf("Mapping(xs:%s): Canonical is nil (all three define a canonical map)", local)
		}
	}
}

func TestBackendUnmapped(t *testing.T) {
	backend := strict.New()
	// A type outside the cohort, and a same-local name in the wrong namespace,
	// are both unmapped.
	for _, q := range []xsd.QName{
		{Space: xsdNS, Local: "integer"},
		{Space: xsdNS, Local: "float"},
		{Space: "urn:other", Local: "decimal"},
		{Local: "decimal"},
	} {
		if _, ok := backend.Mapping(q); ok {
			t.Errorf("Mapping(%s): ok=true, want false", q)
		}
	}
}

func mappingFor(t *testing.T, local string) value.Mapping {
	t.Helper()
	m, ok := strict.New().Mapping(xsd.QName{Space: xsdNS, Local: local})
	if !ok {
		t.Fatalf("strict backend does not map xs:%s", local)
	}
	return m
}

func TestBooleanParseAndCanonical(t *testing.T) {
	m := mappingFor(t, "boolean")
	// f-booleanLexmap / f-booleanCanmap (§3.3.2.2): 1→true, 0→false, canonical
	// is always the word form.
	cases := map[string]string{
		"true":  "true",
		"false": "false",
		"1":     "true",
		"0":     "false",
	}
	for lex, wantCanon := range cases {
		v, err := m.Parse(lex, nil)
		if err != nil {
			t.Errorf("Parse(%q): unexpected error %v", lex, err)
			continue
		}
		got, err := m.Canonical(v)
		if err != nil {
			t.Errorf("Canonical(%q): unexpected error %v", lex, err)
			continue
		}
		if got != wantCanon {
			t.Errorf("Canonical(Parse(%q)) = %q, want %q", lex, got, wantCanon)
		}
	}
}

func TestBooleanReject(t *testing.T) {
	m := mappingFor(t, "boolean")
	// Exactly the four literals — no case or whitespace variants
	// (boolean-lexical-mapping, §3.3.2.1).
	for _, lex := range []string{"True", "FALSE", "TRUE", "yes", "no", " true", "true ", "2", "", "01"} {
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

func TestBooleanEqIdenticalNotOrdered(t *testing.T) {
	m := mappingFor(t, "boolean")
	tru, _ := m.Parse("1", nil)
	tru2, _ := m.Parse("true", nil)
	fls, _ := m.Parse("0", nil)

	eq, ok := tru.(value.Eq)
	if !ok {
		t.Fatal("boolean value does not implement value.Eq")
	}
	if !eq.Eq(tru2) {
		t.Error("Eq(true, true) = false, want true")
	}
	if eq.Eq(fls) {
		t.Error("Eq(true, false) = true, want false")
	}
	if eq.Eq("true") {
		t.Error("Eq(boolean, string) = true, want false")
	}

	id, ok := tru.(value.Identical)
	if !ok {
		t.Fatal("boolean value does not implement value.Identical")
	}
	if !id.Identical(tru2) || id.Identical(fls) {
		t.Error("Identical must coincide with Eq for boolean")
	}

	// ordered=false (§3.3.2.3): boolean must NOT be value.Ordered.
	if _, ok := tru.(value.Ordered); ok {
		t.Error("boolean value implements value.Ordered; it must not")
	}
}

func TestBooleanCanonicalForeign(t *testing.T) {
	m := mappingFor(t, "boolean")
	_, err := m.Canonical(42)
	if err == nil {
		t.Fatal("Canonical(foreign): want error, got nil")
	}
	if rule, ok := xsderr.RuleOf(err); !ok || rule != "cvc-datatype-valid" {
		t.Errorf("Canonical(foreign): rule = %q (ok=%v), want cvc-datatype-valid", rule, ok)
	}
}

func TestStringIdentityAndLength(t *testing.T) {
	m := mappingFor(t, "string")
	// f-stringLexmap / f-stringCanmap are the identity (§3.3.1.2): every string
	// is accepted verbatim, including whitespace (whiteSpace=preserve).
	for _, s := range []string{"", "hello", " padded ", "a\tb\nc", "café", "𝔘nicode"} {
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
		"hello":   5,
		"café":    4, // é is one codepoint but two UTF-8 bytes
		"𝔘nicode": 7, // 𝔘 is one codepoint but four UTF-8 bytes
	}
	for s, want := range cases {
		v, _ := m.Parse(s, nil)
		l, ok := v.(value.Lengthed)
		if !ok {
			t.Fatalf("string value %q does not implement value.Lengthed", s)
		}
		if got := l.Len(); got != want {
			t.Errorf("Len(%q) = %d, want %d", s, got, want)
		}
	}
}

func TestStringEqNotOrdered(t *testing.T) {
	m := mappingFor(t, "string")
	a, _ := m.Parse("abc", nil)
	a2, _ := m.Parse("abc", nil)
	b, _ := m.Parse("abd", nil)

	eq := a.(value.Eq)
	if !eq.Eq(a2) {
		t.Error(`Eq("abc", "abc") = false, want true`)
	}
	if eq.Eq(b) {
		t.Error(`Eq("abc", "abd") = true, want false`)
	}
	if eq.Eq(42) {
		t.Error("Eq(string, int) = true, want false")
	}
	// ordered=false (§3.3.1.3): string must NOT be value.Ordered.
	if _, ok := a.(value.Ordered); ok {
		t.Error("string value implements value.Ordered; it must not")
	}
}
