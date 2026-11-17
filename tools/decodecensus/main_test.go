package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/tools/internal/codeccensus"
	"github.com/kud360/goxsd8/tools/internal/codeccensus/assess"
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

// instanceTest is one instanceTest of the fixture catalog over the document
// doc, under d/.
func instanceTest(name, doc, attrs string) string {
	return `<instanceTest name="` + name + `"` + attrs + `><instanceDocument xlink:href="../d/` + doc + `"/><expected validity="valid"/></instanceTest>`
}

// schemaTest is the schemaTest of a fixture group over the document doc,
// under d/.
func schemaTest(doc string) string {
	return `<schemaTest name="s"><schemaDocument xlink:href="../d/` + doc + `"/><expected validity="valid"/></schemaTest>`
}

// censusSuite is the fixture suite. Group g's schema generates, and its
// cases decode (a.xml, twice, f.xml, of simple type, g.xml, a choice, through
// the branch set the census binds, p.xml, of xs:precisionDecimal, through
// the arm set it binds, n.xml, a nilled simple root, and m.xml and m2.xml, a
// nillable root holding an attribute and a nillable xs:decimal field, not
// nilled and nilled, each through the Nillable instantiations the census
// binds, and v.xml, a union of two basic members of one Go type, through the
// variant set it binds in file order, and c.xml, an optional local element of
// an abstract complex type with no member, left out), decline with an xsi:type
// (e.xml, twice) and a nested model group particle (q.xml), and fail
// otherwise, charged (invalid.xml) and with no declaration for the document
// element (u.xml), which decodes into the placeholder; its case old is withheld.
// Group rec's schema assembles and does not generate, group bad's does not
// assemble, and group none declares no schemaTest.
var censusSuite = map[string]string{
	"suite.xml": `<testSuite xmlns:xlink="http://www.w3.org/1999/xlink"><testSetRef xlink:href="sets/s.testSet"/></testSuite>`,
	"sets/s.testSet": `<testSet name="S" xmlns:xlink="http://www.w3.org/1999/xlink">
  <testGroup name="g">` + schemaTest("s.xsd") +
		instanceTest("decoded", "a.xml", "") + instanceTest("decoded2", "a.xml", "") +
		instanceTest("typed", "e.xml", "") + instanceTest("typed2", "e.xml", "") + instanceTest("variant", "v.xml", "") + instanceTest("simpledecoded", "f.xml", "") +
		instanceTest("abstract", "c.xml", "") + instanceTest("nested", "q.xml", "") + instanceTest("choice", "g.xml", "") + instanceTest("precision", "p.xml", "") + instanceTest("invalid", "invalid.xml", "") +
		instanceTest("nilledsimple", "n.xml", "") + instanceTest("nillable", "m.xml", "") + instanceTest("nilled", "m2.xml", "") +
		instanceTest("undeclared", "u.xml", "") + instanceTest("old", "a.xml", ` version="1.0"`) + `</testGroup>
  <testGroup name="rec">` + schemaTest("rec.xsd") + instanceTest("i", "a.xml", "") + `</testGroup>
  <testGroup name="bad">` + schemaTest("bad.xsd") + instanceTest("i", "a.xml", "") + `</testGroup>
  <testGroup name="none">` + instanceTest("i", "a.xml", "") + `</testGroup>
</testSet>`,
	"d/s.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="a"><xs:complexType><xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence></xs:complexType></xs:element>
  <xs:complexType name="ab" abstract="true"/>
  <xs:element name="c"><xs:complexType><xs:sequence><xs:element name="d" type="ab" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>
  <xs:element name="e" type="xs:string"/>
  <xs:element name="q"><xs:complexType><xs:sequence maxOccurs="2"><xs:element name="b" type="xs:string"/><xs:choice maxOccurs="2"><xs:element name="d" type="xs:string"/></xs:choice></xs:sequence></xs:complexType></xs:element>
  <xs:simpleType name="lower"><xs:restriction base="xs:token"><xs:pattern value="[a-z]+"/></xs:restriction></xs:simpleType>
  <xs:element name="v"><xs:simpleType><xs:union memberTypes="lower xs:string"/></xs:simpleType></xs:element>
  <xs:element name="f" type="xs:string"/>
  <xs:element name="g"><xs:complexType><xs:choice><xs:element name="h" type="xs:string"/><xs:element name="i" type="xs:string"/></xs:choice></xs:complexType></xs:element>
  <xs:element name="p" type="xs:precisionDecimal"/>
  <xs:element name="n" type="xs:string" nillable="true"/>
  <xs:element name="m" nillable="true"><xs:complexType><xs:sequence><xs:element name="v" type="xs:decimal" nillable="true"/></xs:sequence><xs:attribute name="k" type="xs:string"/></xs:complexType></xs:element>
</xs:schema>`,
	"d/rec.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="a"><xs:complexType><xs:sequence><xs:element ref="a"/></xs:sequence></xs:complexType></xs:element>
</xs:schema>`,
	"d/bad.xsd":       `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="a" type="undeclared"/></xs:schema>`,
	"d/a.xml":         `<a><b>x</b></a>`,
	"d/c.xml":         `<c/>`,
	"d/q.xml":         `<q><b>x</b><d>y</d></q>`,
	"d/e.xml":         `<e xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="xs:string" xmlns:xs="http://www.w3.org/2001/XMLSchema">1.0</e>`,
	"d/v.xml":         `<v>X</v>`,
	"d/f.xml":         `<f>x</f>`,
	"d/g.xml":         `<g><i>x</i></g>`,
	"d/p.xml":         `<p>3.00</p>`,
	"d/n.xml":         `<n xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:nil="true"/>`,
	"d/m.xml":         `<m k="1"><v>1.5</v></m>`,
	"d/m2.xml":        `<m xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:nil="true" k="2"/>`,
	"d/invalid.xml":   `<a/>`,
	"d/u.xml":         `<u/>`,
	"d/notes.txt":     "not a document",
	"d/sub/other.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"/>`,
}

// wantCounts is the report's head over censusSuite: every count, and its
// construct lines.
const wantCounts = `decodecensus: 18 instance case(s) under %ROOT%
  schema generates: 15
  decoded: 10
  declined: 3
  undecided: 0
  rejected codegen's own type: 0
  failed other: 2
  decoded where validate rejects: 0

=== First decline, by construct: 2 construct(s) ===
       2  an element carrying xsi:type
       1  a nested model group particle

=== Caveats ===
`

// wantPaths is the -paths sections over censusSuite.
const wantPaths = `
=== Cases, by construct ===
an element carrying xsi:type (2):
  S/g/instance/typed
  S/g/instance/typed2
a nested model group particle (1):
  S/g/instance/nested

=== Cases, by listed count ===
rejected codegen's own type (0):
decoded where validate rejects (0):
Decode's ErrTypeMismatch on the placeholder (0):
`

// withCaveats returns head followed by the caveats' lines and the caveat
// placeholderCaveat renders for rep, as printReport closes its report
// before any -paths section.
func withCaveats(head string, rep report) string {
	var b strings.Builder
	b.WriteString(head)
	for _, c := range caveats {
		b.WriteString("  - " + c + "\n")
	}
	b.WriteString("  - " + placeholderCaveat(rep) + "\n")
	return b.String()
}

// TestCensusCounts runs the whole command over censusSuite, a driver built
// and run included, without and with -paths, and compares each whole report:
// every count the report prints, the decline table, every caveat and the
// placeholder's, and with -paths each declined construct's case IDs in byte
// order. Without -paths the report holds nothing after the placeholder's
// caveat. The functions named below are package codeccensus's and package
// assess's. It fails with instanceCases' withheld test removed, with plan's
// generates count removed, and with the decoded, declined or failed other
// arm of the driver's tallyOutcome removed, and with driverSource's bound
// branch list dropped, which moves g.xml from decoded to rejected, and with
// driverSource emitting codec.WithBranches in place of
// codec.WithPrecisionDecimalArms, which moves p.xml from decoded to rejected,
// and with driverSource's list dropping the Nillable options
// writeNillableOptions declares, or with nillableFamily.root decoding a
// nillable root into its bare type, either of which moves n.xml, m.xml and
// m2.xml from decoded to rejected; undecided, rejected and the differential,
// which no schema document makes codec return, are
// TestReportPrintsEachMergedCount's and package assess's.
func TestCensusCounts(t *testing.T) {
	root := writeTree(t, censusSuite)
	plain := withCaveats(strings.ReplaceAll(wantCounts, "%ROOT%", root), report{Population: codeccensus.Population{Placeholder: 1}})
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"without -paths", []string{root}, plain},
		{"with -paths", []string{"-paths", root}, plain + wantPaths},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := run(&out, tc.args); err != nil {
				t.Fatalf("run: %v", err)
			}
			if out.String() != tc.want {
				t.Errorf("report\n%s\nwant\n%s", out.String(), tc.want)
			}
		})
	}
}

// TestReportPrintsEachMergedCount merges two drivers' summaries, every count
// of each non-zero, one construct in both and each listed count's cases split
// across both, and compares the whole report printReport renders with
// -paths: each count is the sum of the two, and each case ID is printed
// under its own heading, in byte order where the merge met them in reverse,
// so a count or list merge or printReport drops, or prints under another
// label or unsorted, fails here.
func TestReportPrintsEachMergedCount(t *testing.T) {
	rep := report{Population: codeccensus.Population{Root: "r", Considered: 40, Generates: 30, Placeholder: 9, Unresolved: 1}}
	rep.merge(assess.Summary{Decoded: 1, Undecided: 2, FailedOther: 4,
		Rejected:            []string{"S/r/instance/2"},
		Differential:        []string{"S/d/instance/3", "S/d/instance/2"},
		PlaceholderMismatch: []string{"S/m/instance/4", "S/m/instance/3"},
		Declines:            []assess.Tally{{Construct: "x", Cases: []string{"S/x/instance/1"}}, {Construct: "y", Cases: []string{"S/y/instance/3"}}}})
	rep.merge(assess.Summary{Decoded: 10, Undecided: 20, FailedOther: 40,
		Rejected:            []string{"S/r/instance/1"},
		Differential:        []string{"S/d/instance/1"},
		PlaceholderMismatch: []string{"S/m/instance/2", "S/m/instance/1"},
		Declines:            []assess.Tally{{Construct: "y", Cases: []string{"S/y/instance/2", "S/y/instance/1"}}}})
	var out bytes.Buffer
	if err := printReport(&out, rep, true); err != nil {
		t.Fatalf("printReport: %v", err)
	}
	want := withCaveats(`decodecensus: 40 instance case(s) under r
  schema generates: 30
  decoded: 11
  declined: 4
  undecided: 22
  rejected codegen's own type: 2
  failed other: 44
  decoded where validate rejects: 3

=== First decline, by construct: 2 construct(s) ===
       3  y
       1  x

=== Caveats ===
`, rep) + `
=== Cases, by construct ===
y (3):
  S/y/instance/1
  S/y/instance/2
  S/y/instance/3
x (1):
  S/x/instance/1

=== Cases, by listed count ===
rejected codegen's own type (2):
  S/r/instance/1
  S/r/instance/2
decoded where validate rejects (3):
  S/d/instance/1
  S/d/instance/2
  S/d/instance/3
Decode's ErrTypeMismatch on the placeholder (4):
  S/m/instance/1
  S/m/instance/2
  S/m/instance/3
  S/m/instance/4
`
	if out.String() != want {
		t.Errorf("report\n%s\nwant\n%s", out.String(), want)
	}
	if !strings.Contains(out.String(), "Decode's ErrTypeMismatch on the placeholder, 4 case(s)") {
		t.Errorf("the placeholder's caveat in\n%s\ncounts other than the 4 merged cases", out.String())
	}
}

// TestCensusAbsentSuiteIsAMode holds an absent tree, and one with no suite
// index, to the supported mode: a report saying so, and no error.
func TestCensusAbsentSuiteIsAMode(t *testing.T) {
	cases := []struct{ name, root string }{
		{"absent", filepath.Join(t.TempDir(), "absent")},
		{"no index", writeTree(t, map[string]string{"notes.txt": ""})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := run(&out, []string{tc.root}); err != nil {
				t.Fatalf("run: %v", err)
			}
			want := "decodecensus: no suite index at " + filepath.Join(tc.root, "suite.xml") + " — corpus-absent mode, nothing to census."
			if !strings.HasPrefix(out.String(), want) {
				t.Errorf("report = %q, want it to open %q", out.String(), want)
			}
		})
	}
}

// failWriter refuses every write.
type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("refused") }

// TestCensusAbsentNoticeUnwrittenIsAnError holds a corpus-absent notice that
// cannot be written to run's error, which main exits non-zero on.
func TestCensusAbsentNoticeUnwrittenIsAnError(t *testing.T) {
	if err := run(failWriter{}, []string{filepath.Join(t.TempDir(), "absent")}); err == nil {
		t.Error("run wrote its notice to a refusing writer and returned no error")
	}
}
