package main

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
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

	// noNamespaceDoc is written in NO namespace throughout, the shape the
	// corpus carries wherever a wrapper declaring no default xmlns holds a
	// schema (msData/additional/test93490_14.xml). Every element in it is the
	// case #1297 turns on, and its one namespaced attribute is the control.
	noNamespaceDoc = `<?xml version="1.0"?>
<root>
  <wrapper xmlns:o="urn:o" id="w" o:keep="x">
    <inner/>
  </wrapper>
</root>`
)

// ns is the XML Schema namespace in the Clark notation the report renders
// every name in.
const ns = "{http://www.w3.org/2001/XMLSchema}"

// attrPairs renders one hit's matched attributes as "local=value" in the
// order the hit records them, which is what the assertions below compare: a
// []attrHit prints as a wall of struct fields.
func attrPairs(h hit) string {
	var pairs []string
	for _, a := range h.Attrs {
		pairs = append(pairs, a.Name.Local+"="+a.Value)
	}
	return strings.Join(pairs, " ")
}

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
			if got := scan.Hits[0].Attrs[0].Value; got != "urn:b" {
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
	if got := scan.Hits[0].Attrs[0].Value; got != "urn:b" {
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
			t.Errorf("Hits[%d] (%s) Parent = %+v, want %+v", i, scan.Hits[i].Attrs[0].Value, scan.Hits[i].Parent, w)
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

// TestReportSpellsANoNamespaceElementForReEntry pins #1297 at every element
// position a match report prints — the summary line, the `element=` field, the
// parent and the children: a name in no namespace carries an explicit `{}`,
// because a braceless element name re-enters the query language as the XML
// Schema namespace and would name a different element.
func TestReportSpellsANoNamespaceElementForReEntry(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "bare.xml", noNamespaceDoc)

	cases := []struct {
		query string
		want  []string
	}{
		{
			query: "{}wrapper@id",
			want: []string{
				"suiteindex: 1 occurrence(s) of {}wrapper@id in 1 fixture(s) under ",
				`bare.xml:3:3 parent={}root children=[{}inner] id="w"`,
			},
		},
		{
			query: "{}*@id",
			want: []string{
				"suiteindex: 1 occurrence(s) of {}*@id in 1 fixture(s) under ",
				`bare.xml:3:3 element={}wrapper parent={}root children=[{}inner] id="w"`,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			rep, err := census(root, mustQuery(t, tc.query))
			if err != nil {
				t.Fatalf("census: %v", err)
			}
			var out strings.Builder
			if err := printReport(&out, rep); err != nil {
				t.Fatalf("printReport: %v", err)
			}
			for _, want := range tc.want {
				if !strings.Contains(out.String(), want) {
					t.Errorf("report does not carry %q:\n%s", want, out.String())
				}
			}
		})
	}
}

// TestAxisPairSpellsEachHalfForItsOwnPosition pins where #1297's fix stops:
// the element half of a pair takes the `{}`, the attribute half does not,
// since an attribute position of a query already reads a bare name as no
// namespace. Each half re-enters as the query that reported the pair.
func TestAxisPairSpellsEachHalfForItsOwnPosition(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "bare.xml", noNamespaceDoc)

	cases := []struct{ query, want string }{
		{query: "{}*@*", want: "  {}wrapper @id — 1 occurrence(s) in 1 fixture(s)"},
		{query: "{}*@{urn:o}*", want: "  {}wrapper @{urn:o}keep — 1 occurrence(s) in 1 fixture(s)"},
	}
	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			rep, err := census(root, mustQuery(t, tc.query))
			if err != nil {
				t.Fatalf("census: %v", err)
			}
			var out strings.Builder
			if err := printReport(&out, rep); err != nil {
				t.Fatalf("printReport: %v", err)
			}
			if !strings.Contains(out.String(), tc.want) {
				t.Errorf("axis report does not carry %q:\n%s", tc.want, out.String())
			}
		})
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
			t.Errorf("Hits[%d] (%s) is marked unclosed, but the fixture reads to the end", i, h.Attrs[0].Value)
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
	want := "targetNamespace=urn:b form=qualified"
	if got := attrPairs(scan.Hits[0]); got != want {
		t.Errorf("matched attributes = %q, want %q (query order)", got, want)
	}
}

// axisDoc carries every case the wildcard forms turn on: an element with no
// attribute at all, one whose only attributes are in a namespace, two
// carrying one of the queried names each, and one carrying two of them.
const axisDoc = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:o="urn:o" o:keep="x">
  <xs:complexType name="ct" mixed="true" abstract="false"/>
  <xs:element name="a" nillable="true" o:extra="y"/>
  <xs:sequence/>
</xs:schema>`

// TestScanFixtureMatchesEveryElement pins the wildcard element: a census over
// an attribute axis is anchored to no element name, which the query language
// had no spelling for at all before #1391. The impostor namespace still does
// not answer — `*` widens the local name, never the namespace.
func TestScanFixtureMatchesEveryElement(t *testing.T) {
	scan := scanFixture("t.xsd", strings.NewReader(axisDoc), mustQuery(t, "*@name"))
	if scan.Err != nil {
		t.Fatalf("scanFixture: %v", scan.Err)
	}
	want := []xsd.QName{xsdName("complexType"), xsdName("element")}
	if len(scan.Hits) != len(want) {
		t.Fatalf("got %d hit(s), want %d: %+v", len(scan.Hits), len(want), scan.Hits)
	}
	for i, w := range want {
		if scan.Hits[i].Element != w {
			t.Errorf("Hits[%d].Element = %+v, want %+v", i, scan.Hits[i].Element, w)
		}
	}
	if impostor := scanFixture("i.xsd", strings.NewReader(impostorDoc), mustQuery(t, "*@name")); len(impostor.Hits) != 0 {
		t.Errorf("got %d hit(s) in another namespace, want 0: %+v", len(impostor.Hits), impostor.Hits)
	}
}

// TestScanFixtureAnyOfTheQueriedAttributes pins the "|" join against the ","
// join above: an element carrying ANY one of the names is an occurrence, and
// the hit records the names it actually carried rather than the query's.
// #456's population is a disjunction over six names, and the "," spelling of
// it is their intersection — a different census, and here an empty one.
func TestScanFixtureAnyOfTheQueriedAttributes(t *testing.T) {
	scan := scanFixture("t.xsd", strings.NewReader(axisDoc), mustQuery(t, "*@mixed|abstract|nillable"))
	if scan.Err != nil {
		t.Fatalf("scanFixture: %v", scan.Err)
	}
	want := []string{"mixed=true abstract=false", "nillable=true"}
	if len(scan.Hits) != len(want) {
		t.Fatalf("got %d hit(s), want %d: %+v", len(scan.Hits), len(want), scan.Hits)
	}
	for i, w := range want {
		if got := attrPairs(scan.Hits[i]); got != w {
			t.Errorf("Hits[%d] matched %q, want %q", i, got, w)
		}
	}
	and := scanFixture("t.xsd", strings.NewReader(axisDoc), mustQuery(t, "*@mixed,abstract,nillable"))
	if len(and.Hits) != 0 {
		t.Errorf("the \",\" spelling found %d hit(s), want 0 — it is the intersection: %+v", len(and.Hits), and.Hits)
	}
}

// TestScanFixtureCensusesEveryAttributeName pins `@*`: every attribute in the
// wildcard's namespace, in the order the tag spells them. A namespace
// declaration is not an attribute and never appears; an element carrying no
// attribute in that namespace is not an occurrence at all.
func TestScanFixtureCensusesEveryAttributeName(t *testing.T) {
	cases := []struct {
		query string
		want  []string
	}{
		{query: "*@*", want: []string{"name=ct mixed=true abstract=false", "name=a nillable=true"}},
		{query: "*@{urn:o}*", want: []string{"keep=x", "extra=y"}},
	}
	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			scan := scanFixture("t.xsd", strings.NewReader(axisDoc), mustQuery(t, tc.query))
			if scan.Err != nil {
				t.Fatalf("scanFixture: %v", scan.Err)
			}
			if len(scan.Hits) != len(tc.want) {
				t.Fatalf("got %d hit(s), want %d: %+v", len(scan.Hits), len(tc.want), scan.Hits)
			}
			for i, w := range tc.want {
				if got := attrPairs(scan.Hits[i]); got != w {
					t.Errorf("Hits[%d] matched %q, want %q", i, got, w)
				}
			}
		})
	}
}

// TestReportGroupsTheNameAxis pins the shape `@*` reports in, which is the
// one thing #1391 changed about the output: the pairs with their counts, then
// each pair's fixtures, and no per-occurrence section at all. One line per
// occurrence is what the corpus makes unreadable at 172,565 of them.
func TestReportGroupsTheNameAxis(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "a/one.xsd", xsPrefixDoc)
	writeFixture(t, root, "b/two.xsd", utf16LEDoc(unprefixedDoc))
	// Two occurrences of one pair in one fixture: the pair's fixture list
	// names it once and its occurrence count still counts both.
	writeFixture(t, root, "c/three.xsd", `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="p"/>
  <xs:element name="q"/>
</xs:schema>`)

	rep, err := census(root, mustQuery(t, "*@*"))
	if err != nil {
		t.Fatalf("census: %v", err)
	}
	var out strings.Builder
	if err := printReport(&out, rep); err != nil {
		t.Fatalf("printReport: %v", err)
	}
	for _, want := range []string{
		"=== Attribute-name axis: 3 (element, attribute) pair(s), 8 attribute occurrence(s) ===",
		"  " + ns + "complexType @name — 2 occurrence(s) in 2 fixture(s)",
		"  " + ns + "element @name — 4 occurrence(s) in 3 fixture(s)",
		"  " + ns + "element @targetNamespace — 2 occurrence(s) in 2 fixture(s)",
		"=== Fixtures per pair (path order) ===",
		"  " + ns + "element @name\n      a/one.xsd\n      b/two.xsd\n      c/three.xsd\n",
		"  " + ns + "element @targetNamespace\n      a/one.xsd\n      b/two.xsd\n",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("axis report does not carry %q:\n%s", want, out.String())
		}
	}
	if strings.Contains(out.String(), "=== Matches") {
		t.Errorf("the axis report also printed the per-occurrence section:\n%s", out.String())
	}
}

// TestReportNamesTheMatchedElementOnlyForAWildcard pins the one thing the
// match line gained: a query that leaves the element open says which element
// each hit was, and one that fixed it does not repeat itself.
func TestReportNamesTheMatchedElementOnlyForAWildcard(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "one.xsd", xsPrefixDoc)

	cases := []struct {
		query string
		want  bool
	}{
		{query: "*@targetNamespace", want: true},
		{query: "element@targetNamespace", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			rep, err := census(root, mustQuery(t, tc.query))
			if err != nil {
				t.Fatalf("census: %v", err)
			}
			var out strings.Builder
			if err := printReport(&out, rep); err != nil {
				t.Fatalf("printReport: %v", err)
			}
			if got := strings.Contains(out.String(), "element="+ns+"element"); got != tc.want {
				t.Errorf("report names the matched element = %v, want %v:\n%s", got, tc.want, out.String())
			}
		})
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
// exits 0 rather than failing. Every query form is driven through the guard,
// the axis forms included: the mode is a property of the root and must not
// depend on which report the query would have printed.
func TestRunCorpusAbsent(t *testing.T) {
	queries := []string{"element@targetNamespace", "*@*", "*@mixed|abstract"}
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
		for _, q := range queries {
			t.Run(tc.name+" "+q, func(t *testing.T) {
				var out strings.Builder
				if err := run(&out, []string{q, tc.root(t)}); err != nil {
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
}

// TestParseQuery pins the query grammar, including the two defaults that make
// a braceless query mean what a reader expects: an element name is in the XML
// Schema namespace, an attribute name is in none. The wildcard and the two
// joins are pinned here too — the axis forms (#1391) turn on both.
func TestParseQuery(t *testing.T) {
	cases := []struct {
		in    string
		elem  namePat
		attrs []namePat
		join  attrJoin
	}{
		{
			in:   "element",
			elem: namePat{Space: xsd.XMLSchemaNS, Local: "element"},
			join: joinAll,
		},
		{
			in:    "attribute@targetNamespace,form",
			elem:  namePat{Space: xsd.XMLSchemaNS, Local: "attribute"},
			attrs: []namePat{{Local: "targetNamespace"}, {Local: "form"}},
			join:  joinAll,
		},
		{
			in:   "{urn:x}thing",
			elem: namePat{Space: "urn:x", Local: "thing"},
			join: joinAll,
		},
		{
			// A namespace may hold the separators; the brace ends the URI.
			in:    "{urn:a@b,c|d}thing@{urn:d@e}attr",
			elem:  namePat{Space: "urn:a@b,c|d", Local: "thing"},
			attrs: []namePat{{Space: "urn:d@e", Local: "attr"}},
			join:  joinAll,
		},
		{
			in:    "{}bare@x",
			elem:  namePat{Local: "bare"},
			attrs: []namePat{{Local: "x"}},
			join:  joinAll,
		},
		{
			// The name axis: every element in the XSD namespace, every
			// attribute in none.
			in:    "*@*",
			elem:  namePat{Space: xsd.XMLSchemaNS, Local: wildcard},
			attrs: []namePat{{Local: wildcard}},
			join:  joinAll,
		},
		{
			// The value axis: any of six names, on any element.
			in:    "*@mixed|abstract",
			elem:  namePat{Space: xsd.XMLSchemaNS, Local: wildcard},
			attrs: []namePat{{Local: "mixed"}, {Local: "abstract"}},
			join:  joinAny,
		},
		{
			in:    "{urn:x}*@{urn:y}*",
			elem:  namePat{Space: "urn:x", Local: wildcard},
			attrs: []namePat{{Space: "urn:y", Local: wildcard}},
			join:  joinAll,
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
			if q.Join != tc.join {
				t.Errorf("Join = %q, want %q", q.Join, tc.join)
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
// than censusing something the caller did not ask for. The last three are the
// axis grammar's own rejections: one query means one join, and `@*` already
// names every attribute, so a name beside it says nothing under either join.
func TestParseQueryRejects(t *testing.T) {
	for _, in := range []string{
		"", "{urn:x", "@attr", "element@", "element@a@b", "element,a", "{urn:x}a}b",
		"element@a,b|c", "element@a|b,c", "element@*,name", "element@name|*",
	} {
		t.Run(in, func(t *testing.T) {
			if _, err := parseQuery(in); err == nil {
				t.Errorf("parseQuery(%q) = nil error, want a rejection", in)
			}
		})
	}
}

// TestQueryString pins the canonical echo: the report names the namespace
// that was matched, never the prefix a fixture spelled it with, and echoes
// the join it was given so the two readings are never confused in a report
// header.
func TestQueryString(t *testing.T) {
	cases := []struct{ in, want string }{
		{"attribute@targetNamespace,form", "{http://www.w3.org/2001/XMLSchema}attribute@targetNamespace,form"},
		{"*@*", "{http://www.w3.org/2001/XMLSchema}*@*"},
		{"*@mixed|abstract", "{http://www.w3.org/2001/XMLSchema}*@mixed|abstract"},
		// An element name in no namespace keeps the wrapper it was written
		// with: bare, it would echo as a name in the XML Schema namespace
		// (#1297). The attribute half is bare because that position reads a
		// bare name as no namespace already.
		{"{}bare@x", "{}bare@x"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := mustQuery(t, tc.in).String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestQueryStringReEntersAsItself pins what the echo is FOR: the canonical
// form a report prints parses back to the query that produced it, so a reader
// re-runs a census from the line naming it. The no-namespace element name is
// the case that fails without an explicit `{}` — it re-enters as a name in
// the XML Schema namespace, which is a different census (#1297).
func TestQueryStringReEntersAsItself(t *testing.T) {
	for _, in := range []string{
		"element", "{}bare", "{}bare@x", "{urn:x}thing@{urn:y}attr",
		"{}*@*", "*@mixed|abstract", "{}*@{urn:y}*",
	} {
		t.Run(in, func(t *testing.T) {
			q := mustQuery(t, in)
			back, err := parseQuery(q.String())
			if err != nil {
				t.Fatalf("parseQuery(%q): %v", q.String(), err)
			}
			if back.Element != q.Element {
				t.Errorf("%q re-entered as element %+v, want %+v", q.String(), back.Element, q.Element)
			}
			if back.Join != q.Join {
				t.Errorf("%q re-entered with join %q, want %q", q.String(), back.Join, q.Join)
			}
			if len(back.Attrs) != len(q.Attrs) {
				t.Fatalf("%q re-entered with attrs %+v, want %+v", q.String(), back.Attrs, q.Attrs)
			}
			for i, want := range q.Attrs {
				if back.Attrs[i] != want {
					t.Errorf("%q re-entered with Attrs[%d] = %+v, want %+v", q.String(), i, back.Attrs[i], want)
				}
			}
		})
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

// TestSuiteAttributeNameAxis re-derives #1369's census FROM THE TOOL. That
// census walked all 15,470 schema documents from a DIAG-gated test written for
// the one landing and deleted with it — the second hand-rolled corpus
// instrument in four days (#1391).
//
// The figures pinned here are the axis itself, roster-free. #1369's own
// 22/16/21 is this census MINUS the 49-entry Appendix A roster that landing
// added, and the subtraction stays the reader's: the roster is
// `parser/produce_s4sattrs.go`'s data, and a copy of it here would be a second
// encoding of one fact (STYLE D3). What this test pins on that side is that
// the pairs the subtraction turns on are IN the census with their fixtures —
// three attribute names Appendix A declares nowhere on the element carrying
// them, one of them a case-mangled `substitutionGroup`.
func TestSuiteAttributeNameAxis(t *testing.T) {
	root := suiteRoot(t)
	rep, err := census(root, mustQuery(t, "*@*"))
	if err != nil {
		t.Fatalf("census: %v", err)
	}
	groups := groupPairs(rep.Hits)
	occurrences := 0
	for _, g := range groups {
		occurrences += g.Occurrences
	}
	// The second figure below is why the axis report is grouped and the first
	// is what it is grouped into: one line per occurrence is not a report.
	if len(groups) != 185 || occurrences != 172565 {
		t.Errorf("axis = %d pair(s), %d attribute occurrence(s); want 185, 172565", len(groups), occurrences)
	}

	want := map[pairName]pairGroup{
		{Element: xsdName("element"), Attr: xsd.QName{Local: "nullable"}}: {
			Occurrences: 1, Fixtures: []string{"msData/element/elemK007.xsd"},
		},
		{Element: xsdName("element"), Attr: xsd.QName{Local: "SubstitutionGroup"}}: {
			Occurrences: 1, Fixtures: []string{"msData/schema/78029c.xsd"},
		},
		{Element: xsdName("attribute"), Attr: xsd.QName{Local: "value"}}: {
			Occurrences: 5, Fixtures: []string{
				"msData/attribute/attH001.xsd",
				"msData/attribute/attJ017.xsd",
				"msData/simpleType/stB007.xsd",
				"msData/simpleType/stC027.xsd",
				"msData/simpleType/stC028.xsd",
			},
		},
	}
	for _, g := range groups {
		w, ok := want[g.pairName]
		if !ok {
			continue
		}
		delete(want, g.pairName)
		if g.Occurrences != w.Occurrences {
			t.Errorf("%s = %d occurrence(s), want %d", renderPair(g), g.Occurrences, w.Occurrences)
		}
		if strings.Join(g.Fixtures, " ") != strings.Join(w.Fixtures, " ") {
			t.Errorf("%s in %v, want %v", renderPair(g), g.Fixtures, w.Fixtures)
		}
	}
	// The leftovers are drained and sorted before they are reported, because
	// the failure text is output and a map order would spell the same fault
	// differently on every run (STYLE D2).
	missed := make([]pairName, 0, len(want))
	for p := range want {
		missed = append(missed, p)
	}
	sort.Slice(missed, func(i, j int) bool { return lessPair(missed[i], missed[j]) })
	for _, p := range missed {
		t.Errorf("the axis missed %s@%s, which the corpus carries", p.Element, p.Attr)
	}
}

// TestSuiteBooleanAttributeValueCensus re-derives #456's census FROM THE TOOL:
// every occurrence of six boolean-valued attributes in the corpus, partitioned
// on the value's lexical space (xmlschema11-2.md §3.2.2.1 — the boolean
// lexical space is exactly true, false, 1 and 0, and Appendix A fixes
// whiteSpace to collapse, so a padded literal is VALID and the set #456
// improved is the out-of-space partition alone).
//
// The population is a DISJUNCTION over the six names, which is the "|" join;
// the "," join is their intersection and a different census entirely. #456
// hand-rolled this one, reported 10 occurrences in 8 fixtures against 20 out
// of the lexical space, and deleted the script. The occurrence figures
// reproduce; the fixture count does NOT, and the corpus says the tool is
// right — `saxonData/CTA/cta0010.xsd` carries two padded `inheritable`
// attributes, so the padded partition falls in NINE fixtures, not eight.
func TestSuiteBooleanAttributeValueCensus(t *testing.T) {
	root := suiteRoot(t)
	rep, err := census(root, mustQuery(t,
		"*@mixed|abstract|nillable|inheritable|appliesToEmpty|defaultAttributesApply"))
	if err != nil {
		t.Fatalf("census: %v", err)
	}
	lexical := map[string]bool{"true": true, "false": true, "1": true, "0": true}
	padded, outside := 0, 0
	paddedFixtures := map[string]bool{}
	for _, h := range rep.Hits {
		for _, a := range h.Attrs {
			if lexical[a.Value] {
				continue
			}
			if !lexical[strings.TrimSpace(a.Value)] {
				outside++
				continue
			}
			padded++
			paddedFixtures[h.File] = true
		}
	}
	if padded != 10 || len(paddedFixtures) != 9 || outside != 20 {
		t.Errorf("census = %d padded-but-valid occurrence(s) in %d fixture(s), %d out of the lexical space;"+
			" want 10 in 9, and 20", padded, len(paddedFixtures), outside)
	}
}
