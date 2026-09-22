package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// fixtureSuite is a miniature suite catalog carrying every shape the join
// turns on: one instance document named by three test groups, one of them
// withheld; a schema document named both by its own schemaTest and by the
// instance case assessed against it; and the declared outcomes the instance
// lane subtracts on.
var fixtureSuite = map[string]string{
	"suite.xml": `<testSuite xmlns:xlink="http://www.w3.org/1999/xlink">
  <testSetRef xlink:href="sets/s.testSet"/>
</testSuite>`,

	"sets/s.testSet": `<testSet name="S" xmlns:xlink="http://www.w3.org/1999/xlink">
  <testGroup name="g1">
    <schemaTest name="s1">
      <schemaDocument xlink:href="../docs/a.xsd"/>
      <expected validity="valid"/>
    </schemaTest>
    <instanceTest name="i1">
      <instanceDocument xlink:href="../docs/a1.xml"/>
      <expected validity="invalid"/>
    </instanceTest>
  </testGroup>
  <testGroup name="g2">
    <instanceTest name="i2">
      <instanceDocument xlink:href="../docs/a1.xml"/>
      <expected validity="valid"/>
    </instanceTest>
  </testGroup>
  <testGroup name="g3" version="1.0">
    <instanceTest name="i3">
      <instanceDocument xlink:href="../docs/a1.xml"/>
      <expected validity="invalid"/>
    </instanceTest>
  </testGroup>
  <testGroup name="g4">
    <schemaTest name="s4">
      <schemaDocument xlink:href="../docs/b.xsd"/>
      <expected validity="valid"/>
    </schemaTest>
    <instanceTest name="i4">
      <instanceDocument xlink:href="../docs/b1.xml"/>
      <expected validity="invalid"/>
    </instanceTest>
  </testGroup>
</testSet>`,
}

// fixtureLanes are the committed expectations the fixture suite is joined
// against: one case per class the report partitions into.
var fixtureLanes = map[string]string{
	"instance.txt": `S/g1/instance/i1 fail
S/g2/instance/i2 fail
S/g4/instance/i4 pass
`,
	"schema.txt": `S/g1/schema/s1 fail
S/g4/schema/s4 fail
`,
}

// writeTree materializes a name-to-body map under a fresh temp directory and
// returns its root.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, body := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("creating %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}
	return root
}

// runFixture drives the whole command against the fixture suite and lanes,
// with args behind the two directory flags, and returns what it printed.
func runFixture(t *testing.T, stdin string, args ...string) string {
	t.Helper()
	var out strings.Builder
	full := append([]string{"-suite", writeTree(t, fixtureSuite), "-expectations", writeTree(t, fixtureLanes)}, args...)
	if err := run(&out, strings.NewReader(stdin), full); err != nil {
		t.Fatalf("run %q: %v", args, err)
	}
	return out.String()
}

// wantLines fails unless every wanted substring is present, naming the output
// once so a failure is readable.
func wantLines(t *testing.T, got string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("output does not carry %q:\n%s", w, got)
			return
		}
	}
}

// wantRow fails unless the report carries the named partition row with the
// given count. It compares the label and the number and not the padding
// between them, so a column width is not a test to maintain.
func wantRow(t *testing.T, got, label string, count int) {
	t.Helper()
	for line := range strings.SplitSeq(got, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, label) {
			continue
		}
		fields := strings.Fields(trimmed)
		if last := fields[len(fields)-1]; last != strconv.Itoa(count) {
			t.Errorf("row %q counts %s, want %d", label, last, count)
		}
		return
	}
	t.Errorf("no row %q in:\n%s", label, got)
}

// TestIDsNamesEveryCatalogEntryOfAPathIncludingWithheldOnes is the direction
// the relation is one-to-many in: one document, three test groups, three case
// IDs — and the withheld one is reported and marked rather than dropped,
// because it is a catalog entry that merely carries no line in any lane
// (#1412, #1642).
func TestIDsNamesEveryCatalogEntryOfAPathIncludingWithheldOnes(t *testing.T) {
	got := runFixture(t, "", "ids", "docs/a1.xml")
	wantLines(t, got,
		"casejoin: 1 path(s), 3 catalog entry(ies) naming them, 1 withheld",
		"S/g1/instance/i1 — under test\n",
		"S/g2/instance/i2 — under test\n",
		"S/g3/instance/i3 — under test — WITHHELD\n",
	)
}

// TestIDsReportsBothDocumentEdges holds the reader to the two lists a case
// names documents through: a schema document is named by its own schemaTest
// and by the instance case assessed against it, and a tool reporting only the
// first answers zero for every schema-side census.
func TestIDsReportsBothDocumentEdges(t *testing.T) {
	got := runFixture(t, "", "ids", "docs/b.xsd")
	wantLines(t, got,
		"S/g4/instance/i4 — assessed against\n",
		"S/g4/schema/s4 — under test\n",
	)
}

// TestIDsReportsAPathNoCatalogEntryNames keeps a path the catalog never
// mentions visible: a census fixture no case names is a fact about the corpus,
// not an empty line to swallow.
func TestIDsReportsAPathNoCatalogEntryNames(t *testing.T) {
	got := runFixture(t, "", "ids", "docs/unreferenced.xml")
	wantLines(t, got, "(no catalog entry names this path)")
}

// TestJoinCountsOnlyTheBankedFailuresThatCouldFlip is the join itself: of the
// three entries naming the path, the withheld one carries no line, the one
// the suite declares valid is subtracted on this lane (#1561), and one
// candidate is left.
func TestJoinCountsOnlyTheBankedFailuresThatCouldFlip(t *testing.T) {
	got := runFixture(t, "", "join", "instance", "docs/a1.xml")
	wantLines(t, got,
		"casejoin: 1 path(s) → 3 catalog entry(ies) → 1 candidate case(s) in lane instance",
		"\n  S/g1/instance/i1\n",
	)
	wantRow(t, got, "no line in instance.txt — withheld (#1412)", 1)
	wantRow(t, got, "banked fail — suite declares it VALID, subtracted (#1561)", 1)
	wantRow(t, got, "banked fail — CANDIDATE", 1)
	if strings.Contains(got, "\n  S/g2/instance/i2\n") {
		t.Errorf("a case the suite declares VALID is listed as a candidate:\n%s", got)
	}
}

// TestJoinSubtractsSuiteValidCasesOnTheInstanceLaneOnly holds the #1561
// subtraction to the lane it is a fact about. S/g4/schema/s4 is declared valid
// and banked fail: on the schema lane it is a candidate like any other, and a
// blanket subtraction would hide it.
func TestJoinSubtractsSuiteValidCasesOnTheInstanceLaneOnly(t *testing.T) {
	got := runFixture(t, "", "join", "schema", "docs/b.xsd")
	wantLines(t, got,
		"→ 1 candidate case(s) in lane schema",
		"\n  S/g4/schema/s4\n",
		"No VALID-case subtraction is made for lane schema",
	)
	if strings.Contains(got, "subtracted (#1561)") {
		t.Errorf("the instance lane's subtraction row is printed for lane schema:\n%s", got)
	}
}

// TestJoinDoesNotCountAPassOrAnUnbankedCase holds the two remaining classes
// apart from the candidates: a case this lane already passes cannot flip up,
// and a case this lane's file carries no line for is not this lane's to count
// (#1412).
func TestJoinDoesNotCountAPassOrAnUnbankedCase(t *testing.T) {
	got := runFixture(t, "", "join", "instance", "docs/b.xsd", "docs/b1.xml")
	wantLines(t, got, "→ 0 candidate case(s) in lane instance")
	wantRow(t, got, "catalog entries naming a path", 2)
	wantRow(t, got, "no line in instance.txt — not banked by this lane", 1)
	wantRow(t, got, "banked pass — cannot flip up", 1)
}

// TestJoinCountsAnEntryOnceHoweverManyGivenPathsItNames pins the deduplication
// the bound depends on: S/g4/instance/i4 names one path under test and another
// it is assessed against, and counting it twice would inflate every figure it
// appears in.
func TestJoinCountsAnEntryOnceHoweverManyGivenPathsItNames(t *testing.T) {
	got := runFixture(t, "docs/b.xsd\ndocs/b1.xml\n", "join", "instance")
	wantRow(t, got, "catalog entries naming a path", 2)
}

// TestJoinCaveatNamesAllThreeDirections holds the figure's caveat to every
// direction it is wrong in. A caveat naming one of three is the false
// statement #1585 exists to prevent, with a smaller radius.
func TestJoinCaveatNamesAllThreeDirections(t *testing.T) {
	got := runFixture(t, "", "join", "instance", "docs/a1.xml")
	wantLines(t, got,
		"BOUND FROM ABOVE",
		"OVER-counts",
		"<element ref> or <group ref>",
		"UNDER-counts",
		"Read only partly",
		"declares VALID",
	)
	header := strings.Index(got, "casejoin: ")
	caveat := strings.Index(got, "BOUND FROM ABOVE")
	body := strings.Index(got, "=== Join against")
	if !(header < caveat && caveat < body) {
		t.Errorf("the caveat must sit under the header, ahead of the body (#1279):\n%s", got)
	}
}

// TestPathsAreTakenFromStdinAndRestated pins the input contract: paths arrive
// one per line when none is given on the command line, a path still carrying
// the suite root on its front is restated rather than silently matching
// nothing, blanks are skipped, and one fixture named twice is one fixture.
func TestPathsAreTakenFromStdinAndRestated(t *testing.T) {
	suite := writeTree(t, fixtureSuite)
	var out strings.Builder
	stdin := strings.NewReader("\n" + filepath.ToSlash(filepath.Join(suite, "docs/a1.xml")) + "\n./docs/a1.xml\n")
	args := []string{"-suite", suite, "-expectations", writeTree(t, fixtureLanes), "ids"}
	if err := run(&out, stdin, args); err != nil {
		t.Fatalf("run: %v", err)
	}
	wantLines(t, out.String(), "casejoin: 1 path(s), 3 catalog entry(ies) naming them, 1 withheld")
}

// TestJoinRefusesALaneWithNoExpectationFile keeps a mistyped lane from
// answering zero: LoadExpectations is right to read a missing file as an empty
// lane, and a join that counted against one would report a finding.
func TestJoinRefusesALaneWithNoExpectationFile(t *testing.T) {
	var out strings.Builder
	args := []string{"-suite", writeTree(t, fixtureSuite), "-expectations", writeTree(t, fixtureLanes), "join", "instnace", "docs/a1.xml"}
	err := run(&out, strings.NewReader(""), args)
	if err == nil {
		t.Fatalf("join against a lane with no file = nil error; output:\n%s", out.String())
	}
	if !strings.HasPrefix(err.Error(), `lane "instnace" has no expectation file`) {
		t.Errorf("error %q does not open by naming the lane it refused: the file path behind it "+
			"carries the same word, so only the subject position pins it (#1048)", err)
	}
}

// TestUnusableModeIsAnOperationalError holds the two mode errors to a
// non-zero exit rather than an empty report.
func TestUnusableModeIsAnOperationalError(t *testing.T) {
	for _, args := range [][]string{{}, {"jion", "instance"}, {"join"}} {
		var out strings.Builder
		if err := run(&out, strings.NewReader(""), args); err == nil {
			t.Errorf("run(%q) = nil error, want a usage error", args)
		}
	}
}

// TestCorpusAbsentModeReportsAndExitsCleanly holds the fresh-container case to
// a report and a clean exit (#659): the submodule is absent until it is
// initialized, which is a state to name, not a failure.
func TestCorpusAbsentModeReportsAndExitsCleanly(t *testing.T) {
	var out strings.Builder
	args := []string{"-suite", filepath.Join(t.TempDir(), "absent"), "join", "instance", "docs/a1.xml"}
	if err := run(&out, strings.NewReader(""), args); err != nil {
		t.Fatalf("run over an absent corpus: %v, want a clean exit", err)
	}
	wantLines(t, out.String(), "corpus-absent mode", "git submodule update --init")
}

// TestJoinRowsPartitionTheEntries holds the table to its own arithmetic: the
// five classes are disjoint and cover every entry, so a reader may subtract
// one row from another.
func TestJoinRowsPartitionTheEntries(t *testing.T) {
	j := joined{
		Lane:             instanceLane,
		WithheldIDs:      []string{"a"},
		UnscoredIDs:      []string{"b", "c"},
		BankedPassIDs:    []string{"d"},
		DeclaredValidIDs: []string{"e", "f"},
		CandidateIDs:     []string{"g"},
	}
	if j.entries() != 7 {
		t.Errorf("entries() = %d, want 7", j.entries())
	}
	sum := 0
	for _, r := range j.rows() {
		if strings.HasPrefix(r.label, "  ") {
			sum += r.count
		}
	}
	if sum != j.entries() {
		t.Errorf("the indented rows sum to %d, want the entry count %d", sum, j.entries())
	}
}
