package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// schemaDoc wraps body in an XSD schema element binding the xs prefix.
func schemaDoc(body string) string {
	return `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + body + `
</xs:schema>
`
}

// Each fixture body is one top-level element. The declining ones each name
// the construct their first decline is.
const (
	generates = `<xs:element name="a" type="xs:string"/>`
	// unassembled references a type the schema does not declare
	// (src-resolve), so parser.Parse fails.
	unassembled  = `<xs:element name="a" type="undeclared"/>`
	mixedContent = `<xs:element name="a"><xs:complexType mixed="true"><xs:sequence>` +
		`<xs:element name="b" type="xs:string"/></xs:sequence></xs:complexType></xs:element>`
	unionType = `<xs:element name="a"><xs:simpleType><xs:union memberTypes="xs:string xs:int"/>` +
		`</xs:simpleType></xs:element>`
	nillable = `<xs:element name="a" type="xs:string" nillable="true"/>`
)

// writeTree writes files, slash paths relative to a fresh directory, and
// returns that directory.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("creating %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}
	return root
}

// censusTree is the fixture tree. In path order its declines are first seen
// as mixed content (once), union type, then element {nillable} (twice each),
// so the first-seen order, the count order with a first-seen tie-break, and
// the count order with the String tie-break the tool owes are three different
// orders. The non-.xsd file and the dot directory are not documents.
func censusTree(t *testing.T) string {
	return writeTree(t, map[string]string{
		"a.xsd":       schemaDoc(generates),
		"b.xsd":       schemaDoc(mixedContent),
		"c.xsd":       schemaDoc(unionType),
		"d.xsd":       schemaDoc(unassembled),
		"e.xsd":       schemaDoc(nillable),
		"f.xsd":       schemaDoc(unionType),
		"g.xsd":       schemaDoc(nillable),
		"notes.txt":   "not a schema",
		".git/x.xsd":  schemaDoc(nillable),
		"sub/h.xsd":   schemaDoc(generates),
		"sub/i.xsd":   schemaDoc(`<xs:include schemaLocation="j.xsd"/>`),
		"sub/j.xsd":   schemaDoc(generates),
		"sub/k/l.txt": "",
	})
}

// wantCounts is the report's head over censusTree: every count, and each
// construct line, largest first and then by String.
const wantCounts = `gencensus: 10 schema document(s) under %ROOT%
  assembled: 9
  generated: 4
  declined:  5
  Generate error other than a decline: 0

=== First decline, by construct: 3 construct(s) ===
       2  element {nillable}
       2  union type
       1  mixed content
`

// TestCensusCountsAndRanksDeclines pins every count the report prints and its
// construct lines' order. Dropping ranked's sort prints the first-seen order
// (mixed content, union type, element {nillable}); dropping its String
// tie-break prints union type ahead of element {nillable}. Each fails here.
func TestCensusCountsAndRanksDeclines(t *testing.T) {
	root := censusTree(t)
	var out bytes.Buffer
	if err := run(&out, []string{root}); err != nil {
		t.Fatalf("run: %v", err)
	}
	want := strings.ReplaceAll(wantCounts, "%ROOT%", root)
	if !strings.HasPrefix(out.String(), want) {
		t.Errorf("report opens\n%s\nwant\n%s", out.String(), want)
	}
	if strings.Contains(out.String(), "=== Documents, by construct ===") {
		t.Errorf("report without -paths lists documents:\n%s", out.String())
	}
	for _, c := range caveats {
		if !strings.Contains(out.String(), "  - "+c+"\n") {
			t.Errorf("report lacks the caveat %q", c)
		}
	}
}

// TestCensusPathsListsEachConstructsDocuments pins -paths: each construct's
// documents in path order, under the constructs in rank order.
func TestCensusPathsListsEachConstructsDocuments(t *testing.T) {
	root := censusTree(t)
	var out bytes.Buffer
	if err := run(&out, []string{"-paths", root}); err != nil {
		t.Fatalf("run: %v", err)
	}
	want := `
=== Documents, by construct ===
element {nillable} (2):
  e.xsd
  g.xsd
union type (2):
  c.xsd
  f.xsd
mixed content (1):
  b.xsd

=== Generate errors other than a decline: 0 ===
`
	if !strings.HasSuffix(out.String(), want) {
		t.Errorf("report closes\n%s\nwant\n%s", out.String(), want)
	}
}

// TestCensusAbsentRootIsAMode holds an absent tree, and one with no .xsd
// document, to the supported mode: a report saying so, and no error.
func TestCensusAbsentRootIsAMode(t *testing.T) {
	cases := []struct{ name, root, reason string }{
		{"absent", filepath.Join(t.TempDir(), "absent"), "does not exist"},
		{"no document", writeTree(t, map[string]string{"notes.txt": ""}), "holds no .xsd document"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := run(&out, []string{tc.root}); err != nil {
				t.Fatalf("run: %v", err)
			}
			if want := "gencensus: " + tc.root + " " + tc.reason + " — corpus-absent mode"; !strings.HasPrefix(out.String(), want) {
				t.Errorf("report = %q, want it to open %q", out.String(), want)
			}
		})
	}
}
