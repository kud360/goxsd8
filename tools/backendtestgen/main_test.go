package main

import (
	"os"
	"testing"
)

const (
	testStructures = "../../docs/specs/md/xmlschema11-2.md"
	testPrecision  = "../../docs/specs/md/xsd-precisionDecimal.md"
	testCommitted  = "../../value/backendtest/gen_vectors.go"
)

func generate(t *testing.T) []byte {
	t.Helper()
	src, err := build(testStructures, testPrecision)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	return src
}

func readSpec(t *testing.T) string {
	t.Helper()
	content, err := os.ReadFile(testStructures)
	if err != nil {
		t.Fatalf("reading spec: %v", err)
	}
	return string(content)
}

// TestBooleanVectors pins the spec-derived boolean corpus: the valid
// round-trips and the near-miss invalids the generator must produce
// (§3.3.2.2, f-booleanLexmap/f-booleanCanmap, nt-booleanRep). It fails loudly
// if a spec edit or a parser change drifts the vectors.
func TestBooleanVectors(t *testing.T) {
	b, err := parseBoolean(readSpec(t))
	if err != nil {
		t.Fatalf("parseBoolean: %v", err)
	}

	wantValid := []roundtrip{
		{"true", "true"}, {"false", "false"}, {"1", "true"}, {"0", "false"},
	}
	if len(b.Valid) != len(wantValid) {
		t.Fatalf("valid: got %v, want %v", b.Valid, wantValid)
	}
	for i, w := range wantValid {
		if b.Valid[i] != w {
			t.Errorf("valid[%d]: got %v, want %v", i, b.Valid[i], w)
		}
	}

	wantInvalid := []string{"True", "TRUE", "False", "FALSE", "", "2"}
	if len(b.Invalid) != len(wantInvalid) {
		t.Fatalf("invalid: got %q, want %q", b.Invalid, wantInvalid)
	}
	for i, w := range wantInvalid {
		if b.Invalid[i] != w {
			t.Errorf("invalid[%d]: got %q, want %q", i, b.Invalid[i], w)
		}
	}
}

// TestDecimalVectors pins the spec-derived decimal corpus: the worked example
// lexicals of the production (decimal-lexical-representation) with the canonical
// forms f-decimalCanmap assigns (§3.3.3.2), and the regex-verified invalid
// near-misses. It fails loudly if a spec edit or a parser change drifts them.
func TestDecimalVectors(t *testing.T) {
	d, err := parseDecimal(readSpec(t))
	if err != nil {
		t.Fatalf("parseDecimal: %v", err)
	}

	wantValid := []roundtrip{
		{"-1.23", "-1.23"},
		{"12678967.543233", "12678967.543233"},
		{"+100000.00", "100000"}, // '+' dropped, trailing fractional zeros dropped
		{"210", "210"},
	}
	if len(d.Valid) != len(wantValid) {
		t.Fatalf("valid: got %v, want %v", d.Valid, wantValid)
	}
	for i, w := range wantValid {
		if d.Valid[i] != w {
			t.Errorf("valid[%d]: got %v, want %v", i, d.Valid[i], w)
		}
	}

	wantInvalid := []string{"-1.23E2", "+", ".", ""} // exponent, bare sign, bare point, empty
	if len(d.Invalid) != len(wantInvalid) {
		t.Fatalf("invalid: got %q, want %q", d.Invalid, wantInvalid)
	}
	for i, w := range wantInvalid {
		if d.Invalid[i] != w {
			t.Errorf("invalid[%d]: got %q, want %q", i, d.Invalid[i], w)
		}
	}
}

// TestStringVectors pins the string corpus: representative round-trips under the
// identity mapping (§3.3.1.2) and NO invalid lexicals (every Char* sequence is
// in the lexical space, nt-stringRep).
func TestStringVectors(t *testing.T) {
	s, err := parseString(readSpec(t))
	if err != nil {
		t.Fatalf("parseString: %v", err)
	}
	wantValid := []roundtrip{
		{"", ""}, {"abc", "abc"}, {"café", "café"}, {"𝔘nicode", "𝔘nicode"},
	}
	if len(s.Valid) != len(wantValid) {
		t.Fatalf("valid: got %v, want %v", s.Valid, wantValid)
	}
	for i, w := range wantValid {
		if s.Valid[i] != w {
			t.Errorf("valid[%d]: got %v, want %v", i, s.Valid[i], w)
		}
	}
	if len(s.Invalid) != 0 {
		t.Errorf("invalid: got %q, want none (every string is in the lexical space)", s.Invalid)
	}
}

// TestApplicableFacets pins that each cohort type carries its cos-applicable-facets
// list in spec order (§4.1.5), sourced from the shared builtin spec parser.
func TestApplicableFacets(t *testing.T) {
	facets, err := applicableFacets(testStructures, testPrecision)
	if err != nil {
		t.Fatalf("applicableFacets: %v", err)
	}
	want := map[string][]string{
		"boolean": {"whiteSpace", "pattern", "assertions"},
		"string":  {"whiteSpace", "length", "minLength", "maxLength", "pattern", "enumeration", "assertions"},
		"decimal": {"whiteSpace", "totalDigits", "fractionDigits", "pattern", "enumeration", "maxInclusive", "maxExclusive", "minInclusive", "minExclusive", "assertions"},
	}
	for name, w := range want {
		got := facets[name]
		if len(got) != len(w) {
			t.Fatalf("%s: got %q, want %q", name, got, w)
		}
		for i := range w {
			if got[i] != w[i] {
				t.Errorf("%s facet[%d]: got %q, want %q", name, i, got[i], w[i])
			}
		}
	}
}

// TestDeterministic pins STYLE D1: two generation runs are byte-identical.
func TestDeterministic(t *testing.T) {
	first := generate(t)
	second := generate(t)
	if string(first) != string(second) {
		t.Fatal("emit is not byte-identical across runs")
	}
}

// TestCommittedUpToDate fails if value/backendtest/gen_vectors.go has drifted
// from what the generator produces — a hand edit or a missed `go generate`.
func TestCommittedUpToDate(t *testing.T) {
	want, err := os.ReadFile(testCommitted)
	if err != nil {
		t.Fatalf("reading committed file: %v", err)
	}
	if string(generate(t)) != string(want) {
		t.Fatalf("%s is stale; run `go generate ./...`", testCommitted)
	}
}
