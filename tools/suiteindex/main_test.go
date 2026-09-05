package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/kud360/goxsd8/xsd"
)

// utf16LEDoc encodes doc as UTF-16LE behind the byte-order mark XML 1.0
// §4.3.3 requires of a UTF-16 entity — the encoding 93 of the suite's
// fixtures ship in, and the one a UTF-8 grep reads as interleaved NULs
// (#1239). parser/xmltree's own tests hold an equivalent helper, but a test
// file's helpers are private to its package, so this is a second spelling
// rather than a second decoder.
func utf16LEDoc(doc string) string {
	var b []byte
	for _, u := range utf16.Encode([]rune("\uFEFF" + doc)) {
		b = append(b, byte(u), byte(u>>8))
	}
	return string(b)
}

// The three spellings of one construct the suite actually writes. Every
// fixture below declares the same element — an XSD `element` carrying
// `targetNamespace` — and no two agree on how to spell it.
const (
	unprefixedDoc = `<?xml version="1.0"?>
<schema xmlns="http://www.w3.org/2001/XMLSchema">
  <complexType name="ct">
    <sequence>
      <element name="a" targetNamespace="urn:b"/>
    </sequence>
  </complexType>
</schema>`

	xsPrefixDoc = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="ct">
    <xs:sequence>
      <xs:element name="a" targetNamespace="urn:b"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`

	xsdPrefixDoc = `<?xml version="1.0"?>
<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <xsd:complexType name="ct">
    <xsd:sequence>
      <xsd:element name="a" targetNamespace="urn:b"/>
    </xsd:sequence>
  </xsd:complexType>
</xsd:schema>`

	// impostorDoc spells the prefix "xsd" over a namespace that is not the
	// XML Schema one. Its element is not the construct, however it reads.
	impostorDoc = `<?xml version="1.0"?>
<xsd:schema xmlns:xsd="urn:not-the-schema-namespace">
  <xsd:element name="a" targetNamespace="urn:b"/>
</xsd:schema>`
)

// mustQuery parses a query or fails the test; the queries below are all
// literals this package's own parser must accept.
func mustQuery(t *testing.T, s string) query {
	t.Helper()
	q, err := parseQuery(s)
	if err != nil {
		t.Fatalf("parseQuery(%q): %v", s, err)
	}
	return q
}

// TestScanFixtureMatchesEveryPrefixSpelling pins the tool's central claim: a
// query names a construct, so all three prefix spellings of one XSD element
// answer it and a same-spelled element in another namespace does not. A
// regression to prefix-text matching fails here.
func TestScanFixtureMatchesEveryPrefixSpelling(t *testing.T) {
	q := mustQuery(t, "element@targetNamespace")
	cases := []struct {
		name string
		doc  string
		want int
	}{
		{"unprefixed under a default binding", unprefixedDoc, 1},
		{"xs prefix", xsPrefixDoc, 1},
		{"xsd prefix", xsdPrefixDoc, 1},
		{"xsd prefix over another namespace", impostorDoc, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			scan := scanFixture("t.xsd", strings.NewReader(tc.doc), q)
			if scan.Err != nil {
				t.Fatalf("scanFixture: %v", scan.Err)
			}
			if len(scan.Hits) != tc.want {
				t.Fatalf("got %d hit(s), want %d: %+v", len(scan.Hits), tc.want, scan.Hits)
			}
			if tc.want == 0 {
				return
			}
			if got := scan.Hits[0].Values[0]; got != "urn:b" {
				t.Errorf("targetNamespace = %q, want urn:b", got)
			}
		})
	}
}

// TestScanFixtureDecodesUTF16 pins the second axis: the same construct, in
// the encoding a UTF-8 grep cannot see, matches identically. The
// unprefixed-under-default-xmlns spelling is the one the acceptance fixture
// s3_2_3si10.xsd uses, so this case crosses both axes at once.
func TestScanFixtureDecodesUTF16(t *testing.T) {
	q := mustQuery(t, "element@targetNamespace")
	scan := scanFixture("t.xsd", strings.NewReader(utf16LEDoc(unprefixedDoc)), q)
	if scan.Err != nil {
		t.Fatalf("scanFixture: %v", scan.Err)
	}
	if len(scan.Hits) != 1 {
		t.Fatalf("got %d hit(s), want 1: %+v", len(scan.Hits), scan.Hits)
	}
	if got := scan.Hits[0].Values[0]; got != "urn:b" {
		t.Errorf("targetNamespace = %q, want urn:b", got)
	}
	if scan.Hits[0].Line != 5 {
		t.Errorf("Line = %d, want 5 (the decoded stream's line, not the byte's)", scan.Hits[0].Line)
	}
}

// TestScanFixtureRequiresEveryQueriedAttribute pins the AND semantics of a
// multi-attribute query, and that an element missing one named attribute is
// not an occurrence.
func TestScanFixtureRequiresEveryQueriedAttribute(t *testing.T) {
	doc := `<schema xmlns="http://www.w3.org/2001/XMLSchema">
  <attribute name="a" targetNamespace="urn:b"/>
  <attribute name="c" targetNamespace="urn:b" form="qualified"/>
</schema>`
	scan := scanFixture("t.xsd", strings.NewReader(doc), mustQuery(t, "attribute@targetNamespace,form"))
	if scan.Err != nil {
		t.Fatalf("scanFixture: %v", scan.Err)
	}
	if len(scan.Hits) != 1 {
		t.Fatalf("got %d hit(s), want 1: %+v", len(scan.Hits), scan.Hits)
	}
	want := []string{"urn:b", "qualified"}
	if got := scan.Hits[0].Values; len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("Values = %q, want %q (query order)", got, want)
	}
}

// TestScanFixtureKeepsHitsAheadOfAFault pins the malformed-fixture contract:
// the suite ships documents that are not well-formed on purpose, and the
// constructs before the fault are evidence, not collateral.
func TestScanFixtureKeepsHitsAheadOfAFault(t *testing.T) {
	doc := `<schema xmlns="http://www.w3.org/2001/XMLSchema">
  <element name="a" targetNamespace="urn:b"/>
  </notopen>`
	scan := scanFixture("broken.xsd", strings.NewReader(doc), mustQuery(t, "element@targetNamespace"))
	if scan.Err == nil {
		t.Fatal("scanFixture: want a read fault, got none")
	}
	if len(scan.Hits) != 1 {
		t.Fatalf("got %d hit(s) ahead of the fault, want 1: %+v", len(scan.Hits), scan.Hits)
	}
	if scan.Elems == 0 {
		t.Error("Elems = 0, so the caller would count a broken fixture as a non-XML file")
	}
}

// writeFixture writes one fixture into a census tree, creating its
// directories.
func writeFixture(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("creating %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

// TestCensusWalksTheWholeTree pins the scope half of the defect: one query
// over a tree returns fixtures from every directory in it, in path order,
// whatever each one's encoding or prefix. Scoping a census to the directory
// the last finding named is what missed target002.n.xsd (#1239).
func TestCensusWalksTheWholeTree(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "saxonData/TargetNS/plain.xsd", xsPrefixDoc)
	writeFixture(t, root, "ibmData/S3_2_3/wide.xsd", utf16LEDoc(unprefixedDoc))
	writeFixture(t, root, "ibmData/S3_2_3/prefixed.xsd", xsdPrefixDoc)
	writeFixture(t, root, "msData/impostor.xsd", impostorDoc)
	writeFixture(t, root, "nistMeta/notes.txt", "no markup here at all")
	writeFixture(t, root, "msData/not-wf.xsd", `<schema xmlns="http://www.w3.org/2001/XMLSchema">`)

	rep, err := census(root, mustQuery(t, "element@targetNamespace"))
	if err != nil {
		t.Fatalf("census: %v", err)
	}
	want := []string{
		"ibmData/S3_2_3/prefixed.xsd",
		"ibmData/S3_2_3/wide.xsd",
		"saxonData/TargetNS/plain.xsd",
	}
	var got []string
	for _, h := range rep.Hits {
		got = append(got, h.File)
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("hits =\n%s\nwant (path order)\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if rep.Walked != 6 {
		t.Errorf("Walked = %d, want 6", rep.Walked)
	}
	if len(rep.NoElement) != 1 || rep.NoElement[0].File != "nistMeta/notes.txt" {
		t.Errorf("NoElement = %+v, want the one file with no markup", rep.NoElement)
	}
	if len(rep.Partial) != 1 || rep.Partial[0].File != "msData/not-wf.xsd" {
		t.Errorf("Partial = %+v, want the one unclosed fixture", rep.Partial)
	}
}

// TestReportAccountsForEveryWalkedFile pins the census's arithmetic: every
// file walked is read to the end, listed as partly read, or listed as holding
// no element — never counted away.
func TestReportAccountsForEveryWalkedFile(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "good.xsd", xsPrefixDoc)
	writeFixture(t, root, "prose.txt", "no markup here at all")
	writeFixture(t, root, "broken.xsd", `<schema xmlns="http://www.w3.org/2001/XMLSchema">`)

	rep, err := census(root, mustQuery(t, "element@targetNamespace"))
	if err != nil {
		t.Fatalf("census: %v", err)
	}
	var out strings.Builder
	if err := printReport(&out, rep); err != nil {
		t.Fatalf("printReport: %v", err)
	}
	for _, want := range []string{"prose.txt", "broken.xsd", "3 file(s): 1 read to the end"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("report does not name %q:\n%s", want, out.String())
		}
	}
}

// TestCensusIsDeterministic pins STYLE D1: two runs over one tree render
// byte-identical reports.
func TestCensusIsDeterministic(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "b/two.xsd", utf16LEDoc(unprefixedDoc))
	writeFixture(t, root, "a/one.xsd", xsPrefixDoc)
	writeFixture(t, root, "c/three.xsd", xsdPrefixDoc)

	var runs []string
	for range 2 {
		rep, err := census(root, mustQuery(t, "element@targetNamespace"))
		if err != nil {
			t.Fatalf("census: %v", err)
		}
		var out strings.Builder
		if err := printReport(&out, rep); err != nil {
			t.Fatalf("printReport: %v", err)
		}
		runs = append(runs, out.String())
	}
	if runs[0] != runs[1] {
		t.Errorf("two runs differ:\n%s\n---\n%s", runs[0], runs[1])
	}
	if !strings.Contains(runs[0], "a/one.xsd") {
		t.Errorf("report names no fixture:\n%s", runs[0])
	}
}

// TestRunCorpusAbsent pins the supported degraded mode: a fresh container has
// no suite submodule (#659), so the tool says the corpus is not there and
// exits 0 rather than failing.
func TestRunCorpusAbsent(t *testing.T) {
	cases := []struct {
		name string
		root func(t *testing.T) string
		want string
	}{
		{
			name: "missing directory",
			root: func(t *testing.T) string { return filepath.Join(t.TempDir(), "absent") },
			want: "does not exist",
		},
		{
			name: "empty directory",
			root: func(t *testing.T) string { return t.TempDir() },
			want: "holds no files",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			if err := run(&out, []string{"element@targetNamespace", tc.root(t)}); err != nil {
				t.Fatalf("run: %v, want a clean exit", err)
			}
			if !strings.Contains(out.String(), tc.want) {
				t.Errorf("output does not say %q:\n%s", tc.want, out.String())
			}
			if !strings.Contains(out.String(), "git submodule update --init") {
				t.Errorf("output does not name the command that initializes the suite:\n%s", out.String())
			}
		})
	}
}

// TestParseQuery pins the query grammar, including the two defaults that make
// a braceless query mean what a reader expects: an element name is in the XML
// Schema namespace, an attribute name is in none.
func TestParseQuery(t *testing.T) {
	cases := []struct {
		in    string
		elem  xsd.QName
		attrs []xsd.QName
	}{
		{
			in:   "element",
			elem: xsd.QName{Space: xsd.XMLSchemaNS, Local: "element"},
		},
		{
			in:    "attribute@targetNamespace,form",
			elem:  xsd.QName{Space: xsd.XMLSchemaNS, Local: "attribute"},
			attrs: []xsd.QName{{Local: "targetNamespace"}, {Local: "form"}},
		},
		{
			in:   "{urn:x}thing",
			elem: xsd.QName{Space: "urn:x", Local: "thing"},
		},
		{
			// A namespace may hold the separators; the brace ends the URI.
			in:    "{urn:a@b,c}thing@{urn:d@e}attr",
			elem:  xsd.QName{Space: "urn:a@b,c", Local: "thing"},
			attrs: []xsd.QName{{Space: "urn:d@e", Local: "attr"}},
		},
		{
			in:    "{}bare@x",
			elem:  xsd.QName{Local: "bare"},
			attrs: []xsd.QName{{Local: "x"}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			q, err := parseQuery(tc.in)
			if err != nil {
				t.Fatalf("parseQuery: %v", err)
			}
			if q.Element != tc.elem {
				t.Errorf("Element = %+v, want %+v", q.Element, tc.elem)
			}
			if len(q.Attrs) != len(tc.attrs) {
				t.Fatalf("Attrs = %+v, want %+v", q.Attrs, tc.attrs)
			}
			for i, want := range tc.attrs {
				if q.Attrs[i] != want {
					t.Errorf("Attrs[%d] = %+v, want %+v", i, q.Attrs[i], want)
				}
			}
		})
	}
}

// TestParseQueryRejects pins the malformed queries that must exit 2 rather
// than censusing something the caller did not ask for.
func TestParseQueryRejects(t *testing.T) {
	for _, in := range []string{"", "{urn:x", "@attr", "element@", "element@a@b", "element,a", "{urn:x}a}b"} {
		t.Run(in, func(t *testing.T) {
			if _, err := parseQuery(in); err == nil {
				t.Errorf("parseQuery(%q) = nil error, want a rejection", in)
			}
		})
	}
}

// TestQueryString pins the canonical echo: the report names the namespace
// that was matched, never the prefix a fixture spelled it with.
func TestQueryString(t *testing.T) {
	got := mustQuery(t, "attribute@targetNamespace,form").String()
	want := "{http://www.w3.org/2001/XMLSchema}attribute@targetNamespace,form"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

// TestParseArgs pins the command line: query alone censuses the suite, a
// second argument narrows the tree, anything else is a usage error.
func TestParseArgs(t *testing.T) {
	q, root, err := parseArgs([]string{"element"})
	if err != nil {
		t.Fatalf("parseArgs: %v", err)
	}
	if root != defaultRoot {
		t.Errorf("root = %q, want %q", root, defaultRoot)
	}
	if q.Element.Local != "element" {
		t.Errorf("Element.Local = %q, want element", q.Element.Local)
	}
	if _, root, err = parseArgs([]string{"element", "some/dir"}); err != nil || root != "some/dir" {
		t.Errorf("parseArgs with a dir = %q, %v; want some/dir, nil", root, err)
	}
	for _, args := range [][]string{nil, {"a", "b", "c"}} {
		if _, _, err := parseArgs(args); err == nil {
			t.Errorf("parseArgs(%q) = nil error, want a usage error", args)
		}
	}
}

// suiteRoot locates the W3C suite from this package's directory, or skips:
// the submodule is absent in a fresh container (#659), and a census tool's
// own tests must not be the thing that demands it.
func suiteRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Join("..", "..", defaultRoot)
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) || (err == nil && len(entries) == 0) {
		t.Skipf("suite absent: run `git submodule update --init %s`", defaultRoot)
	}
	if err != nil {
		t.Fatalf("reading %s: %v", root, err)
	}
	return root
}

// TestSuiteAcceptanceCases pins #1239's two acceptance cases against the real
// corpus: each fixture the three under-predicted landings missed must come
// back from ONE whole-corpus query. si10 is UTF-16LE and unprefixed,
// target002 is plain text and `xs:`-prefixed, and si03 differs from its
// neighbours only in the clause it violates — so a census that scopes,
// decodes, or matches by spelling drops at least one of them.
func TestSuiteAcceptanceCases(t *testing.T) {
	root := suiteRoot(t)
	cases := []struct {
		query string
		want  []string
	}{
		{
			query: "element@targetNamespace",
			want: []string{
				"ibmData/schema_invalid/S3_2_3/s3_2_3si10.xsd",
				"saxonData/TargetNS/target002.n.xsd",
			},
		},
		{
			query: "attribute@targetNamespace",
			want: []string{
				"ibmData/schema_invalid/S3_2_3/s3_2_3si02.xsd",
				"ibmData/schema_invalid/S3_2_3/s3_2_3si03.xsd",
				"ibmData/schema_invalid/S3_2_3/s3_2_3si05.xsd",
				"ibmData/schema_invalid/S3_2_3/s3_2_3si09.xsd",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			rep, err := census(root, mustQuery(t, tc.query))
			if err != nil {
				t.Fatalf("census: %v", err)
			}
			matched := map[string]bool{}
			for _, h := range rep.Hits {
				matched[h.File] = true
			}
			for _, w := range tc.want {
				if !matched[w] {
					t.Errorf("%s carries the construct but the census missed it", w)
				}
			}
		})
	}
}
