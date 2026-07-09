package main

import (
	"os"
	"testing"
)

const (
	testStructures = "../../docs/specs/md/xmlschema11-2.md"
	testCommitted  = "../../value/backendtest/gen_vectors.go"
)

func generate(t *testing.T) []byte {
	t.Helper()
	content, err := os.ReadFile(testStructures)
	if err != nil {
		t.Fatalf("reading spec: %v", err)
	}
	boolean, err := parseBoolean(string(content))
	if err != nil {
		t.Fatalf("parseBoolean: %v", err)
	}
	src, err := emit([]typeVectors{boolean})
	if err != nil {
		t.Fatalf("emit: %v", err)
	}
	return src
}

// TestBooleanVectors pins the spec-derived boolean corpus: the valid
// round-trips and the near-miss invalids the generator must produce
// (§3.3.2.2, f-booleanLexmap/f-booleanCanmap, nt-booleanRep). It fails loudly
// if a spec edit or a parser change drifts the vectors.
func TestBooleanVectors(t *testing.T) {
	content, err := os.ReadFile(testStructures)
	if err != nil {
		t.Fatalf("reading spec: %v", err)
	}
	b, err := parseBoolean(string(content))
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
