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

// TestScanFixtureRecordsTheParent pins the field #1282 added: each hit names
// the element it is a DIRECT child of, resolved and in document order, so a
// local-versus-top-level census reads off the report instead of being
// hand-written (Appendix A's xs:localElement versus xs:topLevelElement).
func TestScanFixtureRecordsTheParent(t *testing.T) {
	doc := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="top" abstract="true"/>
  <xs:complexType name="ct">
    <xs:sequence>
      <xs:element name="local" abstract="false"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="after" abstract="true"/>
</xs:schema>`
	scan := scanFixture("t.xsd", strings.NewReader(doc), mustQuery(t, "element@abstract"))
	if scan.Err != nil {
		t.Fatalf("scanFixture: %v", scan.Err)
	}
	want := []xsd.QName{
		{Space: xsd.XMLSchemaNS, Local: "schema"},
		{Space: xsd.XMLSchemaNS, Local: "sequence"},
		// The third pins the pop: without it "after" inherits the depth the
		// nested declaration left behind.
		{Space: xsd.XMLSchemaNS, Local: "schema"},
	}
	if len(scan.Hits) != len(want) {
		t.Fatalf("got %d hit(s), want %d: %+v", len(scan.Hits), len(want), scan.Hits)
	}
	for i, w := range want {
		if scan.Hits[i].Parent != w {
			t.Errorf("Hits[%d] (%s) Parent = %+v, want %+v", i, scan.Hits[i].Values[0], scan.Hits[i].Parent, w)
		}
	}
}

// TestScanFixtureParentOfDocumentElement pins the absent case: a match that is
// the document element has no parent, and the zero QName is how that is said.
func TestScanFixtureParentOfDocumentElement(t *testing.T) {
	doc := `<xs:element xmlns:xs="http://www.w3.org/2001/XMLSchema" name="root" abstract="true"/>`
	scan := scanFixture("t.xsd", strings.NewReader(doc), mustQuery(t, "element@abstract"))
	if scan.Err != nil {
		t.Fatalf("scanFixture: %v", scan.Err)
	}
	if len(scan.Hits) != 1 {
		t.Fatalf("got %d hit(s), want 1: %+v", len(scan.Hits), scan.Hits)
	}
	if got := scan.Hits[0].Parent; got != (xsd.QName{}) {
		t.Errorf("Parent = %+v, want the zero QName (no parent)", got)
	}
}

// TestScanFixtureResolvesTheParentInItsOwnScope pins the hazard a prefix-text
// stack would fall into: the parent's name comes from the bindings in force at
// the PARENT, which its child here rebinds to something else entirely.
func TestScanFixtureResolvesTheParentInItsOwnScope(t *testing.T) {
	doc := `<p:schema xmlns:p="http://www.w3.org/2001/XMLSchema">
  <p:element xmlns:p="urn:decoy" name="x"/>
  <q:element xmlns:q="http://www.w3.org/2001/XMLSchema" name="y" abstract="true"/>
</p:schema>`
	scan := scanFixture("t.xsd", strings.NewReader(doc), mustQuery(t, "element@abstract"))
	if scan.Err != nil {
		t.Fatalf("scanFixture: %v", scan.Err)
	}
	if len(scan.Hits) != 1 {
		t.Fatalf("got %d hit(s), want 1: %+v", len(scan.Hits), scan.Hits)
	}
	want := xsd.QName{Space: xsd.XMLSchemaNS, Local: "schema"}
	if got := scan.Hits[0].Parent; got != want {
		t.Errorf("Parent = %+v, want %+v", got, want)
	}
}

// TestReportNamesEachHitsParent pins that the field reaches the reader: the
// default report prints the parent beside file:line:col, and says so for a
// document element too.
func TestReportNamesEachHitsParent(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "nested.xsd", xsPrefixDoc)
	writeFixture(t, root, "bare.xsd",
		`<xs:element xmlns:xs="http://www.w3.org/2001/XMLSchema" name="r" targetNamespace="urn:b"/>`)

	rep, err := census(root, mustQuery(t, "element@targetNamespace"))
	if err != nil {
		t.Fatalf("census: %v", err)
	}
	var out strings.Builder
	if err := printReport(&out, rep); err != nil {
		t.Fatalf("printReport: %v", err)
	}
	for _, want := range []string{
		"bare.xsd:1:1 parent=(none) children=[] targetNamespace=\"urn:b\"",
		"nested.xsd:5:7 parent={http://www.w3.org/2001/XMLSchema}sequence children=[] targetNamespace=\"urn:b\"",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("report does not carry %q:\n%s", want, out.String())
		}
	}
}

// xsdName is one XSD-vocabulary name, for the child lists the tests below
// compare against.
func xsdName(local string) xsd.QName {
	return xsd.QName{Space: xsd.XMLSchemaNS, Local: local}
}

// childNames renders one hit's child list for a failure message, since a
// []xsd.QName prints as a wall of struct fields.
func childNames(h hit) string {
	var got []string
	for _, c := range h.Children {
		got = append(got, c.Local)
	}
	return "[" + strings.Join(got, " ") + "]"
}

// checkChildren compares each hit's child list against want, position by
// position, so a missing repeat fails as loudly as a missing name.
func checkChildren(t *testing.T, scan fixtureScan, want [][]xsd.QName) {
	t.Helper()
	if len(scan.Hits) != len(want) {
		t.Fatalf("got %d hit(s), want %d: %+v", len(scan.Hits), len(want), scan.Hits)
	}
	for i, w := range want {
		got := scan.Hits[i].Children
		if len(got) != len(w) {
			t.Errorf("Hits[%d].Children = %s, want %d name(s): %+v", i, childNames(scan.Hits[i]), len(w), w)
			continue
		}
		for j, name := range w {
			if got[j] != name {
				t.Errorf("Hits[%d].Children[%d] = %+v, want %+v", i, j, got[j], name)
			}
		}
	}
}

// TestScanFixtureRecordsDirectChildren pins the field #1304 added: a hit
// carries the elements DIRECTLY under its match, in document order, with
// repeats kept. The two xs:simpleType children are the multiplicity half of a
// content-model census, which a deduplicated list would discard; the
// xs:documentation is a grandchild and must not appear at all.
func TestScanFixtureRecordsDirectChildren(t *testing.T) {
	doc := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:alternative test="a">
    <xs:annotation><xs:documentation/></xs:annotation>
    <xs:simpleType/>
    <xs:simpleType/>
  </xs:alternative>
  <xs:alternative test="b"/>
  <xs:alternative test="c"></xs:alternative>
</xs:schema>`
	scan := scanFixture("t.xsd", strings.NewReader(doc), mustQuery(t, "alternative@test"))
	if scan.Err != nil {
		t.Fatalf("scanFixture: %v", scan.Err)
	}
	checkChildren(t, scan, [][]xsd.QName{
		{xsdName("annotation"), xsdName("simpleType"), xsdName("simpleType")},
		// Self-closed and empty: no element child, and both fully read.
		nil,
		nil,
	})
	for i, h := range scan.Hits {
		if h.ChildrenUnclosed {
			t.Errorf("Hits[%d] (%s) is marked unclosed, but the fixture reads to the end", i, h.Values[0])
		}
	}
}

// TestScanFixtureRecordsChildrenOfNestedMatches pins the stack: a query
// matching at several depths gives each match its own list, and an outer
// match collects only what stands directly under IT — the inner match here is
// a great-grandchild and belongs to neither list.
func TestScanFixtureRecordsChildrenOfNestedMatches(t *testing.T) {
	doc := `<xs:element xmlns:xs="http://www.w3.org/2001/XMLSchema" name="outer" abstract="true">
  <xs:complexType>
    <xs:sequence>
      <xs:element name="inner" abstract="false">
        <xs:annotation/>
        <xs:complexType/>
      </xs:element>
    </xs:sequence>
  </xs:complexType>
</xs:element>`
	scan := scanFixture("t.xsd", strings.NewReader(doc), mustQuery(t, "element@abstract"))
	if scan.Err != nil {
		t.Fatalf("scanFixture: %v", scan.Err)
	}
	checkChildren(t, scan, [][]xsd.QName{
		{xsdName("complexType")},
		{xsdName("annotation"), xsdName("complexType")},
	})
}

// TestScanFixtureMarksOnlyTheMatchesLeftOpen pins the distinction the
// file-level [report.Partial] flag cannot carry (#1304): a fixture whose read
// faults still holds matches that closed ahead of the fault, and their child
// lists are complete. The corpus's own instance is
// msData/additional/test79253.xsd, which faults at its tail: `element@type`
// finds 13 matches in it, every one of them self-closed with a known-empty
// child list, and a file-level flag would report all 13 as unread.
func TestScanFixtureMarksOnlyTheMatchesLeftOpen(t *testing.T) {
	doc := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="closed" abstract="true">
    <xs:complexType/>
  </xs:element>
  <xs:element name="cut" abstract="true">
    <xs:annotation/>
    <xs:whitespace>
</xs:schema>`
	scan := scanFixture("broken.xsd", strings.NewReader(doc), mustQuery(t, "element@abstract"))
	if scan.Err == nil {
		t.Fatal("scanFixture: want a read fault, got none")
	}
	checkChildren(t, scan, [][]xsd.QName{
		{xsdName("complexType")},
		{xsdName("annotation"), xsdName("whitespace")},
	})
	if scan.Hits[0].ChildrenUnclosed {
		t.Error("Hits[0] (closed) is marked unclosed, but its end tag was read before the fault")
	}
	if !scan.Hits[1].ChildrenUnclosed {
		t.Error("Hits[1] (cut) is not marked unclosed, so a prefix of its children reads as the whole list")
	}
}

// TestReportNamesEachHitsChildren pins that the list reaches the reader: the
// default report prints it beside the parent, "[]" for a match with no
// element child, and an "(unclosed)" tail for one the read never closed.
func TestReportNamesEachHitsChildren(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "full.xsd", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="a" targetNamespace="urn:b">
    <xs:complexType/>
    <xs:complexType/>
  </xs:element>
  <xs:element name="c" targetNamespace="urn:b"/>
</xs:schema>`)
	writeFixture(t, root, "cut.xsd", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="d" targetNamespace="urn:b">
    <xs:annotation>
</xs:schema>`)

	rep, err := census(root, mustQuery(t, "element@targetNamespace"))
	if err != nil {
		t.Fatalf("census: %v", err)
	}
	var out strings.Builder
	if err := printReport(&out, rep); err != nil {
		t.Fatalf("printReport: %v", err)
	}
	const ns = "{http://www.w3.org/2001/XMLSchema}"
	for _, want := range []string{
		"cut.xsd:2:3 parent=" + ns + "schema children=[" + ns + "annotation, (unclosed)]",
		"full.xsd:2:3 parent=" + ns + "schema children=[" + ns + "complexType, " + ns + "complexType]",
		"full.xsd:6:3 parent=" + ns + "schema children=[]",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("report does not carry %q:\n%s", want, out.String())
		}
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

// TestSuiteTopLevelCensusOfFinalAndAbstract pins #1205's census, which two
// agents in one session each hand-wrote a namespace-aware walk to take
// because the tool could not answer it (#1282). The query language is a
// conjunction over one element's own attributes, so "final= OR abstract=" is
// two censuses deduped by position — that composition is the whole method,
// and the figures below are the pin it reproduces.
func TestSuiteTopLevelCensusOfFinalAndAbstract(t *testing.T) {
	root := suiteRoot(t)
	type where struct {
		File string
		Line int
		Col  int
	}
	seen := map[where]bool{}
	var union []hit
	for _, q := range []string{"element@final", "element@abstract"} {
		rep, err := census(root, mustQuery(t, q))
		if err != nil {
			t.Fatalf("census %s: %v", q, err)
		}
		for _, h := range rep.Hits {
			at := where{h.File, h.Line, h.Col}
			if seen[at] {
				continue
			}
			seen[at] = true
			union = append(union, h)
		}
	}

	// The three parents #1205 grouped as top-level. `redefine` admits no
	// `element` child (xmlschema11-1.md:4078), so a hit under one is an
	// invalid fixture reported verbatim — this tool censuses, it does not
	// correct.
	topLevel := map[xsd.QName]bool{
		{Space: xsd.XMLSchemaNS, Local: "schema"}:   true,
		{Space: xsd.XMLSchemaNS, Local: "redefine"}: true,
		{Space: xsd.XMLSchemaNS, Local: "override"}: true,
	}
	top, local := 0, 0
	for _, h := range union {
		if topLevel[h.Parent] {
			top++
			continue
		}
		local++
	}
	if len(union) != 139 || top != 133 || local != 6 {
		t.Errorf("census = %d hit(s), %d top-level, %d local; want 139, 133, 6", len(union), top, local)
	}
}

// TestSuiteAlternativeContentModel re-derives #1275's census FROM THE TOOL.
// Two agents in one session each hand-wrote a walk to reach it and neither
// scan survived (#1304); this reads it off the hits' own child lists.
// xs:altType's model is "(annotation?, (simpleType | complexType)?)"
// (docs/specs/md/xmlschema11-1.md:5137), and #1275 concluded that no suite
// occurrence carries a child outside it, nor a second simpleType/complexType
// beside a first.
//
// A match whose read broke off first fails here rather than passing quietly:
// a prefix of a child list cannot answer a question about the whole of one.
func TestSuiteAlternativeContentModel(t *testing.T) {
	root := suiteRoot(t)
	rep, err := census(root, mustQuery(t, "alternative"))
	if err != nil {
		t.Fatalf("census: %v", err)
	}
	if len(rep.Hits) != 232 || countFiles(rep.Hits) != 81 {
		t.Errorf("census = %d occurrence(s) in %d fixture(s); want 232 in 81",
			len(rep.Hits), countFiles(rep.Hits))
	}
	admitted := map[xsd.QName]bool{
		xsdName("annotation"):  true,
		xsdName("simpleType"):  true,
		xsdName("complexType"): true,
	}
	withType := 0
	for _, h := range rep.Hits {
		if h.ChildrenUnclosed {
			t.Errorf("%s:%d:%d — the read stopped inside this match, so %s is a prefix and settles nothing",
				h.File, h.Line, h.Col, childNames(h))
		}
		types := 0
		for _, c := range h.Children {
			if !admitted[c] {
				t.Errorf("%s:%d:%d carries child %s, which xs:altType does not admit", h.File, h.Line, h.Col, c)
			}
			if c.Local == "simpleType" || c.Local == "complexType" {
				types++
			}
		}
		if types > 1 {
			t.Errorf("%s:%d:%d carries %d type children, %s — xs:altType admits at most one",
				h.File, h.Line, h.Col, types, childNames(h))
		}
		if types == 1 {
			withType++
		}
	}
	// The corpus does exercise the inline type child, so the check above is
	// not passing on 232 empty lists: a census that collected no child at all
	// would satisfy every assertion in this test but this one.
	if withType != 43 {
		t.Errorf("%d occurrence(s) carry an inline type child, want 43", withType)
	}
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
