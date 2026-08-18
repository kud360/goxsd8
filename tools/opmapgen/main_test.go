package main

import (
	"os"
	"strings"
	"testing"
)

const (
	testXPath     = "../../docs/specs/md/xpath20.md"
	testCommitted = "../../xpath/gen_opmap.go"
)

func generate(t *testing.T) []byte {
	t.Helper()
	text, err := os.ReadFile(testXPath)
	if err != nil {
		t.Fatalf("reading %s: %v", testXPath, err)
	}
	rows, err := parse(strings.Split(string(text), "\n"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	src, err := emit(rows)
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

// TestCommittedUpToDate fails if xpath/gen_opmap.go has drifted from what the
// generator produces — i.e. someone edited it by hand or forgot to run
// `go generate ./...`.
func TestCommittedUpToDate(t *testing.T) {
	want, err := os.ReadFile(testCommitted)
	if err != nil {
		t.Fatalf("reading committed file: %v", err)
	}
	if string(generate(t)) != string(want) {
		t.Fatalf("%s is stale; run `go generate ./...`", testCommitted)
	}
}

// TestMacrosExpand pins both macro expansions against B.2's own text, since a
// macro read wrongly would silently admit or deny whole families of types:
// numeric names four types and pairs them freely, Gregorian names five and
// pairs each only with itself.
func TestMacrosExpand(t *testing.T) {
	rows := parseSpec(t)
	numeric := map[string]bool{"integer": true, "decimal": true, "float": true, "double": true}
	gregorian := map[string]bool{"gYearMonth": true, "gYear": true, "gMonthDay": true, "gDay": true, "gMonth": true}
	var numericPairs, gregorianRows int
	for _, r := range rows {
		if r.op == "ctaEqual" && numeric[r.a] && numeric[r.b] {
			numericPairs++
		}
		if !gregorian[r.a] {
			continue
		}
		gregorianRows++
		if r.a != r.b {
			t.Errorf("Gregorian row %+v pairs two different types", r)
		}
	}
	if numericPairs != 16 {
		t.Errorf("numeric eq rows = %d, want 16 (four types, every combination)", numericPairs)
	}
	// Five types, under eq and ne alone.
	if gregorianRows != 10 {
		t.Errorf("Gregorian rows = %d, want 10", gregorianRows)
	}
}

// TestEqOnlyTypes pins the six type pairs B.2 gives eq and ne rows and no
// ordering rows, which is the whole reason this table is generated: reading
// them off the emitted rows is what the xpath package's legality check turns
// on.
func TestEqOnlyTypes(t *testing.T) {
	rows := parseSpec(t)
	ordered := make(map[string]bool)
	compared := make(map[string]bool)
	for _, r := range rows {
		if r.a != r.b {
			continue
		}
		compared[r.a] = true
		if r.op != "ctaEqual" && r.op != "ctaNotEqual" {
			ordered[r.a] = true
		}
	}
	for _, local := range []string{"duration", "gYearMonth", "gYear", "gMonthDay", "gDay", "gMonth", "hexBinary", "base64Binary", "QName", "NOTATION"} {
		if !compared[local] {
			t.Errorf("xs:%s has no equality row", local)
		}
		if ordered[local] {
			t.Errorf("xs:%s has an ordering row", local)
		}
	}
	for _, local := range []string{"boolean", "string", "date", "time", "dateTime", "anyURI", "yearMonthDuration", "dayTimeDuration"} {
		if !ordered[local] {
			t.Errorf("xs:%s has no ordering row", local)
		}
	}
}

func parseSpec(t *testing.T) []row {
	t.Helper()
	text, err := os.ReadFile(testXPath)
	if err != nil {
		t.Fatalf("reading %s: %v", testXPath, err)
	}
	rows, err := parse(strings.Split(string(text), "\n"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return rows
}

// TestParseFailsLoudly pins that a table shape the generator does not
// recognize stops it, rather than being skipped into a short table: a spec
// re-render that renames a macro, drops a column or rewrites an operator cell
// must fail the build.
func TestParseFailsLoudly(t *testing.T) {
	for _, tc := range []struct {
		name  string
		lines []string
	}{
		{"no B.2 heading", []string{"## B.1 Type Promotion"}},
		{"no macro definition", []string{`### <a id="mapping"></a>B.2 Operator Mapping`}},
		{"no operator mapping table", append([]string{`### <a id="mapping"></a>B.2 Operator Mapping`}, macroLines()...)},
		{"row with the wrong column count", table("| A eq B | xs:string | xs:string |")},
		{"operator cell that is not A <op> B", table("| eq | xs:string | xs:string | op:x(A, B) | xs:boolean |")},
		{"type cell naming an unknown macro", table("| A eq B | temporal | temporal | op:x(A, B) | xs:boolean |")},
		{"type cell outside the XSD namespace", table("| A eq B | my:date | my:date | op:x(A, B) | xs:boolean |")},
		{"table holding no comparison row", table("| A + B | xs:date | xs:date | op:x(A, B) | xs:date |")},
	} {
		if _, err := parse(tc.lines); err == nil {
			t.Errorf("%s: parse succeeded, want an error", tc.name)
		}
	}
}

// macroLines are the two Definition spans every well-formed input carries, in
// the form B.2 writes them.
func macroLines() []string {
	return []string{
		"[<a id=\"dt-numeric\"></a>Definition: When referring to a type, the term **numeric**denotes the types `xs:integer`, `xs:decimal`, `xs:float`, and `xs:double`.] More prose.",
		"[<a id=\"dt-gregorian\"></a>Definition: the term **Gregorian**refers to the types `xs:gYearMonth`, `xs:gYear`, `xs:gMonthDay`, `xs:gDay`, and `xs:gMonth`.] For binary operators, both operands must have the same type.",
	}
}

// table is a complete minimal B.2 section holding one row.
func table(row string) []string {
	lines := []string{`### <a id="mapping"></a>B.2 Operator Mapping`}
	lines = append(lines, macroLines()...)
	return append(lines, tableHeader, "| --- | --- | --- | --- | --- |", row, "")
}
