package main

import (
	"bytes"
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
func instanceTest(name, doc string) string {
	return `<instanceTest name="` + name + `"><instanceDocument xlink:href="../d/` + doc + `"/><expected validity="valid"/></instanceTest>`
}

// censusSuite is the fixture suite, one group whose schema generates. Its
// cases encode (a.xml, a sequence, f.xml, of simple type, n.xml, z.xml and
// o.xml, xs:double values NaN, −0 and 1, the last spelled other than
// canonically, g.xml, a choice, and p.xml, of a union type), decline (l.xml,
// a sequence codegen flattens), and lie outside the population: invalid.xml,
// which does not decode, and u.xml, whose document element no declaration
// names, which decodes into the placeholder.
var censusSuite = map[string]string{
	"suite.xml": `<testSuite xmlns:xlink="http://www.w3.org/1999/xlink"><testSetRef xlink:href="sets/s.testSet"/></testSuite>`,
	"sets/s.testSet": `<testSet name="S" xmlns:xlink="http://www.w3.org/1999/xlink">
  <testGroup name="g"><schemaTest name="s"><schemaDocument xlink:href="../d/s.xsd"/><expected validity="valid"/></schemaTest>` +
		instanceTest("seq", "a.xml") + instanceTest("simple", "f.xml") + instanceTest("nan", "n.xml") + instanceTest("negzero", "z.xml") +
		instanceTest("one", "o.xml") + instanceTest("choice", "g.xml") + instanceTest("flattened", "l.xml") + instanceTest("union", "p.xml") +
		instanceTest("invalid", "invalid.xml") + instanceTest("undeclared", "u.xml") + `</testGroup>
</testSet>`,
	"d/s.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="a"><xs:complexType><xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence></xs:complexType></xs:element>
  <xs:element name="f" type="xs:string"/>
  <xs:element name="d" type="xs:double"/>
  <xs:element name="g"><xs:complexType><xs:choice><xs:element name="h" type="xs:string"/><xs:element name="i" type="xs:string"/></xs:choice></xs:complexType></xs:element>
  <xs:element name="l"><xs:complexType><xs:sequence maxOccurs="unbounded"><xs:element name="m" type="xs:string" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>
  <xs:element name="p"><xs:simpleType><xs:union memberTypes="xs:string xs:int"/></xs:simpleType></xs:element>
</xs:schema>`,
	"d/a.xml":       `<a><b>x</b></a>`,
	"d/f.xml":       `<f>x</f>`,
	"d/n.xml":       `<d>NaN</d>`,
	"d/z.xml":       `<d>-0</d>`,
	"d/o.xml":       `<d> 1 </d>`,
	"d/g.xml":       `<g><i>x</i></g>`,
	"d/l.xml":       `<l><m>x</m></l>`,
	"d/p.xml":       `<p>3</p>`,
	"d/invalid.xml": `<a/>`,
	"d/u.xml":       `<u/>`,
}

// wantCounts is the report's head over censusSuite: every count, and its
// construct lines.
const wantCounts = `encodecensus: 10 instance case(s) under %ROOT%
  schema generates: 10
  decoded: 8
  encoded: 7
  declined: 1
  append failed, no decline: 0
  re-validate rejected: 0
  re-decode failed: 0
  round trip unequal: 0

=== First decline, by construct: 1 construct(s) ===
       1  a construct codegen holds in a sealed interface: a choice, a flattened particle or a substitution group

=== Caveats ===
`

// wantPaths is the -paths sections over censusSuite.
const wantPaths = `
=== Cases, by construct ===
a construct codegen holds in a sealed interface: a choice, a flattened particle or a substitution group (1):
  S/g/instance/flattened

=== Cases, by defect count ===
append failed, no decline (0):
re-validate rejected (0):
re-decode failed (0):
round trip unequal (0):
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
// and run included, without and with -paths, and compares each whole report.
// The encoded and declined rows run codec end to end; the four defect rows,
// which no correct codec fills, are package assess's
// TestDriveEncodeCountsEachRow and TestReportPrintsEachMergedCount's. It
// fails with roundTrip comparing by reflect.DeepEqual in place of same, and
// with codeccensus's entryOf naming Drive for every summary.
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

// TestReportPrintsEachMergedCount merges two drivers' summaries, every row
// of each non-empty, one construct in both and each defect count's cases
// split across both, and compares the whole report printReport renders with
// -paths: each count is the sum of the two, decoded the sum of every row,
// and each case ID is printed under its own heading, in byte order where the
// merge met them in reverse, so a count or list merge or printReport drops,
// or prints under another label or unsorted, fails here.
func TestReportPrintsEachMergedCount(t *testing.T) {
	rep := report{Population: codeccensus.Population{Root: "r", Considered: 40, Generates: 30, Placeholder: 9}}
	rep.merge(assess.EncodeSummary{Encoded: 1,
		AppendFailed:       []string{"S/f/instance/2"},
		ReValidateRejected: []string{"S/v/instance/3", "S/v/instance/2"},
		ReDecodeFailed:     []string{"S/d/instance/1"},
		RoundTripUnequal:   []string{"S/u/instance/2"},
		Declines:           []assess.Tally{{Construct: "x", Cases: []string{"S/x/instance/1"}}, {Construct: "y", Cases: []string{"S/y/instance/3"}}}})
	rep.merge(assess.EncodeSummary{Encoded: 10,
		AppendFailed:       []string{"S/f/instance/1"},
		ReValidateRejected: []string{"S/v/instance/1"},
		ReDecodeFailed:     []string{"S/d/instance/3", "S/d/instance/2"},
		RoundTripUnequal:   []string{"S/u/instance/1"},
		Declines:           []assess.Tally{{Construct: "y", Cases: []string{"S/y/instance/2", "S/y/instance/1"}}}})
	var out bytes.Buffer
	if err := printReport(&out, rep, true); err != nil {
		t.Fatalf("printReport: %v", err)
	}
	want := withCaveats(`encodecensus: 40 instance case(s) under r
  schema generates: 30
  decoded: 25
  encoded: 11
  declined: 4
  append failed, no decline: 2
  re-validate rejected: 3
  re-decode failed: 3
  round trip unequal: 2

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

=== Cases, by defect count ===
append failed, no decline (2):
  S/f/instance/1
  S/f/instance/2
re-validate rejected (3):
  S/v/instance/1
  S/v/instance/2
  S/v/instance/3
re-decode failed (3):
  S/d/instance/1
  S/d/instance/2
  S/d/instance/3
round trip unequal (2):
  S/u/instance/1
  S/u/instance/2
`
	if out.String() != want {
		t.Errorf("report\n%s\nwant\n%s", out.String(), want)
	}
}

// TestCensusAbsentSuiteIsAMode holds an absent tree to the supported mode: a
// report saying so, and no error.
func TestCensusAbsentSuiteIsAMode(t *testing.T) {
	root := filepath.Join(t.TempDir(), "absent")
	var out bytes.Buffer
	if err := run(&out, []string{root}); err != nil {
		t.Fatalf("run: %v", err)
	}
	want := "encodecensus: no suite index at " + filepath.Join(root, "suite.xml") + " — corpus-absent mode, nothing to census."
	if !strings.HasPrefix(out.String(), want) {
		t.Errorf("report = %q, want it to open %q", out.String(), want)
	}
}
