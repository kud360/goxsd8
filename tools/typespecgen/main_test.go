package main

import (
	"os"
	"testing"

	"github.com/kud360/goxsd8/tools/hfnextract/builtins"
)

const (
	testStructures = "../../docs/specs/md/xmlschema11-2.md"
	testPrecision  = "../../docs/specs/md/xsd-precisionDecimal.md"
	testCommitted  = "../../builtin/gen_typespec.go"
	testPrimitives = "../../value/gen_primitives.go"
)

func generate(t *testing.T) []byte {
	t.Helper()
	types, err := builtins.Parse(testStructures, testPrecision)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	src, err := emit(types)
	if err != nil {
		t.Fatalf("emit: %v", err)
	}
	return src
}

// TestDeterministic pins STYLE D1: two generation runs are byte-identical.
func TestDeterministic(t *testing.T) {
	first := generate(t)
	second := generate(t)
	if string(first) != string(second) {
		t.Fatal("emit is not byte-identical across runs")
	}
}

// TestCommittedUpToDate fails if builtin/gen_typespec.go has drifted from
// what the generator produces — i.e. someone edited it by hand or forgot to
// run `go generate ./...`.
func TestCommittedUpToDate(t *testing.T) {
	want, err := os.ReadFile(testCommitted)
	if err != nil {
		t.Fatalf("reading committed file: %v", err)
	}
	if string(generate(t)) != string(want) {
		t.Fatalf("%s is stale; run `go generate ./...`", testCommitted)
	}
}

// TestPrimitivesCommittedUpToDate is TestCommittedUpToDate for the -primitives
// output, value/gen_primitives.go.
func TestPrimitivesCommittedUpToDate(t *testing.T) {
	types, err := builtins.Parse(testStructures, testPrecision)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	got, err := emitPrimitives(types)
	if err != nil {
		t.Fatalf("emitPrimitives: %v", err)
	}
	want, err := os.ReadFile(testPrimitives)
	if err != nil {
		t.Fatalf("reading committed file: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("%s is stale; run `go generate ./...`", testPrimitives)
	}
}

// TestPrimitivesCount pins emitPrimitives' guard: a selection that misses a
// primitive (here precisionDecimal, dropped from the parse) is an error, not a
// short table.
func TestPrimitivesCount(t *testing.T) {
	types, err := builtins.Parse(testStructures, testPrecision)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, err := emitPrimitives(types[:len(types)-1]); err == nil {
		t.Fatal("emitPrimitives accepted 19 primitives, want an error")
	}
}
